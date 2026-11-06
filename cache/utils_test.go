package cache

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/openstack-exporter/openstack-exporter/exporters"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
)

// mockSource binds a mock exporter for service-a only and reports readiness
// for every requested service.
type mockSource struct{}

func (mockSource) Collectors(_ context.Context, cloud string, services []string) exporters.Bound {
	var exporter exporters.OpenStackExporter = &mockOpenStackExporter{
		cnt: prometheus.NewCounter(prometheus.CounterOpts{Name: "c1", Help: "Help c1"}),
		gge: prometheus.NewGauge(prometheus.GaugeOpts{Name: "g1", Help: "Help g1"}),
	}
	ready := prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "openstack_exporter_collector_ready", Help: "ready"}, []string{"cloud", "service"})
	bound := exporters.Bound{Exporters: map[string]exporters.OpenStackExporter{}, Ready: ready}
	for _, service := range services {
		if service == "service-a" {
			bound.Exporters[service] = exporter
			ready.WithLabelValues(cloud, service).Set(1)
		} else {
			ready.WithLabelValues(cloud, service).Set(0)
		}
	}
	return bound
}

// MockOpenStackExporter is a mock of OpenStackExporter interface
type mockOpenStackExporter struct {
	cnt prometheus.Counter
	gge prometheus.Gauge
}

func (m *mockOpenStackExporter) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(m, ch)
}

func (m *mockOpenStackExporter) Collect(ch chan<- prometheus.Metric) {
	ch <- m.cnt
	ch <- m.gge
}

func (m *mockOpenStackExporter) GetName() string {
	return "MockOpenStackExporter"
}

func (m *mockOpenStackExporter) AddMetric(name string, fn exporters.ListFunc, labels []string, deprecatedVersion string, constLabels prometheus.Labels) {
}

func (m *mockOpenStackExporter) MetricIsDisabled(name string) bool {
	return false
}

func TestCollectCache(t *testing.T) {
	assert := assert.New(t)

	cache := GetCache()
	defer newSingleCache()

	multiCloud := false
	services := []string{"service-a", "service-b"}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))

	err := CollectCache(context.Background(), mockSource{}, multiCloud, services, "testCloud", logger)
	assert.NoError(err, "Collect cache failed")

	cloudCache, exists := cache.GetCloudCache("testCloud")
	assert.True(exists, "Cloud cache was not set or retrieved properly")

	includeServices := []string{}
	for _, mf := range cloudCache.MetricFamilyCaches {
		includeServices = append(includeServices, mf.Service)
	}

	assert.Contains(includeServices, "service-a", "service-a should be included in the cache data")
	assert.NotContains(includeServices, "service-b", "service-b has no bound collector and must not be in the cache data")
	assert.Contains(includeServices, "", "readiness metrics are stored without a service")

	// A read for service-b alone still carries the readiness metrics.
	buf, err := BufferFromCache("testCloud", []string{"service-b"}, logger)
	assert.NoError(err)
	assert.Contains(buf.String(), `openstack_exporter_collector_ready{cloud="testCloud",service="service-b"} 0`)
	assert.NotContains(buf.String(), "c1")
}

func TestBufferFromCache(t *testing.T) {
	assert := assert.New(t)

	cache := GetCache()
	defer newSingleCache()
	cloudName := "testCloud"
	serviceName := "testService"

	registry := prometheus.NewPedanticRegistry()
	collector := &mockOpenStackExporter{
		cnt: prometheus.NewCounter(prometheus.CounterOpts{Name: "c1", Help: "Help c1"}),
		gge: prometheus.NewGauge(prometheus.GaugeOpts{Name: "g1", Help: "Help g1"}),
	}
	registry.MustRegister(collector)

	cloudCache := NewCloudCache()

	mfs, _ := registry.Gather()
	for _, mf := range mfs {
		cloudCache.SetMetricFamilyCache(
			*mf.Name, MetricFamilyCache{MF: mf, Service: serviceName},
		)
	}
	cache.SetCloudCache(cloudName, cloudCache)

	buf, err := BufferFromCache(cloudName, []string{serviceName}, slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{})))
	assert.NoError(err)

	parser := expfmt.NewTextParser(model.UTF8Validation)
	metricFamilies, err := parser.TextToMetricFamilies(bytes.NewReader(buf.Bytes()))
	assert.NoError(err)

	for _, mf := range mfs {
		mf2 := metricFamilies[*mf.Name]
		assert.Equal(mf.Name, mf2.Name, "The MetricName should be the same")
		assert.Equal(mf.Type, mf2.Type, "The MetricType should be the same")
		assert.Equal(mf.Help, mf2.Help, "The MetricHelp should be the same")
		assert.Equal(mf.Unit, mf2.Unit, "The MetricUnit should be the same")
	}
}

func TestWriteCacheToResponse(t *testing.T) {
	assert := assert.New(t)

	cache := GetCache()
	defer newSingleCache()
	cloudName := "testCloud"
	serviceName := "testService"

	registry := prometheus.NewPedanticRegistry()
	collector := &mockOpenStackExporter{
		cnt: prometheus.NewCounter(prometheus.CounterOpts{Name: "c1", Help: "Help c1"}),
		gge: prometheus.NewGauge(prometheus.GaugeOpts{Name: "g1", Help: "Help g1"}),
	}
	registry.MustRegister(collector)

	mfs, _ := registry.Gather()
	cloudCache := NewCloudCache()
	for _, mf := range mfs {
		cloudCache.SetMetricFamilyCache(
			*mf.Name, MetricFamilyCache{MF: mf, Service: serviceName},
		)
	}
	cache.SetCloudCache(cloudName, cloudCache)

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handlerFunc := func(w http.ResponseWriter, r *http.Request) {
		err := WriteCacheToResponse(w, r, cloudName, []string{serviceName}, slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{})))
		assert.NoError(err, "WriteCacheToResponse failed")
	}
	handler := http.HandlerFunc(handlerFunc)
	handler.ServeHTTP(rr, req)

	assert.Equal(rr.Code, http.StatusOK, "handler returned wrong status code")
	assert.NotEmpty(rr.Header().Get("Content-Type"), "missing Content-Type header")

	parser := expfmt.NewTextParser(model.UTF8Validation)
	metricFamilies, err := parser.TextToMetricFamilies(rr.Body)
	assert.NoError(err)

	for _, mf := range mfs {
		mf2 := metricFamilies[*mf.Name]
		assert.Equal(mf.Name, mf2.Name, "The MetricName should be the same")
		assert.Equal(mf.Type, mf2.Type, "The MetricType should be the same")
		assert.Equal(mf.Help, mf2.Help, "The MetricHelp should be the same")
		assert.Equal(mf.Unit, mf2.Unit, "The MetricUnit should be the same")
	}
}

// TestWriteCacheToResponseProtobuf checks that the cached metrics are encoded
// in the negotiated format when the scraper asks for protobuf.
func TestWriteCacheToResponseProtobuf(t *testing.T) {
	assert := assert.New(t)

	cache := GetCache()
	defer newSingleCache()
	cloudName := "testCloud"
	serviceName := "testService"

	registry := prometheus.NewPedanticRegistry()
	collector := &mockOpenStackExporter{
		cnt: prometheus.NewCounter(prometheus.CounterOpts{Name: "c1", Help: "Help c1"}),
		gge: prometheus.NewGauge(prometheus.GaugeOpts{Name: "g1", Help: "Help g1"}),
	}
	registry.MustRegister(collector)

	mfs, _ := registry.Gather()
	cloudCache := NewCloudCache()
	for _, mf := range mfs {
		cloudCache.SetMetricFamilyCache(
			*mf.Name, MetricFamilyCache{MF: mf, Service: serviceName},
		)
	}
	cache.SetCloudCache(cloudName, cloudCache)

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	// Accept header sent by Prometheus when protobuf scraping is enabled.
	req.Header.Set("Accept", "application/vnd.google.protobuf;proto=io.prometheus.client.MetricFamily;encoding=delimited;q=0.6,text/plain;version=0.0.4;q=0.3")

	rr := httptest.NewRecorder()
	err = WriteCacheToResponse(rr, req, cloudName, []string{serviceName}, slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{})))
	assert.NoError(err, "WriteCacheToResponse failed")
	assert.Equal(http.StatusOK, rr.Code, "handler returned wrong status code")

	format := expfmt.ResponseFormat(rr.Result().Header)
	assert.Equal(expfmt.TypeProtoDelim, format.FormatType(), "expected protobuf Content-Type")

	dec := expfmt.NewDecoder(rr.Body, format)
	got := map[string]*dto.MetricFamily{}
	for {
		mf := &dto.MetricFamily{}
		if err := dec.Decode(mf); err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("failed to decode body as %s: %v", format, err)
		}
		got[mf.GetName()] = mf
	}

	assert.Len(got, len(mfs))
	for _, mf := range mfs {
		mf2, ok := got[mf.GetName()]
		if assert.True(ok, "missing metric family %s", mf.GetName()) {
			assert.Equal(mf.GetType(), mf2.GetType(), "The MetricType should be the same")
			assert.Equal(mf.GetHelp(), mf2.GetHelp(), "The MetricHelp should be the same")
		}
	}
}

// TestFlushExpiredCloudCaches tests flushing of expired cloud caches.
func TestFlushExpiredCloudCaches(t *testing.T) {
	assert := assert.New(t)

	cache := GetCache()
	defer newSingleCache()
	cloudCache := NewCloudCache()
	cloudName := "expiredCloud"
	cache.SetCloudCache(cloudName, cloudCache)

	time.Sleep(1 * time.Nanosecond)
	FlushExpiredCloudCaches(1 * time.Nanosecond)

	_, exists := cache.GetCloudCache(cloudName)
	assert.False(exists, "Expired cloud cache was not flushed")
}
