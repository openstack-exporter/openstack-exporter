package integration

import (
	"testing"

	"github.com/openstack-exporter/openstack-exporter/integration/clients"
)

func TestImagesIntegration(t *testing.T) {
	clients.RequireLong(t)

	cleanup := startExporter(t, "image")
	defer cleanup()
	metrics := scrapeLoggedMetrics(t, "")

	t.Run("openstack_glance_up_metric", func(t *testing.T) {
		metrics.requireUp(t, "openstack_glance_up")
	})

	t.Run("openstack_glance_core_metrics_present", func(t *testing.T) {
		metrics.requireAnyFamily(t,
			"openstack_glance_images",
			"openstack_glance_image_bytes",
			"openstack_glance_image_created_at",
		)
	})

	t.Run("glance_image_bytes_labels_present", func(t *testing.T) {
		metrics.requireSampleWithLabels(t, "openstack_glance_image_bytes", "id", "name", "tenant_id")
	})

	t.Run("glance_image_created_at_labels_present", func(t *testing.T) {
		metrics.requireSampleWithLabels(t, "openstack_glance_image_created_at", "hidden", "id", "name", "status", "tenant_id", "visibility")
	})
}
