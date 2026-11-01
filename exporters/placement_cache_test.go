package exporters

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPlacementCacheStaleAPIs(t *testing.T) {
	// Test that staleAPIs correctly identifies mismatched generations
	data := &cachedProviderData{
		listGeneration:       10,
		traitsGeneration:     10,
		inventoryGeneration:  10,
		usageGeneration:      8, // stale
		allocationGeneration: 10,
	}

	traits, inv, usage, alloc := data.staleAPIs(10)
	assert.False(t, traits, "traits should not be stale")
	assert.False(t, inv, "inventory should not be stale")
	assert.True(t, usage, "usage should be stale (8 != 10)")
	assert.False(t, alloc, "allocations should not be stale")
}

func TestPlacementCacheAllFresh(t *testing.T) {
	data := &cachedProviderData{
		listGeneration:       10,
		traitsGeneration:     10,
		inventoryGeneration:  10,
		usageGeneration:      10,
		allocationGeneration: 10,
	}

	traits, inv, usage, alloc := data.staleAPIs(10)
	assert.False(t, traits)
	assert.False(t, inv)
	assert.False(t, usage)
	assert.False(t, alloc)
}

func TestPlacementCacheTraitsStaleTriggersFullRefetch(t *testing.T) {
	// When traits are stale, we should NOT do a partial refetch
	// (the calling code promotes to full refetch)
	data := &cachedProviderData{
		listGeneration:       10,
		traitsGeneration:     9, // stale
		inventoryGeneration:  10,
		usageGeneration:      10,
		allocationGeneration: 10,
	}

	traits, inv, _, _ := data.staleAPIs(10)
	assert.True(t, traits, "traits stale should be detected")
	assert.False(t, inv, "inventory is fresh")
	// In the calling code, stale traits triggers full refetch
}

func TestPlacementCacheTraitsErrorSetsZeroGeneration(t *testing.T) {
	// When traits fetch fails, generation should be 0 (not listGen)
	// so it's detected as stale on next scrape
	data := &cachedProviderData{
		listGeneration:       10,
		traitsGeneration:     0, // failed fetch
		inventoryGeneration:  10,
		usageGeneration:      10,
		allocationGeneration: 10,
	}

	traits, _, _, _ := data.staleAPIs(10)
	assert.True(t, traits, "zero generation (failed traits) should be detected as stale")
}

func TestPlacementCacheAllMetrics(t *testing.T) {
	data := &cachedProviderData{
		traitsMetrics:     []cachedMetricEntry{{"resource_traits", 1.0, []string{"host", "CUSTOM_A"}}},
		inventoryMetrics:  []cachedMetricEntry{{"resource_total", 96, []string{"host", "VCPU", "CUSTOM_A"}}},
		usageMetrics:      []cachedMetricEntry{{"resource_usage", 10, []string{"host", "VCPU", "CUSTOM_A"}}},
		allocationMetrics: []cachedMetricEntry{{"resource_provider_allocations", 4, []string{"host", "uuid1", "VCPU"}}},
	}

	all := data.allMetrics()
	assert.Equal(t, 4, len(all))
	assert.Equal(t, "resource_traits", all[0].metricName)
	assert.Equal(t, "resource_total", all[1].metricName)
	assert.Equal(t, "resource_usage", all[2].metricName)
	assert.Equal(t, "resource_provider_allocations", all[3].metricName)
}
