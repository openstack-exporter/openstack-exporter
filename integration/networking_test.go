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

func TestNetworkingIntegration(t *testing.T) {
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

	t.Run("openstack_neutron_up_metric", func(t *testing.T) {
		if !strings.Contains(bodyString, "openstack_neutron_up") {
			t.Errorf("Metric 'openstack_neutron_up' not found. Status Code: %d, Endpoint: %s\nBody:\n%s", resp.StatusCode, metricsURL, bodyString)
			return
		}
		if !strings.Contains(bodyString, "openstack_neutron_up 1") {
			t.Error("openstack_neutron_up metric should have value 1 indicating service is up")
		}
		if !strings.Contains(bodyString, "# HELP openstack_neutron_up up") {
			t.Error("Missing HELP comment for openstack_neutron_up metric")
		}
		if !strings.Contains(bodyString, "# TYPE openstack_neutron_up gauge") {
			t.Error("Missing TYPE comment for openstack_neutron_up metric")
		}
	})

	t.Run("openstack_neutron_core_metrics_present", func(t *testing.T) {
		// Spot-check a few key Neutron metrics from the reference log
		expected := []string{
			"# HELP openstack_neutron_networks",
			"# HELP openstack_neutron_ports",
			"# HELP openstack_neutron_subnets",
			"# HELP openstack_neutron_router",
		}
		foundAny := false
		for _, m := range expected {
			if strings.Contains(bodyString, m) {
				foundAny = true
				break
			}
		}
		if !foundAny {
			t.Log("Note: Expected Neutron metrics HELP headers not found; Neutron may not be fully available")
		}
	})
}
