package exporters

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

func TestPlacementCache(t *testing.T) {
	for _, parallel := range []bool{false, true} {
		t.Run(fmt.Sprintf("parallel=%v", parallel), func(t *testing.T) {
			generation, inventoryGeneration, usage := 0, 0, 1
			traitFailure := false
			var mu sync.Mutex
			calls := map[string]int{}
			count := func(path string) int {
				mu.Lock()
				defer mu.Unlock()
				return calls[path]
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				calls[r.URL.Path]++
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/resource_providers":
					fmt.Fprintf(w, `{"resource_providers":[{"uuid":"rp","name":"host","generation":%d}]}`, generation)
				case "/resource_providers/rp/traits":
					if traitFailure {
						w.WriteHeader(http.StatusServiceUnavailable)
						return
					}
					fmt.Fprintf(w, `{"resource_provider_generation":%d,"traits":["CUSTOM_TEST"]}`, generation)
				case "/resource_providers/rp/inventories":
					fmt.Fprintf(w, `{"resource_provider_generation":%d,"inventories":{"VCPU":{"total":10,"allocation_ratio":1}}}`, inventoryGeneration)
				case "/resource_providers/rp/usages":
					fmt.Fprintf(w, `{"resource_provider_generation":%d,"usages":{"VCPU":%d}}`, generation, usage)
				case "/resource_providers/rp/allocations":
					fmt.Fprintf(w, `{"resource_provider_generation":%d,"allocations":{"consumer":{"resources":{"VCPU":%d}}}}`, generation, usage)
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			config := &ExporterConfig{
				ClientV2: &gophercloud.ServiceClient{
					ProviderClient: &gophercloud.ProviderClient{HTTPClient: *server.Client()},
					Endpoint:       server.URL + "/",
				},
				Prefix: "openstack", CollectPlacementTraits: true, CompletePlacementInParallel: parallel,
			}
			exporter, err := NewPlacementExporter(config, slog.Default())
			require.NoError(t, err)
			scrape := func(pe *PlacementExporter) (map[string]float64, error) {
				ch := make(chan prometheus.Metric, 20)
				err := pe.listWithCache(context.Background(), &pe.BaseOpenStackExporter, ch)
				close(ch)
				values := map[string]float64{}
				for metric := range ch {
					var result dto.Metric
					require.NoError(t, metric.Write(&result))
					values[metric.Desc().String()] = result.GetGauge().GetValue()
				}
				return values, err
			}
			checkDynamic := func(values map[string]float64) {
				for _, name := range []string{"resource_usage", "resource_provider_allocations"} {
					require.Equal(t, float64(usage), values[exporter.Metrics[name].Metric.String()])
				}
			}
			values, err := scrape(exporter)
			require.NoError(t, err)
			checkDynamic(values)
			mu.Lock()
			usage = 2 // Allocation changes without a provider generation change.
			mu.Unlock()
			values, err = scrape(exporter)
			require.NoError(t, err)
			checkDynamic(values)
			require.Equal(t, 1, count("/resource_providers/rp/inventories"))
			require.Equal(t, 1, count("/resource_providers/rp/traits"))
			require.Equal(t, 2, count("/resource_providers/rp/usages"))
			require.Equal(t, 2, count("/resource_providers/rp/allocations"))

			// A second cloud at the same endpoint must fetch its own data.
			other, err := NewPlacementExporter(config, slog.Default())
			require.NoError(t, err)
			_, err = scrape(other)
			require.NoError(t, err)
			require.Equal(t, 2, count("/resource_providers/rp/inventories"))
			require.Equal(t, 2, count("/resource_providers/rp/traits"))

			// A failed trait read at generation zero must not become a cache hit.
			other, err = NewPlacementExporter(config, slog.Default())
			require.NoError(t, err)
			mu.Lock()
			traitFailure = true
			mu.Unlock()
			values, err = scrape(other)
			require.Error(t, err)
			require.Empty(t, values)
			require.Empty(t, other.cache.providers)
			mu.Lock()
			traitFailure = false
			mu.Unlock()
			_, err = scrape(other)
			require.NoError(t, err)
			require.Equal(t, "CUSTOM_TEST", other.cache.providers["rp"].traits)

			// A mismatched inventory generation is served once but never cached.
			mu.Lock()
			generation = 1
			mu.Unlock()
			_, err = scrape(exporter)
			require.NoError(t, err)
			require.Equal(t, 0, exporter.cache.providers["rp"].generation)
			mu.Lock()
			inventoryGeneration = 1
			mu.Unlock()
			_, err = scrape(exporter)
			require.NoError(t, err)
			require.Equal(t, 1, exporter.cache.providers["rp"].generation)
		})
	}
}
