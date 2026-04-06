package exporters

import (
	"log/slog"
	"sync"

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
	Name            string
	UUID            string
	Inventories     map[string]resourceproviders.Inventory
	Usages          map[string]int
	err             error
	inventoriesDone chan struct{}
	usagesDone      chan struct{}
}

const maxConcurrentRequests = 50

func ListPlacementResourceProviders(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allResourceProviders []resourceproviders.ResourceProvider

	allPagesResourceProviders, err := resourceproviders.List(exporter.Client, resourceproviders.ListOpts{}).AllPages()
	if err != nil {
		return err
	}

	if allResourceProviders, err = resourceproviders.ExtractResourceProviders(allPagesResourceProviders); err != nil {
		return err
	}

	if len(allResourceProviders) == 0 {
		return nil
	}

	if exporter.CompletePlacementInParallel {
		return listPlacementResourceProvidersParallel(exporter, ch, allResourceProviders)
	}
	return listPlacementResourceProvidersSequential(exporter, ch, allResourceProviders)
}

func listPlacementResourceProvidersSequential(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric, allResourceProviders []resourceproviders.ResourceProvider) error {
	for _, resourceprovider := range allResourceProviders {
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

		for k, v := range usagesResult.Usages {
			ch <- prometheus.MustNewConstMetric(exporter.Metrics["resource_usage"].Metric,
				prometheus.GaugeValue, float64(v), resourceprovider.Name, k)
		}
	}
	return nil
}

func listPlacementResourceProvidersParallel(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric, allResourceProviders []resourceproviders.ResourceProvider) error {
	semaphore := make(chan struct{}, maxConcurrentRequests)
	var wg sync.WaitGroup

	dataChan := make(chan *resourceProviderData, len(allResourceProviders))

	for _, rp := range allResourceProviders {
		data := &resourceProviderData{
			Name:            rp.Name,
			UUID:            rp.UUID,
			Inventories:     make(map[string]resourceproviders.Inventory),
			Usages:          make(map[string]int),
			inventoriesDone: make(chan struct{}),
			usagesDone:      make(chan struct{}),
		}
		dataChan <- data

		wg.Add(1)
		go func(data *resourceProviderData) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			inventoryResult, err := resourceproviders.GetInventories(exporter.Client, data.UUID).Extract()
			if err != nil {
				data.err = err
			} else {
				for k, v := range inventoryResult.Inventories {
					data.Inventories[k] = v
				}
			}
			close(data.inventoriesDone)
		}(data)

		wg.Add(1)
		go func(data *resourceProviderData) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			usagesResult, err := resourceproviders.GetUsages(exporter.Client, data.UUID).Extract()
			if err != nil {
				if data.err == nil {
					data.err = err
				}
			} else {
				for k, v := range usagesResult.Usages {
					data.Usages[k] = v
				}
			}
			close(data.usagesDone)
		}(data)
	}

	wg.Wait()
	close(dataChan)

	var errCollect error
	for data := range dataChan {
		if data.err != nil && errCollect == nil {
			errCollect = data.err
			continue
		}

		<-data.inventoriesDone
		<-data.usagesDone

		for k, v := range data.Inventories {
			ch <- prometheus.MustNewConstMetric(exporter.Metrics["resource_total"].Metric,
				prometheus.GaugeValue, float64(v.Total), data.Name, k)

			ch <- prometheus.MustNewConstMetric(exporter.Metrics["resource_allocation_ratio"].Metric,
				prometheus.GaugeValue, float64(v.AllocationRatio), data.Name, k)

			ch <- prometheus.MustNewConstMetric(exporter.Metrics["resource_reserved"].Metric,
				prometheus.GaugeValue, float64(v.Reserved), data.Name, k)
		}

		for k, v := range data.Usages {
			ch <- prometheus.MustNewConstMetric(exporter.Metrics["resource_usage"].Metric,
				prometheus.GaugeValue, float64(v), data.Name, k)
		}
	}

	return errCollect
}
