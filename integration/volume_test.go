package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"
	"github.com/openstack-exporter/openstack-exporter/integration/clients"
	"github.com/openstack-exporter/openstack-exporter/integration/tools"
)

func TestVolumeImageMetadataIntegration(t *testing.T) {
	clients.RequireLong(t)

	choices, err := clients.AcceptanceTestChoicesFromEnv()
	if err != nil {
		t.Fatal(err)
	}

	client, err := clients.NewBlockStorageV3Client()
	if err != nil {
		t.Fatalf("Failed to build block storage client: %v", err)
	}

	volume, err := volumes.Create(t.Context(), client, volumes.CreateOpts{
		Name:    tools.RandomString("ACPTTEST", 16),
		Size:    1,
		ImageID: choices.ImageID,
	}, nil).Extract()
	if err != nil {
		t.Fatalf("Could not create test volume: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		if err := volumes.Delete(ctx, client, volume.ID, nil).ExtractErr(); err != nil {
			t.Errorf("Unable to delete test volume %s: %v", volume.ID, err)
		}
	})

	err = tools.WaitForTimeout(func(ctx context.Context) (bool, error) {
		current, err := volumes.Get(ctx, client, volume.ID).Extract()
		if err != nil {
			return false, err
		}
		if current.Status == "error" {
			return false, fmt.Errorf("test volume entered error state")
		}
		return current.Status == "available", nil
	}, 120*time.Second)
	if err != nil {
		t.Fatalf("Waiting for test volume %s failed: %v", volume.ID, err)
	}

	cleanup := startExporter(t, "volume")
	defer cleanup()
	metrics := scrapeLoggedMetrics(t, "")

	metrics.requireLabelValue(t, "openstack_cinder_volume_status", labels{"id": volume.ID},
		"volume_image_id", choices.ImageID)
	metrics.requireLabels(t, "openstack_cinder_volume_status", labels{"id": volume.ID},
		"volume_image_name")
}
