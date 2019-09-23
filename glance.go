package main

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

type GlanceExporter struct {
	BaseOpenStackExporter
	Client *gophercloud.ServiceClient
}

var defaultGlanceMetrics = []Metric{
	{Name: "images"},
}

func NewGlanceExporter(client *gophercloud.ProviderClient, prefix string, config *Cloud) (*GlanceExporter, error) {
	image, err := openstack.NewImageServiceV2(client, gophercloud.EndpointOpts{
		Region:       "RegionOne",
		Availability: "internal",
	})
	if err != nil {
		log.Errorln(err)
	}

	exporter := GlanceExporter{BaseOpenStackExporter{
			Name:                 "glance",
			Prefix:               prefix,
			Config:               config,
			AuthenticatedClient: client,
		}, image,
	}

	for _, metric := range defaultGlanceMetrics {
		exporter.AddMetric(metric.Name, metric.Labels, nil)
	}

	return &exporter, nil
}

func (exporter *GlanceExporter) RefreshClient() error {
	log.Infoln("Refresh auth client, in case token has expired")
	client, err := openstack.NewImageServiceV2(exporter.AuthenticatedClient, gophercloud.EndpointOpts{
		Region:       "RegionOne",
		Availability: "internal",
	})
	if err != nil {
		log.Errorln(err)
	}
	exporter.Client = client
	return nil
}

func (exporter *GlanceExporter) Describe(ch chan<- *prometheus.Desc) {
	for _, metric := range exporter.Metrics {
		ch <- metric
	}
}
func (exporter *GlanceExporter) Collect(ch chan<- prometheus.Metric) {
	var allImages []images.Image

	allPagesImage, err := images.List(exporter.Client, images.ListOpts{}).AllPages()
	allImages, err = images.ExtractImages(allPagesImage)
	if err != nil {
		log.Errorln(err)
	}
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["images"],
		prometheus.GaugeValue, float64(len(allImages)))
}