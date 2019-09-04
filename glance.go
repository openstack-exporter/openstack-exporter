package main

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"gopkg.in/niedbalski/goose.v3/client"
	"gopkg.in/niedbalski/goose.v3/glance"
)

type GlanceExporter struct {
	BaseOpenStackExporter
	Client *glance.Client
}

var defaultGlanceMetrics = []Metric{
	{Name: "images"},
}

func NewGlanceExporter(client client.AuthenticatingClient, prefix string, config *Cloud) (*GlanceExporter, error) {
	exporter := GlanceExporter{BaseOpenStackExporter{
		Name:                 "glance",
		Prefix:               prefix,
		Config:               config,
		AuthenticatingClient: client,
	}, glance.New(client)}

	for _, metric := range defaultGlanceMetrics {
		exporter.AddMetric(metric.Name, metric.Labels, nil)
	}

	return &exporter, nil
}

func (exporter *GlanceExporter) RefreshClient() error {
	log.Infoln("Refresh auth client, in case token has expired")
	if err := exporter.AuthenticatingClient.Authenticate(); err != nil {
		return fmt.Errorf("Error authenticating glance client: %s", err)
	}
	exporter.Client = glance.New(exporter.AuthenticatingClient)
	return nil
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

	log.Infoln("Fetching images list")
	images, err := exporter.Client.ListImagesV2()
	if err != nil {
		log.Errorf("%s", err)
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["images"],
		prometheus.GaugeValue, float64(len(images)))
}
