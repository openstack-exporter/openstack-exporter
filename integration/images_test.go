package integration

import (
	"log"
	"testing"

	"github.com/openstack-exporter/openstack-exporter/integration/clients"
)

func TestImagesIntegration(t *testing.T) {
	clients.RequireLong(t)

	// Helper to print body on failure
	failWithBody := func(t *testing.T, body string, msg string, args ...interface{}) {
		t.Helper()
		log.Printf("Metrics body:\n%s\n", body)
		t.Fatalf(msg, args...)
	}

	// Start exporter
	_, cleanup, err := startOpenStackExporter([]string{"image"})
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

	t.Run("openstack_glance_up_metric", func(t *testing.T) {
		sample, ok := findMetric(metricFamilies, "openstack_glance_up", nil)
		if !ok {
			failWithBody(t, body,
				"Metric %q not found in metrics response",
				"openstack_glance_up",
			)
		}
		if sample.value != 1 {
			failWithBody(t, body,
				"openstack_glance_up metric should have value 1 indicating service is up, got %v",
				sample.value,
			)
		}
	})

	t.Run("openstack_glance_core_metrics_present", func(t *testing.T) {
		expected := []string{
			"openstack_glance_images",
			"openstack_glance_image_bytes",
			"openstack_glance_image_created_at",
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
				"Expected Glance core metrics not found; Glance may not be fully available",
			)
		}
	})

	t.Run("glance_image_bytes_labels_present", func(t *testing.T) {
		for _, sample := range metricFamilies["openstack_glance_image_bytes"] {
			if sample.labels["id"] != "" &&
				sample.labels["name"] != "" &&
				sample.labels["tenant_id"] != "" {
				return
			}
		}
		failWithBody(t, body,
			"No 'openstack_glance_image_bytes' metric contained required labels (id,name,tenant_id)",
		)
	})

	t.Run("glance_image_created_at_labels_present", func(t *testing.T) {
		for _, sample := range metricFamilies["openstack_glance_image_created_at"] {
			if sample.labels["hidden"] != "" &&
				sample.labels["id"] != "" &&
				sample.labels["name"] != "" &&
				sample.labels["status"] != "" &&
				sample.labels["tenant_id"] != "" &&
				sample.labels["visibility"] != "" {
				return
			}
		}
		failWithBody(t, body,
			"No 'openstack_glance_image_created_at' metric contained required labels (hidden,id,name,status,tenant_id,visibility)",
		)
	})
}
