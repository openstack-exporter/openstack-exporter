package integration

import (
	"testing"

	"github.com/openstack-exporter/openstack-exporter/integration/clients"
	"github.com/openstack-exporter/openstack-exporter/integration/funcs"
)

func TestImagesIntegration(t *testing.T) {
	clients.RequireLong(t)

	imageClient, err := clients.NewImageV2Client()
	if err != nil {
		t.Fatalf("Failed to build image client: %v", err)
	}

	// A server snapshot carries the nova-set image_type property; creating
	// the image directly with the property avoids booting a server.
	snapshotImage, err := funcs.CreateImage(t, imageClient, map[string]string{"image_type": "snapshot"})
	if err != nil {
		t.Fatalf("Could not create test snapshot image: %v", err)
	}
	defer funcs.DeleteImage(t, imageClient, snapshotImage)

	plainImage, err := funcs.CreateImage(t, imageClient, nil)
	if err != nil {
		t.Fatalf("Could not create test image: %v", err)
	}
	defer funcs.DeleteImage(t, imageClient, plainImage)

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

	t.Run("glance_image_type_label_from_property", func(t *testing.T) {
		metrics.requireLabelValue(t, "openstack_glance_image_bytes", labels{"id": snapshotImage.ID}, "image_type", "snapshot")
		metrics.requireLabelValue(t, "openstack_glance_image_created_at", labels{"id": snapshotImage.ID}, "image_type", "snapshot")
	})

	t.Run("glance_image_type_label_defaults_to_image", func(t *testing.T) {
		metrics.requireLabelValue(t, "openstack_glance_image_bytes", labels{"id": plainImage.ID}, "image_type", "image")
		metrics.requireLabelValue(t, "openstack_glance_image_created_at", labels{"id": plainImage.ID}, "image_type", "image")
	})
}
