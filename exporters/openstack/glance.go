package openstack

import (
	"strconv"

	"github.com/go-kit/log"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
	"github.com/prometheus/client_golang/prometheus"
)

type GlanceExporter struct {
	BaseOpenStackExporter
}

var defaultGlanceMetrics = []Metric{
	{Name: "images", Fn: ListImages},
	{Name: "image_bytes", Labels: []string{"id", "name", "tenant_id"}, Fn: ListImageProperties, Slow: true},
	{Name: "image_created_at", Labels: []string{"id", "name", "tenant_id", "visibility", "hidden", "status"}, Slow: true},
}

func NewGlanceExporter(config *ExporterConfig, logger log.Logger) (*GlanceExporter, error) {
	exporter := GlanceExporter{
		BaseOpenStackExporter{
			Name:           "glance",
			ExporterConfig: *config,
			logger:         logger,
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

func getAllImages(exporter *BaseOpenStackExporter) ([]images.Image, error) {
	var allImages []images.Image

	allPagesImage, err := images.List(exporter.Client, images.ListOpts{}).AllPages()
	if err != nil {
		return nil, err
	}

	if allImages, err = images.ExtractImages(allPagesImage); err != nil {
		return nil, err
	}

	return allImages, nil
}

func ListImages(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	allImages, err := getAllImages(exporter)
	if err != nil {
		return err
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["images"].Metric,
		prometheus.GaugeValue, float64(len(allImages)))

	return nil
}

func ListImageProperties(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	// Image size and created at metrics
	allImages, err := getAllImages(exporter)
	if err != nil {
		return err
	}

	for _, image := range allImages {
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["image_bytes"].Metric,
			prometheus.GaugeValue, float64(image.SizeBytes), image.ID, image.Name,
			image.Owner)
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["image_created_at"].Metric,
			prometheus.GaugeValue, float64(image.CreatedAt.Unix()), image.ID, image.Name,
			image.Owner, string(image.Visibility), strconv.FormatBool(image.Hidden), string(image.Status))

	}

	return nil
}
