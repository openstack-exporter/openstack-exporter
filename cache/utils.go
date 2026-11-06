// Utils provide a series of helper functions.

package cache

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/gophercloud/utils/v2/openstack/clientconfig"
	"github.com/openstack-exporter/openstack-exporter/exporters"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
)

// Source provides the collectors of a cloud, binding the ones that are not
// bound yet. *exporters.Pool implements it.
type Source interface {
	Collectors(ctx context.Context, cloud string, services []string) exporters.Bound
}

// CollectCache gathers the metrics of the required clouds and services from
// source and stores them in the cache. The readiness metrics of the pool are
// stored without a service so every read includes them.
func CollectCache(ctx context.Context, source Source, multiCloud bool, services []string, cloud string, logger *slog.Logger) error {
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
	if cloud != "" && !multiCloud {
		clouds = append(clouds, cloud)
	}

	for _, cloud := range clouds {
		lg := logger.With("cloud", cloud)
		lg.Info("Start update cache data")
		// Update cloud's cache once finish all exporters' collection job. so we won't mix the old
		// and new metrics in the cache and confuse users.
		cloudCache := NewCloudCache()

		bound := source.Collectors(ctx, cloud, services)

		for _, service := range services {
			exp, ok := bound.Exporters[service]
			if !ok {
				lg.Warn("collector is not bound, skipping cache update", "service", service)
				continue
			}
			lg2 := lg.With("service", service)
			lg2.Info("Start collect cache data")
			if err := gatherInto(&cloudCache, service, exp, lg2); err != nil {
				lg2.Error("Create gather failed", "error", err)
				continue
			}
			lg2.Info("Finish update cache data")
		}

		if err := gatherInto(&cloudCache, "", bound.Ready, lg); err != nil {
			lg.Error("Gathering readiness failed", "error", err)
		}

		cacheBackend.SetCloudCache(cloud, cloudCache)
	}

	return nil
}

// gatherInto collects the metric families of collector into cloudCache under
// service.
func gatherInto(cloudCache *CloudCache, service string, collector prometheus.Collector, logger *slog.Logger) error {
	registry := prometheus.NewPedanticRegistry()
	registry.MustRegister(collector)

	metricFamilies, err := registry.Gather()
	if err != nil {
		return err
	}

	for _, mf := range metricFamilies {
		cloudCache.SetMetricFamilyCache(
			*mf.Name,
			MetricFamilyCache{
				Service: service,
				MF:      mf,
			},
		)
		logger.Debug("Update cache data", "MetricsFamily", mf.Name)
	}
	return nil
}

// BufferFromCache reads cloud's MetricsFamily data from cache and writes into a buffer.
func BufferFromCache(cloud string, services []string, logger *slog.Logger) (bytes.Buffer, error) {
	return encodeFromCache(cloud, services, expfmt.NewFormat(expfmt.TypeTextPlain), logger)
}

// encodeFromCache reads cloud's MetricsFamily data from cache and encodes it
// into a buffer using the given exposition format.
func encodeFromCache(cloud string, services []string, format expfmt.Format, logger *slog.Logger) (bytes.Buffer, error) {
	cacheBackend := GetCache()
	var buf bytes.Buffer

	cloudCache, exists := cacheBackend.GetCloudCache(cloud)
	if !exists {
		logger.Debug("Cache not exists", "cloud", cloud)
		return buf, nil
	}

	enc := expfmt.NewEncoder(&buf, format)
	for _, mfCache := range cloudCache.MetricFamilyCaches {
		if mfCache.Service != "" && !slices.Contains(services, mfCache.Service) {
			continue
		}

		if err := enc.Encode(mfCache.MF); err != nil {
			return buf, err
		}
	}

	if closer, ok := enc.(expfmt.Closer); ok {
		if err := closer.Close(); err != nil {
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
	contentType := expfmt.NegotiateAccept(r.Header,
		expfmt.NewFormat(expfmt.TypeProtoDelim),
		expfmt.NewFormat(expfmt.TypeProtoText),
		expfmt.NewFormat(expfmt.TypeProtoCompact),
		expfmt.NewFormat(expfmt.TypeTextPlain),
	)

	// Encode the cached metrics in the negotiated format, so the body matches
	// the Content-Type header (e.g. protobuf when Prometheus asks for it).
	buf, err := encodeFromCache(cloud, enabledServices, contentType, logger)
	if err != nil {
		http.Error(w, "Failed to encode metrics", http.StatusInternalServerError)
		return err
	}

	w.Header().Set("Content-Type", string(contentType))

	if _, err = w.Write(buf.Bytes()); err != nil {
		http.Error(w, "Failed to write cached metrics to response", http.StatusInternalServerError)
		return err
	}

	return nil
}
