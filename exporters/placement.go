package exporters

import (
	"context"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gophercloud/gophercloud/v2/openstack/placement/v1/resourceproviders"
	"github.com/openstack-exporter/openstack-exporter/utils"
	"github.com/prometheus/client_golang/prometheus"
)

type cachedMetricEntry struct {
	metricName string
	value      float64
	labels     []string
}

type cachedProviderData struct {
	listGeneration       int
	traitsGeneration     int
	inventoryGeneration  int
	usageGeneration      int
	allocationGeneration int
	traitsMetrics        []cachedMetricEntry
	inventoryMetrics     []cachedMetricEntry
	usageMetrics         []cachedMetricEntry
	allocationMetrics    []cachedMetricEntry
}

func (c *cachedProviderData) allMetrics() []cachedMetricEntry {
	var all []cachedMetricEntry
	all = append(all, c.traitsMetrics...)
	all = append(all, c.inventoryMetrics...)
	all = append(all, c.usageMetrics...)
	all = append(all, c.allocationMetrics...)
	return all
}

// staleAPIs returns which API calls need re-fetching because their
// generation doesn't match the list generation.
func (c *cachedProviderData) staleAPIs(listGen int) (traits, inventory, usage, allocations bool) {
	traits = c.traitsGeneration != listGen
	inventory = c.inventoryGeneration != listGen
	usage = c.usageGeneration != listGen
	allocations = c.allocationGeneration != listGen
	return
}

type placementCache struct {
	mu        sync.Mutex
	providers map[string]*cachedProviderData
}

var globalPlacementCaches = struct {
	mu     sync.Mutex
	caches map[string]*placementCache
}{caches: make(map[string]*placementCache)}

func getPlacementCache(endpoint string) *placementCache {
	globalPlacementCaches.mu.Lock()
	defer globalPlacementCaches.mu.Unlock()
	if c, ok := globalPlacementCaches.caches[endpoint]; ok {
		return c
	}
	c := &placementCache{providers: make(map[string]*cachedProviderData)}
	globalPlacementCaches.caches[endpoint] = c
	return c
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

const placementLatestSupportedMicroversion = "1.39"

func NewPlacementExporter(config *ExporterConfig, logger *slog.Logger) (*PlacementExporter, error) {
	ctx := context.TODO()

	err := utils.SetupClientMicroversionV2(ctx, config.ClientV2, "OS_PLACEMENT_API_VERSION", placementLatestSupportedMicroversion, logger)
	if err != nil {
		return nil, err
	}

	exporter := &PlacementExporter{
		BaseOpenStackExporter: BaseOpenStackExporter{
			Name:           "placement",
			ExporterConfig: *config,
			logger:         logger,
		},
		cache: getPlacementCache(config.ClientV2.Endpoint),
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
const maxStaleRetries = 3

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

	var providersToFetch []fetchRequest
	var fullFetchCount, partialFetchCount int

	pe.cache.mu.Lock()
	for _, rp := range allResourceProviders {
		currentProviderUUIDs[rp.UUID] = struct{}{}

		cached, exists := pe.cache.providers[rp.UUID]
		if !exists || cached.listGeneration != rp.Generation {
			// Full re-fetch: new provider or generation changed
			providersToFetch = append(providersToFetch, fetchRequest{
				rp: rp, refetchTraits: true, refetchInventory: true,
				refetchUsage: true, refetchAllocations: true,
			})
			fullFetchCount++
		} else {
			// Generation matches list - check per-API staleness
			staleTraits, staleInv, staleUsage, staleAlloc := cached.staleAPIs(rp.Generation)
			if staleTraits || staleInv {
				// Traits/inventory are embedded in all labels - must do full refetch
				providersToFetch = append(providersToFetch, fetchRequest{
					rp: rp, refetchTraits: true, refetchInventory: true,
					refetchUsage: true, refetchAllocations: true,
				})
				fullFetchCount++
			} else if staleUsage || staleAlloc {
				providersToFetch = append(providersToFetch, fetchRequest{
					rp: rp, refetchTraits: false, refetchInventory: false,
					refetchUsage: staleUsage, refetchAllocations: staleAlloc,
				})
				partialFetchCount++
			} else {
				// All generations match - serve from cache
				emitCachedMetrics(exporter, ch, cached)
			}
		}
	}

	for uuid := range pe.cache.providers {
		if _, exists := currentProviderUUIDs[uuid]; !exists {
			delete(pe.cache.providers, uuid)
		}
	}
	pe.cache.mu.Unlock()

	exporter.logger.Info("Placement cache status",
		"total_providers", len(allResourceProviders),
		"cache_hits", len(allResourceProviders)-len(providersToFetch),
		"full_fetches", fullFetchCount,
		"partial_fetches", partialFetchCount,
	)

	if len(providersToFetch) > 0 {
		concurrency := 1
		if exporter.CompletePlacementInParallel {
			concurrency = maxConcurrentPlacementRequests
		}
		err = fetchAndCachePlacementProviders(ctx, exporter, ch, providersToFetch, concurrency, pe.cache)
	}

	return err
}

// ListPlacementResourceProviders is the non-cached version used by tests and as fallback
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

	requests := make([]fetchRequest, len(allResourceProviders))
	for i, rp := range allResourceProviders {
		requests[i] = fetchRequest{
			rp: rp, refetchTraits: true, refetchInventory: true,
			refetchUsage: true, refetchAllocations: true,
		}
	}
	return fetchAndCachePlacementProviders(ctx, exporter, ch, requests, concurrency, nil)
}

type fetchRequest struct {
	rp                 resourceproviders.ResourceProvider
	refetchTraits      bool
	refetchInventory   bool
	refetchUsage       bool
	refetchAllocations bool
}

func fetchAndCachePlacementProviders(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric, requests []fetchRequest, concurrency int, cache *placementCache) error {
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

	for _, req := range requests {
		req := req

		wg.Add(1)
		go func() {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			rp := req.rp
			listGen := rp.Generation

			// Get existing cache entry if doing partial re-fetch.
			// Deep copy slices to avoid aliasing with concurrent scrapes.
			var existing *cachedProviderData
			if cache != nil {
				cache.mu.Lock()
				if e, ok := cache.providers[rp.UUID]; ok {
					existing = &cachedProviderData{
						listGeneration:       e.listGeneration,
						traitsGeneration:     e.traitsGeneration,
						inventoryGeneration:  e.inventoryGeneration,
						usageGeneration:      e.usageGeneration,
						allocationGeneration: e.allocationGeneration,
						traitsMetrics:        append([]cachedMetricEntry(nil), e.traitsMetrics...),
						inventoryMetrics:     append([]cachedMetricEntry(nil), e.inventoryMetrics...),
						usageMetrics:         append([]cachedMetricEntry(nil), e.usageMetrics...),
						allocationMetrics:    append([]cachedMetricEntry(nil), e.allocationMetrics...),
					}
				}
				cache.mu.Unlock()
			}

			var traitsMetrics []cachedMetricEntry
			var inventoryMetrics []cachedMetricEntry
			var usageMetrics []cachedMetricEntry
			var allocationMetrics []cachedMetricEntry
			var traitsGen, invGen, usageGen, allocGen int

			// Traits
			traitsStr := ""
			if req.refetchTraits {
				if traitResult, err := resourceproviders.GetTraits(ctx, exporter.ClientV2, rp.UUID).Extract(); err == nil {
					traitsGen = traitResult.ResourceProviderGeneration
					traits := make([]string, 0, len(traitResult.Traits))
					for _, trait := range traitResult.Traits {
						if strings.HasPrefix(trait, "CUSTOM_") {
							traits = append(traits, trait)
						}
					}
					sort.Strings(traits)
					traitsStr = strings.Join(traits, ",")
				} else {
					exporter.logger.Warn("failed to retrieve placement resource provider traits", "resource_provider", rp.UUID, "error", err)
					traitsGen = 0 // force refetch next scrape
				}
				traitsMetrics = append(traitsMetrics, cachedMetricEntry{"resource_traits", 1.0, []string{rp.Name, traitsStr}})
			} else if existing != nil {
				traitsMetrics = existing.traitsMetrics
				traitsGen = existing.traitsGeneration
				// Extract traitsStr from cached metrics for use in other fetches
				for _, m := range existing.traitsMetrics {
					if m.metricName == "resource_traits" && len(m.labels) >= 2 {
						traitsStr = m.labels[1]
					}
				}
			}

			// Inventories
			if req.refetchInventory {
				inventoryResult, err := resourceproviders.GetInventories(ctx, exporter.ClientV2, rp.UUID).Extract()
				if err != nil {
					setError(err)
					return
				}
				invGen = inventoryResult.ResourceProviderGeneration
				for k, v := range inventoryResult.Inventories {
					inventoryMetrics = append(inventoryMetrics,
						cachedMetricEntry{"resource_total", float64(v.Total), []string{rp.Name, k, traitsStr}},
						cachedMetricEntry{"resource_allocation_ratio", float64(v.AllocationRatio), []string{rp.Name, k, traitsStr}},
						cachedMetricEntry{"resource_generation", float64(inventoryResult.ResourceProviderGeneration), []string{rp.Name, k, traitsStr}},
						cachedMetricEntry{"resource_reserved", float64(v.Reserved), []string{rp.Name, k, traitsStr}},
					)
				}
			} else if existing != nil {
				inventoryMetrics = existing.inventoryMetrics
				invGen = existing.inventoryGeneration
			}

			// Usages - retry if generation is stale
			if req.refetchUsage {
				for attempt := 0; attempt < maxStaleRetries; attempt++ {
					usagesResult, err := resourceproviders.GetUsages(ctx, exporter.ClientV2, rp.UUID).Extract()
					if err != nil {
						setError(err)
						return
					}
					usageGen = usagesResult.ResourceProviderGeneration
					usageMetrics = nil
					for k, v := range usagesResult.Usages {
						usageMetrics = append(usageMetrics,
							cachedMetricEntry{"resource_usage", float64(v), []string{rp.Name, k, traitsStr}},
						)
					}
					if usageGen >= listGen {
						break
					}
					exporter.logger.Warn("Placement usage generation stale, retrying",
						"resource_provider", rp.Name,
						"list_generation", listGen,
						"usage_generation", usageGen,
						"attempt", attempt+1,
					)
					time.Sleep(100 * time.Millisecond)
				}
			} else if existing != nil {
				usageMetrics = existing.usageMetrics
				usageGen = existing.usageGeneration
			}

			// Allocations - retry if generation is stale
			if req.refetchAllocations {
				if _, ok := exporter.Metrics["resource_provider_allocations"]; ok {
					for attempt := 0; attempt < maxStaleRetries; attempt++ {
						allocationsResult, err := resourceproviders.GetAllocations(ctx, exporter.ClientV2, rp.UUID).Extract()
						if err != nil {
							setError(err)
							return
						}
						allocGen = allocationsResult.ResourceProviderGeneration
						allocationMetrics = nil
						for consumerID, allocation := range allocationsResult.Allocations {
							for resourceClass, amount := range allocation.Resources {
								allocationMetrics = append(allocationMetrics,
									cachedMetricEntry{"resource_provider_allocations", float64(amount), []string{rp.Name, consumerID, resourceClass}},
								)
							}
						}
						if allocGen >= listGen {
							break
						}
						exporter.logger.Warn("Placement allocations generation stale, retrying",
							"resource_provider", rp.Name,
							"list_generation", listGen,
							"allocations_generation", allocGen,
							"attempt", attempt+1,
						)
						time.Sleep(100 * time.Millisecond)
					}
				} else {
					allocGen = listGen
				}
			} else if existing != nil {
				allocationMetrics = existing.allocationMetrics
				allocGen = existing.allocationGeneration
			}

			// Build the cache entry
			entry := &cachedProviderData{
				listGeneration:       listGen,
				traitsGeneration:     traitsGen,
				inventoryGeneration:  invGen,
				usageGeneration:      usageGen,
				allocationGeneration: allocGen,
				traitsMetrics:        traitsMetrics,
				inventoryMetrics:     inventoryMetrics,
				usageMetrics:         usageMetrics,
				allocationMetrics:    allocationMetrics,
			}

			// Emit metrics
			emitCachedMetrics(exporter, ch, entry)

			// Store in cache - only if our data is at least as fresh as
			// what's currently stored (avoids TOCTOU with overlapping scrapes)
			if cache != nil {
				cache.mu.Lock()
				if existing, ok := cache.providers[rp.UUID]; !ok || existing.listGeneration <= listGen {
					cache.providers[rp.UUID] = entry
				}
				cache.mu.Unlock()
			}
		}()
	}

	wg.Wait()
	return errCollect
}

func emitCachedMetrics(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric, data *cachedProviderData) {
	for _, entry := range data.allMetrics() {
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
