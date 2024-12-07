/*
This package implements a caching system for storing and managing cloud-based metric families.
It provides a thread-safe, singleton CacheBackend which manages CloudCache objects.
Each CloudCache can hold multiple MetricFamilyCaches, indexed by the metric family name to avoid duplication.
The system includes functionality to:
- Initialize and retrieve a singleton CacheBackend
- Add or update MetricFamily data in a CloudCache
- Retrieve a CloudCache by cloud name
- Set a new CloudCache in the backend with a current timestamp
- Flush CloudCaches that have not been updated within a specified time-to-live (TTL) period.

An example for using the cloud cache functionality:

```go
// Set MetricFamily in CloudCache object
newCloudCache := NewCloudCache()
newCloudCache.SetMetricFamilyCache("mf-name-a", metricFamilyA)
newCloudCache.SetMetricFamilyCache("mf-name-a", metricFamilyB)

// Get singleton cache backend and atomically set this object in the cache, also setting the cache timestamp.
cache := GetCache()
cache.SetCloudCache("cloudNameA", newCloudCache)

// Get cache data from the cache backend
mycache, exists := cache.GetCloudCache("cloudNameA")

// To ensure caches don't get too stale, call this with a ttl to delete old cloud caches
FlushExpiredCloudCaches(TTL)
```
*/

package cache

import (
	"sync"
	"time"

	dto "github.com/prometheus/client_model/go"
)

var singleCache CacheBackend
var once sync.Once

type CacheBackend interface {
	// Set CloudCache in CacheBackend with cloud name.
	SetCloudCache(cloud string, cloudCache CloudCache)
	// Get CloudCache from CacheBackend with cloud name.
	GetCloudCache(cloud string) (CloudCache, bool)
	// Flush expired caches based on cloud's update time.
	// Cache will be deleted if their update time is older than the ttl.
	FlushExpiredCloudCaches(ttl time.Duration)
}

// MetricFamily Cache Data
type MetricFamilyCache struct {
	Service string
	MF      *dto.MetricFamily
}

// Cloud Cache Data
type CloudCache struct {
	// Latest update time.
	Time time.Time
	// The key of MetricFamilyCaches is metric family name
	// to avoid duplicate MFs in the map.
	MetricFamilyCaches map[string]*MetricFamilyCache
}

// GetCache return a singleton CacheBackend
func GetCache() CacheBackend {
	once.Do(
		func() {
			singleCache = &InMemoryCache{
				CloudCaches: make(map[string]*CloudCache),
			}
		},
	)
	return singleCache
}

// InMemoryCache is a in-memory store based CacheBackend implementation.
type InMemoryCache struct {
	mu          sync.Mutex
	CloudCaches map[string]*CloudCache
}

// init ensures the values are not missing in the nested map.
func (c *InMemoryCache) init(cloud *string) {
	if c.CloudCaches == nil {
		c.CloudCaches = make(map[string]*CloudCache)
	}
	if cloud != nil {
		if _, ok := c.CloudCaches[*cloud]; !ok {
			cloudCache := NewCloudCache()
			c.CloudCaches[*cloud] = &cloudCache
		}
	}
}

// GetCloudCache return CloudCache from in-memory map.
func (c *InMemoryCache) GetCloudCache(cloud string) (CloudCache, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if cacheData, exists := c.CloudCaches[cloud]; exists {
		return *cacheData, exists
	}
	return CloudCache{}, false
}

// SetCloudCache store CloudCache in a in-memory map with key cloud's name.
// The CloudCache's Time attribute will be updated to now.
func (c *InMemoryCache) SetCloudCache(cloud string, data CloudCache) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.init(&cloud)
	data.Time = time.Now()
	c.CloudCaches[cloud] = &data
}

// Flush expired caches based on cloud's update time.
// Cache will be deleted if their update time is older than the ttl.
func (c *InMemoryCache) FlushExpiredCloudCaches(ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key, cloudCache := range c.CloudCaches {
		expirationTime := cloudCache.Time.Add(ttl)
		if time.Now().After(expirationTime) {
			delete(c.CloudCaches, key)
		}
	}
}

// NewCloudCache return a new CloudCache object.
func NewCloudCache() CloudCache {
	cloud := CloudCache{
		Time:               time.Now(),
		MetricFamilyCaches: make(map[string]*MetricFamilyCache),
	}
	return cloud
}

// SetMetricFamilyCache updates the MetricFamilyCaches by associating a key, which is the metric family name.
func (c *CloudCache) SetMetricFamilyCache(mfName string, data MetricFamilyCache) {
	c.MetricFamilyCaches[mfName] = &data
}
