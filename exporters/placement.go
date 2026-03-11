package exporters

import (
	"log/slog"

	"github.com/gophercloud/gophercloud/openstack/placement/v1/resourceproviders"
	"github.com/prometheus/client_golang/prometheus"
)

const PLACEMENT_SERVICE string = "placement"

type PlacementExporter struct {
	BaseOpenStackExporter
}

var defaultPlacementMetrics = []Metric{
	{Name: "resource_total", Fn: ListPlacementResourceProviders, Labels: []string{"hostname", "resourcetype"}},
	{Name: "resource_allocation_ratio", Labels: []string{"hostname", "resourcetype"}},
	{Name: "resource_reserved", Labels: []string{"hostname", "resourcetype"}},
	{Name: "resource_usage", Labels: []string{"hostname", "resourcetype"}},
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
			labels := computeMetricLabels(PLACEMENT_SERVICE, metric, exporter.ExtraLabels)
			constLabels := computeConstantLabels(PLACEMENT_SERVICE, metric, exporter.ExtraLabels)
			exporter.AddMetric(metric.Name, metric.Fn, labels, metric.DeprecatedVersion, constLabels)
		}
	}
	return &exporter, nil
}

func ListPlacementResourceProviders(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allResourceProviders []resourceproviders.ResourceProvider

	allPagesResourceProviders, err := resourceproviders.List(exporter.Client, resourceproviders.ListOpts{}).AllPages()
	if err != nil {
		return err
	}

	if allResourceProviders, err = resourceproviders.ExtractResourceProviders(allPagesResourceProviders); err != nil {
		return err
	}

	uuidToNameMap := map[string]string{}

	resourceTotalSpec := exporter.ExtraLabels.Extract(PLACEMENT_SERVICE, "resource_total")
	resourceAllocationRatioSpec := exporter.ExtraLabels.Extract(PLACEMENT_SERVICE, "resource_allocation_ratio")
	resourceReservedSpec := exporter.ExtraLabels.Extract(PLACEMENT_SERVICE, "resource_reserved")
	resourceUsageSpec := exporter.ExtraLabels.Extract(PLACEMENT_SERVICE, "resource_usage")
	for _, resourceprovider := range allResourceProviders {
		uuidToNameMap[resourceprovider.UUID] = resourceprovider.Name

		inventoryResult, err := resourceproviders.GetInventories(exporter.Client, resourceprovider.UUID).Extract()
		if err != nil {
			return err
		}

		for k, v := range inventoryResult.Inventories {

			ch <- prometheus.MustNewConstMetric(exporter.Metrics["resource_total"].Metric,
				prometheus.GaugeValue, float64(v.Total), append([]string{resourceprovider.Name, k}, resolveExtraLabelValues(resourceprovider, resourceTotalSpec)...)...)

			ch <- prometheus.MustNewConstMetric(exporter.Metrics["resource_allocation_ratio"].Metric,
				prometheus.GaugeValue, float64(v.AllocationRatio), append([]string{resourceprovider.Name, k}, resolveExtraLabelValues(resourceprovider, resourceAllocationRatioSpec)...)...)

			ch <- prometheus.MustNewConstMetric(exporter.Metrics["resource_reserved"].Metric,
				prometheus.GaugeValue, float64(v.Reserved), append([]string{resourceprovider.Name, k}, resolveExtraLabelValues(resourceprovider, resourceReservedSpec)...)...)
		}

		usagesResult, err := resourceproviders.GetUsages(exporter.Client, resourceprovider.UUID).Extract()
		if err != nil {
			return err
		}

		for k, v := range usagesResult.Usages {
			ch <- prometheus.MustNewConstMetric(exporter.Metrics["resource_usage"].Metric,
				prometheus.GaugeValue, float64(v), append([]string{resourceprovider.Name, k}, resolveExtraLabelValues(resourceprovider, resourceUsageSpec)...)...)
		}

	}

	return nil

}
