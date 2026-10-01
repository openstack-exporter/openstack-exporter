package funcs

import (
	"context"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/images"
	"github.com/openstack-exporter/openstack-exporter/integration/tools"
)

// CreateImage creates an image with a randomly generated name and the given
// properties. No image data is uploaded; a queued image is enough for the
// exporter to report it.
// An error will be returned if the image was unable to be created.
func CreateImage(t *testing.T, client *gophercloud.ServiceClient, properties map[string]string) (*images.Image, error) {
	name := tools.RandomString("ACPTTEST", 16)
	t.Logf("Attempting to create image: %s", name)

	image, err := images.Create(context.TODO(), client, images.CreateOpts{
		Name:            name,
		ContainerFormat: "bare",
		DiskFormat:      "qcow2",
		Properties:      properties,
	}).Extract()
	if err != nil {
		return nil, err
	}

	t.Logf("Created image: %s", image.ID)
	return image, nil
}

// DeleteImage deletes an image via its ID.
// A fatal error will occur if the image failed to be deleted. This works
// best when using it as a deferred function.
func DeleteImage(t *testing.T, client *gophercloud.ServiceClient, image *images.Image) {
	if err := images.Delete(context.TODO(), client, image.ID).ExtractErr(); err != nil {
		t.Fatalf("Unable to delete image %s: %s", image.ID, err)
	}
	t.Logf("Deleted image: %s", image.ID)
}
