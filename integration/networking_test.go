package integration

import (
	"log"
	"testing"

	"github.com/openstack-exporter/openstack-exporter/integration/clients"
)

func TestNetworkingIntegration(t *testing.T) {
	clients.RequireLong(t)

	// Helper to print body on failure
	failWithBody := func(t *testing.T, body string, msg string, args ...interface{}) {
		t.Helper()
		log.Printf("Metrics body:\n%s\n", body)
		t.Fatalf(msg, args...)
	}

	// Start exporter
	_, cleanup, err := startOpenStackExporter([]string{"network"})
	if err != nil {
		t.Fatalf("Failed to start exporter: %v", err)
	}
	defer cleanup()

	_, bodyBytes, err := httpGetRetry(defaultMetricsURL, 10, t)
	if err != nil {
		t.Fatalf("Failed to fetch metrics: %v", err)
	}
	body := string(bodyBytes)
	t.Logf("Metrics response body:\n%s", body)

	metricFamilies, err := parseMetrics(bodyBytes)
	if err != nil {
		failWithBody(t, body, "Failed to parse metrics response: %v", err)
	}

	t.Run("openstack_neutron_up_metric", func(t *testing.T) {
		sample, ok := findMetric(metricFamilies, "openstack_neutron_up", nil)
		if !ok {
			failWithBody(t, body,
				"Metric %q not found in metrics response",
				"openstack_neutron_up",
			)
		}
		if sample.value != 1 {
			failWithBody(t, body,
				"openstack_neutron_up metric should have value 1 indicating service is up, got %v",
				sample.value,
			)
		}
	})

	t.Run("openstack_neutron_core_metrics_present", func(t *testing.T) {
		expected := []string{
			"openstack_neutron_networks",
			"openstack_neutron_ports",
			"openstack_neutron_subnets",
			"openstack_neutron_router",
		}
		foundAny := false
		for _, m := range expected {
			if _, ok := metricFamilies[m]; ok {
				foundAny = true
				break
			}
		}
		if !foundAny {
			failWithBody(t, body,
				"Expected Neutron core metrics not found; Neutron may not be fully available",
			)
		}
	})

	t.Run("neutron_network_labels_present", func(t *testing.T) {
		for _, sample := range metricFamilies["openstack_neutron_network"] {
			if sample.labels["id"] != "" &&
				sample.labels["name"] != "" &&
				sample.labels["is_external"] != "" &&
				sample.labels["is_shared"] != "" {
				if _, ok := sample.labels["provider_network_type"]; ok {
					return
				}
			}
		}
		failWithBody(t, body,
			"No 'openstack_neutron_network' metric contained required labels (id,name,is_external,is_shared,provider_network_type)",
		)
	})
}
