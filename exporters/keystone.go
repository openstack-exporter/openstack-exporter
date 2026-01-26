package exporters

import (
	"errors"
	"log/slog"
	"strconv"
	"strings"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
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
	{Name: "domain_info", Labels: []string{"description", "enabled", "id", "name"}},
	{Name: "users", Fn: ListUsers},
	{Name: "groups", Fn: ListGroups},
	{Name: "projects", Fn: ListProjects},
	{Name: "project_info", Labels: []string{"is_domain", "description", "domain_id", "enabled", "id", "name", "parent_id", "tags"}},
	{Name: "regions", Fn: ListRegions},
}

func NewKeystoneExporter(config *ExporterConfig, logger *slog.Logger) (*KeystoneExporter, error) {
	exporter := KeystoneExporter{
		BaseOpenStackExporter{
			Name:           "identity",
			ExporterConfig: *config,
			logger:         logger,
		},
	}

	for _, metric := range defaultKeystoneMetrics {
		if exporter.isDeprecatedMetric(&metric) {
			continue
		}
		if !exporter.isSlowMetric(&metric) {
			exporter.AddMetric(metric.Name, metric.Fn, metric.Labels, metric.DeprecatedVersion, nil)
		}
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
	if !exporter.MetricIsDisabled("domain_info") {
		for _, d := range allDomains {
			ch <- prometheus.MustNewConstMetric(exporter.Metrics["domain_info"].Metric,
				prometheus.GaugeValue, 1.0,
				d.Description, strconv.FormatBool(d.Enabled), d.ID, d.Name)
		}
	}
	return nil
}

func ListProjects(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allProjects []projects.Project

	allPagesProject, err := projects.List(exporter.Client, projects.ListOpts{DomainID: exporter.DomainID}).AllPages()
	if err != nil {
		return err
	}

	allProjects, err = projects.ExtractProjects(allPagesProject)
	if err != nil {
		return err
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["projects"].Metric,
		prometheus.GaugeValue, float64(len(allProjects)))
	if !exporter.MetricIsDisabled("project_info") {
		for _, p := range allProjects {
			ch <- prometheus.MustNewConstMetric(exporter.Metrics["project_info"].Metric,
				prometheus.GaugeValue, 1.0, strconv.FormatBool(p.IsDomain),
				p.Description, p.DomainID, strconv.FormatBool(p.Enabled), p.ID, p.Name,
				p.ParentID, strings.Join(p.Tags, ","))
		}
	}
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

	allPagesUser, err := users.List(exporter.Client, users.ListOpts{DomainID: exporter.DomainID}).AllPages()
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

	allPagesGroup, err := groups.List(exporter.Client, groups.ListOpts{DomainID: exporter.DomainID}).AllPages()
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

func MapProjectsName(exporter *BaseOpenStackExporter) (map[string]string, error) {
	var allProjects []projects.Project
	var eo gophercloud.EndpointOpts

	projectsMap := make(map[string]string)

	// We create a map of project ID to name to to add it to the data
	if v, ok := endpointOpts["identity"]; ok {
		eo = v
	} else if v, ok := endpointOpts["volume"]; ok {
		eo = v
	} else {
		return nil, errors.New("no EndpointOpts available to create Identity client")
	}

	c, err := openstack.NewIdentityV3(exporter.Client.ProviderClient, eo)
	if err != nil {
		return nil, err
	}

	allPagesProject, err := projects.List(c, projects.ListOpts{DomainID: exporter.DomainID}).AllPages()
	if err != nil {
		return nil, err
	}

	allProjects, err = projects.ExtractProjects(allPagesProject)
	if err != nil {
		return nil, err
	}

	for _, p := range allProjects {
		projectsMap[p.ID] = p.Name
	}

	return projectsMap, nil
}
