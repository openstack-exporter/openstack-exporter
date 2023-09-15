package exporters

import (
	"github.com/go-kit/log"
	"github.com/gophercloud/gophercloud/openstack/placement/v1/resourceproviders"
	"github.com/prometheus/client_golang/prometheus"
)

type PlacementExporter struct {
	BaseOpenStackExporter
}

var defaultPlacementMetrics = []Metric{
	{Name: "resource_total", Fn: ListPlacementResourceProviders, Labels: []string{"hostname", "resourcetype"}},
	{Name: "resource_allocation_ratio", Labels: []string{"hostname", "resourcetype"}},
	{Name: "resource_reserved", Labels: []string{"hostname", "resourcetype"}},
	{Name: "resource_usage", Labels: []string{"hostname", "resourcetype"}},
	{Name: "vcpus_available", Labels: []string{"hostname"}},
	{Name: "vcpus_used", Labels: []string{"hostname"}},
	{Name: "memory_available_bytes", Labels: []string{"hostname"}},
	{Name: "memory_used_bytes", Labels: []string{"hostname"}},
	{Name: "local_storage_available_bytes", Labels: []string{"hostname"}},
	{Name: "local_storage_used_bytes", Labels: []string{"hostname"}},
}

func NewPlacementExporter(config *ExporterConfig, logger log.Logger) (*PlacementExporter, error) {
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

	for _, resourceprovider := range allResourceProviders {
		uuidToNameMap[resourceprovider.UUID] = resourceprovider.Name

		inventoryResult, err := resourceproviders.GetInventories(exporter.Client, resourceprovider.UUID).Extract()
		if err != nil {
			return err
		}

		for k, v := range inventoryResult.Inventories {

			ch <- prometheus.MustNewConstMetric(exporter.Metrics["resource_total"].Metric,
				prometheus.GaugeValue, float64(v.Total), resourceprovider.Name, k)

			ch <- prometheus.MustNewConstMetric(exporter.Metrics["resource_allocation_ratio"].Metric,
				prometheus.GaugeValue, float64(v.AllocationRatio), resourceprovider.Name, k)

			ch <- prometheus.MustNewConstMetric(exporter.Metrics["resource_reserved"].Metric,
				prometheus.GaugeValue, float64(v.Reserved), resourceprovider.Name, k)
		}

		usagesResult, err := resourceproviders.GetUsages(exporter.Client, resourceprovider.UUID).Extract()
		if err != nil {
			return err
		}

		if diskGb, ok := inventoryResult.Inventories["DISK_GB"]; ok {
			ch <- prometheus.MustNewConstMetric(exporter.Metrics["local_storage_available_bytes"].Metric,
				prometheus.GaugeValue, float64(diskGb.Total*GIGABYTE), resourceprovider.Name)
		}

		if diskGbUsage, ok := usagesResult.Usages["DISK_GB"]; ok {
			ch <- prometheus.MustNewConstMetric(exporter.Metrics["local_storage_used_bytes"].Metric,
				prometheus.GaugeValue, float64(diskGbUsage*GIGABYTE), resourceprovider.Name)

		}
		if memoryMb, ok := inventoryResult.Inventories["MEMORY_MB"]; ok {
			ch <- prometheus.MustNewConstMetric(exporter.Metrics["memory_available_bytes"].Metric,
				prometheus.GaugeValue, float64(memoryMb.Total*MEGABYTE), resourceprovider.Name)

		}
		if memoryMbUsage, ok := usagesResult.Usages["MEMORY_MB"]; ok {
			ch <- prometheus.MustNewConstMetric(exporter.Metrics["memory_used_bytes"].Metric,
				prometheus.GaugeValue, float64(memoryMbUsage*MEGABYTE), resourceprovider.Name)

		}
		if vcpus, ok := inventoryResult.Inventories["VCPU"]; ok {
			ch <- prometheus.MustNewConstMetric(exporter.Metrics["vcpus_available"].Metric,
				prometheus.GaugeValue, float64(vcpus.Total), resourceprovider.Name)

		}
		if vcpusUsage, ok := usagesResult.Usages["VCPU"]; ok {
			ch <- prometheus.MustNewConstMetric(exporter.Metrics["vcpus_used"].Metric,
				prometheus.GaugeValue, float64(vcpusUsage), resourceprovider.Name)

		}

		for k, v := range usagesResult.Usages {
			ch <- prometheus.MustNewConstMetric(exporter.Metrics["resource_usage"].Metric,
				prometheus.GaugeValue, float64(v), resourceprovider.Name, k)
		}

	}

	return nil

}
