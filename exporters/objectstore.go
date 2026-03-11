package exporters

import (
	"log/slog"

	"github.com/gophercloud/gophercloud/openstack/objectstorage/v1/containers"
	"github.com/gophercloud/gophercloud/pagination"
	"github.com/prometheus/client_golang/prometheus"
)

const OBJECT_STORE_SERVICE string = "object_store"

type ObjectStoreExporter struct {
	BaseOpenStackExporter
}

var defaultObjectStoreMetrics = []Metric{
	{Name: "objects", Labels: []string{"container_name"}, Fn: ListContainers},
	{Name: "bytes", Labels: []string{"container_name"}, Fn: nil},
}

func NewObjectStoreExporter(config *ExporterConfig, logger *slog.Logger) (*ObjectStoreExporter, error) {
	exporter := ObjectStoreExporter{
		BaseOpenStackExporter{
			Name:           "object_store",
			ExporterConfig: *config,
			logger:         logger,
		},
	}

	for _, metric := range defaultObjectStoreMetrics {
		if exporter.isDeprecatedMetric(&metric) {
			continue
		}
		if !exporter.isSlowMetric(&metric) {
			labels := computeMetricLabels(OBJECT_STORE_SERVICE, metric, exporter.ExtraLabels)
			constLabels := computeConstantLabels(OBJECT_STORE_SERVICE, metric, exporter.ExtraLabels)
			exporter.AddMetric(metric.Name, metric.Fn, labels, metric.DeprecatedVersion, constLabels)
		}
	}

	return &exporter, nil
}

func ListContainers(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	objectsSpec := exporter.ExtraLabels.Extract(OBJECT_STORE_SERVICE, "objects")
	bytesSpec := exporter.ExtraLabels.Extract(OBJECT_STORE_SERVICE, "bytes")
	err := containers.List(exporter.Client, containers.ListOpts{Full: true}).EachPage(func(page pagination.Page) (bool, error) {
		containerList, err := containers.ExtractInfo(page)
		if err != nil {
			return false, err
		}

		for _, c := range containerList {
			ch <- prometheus.MustNewConstMetric(exporter.Metrics["objects"].Metric,
				prometheus.GaugeValue, float64(c.Count), append([]string{c.Name}, resolveExtraLabelValues(c, objectsSpec)...)...)
			ch <- prometheus.MustNewConstMetric(exporter.Metrics["bytes"].Metric,
				prometheus.GaugeValue, float64(c.Bytes), append([]string{c.Name}, resolveExtraLabelValues(c, bytesSpec)...)...)
		}
		return true, nil
	})

	if err != nil {
		return err
	}
	return nil
}
