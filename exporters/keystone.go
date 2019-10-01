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
)

type KeystoneExporter struct {
	BaseOpenStackExporter
}

var defaultKeystoneMetrics = []Metric{
	{Name: "domains", Fn: ListDomains},
	{Name: "users", Fn: ListUsers},
	{Name: "groups", Fn: ListGroups},
	{Name: "projects", Fn: ListProjects},
	{Name: "regions", Fn: ListRegions},
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
		exporter.AddMetric(metric.Name, metric.Fn, metric.Labels, nil)
	}

	return &exporter, nil
}

func (exporter *KeystoneExporter) Describe(ch chan<- *prometheus.Desc) {
	for _, metric := range exporter.Metrics {
		ch <- metric.Metric
	}
}

func ListDomains(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) {

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
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["domains"].Metric,
		prometheus.GaugeValue, float64(len(allDomains)))

}

func ListProjects(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) {

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

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["projects"].Metric,
		prometheus.GaugeValue, float64(len(allProjects)))
}

func ListRegions(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) {

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
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["regions"].Metric,
		prometheus.GaugeValue, float64(len(allRegions)))
}

func ListUsers(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) {

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
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["users"].Metric,
		prometheus.GaugeValue, float64(len(allUsers)))
}

func ListGroups(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) {

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
		return
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["groups"].Metric,
		prometheus.GaugeValue, float64(len(allGroups)))

}

func (exporter *KeystoneExporter) Collect(ch chan<- prometheus.Metric) {
	exporter.CollectMetrics(ch)
}
