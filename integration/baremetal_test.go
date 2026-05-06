package integration

import (
	"log"
	"testing"

	th "github.com/gophercloud/gophercloud/v2/testhelper"
	"github.com/openstack-exporter/openstack-exporter/integration/clients"
	"github.com/openstack-exporter/openstack-exporter/integration/funcs"
)

func TestBaremetalIntegration(t *testing.T) {
	clients.RequireLong(t)

	client, err := clients.NewBareMetalV1Client()
	th.AssertNoErr(t, err)
	client.Microversion = "1.87"

	node, err := funcs.CreateFakeNode(t, client)
	th.AssertNoErr(t, err)

	_, err = funcs.DeployFakeNode(t, client, node)
	th.AssertNoErr(t, err)

	// Helper to print body on failure
	failWithBody := func(t *testing.T, body string, msg string, args ...interface{}) {
		t.Helper()
		log.Printf("Metrics body:\n%s\n", body)
		t.Fatalf(msg, args...)
	}

	// Start exporter
	_, cleanup, err := startOpenStackExporter([]string{"baremetal"})
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

	t.Run("openstack_ironic_node_metric", func(t *testing.T) {
		sample, ok := findMetric(metricFamilies, "openstack_ironic_node", map[string]string{
			"id": node.UUID,
		})
		if !ok {
			failWithBody(t, body,
				"Expected openstack_ironic_node metric for node %s not found",
				node.UUID,
			)
		}

		expectedLabels := []string{
			"console_enabled",
			"deploy_kernel",
			"deploy_ramdisk",
			"id",
			"maintenance",
			"name",
			"power_state",
			"provision_state",
			"resource_class",
			"retired",
			"retired_reason",
		}

		for _, label := range expectedLabels {
			if _, ok := sample.labels[label]; !ok {
				failWithBody(t, body,
					"Expected label %q not found in openstack_ironic_node metric for node %s",
					label,
					node.UUID,
				)
			}
		}
	})

	t.Run("openstack_ironic_node_boolean_labels", func(t *testing.T) {
		sample, ok := findMetric(metricFamilies, "openstack_ironic_node", map[string]string{
			"id": node.UUID,
		})
		if !ok {
			failWithBody(t, body,
				"Expected openstack_ironic_node metric for node %s not found",
				node.UUID,
			)
		}

		for _, label := range []string{"console_enabled", "maintenance", "retired"} {
			switch sample.labels[label] {
			case "true", "false":
			default:
				failWithBody(t, body,
					"Expected label %q to be either true or false, got %q",
					label,
					sample.labels[label],
				)
			}
		}
	})

	t.Run("openstack_ironic_node_provision_state", func(t *testing.T) {
		if _, ok := findMetric(metricFamilies, "openstack_ironic_node", map[string]string{
			"id":              node.UUID,
			"provision_state": "active",
		}); !ok {
			failWithBody(t, body,
				`Expected provision_state="active" for node %s not found`,
				node.UUID,
			)
		}
	})

	t.Run("openstack_ironic_up_metric", func(t *testing.T) {
		sample, ok := findMetric(metricFamilies, "openstack_ironic_up", nil)
		if !ok {
			failWithBody(t, body,
				"Metric %q not found in metrics response",
				"openstack_ironic_up",
			)
		}
		if sample.value != 1 {
			failWithBody(t, body,
				"openstack_ironic_up metric should have value 1 indicating service is up, got %v",
				sample.value,
			)
		}
	})

	t.Run("openstack_ironic_core_metrics_present", func(t *testing.T) {
		expected := []string{
			"openstack_ironic_node",
			"openstack_ironic_up",
		}
		for _, metric := range expected {
			if _, ok := metricFamilies[metric]; !ok {
				failWithBody(t, body,
					"Expected Ironic metric %q not found",
					metric,
				)
			}
		}
	})
}
