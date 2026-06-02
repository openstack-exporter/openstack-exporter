package funcs

import (
	"context"
	"fmt"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/vpnaas/endpointgroups"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/vpnaas/ikepolicies"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/vpnaas/ipsecpolicies"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/vpnaas/services"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/vpnaas/siteconnections"
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

// CreateRouter creates a basic Neutron router with a random acceptance-test
// name. An error is returned if the router could not be created.
func CreateRouter(t *testing.T, client *gophercloud.ServiceClient) (*routers.Router, error) {
	t.Helper()

	routerName := tools.RandomString("ACPTTEST", 16)
	adminStateUp := true
	createOpts := routers.CreateOpts{
		Name:         routerName,
		AdminStateUp: &adminStateUp,
	}

	t.Logf("Attempting to create router: %s", routerName)

	router, err := routers.Create(context.TODO(), client, createOpts).Extract()
	if err != nil {
		return router, err
	}

	t.Logf("Successfully created router: %s", router.ID)
	return router, nil
}

// DeleteRouter deletes a Neutron router. A fatal error occurs if the delete
// was not successful, which makes this suitable for deferred cleanup.
func DeleteRouter(t *testing.T, client *gophercloud.ServiceClient, router *routers.Router) {
	t.Helper()

	t.Logf("Attempting to delete router: %s", router.ID)

	if err := routers.Delete(context.TODO(), client, router.ID).ExtractErr(); err != nil {
		t.Fatalf("Unable to delete router %s: %v", router.ID, err)
	}

	t.Logf("Deleted router: %s", router.ID)
}

// AddRouterInterface attaches a subnet to a router.
func AddRouterInterface(t *testing.T, client *gophercloud.ServiceClient, router *routers.Router, subnet *subnets.Subnet) error {
	t.Helper()

	t.Logf("Attempting to add subnet %s to router %s", subnet.ID, router.ID)

	_, err := routers.AddInterface(context.TODO(), client, router.ID, routers.AddInterfaceOpts{
		SubnetID: subnet.ID,
	}).Extract()
	if err != nil {
		return err
	}

	t.Logf("Added subnet %s to router %s", subnet.ID, router.ID)
	return nil
}

// RemoveRouterInterface detaches a subnet from a router. A fatal error occurs
// if the detach was not successful, which makes this suitable for cleanup.
func RemoveRouterInterface(t *testing.T, client *gophercloud.ServiceClient, router *routers.Router, subnet *subnets.Subnet) {
	t.Helper()

	t.Logf("Attempting to remove subnet %s from router %s", subnet.ID, router.ID)

	if _, err := routers.RemoveInterface(context.TODO(), client, router.ID, routers.RemoveInterfaceOpts{
		SubnetID: subnet.ID,
	}).Extract(); err != nil {
		t.Fatalf("Unable to remove subnet %s from router %s: %v", subnet.ID, router.ID, err)
	}

	t.Logf("Removed subnet %s from router %s", subnet.ID, router.ID)
}

// CreateVPNIKEPolicy creates a VPNaaS IKE policy with a random acceptance-test
// name. An error is returned if the policy could not be created.
func CreateVPNIKEPolicy(t *testing.T, client *gophercloud.ServiceClient) (*ikepolicies.Policy, error) {
	t.Helper()

	policyName := tools.RandomString("ACPTTEST", 16)
	createOpts := ikepolicies.CreateOpts{
		Name:                policyName,
		AuthAlgorithm:       ikepolicies.AuthAlgorithmSHA256,
		EncryptionAlgorithm: ikepolicies.EncryptionAlgorithmAES128,
		PFS:                 ikepolicies.PFSGroup14,
		IKEVersion:          ikepolicies.IKEVersionv2,
	}

	t.Logf("Attempting to create VPN IKE policy: %s", policyName)

	policy, err := ikepolicies.Create(context.TODO(), client, createOpts).Extract()
	if err != nil {
		return policy, err
	}

	t.Logf("Successfully created VPN IKE policy: %s", policy.ID)
	return policy, nil
}

// DeleteVPNIKEPolicy deletes a VPNaaS IKE policy. A fatal error occurs if the
// delete was not successful, which makes this suitable for deferred cleanup.
func DeleteVPNIKEPolicy(t *testing.T, client *gophercloud.ServiceClient, policyID string) {
	t.Helper()

	t.Logf("Attempting to delete VPN IKE policy: %s", policyID)

	if err := ikepolicies.Delete(context.TODO(), client, policyID).ExtractErr(); err != nil {
		t.Fatalf("Unable to delete VPN IKE policy %s: %v", policyID, err)
	}

	t.Logf("Deleted VPN IKE policy: %s", policyID)
}

// CreateVPNIPSecPolicy creates a VPNaaS IPsec policy with a random
// acceptance-test name. An error is returned if the policy could not be
// created.
func CreateVPNIPSecPolicy(t *testing.T, client *gophercloud.ServiceClient) (*ipsecpolicies.Policy, error) {
	t.Helper()

	policyName := tools.RandomString("ACPTTEST", 16)
	createOpts := ipsecpolicies.CreateOpts{
		Name:                policyName,
		AuthAlgorithm:       ipsecpolicies.AuthAlgorithmSHA256,
		EncapsulationMode:   ipsecpolicies.EncapsulationModeTunnel,
		EncryptionAlgorithm: ipsecpolicies.EncryptionAlgorithmAES128,
		PFS:                 ipsecpolicies.PFSGroup14,
		TransformProtocol:   ipsecpolicies.TransformProtocolESP,
	}

	t.Logf("Attempting to create VPN IPsec policy: %s", policyName)

	policy, err := ipsecpolicies.Create(context.TODO(), client, createOpts).Extract()
	if err != nil {
		return policy, err
	}

	t.Logf("Successfully created VPN IPsec policy: %s", policy.ID)
	return policy, nil
}

// DeleteVPNIPSecPolicy deletes a VPNaaS IPsec policy. A fatal error occurs if
// the delete was not successful, which makes this suitable for deferred
// cleanup.
func DeleteVPNIPSecPolicy(t *testing.T, client *gophercloud.ServiceClient, policyID string) {
	t.Helper()

	t.Logf("Attempting to delete VPN IPsec policy: %s", policyID)

	if err := ipsecpolicies.Delete(context.TODO(), client, policyID).ExtractErr(); err != nil {
		t.Fatalf("Unable to delete VPN IPsec policy %s: %v", policyID, err)
	}

	t.Logf("Deleted VPN IPsec policy: %s", policyID)
}

// CreateVPNEndpointGroup creates a VPNaaS endpoint group. An error is returned
// if the endpoint group could not be created.
func CreateVPNEndpointGroup(t *testing.T, client *gophercloud.ServiceClient, endpointType endpointgroups.EndpointType, endpoints []string) (*endpointgroups.EndpointGroup, error) {
	t.Helper()

	groupName := tools.RandomString("ACPTTEST", 16)
	createOpts := endpointgroups.CreateOpts{
		Name:      groupName,
		Type:      endpointType,
		Endpoints: endpoints,
	}

	t.Logf("Attempting to create VPN endpoint group: %s", groupName)

	group, err := endpointgroups.Create(context.TODO(), client, createOpts).Extract()
	if err != nil {
		return group, err
	}

	t.Logf("Successfully created VPN endpoint group: %s", group.ID)
	return group, nil
}

// DeleteVPNEndpointGroup deletes a VPNaaS endpoint group. A fatal error occurs
// if the delete was not successful, which makes this suitable for deferred
// cleanup.
func DeleteVPNEndpointGroup(t *testing.T, client *gophercloud.ServiceClient, groupID string) {
	t.Helper()

	t.Logf("Attempting to delete VPN endpoint group: %s", groupID)

	if err := endpointgroups.Delete(context.TODO(), client, groupID).ExtractErr(); err != nil {
		t.Fatalf("Unable to delete VPN endpoint group %s: %v", groupID, err)
	}

	t.Logf("Deleted VPN endpoint group: %s", groupID)
}

// CreateVPNService creates a VPNaaS service for the specified router. An error
// is returned if the service could not be created.
func CreateVPNService(t *testing.T, client *gophercloud.ServiceClient, router *routers.Router) (*services.Service, error) {
	t.Helper()

	serviceName := tools.RandomString("ACPTTEST", 16)
	adminStateUp := true
	createOpts := services.CreateOpts{
		Name:         serviceName,
		AdminStateUp: &adminStateUp,
		RouterID:     router.ID,
	}

	t.Logf("Attempting to create VPN service: %s", serviceName)

	service, err := services.Create(context.TODO(), client, createOpts).Extract()
	if err != nil {
		return service, err
	}

	t.Logf("Successfully created VPN service: %s", service.ID)
	return service, nil
}

// DeleteVPNService deletes a VPNaaS service. A fatal error occurs if the delete
// was not successful, which makes this suitable for deferred cleanup.
func DeleteVPNService(t *testing.T, client *gophercloud.ServiceClient, serviceID string) {
	t.Helper()

	t.Logf("Attempting to delete VPN service: %s", serviceID)

	if err := services.Delete(context.TODO(), client, serviceID).ExtractErr(); err != nil {
		t.Fatalf("Unable to delete VPN service %s: %v", serviceID, err)
	}

	t.Logf("Deleted VPN service: %s", serviceID)
}

// CreateVPNSiteConnection creates a VPNaaS IPsec site connection. An error is
// returned if the connection could not be created.
func CreateVPNSiteConnection(t *testing.T, client *gophercloud.ServiceClient, ikePolicyID string, ipsecPolicyID string, serviceID string, peerEndpointGroupID string, localEndpointGroupID string) (*siteconnections.Connection, error) {
	t.Helper()

	connectionName := tools.RandomString("ACPTTEST", 16)
	peerAddress := "172.24.4.233"
	createOpts := siteconnections.CreateOpts{
		Name:           connectionName,
		PSK:            "secret",
		Initiator:      siteconnections.InitiatorBiDirectional,
		AdminStateUp:   gophercloud.Enabled,
		IPSecPolicyID:  ipsecPolicyID,
		PeerEPGroupID:  peerEndpointGroupID,
		IKEPolicyID:    ikePolicyID,
		VPNServiceID:   serviceID,
		LocalEPGroupID: localEndpointGroupID,
		PeerAddress:    peerAddress,
		PeerID:         peerAddress,
		MTU:            1500,
	}

	t.Logf("Attempting to create VPN site connection: %s", connectionName)

	connection, err := siteconnections.Create(context.TODO(), client, createOpts).Extract()
	if err != nil {
		return connection, err
	}

	t.Logf("Successfully created VPN site connection: %s", connection.ID)
	return connection, nil
}

// DeleteVPNSiteConnection deletes a VPNaaS IPsec site connection. A fatal
// error occurs if the delete was not successful, which makes this suitable for
// deferred cleanup.
func DeleteVPNSiteConnection(t *testing.T, client *gophercloud.ServiceClient, connectionID string) {
	t.Helper()

	t.Logf("Attempting to delete VPN site connection: %s", connectionID)

	if err := siteconnections.Delete(context.TODO(), client, connectionID).ExtractErr(); err != nil {
		t.Fatalf("Unable to delete VPN site connection %s: %v", connectionID, err)
	}

	t.Logf("Deleted VPN site connection: %s", connectionID)
}
