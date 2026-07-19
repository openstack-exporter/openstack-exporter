package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2/openstack/common/extensions"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/mtu"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/vpnaas/endpointgroups"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
	"github.com/openstack-exporter/openstack-exporter/integration/clients"
	"github.com/openstack-exporter/openstack-exporter/integration/funcs"
)

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

	networkClient := funcs.NewNetworkClient(t)
	cleanup := startExporter(t, "network")
	defer cleanup()

	network, deleteNetwork := funcs.MustCreateNetwork(t, networkClient)

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

	networkClient := funcs.NewNetworkClient(t)

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
	var createdNetwork funcs.NetworkWithMTU
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
	var updatedNetwork funcs.NetworkWithMTU
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

	networkClient := funcs.NewNetworkClient(t)
	cleanup := startExporter(t, "network")
	defer cleanup()

	network, deleteNetwork := funcs.MustCreateNetwork(t, networkClient)
	subnet, deleteSubnet := funcs.MustCreateSubnet(t, networkClient, network)

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

	networkClient := funcs.NewNetworkClient(t)
	cleanup := startExporter(t, "network")
	defer cleanup()

	network, deleteNetwork := funcs.MustCreateNetwork(t, networkClient)
	port, deletePort := funcs.MustCreatePort(t, networkClient, network)

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

	networkClient := funcs.NewNetworkClient(t)
	cleanup := startExporter(t, "network")
	defer cleanup()

	network, deleteNetwork := funcs.MustCreateNetwork(t, networkClient)
	subnet, deleteSubnet := funcs.MustCreateSubnet(t, networkClient, network)

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

	networkClient := funcs.NewNetworkClient(t)
	funcs.RequireVPNaaSExtension(t, networkClient)

	cleanup := startExporter(t, "network")
	defer cleanup()

	network, _ := funcs.MustCreateNetwork(t, networkClient)
	subnet, _ := funcs.MustCreateSubnet(t, networkClient, network)
	router, _ := funcs.MustCreateRouter(t, networkClient, funcs.RequireExternalNetworkID(t))
	funcs.MustAddRouterInterface(t, networkClient, router, subnet)

	ikePolicy, deleteIKEPolicy := funcs.MustCreateVPNIKEPolicy(t, networkClient)
	ipsecPolicy, deleteIPSecPolicy := funcs.MustCreateVPNIPSecPolicy(t, networkClient)
	vpnService, deleteVPNService := funcs.MustCreateVPNService(t, networkClient, router)
	localEndpointGroup, deleteLocalEndpointGroup := funcs.MustCreateVPNEndpointGroup(t, networkClient, endpointgroups.TypeSubnet, []string{subnet.ID})
	peerEndpointGroup, deletePeerEndpointGroup := funcs.MustCreateVPNEndpointGroup(t, networkClient, endpointgroups.TypeCIDR, []string{"10.42.0.0/24"})
	siteConnection, deleteSiteConnection := funcs.MustCreateVPNSiteConnection(t, networkClient, ikePolicy.ID, ipsecPolicy.ID, vpnService.ID, peerEndpointGroup.ID, localEndpointGroup.ID)

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
