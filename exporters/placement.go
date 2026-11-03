package exporters

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/placement/v1/resourceproviders"
	"github.com/prometheus/client_golang/prometheus"
)

type PlacementExporter struct {
	BaseOpenStackExporter
}

var placementResourceLabels = []string{"hostname", "resourcetype", "zone"}
var placementAllocationLabels = []string{"hostname", "uuid", "resourcetype", "zone"}

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

func ListPlacementResourceProviders(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allResourceProviders []resourceproviders.ResourceProvider

	// Provider trees (including GPU and PCI devices) are exposed from 1.14.
	// Use a copy so other requests keep their configured microversion.
	client := *exporter.ClientV2
	if client.Microversion == "" {
		client.Microversion = "1.14"
	}
	allPagesResourceProviders, err := resourceproviders.List(&client, resourceproviders.ListOpts{}).AllPages(ctx)
	if err != nil && exporter.ClientV2.Microversion == "" && gophercloud.ResponseCodeIs(err, http.StatusNotAcceptable) {
		// Older Placement APIs still expose root compute providers by name.
		allPagesResourceProviders, err = resourceproviders.List(exporter.ClientV2, resourceproviders.ListOpts{}).AllPages(ctx)
	}
	if err != nil {
		return err
	}

	if allResourceProviders, err = resourceproviders.ExtractResourceProviders(allPagesResourceProviders); err != nil {
		return err
	}

	zones := computeResourceZones(ctx, exporter)
	providersByID := make(map[string]resourceproviders.ResourceProvider, len(allResourceProviders))
	for _, provider := range allResourceProviders {
		providersByID[provider.UUID] = provider
	}

	for _, resourceprovider := range allResourceProviders {
		zone := placementResourceZone(resourceprovider, providersByID, zones)
		inventoryResult, err := resourceproviders.GetInventories(ctx, exporter.ClientV2, resourceprovider.UUID).Extract()
		if err != nil {
			return err
		}

		for k, v := range inventoryResult.Inventories {
			emitPlacementResourceMetric(exporter, ch, "resource_total", float64(v.Total), resourceprovider.Name, k, zone)
			emitPlacementResourceMetric(exporter, ch, "resource_allocation_ratio", float64(v.AllocationRatio), resourceprovider.Name, k, zone)
			emitPlacementResourceMetric(exporter, ch, "resource_generation", float64(inventoryResult.ResourceProviderGeneration), resourceprovider.Name, k, zone)
			emitPlacementResourceMetric(exporter, ch, "resource_reserved", float64(v.Reserved), resourceprovider.Name, k, zone)
		}

		usagesResult, err := resourceproviders.GetUsages(ctx, exporter.ClientV2, resourceprovider.UUID).Extract()
		if err != nil {
			return err
		}

		for k, v := range usagesResult.Usages {
			emitPlacementResourceMetric(exporter, ch, "resource_usage", float64(v), resourceprovider.Name, k, zone)
		}

		if _, ok := exporter.Metrics["resource_provider_allocations"]; ok {
			allocationsResult, err := resourceproviders.GetAllocations(ctx, exporter.ClientV2, resourceprovider.UUID).Extract()
			if err != nil {
				return err
			}

			for consumerID, allocation := range allocationsResult.Allocations {
				for resourceClass, amount := range allocation.Resources {
					ch <- prometheus.MustNewConstMetric(
						exporter.Metrics["resource_provider_allocations"].Metric,
						prometheus.GaugeValue,
						float64(amount),
						resourceprovider.Name,
						consumerID,
						resourceClass,
						zone,
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
	zone string,
) {
	ch <- prometheus.MustNewConstMetric(
		exporter.Metrics[metricName].Metric,
		prometheus.GaugeValue,
		value,
		hostname,
		resourceType,
		zone,
	)
}

func placementResourceZone(provider resourceproviders.ResourceProvider, providers map[string]resourceproviders.ResourceProvider, zones map[string]string) string {
	// A nested resource belongs to its root compute host's zone. Resolve the
	// root first rather than interpreting a device name as a compute hostname.
	if zone := zones[provider.RootProviderUUID]; zone != "" {
		return zone
	}
	if root, ok := providers[provider.RootProviderUUID]; ok {
		provider = root
	}
	// Fall back to the parent chain if the response omits root_provider_uuid.
	for remaining := len(providers) + 1; remaining > 0; remaining-- {
		if zone := zones[provider.UUID]; zone != "" {
			return zone
		}
		if zone := zones[provider.Name]; zone != "" {
			return zone
		}
		parent, ok := providers[provider.ParentProviderUUID]
		if !ok {
			break
		}
		provider = parent
	}
	return ""
}
