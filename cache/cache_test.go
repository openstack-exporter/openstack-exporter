package cache

import (
	"testing"

	"time"

	dto "github.com/prometheus/client_model/go"
)

// TestSetAndGetCloudCache tests setting and getting cloud cache data.
func TestInMemoryCachSetAndGetCloudCache(t *testing.T) {
	cache := GetCache()
	defer cache.Clear()
	cloudName := "testCloud"

	_, exists := cache.GetCloudCache(cloudName)
	if exists {
		t.Errorf("Cloud cache not retrieved properly")
	}
	cloudData := CloudCache{
		Time:          time.Now(),
		ServiceCaches: make(map[string]*ServiceCache),
	}

	cache.SetCloudCache(cloudName, cloudData)
	retrievedCloudData, exists := cache.GetCloudCache(cloudName)

	if !exists || retrievedCloudData.Time.IsZero() {
		t.Errorf("Cloud cache was not set or retrieved properly")
	}
}

// TestSetAndGetServiceCache tests setting and getting service cache data.
func TestInMemoryCachSetAndGetServiceCache(t *testing.T) {
	cache := GetCache()
	defer cache.Clear()
	cloudName := "testCloud"
	serviceName := "testService"

	_, exists := cache.GetServiceCache(cloudName, serviceName)
	if exists {
		t.Errorf("Cloud cache not retrieved properly")
	}

	serviceData := ServiceCache{
		MetricFamilyCaches: make(map[string]*MetricFamilyCache),
	}

	cache.SetServiceCache(cloudName, serviceName, serviceData)
	retrievedServiceData, exists := cache.GetServiceCache(cloudName, serviceName)

	if !exists || retrievedServiceData.Time.IsZero() {
		t.Errorf("Service cache was not set or retrieved properly")
	}
}

// TestSetAndGetMetricFamilyCache tests setting and getting metric family cache data.
func TestInMemoryCachSetAndGetMetricFamilyCache(t *testing.T) {
	cache := GetCache()
	defer cache.Clear()
	cloudName := "testCloud"
	serviceName := "testService"
	mfName := "testMetricFamily"

	_, exists := cache.GetMetricFamilyCache(cloudName, serviceName, mfName)
	if exists {
		t.Errorf("Cloud cache not retrieved properly")
	}

	mfData := MetricFamilyCache{
		MF: &dto.MetricFamily{},
	}

	cache.SetMetricFamilyCache(cloudName, serviceName, mfName, mfData)
	retrievedMfData, exists := cache.GetMetricFamilyCache(cloudName, serviceName, mfName)

	if !exists || retrievedMfData.Time.IsZero() {
		t.Errorf("Metric family cache was not set or retrieved properly")
	}
}

// TestFlushExpiredCloudCaches tests flushing of expired cloud caches.
func TestInMemoryCacheFlushExpiredCloudCaches(t *testing.T) {
	cache := GetCache()
	defer cache.Clear()
	cloudData := CloudCache{
		ServiceCaches: make(map[string]*ServiceCache),
	}
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
	defer cache.Clear()
	cloudName := "testCloud"
	serviceName := "testService"
	mfName := "testMetricFamily"

	cache.init(&cloudName, &serviceName, &mfName)
	_, exists := cache.GetMetricFamilyCache(cloudName, serviceName, mfName)
	if !exists {
		t.Errorf("Init function not works properly")
	}
}
