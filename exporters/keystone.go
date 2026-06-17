package exporters

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/domains"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/groups"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/projects"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/regions"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/users"
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	RegisterTypedExporter("identity", NewKeystoneExporter)
}

type KeystoneExporter struct {
	BaseOpenStackExporter
	sched Schedule
	descs keystoneDescs
}

type keystoneDescs struct {
	Domains     *prometheus.Desc `metric:"domains"`
	DomainInfo  *prometheus.Desc `metric:"domain_info"   labels:"description,enabled,id,name"`
	Users       *prometheus.Desc `metric:"users"`
	Groups      *prometheus.Desc `metric:"groups"`
	Projects    *prometheus.Desc `metric:"projects"`
	ProjectInfo *prometheus.Desc `metric:"project_info"  labels:"is_domain,description,domain_id,enabled,id,name,parent_id,tags"`
	Regions     *prometheus.Desc `metric:"regions"`
}

type keystoneScrape struct {
	allDomains  []domains.Domain
	allUsers    []users.User
	allGroups   []groups.Group
	allProjects []projects.Project
	allRegions  []regions.Region
}

var keystoneGraph = Graph[*KeystoneExporter, keystoneScrape]{
	Sources: []Source[*KeystoneExporter, keystoneScrape]{
		{Name: "domains", Fetch: (*KeystoneExporter).fetchDomains},
		{Name: "users", Fetch: (*KeystoneExporter).fetchUsers},
		{Name: "groups", Fetch: (*KeystoneExporter).fetchGroups},
		{Name: "projects", Fetch: (*KeystoneExporter).fetchProjects},
		{Name: "regions", Fetch: (*KeystoneExporter).fetchRegions},
	},
	Emitters: []Emitter[*KeystoneExporter, keystoneScrape]{
		{Name: "domains", Metrics: []string{"domains", "domain_info"}, Sources: []string{"domains"}, Emit: (*KeystoneExporter).emitDomains},
		{Name: "users", Metrics: []string{"users"}, Sources: []string{"users"}, Emit: (*KeystoneExporter).emitUsers},
		{Name: "groups", Metrics: []string{"groups"}, Sources: []string{"groups"}, Emit: (*KeystoneExporter).emitGroups},
		{Name: "projects", Metrics: []string{"projects", "project_info"}, Sources: []string{"projects"}, Emit: (*KeystoneExporter).emitProjects},
		{Name: "regions", Metrics: []string{"regions"}, Sources: []string{"regions"}, Emit: (*KeystoneExporter).emitRegions},
	},
}

func NewKeystoneExporter(config *ExporterConfig, logger *slog.Logger) (*KeystoneExporter, error) {
	e := &KeystoneExporter{
		BaseOpenStackExporter: BaseOpenStackExporter{
			Name:           "identity",
			ExporterConfig: *config,
			logger:         logger,
		},
	}
	e.RegisterAndFillDescs(&e.descs)
	sched, err := keystoneGraph.PruneSchedule(&e.BaseOpenStackExporter)
	if err != nil {
		return nil, err
	}
	e.sched = sched
	keystoneGraph.LogDAG(&e.BaseOpenStackExporter, e.sched)
	return e, nil
}

func (e *KeystoneExporter) Collect(ch chan<- prometheus.Metric) {
	e.RunCollect(ch, e.sched, func(ch chan<- prometheus.Metric) int {
		s := new(keystoneScrape)
		return runSchedule(e, &e.BaseOpenStackExporter, &keystoneGraph, e.sched, s, ch)
	})
}

func (e *KeystoneExporter) fetchDomains(ctx context.Context, s *keystoneScrape) error {
	allPages, err := domains.List(e.ClientV2, domains.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}
	s.allDomains, err = domains.ExtractDomains(allPages)
	return err
}

func (e *KeystoneExporter) fetchUsers(ctx context.Context, s *keystoneScrape) error {
	allPages, err := users.List(e.ClientV2, users.ListOpts{DomainID: e.DomainID}).AllPages(ctx)
	if err != nil {
		return err
	}
	s.allUsers, err = users.ExtractUsers(allPages)
	return err
}

func (e *KeystoneExporter) fetchGroups(ctx context.Context, s *keystoneScrape) error {
	allPages, err := groups.List(e.ClientV2, groups.ListOpts{DomainID: e.DomainID}).AllPages(ctx)
	if err != nil {
		return err
	}
	s.allGroups, err = groups.ExtractGroups(allPages)
	return err
}

func (e *KeystoneExporter) fetchProjects(ctx context.Context, s *keystoneScrape) error {
	var err error
	s.allProjects, err = GetProjects(ctx, &e.BaseOpenStackExporter)
	return err
}

func (e *KeystoneExporter) fetchRegions(ctx context.Context, s *keystoneScrape) error {
	allPages, err := regions.List(e.ClientV2, regions.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}
	s.allRegions, err = regions.ExtractRegions(allPages)
	return err
}

func (e *KeystoneExporter) emitDomains(ctx context.Context, s *keystoneScrape, ch chan<- prometheus.Metric) error {
	emitGauge(ch, e.descs.Domains, float64(len(s.allDomains)))
	for _, d := range s.allDomains {
		emitGauge(ch, e.descs.DomainInfo, 1.0, d.Description, strconv.FormatBool(d.Enabled), d.ID, d.Name)
	}
	return nil
}

func (e *KeystoneExporter) emitUsers(ctx context.Context, s *keystoneScrape, ch chan<- prometheus.Metric) error {
	emitGauge(ch, e.descs.Users, float64(len(s.allUsers)))
	return nil
}

func (e *KeystoneExporter) emitGroups(ctx context.Context, s *keystoneScrape, ch chan<- prometheus.Metric) error {
	emitGauge(ch, e.descs.Groups, float64(len(s.allGroups)))
	return nil
}

func (e *KeystoneExporter) emitProjects(ctx context.Context, s *keystoneScrape, ch chan<- prometheus.Metric) error {
	emitGauge(ch, e.descs.Projects, float64(len(s.allProjects)))
	for _, p := range s.allProjects {
		emitGauge(ch, e.descs.ProjectInfo, 1.0, strconv.FormatBool(p.IsDomain),
			p.Description, p.DomainID, strconv.FormatBool(p.Enabled), p.ID, p.Name,
			p.ParentID, strings.Join(p.Tags, ","))
	}
	return nil
}

func (e *KeystoneExporter) emitRegions(ctx context.Context, s *keystoneScrape, ch chan<- prometheus.Metric) error {
	emitGauge(ch, e.descs.Regions, float64(len(s.allRegions)))
	return nil
}

func newIdentityV3ClientV2FromExporter(exporter *BaseOpenStackExporter, fallbackServiceName string) (*gophercloud.ServiceClient, error) {
	var eo gophercloud.EndpointOpts

	if v, ok := endpointOptsV2["identity"]; ok {
		eo = v
	} else if v, ok := endpointOptsV2[fallbackServiceName]; ok {
		eo = v
	} else {
		return nil, errors.New("no EndpointOpts available to create Identity client")
	}

	cli, err := openstack.NewIdentityV3(exporter.ClientV2.ProviderClient, eo)
	if err != nil {
		return nil, err
	}

	return cli, nil
}
