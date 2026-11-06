package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/openstack-exporter/openstack-exporter/exporters"
	"github.com/prometheus/client_golang/prometheus"
	pver "github.com/prometheus/client_golang/prometheus/collectors/version"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetAutoServicesState(t *testing.T) {
	serviceStates := map[string]serviceState{
		"compute": serviceEnabled,
		"network": serviceAuto,
		"image":   serviceAuto,
	}

	setAutoServicesState(serviceStates, serviceDisabled)
	assert.Equal(t, serviceEnabled, serviceStates["compute"])
	assert.Equal(t, serviceDisabled, serviceStates["network"])
	assert.Equal(t, serviceDisabled, serviceStates["image"])
}

func TestApplyAutodetection(t *testing.T) {
	serviceStates := map[string]serviceState{
		"compute": serviceEnabled,
		"network": serviceAuto,
		"image":   serviceAuto,
		"dns":     serviceDisabled,
	}

	applyAutodetection(serviceStates, []string{"network"})
	assert.Equal(t, serviceEnabled, serviceStates["compute"])
	assert.Equal(t, serviceEnabled, serviceStates["network"])
	assert.Equal(t, serviceDisabled, serviceStates["image"])
	assert.Equal(t, serviceDisabled, serviceStates["dns"])
}

func TestGetEnabledServicesFromStates(t *testing.T) {
	serviceStates := make(map[string]serviceState, len(exporters.SupportedExporters))
	for _, service := range exporters.SupportedExporters {
		serviceStates[service] = serviceDisabled
	}
	serviceStates["compute"] = serviceEnabled
	serviceStates["network"] = serviceEnabled

	services := getEnabledServicesFromStates(serviceStates)
	assert.ElementsMatch(t, []string{"compute", "network"}, services)
}

func TestSelectServicesForRequest(t *testing.T) {
	configured := []string{"compute", "network", "image"}
	tests := []struct {
		name      string
		url       string
		expected  []string
		errSubstr string
	}{
		{
			name:     "filters with include and exclude",
			url:      "/probe?include_services=compute,image&exclude_services=image",
			expected: []string{"compute"},
		},
		{
			name:     "normalizes spaces and duplicates",
			url:      "/probe?include_services=compute,%20compute%20,network",
			expected: []string{"compute", "network"},
		},
		{
			name:      "rejects invalid include service",
			url:       "/probe?include_services=compute,bad",
			errSubstr: "invalid include_services",
		},
		{
			name:      "rejects invalid exclude service",
			url:       "/probe?exclude_services=bad",
			errSubstr: "invalid exclude_services",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.url, nil)
			services, err := selectServicesForRequest(configured, req)

			if tc.errSubstr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errSubstr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expected, services)
		})
	}
}

// stubExporter is the smallest OpenStackExporter the pool can serve.
type stubExporter struct{ name string }

func (s *stubExporter) Describe(chan<- *prometheus.Desc) {}
func (s *stubExporter) Collect(chan<- prometheus.Metric) {}
func (s *stubExporter) GetName() string                  { return s.name }
func (s *stubExporter) AddMetric(string, exporters.ListFunc, []string, string, prometheus.Labels) {
}
func (s *stubExporter) MetricIsDisabled(string) bool { return false }

func stubFactory(failing ...string) func(string, exporters.ExporterOptions, *slog.Logger) (exporters.OpenStackExporter, error) {
	return func(service string, _ exporters.ExporterOptions, _ *slog.Logger) (exporters.OpenStackExporter, error) {
		if slices.Contains(failing, service) {
			return nil, errors.New("service unavailable")
		}
		return &stubExporter{name: service}, nil
	}
}

// statesWith disables every service except the ones given with their state.
func statesWith(states map[string]serviceState) map[string]serviceState {
	all := map[string]serviceState{}
	for _, service := range exporters.SupportedExporters {
		all[service] = serviceDisabled
	}
	for service, state := range states {
		all[service] = state
	}
	return all
}

func newStubPool(failing ...string) *exporters.Pool {
	pool := exporters.NewPool(exporters.ExporterOptions{Prefix: "openstack"}, slog.New(slog.DiscardHandler))
	pool.Factory = stubFactory(failing...)
	return pool
}

func TestResolveServiceConfigLeavesStatesUntouched(t *testing.T) {
	states := statesWith(map[string]serviceState{"compute": serviceEnabled, "network": serviceAuto})
	logger := slog.New(slog.DiscardHandler)

	services, err := resolveServiceConfig(ResolveServiceConfigParams{Cloud: "papa", DisableAutodetect: true, ServiceStates: states}, logger)
	require.NoError(t, err)
	assert.Equal(t, []string{"network", "compute"}, services)
	assert.Equal(t, serviceAuto, states["network"], "auto services must stay auto for the next reload")
}

func TestReloadSingleCloudBindsConfiguredServices(t *testing.T) {
	pool := newStubPool("network")
	detection := &serviceDetection{cloud: "papa"}
	params := ResolveServiceConfigParams{
		Cloud:             "papa",
		DisableAutodetect: true,
		ServiceStates:     statesWith(map[string]serviceState{"compute": serviceEnabled, "network": serviceEnabled}),
	}

	require.NoError(t, reloadSingleCloud(context.Background(), pool, params, detection, slog.New(slog.DiscardHandler)))
	assert.Equal(t, []string{"network", "compute"}, pool.Services("papa"))
	bound := pool.Snapshot("papa", pool.Services("papa"))
	assert.Contains(t, bound.Exporters, "compute")
	assert.NotContains(t, bound.Exporters, "network")
	assert.Equal(t, 1, testutil.CollectAndCount(detection))
}

func TestReloadSingleCloudFallsBackWhenDetectionFails(t *testing.T) {
	cloudsFile := filepath.Join(t.TempDir(), "clouds.yaml")
	require.NoError(t, os.WriteFile(cloudsFile, []byte("clouds: {}\n"), 0o600))
	t.Setenv("OS_CLIENT_CONFIG_FILE", cloudsFile)
	t.Setenv("OS_CLOUD", "")

	pool := newStubPool()
	detection := &serviceDetection{cloud: "missing"}
	params := ResolveServiceConfigParams{
		Cloud:         "missing",
		ServiceStates: statesWith(map[string]serviceState{"compute": serviceEnabled, "network": serviceAuto}),
	}
	logger := slog.New(slog.DiscardHandler)

	// Nothing detected yet: only the explicitly enabled services are bound.
	err := reloadSingleCloud(context.Background(), pool, params, detection, logger)
	require.Error(t, err)
	assert.Equal(t, []string{"compute"}, pool.Services("missing"))
	assert.Contains(t, pool.Snapshot("missing", []string{"compute"}).Exporters, "compute")

	expected := "# HELP openstack_exporter_service_autodetect_success 1 when the last lookup of the service catalog succeeded, 0 when the exporter runs on the previous or the explicitly enabled service set.\n" +
		"# TYPE openstack_exporter_service_autodetect_success gauge\n" +
		"openstack_exporter_service_autodetect_success{cloud=\"missing\"} 0\n"
	require.NoError(t, testutil.CollectAndCompare(detection, strings.NewReader(expected)))

	// A previously known set survives a later detection failure.
	pool.Reload(context.Background(), "missing", []string{"compute", "network"})
	require.Error(t, reloadSingleCloud(context.Background(), pool, params, detection, logger))
	assert.Equal(t, []string{"compute", "network"}, pool.Services("missing"))
}

func TestRunReloaderRunsAtStartAndOnSIGHUP(t *testing.T) {
	calls := make(chan struct{}, 8)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		runReloader(ctx, 0, func(context.Context) error {
			calls <- struct{}{}
			return nil
		}, slog.New(slog.DiscardHandler))
	}()

	select {
	case <-calls:
	case <-time.After(5 * time.Second):
		t.Fatal("initial reload did not run")
	}

	require.NoError(t, syscall.Kill(os.Getpid(), syscall.SIGHUP))
	select {
	case <-calls:
	case <-time.After(5 * time.Second):
		t.Fatal("SIGHUP did not trigger a reload")
	}

	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("reloader did not stop on context cancellation")
	}
}

func TestServeCollectorsReportsReadinessAndBuildInfo(t *testing.T) {
	pool := newStubPool("volume")
	pool.Reload(context.Background(), "papa", []string{"compute", "volume"})
	detection := &serviceDetection{cloud: "papa"}
	detection.record(nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	serveCollectors(rec, req, pool.Collectors(req.Context(), "papa", []string{"compute", "volume"}), pver.NewCollector("openstack_exporter"), detection)

	body := rec.Body.String()
	assert.Contains(t, body, `openstack_exporter_collector_ready{cloud="papa",service="compute"} 1`)
	assert.Contains(t, body, `openstack_exporter_collector_ready{cloud="papa",service="volume"} 0`)
	assert.Contains(t, body, `openstack_exporter_service_autodetect_success{cloud="papa"} 1`)
	assert.Contains(t, body, "openstack_exporter_build_info")
}
