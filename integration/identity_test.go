package integration

import (
	"testing"

	"github.com/openstack-exporter/openstack-exporter/integration/clients"
)

func TestIdentityIntegration(t *testing.T) {
	clients.RequireLong(t)

	cleanup := startExporter(t, "identity")
	defer cleanup()
	metrics := scrapeLoggedMetrics(t, "")

	t.Run("openstack_identity_up_metric", func(t *testing.T) {
		metrics.requireUp(t, "openstack_identity_up")
	})

	t.Run("openstack_identity_core_metrics_present", func(t *testing.T) {
		metrics.requireAnyFamily(t,
			"openstack_identity_projects",
			"openstack_identity_users",
			"openstack_identity_domains",
		)
	})

	t.Run("identity_project_info_labels_present", func(t *testing.T) {
		sample := metrics.requireSampleWithLabels(t, "openstack_identity_project_info", "id", "name", "domain_id", "enabled")
		if _, ok := sample.labels["parent_id"]; !ok {
			failMetrics(t, metrics.body, "Expected openstack_identity_project_info metric to include parent_id label")
		}
	})
}
