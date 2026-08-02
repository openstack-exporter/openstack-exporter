package integration

import (
	"testing"

	"github.com/openstack-exporter/openstack-exporter/integration/clients"
)

func TestVolumeImageMetadataIntegration(t *testing.T) {
	clients.RequireLong(t)

	cleanup := startExporter(t, "volume")
	defer cleanup()
	metrics := scrapeLoggedMetrics(t, "")

	metrics.requirePresentLabels(t, "openstack_cinder_volume_status", nil,
		"volume_image_id", "volume_image_name")
}
