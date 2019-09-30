package exporters

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/domains"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/groups"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/projects"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/regions"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/users"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"sync"
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

func ListDomains(exporter *KeystoneExporter, wg *sync.WaitGroup, ch chan<- prometheus.Metric) {
	log.Infoln("Fetching domains information")
	var allDomains []domains.Domain

	allPagesDomain, err := domains.List(exporter.Client, domains.ListOpts{}).AllPages()
	if err != nil {
		log.Errorln(err)
		return
	}

	allDomains, err = domains.ExtractDomains(allPagesDomain)
	if err != nil {
		log.Errorln(err)
		return
	}
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["domains"],
		prometheus.GaugeValue, float64(len(allDomains)))

}

func ListProjects(exporter *KeystoneExporter, wg *sync.WaitGroup, ch chan<- prometheus.Metric) {
	log.Infoln("Fetching projects information")
	var allProjects []projects.Project

	allPagesProject, err := projects.List(exporter.Client, projects.ListOpts{}).AllPages()
	if err != nil {
		log.Errorln(err)
		return
	}

	allProjects, err = projects.ExtractProjects(allPagesProject)
	if err != nil {
		log.Errorln(err)
		return
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["projects"],
		prometheus.GaugeValue, float64(len(allProjects)))
}

func ListRegions(exporter *KeystoneExporter, wg *sync.WaitGroup, ch chan<- prometheus.Metric) {
	log.Infoln("Fetching regions information")
	var allRegions []regions.Region

	allPagesRegion, err := regions.List(exporter.Client, regions.ListOpts{}).AllPages()
	if err != nil {
		log.Errorln(err)
		return
	}

	allRegions, err = regions.ExtractRegions(allPagesRegion)
	if err != nil {
		log.Errorln(err)
		return
	}
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["regions"],
		prometheus.GaugeValue, float64(len(allRegions)))
}

func ListUsers(exporter *KeystoneExporter, wg *sync.WaitGroup, ch chan<- prometheus.Metric) {
	log.Infoln("Fetching users information")
	var allUsers []users.User

	allPagesUser, err := users.List(exporter.Client, users.ListOpts{}).AllPages()
	if err != nil {
		log.Errorln(err)
		return
	}

	allUsers, err = users.ExtractUsers(allPagesUser)
	if err != nil {
		log.Errorln(err)
	}
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["users"],
		prometheus.GaugeValue, float64(len(allUsers)))
}

func ListGroups(exporter *KeystoneExporter, wg *sync.WaitGroup, ch chan<- prometheus.Metric) {
	log.Infoln("Fetching groups information")
	var allGroups []groups.Group

	allPagesGroup, err := groups.List(exporter.Client, groups.ListOpts{}).AllPages()
	if err != nil {
		log.Errorln(err)
		return
	}

	allGroups, err = groups.ExtractGroups(allPagesGroup)
	if err != nil {
		log.Errorln(err)
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["groups"],
		prometheus.GaugeValue, float64(len(allGroups)))

}

type ListFunc func(exporter *KeystoneExporter, wg *sync.WaitGroup, ch chan<- prometheus.Metric)

func (exporter *KeystoneExporter) Collect(ch chan<- prometheus.Metric) {
	var listMethods = []ListFunc{ListProjects, ListDomains, ListRegions, ListUsers, ListGroups}
	wg := new(sync.WaitGroup)

	for _, method := range listMethods {
		go method(exporter, wg, ch)
	}

}
