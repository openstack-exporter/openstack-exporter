package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"gopkg.in/niedbalski/goose.v3/cinder"
	"gopkg.in/niedbalski/goose.v3/client"
	"net/http"
	"net/url"
)

type CinderExporter struct {
	BaseOpenStackExporter
	Client *cinder.Client
}

var defaultCinderMetrics = []Metric{
	{Name: "volumes"},
	{Name: "snapshots"},
	{Name: "service_state", Labels: []string{"hostname", "service", "status", "zone"}},
}

func NewCinderExporter(client client.AuthenticatingClient, config *Cloud) (*CinderExporter, error) {
	endpoint := client.EndpointsForRegion(config.Region)["volumev3"]
	endpointUrl, err := url.Parse(endpoint)

	if err != nil {
		return nil, err
	}

	exporter := CinderExporter{BaseOpenStackExporter{
		Name:   "cinder",
		Config: config,
	}, cinder.NewClient(client.TenantId(), endpointUrl,
		cinder.SetAuthHeaderFn(client.Token,
			http.DefaultClient.Do),
	)}

	for _, metric := range defaultCinderMetrics {
		exporter.AddMetric(metric.Name, metric.Labels, nil)
	}

	return &exporter, nil
}

func (exporter *CinderExporter) Describe(ch chan<- *prometheus.Desc) {
	for _, metric := range exporter.GetMetrics() {
		ch <- metric
	}
}

func (exporter *CinderExporter) Collect(ch chan<- prometheus.Metric) {
	volumes, err := exporter.Client.GetVolumesSimple()
	if err != nil {
		log.Errorf("%s", err)
	}

	services, err := exporter.Client.GetServices()
	if err != nil {
		log.Errorf("%s", err)
	}

	for _, service := range services.Services {
		var state int = 0
		if service.State == "up" {
			state = 1
		}
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["service_state"],
			prometheus.CounterValue, float64(state), service.Host, service.Binary, service.Status, service.Zone)
	}

	ch <- prometheus.MustNewConstMetric(exporter.GetMetrics()["volumes"],
		prometheus.GaugeValue, float64(len(volumes.Volumes)))

	snapshots, err := exporter.Client.GetSnapshotsSimple()
	if err != nil {
		log.Errorf("%s", err)
	}

	ch <- prometheus.MustNewConstMetric(exporter.GetMetrics()["snapshots"],
		prometheus.GaugeValue, float64(len(snapshots.Snapshots)))
}
