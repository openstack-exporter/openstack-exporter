package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/common/extensions"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/mtu"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/vpnaas/endpointgroups"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/vpnaas/ikepolicies"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/vpnaas/ipsecpolicies"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/vpnaas/services"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/vpnaas/siteconnections"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets"
	"github.com/openstack-exporter/openstack-exporter/integration/clients"
	"github.com/openstack-exporter/openstack-exporter/integration/funcs"
)

type networkWithMTU struct {
	networks.Network
	mtu.NetworkMTUExt
}

func TestNetworkingIntegration(t *testing.T) {
	clients.RequireLong(t)

	cleanup := startExporter(t, "network")
	defer cleanup()

	metrics := scrapeMetrics(t, "")

	t.Run("openstack_neutron_up_metric", func(t *testing.T) {
		metrics.requireUp(t, "openstack_neutron_up")
	})

	t.Run("openstack_neutron_core_metrics_present", func(t *testing.T) {
		metrics.requireAnyFamily(t,
			"openstack_neutron_networks",
			"openstack_neutron_ports",
			"openstack_neutron_subnets",
			"openstack_neutron_router",
		)
	})

	t.Run("neutron_network_labels_present", func(t *testing.T) {
		metrics.requireSampleWithLabels(t, "openstack_neutron_network", "id", "name", "is_external", "is_shared", "provider_network_type")
	})
}

func TestNetworkingNetworkCreateDeleteUpdatesExporterMetrics(t *testing.T) {
	clients.RequireLong(t)

	networkClient := newNetworkClient(t)
	cleanup := startExporter(t, "network")
	defer cleanup()

	network, deleteNetwork := createNetwork(t, networkClient)

	metrics := scrapeMetrics(t, "after network create")
	metrics.requireMetric(t, "openstack_neutron_network", labels{
		"id":   network.ID,
		"name": network.Name,
	})

	deleteNetwork()

	scrapeMetrics(t, "after network delete").requireNoMetric(t, "openstack_neutron_network", labels{"id": network.ID})
}

func TestNetworkingNetworkMTUCreateUpdateDeleteUpdatesExporterMetrics(t *testing.T) {
	clients.RequireLong(t)

	networkClient := newNetworkClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if _, err := extensions.Get(ctx, networkClient, "net-mtu-writable").Extract(); err != nil {
		t.Skipf("Neutron net-mtu-writable extension is not available: %v", err)
	}

	cleanup := startExporter(t, "network")
	defer cleanup()

	createOpts := mtu.CreateOptsExt{
		CreateOptsBuilder: networks.CreateOpts{Name: fmt.Sprintf("openstack-exporter-mtu-%d", time.Now().UnixNano())},
		MTU:               1440,
	}
	var createdNetwork networkWithMTU
	if err := networks.Create(ctx, networkClient, createOpts).ExtractInto(&createdNetwork); err != nil {
		t.Fatalf("Failed to create Neutron network with MTU: %v", err)
	}
	networkDeleted := false
	t.Cleanup(func() {
		if !networkDeleted {
			_ = networks.Delete(context.Background(), networkClient, createdNetwork.ID).ExtractErr()
		}
	})

	scrapeMetrics(t, "after MTU network create").requireLabelValue(t, "openstack_neutron_network", labels{"id": createdNetwork.ID}, "mtu", "1440")

	updateOpts := mtu.UpdateOptsExt{UpdateOptsBuilder: networks.UpdateOpts{}, MTU: 1350}
	var updatedNetwork networkWithMTU
	if err := networks.Update(ctx, networkClient, createdNetwork.ID, updateOpts).ExtractInto(&updatedNetwork); err != nil {
		t.Fatalf("Failed to update Neutron network MTU: %v", err)
	}
	scrapeMetrics(t, "after MTU network update").requireLabelValue(t, "openstack_neutron_network", labels{"id": createdNetwork.ID}, "mtu", "1350")

	if err := networks.Delete(ctx, networkClient, createdNetwork.ID).ExtractErr(); err != nil {
		t.Fatalf("Failed to delete Neutron network with MTU: %v", err)
	}
	networkDeleted = true

	scrapeMetrics(t, "after MTU network delete").requireNoMetric(t, "openstack_neutron_network", labels{"id": createdNetwork.ID})
}

func TestNetworkingSubnetCreateDeleteUpdatesExporterMetrics(t *testing.T) {
	clients.RequireLong(t)

	networkClient := newNetworkClient(t)
	cleanup := startExporter(t, "network")
	defer cleanup()

	network, deleteNetwork := createNetwork(t, networkClient)
	subnet, deleteSubnet := createSubnet(t, networkClient, network)

	scrapeMetrics(t, "after subnet create").requireMetric(t, "openstack_neutron_subnet", labels{
		"id":         subnet.ID,
		"name":       subnet.Name,
		"network_id": network.ID,
		"cidr":       subnet.CIDR,
	})

	deleteSubnet()

	scrapeMetrics(t, "after subnet delete").requireNoMetric(t, "openstack_neutron_subnet", labels{"id": subnet.ID})

	deleteNetwork()
}

func TestNetworkingPortCreateDeleteUpdatesExporterMetrics(t *testing.T) {
	clients.RequireLong(t)

	networkClient := newNetworkClient(t)
	cleanup := startExporter(t, "network")
	defer cleanup()

	network, deleteNetwork := createNetwork(t, networkClient)
	port, deletePort := createPort(t, networkClient, network)

	scrapeMetrics(t, "after port create").requireMetric(t, "openstack_neutron_port", labels{
		"uuid":        port.ID,
		"network_id":  network.ID,
		"mac_address": port.MACAddress,
	})

	deletePort()

	scrapeMetrics(t, "after port delete").requireNoMetric(t, "openstack_neutron_port", labels{"uuid": port.ID})

	deleteNetwork()
}

func TestNetworkingIPAvailabilityIncludesCreatedSubnet(t *testing.T) {
	clients.RequireLong(t)

	networkClient := newNetworkClient(t)
	cleanup := startExporter(t, "network")
	defer cleanup()

	network, deleteNetwork := createNetwork(t, networkClient)
	subnet, deleteSubnet := createSubnet(t, networkClient, network)

	metrics := scrapeMetrics(t, "after subnet create")
	availabilityLabels := labels{
		"network_id":   network.ID,
		"network_name": network.Name,
		"subnet_name":  subnet.Name,
		"cidr":         subnet.CIDR,
		"ip_version":   "4",
	}
	for _, name := range []string{
		"openstack_neutron_network_ip_availabilities_total",
		"openstack_neutron_network_ip_availabilities_used",
	} {
		metrics.requireMetric(t, name, availabilityLabels)
	}

	deleteSubnet()
	deleteNetwork()
}

func TestNetworkingQuotaMetricsHaveExpectedLabels(t *testing.T) {
	clients.RequireLong(t)

	cleanup := startExporter(t, "network")
	defer cleanup()

	metrics := scrapeMetrics(t, "")
	for _, metricName := range []string{
		"openstack_neutron_quota_network",
		"openstack_neutron_quota_subnet",
		"openstack_neutron_quota_port",
		"openstack_neutron_quota_router",
		"openstack_neutron_quota_floatingip",
		"openstack_neutron_quota_security_group",
		"openstack_neutron_quota_security_group_rule",
	} {
		metrics.requireLabels(t, metricName, labels{"type": "limit"}, "tenant", "tenant_id", "type")
	}
}

func TestNetworkingVPNaaSCreateDeleteUpdatesExporterMetrics(t *testing.T) {
	clients.RequireLong(t)

	networkClient := newNetworkClient(t)
	requireVPNaaSExtension(t, networkClient)

	cleanup := startExporter(t, "network")
	defer cleanup()

	network, _ := createNetwork(t, networkClient)
	subnet, _ := createSubnet(t, networkClient, network)
	router, _ := createRouter(t, networkClient, requireExternalNetworkID(t))
	addRouterInterface(t, networkClient, router, subnet)

	ikePolicy, deleteIKEPolicy := createVPNIKEPolicy(t, networkClient)
	ipsecPolicy, deleteIPSecPolicy := createVPNIPSecPolicy(t, networkClient)
	vpnService, deleteVPNService := createVPNService(t, networkClient, router)
	localEndpointGroup, deleteLocalEndpointGroup := createVPNEndpointGroup(t, networkClient, endpointgroups.TypeSubnet, []string{subnet.ID})
	peerEndpointGroup, deletePeerEndpointGroup := createVPNEndpointGroup(t, networkClient, endpointgroups.TypeCIDR, []string{"10.42.0.0/24"})
	siteConnection, deleteSiteConnection := createVPNSiteConnection(t, networkClient, ikePolicy.ID, ipsecPolicy.ID, vpnService.ID, peerEndpointGroup.ID, localEndpointGroup.ID)

	metrics := scrapeMetrics(t, "after VPNaaS resources create")
	metrics.requireMinValue(t, "openstack_neutron_vpn_endpoint_groups", nil, 2)
	metrics.requireMinValue(t, "openstack_neutron_vpn_ike_policies", nil, 1)
	metrics.requireMinValue(t, "openstack_neutron_vpn_ipsec_policies", nil, 1)
	metrics.requireMinValue(t, "openstack_neutron_vpn_services", nil, 1)
	metrics.requireMetric(t, "openstack_neutron_vpn_service", labels{
		"id":        vpnService.ID,
		"router_id": router.ID,
		"name":      vpnService.Name,
	})
	metrics.requireMinValue(t, "openstack_neutron_vpn_siteconnections", nil, 1)
	metrics.requireMetric(t, "openstack_neutron_vpn_siteconnection", labels{
		"id":                siteConnection.ID,
		"name":              siteConnection.Name,
		"vpn_service_id":    vpnService.ID,
		"ike_policy_id":     ikePolicy.ID,
		"ipsec_policy_id":   ipsecPolicy.ID,
		"peer_ep_group_id":  peerEndpointGroup.ID,
		"local_ep_group_id": localEndpointGroup.ID,
		"peer_id":           siteConnection.PeerID,
		"admin_state_up":    "true",
	})

	deleteSiteConnection()
	deleteVPNService()
	deletePeerEndpointGroup()
	deleteLocalEndpointGroup()
	deleteIPSecPolicy()
	deleteIKEPolicy()

	metrics = scrapeMetrics(t, "after VPNaaS resources delete")
	metrics.requireNoMetric(t, "openstack_neutron_vpn_service", labels{"id": vpnService.ID})
	metrics.requireNoMetric(t, "openstack_neutron_vpn_siteconnection", labels{"id": siteConnection.ID})
}

func newNetworkClient(t *testing.T) *gophercloud.ServiceClient {
	t.Helper()

	client, err := clients.NewNetworkV2Client()
	if err != nil {
		t.Fatalf("Failed to build network client: %v", err)
	}
	return client
}

func requireVPNaaSExtension(t *testing.T, client *gophercloud.ServiceClient) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if _, err := extensions.Get(ctx, client, "vpnaas").Extract(); err != nil {
		t.Skipf("Neutron VPNaaS extension is not available; enable neutron-vpnaas in DevStack: %v", err)
	}
}

func requireExternalNetworkID(t *testing.T) string {
	t.Helper()

	externalNetworkID := os.Getenv("OS_EXTGW_ID")
	if externalNetworkID == "" {
		t.Fatal("OS_EXTGW_ID must be set to create a VPNaaS router with an external gateway")
	}
	return externalNetworkID
}

func createNetwork(t *testing.T, client *gophercloud.ServiceClient) (*networks.Network, func()) {
	t.Helper()

	network, err := funcs.CreateNetwork(t, client)
	if err != nil {
		t.Fatalf("Could not create test network: %v", err)
	}
	deleted := false
	delete := func() {
		if !deleted {
			funcs.DeleteNetwork(t, client, network)
			deleted = true
		}
	}
	t.Cleanup(delete)
	return network, delete
}

func createRouter(t *testing.T, client *gophercloud.ServiceClient, externalNetworkID string) (*routers.Router, func()) {
	t.Helper()

	router, err := funcs.CreateRouter(t, client, externalNetworkID)
	if err != nil {
		t.Fatalf("Could not create test router: %v", err)
	}
	deleted := false
	delete := func() {
		if !deleted {
			funcs.DeleteRouter(t, client, router)
			deleted = true
		}
	}
	t.Cleanup(delete)
	return router, delete
}

func addRouterInterface(t *testing.T, client *gophercloud.ServiceClient, router *routers.Router, subnet *subnets.Subnet) func() {
	t.Helper()

	if err := funcs.AddRouterInterface(t, client, router, subnet); err != nil {
		t.Fatalf("Could not add subnet %s to router %s: %v", subnet.ID, router.ID, err)
	}
	removed := false
	remove := func() {
		if !removed {
			funcs.RemoveRouterInterface(t, client, router, subnet)
			removed = true
		}
	}
	t.Cleanup(remove)
	return remove
}

func createSubnet(t *testing.T, client *gophercloud.ServiceClient, network *networks.Network) (*subnets.Subnet, func()) {
	t.Helper()

	subnet, err := funcs.CreateSubnet(t, client, network)
	if err != nil {
		t.Fatalf("Could not create test subnet: %v", err)
	}
	deleted := false
	delete := func() {
		if !deleted {
			funcs.DeleteSubnet(t, client, subnet)
			deleted = true
		}
	}
	t.Cleanup(delete)
	return subnet, delete
}

func createPort(t *testing.T, client *gophercloud.ServiceClient, network *networks.Network) (*ports.Port, func()) {
	t.Helper()

	port, err := funcs.CreatePort(t, client, network)
	if err != nil {
		t.Fatalf("Could not create test port: %v", err)
	}
	deleted := false
	delete := func() {
		if !deleted {
			funcs.DeletePort(t, client, port)
			deleted = true
		}
	}
	t.Cleanup(delete)
	return port, delete
}

func createVPNIKEPolicy(t *testing.T, client *gophercloud.ServiceClient) (*ikepolicies.Policy, func()) {
	t.Helper()

	policy, err := funcs.CreateVPNIKEPolicy(t, client)
	if err != nil {
		t.Fatalf("Could not create VPN IKE policy: %v", err)
	}
	deleted := false
	delete := func() {
		if !deleted {
			funcs.DeleteVPNIKEPolicy(t, client, policy.ID)
			deleted = true
		}
	}
	t.Cleanup(delete)
	return policy, delete
}

func createVPNIPSecPolicy(t *testing.T, client *gophercloud.ServiceClient) (*ipsecpolicies.Policy, func()) {
	t.Helper()

	policy, err := funcs.CreateVPNIPSecPolicy(t, client)
	if err != nil {
		t.Fatalf("Could not create VPN IPsec policy: %v", err)
	}
	deleted := false
	delete := func() {
		if !deleted {
			funcs.DeleteVPNIPSecPolicy(t, client, policy.ID)
			deleted = true
		}
	}
	t.Cleanup(delete)
	return policy, delete
}

func createVPNEndpointGroup(t *testing.T, client *gophercloud.ServiceClient, endpointType endpointgroups.EndpointType, endpoints []string) (*endpointgroups.EndpointGroup, func()) {
	t.Helper()

	group, err := funcs.CreateVPNEndpointGroup(t, client, endpointType, endpoints)
	if err != nil {
		t.Fatalf("Could not create VPN endpoint group: %v", err)
	}
	deleted := false
	delete := func() {
		if !deleted {
			funcs.DeleteVPNEndpointGroup(t, client, group.ID)
			deleted = true
		}
	}
	t.Cleanup(delete)
	return group, delete
}

func createVPNService(t *testing.T, client *gophercloud.ServiceClient, router *routers.Router) (*services.Service, func()) {
	t.Helper()

	service, err := funcs.CreateVPNService(t, client, router)
	if err != nil {
		t.Fatalf("Could not create VPN service: %v", err)
	}
	deleted := false
	delete := func() {
		if !deleted {
			funcs.DeleteVPNService(t, client, service.ID)
			deleted = true
		}
	}
	t.Cleanup(delete)
	return service, delete
}

func createVPNSiteConnection(t *testing.T, client *gophercloud.ServiceClient, ikePolicyID string, ipsecPolicyID string, serviceID string, peerEndpointGroupID string, localEndpointGroupID string) (*siteconnections.Connection, func()) {
	t.Helper()

	connection, err := funcs.CreateVPNSiteConnection(t, client, ikePolicyID, ipsecPolicyID, serviceID, peerEndpointGroupID, localEndpointGroupID)
	if err != nil {
		t.Fatalf("Could not create VPN site connection: %v", err)
	}
	deleted := false
	delete := func() {
		if !deleted {
			funcs.DeleteVPNSiteConnection(t, client, connection.ID)
			deleted = true
		}
	}
	t.Cleanup(delete)
	return connection, delete
}
