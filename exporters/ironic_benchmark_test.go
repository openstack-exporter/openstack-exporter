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
	"strconv"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/openstack-exporter/openstack-exporter/cache"
	"github.com/openstack-exporter/openstack-exporter/exporters"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	ironicBenchmarkNodeCount = 1000
	ironicBenchmarkCloud     = "ironic-benchmark"
	ironicBenchmarkService   = "baremetal"
)

type benchmarkIronicNode struct {
	UUID                 string         `json:"uuid"`
	Name                 string         `json:"name"`
	ProvisionState       string         `json:"provision_state"`
	PowerState           string         `json:"power_state"`
	Maintenance          bool           `json:"maintenance"`
	MaintenanceReason    string         `json:"maintenance_reason"`
	ConsoleEnabled       bool           `json:"console_enabled"`
	ResourceClass        string         `json:"resource_class"`
	DriverInfo           map[string]any `json:"driver_info"`
	UpdatedAt            string         `json:"updated_at"`
	ProvisionUpdatedAt   string         `json:"provision_updated_at"`
	Retired              bool           `json:"retired"`
	RetiredReason        string         `json:"retired_reason"`
	Driver               string         `json:"driver"`
	BootInterface        string         `json:"boot_interface"`
	DeployInterface      string         `json:"deploy_interface"`
	ManagementInterface  string         `json:"management_interface"`
	PowerInterface       string         `json:"power_interface"`
	NetworkInterface     string         `json:"network_interface"`
	Properties           map[string]any `json:"properties"`
	Extra                map[string]any `json:"extra"`
	InstanceInfo         map[string]any `json:"instance_info"`
	DriverInternalInfo   map[string]any `json:"driver_internal_info"`
	TargetPowerState     *string        `json:"target_power_state"`
	TargetProvisionState *string        `json:"target_provision_state"`
	LastError            *string        `json:"last_error"`
	Reservation          *string        `json:"reservation"`
	InstanceUUID         *string        `json:"instance_uuid"`
	ChassisUUID          string         `json:"chassis_uuid"`
	AllocationUUID       *string        `json:"allocation_uuid"`
	Fault                *string        `json:"fault"`
	Conductor            string         `json:"conductor"`
	ConductorGroup       string         `json:"conductor_group"`
	Protected            bool           `json:"protected"`
	ProtectedReason      *string        `json:"protected_reason"`
	AutomatedClean       *bool          `json:"automated_clean"`
	InspectionStartedAt  *string        `json:"inspection_started_at"`
	InspectionFinishedAt *string        `json:"inspection_finished_at"`
	CleanStep            map[string]any `json:"clean_step"`
	DeployStep           map[string]any `json:"deploy_step"`
	RaidConfig           map[string]any `json:"raid_config"`
	TargetRaidConfig     map[string]any `json:"target_raid_config"`
	Traits               []string       `json:"traits"`
	Owner                *string        `json:"owner"`
	CreatedAt            string         `json:"created_at"`
}

type ironicBenchmarkFixture struct {
	server       *httptest.Server
	exporter     *exporters.IronicExporter
	requestCount int
}

func BenchmarkIronicListNodes1000(b *testing.B) {
	for _, pageSize := range []int{1000, 100} {
		b.Run(fmt.Sprintf("page_size_%d", pageSize), func(b *testing.B) {
			fixture := newIronicBenchmarkFixture(b, pageSize)
			defer fixture.server.Close()

			ch := make(chan prometheus.Metric)
			done := drainMetrics(ch)
			b.Cleanup(func() {
				close(ch)
				<-done
			})

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := exporters.ListNodes(context.Background(), &fixture.exporter.BaseOpenStackExporter, ch); err != nil {
					b.Fatalf("ListNodes failed: %v", err)
				}
			}
		})
	}
}

func BenchmarkIronicCollect1000(b *testing.B) {
	for _, pageSize := range []int{1000, 100} {
		b.Run(fmt.Sprintf("page_size_%d", pageSize), func(b *testing.B) {
			fixture := newIronicBenchmarkFixture(b, pageSize)
			defer fixture.server.Close()

			registry := prometheus.NewPedanticRegistry()
			registry.MustRegister(fixture.exporter)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := registry.Gather(); err != nil {
					b.Fatalf("gather failed: %v", err)
				}
			}
		})
	}
}

func BenchmarkIronicCacheWrite1000(b *testing.B) {
	for _, pageSize := range []int{1000, 100} {
		b.Run(fmt.Sprintf("page_size_%d", pageSize), func(b *testing.B) {
			fixture := newIronicBenchmarkFixture(b, pageSize)
			defer fixture.server.Close()

			registry := prometheus.NewPedanticRegistry()
			registry.MustRegister(fixture.exporter)
			metricFamilies, err := registry.Gather()
			if err != nil {
				b.Fatalf("initial gather failed: %v", err)
			}

			cloud := fmt.Sprintf("%s-page-%d", ironicBenchmarkCloud, pageSize)
			cloudCache := cache.NewCloudCache()
			for _, mf := range metricFamilies {
				cloudCache.SetMetricFamilyCache(*mf.Name, cache.MetricFamilyCache{
					Service: ironicBenchmarkService,
					MF:      mf,
				})
			}
			cache.GetCache().SetCloudCache(cloud, cloudCache)

			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			request := httptest.NewRequest(http.MethodGet, "/metrics", nil)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				response := httptest.NewRecorder()
				if err := cache.WriteCacheToResponse(response, request, cloud, []string{ironicBenchmarkService}, logger); err != nil {
					b.Fatalf("cache write failed: %v", err)
				}
				if response.Code != http.StatusOK {
					b.Fatalf("cache write returned status %d", response.Code)
				}
			}
		})
	}
}

func newIronicBenchmarkFixture(b *testing.B, pageSize int) *ironicBenchmarkFixture {
	b.Helper()

	nodes := makeBenchmarkIronicNodes(ironicBenchmarkNodeCount)
	pageDelay := ironicBenchmarkPageDelay(b)
	fixture := &ironicBenchmarkFixture{}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fixture.requestCount++
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/", "/v1", "/v1/":
			writeIronicBenchmarkVersions(b, w)
		case "/v1/nodes/detail":
			if pageDelay > 0 {
				time.Sleep(pageDelay)
			}
			writeIronicBenchmarkNodePage(b, w, r, nodes, pageSize)
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
		ResourceBase: fixture.server.URL + "/v1/",
		Type:         "baremetal",
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	exporter, err := exporters.NewIronicExporter(&exporters.ExporterConfig{
		ClientV2:    client,
		ServiceName: ironicBenchmarkService,
		Prefix:      "openstack",
		CollectTime: true,
	}, logger)
	if err != nil {
		fixture.server.Close()
		b.Fatalf("failed to create ironic exporter: %v", err)
	}
	fixture.exporter = exporter

	verifyIronicBenchmarkFixture(b, fixture, pageSize)
	return fixture
}

func verifyIronicBenchmarkFixture(b *testing.B, fixture *ironicBenchmarkFixture, pageSize int) {
	b.Helper()

	ch := make(chan prometheus.Metric)
	done := drainMetrics(ch)
	err := exporters.ListNodes(context.Background(), &fixture.exporter.BaseOpenStackExporter, ch)
	close(ch)
	<-done
	if err != nil {
		b.Fatalf("fixture ListNodes verification failed: %v", err)
	}

	wantRequests := ironicBenchmarkNodeCount / pageSize
	if ironicBenchmarkNodeCount%pageSize != 0 {
		wantRequests++
	}
	if fixture.requestCount < wantRequests {
		b.Fatalf("fixture made %d node page requests, want at least %d", fixture.requestCount, wantRequests)
	}
	fixture.requestCount = 0
}

func drainMetrics(ch <-chan prometheus.Metric) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		for range ch {
		}
		close(done)
	}()
	return done
}

func makeBenchmarkIronicNodes(count int) []benchmarkIronicNode {
	provisionStates := []string{"available", "active", "manageable", "error"}
	powerStates := []string{"power off", "power on"}
	nodes := make([]benchmarkIronicNode, count)

	for i := range count {
		maintenance := i%17 == 0
		maintenanceReason := ""
		if maintenance {
			maintenanceReason = "scheduled firmware update"
		}
		retired := i%29 == 0
		retiredReason := ""
		if retired {
			retiredReason = "capacity refresh"
		}

		nodes[i] = benchmarkIronicNode{
			UUID:                fmt.Sprintf("00000000-0000-4000-8000-%012d", i),
			Name:                fmt.Sprintf("rack-%02d-node-%04d", i/40, i),
			ProvisionState:      provisionStates[i%len(provisionStates)],
			PowerState:          powerStates[i%len(powerStates)],
			Maintenance:         maintenance,
			MaintenanceReason:   maintenanceReason,
			ConsoleEnabled:      i%3 != 0,
			ResourceClass:       "baremetal",
			DriverInfo:          map[string]any{"deploy_kernel": "7ff5ef56-daaa-4256-9dd8-c3f1f9964ebc", "deploy_ramdisk": "e9c96d45-a4c8-4165-8753-9d8f32779e99", "ipmi_address": fmt.Sprintf("10.10.%d.%d", i/254, i%254+1)},
			UpdatedAt:           time.Date(2026, 1, 1, 0, 0, i%60, 0, time.UTC).Format(time.RFC3339),
			ProvisionUpdatedAt:  time.Date(2026, 1, 1, 1, 0, i%60, 0, time.UTC).Format(time.RFC3339),
			Retired:             retired,
			RetiredReason:       retiredReason,
			Driver:              "fake-hardware",
			BootInterface:       "fake",
			DeployInterface:     "fake",
			ManagementInterface: "fake",
			PowerInterface:      "fake",
			NetworkInterface:    "noop",
			Properties:          map[string]any{"cpus": 48, "memory_mb": 131072, "local_gb": 128, "cpu_arch": "x86_64"},
			Extra:               map[string]any{"rack": fmt.Sprintf("rack-%02d", i/40)},
			InstanceInfo:        map[string]any{},
			DriverInternalInfo:  map[string]any{"benchmark_node": true},
			ChassisUUID:         fmt.Sprintf("10000000-0000-4000-8000-%012d", i/40),
			Conductor:           "ironic-conductor-0",
			ConductorGroup:      "",
			Protected:           false,
			CleanStep:           map[string]any{},
			DeployStep:          map[string]any{},
			RaidConfig:          map[string]any{},
			TargetRaidConfig:    map[string]any{},
			Traits:              []string{"CUSTOM_BENCHMARK"},
			CreatedAt:           time.Date(2025, 12, 1, 0, 0, i%60, 0, time.UTC).Format(time.RFC3339),
		}
	}

	return nodes
}

func writeIronicBenchmarkVersions(b *testing.B, w http.ResponseWriter) {
	b.Helper()

	_, err := io.WriteString(w, `{
  "default_version": {
    "status": "CURRENT",
    "min_version": "1.1",
    "version": "1.90",
    "id": "v1",
    "links": [{"href": "/v1/", "rel": "self"}]
  },
  "versions": [{
    "status": "CURRENT",
    "min_version": "1.1",
    "version": "1.90",
    "id": "v1",
    "links": [{"href": "/v1/", "rel": "self"}]
  }],
  "name": "OpenStack Ironic API"
}`)
	if err != nil {
		b.Fatalf("failed to write version response: %v", err)
	}
}

func writeIronicBenchmarkNodePage(b *testing.B, w http.ResponseWriter, r *http.Request, nodes []benchmarkIronicNode, pageSize int) {
	b.Helper()

	limit := pageSize
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		requestedLimit, err := strconv.Atoi(rawLimit)
		if err != nil {
			http.Error(w, "invalid limit", http.StatusBadRequest)
			return
		}
		if requestedLimit > 0 && requestedLimit < limit {
			limit = requestedLimit
		}
	}

	start := 0
	if marker := r.URL.Query().Get("marker"); marker != "" {
		start = -1
		for i, node := range nodes {
			if node.UUID == marker {
				start = i + 1
				break
			}
		}
		if start == -1 {
			http.Error(w, "unknown marker", http.StatusBadRequest)
			return
		}
	}

	end := start + limit
	if end > len(nodes) {
		end = len(nodes)
	}

	response := struct {
		Nodes []benchmarkIronicNode `json:"nodes"`
		Links []map[string]string   `json:"nodes_links,omitempty"`
	}{
		Nodes: nodes[start:end],
	}
	if end < len(nodes) {
		nextURL := fmt.Sprintf("%s://%s/v1/nodes/detail?marker=%s", scheme(r), r.Host, nodes[end-1].UUID)
		response.Links = []map[string]string{{"rel": "next", "href": nextURL}}
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		b.Fatalf("failed to encode node page: %v", err)
	}
}

func scheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

func ironicBenchmarkPageDelay(b *testing.B) time.Duration {
	b.Helper()

	rawDelay := os.Getenv("IRONIC_BENCH_PAGE_DELAY_MS")
	if rawDelay == "" {
		return 0
	}
	delayMS, err := strconv.Atoi(rawDelay)
	if err != nil {
		b.Fatalf("invalid IRONIC_BENCH_PAGE_DELAY_MS %q: %v", rawDelay, err)
	}
	if delayMS <= 0 {
		return 0
	}
	return time.Duration(delayMS) * time.Millisecond
}
