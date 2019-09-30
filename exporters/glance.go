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
	{Name: "images"},
}

func NewGlanceExporter(client *gophercloud.ServiceClient, prefix string) (*GlanceExporter, error) {
	exporter := GlanceExporter{
		BaseOpenStackExporter{
			Name:   "glance",
			Prefix: prefix,
			Client: client,
		},
	}

	for _, metric := range defaultGlanceMetrics {
		exporter.AddMetric(metric.Name, metric.Labels, nil)
	}

	return &exporter, nil
}

func (exporter *GlanceExporter) Describe(ch chan<- *prometheus.Desc) {
	for _, metric := range exporter.Metrics {
		ch <- metric
	}
}
func (exporter *GlanceExporter) Collect(ch chan<- prometheus.Metric) {
	if err := exporter.RefreshClient(); err != nil {
		log.Error(err)
		return
	}

	var allImages []images.Image

	log.Infoln("Fetching images list")
	allPagesImage, err := images.List(exporter.Client, images.ListOpts{}).AllPages()
	if err != nil {
		log.Errorln(err)
		return
	}

	if allImages, err = images.ExtractImages(allPagesImage); err != nil {
		log.Errorln(err)
		return
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["images"],
		prometheus.GaugeValue, float64(len(allImages)))
}
