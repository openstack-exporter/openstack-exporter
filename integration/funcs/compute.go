// Package funcs contains exporter-specific integration resource setup.
package funcs

import (
	"context"
	"fmt"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	acceptance "github.com/gophercloud/gophercloud/v2/internal/acceptance/clients"
	compute "github.com/gophercloud/gophercloud/v2/internal/acceptance/openstack/compute/v2"
	"github.com/gophercloud/gophercloud/v2/internal/acceptance/tools"
	"github.com/gophercloud/gophercloud/v2/openstack-exporter-integration/clients"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	neutron "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

// CreateServer creates a basic instance with a randomly generated name.
// The flavor of the instance will be the value of the OS_FLAVOR_ID environment variable.
// The image will be the value of the OS_IMAGE_ID environment variable.
// The instance will be launched on the network specified in OS_NETWORK_NAME.
// An error will be returned if the instance was unable to be created.
func CreateServer(t *testing.T, client *gophercloud.ServiceClient) (*servers.Server, error) {
	choices, err := acceptance.AcceptanceTestChoicesFromEnv()
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

	if err := compute.WaitForComputeStatus(client, server, "ACTIVE"); err != nil {
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
