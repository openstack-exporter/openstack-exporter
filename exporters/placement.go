package exporters

import (
	"sync"

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

// resourceProviderData holds the inventory and usage data for a single resource provider
type resourceProviderData struct {
	name       string
	inventries *resourceproviders.ResourceProviderInventories
	usages     *resourceproviders.ResourceProviderUsage
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

	// Limit concurrency to avoid overwhelming the API
	// Using a semaphore pattern with buffered channel
	const maxConcurrency = 10
	sem := make(chan struct{}, maxConcurrency)

	// Collect results with mutex protection
	var mu sync.Mutex
	results := make([]resourceProviderData, 0, len(allResourceProviders))

	// Error channel to collect errors from goroutines
	errCh := make(chan error, len(allResourceProviders))
	var wg sync.WaitGroup

	for _, rp := range allResourceProviders {
		wg.Add(1)
		// Capture loop variable
		resourceProvider := rp

		go func() {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			// Fetch inventories and usages concurrently for this provider
			var inventory *resourceproviders.ResourceProviderInventories
			var usage *resourceproviders.ResourceProviderUsage
			var inventoryErr, usageErr error

			var innerWg sync.WaitGroup
			innerWg.Add(2)

			// Fetch inventories
			go func() {
				defer innerWg.Done()
				inventory, inventoryErr = resourceproviders.GetInventories(exporter.Client, resourceProvider.UUID).Extract()
			}()

			// Fetch usages
			go func() {
				defer innerWg.Done()
				usage, usageErr = resourceproviders.GetUsages(exporter.Client, resourceProvider.UUID).Extract()
			}()

			innerWg.Wait()

			// Check for errors
			if inventoryErr != nil {
				errCh <- inventoryErr
				return
			}
			if usageErr != nil {
				errCh <- usageErr
				return
			}

			// Store results with mutex protection
			mu.Lock()
			results = append(results, resourceProviderData{
				name:       resourceProvider.Name,
				inventries: inventory,
				usages:     usage,
			})
			mu.Unlock()
		}()
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errCh)

	// Check if any errors occurred
	for err := range errCh {
		if err != nil {
			return err
		}
	}

	// Emit metrics from collected results
	for _, data := range results {
		// Emit inventory metrics
		for resourceType, inv := range data.inventries.Inventories {
			ch <- prometheus.MustNewConstMetric(exporter.Metrics["resource_total"].Metric,
				prometheus.GaugeValue, float64(inv.Total), data.name, resourceType)

			ch <- prometheus.MustNewConstMetric(exporter.Metrics["resource_allocation_ratio"].Metric,
				prometheus.GaugeValue, float64(inv.AllocationRatio), data.name, resourceType)

			ch <- prometheus.MustNewConstMetric(exporter.Metrics["resource_reserved"].Metric,
				prometheus.GaugeValue, float64(inv.Reserved), data.name, resourceType)
		}

		// Emit usage metrics
		for resourceType, usageValue := range data.usages.Usages {
			ch <- prometheus.MustNewConstMetric(exporter.Metrics["resource_usage"].Metric,
				prometheus.GaugeValue, float64(usageValue), data.name, resourceType)
		}
	}

	return nil
}
