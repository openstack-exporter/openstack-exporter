package integration

import (
	"testing"

	"github.com/openstack-exporter/openstack-exporter/integration/clients"
	"github.com/openstack-exporter/openstack-exporter/integration/funcs"
)

func TestComputeIntegration(t *testing.T) {
	clients.RequireLong(t)

	computeClient, err := clients.NewComputeV2Client()
	if err != nil {
		t.Fatalf("Failed to build compute client: %v", err)
	}

	server, err := funcs.CreateServer(t, computeClient)
	if err != nil {
		t.Fatalf("Could not create test server: %v", err)
	}
	defer funcs.DeleteServer(t, computeClient, server)

	cleanup := startExporter(t, "compute")
	defer cleanup()
	metrics := scrapeLoggedMetrics(t, "")

	t.Run("nova_server_status_metric_present", func(t *testing.T) {
		metrics.requireMetric(t, "openstack_nova_server_status", labels{"id": server.ID})
	})

	t.Run("nova_total_vms_incremented", func(t *testing.T) {
		metrics.requireMinValue(t, "openstack_nova_total_vms", nil, 1)
	})

	t.Run("nova_server_local_gb_metric_present", func(t *testing.T) {
		metrics.requireMetric(t, "openstack_nova_server_local_gb", labels{"id": server.ID})
	})

	t.Run("nova_server_az_label_present", func(t *testing.T) {
		metrics.requireMetric(t, "openstack_nova_server_status", labels{
			"id":                server.ID,
			"availability_zone": server.AvailabilityZone,
		})
	})

	t.Run("nova_server_status_labels_present", func(t *testing.T) {
		metrics.requireLabels(t, "openstack_nova_server_status", labels{
			"id": server.ID, "name": server.Name, "status": "ACTIVE", "uuid": server.ID,
		}, "flavor_id", "tenant_id", "user_id", "host_id", "hypervisor_hostname")
	})

	t.Run("nova_quota_instances_admin_in_use_present", func(t *testing.T) {
		metrics.requireMinValueWithLabels(t, "openstack_nova_quota_instances", labels{
			"tenant": "admin",
			"type":   "in_use",
		}, 1, "tenant_id")
	})

	t.Run("nova_quota_key_pairs_admin_limit_present", func(t *testing.T) {
		metrics.requireMinValueWithLabels(t, "openstack_nova_quota_key_pairs", labels{
			"tenant": "admin",
			"type":   "limit",
		}, 1, "tenant_id")
	})

	t.Run("nova_quota_admin_usage_metrics_present", func(t *testing.T) {
		for _, tc := range []struct {
			name      string
			metric    string
			quotaType string
			minValue  float64
		}{
			{name: "cores_in_use", metric: "openstack_nova_quota_cores", quotaType: "in_use", minValue: 1},
			{name: "ram_in_use", metric: "openstack_nova_quota_ram", quotaType: "in_use", minValue: 1},
			{name: "instances_limit", metric: "openstack_nova_quota_instances", quotaType: "limit", minValue: 1},
		} {
			t.Run(tc.name, func(t *testing.T) {
				metrics.requireMinValueWithLabels(t, tc.metric, labels{
					"tenant": "admin",
					"type":   tc.quotaType,
				}, tc.minValue, "tenant_id")
			})
		}
	})
}
