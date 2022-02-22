package exporters

import (
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
	"github.com/prometheus/client_golang/prometheus"
)

type GlanceExporter struct {
	BaseOpenStackExporter
}

var defaultGlanceMetrics = []Metric{
	{Name: "images", Fn: ListImages},
	{Name: "image_bytes", Labels: []string{"id", "name", "tenant_id"}, Fn: nil, Slow: true},
}

func NewGlanceExporter(config *ExporterConfig) (*GlanceExporter, error) {
	exporter := GlanceExporter{
		BaseOpenStackExporter{
			Name:           "glance",
			ExporterConfig: *config,
		},
	}

	for _, metric := range defaultGlanceMetrics {
		if exporter.isDeprecatedMetric(&metric) {
			continue
		}
		if !exporter.isSlowMetric(&metric) {
			exporter.AddMetric(metric.Name, metric.Fn, metric.Labels, metric.DeprecatedVersion, nil)
		}
	}

	return &exporter, nil
}

func ListImages(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allImages []images.Image

	allPagesImage, err := images.List(exporter.Client, images.ListOpts{}).AllPages()
	if err != nil {
		return err
	}

	if allImages, err = images.ExtractImages(allPagesImage); err != nil {
		return err
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["images"].Metric,
		prometheus.GaugeValue, float64(len(allImages)))

	// Image size metrics
	for _, image := range allImages {
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["image_bytes"].Metric,
			prometheus.GaugeValue, float64(image.SizeBytes), image.ID, image.Name,
			image.Owner)
	}

	return nil
}
