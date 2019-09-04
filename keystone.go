package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"gopkg.in/niedbalski/goose.v3/client"
	"gopkg.in/niedbalski/goose.v3/keystone"
)

type KeystoneExporter struct {
	BaseOpenStackExporter
	Client *keystone.Client
}

var defaultKeystoneMetrics = []Metric{
	{Name: "domains"},
	{Name: "users"},
	{Name: "groups"},
	{Name: "projects"},
	{Name: "regions"},
}

func NewKeystoneExporter(client client.AuthenticatingClient, prefix string, config *Cloud) (*KeystoneExporter, error) {
	exporter := KeystoneExporter{
		BaseOpenStackExporter{
			Name:                 "identity",
			Prefix:               prefix,
			Config:               config,
			AuthenticatingClient: client,
		}, keystone.New(client)}

	for _, metric := range defaultKeystoneMetrics {
		exporter.AddMetric(metric.Name, metric.Labels, nil)
	}

	return &exporter, nil
}

func (exporter *KeystoneExporter) Describe(ch chan<- *prometheus.Desc) {
	for _, metric := range exporter.Metrics {
		ch <- metric
	}
}

func (exporter *KeystoneExporter) RefreshClient() error {
	log.Infoln("Refresh auth client, in case token has expired")
	if err := exporter.AuthenticatingClient.Authenticate(); err != nil {
		log.Errorf("Error authenticating keystone client: %s", err)
		return err
	}
	exporter.Client = keystone.New(exporter.AuthenticatingClient)
	return nil
}

func (exporter *KeystoneExporter) Collect(ch chan<- prometheus.Metric) {
	log.Infoln("Fetching domains information")
	domains, err := exporter.Client.ListDomains()
	if err != nil {
		log.Errorln(err)
	}
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["domains"],
		prometheus.GaugeValue, float64(len(domains)))

	log.Infoln("Fetching users information")
	users, err := exporter.Client.ListUsers()
	if err != nil {
		log.Errorln(err)
	}
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["users"],
		prometheus.GaugeValue, float64(len(users)))

	log.Infoln("Fetching projects information")
	projects, err := exporter.Client.ListProjects()
	if err != nil {
		log.Errorln(err)
	}
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["projects"],
		prometheus.GaugeValue, float64(len(projects)))

	log.Infoln("Fetching groups information")
	groups, err := exporter.Client.ListGroups()
	if err != nil {
		log.Errorln(err)
	}
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["groups"],
		prometheus.GaugeValue, float64(len(groups)))

	log.Infoln("Fetching regions information")
	regions, err := exporter.Client.ListRegions()
	if err != nil {
		log.Errorln(err)
	}
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["regions"],
		prometheus.GaugeValue, float64(len(regions)))

}
