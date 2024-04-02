// The cache package provides a concurrency-safe in-memory caching system designed for storing and managing
// hierarchical cache data related to cloud environments, services, and metric families. The system
// is designed around a singleton pattern, ensuring only one instance of the cache backend is created and used
// throughout the application. It supports operations for setting and retrieving cache data at different levels
// (cloud, service, and metric family), with each level structured to contain relevant data and timestamps for
// cache validity checks. Additionally, the cache provides functionality to flush data based on age (TTL) and
// to clear the entire cache.

package cache

import (
	"fmt"
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
	// Set cache with service name.
	SetServiceCache(string, string, ServiceCache)
	// Get cache with service name.
	GetServiceCache(string, string) (ServiceCache, bool)
	// Set cache with metric family name.
	SetMetricFamilyCache(string, string, string, MetricFamilyCache)
	// Get cache with metric family name.
	GetMetricFamilyCache(string, string, string) (MetricFamilyCache, bool)
	// Flush expired caches based on cloud's update time.
	FlushExpiredCloudCaches(time.Duration)
	// Flush all cache data
	Clear()
}

// MetricFamily Cache Data
type MetricFamilyCache struct {
	Time time.Time
	MF   *dto.MetricFamily
}

// Service Cache Data
type ServiceCache struct {
	Time               time.Time
	MetricFamilyCaches map[string]*MetricFamilyCache
}

// Cloud Cache Data
type CloudCache struct {
	Time          time.Time
	ServiceCaches map[string]*ServiceCache
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
func (c *InMemoryCache) init(cloud *string, service *string, mfName *string) {
	if c.CloudCaches == nil {
		c.CloudCaches = make(map[string]*CloudCache)
	}
	if cloud != nil {
		if _, ok := c.CloudCaches[*cloud]; !ok {
			c.CloudCaches[*cloud] = &CloudCache{
				Time:          time.Now(),
				ServiceCaches: make(map[string]*ServiceCache),
			}
		}
	}
	if cloud != nil && service != nil {
		if _, ok := c.CloudCaches[*cloud].ServiceCaches[*service]; !ok {
			c.CloudCaches[*cloud].ServiceCaches[*service] = &ServiceCache{
				Time:               time.Now(),
				MetricFamilyCaches: make(map[string]*MetricFamilyCache),
			}
		}
	}
	if cloud != nil && service != nil && mfName != nil {
		if _, ok := c.CloudCaches[*cloud].ServiceCaches[*service].MetricFamilyCaches[*mfName]; !ok {
			c.CloudCaches[*cloud].ServiceCaches[*service].MetricFamilyCaches[*mfName] = &MetricFamilyCache{
				Time: time.Now(),
			}
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
	c.init(&cloud, nil, nil)
	data.Time = time.Now()
	c.CloudCaches[cloud] = &data
}

func (c *InMemoryCache) GetServiceCache(cloud string, service string) (ServiceCache, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if cloudCache, ok := c.CloudCaches[cloud]; ok {
		if serviceCache, ok := cloudCache.ServiceCaches[service]; ok {
			return *serviceCache, ok
		}
	}
	return ServiceCache{}, false
}

func (c *InMemoryCache) SetServiceCache(cloud string, service string, data ServiceCache) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.init(&cloud, nil, nil)
	data.Time = time.Now()
	c.CloudCaches[cloud].ServiceCaches[service] = &data
}

func (c *InMemoryCache) GetMetricFamilyCache(cloud string, service string, mfName string) (MetricFamilyCache, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if cloudCache, ok := c.CloudCaches[cloud]; ok {
		if serviceCache, ok := cloudCache.ServiceCaches[service]; ok {
			if mfCache, ok := serviceCache.MetricFamilyCaches[mfName]; ok {
				return *mfCache, ok
			}
		}
	}
	return MetricFamilyCache{}, false
}

func (c *InMemoryCache) SetMetricFamilyCache(cloud string, service string, mfName string, data MetricFamilyCache) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.init(&cloud, &service, &mfName)
	data.Time = time.Now()
	c.CloudCaches[cloud].ServiceCaches[service].MetricFamilyCaches[mfName] = &data
}

func (c *InMemoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for key := range c.CloudCaches {
		delete(c.CloudCaches, key)
	}
}

func (c *InMemoryCache) FlushExpiredCloudCaches(ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key, cloudCache := range c.CloudCaches {
		expirationTime := cloudCache.Time.Add(ttl)
		fmt.Println(expirationTime, "\n", time.Now())
		if time.Now().After(expirationTime) {
			delete(c.CloudCaches, key)
		}
	}
}
