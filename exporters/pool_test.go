package exporters

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeExporter is the smallest OpenStackExporter a pool can hold.
type fakeExporter struct {
	service string
	gen     int
}

func (f *fakeExporter) Describe(chan<- *prometheus.Desc) {}
func (f *fakeExporter) Collect(chan<- prometheus.Metric) {}
func (f *fakeExporter) GetName() string                  { return f.service }
func (f *fakeExporter) AddMetric(string, ListFunc, []string, string, prometheus.Labels) {
}
func (f *fakeExporter) MetricIsDisabled(string) bool { return false }

// fakeFactory stands in for NewExporter: it fails the services listed in
// failing and counts every call.
type fakeFactory struct {
	mu      sync.Mutex
	failing map[string]bool
	calls   map[string]int
}

func newFakeFactory() *fakeFactory {
	return &fakeFactory{failing: map[string]bool{}, calls: map[string]int{}}
}

func (f *fakeFactory) fail(service string, v bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failing[service] = v
}

func (f *fakeFactory) count(service string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls[service]
}

func (f *fakeFactory) new(service string, opts ExporterOptions, _ *slog.Logger) (OpenStackExporter, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls[service]++
	if f.failing[service] {
		return nil, errors.New("keystone unavailable")
	}
	return &fakeExporter{service: opts.Cloud + "/" + service, gen: f.calls[service]}, nil
}

type testClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *testClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *testClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func newTestPool(t *testing.T) (*Pool, *fakeFactory, *testClock) {
	t.Helper()
	factory := newFakeFactory()
	clock := &testClock{now: time.Unix(1_700_000_000, 0)}
	pool := NewPool(ExporterOptions{Prefix: "openstack"}, slog.New(slog.DiscardHandler))
	pool.Factory = factory.new
	pool.now = clock.Now
	return pool, factory, clock
}

func readyOutput(cloud string, states map[string]bool) string {
	var b strings.Builder
	b.WriteString("# HELP openstack_exporter_collector_ready 1 when the collector for the service is bound and serving metrics, 0 while it waits for a successful bind.\n")
	b.WriteString("# TYPE openstack_exporter_collector_ready gauge\n")
	for _, service := range []string{"compute", "network", "volume"} {
		ready, ok := states[service]
		if !ok {
			continue
		}
		v := "0"
		if ready {
			v = "1"
		}
		b.WriteString(`openstack_exporter_collector_ready{cloud="` + cloud + `",service="` + service + `"} ` + v + "\n")
	}
	return b.String()
}

func TestPoolReloadBindsAndReportsReady(t *testing.T) {
	pool, factory, _ := newTestPool(t)
	factory.fail("volume", true)

	pool.Reload(context.Background(), "papa", []string{"compute", "network", "volume"})

	bound := pool.Snapshot("papa", []string{"compute", "network", "volume"})
	assert.Len(t, bound.Exporters, 2)
	assert.Equal(t, "papa/compute", bound.Exporters["compute"].GetName())
	assert.NotContains(t, bound.Exporters, "volume")
	assert.Equal(t, []string{"compute", "network", "volume"}, pool.Services("papa"))
	assert.Equal(t, []string{"papa"}, pool.Clouds())

	expected := readyOutput("papa", map[string]bool{"compute": true, "network": true, "volume": false})
	require.NoError(t, testutil.CollectAndCompare(bound.Ready, strings.NewReader(expected), "openstack_exporter_collector_ready"))
	assert.Equal(t, 2, testutil.CollectAndCount(bound.Ready, "openstack_exporter_collector_bound_timestamp_seconds"))
}

func TestPoolEnsureRetriesUnboundWithBackoff(t *testing.T) {
	pool, factory, clock := newTestPool(t)
	factory.fail("compute", true)
	ctx := context.Background()

	pool.Ensure(ctx, "papa", []string{"compute"})
	assert.Equal(t, 1, factory.count("compute"))

	// Inside the retry delay nothing is attempted.
	pool.Ensure(ctx, "papa", []string{"compute"})
	assert.Equal(t, 1, factory.count("compute"))

	// Past the delay the bind is retried, and succeeds once the service is back.
	factory.fail("compute", false)
	clock.Advance(pool.RetryMax)
	bound := pool.Collectors(ctx, "papa", []string{"compute"})
	assert.Equal(t, 2, factory.count("compute"))
	assert.Contains(t, bound.Exporters, "compute")

	// Bound entries are never touched by Ensure.
	pool.Ensure(ctx, "papa", []string{"compute"})
	assert.Equal(t, 2, factory.count("compute"))
}

func TestPoolReloadKeepsPreviousCollectorOnFailure(t *testing.T) {
	pool, factory, _ := newTestPool(t)
	ctx := context.Background()

	pool.Reload(ctx, "papa", []string{"compute"})
	first := pool.Snapshot("papa", []string{"compute"}).Exporters["compute"]
	require.NotNil(t, first)

	factory.fail("compute", true)
	pool.Reload(ctx, "papa", []string{"compute"})
	assert.Same(t, first, pool.Snapshot("papa", []string{"compute"}).Exporters["compute"])
	expected := readyOutput("papa", map[string]bool{"compute": true})
	require.NoError(t, testutil.CollectAndCompare(pool.Snapshot("papa", []string{"compute"}).Ready, strings.NewReader(expected), "openstack_exporter_collector_ready"))

	factory.fail("compute", false)
	pool.Reload(ctx, "papa", []string{"compute"})
	second := pool.Snapshot("papa", []string{"compute"}).Exporters["compute"]
	assert.NotSame(t, first, second)
	assert.Equal(t, 3, second.(*fakeExporter).gen)
}

func TestPoolReloadDropsUnlistedServicesAndEnsureAdds(t *testing.T) {
	pool, _, _ := newTestPool(t)
	ctx := context.Background()

	pool.Reload(ctx, "papa", []string{"compute", "volume"})
	pool.Reload(ctx, "papa", []string{"compute"})
	assert.Equal(t, []string{"compute"}, pool.Services("papa"))
	assert.Empty(t, pool.Snapshot("papa", []string{"volume"}).Exporters)

	pool.Ensure(ctx, "papa", []string{"network"})
	assert.Equal(t, []string{"compute", "network"}, pool.Services("papa"))
	assert.Contains(t, pool.Snapshot("papa", []string{"network"}).Exporters, "network")
}

func TestPoolReloadAllCoversEveryCloud(t *testing.T) {
	pool, factory, _ := newTestPool(t)
	ctx := context.Background()

	pool.Ensure(ctx, "papa", []string{"compute"})
	pool.Ensure(ctx, "golf", []string{"compute", "network"})
	assert.Equal(t, []string{"golf", "papa"}, pool.Clouds())

	pool.ReloadAll(ctx)
	assert.Equal(t, 4, factory.count("compute"))
	assert.Equal(t, 2, factory.count("network"))
}

func TestPoolReloadStopsOnCancelledContext(t *testing.T) {
	pool, factory, _ := newTestPool(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	pool.Reload(ctx, "papa", []string{"compute"})
	assert.Equal(t, 0, factory.count("compute"))
	assert.Equal(t, []string{"compute"}, pool.Services("papa"))
}
