package exporters

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/version"
)

type CommonMetricsExporter struct {
	totalScrapes   prometheus.Counter
	scrapeDuration prometheus.Histogram
	scrapeErrors   prometheus.Counter
	buildInfo      prometheus.Metric
}

func NewCommonMetricsExporter(prefix string) *CommonMetricsExporter {
	totalScrapes := prometheus.NewCounter(prometheus.CounterOpts{
		Name: fmt.Sprintf("%s_exporter_scrapes_total", prefix),
		Help: "Total number of scrapes",
	})

	scrapeDuration := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    fmt.Sprintf("%s_exporter_scrape_duration_miliseconds", prefix),
		Help:    "Duration of scrapes",
		Buckets: []float64{500, 750, 1000, 5000, 10000, 30000},
	})

	scrapeErrors := prometheus.NewCounter(prometheus.CounterOpts{
		Name: fmt.Sprintf("%s_exporter_scrape_errors_total", prefix),
		Help: "Total number of scrape errors",
	})

	buildInfo := prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			fmt.Sprintf("%s_exporter_build_info", prefix),
			"A metric with a constant '1' value labeled by version, revision, branch, and goversion.",
			nil,
			prometheus.Labels{
				"version":   version.Version,
				"revision":  version.Revision,
				"branch":    version.Branch,
				"goversion": version.GoVersion,
			},
		),
		prometheus.GaugeValue, 1,
	)

	return &CommonMetricsExporter{
		totalScrapes:   totalScrapes,
		scrapeDuration: scrapeDuration,
		scrapeErrors:   scrapeErrors,
		buildInfo:      buildInfo,
	}
}

func (e *CommonMetricsExporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.totalScrapes.Desc()
	ch <- e.scrapeDuration.Desc()
	ch <- e.scrapeErrors.Desc()
	ch <- e.buildInfo.Desc()
}

func (e *CommonMetricsExporter) Collect(ch chan<- prometheus.Metric) {
	ch <- e.totalScrapes
	ch <- e.scrapeDuration
	ch <- e.scrapeErrors
	ch <- e.buildInfo
}

func (e *CommonMetricsExporter) MetricIsDisabled(name string) bool {
	return false
}

func (e *CommonMetricsExporter) TotalScrapes() prometheus.Counter {
	return e.totalScrapes
}

func (e *CommonMetricsExporter) ScrapeDuration() prometheus.Histogram {
	return e.scrapeDuration
}

func (e *CommonMetricsExporter) ScrapeErrors() prometheus.Counter {
	return e.scrapeErrors
}
