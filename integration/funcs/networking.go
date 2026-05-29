package funcs

import (
	"context"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
	"github.com/openstack-exporter/openstack-exporter/integration/tools"
)

// CreateNetwork creates a basic Neutron network with a random acceptance-test
// name. An error is returned if the network could not be created.
func CreateNetwork(t *testing.T, client *gophercloud.ServiceClient) (*networks.Network, error) {
	t.Helper()

	networkName := tools.RandomString("ACPTTEST", 16)
	createOpts := networks.CreateOpts{
		Name:         networkName,
		AdminStateUp: gophercloud.Enabled,
	}

	t.Logf("Attempting to create network: %s", networkName)

	network, err := networks.Create(context.TODO(), client, createOpts).Extract()
	if err != nil {
		return network, err
	}

	t.Logf("Successfully created network: %s", network.ID)
	return network, nil
}

// DeleteNetwork deletes a Neutron network. A fatal error occurs if the delete
// was not successful, which makes this suitable for deferred cleanup.
func DeleteNetwork(t *testing.T, client *gophercloud.ServiceClient, network *networks.Network) {
	t.Helper()

	t.Logf("Attempting to delete network: %s", network.ID)

	if err := networks.Delete(context.TODO(), client, network.ID).ExtractErr(); err != nil {
		t.Fatalf("Unable to delete network %s: %v", network.ID, err)
	}

	t.Logf("Deleted network: %s", network.ID)
}
