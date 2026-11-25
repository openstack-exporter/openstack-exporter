package funcs

import (
	"context"
	"path"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/keymanager/v1/secrets"
	"github.com/openstack-exporter/openstack-exporter/integration/tools"
)

// CreateSecret stores a small passphrase secret with a randomly generated
// name and returns its ID.
// An error will be returned if the secret was unable to be created.
func CreateSecret(t *testing.T, client *gophercloud.ServiceClient) (string, error) {
	name := tools.RandomString("ACPTTEST", 16)
	t.Logf("Attempting to create secret: %s", name)

	secret, err := secrets.Create(context.TODO(), client, secrets.CreateOpts{
		Name:               name,
		SecretType:         secrets.PassphraseSecret,
		Payload:            tools.RandomString("", 16),
		PayloadContentType: "text/plain",
	}).Extract()
	if err != nil {
		return "", err
	}

	id := path.Base(secret.SecretRef)
	t.Logf("Created secret: %s", id)
	return id, nil
}

// DeleteSecret deletes the secret with the given ID. A fatal error will
// occur if the secret could not be deleted.
func DeleteSecret(t *testing.T, client *gophercloud.ServiceClient, id string) {
	if err := secrets.Delete(context.TODO(), client, id).ExtractErr(); err != nil {
		t.Fatalf("Unable to delete secret %s: %s", id, err)
	}
	t.Logf("Deleted secret: %s", id)
}
