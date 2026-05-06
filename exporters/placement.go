package exporters

import (
	"context"
	"log/slog"

	"github.com/gophercloud/gophercloud/v2/openstack/placement/v1/resourceproviders"
	"github.com/prometheus/client_golang/prometheus"
)

type PlacementExporter struct {
	BaseOpenStackExporter
}

var placementResourceLabels = []string{"hostname", "resourcetype"}

var defaultPlacementMetrics = []Metric{
	{Name: "resource_total", Fn: ListPlacementResourceProviders, Labels: placementResourceLabels},
	{Name: "resource_allocation_ratio", Labels: placementResourceLabels},
	{Name: "resource_generation", Labels: placementResourceLabels},
	{Name: "resource_reserved", Labels: placementResourceLabels},
	{Name: "resource_usage", Labels: placementResourceLabels},
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

	allPagesResourceProviders, err := resourceproviders.List(exporter.ClientV2, resourceproviders.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}

	if allResourceProviders, err = resourceproviders.ExtractResourceProviders(allPagesResourceProviders); err != nil {
		return err
	}

	for _, resourceprovider := range allResourceProviders {
		inventoryResult, err := resourceproviders.GetInventories(ctx, exporter.ClientV2, resourceprovider.UUID).Extract()
		if err != nil {
			return err
		}

		for k, v := range inventoryResult.Inventories {
			emitPlacementResourceMetric(exporter, ch, "resource_total", float64(v.Total), resourceprovider.Name, k)
			emitPlacementResourceMetric(exporter, ch, "resource_allocation_ratio", float64(v.AllocationRatio), resourceprovider.Name, k)
			emitPlacementResourceMetric(exporter, ch, "resource_generation", float64(inventoryResult.ResourceProviderGeneration), resourceprovider.Name, k)
			emitPlacementResourceMetric(exporter, ch, "resource_reserved", float64(v.Reserved), resourceprovider.Name, k)
		}

		usagesResult, err := resourceproviders.GetUsages(ctx, exporter.ClientV2, resourceprovider.UUID).Extract()
		if err != nil {
			return err
		}

		for k, v := range usagesResult.Usages {
			emitPlacementResourceMetric(exporter, ch, "resource_usage", float64(v), resourceprovider.Name, k)
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
