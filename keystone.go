package main

import (
	"fmt"
	"github.com/gophercloud/gophercloud"
	domains2 "github.com/gophercloud/gophercloud/openstack/identity/v3/domains"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

type KeystoneExporter struct {
	BaseOpenStackExporter
}

var defaultKeystoneMetrics = []Metric{
	{Name: "domains"},
	{Name: "users"},
	{Name: "groups"},
	{Name: "projects"},
	{Name: "regions"},
}

func NewKeystoneExporter(client *gophercloud.ServiceClient, prefix string) (*KeystoneExporter, error) {
	exporter := KeystoneExporter{
		BaseOpenStackExporter{
			Name:   "identity",
			Prefix: prefix,
			Client: client,
		},
	}

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

func (exporter *KeystoneExporter) Collect(ch chan<- prometheus.Metric) {
	log.Infoln("Fetching domains information")
	domains := domains2.List(exporter.Client, domains2.ListOpts{})
	fmt.Println(domains)
	//if err != nil {
	//	log.Errorln(err)
	//}
	//ch <- prometheus.MustNewConstMetric(exporter.Metrics["domains"],
	//	prometheus.GaugeValue, float64(len(domains)))
	//
	//log.Infoln("Fetching users information")
	//users, err := exporter.Client.ListUsers()
	//if err != nil {
	//	log.Errorln(err)
	//}
	//ch <- prometheus.MustNewConstMetric(exporter.Metrics["users"],
	//	prometheus.GaugeValue, float64(len(users)))
	//
	//log.Infoln("Fetching projects information")
	//projects, err := exporter.Client.ListProjects()
	//if err != nil {
	//	log.Errorln(err)
	//}
	//ch <- prometheus.MustNewConstMetric(exporter.Metrics["projects"],
	//	prometheus.GaugeValue, float64(len(projects)))
	//
	//log.Infoln("Fetching groups information")
	//groups, err := exporter.Client.ListGroups()
	//if err != nil {
	//	log.Errorln(err)
	//}
	//ch <- prometheus.MustNewConstMetric(exporter.Metrics["groups"],
	//	prometheus.GaugeValue, float64(len(groups)))
	//
	//log.Infoln("Fetching regions information")
	//regions, err := exporter.Client.ListRegions()
	//if err != nil {
	//	log.Errorln(err)
	//}
	//ch <- prometheus.MustNewConstMetric(exporter.Metrics["regions"],
	//	prometheus.GaugeValue, float64(len(regions)))

}
