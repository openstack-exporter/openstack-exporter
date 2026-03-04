package integration

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/openstack-exporter/openstack-exporter/exporters"
	"github.com/openstack-exporter/openstack-exporter/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promslog"
)

var DEFAULT_OS_CLIENT_CONFIG = "/etc/openstack/clouds.yaml"

const defaultMetricsURL = "http://localhost:9180/metrics"

// httpGetRetry fetches url, retrying up to max times (with 1s sleep between
// attempts), and returns the response and body on the first success.
// The sleep between retries avoids a tight spin but still terminates quickly.
func httpGetRetry(url string, max int, t interface {
	Helper()
	Logf(string, ...interface{})
}) (*http.Response, []byte, error) {
	var (
		resp *http.Response
		err  error
	)
	for i := 0; i < max; i++ {
		resp, err = http.Get(url) //nolint:noctx
		if err == nil && resp.StatusCode == http.StatusOK {
			body, readErr := io.ReadAll(resp.Body)
			resp.Body.Close()
			if readErr == nil {
				return resp, body, nil
			}
			t.Helper()
			t.Logf("Attempt %d: failed to read response body: %v", i+1, readErr)
		} else {
			statusCode := 0
			if resp != nil {
				statusCode = resp.StatusCode
				resp.Body.Close()
			}
			t.Helper()
			t.Logf("Attempt %d: failed to get metrics, status=%d, err=%v", i+1, statusCode, err)
		}
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get metrics after %d retries: %w", max, err)
	}
	return nil, nil, fmt.Errorf("failed to get metrics after %d retries", max)
}

// newEmptyNovaMetadataMapping returns a non-nil LabelMappingFlag equivalent
// to having no extra metadata labels configured.
func newEmptyNovaMetadataMapping() *utils.LabelMappingFlag {
	return &utils.LabelMappingFlag{
		Labels: []string{},
		Keys:   []string{},
	}
}

// startOpenStackExporter starts an instance of the OpenStack exporter for
// testing purposes. It returns a cleanup function that should be called
// after the test is complete to shut down the exporter.
//
// enabledServices controls which OpenStack services' exporters are started.
// For example: []string{"baremetal"} in the baremetal integration test.
func startOpenStackExporter(enabledServices []string) (string, func(), error) {
	metricsPath := "/metrics"
	listenAddress := ":9180"
	prefix := "openstack"
	endpointType := "public"
	collectTime := false
	disabledMetrics := []string{}
	disableSlowMetrics := false
	disableDeprecatedMetrics := false
	disableCinderAgentUUID := false
	cloud := "devstack-system-admin" // Must exist in CI clouds.yaml
	domainID := ""
	tenantID := ""

	// Logger similar to main.go
	promlogConfig := &promslog.Config{}
	logger := promslog.New(promlogConfig)

	// Use an empty, but non-nil nova metadata mapping so Nova exporter
	// can safely dereference NovaMetadataMapping.
	novaMetadataMapping := newEmptyNovaMetadataMapping()

	// Context to control exporter lifecycle
	ctx, cancel := context.WithCancel(context.Background())
	_ = ctx // currently unused but kept for potential future use

	registry := prometheus.NewPedanticRegistry()

	enabledExporters := 0
	for _, service := range enabledServices {
		exp, err := exporters.EnableExporter(
			service,
			prefix,
			cloud,
			disabledMetrics,
			endpointType,
			collectTime,
			disableSlowMetrics,
			disableDeprecatedMetrics,
			disableCinderAgentUUID,
			domainID,
			tenantID,
			novaMetadataMapping, // non-nil here
			nil,
			logger,
		)
		if err != nil {
			slog.Error(
				"enabling exporter for service failed",
				"service", service,
				"error", err,
			)
			continue
		}
		if exp == nil {
			slog.Error(
				"got nil exporter instance",
				"service", service,
			)
			continue
		}

		registry.MustRegister(*exp)
		slog.Info(
			"Enabled exporter for service",
			"service", service,
		)
		enabledExporters++
	}

	if enabledExporters == 0 {
		cancel()
		return "", nil, fmt.Errorf("no exporter has been enabled")
	}

	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})

	// Dedicated HTTP server with explicit handler
	server := &http.Server{
		Addr:    listenAddress,
		Handler: handler,
	}

	go func() {
		slog.Info(
			"Starting OpenStack exporter",
			"address", listenAddress,
		)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error(
				"HTTP server failed",
				"error", err,
			)
		}
	}()

	cleanup := func() {
		slog.Info("Shutting down OpenStack exporter")
		cancel()
		ctxShutdown, cancelShutdown := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cancelShutdown()
		if err := server.Shutdown(ctxShutdown); err != nil {
			slog.Error(
				"HTTP server shutdown failed",
				"error", err,
			)
		}
	}

	// Wait until the server is actually up
	const maxTries = 10
	for i := 0; i < maxTries; i++ {
		resp, err := http.Get("http://localhost" + listenAddress + metricsPath)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return listenAddress, cleanup, nil
			}
		}
		time.Sleep(1 * time.Second)
	}

	cleanup()
	return "", nil, fmt.Errorf("failed to start OpenStack exporter in time")
}
