package cache

import (
	"testing"

	"time"
)

// TestSetAndGetCloudCache tests setting and getting cloud cache data.
func TestInMemoryCachSetAndGetCloudCache(t *testing.T) {
	cache := GetCache()
	defer cache.FlushExpiredCloudCaches(1 * time.Nanosecond)
	cloudName := "testCloud"

	_, exists := cache.GetCloudCache(cloudName)
	if exists {
		t.Errorf("Cloud cache not retrieved properly")
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
	defer cache.FlushExpiredCloudCaches(1 * time.Nanosecond)
	cloudData := NewCloudCache()
	cloudName := "expiredCloud"
	cache.SetCloudCache(cloudName, cloudData)

	time.Sleep(1 * time.Nanosecond)
	cache.FlushExpiredCloudCaches(1 * time.Nanosecond)

	if _, exists := cache.GetCloudCache(cloudName); exists {
		t.Errorf("Expired cloud cache was not flushed")
	}
}

func TestInMemoryCachInit(t *testing.T) {
	cache := InMemoryCache{}
	defer cache.FlushExpiredCloudCaches(1 * time.Nanosecond)
	cloudName := "testCloud"

	cache.init(&cloudName)
	_, exists := cache.GetCloudCache(cloudName)
	if !exists {
		t.Errorf("Init function not works properly")
	}
}

func TestNewCloudCache(t *testing.T) {
	cloudCache := NewCloudCache()
	if cloudCache.Time.IsZero() {
		t.Errorf("Time doesn't been setup properly")
	}
	if cloudCache.MetricFamilyCaches == nil {
		t.Errorf("MetricFamilyCaches doesn't been setup properly")
	}
}

func TestCloudCacheSetMetricFamilyCache(t *testing.T) {
	cache := GetCache()
	defer cache.FlushExpiredCloudCaches(1 * time.Nanosecond)
	cloudCache := NewCloudCache()

	serviceName := "testService"
	mfName := "testMF"
	cloudCache.SetMetricFamilyCache(
		mfName, MetricFamilyCache{MF: nil, Service: serviceName},
	)

	if cloudCache.Time.IsZero() {
		t.Errorf("Time doesn't been setup properly")
	}
	if l := len(cloudCache.MetricFamilyCaches); l != 1 {
		t.Errorf("SetMetricFamilyCache not set value properly")
	}
}
