package integration

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-kit/log/level"
	"github.com/openstack-exporter/openstack-exporter/exporters"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
)

var DEFAULT_OS_CLIENT_CONFIG = "/etc/openstack/clouds.yaml"

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
	cloud := "devstack-system-admin" // Or any suitable default for testing
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
	var enabledServices = []string{"network", "compute", "image", "volume", "identity", "object-store", "load-balancer", "container-infra", "dns", "baremetal", "gnocchi", "database", "orchestration", "placement", "sharev2"}

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
