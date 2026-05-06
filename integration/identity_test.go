package integration

import (
	"log"
	"testing"

	"github.com/openstack-exporter/openstack-exporter/integration/clients"
)

func TestIdentityIntegration(t *testing.T) {
	clients.RequireLong(t)

	// Helper to print body on failure
	failWithBody := func(t *testing.T, body string, msg string, args ...interface{}) {
		t.Helper()
		log.Printf("Metrics body:\n%s\n", body)
		t.Fatalf(msg, args...)
	}

	// Start exporter
	_, cleanup, err := startOpenStackExporter([]string{"identity"})
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

	t.Run("openstack_identity_up_metric", func(t *testing.T) {
		sample, ok := findMetric(metricFamilies, "openstack_identity_up", nil)
		if !ok {
			failWithBody(t, body,
				"Metric %q not found in metrics response",
				"openstack_identity_up",
			)
		}
		if sample.value != 1 {
			failWithBody(t, body,
				"openstack_identity_up metric should have value 1 indicating service is up, got %v",
				sample.value,
			)
		}
	})

	t.Run("openstack_identity_core_metrics_present", func(t *testing.T) {
		expected := []string{
			"openstack_identity_projects",
			"openstack_identity_users",
			"openstack_identity_domains",
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
				"Expected Identity core metrics not found; Keystone may not be fully available",
			)
		}
	})

	t.Run("identity_project_info_labels_present", func(t *testing.T) {
		for _, sample := range metricFamilies["openstack_identity_project_info"] {
			if sample.labels["id"] != "" &&
				sample.labels["name"] != "" &&
				sample.labels["domain_id"] != "" &&
				sample.labels["enabled"] != "" {
				if _, ok := sample.labels["parent_id"]; ok {
					return
				}
			}
		}
		failWithBody(t, body,
			"No 'openstack_identity_project_info' metric contained required labels (id,name,domain_id,enabled,parent_id)",
		)
	})
}
