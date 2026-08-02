package integration

import (
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

	cleanup := startExporter(t, "baremetal")
	defer cleanup()
	metrics := scrapeLoggedMetrics(t, "")

	t.Run("openstack_ironic_node_metric", func(t *testing.T) {
		metrics.requirePresentLabels(t, "openstack_ironic_node", labels{"id": node.UUID},
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
		)
	})

	t.Run("openstack_ironic_node_boolean_labels", func(t *testing.T) {
		sample := metrics.requireMetric(t, "openstack_ironic_node", labels{"id": node.UUID})
		for _, label := range []string{"console_enabled", "maintenance", "retired"} {
			switch sample.labels[label] {
			case "true", "false":
			default:
				failMetrics(t, metrics.body,
					"Expected label %q to be either true or false, got %q",
					label,
					sample.labels[label],
				)
			}
		}
	})

	t.Run("openstack_ironic_node_provision_state", func(t *testing.T) {
		metrics.requireMetric(t, "openstack_ironic_node", labels{
			"id":              node.UUID,
			"provision_state": "active",
		})
	})

	t.Run("openstack_ironic_up_metric", func(t *testing.T) {
		metrics.requireUp(t, "openstack_ironic_up")
	})

	t.Run("openstack_ironic_core_metrics_present", func(t *testing.T) {
		for _, metric := range []string{
			"openstack_ironic_node",
			"openstack_ironic_up",
		} {
			metrics.requireAnyFamily(t, metric)
		}
	})
}
