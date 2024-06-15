package openstack

import (
	"github.com/go-kit/log"
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
	var allMetrics []metrics.Metric
	allPagesMetrics, err := metrics.List(exporter.Client, metrics.ListOpts{}).AllPages()
	if err != nil {
		return err
	}
	allMetrics, err = metrics.ExtractMetrics(allPagesMetrics)
	if err != nil {
		return err
	}
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["total_metrics"].Metric,
		prometheus.GaugeValue, float64(len(allMetrics)))

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
