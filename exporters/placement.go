package exporters

import (
	"context"
	"log/slog"
	"sort"
	"strings"
	"sync"

	"github.com/gophercloud/gophercloud/v2/openstack/placement/v1/resourceproviders"
	"github.com/prometheus/client_golang/prometheus"
)

type cachedMetricEntry struct {
	metricName string
	value      float64
	labels     []string
}

type cachedProviderData struct {
	generation int
	name       string
	traits     string
	metrics    []cachedMetricEntry
}

type placementCache struct {
	mu        sync.Mutex
	providers map[string]*cachedProviderData
}

type PlacementExporter struct {
	BaseOpenStackExporter
	cache *placementCache
}

var placementResourceLabels = []string{"hostname", "resourcetype", "resource_traits"}
var placementTraitLabels = []string{"hostname", "resource_traits"}
var placementAllocationLabels = []string{"hostname", "uuid", "resourcetype"}

var defaultPlacementMetrics = []Metric{
	{Name: "resource_total", Fn: ListPlacementResourceProviders, Labels: placementResourceLabels},
	{Name: "resource_allocation_ratio", Labels: placementResourceLabels},
	{Name: "resource_generation", Labels: placementResourceLabels},
	{Name: "resource_reserved", Labels: placementResourceLabels},
	{Name: "resource_usage", Labels: placementResourceLabels},
	{Name: "resource_traits", Labels: placementTraitLabels},
	{Name: "resource_provider_allocations", Labels: placementAllocationLabels},
}

func NewPlacementExporter(config *ExporterConfig, logger *slog.Logger) (*PlacementExporter, error) {
	exporter := &PlacementExporter{
		BaseOpenStackExporter: BaseOpenStackExporter{
			Name:           "placement",
			ExporterConfig: *config,
			logger:         logger,
		},
		cache: &placementCache{providers: make(map[string]*cachedProviderData)},
	}

	for _, metric := range defaultPlacementMetrics {
		if exporter.isDeprecatedMetric(&metric) {
			continue
		}
		if !exporter.isSlowMetric(&metric) {
			fn := metric.Fn
			if fn != nil {
				fn = exporter.listWithCache
			}
			exporter.AddMetric(metric.Name, fn, metric.Labels, metric.DeprecatedVersion, nil)
		}
	}
	return exporter, nil
}

const maxConcurrentPlacementRequests = 50

func (pe *PlacementExporter) listWithCache(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	allPagesResourceProviders, err := resourceproviders.List(exporter.ClientV2, resourceproviders.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}

	allResourceProviders, err := resourceproviders.ExtractResourceProviders(allPagesResourceProviders)
	if err != nil {
		return err
	}

	currentProviderUUIDs := make(map[string]struct{}, len(allResourceProviders))
	pe.cache.mu.Lock()
	for _, rp := range allResourceProviders {
		currentProviderUUIDs[rp.UUID] = struct{}{}
	}

	for uuid := range pe.cache.providers {
		if _, exists := currentProviderUUIDs[uuid]; !exists {
			delete(pe.cache.providers, uuid)
		}
	}
	pe.cache.mu.Unlock()

	concurrency := 1
	if exporter.CompletePlacementInParallel {
		concurrency = maxConcurrentPlacementRequests
	}
	return collectAndCachePlacementProviders(ctx, exporter, ch, allResourceProviders, concurrency, pe.cache)
}

// ListPlacementResourceProviders is the non-cached version used by tests and as fallback.
func ListPlacementResourceProviders(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allResourceProviders []resourceproviders.ResourceProvider

	allPagesResourceProviders, err := resourceproviders.List(exporter.ClientV2, resourceproviders.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}

	if allResourceProviders, err = resourceproviders.ExtractResourceProviders(allPagesResourceProviders); err != nil {
		return err
	}

	concurrency := 1
	if exporter.CompletePlacementInParallel {
		concurrency = maxConcurrentPlacementRequests
	}
	return collectAndCachePlacementProviders(ctx, exporter, ch, allResourceProviders, concurrency, nil)
}

func collectAndCachePlacementProviders(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric, providers []resourceproviders.ResourceProvider, concurrency int, cache *placementCache) error {
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

	for _, resourceprovider := range providers {
		resourceprovider := resourceprovider

		wg.Add(1)
		go func() {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			var cached *cachedProviderData
			if cache != nil {
				cache.mu.Lock()
				cached = cache.providers[resourceprovider.UUID]
				cache.mu.Unlock()
			}
			if cached != nil && (cached.generation != resourceprovider.Generation || cached.name != resourceprovider.Name) {
				cached = nil
			}

			if cached == nil {
				var entries []cachedMetricEntry
				cacheable := true
				traitsStr := ""
				if exporter.CollectPlacementTraits {
					if traitResult, err := resourceproviders.GetTraits(ctx, exporter.ClientV2, resourceprovider.UUID).Extract(); err == nil {
						cacheable = traitResult.ResourceProviderGeneration == resourceprovider.Generation
						traits := make([]string, 0, len(traitResult.Traits))
						for _, trait := range traitResult.Traits {
							if strings.HasPrefix(trait, "CUSTOM_") {
								traits = append(traits, trait)
							}
						}
						sort.Strings(traits)
						traitsStr = strings.Join(traits, ",")
					} else {
						setError(err)
						return
					}
				}

				inventoryResult, err := resourceproviders.GetInventories(ctx, exporter.ClientV2, resourceprovider.UUID).Extract()
				if err != nil {
					setError(err)
					return
				}

				for k, v := range inventoryResult.Inventories {
					entries = append(entries,
						cachedMetricEntry{"resource_total", float64(v.Total), []string{resourceprovider.Name, k, traitsStr}},
						cachedMetricEntry{"resource_allocation_ratio", float64(v.AllocationRatio), []string{resourceprovider.Name, k, traitsStr}},
						cachedMetricEntry{"resource_generation", float64(inventoryResult.ResourceProviderGeneration), []string{resourceprovider.Name, k, traitsStr}},
						cachedMetricEntry{"resource_reserved", float64(v.Reserved), []string{resourceprovider.Name, k, traitsStr}},
					)
				}

				entries = append(entries, cachedMetricEntry{"resource_traits", 1.0, []string{resourceprovider.Name, traitsStr}})
				cached = &cachedProviderData{
					generation: resourceprovider.Generation,
					name:       resourceprovider.Name,
					traits:     traitsStr,
					metrics:    entries,
				}
				// Only cache static data when its API generations agree with the list.
				if cache != nil && cacheable && inventoryResult.ResourceProviderGeneration == resourceprovider.Generation {
					cache.mu.Lock()
					if existing := cache.providers[resourceprovider.UUID]; existing == nil || existing.generation <= cached.generation {
						cache.providers[resourceprovider.UUID] = cached
					}
					cache.mu.Unlock()
				}
			}
			emitPlacementMetrics(exporter, ch, cached.metrics)

			// Allocation changes may leave the provider generation unchanged.
			// Read usage and allocations on every scrape.
			traitsStr := cached.traits

			usagesResult, err := resourceproviders.GetUsages(ctx, exporter.ClientV2, resourceprovider.UUID).Extract()
			if err != nil {
				setError(err)
				return
			}

			for k, v := range usagesResult.Usages {
				emitPlacementMetrics(exporter, ch, []cachedMetricEntry{
					{"resource_usage", float64(v), []string{resourceprovider.Name, k, traitsStr}},
				})
			}

			if _, ok := exporter.Metrics["resource_provider_allocations"]; ok {
				allocationsResult, err := resourceproviders.GetAllocations(ctx, exporter.ClientV2, resourceprovider.UUID).Extract()
				if err != nil {
					setError(err)
					return
				}

				for consumerID, allocation := range allocationsResult.Allocations {
					for resourceClass, amount := range allocation.Resources {
						emitPlacementMetrics(exporter, ch, []cachedMetricEntry{
							{"resource_provider_allocations", float64(amount), []string{resourceprovider.Name, consumerID, resourceClass}},
						})
					}
				}
			}
		}()
	}

	wg.Wait()
	return errCollect
}

func emitPlacementMetrics(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric, entries []cachedMetricEntry) {
	for _, entry := range entries {
		if desc, ok := exporter.Metrics[entry.metricName]; ok {
			ch <- prometheus.MustNewConstMetric(
				desc.Metric,
				prometheus.GaugeValue,
				entry.value,
				entry.labels...,
			)
		}
	}
}
