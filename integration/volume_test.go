package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"
	"github.com/openstack-exporter/openstack-exporter/integration/clients"
	"github.com/openstack-exporter/openstack-exporter/integration/funcs"
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

func TestVolumeIntegration(t *testing.T) {
	clients.RequireLong(t)

	blockClient, err := clients.NewBlockStorageV3Client()
	if err != nil {
		t.Fatalf("Failed to build block storage client: %v", err)
	}

	volume, err := funcs.CreateVolume(t, blockClient)
	if err != nil {
		t.Fatalf("Could not create test volume: %v", err)
	}
	defer funcs.DeleteVolume(t, blockClient, volume)

	snapshot, err := funcs.CreateSnapshot(t, blockClient, volume)
	if err != nil {
		t.Fatalf("Could not create test snapshot: %v", err)
	}
	defer funcs.DeleteSnapshot(t, blockClient, snapshot)

	backup, err := funcs.CreateBackup(t, blockClient, volume)
	if err != nil {
		t.Fatalf("Could not create test backup: %v", err)
	}
	defer funcs.DeleteBackup(t, blockClient, backup)

	cleanup := startExporter(t, "volume")
	defer cleanup()
	metrics := scrapeLoggedMetrics(t, "")

	t.Run("openstack_cinder_up_metric", func(t *testing.T) {
		metrics.requireUp(t, "openstack_cinder_up")
	})

	t.Run("cinder_volume_gb_metric_present", func(t *testing.T) {
		metrics.requireMinValue(t, "openstack_cinder_volume_gb", labels{"id": volume.ID}, 1)
	})

	t.Run("cinder_snapshots_counted", func(t *testing.T) {
		metrics.requireMinValue(t, "openstack_cinder_snapshots", nil, 1)
	})

	t.Run("cinder_snapshot_gb_metric_present", func(t *testing.T) {
		metrics.requireMinValueWithLabels(t, "openstack_cinder_snapshot_gb", labels{
			"id":        snapshot.ID,
			"volume_id": volume.ID,
			"status":    "available",
		}, 1, "name", "tenant_id")
	})

	t.Run("cinder_backups_counted", func(t *testing.T) {
		metrics.requireMinValue(t, "openstack_cinder_backups", nil, 1)
	})

	t.Run("cinder_backup_gb_metric_present", func(t *testing.T) {
		metrics.requireMinValueWithLabels(t, "openstack_cinder_backup_gb", labels{
			"id":          backup.ID,
			"volume_id":   volume.ID,
			"status":      "available",
			"incremental": "false",
		}, 1, "name", "tenant_id")
	})
}
