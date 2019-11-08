package exporters

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

type GlanceExporter struct {
	BaseOpenStackExporter
}

var defaultGlanceMetrics = []Metric{
	{Name: "images", Fn: ListImages},
}

func NewGlanceExporter(client *gophercloud.ServiceClient, prefix string, disabledMetrics []string) (*GlanceExporter, error) {
	exporter := GlanceExporter{
		BaseOpenStackExporter{
			Name:            "glance",
			Prefix:          prefix,
			Client:          client,
			DisabledMetrics: disabledMetrics,
		},
	}

	for _, metric := range defaultGlanceMetrics {
		exporter.AddMetric(metric.Name, metric.Fn, metric.Labels, nil)
	}

	return &exporter, nil
}

func (exporter *GlanceExporter) Describe(ch chan<- *prometheus.Desc) {
	for _, metric := range exporter.Metrics {
		ch <- metric.Metric
	}
}
func (exporter *GlanceExporter) Collect(ch chan<- prometheus.Metric) {
	exporter.CollectMetrics(ch)
}

func ListImages(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) {

	log.Infoln("Fetching images list")
	var allImages []images.Image

	allPagesImage, err := images.List(exporter.Client, images.ListOpts{}).AllPages()
	if err != nil {
		log.Errorln(err)
		return
	}

	if allImages, err = images.ExtractImages(allPagesImage); err != nil {
		log.Errorln(err)
		return
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["images"].Metric,
		prometheus.GaugeValue, float64(len(allImages)))
}
