// Utils provide a series of helper functions.

package cache

import (
	"bytes"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/gophercloud/utils/v2/openstack/clientconfig"
	"github.com/openstack-exporter/openstack-exporter/exporters"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/expfmt"
)

// CollectCache collects the MetricsFamily for required clouds and services and stores in the cache.
func CollectCache(
	enableExporterFunc func(string, exporters.ExporterOptions, *slog.Logger) (*exporters.OpenStackExporter, error),
	multiCloud bool,
	services []string,
	opts exporters.ExporterOptions,
	logger *slog.Logger,
) error {
	logger.Info("Run collect cache job")
	cacheBackend := GetCache()

	clouds := []string{}

	if multiCloud {
		cloudsConfig, err := clientconfig.LoadCloudsYAML()
		if err != nil {
			return err
		}

		for cloud := range cloudsConfig {
			clouds = append(clouds, cloud)
		}
	}
	if opts.Cloud != "" && !multiCloud {
		clouds = append(clouds, opts.Cloud)
	}

	for _, cloud := range clouds {
		lg := logger.With("cloud", cloud)
		lg.Info("Start update cache data")
		// Update cloud's cache once finish all exporters' collection job. so we won't mix the old
		// and new metrics in the cache and confuse users.
		cloudCache := NewCloudCache()

		cloudOpts := opts
		cloudOpts.Cloud = cloud

		for _, service := range services {
			lg2 := lg.With("service", service)
			lg2.Info("Start collect cache data")

			exp, err := enableExporterFunc(service, cloudOpts, logger)
			if err != nil {
				// Log error and continue with enabling other exporters
				lg2.Error("enabling exporter for service failed", "error", err)
				continue
			}

			registry := prometheus.NewPedanticRegistry()
			registry.MustRegister(*exp)

			metricFamilies, err := registry.Gather()
			if err != nil {
				lg2.Error("Create gather failed", "error", err)
				continue
			}

			for _, mf := range metricFamilies {
				cloudCache.SetMetricFamilyCache(
					*mf.Name,
					MetricFamilyCache{
						Service: service,
						MF:      mf,
					},
				)
				lg2.Debug("Update cache data", "MetricsFamily", mf.Name)
			}

			lg2.Info("Finish update cache data")
		}

		cacheBackend.SetCloudCache(cloud, cloudCache)
	}

	return nil
}

// BufferFromCache reads cloud's MetricsFamily data from cache and writes into a buffer.
func BufferFromCache(cloud string, services []string, logger *slog.Logger) (bytes.Buffer, error) {
	cacheBackend := GetCache()
	var buf bytes.Buffer

	cloudCache, exists := cacheBackend.GetCloudCache(cloud)
	if !exists {
		logger.Debug("Cache not exists", "cloud", cloud)
		return buf, nil
	}

	for _, mfCache := range cloudCache.MetricFamilyCaches {
		if !slices.Contains(services, mfCache.Service) {
			continue
		}

		if _, err := expfmt.MetricFamilyToText(&buf, mfCache.MF); err != nil {
			return buf, err
		}
	}

	return buf, nil
}

// FlushExpiredCloudCaches flush expired caches based on cloud's update time
func FlushExpiredCloudCaches(ttl time.Duration) {
	cacheBackend := GetCache()
	cacheBackend.FlushExpiredCloudCaches(ttl)
}

// WriteCacheToResponse read cache and write to the connection as part of an HTTP reply.
func WriteCacheToResponse(w http.ResponseWriter, r *http.Request, cloud string, enabledServices []string, logger *slog.Logger) error {
	buf, err := BufferFromCache(cloud, enabledServices, logger)
	if err != nil {
		http.Error(w, "Failed to encode metrics", http.StatusInternalServerError)
		return err
	}

	opts := promhttp.HandlerOpts{}

	// Follow the way how promehttp package set up the contentType
	var contentType expfmt.Format
	if opts.EnableOpenMetrics {
		contentType = expfmt.NegotiateIncludingOpenMetrics(r.Header)
	} else {
		contentType = expfmt.Negotiate(r.Header)
	}
	w.Header().Set("Content-Type", string(contentType))

	if _, err = w.Write(buf.Bytes()); err != nil {
		http.Error(w, "Failed to write cached metrics to response", http.StatusInternalServerError)
		return err
	}

	return nil
}
