package integration

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/go-kit/log/level"
	"github.com/openstack-exporter/openstack-exporter/exporters"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
)

// startOpenStackExporter starts an instance of the OpenStack exporter for
// testing purposes. It returns a cleanup function that should be called
// after the test is complete to shut down the exporter.
func startOpenStackExporter() (string, func(), error) {
	// Define flags (copied from main.go)
	metricsPath := "/metrics"
	listenAddress := ":9180"
	prefix := "openstack"
	endpointType := "public"
	collectTime := false
	disabledMetrics := []string{}
	disableSlowMetrics := false
	disableDeprecatedMetrics := false
	disableCinderAgentUUID := false
	cloud := "devstack" // Or any suitable default for testing
	domainID := ""
	tenantID := ""

	// Create a logger for the test
	promlogConfig := &promlog.Config{}
	logger := promlog.New(promlogConfig)

	// Create a context to control the exporter lifecycle
	_, cancel := context.WithCancel(context.Background())

	// Create a registry and handler
	registry := prometheus.NewPedanticRegistry()

	// Define services to enable. For simplicity, we'll enable a minimal
	// set. Adjust as needed for your tests.
	enabledServices := []string{"baremetal"}

	// Enable exporters
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
			nil,
			logger,
		)
		if err != nil {
			level.Error(logger).Log(
				"err",
				"enabling exporter for service failed",
				"service",
				service,
				"error",
				err,
			)
			continue
		}
		registry.MustRegister(*exp)
		level.Info(logger).Log("msg", "Enabled exporter for service", "service", service)
		enabledExporters++
	}

	if enabledExporters == 0 {
		cancel()
		return "", nil, fmt.Errorf("no exporter has been enabled")
	}

	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})

	// Start the HTTP server in a goroutine
	server := &http.Server{Addr: listenAddress}
	http.Handle(metricsPath, handler)

	go func() {
		level.Info(logger).Log("msg", "Starting OpenStack exporter", "address", listenAddress)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			level.Error(logger).Log("err", "HTTP server failed", "error", err)
		}
	}()

	// Define the cleanup function
	cleanup := func() {
		level.Info(logger).Log("msg", "Shutting down OpenStack exporter")
		cancel() // Cancel the context
		ctxShutdown, cancelShutdown := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cancelShutdown()
		if err := server.Shutdown(ctxShutdown); err != nil {
			level.Error(logger).Log("err", "HTTP server shutdown failed", "error", err)
		}
	}

	// Wait for the server to start. A simple check is to see if we can GET
	// the metrics endpoint.
	const maxTries = 10
	for i := 0; i < maxTries; i++ {
		resp, err := http.Get("http://localhost" + listenAddress + metricsPath)
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			return listenAddress, cleanup, nil // Success!
		}
		time.Sleep(1 * time.Second)
	}

	// If we get here, the server didn't start in time. Clean up and return an
	// error.
	cleanup()
	return "", nil, fmt.Errorf("failed to start OpenStack exporter in time")
}

func TestIntegration(t *testing.T) {
	// Start the OpenStack exporter
	listenAddress, cleanup, err := startOpenStackExporter()
	if err != nil {
		t.Fatalf("Failed to start OpenStack exporter: %v", err)
	}
	defer cleanup()

	// Construct the metrics URL
	metricsURL := "http://localhost" + listenAddress + "/metrics"

	// Helper function to fetch metrics with retries
	fetchMetrics := func(
		url string,
		maxTries int,
	) (resp *http.Response, body []byte, err error) {
		for i := 0; i < maxTries; i++ {
			resp, err = http.Get(url)
			if err == nil && resp.StatusCode == http.StatusOK {
				defer resp.Body.Close()
				body, err = io.ReadAll(resp.Body)
				if err == nil {
					return resp, body, nil // Success!
				}
				t.Logf("Attempt %d: Failed to read response body: %v", i+1, err)
			} else {
				var statusCode int
				if resp != nil {
					statusCode = resp.StatusCode
				}
				t.Logf(
					"Attempt %d: Failed to get metrics, status code: %d, error: %v",
					i+1,
					statusCode,
					err,
				)
			}
			if resp != nil && resp.Body != nil {
				resp.Body.Close() // Close the body on each retry
			}
			time.Sleep(1 * time.Second)
		}
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get metrics after %d retries: %w", maxTries, err)
		}

		return nil, nil, fmt.Errorf(
			"failed to get metrics after %d retries, "+
				"but the error is nil (this should not happen)",
			maxTries,
		)
	}

	// Fetch the metrics
	const maxTriesFetch = 10
	resp, body, err := fetchMetrics(metricsURL, maxTriesFetch)
	if err != nil {
		t.Fatalf("Failed to fetch metrics after multiple retries: %v", err)
	}

	// Convert the response body to a string for easier handling
	bodyString := string(body)

	// Check for the expected metric and provide a clearer error message
	expectedMetric := "openstack_ironic_up"
	if !strings.Contains(bodyString, expectedMetric) {
		t.Errorf(
			"Metric '%s' not found in metrics response.\n\n"+
				"Status Code: %d\n\n"+
				"Metrics Endpoint: %s\n\n"+
				"Response Body:\n%s\n",
			expectedMetric,
			resp.StatusCode,
			metricsURL,
			bodyString,
		)
	}
}
