package cache

import (
	"testing"
	"time"
)

func newSingleCache() {
	singleCache = &InMemoryCache{
		CloudCaches: make(map[string]*CloudCache),
	}
}

// TestSetAndGetCloudCache tests setting and getting cloud cache data.
func TestInMemoryCachSetAndGetCloudCache(t *testing.T) {
	cache := GetCache()
	defer newSingleCache()
	cloudName := "testCloud"

	_, exists := cache.GetCloudCache(cloudName)
	if exists {
		t.Errorf("Cloud cache reported to exist, but it shouldn't")
	}
	cloudData := NewCloudCache()

	cache.SetCloudCache(cloudName, cloudData)
	retrievedCloudData, exists := cache.GetCloudCache(cloudName)

	if !exists || retrievedCloudData.Time.IsZero() {
		t.Errorf("Cloud cache was not set or retrieved properly")
	}
}

// TestFlushExpiredCloudCaches tests flushing of expired cloud caches.
func TestInMemoryCacheFlushExpiredCloudCaches(t *testing.T) {
	cache := GetCache()
	defer newSingleCache()
	cloudData := NewCloudCache()
	cloudName := "expiredCloud"
	cache.SetCloudCache(cloudName, cloudData)

	time.Sleep(2 * time.Nanosecond)
	cache.FlushExpiredCloudCaches(1 * time.Nanosecond)

	if _, exists := cache.GetCloudCache(cloudName); exists {
		t.Errorf("Expired cloud cache was not flushed")
	}
}

func TestInMemoryCachInit(t *testing.T) {
	cache := InMemoryCache{}
	defer newSingleCache()
	cloudName := "testCloud"

	cache.init(&cloudName)
	_, exists := cache.GetCloudCache(cloudName)
	if !exists {
		t.Errorf("Init function did not init cache for cloud")
	}
}

func TestNewCloudCache(t *testing.T) {
	cloudCache := NewCloudCache()
	if cloudCache.Time.IsZero() {
		t.Errorf("Time doesn't been setup properly")
	}
	if cloudCache.MetricFamilyCaches == nil {
		t.Errorf("MetricFamilyCaches not initialised correctly")
	}
}

func TestCloudCacheSetMetricFamilyCache(t *testing.T) {
	cloudCache := NewCloudCache()

	serviceName := "testService"
	mfName := "testMF"
	cloudCache.SetMetricFamilyCache(
		mfName, MetricFamilyCache{MF: nil, Service: serviceName},
	)

	if cloudCache.Time.IsZero() {
		t.Errorf("CloudCache.Time was not set")
	}
	if len(cloudCache.MetricFamilyCaches) != 1 {
		t.Errorf("SetMetricFamilyCache value not set")
	}
}
