package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/projects"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/domains"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/users"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/groups"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/regions"

)

type KeystoneExporter struct {
	BaseOpenStackExporter
	Client *gophercloud.ServiceClient
}

var defaultKeystoneMetrics = []Metric{
	{Name: "domains"},
	{Name: "users"},
	{Name: "groups"},
	{Name: "projects"},
	{Name: "regions"},
}

func NewKeystoneExporter(client *gophercloud.ProviderClient, prefix string, config *Cloud) (*KeystoneExporter, error) {
	identity, err := openstack.NewIdentityV3(client, gophercloud.EndpointOpts{
		Region:       "RegionOne",
		Availability: "internal",
	})
	if err != nil {
		log.Errorln(err)
	}

	exporter := KeystoneExporter{
		BaseOpenStackExporter{
			Name:                 "identity",
			Prefix:               prefix,
			Config:               config,
			AuthenticatedClient:  client,
		}, identity,
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

func (exporter *KeystoneExporter) RefreshClient() error {
	log.Infoln("Refresh auth client, in case token has expired")
	client, err := openstack.NewIdentityV3(exporter.AuthenticatedClient, gophercloud.EndpointOpts{
		Region:       "RegionOne",
		Availability: "internal",
	})
	if err != nil {
		log.Errorln(err)
	}
	exporter.Client = client
	return nil
}

func (exporter *KeystoneExporter) Collect(ch chan<- prometheus.Metric) {
	log.Infoln("Fetching projects information")
	var allProjects []projects.Project

	allPagesProject, err := projects.List(exporter.Client, projects.ListOpts{}).AllPages()
	allProjects, err = projects.ExtractProjects(allPagesProject)
	if err != nil {
		log.Errorln(err)
	}
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["projects"],
		prometheus.GaugeValue, float64(len(allProjects)))

	log.Infoln("Fetching domains information")
	var allDomains []domains.Domain

	allPagesDomain, err := domains.List(exporter.Client, domains.ListOpts{}).AllPages()
	allDomains, err = domains.ExtractDomains(allPagesDomain)
	if err != nil {
		log.Errorln(err)
	}
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["domains"],
		prometheus.GaugeValue, float64(len(allDomains)))

	log.Infoln("Fetching regions information")
	var allRegions []regions.Region

	allPagesRegion, err := regions.List(exporter.Client, regions.ListOpts{}).AllPages()
	allRegions, err = regions.ExtractRegions(allPagesRegion)
	if err != nil {
		log.Errorln(err)
	}
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["regions"],
		prometheus.GaugeValue, float64(len(allRegions)))

	log.Infoln("Fetching users information")
	var allUsers []users.User

	allPagesUser, err := users.List(exporter.Client, users.ListOpts{}).AllPages()
	allUsers, err = users.ExtractUsers(allPagesUser)
	if err != nil {
		log.Errorln(err)
	}
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["users"],
		prometheus.GaugeValue, float64(len(allUsers)))

	log.Infoln("Fetching groups information")
	var allGroups []groups.Group

	allPagesGroup, err := groups.List(exporter.Client, groups.ListOpts{}).AllPages()
	allGroups, err = groups.ExtractGroups(allPagesGroup)
	if err != nil {
		log.Errorln(err)
	}
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["groups"],
		prometheus.GaugeValue, float64(len(allGroups)))
}
