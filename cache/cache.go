// The cache package provides a concurrency-safe in-memory caching system designed for storing and managing
// hierarchical cache data related to cloud environments, services, and metric families. The system
// is designed around a singleton pattern, ensuring only one instance of the cache backend is created and used
// throughout the application. It supports operations for setting and retrieving cache data at different levels
// (cloud, service, and metric family), with each level structured to contain relevant data and timestamps for
// cache validity checks. Additionally, the cache provides functionality to flush data based on age (TTL) and
// to clear the entire cache.

package cache

import (
	"sync"
	"time"

	dto "github.com/prometheus/client_model/go"
)

var singleCache CacheBackend
var once sync.Once

type CacheBackend interface {
	// Set cache with cloud name.
	SetCloudCache(string, CloudCache)
	// Get cache with cloud name.
	GetCloudCache(string) (CloudCache, bool)
	// Flush expired caches based on cloud's update time.
	FlushExpiredCloudCaches(time.Duration)
}

// MetricFamily Cache Data
type MetricFamilyCache struct {
	Service string
	MF      *dto.MetricFamily
}

// Cloud Cache Data
type CloudCache struct {
	Time               time.Time
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

func (c *InMemoryCache) GetCloudCache(cloud string) (CloudCache, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if cacheData, exists := c.CloudCaches[cloud]; exists {
		return *cacheData, exists
	}
	return CloudCache{}, false
}

func (c *InMemoryCache) SetCloudCache(cloud string, data CloudCache) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.init(&cloud)
	data.Time = time.Now()
	c.CloudCaches[cloud] = &data
}

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

func NewCloudCache() CloudCache {
	cloud := CloudCache{
		Time:               time.Now(),
		MetricFamilyCaches: make(map[string]*MetricFamilyCache),
	}
	return cloud
}

func (c *CloudCache) SetMetricFamilyCache(mfName string, data MetricFamilyCache) {
	c.Time = time.Now()
	c.MetricFamilyCaches[mfName] = &data
}
