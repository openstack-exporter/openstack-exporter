package integration

import (
	"strings"
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

	// Start the OpenStack exporter
	_, cleanup, err := startOpenStackExporter([]string{
		"baremetal",
	})
	if err != nil {
		t.Fatalf("Failed to start OpenStack exporter: %v", err)
	}
	defer cleanup()

	const maxTriesFetch = 10
	resp, body, err := httpGetRetry(defaultMetricsURL, maxTriesFetch, t)
	if err != nil {
		t.Fatalf("Failed to fetch metrics after multiple retries: %v", err)
	}

	bodyString := string(body)

	// Helper to always dump the full body on failures in subtests
	logOnFailure := func(t *testing.T) {
		t.Helper()
		statusCode := 0
		if resp != nil {
			statusCode = resp.StatusCode
		}
		t.Logf(
			"\nStatus Code: %d\nMetrics Endpoint: %s\nResponse Body:\n%s\n",
			statusCode,
			defaultMetricsURL,
			bodyString,
		)
	}

	// Test for openstack_ironic_node metric
	t.Run("openstack_ironic_node_metric", func(t *testing.T) {
		expectedMetric := "openstack_ironic_node"
		if !strings.Contains(bodyString, expectedMetric) {
			logOnFailure(t)
			t.Fatalf("Metric %q not found in metrics response", expectedMetric)
		}

		// Validate that the metric has expected labels
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
			if !strings.Contains(bodyString, label+"=") {
				logOnFailure(t)
				t.Errorf(
					"Expected label %q not found in openstack_ironic_node metric",
					label,
				)
			}
		}

		// Validate that the metric line contains the expected structure
		// Should have format: openstack_ironic_node{...labels...} 1
		if !strings.Contains(bodyString, "openstack_ironic_node{") {
			logOnFailure(t)
			t.Error(
				"openstack_ironic_node metric does not contain expected label structure",
			)
		}

		// Check that provision_state="active" is present (from the logs)
		if !strings.Contains(bodyString, `provision_state="active"`) {
			// This is informational; still helpful to see the full body.
			logOnFailure(t)
			t.Log(
				`Note: provision_state="active" not found, node might not be in active state yet`,
			)
		}

		// Check that console_enabled, maintenance, and retired are boolean strings
		if strings.Contains(bodyString, "openstack_ironic_node") {
			if !strings.Contains(bodyString, `console_enabled="false"`) &&
				!strings.Contains(bodyString, `console_enabled="true"`) {
				logOnFailure(t)
				t.Error(`console_enabled should be either "true" or "false"`)
			}
			if !strings.Contains(bodyString, `maintenance="false"`) &&
				!strings.Contains(bodyString, `maintenance="true"`) {
				logOnFailure(t)
				t.Error(`maintenance should be either "true" or "false"`)
			}
			if !strings.Contains(bodyString, `retired="false"`) &&
				!strings.Contains(bodyString, `retired="true"`) {
				logOnFailure(t)
				t.Error(`retired should be either "true" or "false"`)
			}
		}
	})

	// Test for openstack_ironic_up metric
	t.Run("openstack_ironic_up_metric", func(t *testing.T) {
		expectedMetric := "openstack_ironic_up"
		if !strings.Contains(bodyString, expectedMetric) {
			logOnFailure(t)
			t.Fatalf("Metric %q not found in metrics response", expectedMetric)
		}

		// Check that the up metric shows service is up (value should be 1)
		if !strings.Contains(bodyString, "openstack_ironic_up 1") {
			logOnFailure(t)
			t.Error(
				"openstack_ironic_up metric should have value 1 indicating service is up",
			)
		}
	})

	// Test for metric help and type comments
	t.Run("metric_metadata", func(t *testing.T) {
		// Check for HELP comments
		if !strings.Contains(bodyString, "# HELP openstack_ironic_node node") {
			logOnFailure(t)
			t.Error("Missing HELP comment for openstack_ironic_node metric")
		}
		if !strings.Contains(bodyString, "# HELP openstack_ironic_up up") {
			logOnFailure(t)
			t.Error("Missing HELP comment for openstack_ironic_up metric")
		}

		// Check for TYPE comments
		if !strings.Contains(bodyString, "# TYPE openstack_ironic_node gauge") {
			logOnFailure(t)
			t.Error("Missing TYPE comment for openstack_ironic_node metric")
		}
		if !strings.Contains(bodyString, "# TYPE openstack_ironic_up gauge") {
			logOnFailure(t)
			t.Error("Missing TYPE comment for openstack_ironic_up metric")
		}
	})
}
