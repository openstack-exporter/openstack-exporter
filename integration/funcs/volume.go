package funcs

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/backups"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/snapshots"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"
	"github.com/openstack-exporter/openstack-exporter/integration/tools"
)

// CreateVolume creates a 1 GB volume with a randomly generated name and
// waits for it to become available.
// An error will be returned if the volume was unable to be created.
func CreateVolume(t *testing.T, client *gophercloud.ServiceClient) (*volumes.Volume, error) {
	name := tools.RandomString("ACPTTEST", 16)
	t.Logf("Attempting to create volume: %s", name)

	volume, err := volumes.Create(context.TODO(), client, volumes.CreateOpts{
		Size: 1,
		Name: name,
	}, nil).Extract()
	if err != nil {
		return nil, err
	}

	if err := waitForStatus(volume.ID, "available", func(ctx context.Context) (string, error) {
		latest, err := volumes.Get(ctx, client, volume.ID).Extract()
		if err != nil {
			return "", err
		}
		return latest.Status, nil
	}); err != nil {
		return nil, err
	}

	t.Logf("Created volume: %s", volume.ID)
	return volumes.Get(context.TODO(), client, volume.ID).Extract()
}

// DeleteVolume deletes a volume via its ID.
// A fatal error will occur if the volume failed to be deleted. This works
// best when using it as a deferred function.
func DeleteVolume(t *testing.T, client *gophercloud.ServiceClient, volume *volumes.Volume) {
	if err := volumes.Delete(context.TODO(), client, volume.ID, volumes.DeleteOpts{}).ExtractErr(); err != nil {
		t.Fatalf("Unable to delete volume %s: %s", volume.ID, err)
	}
	t.Logf("Deleted volume: %s", volume.ID)
}

// CreateSnapshot creates a snapshot of the given volume with a randomly
// generated name and waits for it to become available.
// An error will be returned if the snapshot was unable to be created.
func CreateSnapshot(t *testing.T, client *gophercloud.ServiceClient, volume *volumes.Volume) (*snapshots.Snapshot, error) {
	name := tools.RandomString("ACPTTEST", 16)
	t.Logf("Attempting to create snapshot: %s", name)

	snapshot, err := snapshots.Create(context.TODO(), client, snapshots.CreateOpts{
		VolumeID: volume.ID,
		Name:     name,
	}).Extract()
	if err != nil {
		return nil, err
	}

	if err := waitForStatus(snapshot.ID, "available", func(ctx context.Context) (string, error) {
		latest, err := snapshots.Get(ctx, client, snapshot.ID).Extract()
		if err != nil {
			return "", err
		}
		return latest.Status, nil
	}); err != nil {
		return nil, err
	}

	t.Logf("Created snapshot: %s", snapshot.ID)
	return snapshots.Get(context.TODO(), client, snapshot.ID).Extract()
}

// DeleteSnapshot deletes a snapshot via its ID and waits until it is gone,
// so that a deferred DeleteVolume for its volume can succeed afterwards.
// A fatal error will occur if the snapshot failed to be deleted.
func DeleteSnapshot(t *testing.T, client *gophercloud.ServiceClient, snapshot *snapshots.Snapshot) {
	if err := snapshots.Delete(context.TODO(), client, snapshot.ID).ExtractErr(); err != nil {
		t.Fatalf("Unable to delete snapshot %s: %s", snapshot.ID, err)
	}

	if err := waitForDeleted(func(ctx context.Context) error {
		return snapshots.Get(ctx, client, snapshot.ID).Err
	}); err != nil {
		t.Fatalf("Error waiting for snapshot %s to be deleted: %s", snapshot.ID, err)
	}
	t.Logf("Deleted snapshot: %s", snapshot.ID)
}

// CreateBackup creates a backup of the given volume with a randomly
// generated name and waits for it to become available.
// An error will be returned if the backup was unable to be created.
func CreateBackup(t *testing.T, client *gophercloud.ServiceClient, volume *volumes.Volume) (*backups.Backup, error) {
	name := tools.RandomString("ACPTTEST", 16)
	t.Logf("Attempting to create backup: %s", name)

	backup, err := backups.Create(context.TODO(), client, backups.CreateOpts{
		VolumeID: volume.ID,
		Name:     name,
	}).Extract()
	if err != nil {
		return nil, err
	}

	if err := waitForStatus(backup.ID, "available", func(ctx context.Context) (string, error) {
		latest, err := backups.Get(ctx, client, backup.ID).Extract()
		if err != nil {
			return "", err
		}
		return latest.Status, nil
	}); err != nil {
		return nil, err
	}

	t.Logf("Created backup: %s", backup.ID)
	return backups.Get(context.TODO(), client, backup.ID).Extract()
}

// DeleteBackup deletes a backup via its ID and waits until it is gone.
// A fatal error will occur if the backup failed to be deleted.
func DeleteBackup(t *testing.T, client *gophercloud.ServiceClient, backup *backups.Backup) {
	if err := backups.Delete(context.TODO(), client, backup.ID).ExtractErr(); err != nil {
		t.Fatalf("Unable to delete backup %s: %s", backup.ID, err)
	}

	if err := waitForDeleted(func(ctx context.Context) error {
		return backups.Get(ctx, client, backup.ID).Err
	}); err != nil {
		t.Fatalf("Error waiting for backup %s to be deleted: %s", backup.ID, err)
	}
	t.Logf("Deleted backup: %s", backup.ID)
}

// waitForStatus polls a resource's status until it matches the wanted one,
// failing early if the resource enters an error status.
func waitForStatus(id, status string, get func(context.Context) (string, error)) error {
	return tools.WaitFor(func(ctx context.Context) (bool, error) {
		latest, err := get(ctx)
		if err != nil {
			return false, err
		}
		if latest == status {
			return true, nil
		}
		if latest == "error" {
			return false, fmt.Errorf("resource %s is in error status", id)
		}
		return false, nil
	})
}

// waitForDeleted polls a resource until requests for it return 404.
func waitForDeleted(get func(context.Context) error) error {
	return tools.WaitFor(func(ctx context.Context) (bool, error) {
		err := get(ctx)
		if err == nil {
			return false, nil
		}
		if gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
			return true, nil
		}
		return false, err
	})
}
