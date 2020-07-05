package exporters

import (
	"github.com/gophercloud/gophercloud/openstack/identity/v3/domains"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/groups"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/projects"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/regions"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/users"
	"github.com/prometheus/client_golang/prometheus"
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

func NewKeystoneExporter(config *ExporterConfig) (*KeystoneExporter, error) {
	exporter := KeystoneExporter{
		BaseOpenStackExporter{
			Name:           "identity",
			ExporterConfig: *config,
		},
	}

	for _, metric := range defaultKeystoneMetrics {
		exporter.AddMetric(metric.Name, metric.Fn, metric.Labels, nil)
	}

	return &exporter, nil
}

func ListDomains(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allDomains []domains.Domain

	allPagesDomain, err := domains.List(exporter.Client, domains.ListOpts{}).AllPages()
	if err != nil {
		return err
	}

	allDomains, err = domains.ExtractDomains(allPagesDomain)
	if err != nil {
		return err
	}
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["domains"].Metric,
		prometheus.GaugeValue, float64(len(allDomains)))

	return nil
}

func ListProjects(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allProjects []projects.Project

	allPagesProject, err := projects.List(exporter.Client, projects.ListOpts{}).AllPages()
	if err != nil {
		return err
	}

	allProjects, err = projects.ExtractProjects(allPagesProject)
	if err != nil {
		return err
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["projects"].Metric,
		prometheus.GaugeValue, float64(len(allProjects)))

	return nil
}

func ListRegions(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allRegions []regions.Region

	allPagesRegion, err := regions.List(exporter.Client, regions.ListOpts{}).AllPages()
	if err != nil {
		return err
	}

	allRegions, err = regions.ExtractRegions(allPagesRegion)
	if err != nil {
		return err
	}
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["regions"].Metric,
		prometheus.GaugeValue, float64(len(allRegions)))

	return nil
}

func ListUsers(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allUsers []users.User

	allPagesUser, err := users.List(exporter.Client, users.ListOpts{}).AllPages()
	if err != nil {
		return err
	}

	allUsers, err = users.ExtractUsers(allPagesUser)
	if err != nil {
		return err
	}
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["users"].Metric,
		prometheus.GaugeValue, float64(len(allUsers)))

	return nil
}

func ListGroups(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allGroups []groups.Group

	allPagesGroup, err := groups.List(exporter.Client, groups.ListOpts{}).AllPages()
	if err != nil {
		return err
	}

	allGroups, err = groups.ExtractGroups(allPagesGroup)
	if err != nil {
		return err
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["groups"].Metric,
		prometheus.GaugeValue, float64(len(allGroups)))

	return nil
}
