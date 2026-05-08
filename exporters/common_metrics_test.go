package exporters

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestCommonMetricsExporter(t *testing.T) {
	exporter := NewCommonMetricsExporter("test")

	exporter.TotalScrapes().Inc()
	exporter.ScrapeErrors().Add(2)
	exporter.ScrapeDuration().Observe(1234)

	// Register to a test registry
	reg := prometheus.NewRegistry()
	err := reg.Register(exporter)
	assert.NoError(t, err)

	metrics, err := reg.Gather()
	assert.NoError(t, err)

	metricsMap := map[string]bool{
		"test_exporter_scrapes_total":               false,
		"test_exporter_scrape_errors_total":         false,
		"test_exporter_scrape_duration_miliseconds": false,
		"test_exporter_build_info":                  false,
	}

	for _, mf := range metrics {
		if _, ok := metricsMap[mf.GetName()]; ok {
			metricsMap[mf.GetName()] = true
		}
	}

	for name, found := range metricsMap {
		assert.True(t, found, "Expected to find metric: %s", name)
	}
}

func TestCommonMetricsExporterScrapeCounters(t *testing.T) {
	exporter := NewCommonMetricsExporter("unit")

	exporter.TotalScrapes().Inc()
	exporter.ScrapeErrors().Add(5)
	exporter.ScrapeDuration().Observe(1000)

	assert.Equal(t, float64(1), testutil.ToFloat64(exporter.TotalScrapes()))
	assert.Equal(t, float64(5), testutil.ToFloat64(exporter.ScrapeErrors()))
}
