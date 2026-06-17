package exporters

import (
	"context"
	"log/slog"

	"github.com/gophercloud/gophercloud/v2/openstack/placement/v1/resourceproviders"
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	RegisterTypedExporter("placement", NewPlacementExporter)
}

type PlacementExporter struct {
	BaseOpenStackExporter
	sched Schedule
	descs placementDescs
}

type placementDescs struct {
	ResourceTotal               *prometheus.Desc `metric:"resource_total"                labels:"hostname,resourcetype"`
	ResourceAllocationRatio     *prometheus.Desc `metric:"resource_allocation_ratio"     labels:"hostname,resourcetype"`
	ResourceGeneration          *prometheus.Desc `metric:"resource_generation"           labels:"hostname,resourcetype"`
	ResourceReserved            *prometheus.Desc `metric:"resource_reserved"             labels:"hostname,resourcetype"`
	ResourceUsage               *prometheus.Desc `metric:"resource_usage"                labels:"hostname,resourcetype"`
	ResourceProviderAllocations *prometheus.Desc `metric:"resource_provider_allocations" labels:"hostname,uuid,resourcetype"`
}

type placementProviderEntry struct {
	provider    resourceproviders.ResourceProvider
	inventories *resourceproviders.ResourceProviderInventories
	usages      *resourceproviders.ResourceProviderUsage
	allocations *resourceproviders.ResourceProviderAllocations
}

type placementScrape struct {
	providers []placementProviderEntry
}

var placementGraph = Graph[*PlacementExporter, placementScrape]{
	Sources: []Source[*PlacementExporter, placementScrape]{
		{Name: "providers", Fetch: (*PlacementExporter).fetchProviders},
		{Name: "inventories", DependsOn: []string{"providers"}, Fetch: (*PlacementExporter).fetchInventories},
		{Name: "usages", DependsOn: []string{"providers"}, Fetch: (*PlacementExporter).fetchUsages},
		{Name: "allocations", DependsOn: []string{"providers"}, Fetch: (*PlacementExporter).fetchAllocations},
	},
	Emitters: []Emitter[*PlacementExporter, placementScrape]{
		{
			Name:    "inventories",
			Metrics: []string{"resource_total", "resource_allocation_ratio", "resource_generation", "resource_reserved"},
			Sources: []string{"inventories"},
			Emit:    (*PlacementExporter).emitInventories,
		},
		{
			Name:    "usages",
			Metrics: []string{"resource_usage"},
			Sources: []string{"usages"},
			Emit:    (*PlacementExporter).emitUsages,
		},
		{
			Name:    "allocations",
			Metrics: []string{"resource_provider_allocations"},
			Sources: []string{"allocations"},
			Emit:    (*PlacementExporter).emitAllocations,
		},
	},
}

func NewPlacementExporter(config *ExporterConfig, logger *slog.Logger) (*PlacementExporter, error) {
	e := &PlacementExporter{
		BaseOpenStackExporter: BaseOpenStackExporter{
			Name:           "placement",
			ExporterConfig: *config,
			logger:         logger,
		},
	}
	e.RegisterAndFillDescs(&e.descs)
	sched, err := placementGraph.PruneSchedule(&e.BaseOpenStackExporter)
	if err != nil {
		return nil, err
	}
	e.sched = sched
	placementGraph.LogDAG(&e.BaseOpenStackExporter, e.sched)
	return e, nil
}

func (e *PlacementExporter) Collect(ch chan<- prometheus.Metric) {
	e.RunCollect(ch, e.sched, func(ch chan<- prometheus.Metric) int {
		s := new(placementScrape)
		return runSchedule(e, &e.BaseOpenStackExporter, &placementGraph, e.sched, s, ch)
	})
}

func (e *PlacementExporter) fetchProviders(ctx context.Context, s *placementScrape) error {
	allPages, err := resourceproviders.List(e.ClientV2, resourceproviders.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}
	allProviders, err := resourceproviders.ExtractResourceProviders(allPages)
	if err != nil {
		return err
	}
	s.providers = make([]placementProviderEntry, len(allProviders))
	for i, rp := range allProviders {
		s.providers[i] = placementProviderEntry{provider: rp}
	}
	return nil
}

func (e *PlacementExporter) fetchInventories(ctx context.Context, s *placementScrape) error {
	for i := range s.providers {
		inv, err := resourceproviders.GetInventories(ctx, e.ClientV2, s.providers[i].provider.UUID).Extract()
		if err != nil {
			return err
		}
		s.providers[i].inventories = inv
	}
	return nil
}

func (e *PlacementExporter) fetchUsages(ctx context.Context, s *placementScrape) error {
	for i := range s.providers {
		u, err := resourceproviders.GetUsages(ctx, e.ClientV2, s.providers[i].provider.UUID).Extract()
		if err != nil {
			return err
		}
		s.providers[i].usages = u
	}
	return nil
}

func (e *PlacementExporter) fetchAllocations(ctx context.Context, s *placementScrape) error {
	for i := range s.providers {
		a, err := resourceproviders.GetAllocations(ctx, e.ClientV2, s.providers[i].provider.UUID).Extract()
		if err != nil {
			return err
		}
		s.providers[i].allocations = a
	}
	return nil
}

// Placement emitters index into s.providers instead of ranging by value:
// copying placementProviderEntry would read sibling fields written by
// independent sources under the parallel DAG scheduler.
func (e *PlacementExporter) emitInventories(ctx context.Context, s *placementScrape, ch chan<- prometheus.Metric) error {
	dTotal, dRatio, dGen, dReserved := e.descs.ResourceTotal, e.descs.ResourceAllocationRatio, e.descs.ResourceGeneration, e.descs.ResourceReserved
	for i := range s.providers {
		entry := &s.providers[i]
		name := entry.provider.Name
		gen := float64(entry.inventories.ResourceProviderGeneration)
		for rt, inv := range entry.inventories.Inventories {
			emitGauge(ch, dTotal, float64(inv.Total), name, rt)
			emitGauge(ch, dRatio, float64(inv.AllocationRatio), name, rt)
			emitGauge(ch, dGen, gen, name, rt)
			emitGauge(ch, dReserved, float64(inv.Reserved), name, rt)
		}
	}
	return nil
}

func (e *PlacementExporter) emitUsages(ctx context.Context, s *placementScrape, ch chan<- prometheus.Metric) error {
	for i := range s.providers {
		entry := &s.providers[i]
		for rt, v := range entry.usages.Usages {
			emitGauge(ch, e.descs.ResourceUsage, float64(v), entry.provider.Name, rt)
		}
	}
	return nil
}

func (e *PlacementExporter) emitAllocations(ctx context.Context, s *placementScrape, ch chan<- prometheus.Metric) error {
	for i := range s.providers {
		entry := &s.providers[i]
		for consumerID, alloc := range entry.allocations.Allocations {
			for rc, amount := range alloc.Resources {
				emitGauge(ch, e.descs.ResourceProviderAllocations, float64(amount), entry.provider.Name, consumerID, rc)
			}
		}
	}
	return nil
}
