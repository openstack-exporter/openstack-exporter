package exporters

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"log/slog"

	gophercloudv2 "github.com/gophercloud/gophercloud/v2"
	clientutilsv2 "github.com/gophercloud/utils/v2/client"
	clientconfigv2 "github.com/gophercloud/utils/v2/openstack/clientconfig"
	"github.com/hashicorp/go-uuid"
	"github.com/mitchellh/go-homedir"
	"github.com/openstack-exporter/openstack-exporter/utils"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	//nolint:unused
	BYTE = 1 << (10 * iota)
	//nolint:unused
	KILOBYTE
	MEGABYTE
	GIGABYTE
	//nolint:unused
	TERABYTE
)

// SupportedExporters contains all registered exporters.
var SupportedExporters = []string{}

type OpenStackExporter interface {
	prometheus.Collector

	GetName() string
	IsMetricEnabled(names ...string) bool
}

// DescribeDescs emits every non-nil *prometheus.Desc field found in descsPtr
// (a pointer to a struct containing *prometheus.Desc fields). Use it in
// exporter Describe() implementations as a reflection-based alternative to
// hand-coding each descriptor.
func DescribeDescs(ch chan<- *prometheus.Desc, descsPtr interface{}) {
	rv := reflect.ValueOf(descsPtr).Elem()
	descPtrType := reflect.TypeOf((*prometheus.Desc)(nil))
	for i := 0; i < rv.NumField(); i++ {
		if rv.Type().Field(i).Type == descPtrType {
			if d, ok := rv.Field(i).Interface().(*prometheus.Desc); ok && d != nil {
				ch <- d
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Exporter factory registry (populated by each exporter's init())
// ---------------------------------------------------------------------------

// ExporterFactory is a constructor function for an exporter.
type ExporterFactory func(config *ExporterConfig, logger *slog.Logger) (OpenStackExporter, error)

var exporterRegistry = map[string]ExporterFactory{}

// RegisterExporter registers a factory for the given OpenStack service name.
// Call from each exporter's init() so NewExporter() can dispatch without a switch.
func RegisterExporter(serviceName string, factory ExporterFactory) {
	exporterRegistry[serviceName] = factory
	SupportedExporters = append(SupportedExporters, serviceName)
}

// RegisterTypedExporter adapts constructors that return concrete exporter
// types, keeping service init functions free of boilerplate wrappers.
func RegisterTypedExporter[T OpenStackExporter](
	serviceName string,
	factory func(config *ExporterConfig, logger *slog.Logger) (T, error),
) {
	RegisterExporter(serviceName, func(config *ExporterConfig, logger *slog.Logger) (OpenStackExporter, error) {
		return factory(config, logger)
	})
}

// ---------------------------------------------------------------------------
// Reflection-based descriptor filling
// ---------------------------------------------------------------------------

var (
	reFirstCap = regexp.MustCompile(`(.)([A-Z][a-z]+)`)
	reAllCap   = regexp.MustCompile(`([a-z0-9])([A-Z])`)
)

// camelToSnake converts a CamelCase identifier to snake_case.
// e.g. VolumeGB → volume_gb, PoolCapacityFreeGB → pool_capacity_free_gb.
func camelToSnake(s string) string {
	s = reFirstCap.ReplaceAllString(s, "${1}_${2}")
	s = reAllCap.ReplaceAllString(s, "${1}_${2}")
	return strings.ToLower(s)
}

// RegisterAndFillDescs reads exported *prometheus.Desc fields from dst using
// reflection. For each field it:
//   - derives the metric name from the "metric" struct tag (fallback: camelToSnake(FieldName))
//   - splits the "labels" tag on commas to get label names
//   - reads optional "slow" ("true") and "deprecated" (version string) tags
//   - skips the field (leaves it nil) when the metric is disabled/slow/deprecated
//   - otherwise creates a prometheus.Desc, sets the field, and appends to allDescs
//
// Also creates base.upDesc on the first call. Call once during construction.
func (base *BaseOpenStackExporter) RegisterAndFillDescs(dst interface{}) {
	if base.upDesc == nil {
		base.upDesc = prometheus.NewDesc(
			prometheus.BuildFQName(base.GetName(), "", "up"),
			"up", nil, nil)
		ns := base.GetName()
		base.scrapesTotal = prometheus.NewCounter(prometheus.CounterOpts{
			Name: prometheus.BuildFQName(ns, "exporter", "scrapes_total"),
			Help: "Total number of scrapes.",
		})
		base.scrapeErrors = prometheus.NewCounter(prometheus.CounterOpts{
			Name: prometheus.BuildFQName(ns, "exporter", "scrape_errors_total"),
			Help: "Total number of scrapes that had at least one source fetch error.",
		})
		if base.CollectTime {
			base.sourceFetchDuration = prometheus.NewHistogramVec(
				prometheus.HistogramOpts{
					Name:    prometheus.BuildFQName(base.Prefix, "exporter", "source_fetch_duration_seconds"),
					Help:    "Duration of source fetches in seconds.",
					Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60},
					ConstLabels: prometheus.Labels{
						"service": base.Name,
					},
				},
				[]string{"source"},
			)
		}
	}

	rv := reflect.ValueOf(dst).Elem()
	rt := rv.Type()
	descPtrType := reflect.TypeOf((*prometheus.Desc)(nil))

	for i := 0; i < rt.NumField(); i++ {
		sf := rt.Field(i)
		if !sf.IsExported() || sf.Type != descPtrType {
			continue
		}

		name := sf.Tag.Get("metric")
		if name == "" {
			name = camelToSnake(sf.Name)
		}

		labelsRaw := sf.Tag.Get("labels")
		var labels []string
		if labelsRaw != "" {
			labels = strings.Split(labelsRaw, ",")
		}

		deprecated := sf.Tag.Get("deprecated")
		slow := sf.Tag.Get("slow") == "true"

		if !base.IsMetricEnabled(name) {
			base.logger.Warn("metric disabled for exporter", "metric", name, "exporter", base.Name)
			continue
		}
		if base.DisableDeprecatedMetrics && deprecated != "" && !base.isExplicitlyEnabled(name) {
			continue
		}
		if base.DisableSlowMetrics && slow && !base.isExplicitlyEnabled(name) {
			continue
		}
		if deprecated != "" {
			base.logger.Warn("metric deprecated", "metric", name, "exporter", base.Name, "version", deprecated)
		}

		base.logger.Info("Adding metric to exporter", "metric", name, "exporter", base.Name, "disable_key", base.qualifiedMetricName(name))
		desc := prometheus.NewDesc(
			prometheus.BuildFQName(base.GetName(), "", name),
			name, labels, nil)
		rv.Field(i).Set(reflect.ValueOf(desc))
		base.allDescs = append(base.allDescs, desc)
	}
}

// RegisterDesc adds an externally-created *prometheus.Desc to the list that
// Describe will emit. Use for metrics whose label set cannot be expressed via
// struct tags (e.g. metrics with dynamically-appended labels).
func (base *BaseOpenStackExporter) RegisterDesc(d *prometheus.Desc) {
	if d != nil {
		base.allDescs = append(base.allDescs, d)
	}
}

// ---------------------------------------------------------------------------
// Runtime DAG collection engine
// ---------------------------------------------------------------------------

// Source is a data-fetching node in the collection DAG. E is the concrete
// exporter type (e.g. *CinderExporter) and S is the per-scrape state struct.
// Fetch writes results into *S; concurrent Sources must write to disjoint fields.
type Source[E, S any] struct {
	Name      string
	DependsOn []string // names of Sources this one depends on
	Fetch     func(E, context.Context, *S) error
}

// Emitter is a metric-emitting node that reads from the scrape state S.
// Emit can run once all of the Source names it declares have completed.
type Emitter[E, S any] struct {
	Name    string
	Metrics []string // metric names this emitter may emit (for pruning)
	Sources []string // source names this emitter requires
	Emit    func(E, context.Context, *S, chan<- prometheus.Metric) error
}

// Graph declares the static DAG topology for an exporter.
// ComputeSchedule() computes a topo-sorted Schedule at runtime.
type Graph[E, S any] struct {
	Sources  []Source[E, S]
	Emitters []Emitter[E, S]
}

type scheduleNodeKind int

const (
	scheduleSource scheduleNodeKind = iota
	scheduleEmitter
)

type scheduleNode struct {
	kind  scheduleNodeKind
	index int
}

// Schedule is a dependency graph execution plan for a Graph. The node and
// edge fields drive execution; waves are retained for deterministic DAG logging.
type Schedule struct {
	nodes      []scheduleNode
	deps       [][]int
	dependents [][]int
	waves      [][]int
}

// ComputeSchedule computes a topo-sorted Schedule at runtime using Kahn's
// algorithm. It validates the DAG (detecting cycles and missing dependencies).
func (g *Graph[E, S]) ComputeSchedule() (Schedule, error) {
	liveSrc := make([]bool, len(g.Sources))
	for i := range liveSrc {
		liveSrc[i] = true
	}
	liveEmit := make([]bool, len(g.Emitters))
	for i := range liveEmit {
		liveEmit[i] = true
	}
	return g.computeSchedule(liveSrc, liveEmit)
}

// PruneSchedule computes a schedule containing only nodes needed given base's
// enabled/disabled metric configuration. An emitter is dropped when all of its
// declared metrics are disabled; a source is dropped when no live emitter
// (transitively) depends on it. Call once at construction time.
func (g *Graph[E, S]) PruneSchedule(base *BaseOpenStackExporter) (Schedule, error) {
	liveEmit := make([]bool, len(g.Emitters))
	for i, em := range g.Emitters {
		for _, m := range em.Metrics {
			if base.IsMetricEnabled(m) {
				liveEmit[i] = true
				break
			}
		}
	}

	srcIdx := func(name string) int {
		for i, s := range g.Sources {
			if s.Name == name {
				return i
			}
		}
		return -1
	}

	liveSrc := make([]bool, len(g.Sources))
	var markSrc func(int)
	markSrc = func(idx int) {
		if idx < 0 || liveSrc[idx] {
			return
		}
		liveSrc[idx] = true
		for _, dep := range g.Sources[idx].DependsOn {
			markSrc(srcIdx(dep))
		}
	}
	for i, em := range g.Emitters {
		if !liveEmit[i] {
			continue
		}
		for _, sName := range em.Sources {
			markSrc(srcIdx(sName))
		}
	}

	sched, err := g.computeSchedule(liveSrc, liveEmit)
	if err != nil {
		return Schedule{}, err
	}
	return sched, nil
}

func (g *Graph[E, S]) computeSchedule(liveSrc, liveEmit []bool) (Schedule, error) {
	if len(liveSrc) != len(g.Sources) {
		return Schedule{}, fmt.Errorf("source liveness length mismatch")
	}
	if len(liveEmit) != len(g.Emitters) {
		return Schedule{}, fmt.Errorf("emitter liveness length mismatch")
	}

	srcIdx := make(map[string]int, len(g.Sources))
	for i, s := range g.Sources {
		if _, exists := srcIdx[s.Name]; exists {
			return Schedule{}, fmt.Errorf("duplicate source name: %q", s.Name)
		}
		srcIdx[s.Name] = i
	}
	for _, s := range g.Sources {
		for _, dep := range s.DependsOn {
			if _, ok := srcIdx[dep]; !ok {
				return Schedule{}, fmt.Errorf("source %q depends on missing source %q", s.Name, dep)
			}
		}
	}

	emIdx := make(map[string]int, len(g.Emitters))
	for i, em := range g.Emitters {
		if _, exists := emIdx[em.Name]; exists {
			return Schedule{}, fmt.Errorf("duplicate emitter name: %q", em.Name)
		}
		emIdx[em.Name] = i
		for _, src := range em.Sources {
			if _, ok := srcIdx[src]; !ok {
				return Schedule{}, fmt.Errorf("emitter %q depends on missing source %q", em.Name, src)
			}
		}
	}

	var nodes []scheduleNode
	sourceNode := make([]int, len(g.Sources))
	for i := range sourceNode {
		sourceNode[i] = -1
	}
	for i := range g.Sources {
		if liveSrc[i] {
			sourceNode[i] = len(nodes)
			nodes = append(nodes, scheduleNode{kind: scheduleSource, index: i})
		}
	}
	emitterNode := make([]int, len(g.Emitters))
	for i := range emitterNode {
		emitterNode[i] = -1
	}
	for i := range g.Emitters {
		if liveEmit[i] {
			emitterNode[i] = len(nodes)
			nodes = append(nodes, scheduleNode{kind: scheduleEmitter, index: i})
		}
	}

	deps := make([][]int, len(nodes))
	dependents := make([][]int, len(nodes))
	addDep := func(node, dep int) {
		if node < 0 || dep < 0 {
			return
		}
		deps[node] = append(deps[node], dep)
		dependents[dep] = append(dependents[dep], node)
	}

	for src, node := range sourceNode {
		if node < 0 {
			continue
		}
		for _, depName := range g.Sources[src].DependsOn {
			addDep(node, sourceNode[srcIdx[depName]])
		}
	}
	for em, node := range emitterNode {
		if node < 0 {
			continue
		}
		for _, srcName := range g.Emitters[em].Sources {
			addDep(node, sourceNode[srcIdx[srcName]])
		}
	}

	waves, err := scheduleWaves(deps, dependents)
	if err != nil {
		return Schedule{}, err
	}

	return Schedule{
		nodes:      nodes,
		deps:       deps,
		dependents: dependents,
		waves:      waves,
	}, nil
}

func scheduleWaves(deps, dependents [][]int) ([][]int, error) {
	remaining := make([]int, len(deps))
	var ready []int
	for i := range deps {
		remaining[i] = len(deps[i])
		if remaining[i] == 0 {
			ready = append(ready, i)
		}
	}

	var waves [][]int
	var visited int
	for len(ready) > 0 {
		wave := append([]int(nil), ready...)
		waves = append(waves, wave)
		visited += len(wave)

		var next []int
		for _, node := range wave {
			for _, dependent := range dependents[node] {
				remaining[dependent]--
				if remaining[dependent] == 0 {
					next = append(next, dependent)
				}
			}
		}
		ready = next
	}

	if visited != len(deps) {
		return nil, fmt.Errorf("cycle detected in DAG")
	}
	return waves, nil
}

// LogDAG emits a DEBUG log describing the pruned schedule topology.
// Each active source wave is shown with the source names; each active emitter
// is shown with its source dependencies and the metrics it will emit.
// Disabled/pruned nodes are listed separately so the operator can see what
// was dropped and why.
func (g *Graph[E, S]) LogDAG(base *BaseOpenStackExporter, sched Schedule) {
	// Build sets of live indices for quick membership check.
	liveSrc := make(map[int]bool)
	liveEm := make(map[int]bool)
	for _, node := range sched.nodes {
		switch node.kind {
		case scheduleSource:
			liveSrc[node.index] = true
		case scheduleEmitter:
			liveEm[node.index] = true
		}
	}

	// Log topological waves for visibility. Execution is dependency-driven, so
	// later-wave nodes can start as soon as their own predecessors finish.
	for wi, wave := range sched.waves {
		names := make([]string, len(wave))
		for i, nodeIdx := range wave {
			names[i] = g.nodeLogName(sched.nodes[nodeIdx])
		}
		base.logger.Debug("DAG wave", "exporter", base.Name, "wave", wi, "nodes", strings.Join(names, ", "))
	}

	// Log active emitters.
	for _, node := range sched.nodes {
		if node.kind == scheduleEmitter {
			em := g.Emitters[node.index]
			base.logger.Debug("DAG emitter",
				"exporter", base.Name,
				"emitter", em.Name,
				"sources", strings.Join(em.Sources, ","),
				"metrics", strings.Join(em.Metrics, ","),
			)
		}
	}

	// Log pruned sources.
	var prunedSrc []string
	for i, src := range g.Sources {
		if !liveSrc[i] {
			prunedSrc = append(prunedSrc, src.Name)
		}
	}
	if len(prunedSrc) > 0 {
		base.logger.Debug("DAG pruned sources (no live emitters depend on them)",
			"exporter", base.Name, "sources", strings.Join(prunedSrc, ", "))
	}

	// Log pruned emitters.
	var prunedEm []string
	for i, em := range g.Emitters {
		if !liveEm[i] {
			prunedEm = append(prunedEm, em.Name+"["+strings.Join(em.Metrics, ",")+"]")
		}
	}
	if len(prunedEm) > 0 {
		base.logger.Debug("DAG pruned emitters (all metrics disabled)",
			"exporter", base.Name, "emitters", strings.Join(prunedEm, ", "))
	}
}

func (g *Graph[E, S]) nodeLogName(node scheduleNode) string {
	switch node.kind {
	case scheduleSource:
		src := g.Sources[node.index]
		if len(src.DependsOn) > 0 {
			return "source:" + src.Name + "[deps:" + strings.Join(src.DependsOn, ",") + "]"
		}
		return "source:" + src.Name
	case scheduleEmitter:
		em := g.Emitters[node.index]
		if len(em.Sources) > 0 {
			return "emitter:" + em.Name + "[sources:" + strings.Join(em.Sources, ",") + "]"
		}
		return "emitter:" + em.Name
	default:
		return "unknown"
	}
}

// runSchedule executes the pruned wave schedule for exporter e. It runs
// each source or emitter as soon as its dependencies have completed.
// Source errors are counted; emitter execution writes metrics directly to ch.
func runSchedule[E any, S any](
	e E, base *BaseOpenStackExporter, g *Graph[E, S], sched Schedule, s *S, ch chan<- prometheus.Metric,
) int {
	ctx := context.TODO()
	var failures atomic.Int64
	if len(sched.nodes) == 0 {
		return 0
	}

	remaining := make([]int, len(sched.deps))
	completed := make(chan int, len(sched.nodes))
	var wg sync.WaitGroup

	startNode := func(nodeIdx int) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			node := sched.nodes[nodeIdx]
			switch node.kind {
			case scheduleSource:
				src := g.Sources[node.index]
				start := time.Now()
				err := src.Fetch(e, ctx, s)
				base.observeSourceFetchDuration(src.Name, time.Since(start))
				if err != nil {
					failures.Add(1)
					base.logger.Error("Source fetch error", "source", src.Name, "err", err)
				}
			case scheduleEmitter:
				em := g.Emitters[node.index]
				if err := em.Emit(e, ctx, s, ch); err != nil {
					base.logger.Error("Emitter error", "emitter", em.Name, "err", err)
				}
			}
			completed <- nodeIdx
		}()
	}

	for i := range sched.nodes {
		remaining[i] = len(sched.deps[i])
		if remaining[i] == 0 {
			startNode(i)
		}
	}

	for completedCount := 0; completedCount < len(sched.nodes); completedCount++ {
		nodeIdx := <-completed
		for _, dependent := range sched.dependents[nodeIdx] {
			remaining[dependent]--
			if remaining[dependent] == 0 {
				startNode(dependent)
			}
		}
	}
	wg.Wait()

	return int(failures.Load())
}

// ExporterOptions holds all user-supplied configuration for an exporter.
// It is passed directly to NewExporter and is embedded in
// ExporterConfig, which adds the resolved service client and service name.
type ExporterOptions struct {
	// Cloud is the name of the cloud entry in clouds.yaml to scrape.
	Cloud string
	// Prefix is the metric name prefix (default: "openstack").
	Prefix string
	// DisabledMetrics is a list of "exporter-metric" keys to suppress entirely
	// (e.g. "cinder-snapshots"). Takes precedence over EnabledMetrics.
	DisabledMetrics []string
	// EnabledMetrics overrides DisableSlowMetrics / DisableDeprecatedMetrics for
	// individual metrics, using the same "exporter-metric" format
	// (e.g. "nova-limits_vcpus_max").
	EnabledMetrics []string
	// EndpointType selects the OpenStack endpoint to connect to
	// ("public", "internal", or "admin").
	EndpointType string
	// CollectTime enables per-source fetch duration metrics.
	CollectTime bool
	// DisableSlowMetrics suppresses metrics marked Slow: true in their
	// definition. Individual metrics can be re-enabled via EnabledMetrics.
	DisableSlowMetrics bool
	// DisableDeprecatedMetrics suppresses metrics that carry a DeprecatedVersion.
	// Individual metrics can be re-enabled via EnabledMetrics.
	DisableDeprecatedMetrics bool
	// DisableCinderAgentUUID disables UUID generation for Cinder agent metrics.
	DisableCinderAgentUUID bool
	// DomainID restricts metric collection to a single Keystone domain.
	// Empty string means all domains.
	DomainID string
	// TenantID restricts metric collection to a single project.
	// Empty string means all projects.
	TenantID string
	// NovaMetadataMapping maps Nova server metadata keys to extra Prometheus labels
	// on the openstack_nova_server_status metric.
	NovaMetadataMapping *utils.LabelMappingFlag
	// DnsConcurrentCount controls the number of concurrent requests used when
	// collecting DNS recordsets.
	DnsConcurrentCount int
	// UUIDGenFunc is the function used to generate UUIDs for Cinder agents.
	// Defaults to uuid.GenerateUUID when nil.
	UUIDGenFunc func() (string, error)
}

type ExporterConfig struct {
	ExporterOptions
	ClientV2    *gophercloudv2.ServiceClient
	ServiceName string
}

type BaseOpenStackExporter struct {
	ExporterConfig
	Name     string
	upDesc   *prometheus.Desc   // the "up" gauge, created by RegisterAndFillDescs
	allDescs []*prometheus.Desc // all descs created by RegisterAndFillDescs + RegisterDesc
	logger   *slog.Logger
	// scrape instrumentation (initialised by RegisterAndFillDescs)
	scrapesTotal        prometheus.Counter
	scrapeErrors        prometheus.Counter
	sourceFetchDuration *prometheus.HistogramVec
}

var (
	endpointOptsV2   map[string]gophercloudv2.EndpointOpts
	endpointOptsV2Mu sync.Mutex
)

func (exporter *BaseOpenStackExporter) GetName() string {
	return fmt.Sprintf("%s_%s", exporter.Prefix, exporter.Name)
}

// qualifiedMetricName returns the backward-compatible "exporter-metric" key
// used in DisabledMetrics / EnabledMetrics lists (e.g. "nova-limits_vcpus_max").
func (exporter *BaseOpenStackExporter) qualifiedMetricName(name string) string {
	return exporter.Name + "-" + name
}

// isExplicitlyEnabled reports whether name appears in the EnabledMetrics list.
// An explicitly-enabled metric overrides global DisableSlowMetrics /
// DisableDeprecatedMetrics flags.
func (exporter *BaseOpenStackExporter) isExplicitlyEnabled(name string) bool {
	return slices.Contains(exporter.EnabledMetrics, exporter.qualifiedMetricName(name))
}

// IsMetricEnabled reports whether any of the given metrics should be collected
// and emitted. A metric is disabled when it appears in DisabledMetrics; an
// explicit entry in EnabledMetrics does NOT override a DisabledMetrics entry.
// Passing several names lets callers guard an expensive API call that feeds
// multiple metrics, collecting only when at least one of them is enabled.
func (exporter *BaseOpenStackExporter) IsMetricEnabled(names ...string) bool {
	for _, name := range names {
		if !slices.Contains(exporter.DisabledMetrics, exporter.qualifiedMetricName(name)) {
			return true
		}
	}
	return false
}

func (exporter *BaseOpenStackExporter) Describe(ch chan<- *prometheus.Desc) {
	if exporter.upDesc != nil {
		ch <- exporter.upDesc
	}
	if exporter.scrapesTotal != nil {
		ch <- exporter.scrapesTotal.Desc()
		ch <- exporter.scrapeErrors.Desc()
	}
	if exporter.sourceFetchDuration != nil {
		exporter.sourceFetchDuration.Describe(ch)
	}
	for _, d := range exporter.allDescs {
		ch <- d
	}
}

// emitUp emits the exporter's "up" gauge (1 = healthy, 0 = all sources failed).
func (exporter *BaseOpenStackExporter) emitUp(ch chan<- prometheus.Metric, up float64) {
	if exporter.upDesc != nil {
		ch <- prometheus.MustNewConstMetric(exporter.upDesc, prometheus.GaugeValue, up)
	}
}

// TotalSources returns the total number of source nodes in the schedule.
func (ws Schedule) TotalSources() int {
	n := 0
	for _, node := range ws.nodes {
		if node.kind == scheduleSource {
			n++
		}
	}
	return n
}

// RunCollect runs the provided collection function, records failure counters,
// emits the instrumentation metrics, and emits the "up" gauge.
// Each exporter's Collect() should delegate to this method.
func (base *BaseOpenStackExporter) RunCollect(
	ch chan<- prometheus.Metric,
	sched Schedule,
	run func(chan<- prometheus.Metric) int,
) {
	base.scrapesTotal.Inc()
	failures := run(ch)
	if failures > 0 {
		base.scrapeErrors.Inc()
	}
	ch <- base.scrapesTotal
	ch <- base.scrapeErrors
	if base.sourceFetchDuration != nil {
		base.sourceFetchDuration.Collect(ch)
	}
	if sched.TotalSources() == 0 || failures >= sched.TotalSources() {
		base.emitUp(ch, 0)
	} else {
		base.emitUp(ch, 1)
	}
}

func (exporter *BaseOpenStackExporter) observeSourceFetchDuration(source string, duration time.Duration) {
	if exporter.sourceFetchDuration != nil {
		exporter.sourceFetchDuration.WithLabelValues(source).Observe(duration.Seconds())
	}
}

// emitGauge emits a GaugeValue metric if desc is non-nil. It is a no-op when the
// metric was pruned (desc == nil), removing the need for per-call nil checks.
func emitGauge(ch chan<- prometheus.Metric, desc *prometheus.Desc, value float64, labelValues ...string) {
	if desc != nil {
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, value, labelValues...)
	}
}

func (exporter *BaseOpenStackExporter) GetDnsConcurrencyCount() int {
	return exporter.DnsConcurrentCount
}

// took from here:
// https://github.com/gophercloud/utils/blob/4c0f6d93d3a9b027a21d9206b6bdd09123de7a09/internal/util.go#L87
func pathOrContents(poc string) ([]byte, bool, error) {
	if len(poc) == 0 {
		return nil, false, nil
	}

	path := poc
	if path[0] == '~' {
		var err error
		path, err = homedir.Expand(path)
		if err != nil {
			return []byte(path), true, err
		}
	}

	if _, err := os.Stat(path); err == nil {
		contents, err := os.ReadFile(path)
		if err != nil {
			return contents, true, err
		}
		return contents, true, nil
	}

	return []byte(poc), false, nil
}

func NewExporter(name string, opts ExporterOptions, logger *slog.Logger) (OpenStackExporter, error) {
	var transport http.RoundTripper
	var tlsConfig tls.Config

	optsv2 := clientconfigv2.ClientOpts{Cloud: opts.Cloud}

	config, err := clientconfigv2.GetCloudFromYAML(&optsv2)
	if err != nil {
		return nil, err
	}

	var configureTransport = false
	if !*config.Verify {
		logger.Info("SSL verification disabled on transport")
		tlsConfig.InsecureSkipVerify = true
		configureTransport = true
	} else if config.CACertFile != "" {
		certPool, err := additionalTLSTrust(config.CACertFile, logger)
		if err != nil {
			logger.Error("Failed to include additional certificates to ca-trust", "err", err)
		}
		tlsConfig.RootCAs = certPool
		configureTransport = true
	}

	// took from here:
	// https://github.com/gophercloud/utils/blob/4c0f6d93d3a9b027a21d9206b6bdd09123de7a09/internal/util.go#L65
	if config.ClientCertFile != "" && config.ClientKeyFile != "" {
		clientCert, _, err := pathOrContents(config.ClientCertFile)
		if err != nil {
			return nil, fmt.Errorf("error reading Client Cert: %s", err)
		}
		clientKey, _, err := pathOrContents(config.ClientKeyFile)
		if err != nil {
			return nil, fmt.Errorf("error reading Client Key: %s", err)
		}
		cert, err := tls.X509KeyPair(clientCert, clientKey)
		if err != nil {
			return nil, err
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
		configureTransport = true
	}
	if configureTransport {
		transport = &http.Transport{TLSClientConfig: &tlsConfig}
	}

	if _, ok := os.LookupEnv("OS_DEBUG"); ok {
		if transport == nil {
			transport = http.DefaultTransport
		}

		transport = &clientutilsv2.RoundTripper{
			Rt:     transport,
			Logger: &clientutilsv2.DefaultLogger{},
		}
	}

	clientV2, err := NewServiceClientV2(name, &optsv2, transport, opts.EndpointType)
	if err != nil {
		return nil, err
	}

	uuidGenFunc := opts.UUIDGenFunc
	if uuidGenFunc == nil {
		uuidGenFunc = uuid.GenerateUUID
	}
	opts.UUIDGenFunc = uuidGenFunc

	exporterConfig := ExporterConfig{
		ExporterOptions: opts,
		ClientV2:        clientV2,
		ServiceName:     name,
	}

	factory, ok := exporterRegistry[name]
	if !ok {
		return nil, fmt.Errorf("couldn't find a handler for %s exporter", name)
	}

	return factory(&exporterConfig, logger)
}
