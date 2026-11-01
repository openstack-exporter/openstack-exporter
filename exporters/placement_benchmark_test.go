package exporters_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/openstack-exporter/openstack-exporter/cache"
	"github.com/openstack-exporter/openstack-exporter/exporters"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	placementBenchmarkResourceProviderCount = 10000
	placementBenchmarkCloud                 = "placement-benchmark"
	placementBenchmarkService               = "placement"
)

type placementBenchmarkResourceProvider struct {
	Generation int    `json:"generation"`
	UUID       string `json:"uuid"`
	Name       string `json:"name"`
}

type placementBenchmarkFixture struct {
	server    *httptest.Server
	exporter  *exporters.PlacementExporter
	providers []placementBenchmarkResourceProvider
	delay     time.Duration
}

func BenchmarkPlacementListResourceProviders10000(b *testing.B) {
	for _, parallel := range []bool{false, true} {
		b.Run(placementBenchmarkParallelName(parallel), func(b *testing.B) {
			fixture := newPlacementBenchmarkFixture(b, parallel)
			defer fixture.server.Close()

			ch := make(chan prometheus.Metric)
			done := drainPlacementBenchmarkMetrics(ch)
			b.Cleanup(func() {
				close(ch)
				<-done
			})

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := exporters.ListPlacementResourceProviders(context.Background(), &fixture.exporter.BaseOpenStackExporter, ch); err != nil {
					b.Fatalf("ListPlacementResourceProviders failed: %v", err)
				}
			}
		})
	}
}

func BenchmarkPlacementCollectCold10000(b *testing.B) {
	for _, parallel := range []bool{false, true} {
		b.Run(placementBenchmarkParallelName(parallel), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				fixture := newPlacementBenchmarkFixture(b, parallel)
				registry := prometheus.NewPedanticRegistry()
				registry.MustRegister(fixture.exporter)

				b.StartTimer()
				if _, err := registry.Gather(); err != nil {
					b.Fatalf("cold gather failed: %v", err)
				}
				b.StopTimer()
				fixture.server.Close()
			}
		})
	}
}

func BenchmarkPlacementCollectWarm10000(b *testing.B) {
	for _, parallel := range []bool{false, true} {
		b.Run(placementBenchmarkParallelName(parallel), func(b *testing.B) {
			fixture := newPlacementBenchmarkFixture(b, parallel)
			defer fixture.server.Close()

			registry := prometheus.NewPedanticRegistry()
			registry.MustRegister(fixture.exporter)
			if _, err := registry.Gather(); err != nil {
				b.Fatalf("warm-up gather failed: %v", err)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := registry.Gather(); err != nil {
					b.Fatalf("warm gather failed: %v", err)
				}
			}
		})
	}
}

func BenchmarkPlacementCacheWrite10000(b *testing.B) {
	fixture := newPlacementBenchmarkFixture(b, true)
	defer fixture.server.Close()

	registry := prometheus.NewPedanticRegistry()
	registry.MustRegister(fixture.exporter)
	metricFamilies, err := registry.Gather()
	if err != nil {
		b.Fatalf("initial gather failed: %v", err)
	}

	cloudCache := cache.NewCloudCache()
	for _, mf := range metricFamilies {
		cloudCache.SetMetricFamilyCache(*mf.Name, cache.MetricFamilyCache{
			Service: placementBenchmarkService,
			MF:      mf,
		})
	}
	cache.GetCache().SetCloudCache(placementBenchmarkCloud, cloudCache)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	request := httptest.NewRequest(http.MethodGet, "/metrics", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		response := httptest.NewRecorder()
		if err := cache.WriteCacheToResponse(response, request, placementBenchmarkCloud, []string{placementBenchmarkService}, logger); err != nil {
			b.Fatalf("cache write failed: %v", err)
		}
		if response.Code != http.StatusOK {
			b.Fatalf("cache write returned status %d", response.Code)
		}
	}
}

func newPlacementBenchmarkFixture(b *testing.B, parallel bool) *placementBenchmarkFixture {
	b.Helper()

	fixture := &placementBenchmarkFixture{
		providers: makePlacementBenchmarkResourceProviders(placementBenchmarkResourceProviderCount),
		delay:     placementBenchmarkRequestDelay(b),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if fixture.delay > 0 {
			time.Sleep(fixture.delay)
		}

		switch {
		case r.URL.Path == "/resource_providers":
			writePlacementBenchmarkResourceProviders(b, w, fixture.providers)
		case strings.HasPrefix(r.URL.Path, "/resource_providers/"):
			writePlacementBenchmarkResourceProviderDetail(b, w, r.URL.Path, fixture.providers)
		default:
			http.NotFound(w, r)
		}
	})

	fixture.server = httptest.NewServer(mux)
	client := &gophercloud.ServiceClient{
		ProviderClient: &gophercloud.ProviderClient{
			HTTPClient: *fixture.server.Client(),
		},
		Endpoint:     fixture.server.URL + "/",
		ResourceBase: fixture.server.URL + "/",
		Type:         placementBenchmarkService,
		Microversion: "1.39",
	}

	config := &exporters.ExporterConfig{
		ClientV2:        client,
		ServiceName:     placementBenchmarkService,
		Prefix:          "openstack",
		CollectTime:     true,
		DisabledMetrics: []string{},
	}
	setPlacementBenchmarkBoolField(config, "CompletePlacementInParallel", parallel)
	setPlacementBenchmarkBoolField(config, "CollectPlacementTraits", false)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	exporter, err := exporters.NewPlacementExporter(config, logger)
	if err != nil {
		fixture.server.Close()
		b.Fatalf("failed to create placement exporter: %v", err)
	}
	fixture.exporter = exporter

	return fixture
}

func placementBenchmarkParallelName(parallel bool) string {
	if parallel {
		return "parallel_enabled"
	}
	return "parallel_disabled"
}

func setPlacementBenchmarkBoolField(config *exporters.ExporterConfig, fieldName string, value bool) {
	field := reflect.ValueOf(config).Elem().FieldByName(fieldName)
	if field.IsValid() && field.CanSet() && field.Kind() == reflect.Bool {
		field.SetBool(value)
	}
}

func drainPlacementBenchmarkMetrics(ch <-chan prometheus.Metric) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		for range ch {
		}
		close(done)
	}()
	return done
}

func makePlacementBenchmarkResourceProviders(count int) []placementBenchmarkResourceProvider {
	providers := make([]placementBenchmarkResourceProvider, count)
	for i := range count {
		providers[i] = placementBenchmarkResourceProvider{
			Generation: 1000 + i,
			UUID:       fmt.Sprintf("00000000-0000-4000-8000-%012d", i),
			Name:       fmt.Sprintf("compute-%04d.example.org", i),
		}
	}
	return providers
}

func writePlacementBenchmarkResourceProviders(b *testing.B, w http.ResponseWriter, providers []placementBenchmarkResourceProvider) {
	b.Helper()
	if err := json.NewEncoder(w).Encode(struct {
		ResourceProviders []placementBenchmarkResourceProvider `json:"resource_providers"`
	}{ResourceProviders: providers}); err != nil {
		b.Fatalf("failed to encode resource providers: %v", err)
	}
}

func writePlacementBenchmarkResourceProviderDetail(b *testing.B, w http.ResponseWriter, path string, providers []placementBenchmarkResourceProvider) {
	b.Helper()

	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 3 || parts[0] != "resource_providers" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	provider, ok := findPlacementBenchmarkResourceProvider(parts[1], providers)
	if !ok {
		http.Error(w, "unknown resource provider", http.StatusNotFound)
		return
	}

	switch parts[2] {
	case "inventories":
		writePlacementBenchmarkInventories(b, w, provider)
	case "usages":
		writePlacementBenchmarkUsages(b, w, provider)
	case "allocations":
		writePlacementBenchmarkAllocations(b, w, provider)
	case "traits":
		writePlacementBenchmarkTraits(b, w, provider)
	default:
		http.Error(w, "not found", http.StatusNotFound)
	}
}

func findPlacementBenchmarkResourceProvider(uuid string, providers []placementBenchmarkResourceProvider) (placementBenchmarkResourceProvider, bool) {
	for _, provider := range providers {
		if provider.UUID == uuid {
			return provider, true
		}
	}
	return placementBenchmarkResourceProvider{}, false
}

func writePlacementBenchmarkInventories(b *testing.B, w http.ResponseWriter, provider placementBenchmarkResourceProvider) {
	b.Helper()
	response := map[string]any{
		"resource_provider_generation": provider.Generation,
		"inventories": map[string]any{
			"VCPU": map[string]any{
				"total": 96, "reserved": 0, "min_unit": 1, "max_unit": 96, "step_size": 1, "allocation_ratio": 16.0,
			},
			"MEMORY_MB": map[string]any{
				"total": 772447, "reserved": 8192, "min_unit": 1, "max_unit": 772447, "step_size": 1, "allocation_ratio": 1.5,
			},
			"DISK_GB": map[string]any{
				"total": 2047, "reserved": 0, "min_unit": 1, "max_unit": 2047, "step_size": 1, "allocation_ratio": 1.0,
			},
		},
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		b.Fatalf("failed to encode inventories: %v", err)
	}
}

func writePlacementBenchmarkUsages(b *testing.B, w http.ResponseWriter, provider placementBenchmarkResourceProvider) {
	b.Helper()
	response := map[string]any{
		"resource_provider_generation": provider.Generation,
		"usages": map[string]int{
			"VCPU":      provider.Generation % 96,
			"MEMORY_MB": 1024 * (provider.Generation % 128),
			"DISK_GB":   provider.Generation % 512,
		},
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		b.Fatalf("failed to encode usages: %v", err)
	}
}

func writePlacementBenchmarkAllocations(b *testing.B, w http.ResponseWriter, provider placementBenchmarkResourceProvider) {
	b.Helper()
	response := map[string]any{
		"resource_provider_generation": provider.Generation,
		"allocations": map[string]any{
			fmt.Sprintf("10000000-0000-4000-8000-%012d", provider.Generation): map[string]any{
				"resources": map[string]int{
					"VCPU":      2,
					"MEMORY_MB": 4096,
					"DISK_GB":   40,
				},
			},
		},
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		b.Fatalf("failed to encode allocations: %v", err)
	}
}

func writePlacementBenchmarkTraits(b *testing.B, w http.ResponseWriter, provider placementBenchmarkResourceProvider) {
	b.Helper()
	response := map[string]any{
		"resource_provider_generation": provider.Generation,
		"traits": []string{
			"CUSTOM_BENCHMARK",
			fmt.Sprintf("CUSTOM_RACK_%02d", provider.Generation%40),
			"HW_CPU_X86_AVX2",
		},
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		b.Fatalf("failed to encode traits: %v", err)
	}
}

func placementBenchmarkRequestDelay(b *testing.B) time.Duration {
	b.Helper()

	rawDelay := os.Getenv("PLACEMENT_BENCH_REQUEST_DELAY_MS")
	if rawDelay == "" {
		return 0
	}
	delayMilliseconds, err := strconv.Atoi(rawDelay)
	if err != nil {
		b.Fatalf("invalid PLACEMENT_BENCH_REQUEST_DELAY_MS %q: %v", rawDelay, err)
	}
	if delayMilliseconds < 0 {
		b.Fatalf("invalid PLACEMENT_BENCH_REQUEST_DELAY_MS %q: must be non-negative", rawDelay)
	}
	return time.Duration(delayMilliseconds) * time.Millisecond
}
