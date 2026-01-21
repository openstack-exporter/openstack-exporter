package cache

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/openstack-exporter/openstack-exporter/exporters"
	"github.com/openstack-exporter/openstack-exporter/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
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
	novaMetadataMapping *utils.LabelMappingFlag,
	uuidGenFunc func() (string, error),
	logger *slog.Logger,
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
	assert := assert.New(t)

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
	novaMetadataMapping := new(utils.LabelMappingFlag)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))

	err := CollectCache(
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
		novaMetadataMapping,
		nil,
		logger,
	)
	assert.NoError(err, "Collect cache failed")

	cloudCache, exists := cache.GetCloudCache(cloud)
	assert.True(exists, "Cloud cache was not set or retrieved properly")

	includeServices := []string{}
	for _, mf := range cloudCache.MetricFamilyCaches {
		includeServices = append(includeServices, mf.Service)
	}

	assert.Contains(includeServices, "service-a", "service-a should be included in the cache data")
	assert.NotContains(includeServices, "service-b", "service-b should not be included in the cache data")
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

	parser := expfmt.NewTextParser(model.LegacyValidation)
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

	parser := expfmt.NewTextParser(model.LegacyValidation)
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
