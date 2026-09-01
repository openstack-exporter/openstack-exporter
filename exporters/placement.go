package exporters

import (
	"context"
	"log/slog"
	"sync"

	"github.com/gophercloud/gophercloud/v2/openstack/placement/v1/resourceproviders"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
)

type PlacementExporter struct {
	BaseOpenStackExporter
}

var placementResourceLabels = []string{"hostname", "resourcetype"}
var placementAllocationLabels = []string{"hostname", "uuid", "resourcetype"}

var defaultPlacementMetrics = []Metric{
	{Name: "resource_total", Fn: ListPlacementResourceProviders, Labels: placementResourceLabels},
	{Name: "resource_allocation_ratio", Labels: placementResourceLabels},
	{Name: "resource_generation", Labels: placementResourceLabels},
	{Name: "resource_reserved", Labels: placementResourceLabels},
	{Name: "resource_usage", Labels: placementResourceLabels},
	{Name: "resource_provider_allocations", Labels: placementAllocationLabels},
}

func NewPlacementExporter(config *ExporterConfig, logger *slog.Logger) (*PlacementExporter, error) {
	exporter := PlacementExporter{
		BaseOpenStackExporter{
			Name:           "placement",
			ExporterConfig: *config,
			logger:         logger,
		},
	}
	for _, metric := range defaultPlacementMetrics {
		if exporter.isDeprecatedMetric(&metric) {
			continue
		}
		if !exporter.isSlowMetric(&metric) {
			exporter.AddMetric(metric.Name, metric.Fn, metric.Labels, metric.DeprecatedVersion, nil)
		}
	}
	return &exporter, nil
}

type resourceProviderData struct {
	name        string
	inventories *resourceproviders.ResourceProviderInventories
	usages      *resourceproviders.ResourceProviderUsage
	allocations *resourceproviders.ResourceProviderAllocations
}

func ListPlacementResourceProviders(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allResourceProviders []resourceproviders.ResourceProvider

	allPagesResourceProviders, err := resourceproviders.List(exporter.ClientV2, resourceproviders.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}

	if allResourceProviders, err = resourceproviders.ExtractResourceProviders(allPagesResourceProviders); err != nil {
		return err
	}

	var (
		mu      sync.Mutex
		results = make([]resourceProviderData, 0, len(allResourceProviders))
	)

	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(10)

	collectAllocations := exporter.Metrics["resource_provider_allocations"] != nil
	for _, resourceprovider := range allResourceProviders {
		resourceprovider := resourceprovider
		g.Go(func() error {
			inventoryResult, err := resourceproviders.GetInventories(gCtx, exporter.ClientV2, resourceprovider.UUID).Extract()
			if err != nil {
				return err
			}

			usagesResult, err := resourceproviders.GetUsages(gCtx, exporter.ClientV2, resourceprovider.UUID).Extract()
			if err != nil {
				return err
			}

			var allocationsResult *resourceproviders.ResourceProviderAllocations
			if collectAllocations {
				allocationsResult, err = resourceproviders.GetAllocations(gCtx, exporter.ClientV2, resourceprovider.UUID).Extract()
				if err != nil {
					return err
				}
			}

			mu.Lock()
			results = append(results, resourceProviderData{
				name:        resourceprovider.Name,
				inventories: inventoryResult,
				usages:      usagesResult,
				allocations: allocationsResult,
			})
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	for _, data := range results {
		for k, v := range data.inventories.Inventories {
			emitPlacementResourceMetric(exporter, ch, "resource_total", float64(v.Total), data.name, k)
			emitPlacementResourceMetric(exporter, ch, "resource_allocation_ratio", float64(v.AllocationRatio), data.name, k)
			emitPlacementResourceMetric(exporter, ch, "resource_generation", float64(data.inventories.ResourceProviderGeneration), data.name, k)
			emitPlacementResourceMetric(exporter, ch, "resource_reserved", float64(v.Reserved), data.name, k)
		}

		for k, v := range data.usages.Usages {
			emitPlacementResourceMetric(exporter, ch, "resource_usage", float64(v), data.name, k)
		}

		if data.allocations != nil {
			for consumerID, allocation := range data.allocations.Allocations {
				for resourceClass, amount := range allocation.Resources {
					ch <- prometheus.MustNewConstMetric(
						exporter.Metrics["resource_provider_allocations"].Metric,
						prometheus.GaugeValue,
						float64(amount),
						data.name,
						consumerID,
						resourceClass,
					)
				}
			}
		}
	}

	return nil
}

func emitPlacementResourceMetric(
	exporter *BaseOpenStackExporter,
	ch chan<- prometheus.Metric,
	metricName string,
	value float64,
	hostname string,
	resourceType string,
) {
	ch <- prometheus.MustNewConstMetric(
		exporter.Metrics[metricName].Metric,
		prometheus.GaugeValue,
		value,
		hostname,
		resourceType,
	)
}
