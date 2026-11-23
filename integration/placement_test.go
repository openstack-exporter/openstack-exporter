package integration

import (
	"testing"

	"github.com/openstack-exporter/openstack-exporter/integration/clients"
)

func TestPlacementIntegration(t *testing.T) {
	clients.RequireLong(t)

	cleanup := startExporter(t, "placement")
	defer cleanup()
	metrics := scrapeLoggedMetrics(t, "")

	t.Run("openstack_placement_up_metric", func(t *testing.T) {
		metrics.requireUp(t, "openstack_placement_up")
	})

	t.Run("openstack_placement_core_metrics_present", func(t *testing.T) {
		metrics.requireAnyFamily(t,
			"openstack_placement_resource_total",
			"openstack_placement_resource_usage",
			"openstack_placement_resource_provider_allocations",
			"openstack_placement_resource_provider_allocations",
			"openstack_placement_resource_generation",
			"openstack_placement_resource_allocation_ratio",
		)
	})

	t.Run("openstack_trait_label_present", func(t *testing.T) {
		metrics.requireAnyFamily(t,
			"openstack_placement_resource_provider_trait",
		)
	})

	t.Run("openstack_trait_label_value", func(t *testing.T) {
		metrics.requireMetric(t,
			"openstack_placement_resource_provider_trait",
			labels{"trait": "CUSTOM_MY_TRAIT"},
		)
	})
}
