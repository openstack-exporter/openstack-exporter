package integration

import (
	"testing"

	"github.com/openstack-exporter/openstack-exporter/integration/clients"
	"github.com/openstack-exporter/openstack-exporter/integration/funcs"
)

func TestMasakariIntegration(t *testing.T) {
	clients.RequireLong(t)

	masakariClient, err := clients.NewInstanceHAV1Client()
	if err != nil {
		t.Skipf("Masakari service is not available: %v", err)
	}

	segment, err := funcs.CreateMasakariSegment(t, masakariClient)
	if err != nil {
		t.Fatalf("Could not create test segment: %v", err)
	}
	defer funcs.DeleteMasakariSegment(t, masakariClient, segment.UUID)

	host, err := funcs.CreateMasakariHost(t, masakariClient, segment.UUID)
	if err != nil {
		t.Fatalf("Could not create test host: %v", err)
	}
	defer funcs.DeleteMasakariHost(t, masakariClient, segment.UUID, host.UUID)

	cleanup := startExporter(t, "instance-ha")
	defer cleanup()
	metrics := scrapeLoggedMetrics(t, "")

	t.Run("openstack_masakari_up_metric", func(t *testing.T) {
		metrics.requireUp(t, "openstack_masakari_up")
	})

	t.Run("openstack_masakari_core_metrics_present", func(t *testing.T) {
		metrics.requireAllFamilies(t,
			"openstack_masakari_segment",
			"openstack_masakari_host",
			"openstack_masakari_host_on_maintenance",
			"openstack_masakari_host_reserved",
		)
	})

	t.Run("masakari_host_labels_present", func(t *testing.T) {
		metrics.requireSampleWithLabels(t, "openstack_masakari_host",
			"id", "uuid", "hostname", "failover_segment_id",
			"failover_segment_name", "type", "control_attributes")
	})
}
