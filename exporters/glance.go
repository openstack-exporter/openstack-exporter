package exporters

import (
	"strconv"

	"log/slog"

	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
	"github.com/prometheus/client_golang/prometheus"
)

const GLANCE_SERVICE string = "glance"

type GlanceExporter struct {
	BaseOpenStackExporter
}

var defaultGlanceMetrics = []Metric{
	{Name: "images", Fn: ListImages},
	{Name: "image_bytes", Labels: []string{"id", "name", "tenant_id"}, Fn: ListImageProperties, Slow: true},
	{Name: "image_created_at", Labels: []string{"id", "name", "tenant_id", "visibility", "hidden", "status"}, Slow: true},
}

func NewGlanceExporter(config *ExporterConfig, logger *slog.Logger) (*GlanceExporter, error) {
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
			labels := computeMetricLabels(GLANCE_SERVICE, metric, exporter.ExtraLabels)
			constLabels := computeConstantLabels(GLANCE_SERVICE, metric, exporter.ExtraLabels)
			exporter.AddMetric(metric.Name, metric.Fn, labels, metric.DeprecatedVersion, constLabels)
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

	imageBytesSpec := exporter.ExtraLabels.Extract(GLANCE_SERVICE, "image_bytes")
	imageCreatedSpec := exporter.ExtraLabels.Extract(GLANCE_SERVICE, "image_created_at")
	for _, image := range allImages {
		extraBytes := resolveExtraLabelValues(image, imageBytesSpec)
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["image_bytes"].Metric,
			prometheus.GaugeValue, float64(image.SizeBytes), append([]string{image.ID, image.Name, image.Owner}, extraBytes...)...)
		extraCreated := resolveExtraLabelValues(image, imageCreatedSpec)
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["image_created_at"].Metric,
			prometheus.GaugeValue, float64(image.CreatedAt.Unix()), append([]string{image.ID, image.Name,
				image.Owner, string(image.Visibility), strconv.FormatBool(image.Hidden), string(image.Status)}, extraCreated...)...)

	}

	return nil
}
