package exporters

import (
	"context"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/jpillora/backoff"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// DefaultBindRetryMin is the first delay before a failed bind is retried.
	DefaultBindRetryMin = 5 * time.Second
	// DefaultBindRetryMax caps the delay between bind retries.
	DefaultBindRetryMax = 5 * time.Minute
)

var (
	collectorReadyDesc = prometheus.NewDesc(
		"openstack_exporter_collector_ready",
		"1 when the collector for the service is bound and serving metrics, 0 while it waits for a successful bind.",
		[]string{"cloud", "service"}, nil,
	)
	collectorBoundDesc = prometheus.NewDesc(
		"openstack_exporter_collector_bound_timestamp_seconds",
		"Unix time of the last successful bind of the collector.",
		[]string{"cloud", "service"}, nil,
	)
)

// Pool owns one collector per cloud and service and rebinds them on demand.
//
// Binding a collector means running the service constructor: Keystone
// authentication, endpoint lookup and whatever probing the service does. A
// failed bind never replaces a working collector. An entry that has no
// collector yet is retried with backoff on every Ensure until it binds, so a
// service that was unreachable when the process started does not stay missing
// for the life of the process. Reload rebinds everything regardless of state,
// which is how the periodic timer and SIGHUP pick up new tokens, catalog
// changes and probe results.
type Pool struct {
	// RetryMin and RetryMax bound the backoff between bind attempts of an
	// entry that has no collector.
	RetryMin time.Duration
	RetryMax time.Duration
	// Factory builds the collector of a service; NewExporter by default.
	Factory func(service string, opts ExporterOptions, logger *slog.Logger) (OpenStackExporter, error)

	opts   ExporterOptions
	logger *slog.Logger
	now    func() time.Time

	mu      sync.Mutex
	desired map[string][]string
	entries map[poolKey]*poolEntry
}

type poolKey struct {
	cloud   string
	service string
}

type poolEntry struct {
	exporter OpenStackExporter
	boundAt  time.Time
	err      error
	retry    backoff.Backoff
	retryAt  time.Time
	binding  bool
}

// Bound is the state of a cloud's collectors at one moment.
type Bound struct {
	// Exporters holds the collectors that are bound, keyed by service.
	Exporters map[string]OpenStackExporter
	// Ready reports openstack_exporter_collector_ready and
	// openstack_exporter_collector_bound_timestamp_seconds for every
	// requested service, bound or not.
	Ready prometheus.Collector
}

// NewPool returns an empty pool. opts.Cloud is ignored: the cloud is given per
// call.
func NewPool(opts ExporterOptions, logger *slog.Logger) *Pool {
	return &Pool{
		RetryMin: DefaultBindRetryMin,
		RetryMax: DefaultBindRetryMax,
		Factory:  NewExporter,
		opts:     opts,
		logger:   logger,
		now:      time.Now,
		desired:  map[string][]string{},
		entries:  map[poolKey]*poolEntry{},
	}
}

// Services returns the services the pool currently tracks for cloud, in the
// order they were requested.
func (p *Pool) Services(cloud string) []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.desired[cloud])
}

// Clouds returns every cloud the pool tracks, sorted.
func (p *Pool) Clouds() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	clouds := make([]string, 0, len(p.desired))
	for cloud := range p.desired {
		clouds = append(clouds, cloud)
	}
	slices.Sort(clouds)
	return clouds
}

// Reload makes services the full set tracked for cloud and rebinds each of
// them, dropping collectors of services no longer listed. A bind that fails
// keeps the previous collector. Binding stops early when ctx is cancelled.
func (p *Pool) Reload(ctx context.Context, cloud string, services []string) {
	p.mu.Lock()
	p.desired[cloud] = slices.Clone(services)
	for key := range p.entries {
		if key.cloud == cloud && !slices.Contains(services, key.service) {
			delete(p.entries, key)
			p.logger.Info("collector dropped", "cloud", cloud, "service", key.service)
		}
	}
	p.mu.Unlock()

	for _, service := range services {
		if ctx.Err() != nil {
			return
		}
		key := poolKey{cloud: cloud, service: service}
		p.mu.Lock()
		entry := p.entry(key)
		if entry.binding {
			p.mu.Unlock()
			continue
		}
		entry.binding = true
		p.mu.Unlock()
		p.bind(key, entry)
	}
}

// ReloadAll rebinds every tracked cloud with its current services.
func (p *Pool) ReloadAll(ctx context.Context) {
	for _, cloud := range p.Clouds() {
		p.Reload(ctx, cloud, p.Services(cloud))
	}
}

// Ensure adds services to the set tracked for cloud and binds the ones that
// have no collector yet and whose retry delay has passed. Services that are
// bound, or currently being bound by another caller, are left alone.
func (p *Pool) Ensure(ctx context.Context, cloud string, services []string) {
	for _, service := range services {
		if ctx.Err() != nil {
			return
		}
		key := poolKey{cloud: cloud, service: service}
		p.mu.Lock()
		if !slices.Contains(p.desired[cloud], service) {
			p.desired[cloud] = append(p.desired[cloud], service)
		}
		entry := p.entry(key)
		if entry.exporter != nil || entry.binding || p.now().Before(entry.retryAt) {
			p.mu.Unlock()
			continue
		}
		entry.binding = true
		p.mu.Unlock()
		p.bind(key, entry)
	}
}

// Collectors is Ensure followed by Snapshot: the collectors to register for a
// scrape of services from cloud.
func (p *Pool) Collectors(ctx context.Context, cloud string, services []string) Bound {
	p.Ensure(ctx, cloud, services)
	return p.Snapshot(cloud, services)
}

// Snapshot returns the bound collectors among services plus a readiness
// collector covering all of them.
func (p *Pool) Snapshot(cloud string, services []string) Bound {
	p.mu.Lock()
	defer p.mu.Unlock()
	bound := Bound{
		Exporters: map[string]OpenStackExporter{},
		Ready:     &readyCollector{pool: p, cloud: cloud, services: slices.Clone(services)},
	}
	for _, service := range services {
		if entry, ok := p.entries[poolKey{cloud: cloud, service: service}]; ok && entry.exporter != nil {
			bound.Exporters[service] = entry.exporter
		}
	}
	return bound
}

// entry returns the entry for key, creating it. Caller holds p.mu.
func (p *Pool) entry(key poolKey) *poolEntry {
	entry, ok := p.entries[key]
	if !ok {
		entry = &poolEntry{retry: backoff.Backoff{Min: p.RetryMin, Max: p.RetryMax, Factor: 2, Jitter: true}}
		p.entries[key] = entry
	}
	return entry
}

// bind runs the constructor for key outside the lock and records the result
// in entry, which the caller has marked as binding.
func (p *Pool) bind(key poolKey, entry *poolEntry) {
	logger := p.logger.With("cloud", key.cloud, "service", key.service)
	opts := p.opts
	opts.Cloud = key.cloud
	exporter, err := p.Factory(key.service, opts, logger)

	p.mu.Lock()
	defer p.mu.Unlock()
	entry.binding = false
	if err != nil {
		entry.err = err
		delay := entry.retry.Duration()
		entry.retryAt = p.now().Add(delay)
		if entry.exporter != nil {
			logger.Warn("collector rebind failed, keeping the previous collector", "error", err, "bound_at", entry.boundAt)
		} else {
			logger.Warn("collector bind failed", "error", err, "retry_in", delay)
		}
		return
	}
	entry.exporter = exporter
	entry.boundAt = p.now()
	entry.err = nil
	entry.retry.Reset()
	entry.retryAt = time.Time{}
	logger.Info("collector bound")
}

// readyCollector reports the bind state of a cloud's services at collect
// time.
type readyCollector struct {
	pool     *Pool
	cloud    string
	services []string
}

func (c *readyCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collectorReadyDesc
	ch <- collectorBoundDesc
}

func (c *readyCollector) Collect(ch chan<- prometheus.Metric) {
	boundAt := make([]time.Time, len(c.services))
	c.pool.mu.Lock()
	for i, service := range c.services {
		if entry, ok := c.pool.entries[poolKey{cloud: c.cloud, service: service}]; ok && entry.exporter != nil {
			boundAt[i] = entry.boundAt
		}
	}
	c.pool.mu.Unlock()

	for i, service := range c.services {
		ready := 0.0
		if !boundAt[i].IsZero() {
			ready = 1
			ch <- prometheus.MustNewConstMetric(collectorBoundDesc, prometheus.GaugeValue, float64(boundAt[i].Unix()), c.cloud, service)
		}
		ch <- prometheus.MustNewConstMetric(collectorReadyDesc, prometheus.GaugeValue, ready, c.cloud, service)
	}
}
