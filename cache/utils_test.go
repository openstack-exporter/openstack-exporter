package cache

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/openstack-exporter/openstack-exporter/exporters"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/assert"
)

func mockEnableExporter(
	service,
	prefix,
	cloud string,
	disabledMetrics []string,
	endpointType string,
	collectTime bool,
	disableSlowMetrics bool,
	disableDeprecatedMetrics bool,
	disableCinderAgentUUID bool,
	domainID string,
	tenantID string,
	uuidGenFunc func() (string, error),
	logger log.Logger,
) (*exporters.OpenStackExporter, error) {
	var exporter exporters.OpenStackExporter = &mockOpenStackExporter{
		cnt: prometheus.NewCounter(prometheus.CounterOpts{Name: "c1", Help: "Help c1"}),
		gge: prometheus.NewGauge(prometheus.GaugeOpts{Name: "g1", Help: "Help g1"}),
	}
	return &exporter, nil
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
	cache := GetCache()
	defer newSingleCache()

	multiCloud := false
	services := make(map[string]*bool)
	serviceADisable := false
	serviceBDisable := true
	services["service-a"] = &serviceADisable
	services["service-b"] = &serviceBDisable
	prefix := "testPrefix"
	cloud := "testCloud"
	disabledMetrics := []string{}
	endpointType := "public"
	collectTime := true
	disableSlowMetrics := false
	disableDeprecatedMetrics := true
	disableCinderAgentUUID := false
	domainID := ""
	tenantID := ""
	logger := log.NewLogfmtLogger(os.Stdout)

	if err := CollectCache(
		mockEnableExporter,
		multiCloud,
		services,
		prefix,
		cloud,
		disabledMetrics,
		endpointType,
		collectTime,
		disableSlowMetrics,
		disableDeprecatedMetrics,
		disableCinderAgentUUID,
		domainID,
		tenantID,
		nil,
		logger,
	); err != nil {
		t.Errorf("Collect cache failed")
	}

	cloudCache, exists := cache.GetCloudCache(cloud)
	if !exists {
		t.Errorf("Cloud cache was not set or retrieved properly")
	}

	includeServices := []string{}
	for _, mf := range cloudCache.MetricFamilyCaches {
		includeServices = append(includeServices, mf.Service)
	}
	if !slices.Contains(includeServices, "service-a") {
		t.Errorf("service-a should be included in the cache data")
	}

	if slices.Contains(includeServices, "service-b") {
		t.Errorf("service-b should not be included in the cache data")
	}
}

func TestBufferFromCache(t *testing.T) {
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

	buf, err := BufferFromCache(cloudName, []string{serviceName}, log.NewLogfmtLogger(os.Stdout))
	if err != nil {
		t.Error(err)
	}

	parser := expfmt.TextParser{}
	metricFamilies, err := parser.TextToMetricFamilies(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Error(err)
	}
	for _, mf := range mfs {
		assert.Equal(t, mf.String(), metricFamilies[*mf.Name].String(), "The MetricFamily should be the same")
	}

}

func TestWriteCacheToResponse(t *testing.T) {
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
		if err := WriteCacheToResponse(
			w, r, cloudName, []string{serviceName}, log.NewLogfmtLogger(os.Stdout),
		); err != nil {
			t.Errorf("WriteCacheToResponse failed")
		}
	}
	handler := http.HandlerFunc(handlerFunc)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	parser := expfmt.TextParser{}
	metricFamilies, err := parser.TextToMetricFamilies(rr.Body)
	if err != nil {
		t.Error(err)
	}
	for _, mf := range mfs {
		assert.Equal(t, mf.String(), metricFamilies[*mf.Name].String(), "The MetricFamily should be the same")
	}
}

// TestFlushExpiredCloudCaches tests flushing of expired cloud caches.
func TestFlushExpiredCloudCaches(t *testing.T) {
	cache := GetCache()
	defer newSingleCache()
	cloudCache := NewCloudCache()
	cloudName := "expiredCloud"
	cache.SetCloudCache(cloudName, cloudCache)

	time.Sleep(1 * time.Nanosecond)
	FlushExpiredCloudCaches(1 * time.Nanosecond)

	if _, exists := cache.GetCloudCache(cloudName); exists {
		t.Errorf("Expired cloud cache was not flushed")
	}
}
