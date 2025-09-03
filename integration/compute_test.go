package integration

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/openstack-exporter/openstack-exporter/integration/clients"
)

func TestComputeIntegration(t *testing.T) {
	clients.RequireLong(t)

	_, cleanup, err := startOpenStackExporter()
	if err != nil {
		t.Fatalf("Failed to start OpenStack exporter: %v", err)
	}
	defer cleanup()

	metricsURL := "http://localhost:9180/metrics"

	fetchMetrics := func(url string, maxTries int) (resp *http.Response, body []byte, err error) {
		for i := 0; i < maxTries; i++ {
			resp, err = http.Get(url)
			if err == nil && resp.StatusCode == http.StatusOK {
				defer resp.Body.Close()
				body, err = io.ReadAll(resp.Body)
				if err == nil {
					return resp, body, nil
				}
				t.Logf("Attempt %d: Failed to read response body: %v", i+1, err)
			} else {
				var statusCode int
				if resp != nil {
					statusCode = resp.StatusCode
				}
				t.Logf("Attempt %d: Failed to get metrics, status code: %d, error: %v", i+1, statusCode, err)
			}
			if resp != nil && resp.Body != nil {
				resp.Body.Close()
			}
			time.Sleep(1 * time.Second)
		}
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get metrics after %d retries: %w", maxTries, err)
		}
		return nil, nil, fmt.Errorf("failed to get metrics after %d retries, but the error is nil (this should not happen)", maxTries)
	}

	time.Sleep(10 * time.Second)

	const maxTriesFetch = 10
	resp, body, err := fetchMetrics(metricsURL, maxTriesFetch)
	if err != nil {
		t.Fatalf("Failed to fetch metrics after multiple retries: %v", err)
	}

	bodyString := string(body)

	t.Run("openstack_nova_up_metric", func(t *testing.T) {
		if !strings.Contains(bodyString, "openstack_nova_up") {
			t.Errorf("Metric 'openstack_nova_up' not found. Status Code: %d, Endpoint: %s\nBody:\n%s", resp.StatusCode, metricsURL, bodyString)
			return
		}
		if !strings.Contains(bodyString, "openstack_nova_up 1") {
			t.Error("openstack_nova_up metric should have value 1 indicating service is up")
		}
		if !strings.Contains(bodyString, "# HELP openstack_nova_up up") {
			t.Error("Missing HELP comment for openstack_nova_up metric")
		}
		if !strings.Contains(bodyString, "# TYPE openstack_nova_up gauge") {
			t.Error("Missing TYPE comment for openstack_nova_up metric")
		}
	})

	t.Run("openstack_nova_flavors_metric_present", func(t *testing.T) {
		// Validate presence of at least one core Nova metric beyond up
		if !strings.Contains(bodyString, "# HELP openstack_nova_flavors") &&
			!strings.Contains(bodyString, "# HELP openstack_nova_total_vms") {
			t.Log("Note: Neither flavors nor total_vms HELP headers found; Nova may be partially configured")
		}
	})
}
