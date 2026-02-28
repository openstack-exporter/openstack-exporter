package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func newSingleCache() {
	singleCache = &InMemoryCache{
		CloudCaches: make(map[string]*CloudCache),
	}
}

// TestSetAndGetCloudCache tests setting and getting cloud cache data.
func TestInMemoryCacheSetAndGetCloudCache(t *testing.T) {
	assert := assert.New(t)

	cache := GetCache()
	defer newSingleCache()
	cloudName := "testCloud"

	_, exists := cache.GetCloudCache(cloudName)
	assert.False(exists, "Cloud cache reported to exist, but it shouldn't")

	cloudData := NewCloudCache()

	cache.SetCloudCache(cloudName, cloudData)
	retrievedCloudData, exists := cache.GetCloudCache(cloudName)
	assert.True(exists, "Cloud cache was not set")
	assert.NotZero(retrievedCloudData.Time, "Cloud cache was not retrieved properly")
}

// TestFlushExpiredCloudCaches tests flushing of expired cloud caches.
func TestInMemoryCacheFlushExpiredCloudCaches(t *testing.T) {
	assert := assert.New(t)

	cache := GetCache()
	defer newSingleCache()

	cloudData := NewCloudCache()
	cloudName := "expiredCloud"
	cache.SetCloudCache(cloudName, cloudData)

	time.Sleep(2 * time.Nanosecond)
	cache.FlushExpiredCloudCaches(1 * time.Nanosecond)

	_, exists := cache.GetCloudCache(cloudName)
	assert.False(exists, "Expired cloud cache was not flushed")
}

func TestInMemoryCachInit(t *testing.T) {
	assert := assert.New(t)

	cache := InMemoryCache{}
	defer newSingleCache()
	cloudName := "testCloud"

	cache.init(&cloudName)
	_, exists := cache.GetCloudCache(cloudName)
	assert.True(exists, "Init function did not init cache for cloud")
}

func TestNewCloudCache(t *testing.T) {
	assert := assert.New(t)

	cloudCache := NewCloudCache()
	assert.NotZero(cloudCache.Time, "Time doesn't been setup properly")
	assert.NotNil(cloudCache.MetricFamilyCaches, "MetricFamilyCaches not initialised correctly")
}

func TestCloudCacheSetMetricFamilyCache(t *testing.T) {
	assert := assert.New(t)

	cloudCache := NewCloudCache()

	serviceName := "testService"
	mfName := "testMF"
	cloudCache.SetMetricFamilyCache(
		mfName, MetricFamilyCache{MF: nil, Service: serviceName},
	)

	assert.NotZero(cloudCache.Time, "CloudCache.Time was not set")
	assert.Len(cloudCache.MetricFamilyCaches, 1, "SetMetricFamilyCache value not set")
}
