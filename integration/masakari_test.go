package integration

import (
	"log"
	"testing"

	"github.com/openstack-exporter/openstack-exporter/integration/clients"
)

func TestMasakariIntegration(t *testing.T) {
	clients.RequireLong(t)

	failWithBody := func(t *testing.T, body string, msg string, args ...interface{}) {
		t.Helper()
		log.Printf("Metrics body:\n%s\n", body)
		t.Fatalf(msg, args...)
	}

	_, cleanup, err := startOpenStackExporter([]string{"instance-ha"})
	if err != nil {
		t.Skipf("Masakari exporter is not available yet: %v", err)
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

	t.Run("openstack_masakari_up_metric", func(t *testing.T) {
		sample, ok := findMetric(metricFamilies, "openstack_masakari_up", nil)
		if !ok {
			failWithBody(t, body, "Metric %q not found in metrics response", "openstack_masakari_up")
		}
		if sample.value != 1 {
			failWithBody(t, body, "openstack_masakari_up metric should have value 1 indicating service is up, got %v", sample.value)
		}
	})

	t.Run("openstack_masakari_core_metrics_present", func(t *testing.T) {
		for _, metric := range []string{
			"openstack_masakari_segment",
			"openstack_masakari_host",
			"openstack_masakari_host_on_maintenance",
			"openstack_masakari_host_reserved",
		} {
			if _, ok := metricFamilies[metric]; !ok {
				failWithBody(t, body, "Expected Masakari metric %q not found", metric)
			}
		}
	})

	t.Run("masakari_host_labels_present", func(t *testing.T) {
		for _, sample := range metricFamilies["openstack_masakari_host"] {
			if sample.labels["id"] != "" &&
				sample.labels["uuid"] != "" &&
				sample.labels["hostname"] != "" &&
				sample.labels["failover_segment_id"] != "" &&
				sample.labels["failover_segment_name"] != "" &&
				sample.labels["type"] != "" &&
				sample.labels["control_attributes"] != "" {
				return
			}
		}
		failWithBody(t, body, "No openstack_masakari_host metric contained all expected labels")
	})
}
