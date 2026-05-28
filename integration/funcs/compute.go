// Package funcs contains common functions for creating compute-based resources
// for use in acceptance tests. See the `*_test.go` files for example usages.
package funcs

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	neutron "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
	"github.com/openstack-exporter/openstack-exporter/integration/clients"
	"github.com/openstack-exporter/openstack-exporter/integration/tools"
)

// CreateServer creates a basic instance with a randomly generated name.
// The flavor of the instance will be the value of the OS_FLAVOR_ID environment variable.
// The image will be the value of the OS_IMAGE_ID environment variable.
// The instance will be launched on the network specified in OS_NETWORK_NAME.
// An error will be returned if the instance was unable to be created.
func CreateServer(t *testing.T, client *gophercloud.ServiceClient) (*servers.Server, error) {
	choices, err := clients.AcceptanceTestChoicesFromEnv()
	if err != nil {
		t.Fatal(err)
	}

	networkID, err := GetNetworkIDFromNetworks(t, client, choices.NetworkName)
	if err != nil {
		return nil, err
	}

	name := tools.RandomString("ACPTTEST", 16)
	t.Logf("Attempting to create server: %s", name)

	pwd := tools.MakeNewPassword("")

	server, err := servers.Create(context.TODO(), client, servers.CreateOpts{
		Name:      name,
		FlavorRef: choices.FlavorID,
		ImageRef:  choices.ImageID,
		AdminPass: pwd,
		Networks: []servers.Network{
			{UUID: networkID},
		},
		Metadata: map[string]string{
			"abc": "def",
		},
		Personality: servers.Personality{
			&servers.File{
				Path:     "/etc/test",
				Contents: []byte("hello world"),
			},
		},
	}, nil).Extract()
	if err != nil {
		return server, err
	}

	if err := WaitForComputeStatus(client, server, "ACTIVE"); err != nil {
		return nil, err
	}

	newServer, err := servers.Get(context.TODO(), client, server.ID).Extract()
	if err != nil {
		return nil, err
	}

	th.AssertEquals(t, name, newServer.Name)
	th.AssertEquals(t, choices.FlavorID, newServer.Flavor["id"])
	th.AssertEquals(t, choices.ImageID, newServer.Image["id"])

	return newServer, nil
}

// WaitForComputeStatus will poll an instance's status until it either matches
// the specified status or the status becomes ERROR.
func WaitForComputeStatus(client *gophercloud.ServiceClient, server *servers.Server, status string) error {
	return tools.WaitFor(func(ctx context.Context) (bool, error) {
		latest, err := servers.Get(ctx, client, server.ID).Extract()
		if err != nil {
			return false, err
		}

		if latest.Status == status {
			// Success!
			return true, nil
		}

		if latest.Status == "ERROR" {
			return false, fmt.Errorf("instance in ERROR state")
		}

		return false, nil
	})
}

// GetNetworkIDFromNetworks will return the network UUID for a given network
// name using the Neutron API.
// An error will be returned if the network could not be retrieved.
func GetNetworkIDFromNetworks(t *testing.T, client *gophercloud.ServiceClient, networkName string) (string, error) {
	networkClient, err := clients.NewNetworkV2Client()
	th.AssertNoErr(t, err)

	allPages2, err := neutron.List(networkClient, nil).AllPages(context.TODO())
	th.AssertNoErr(t, err)

	allNetworks, err := neutron.ExtractNetworks(allPages2)
	th.AssertNoErr(t, err)

	for _, network := range allNetworks {
		if network.Name == networkName {
			return network.ID, nil
		}
	}

	return "", fmt.Errorf("failed to obtain network ID for network %s", networkName)
}

// DeleteServer deletes an instance via its UUID.
// A fatal error will occur if the instance failed to be destroyed. This works
// best when using it as a deferred function.
func DeleteServer(t *testing.T, client *gophercloud.ServiceClient, server *servers.Server) {
	err := servers.Delete(context.TODO(), client, server.ID).ExtractErr()
	if err != nil {
		t.Fatalf("Unable to delete server %s: %s", server.ID, err)
	}

	if err := WaitForComputeStatus(client, server, "DELETED"); err != nil {
		if gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
			t.Logf("Deleted server: %s", server.ID)
			return
		}
		t.Fatalf("Error deleting server %s: %s", server.ID, err)
	}

	// If we reach this point, the API returned an actual DELETED status
	// which is a very short window of time, but happens occasionally.
	t.Logf("Deleted server: %s", server.ID)
}
