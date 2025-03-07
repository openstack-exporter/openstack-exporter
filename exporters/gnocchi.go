package exporters

import (
	"github.com/go-kit/log"
	"github.com/gophercloud/gophercloud/pagination"
	"github.com/gophercloud/utils/gnocchi/metric/v1/metrics"
	"github.com/gophercloud/utils/gnocchi/metric/v1/status"
	"github.com/prometheus/client_golang/prometheus"
)

type GnocchiExporter struct {
	BaseOpenStackExporter
}

var defaultGnocchiMetrics = []Metric{
	{Name: "status_metricd_processors", Fn: getMetricStatus},
	{Name: "status_metric_having_measures_to_process", Fn: nil},
	{Name: "status_measures_to_process", Fn: nil},
	{Name: "total_metrics", Fn: ListAllMetrics},
}

func NewGnocchiExporter(config *ExporterConfig, logger log.Logger) (*GnocchiExporter, error) {
	exporter := GnocchiExporter{
		BaseOpenStackExporter{
			Name:           "gnocchi",
			ExporterConfig: *config,
			logger:         logger,
		},
	}
	for _, metric := range defaultGnocchiMetrics {
		if exporter.isDeprecatedMetric(&metric) {
			continue
		}
		if !exporter.isSlowMetric(&metric) {
			exporter.AddMetric(metric.Name, metric.Fn, metric.Labels, metric.DeprecatedVersion, nil)
		}
	}
	return &exporter, nil
}

func ListAllMetrics(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	// Use pagination to count metrics without storing all in memory
	// If you want you can set Limit for each page that by default set to max_limit value of gnocchi_api config
	// If you have not max_limit, it may be set to 1000
	var totalMetrics int
	pager := metrics.List(exporter.Client, metrics.ListOpts{})
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		metricList, err := metrics.ExtractMetrics(page)
		if err != nil {
			return false, err
		}
		totalMetrics += len(metricList)
		return true, nil
	})

	if err != nil {
		return err
	}
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["total_metrics"].Metric,
		prometheus.GaugeValue, float64(totalMetrics))

	return nil
}

func getMetricStatus(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	details := true
	metricStatus, err := status.Get(exporter.Client, status.GetOpts{Details: &details}).Extract()
	if err != nil {
		return err
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["status_metricd_processors"].Metric,
		prometheus.GaugeValue, float64(len(metricStatus.Metricd.Processors)))
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["status_metric_having_measures_to_process"].Metric,
		prometheus.GaugeValue, float64(metricStatus.Storage.Summary.Metrics))
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["status_measures_to_process"].Metric,
		prometheus.GaugeValue, float64(metricStatus.Storage.Summary.Measures))

	return nil
}
