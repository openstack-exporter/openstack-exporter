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

	concurrency := 1
	if exporter.CompletePlacementInParallel {
		concurrency = maxConcurrentRequests
	}
	return collectPlacementResourceProviders(exporter, ch, allResourceProviders, concurrency)
}

func collectPlacementResourceProviders(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric, allResourceProviders []resourceproviders.ResourceProvider, concurrency int) error {
	semaphore := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var errMu sync.Mutex
	var errCollect error

	setError := func(err error) {
		errMu.Lock()
		defer errMu.Unlock()
		if errCollect == nil {
			errCollect = err
		}
	}

	for _, rp := range allResourceProviders {
		rp := rp

		wg.Add(1)
		go func() {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			inventoryResult, err := resourceproviders.GetInventories(exporter.Client, rp.UUID).Extract()
			if err != nil {
				setError(err)
				return
			}

			for k, v := range inventoryResult.Inventories {
				ch <- prometheus.MustNewConstMetric(exporter.Metrics["resource_total"].Metric,
					prometheus.GaugeValue, float64(v.Total), rp.Name, k)

				ch <- prometheus.MustNewConstMetric(exporter.Metrics["resource_allocation_ratio"].Metric,
					prometheus.GaugeValue, float64(v.AllocationRatio), rp.Name, k)

				ch <- prometheus.MustNewConstMetric(exporter.Metrics["resource_reserved"].Metric,
					prometheus.GaugeValue, float64(v.Reserved), rp.Name, k)
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			usagesResult, err := resourceproviders.GetUsages(exporter.Client, rp.UUID).Extract()
			if err != nil {
				setError(err)
				return
			}

			for k, v := range usagesResult.Usages {
				ch <- prometheus.MustNewConstMetric(exporter.Metrics["resource_usage"].Metric,
					prometheus.GaugeValue, float64(v), rp.Name, k)
			}
		}()
	}

	wg.Wait()
	return errCollect
}
