package funcs

import (
	"context"
	"fmt"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets"
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

// CreateSubnet creates a basic IPv4 subnet on the specified Neutron network.
// An error is returned if the subnet could not be created.
func CreateSubnet(t *testing.T, client *gophercloud.ServiceClient, network *networks.Network) (*subnets.Subnet, error) {
	t.Helper()

	subnetOctet := tools.RandomInt(1, 250)
	subnetCIDR := fmt.Sprintf("192.168.%d.0/24", subnetOctet)
	subnetGateway := fmt.Sprintf("192.168.%d.1", subnetOctet)
	subnetName := tools.RandomString("ACPTTEST", 16)
	createOpts := subnets.CreateOpts{
		NetworkID:  network.ID,
		CIDR:       subnetCIDR,
		IPVersion:  4,
		Name:       subnetName,
		EnableDHCP: gophercloud.Disabled,
		GatewayIP:  &subnetGateway,
	}

	t.Logf("Attempting to create subnet: %s", subnetName)

	subnet, err := subnets.Create(context.TODO(), client, createOpts).Extract()
	if err != nil {
		return subnet, err
	}

	t.Logf("Successfully created subnet: %s", subnet.ID)
	return subnet, nil
}

// DeleteSubnet deletes a Neutron subnet. A fatal error occurs if the delete
// was not successful, which makes this suitable for deferred cleanup.
func DeleteSubnet(t *testing.T, client *gophercloud.ServiceClient, subnet *subnets.Subnet) {
	t.Helper()

	t.Logf("Attempting to delete subnet: %s", subnet.ID)

	if err := subnets.Delete(context.TODO(), client, subnet.ID).ExtractErr(); err != nil {
		t.Fatalf("Unable to delete subnet %s: %v", subnet.ID, err)
	}

	t.Logf("Deleted subnet: %s", subnet.ID)
}

// CreatePort creates a basic Neutron port on the specified network. An error
// is returned if the port could not be created.
func CreatePort(t *testing.T, client *gophercloud.ServiceClient, network *networks.Network) (*ports.Port, error) {
	t.Helper()

	portName := tools.RandomString("ACPTTEST", 16)
	createOpts := ports.CreateOpts{
		NetworkID:    network.ID,
		Name:         portName,
		AdminStateUp: gophercloud.Enabled,
	}

	t.Logf("Attempting to create port: %s", portName)

	port, err := ports.Create(context.TODO(), client, createOpts).Extract()
	if err != nil {
		return port, err
	}

	t.Logf("Successfully created port: %s", port.ID)
	return port, nil
}

// DeletePort deletes a Neutron port. A fatal error occurs if the delete was
// not successful, which makes this suitable for deferred cleanup.
func DeletePort(t *testing.T, client *gophercloud.ServiceClient, port *ports.Port) {
	t.Helper()

	t.Logf("Attempting to delete port: %s", port.ID)

	if err := ports.Delete(context.TODO(), client, port.ID).ExtractErr(); err != nil {
		t.Fatalf("Unable to delete port %s: %v", port.ID, err)
	}

	t.Logf("Deleted port: %s", port.ID)
}
