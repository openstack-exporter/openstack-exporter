package exporters

import (
	"github.com/gophercloud/gophercloud/openstack/objectstorage/v1/containers"
	"github.com/gophercloud/gophercloud/pagination"
	"github.com/prometheus/client_golang/prometheus"
)

type ObjectStoreExporter struct {
	BaseOpenStackExporter
}

var defaultObjectStoreMetrics = []Metric{
	{Name: "objects", Labels: []string{"container_name"}, Fn: ListContainers},
	{Name: "bytes", Labels: []string{"container_name"}, Fn: nil},
}

func NewObjectStoreExporter(config *ExporterConfig) (*ObjectStoreExporter, error) {
	exporter := ObjectStoreExporter{
		BaseOpenStackExporter{
			Name:           "object_store",
			ExporterConfig: *config,
		},
	}

	for _, metric := range defaultObjectStoreMetrics {
		exporter.AddMetric(metric.Name, metric.Fn, metric.Labels, nil)
	}

	return &exporter, nil
}

func ListContainers(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	err := containers.List(exporter.Client, containers.ListOpts{Full: true}).EachPage(func(page pagination.Page) (bool, error) {
		containerList, err := containers.ExtractInfo(page)
		if err != nil {
			return false, err
		}

		for _, c := range containerList {
			ch <- prometheus.MustNewConstMetric(exporter.Metrics["objects"].Metric,
				prometheus.GaugeValue, float64(c.Count), c.Name)
			ch <- prometheus.MustNewConstMetric(exporter.Metrics["bytes"].Metric,
				prometheus.GaugeValue, float64(c.Bytes), c.Name)
		}
		return true, nil
	})

	if err != nil {
		return err
	}
	return nil
}
