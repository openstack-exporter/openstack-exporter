package openstack

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

type NeutronTestSuite struct {
	BaseOpenStackTestSuite
}

var neutronExpectedUp = `
# HELP openstack_neutron_agent_state agent_state
# TYPE openstack_neutron_agent_state counter
openstack_neutron_agent_state{adminState="up",availability_zone="",hostname="agenthost1",id="04c62b91-b799-48b7-9cd5-2982db6df9c6",service="neutron-openvswitch-agent"} 1
openstack_neutron_agent_state{adminState="up",availability_zone="",hostname="agenthost1",id="2bf84eaf-d869-49cc-8401-cbbca5177e59",service="neutron-lbaasv2-agent"} 1
openstack_neutron_agent_state{adminState="up",availability_zone="nova",hostname="agenthost1",id="840d5d68-5759-4e9e-812f-f3bd19214c7f",service="neutron-dhcp-agent"} 1
openstack_neutron_agent_state{adminState="up",availability_zone="nova",hostname="agenthost1",id="a09b81fc-5a42-46d3-a306-1a5d122a7787",service="neutron-l3-agent"} 1
openstack_neutron_agent_state{adminState="up",availability_zone="",hostname="agenthost1",id="c876c9f7-1058-4b9b-90ed-20fb3f905ec4",service="neutron-metadata-agent"} 1
# HELP openstack_neutron_floating_ip floating_ip
# TYPE openstack_neutron_floating_ip gauge
openstack_neutron_floating_ip{floating_ip_address="172.24.4.227",floating_network_id="1c93472c-4d8a-11ea-92e9-08002759fd91",id="231facca-4d8a-11ea-a143-08002759fd91",project_id="0042b7564d8a11eabc2d08002759fd91",router_id="",status="DOWN"} 1
openstack_neutron_floating_ip{floating_ip_address="172.24.4.227",floating_network_id="376da547-b977-4cfe-9cba-275c80debf57",id="61cea855-49cb-4846-997d-801b70c71bdd",project_id="4969c491a3c74ee4af974e6d800c62de",router_id="",status="DOWN"} 1
openstack_neutron_floating_ip{floating_ip_address="172.24.4.228",floating_network_id="376da547-b977-4cfe-9cba-275c80debf57",id="2f245a7b-796b-4f26-9cf9-9e82d248fda7",project_id="4969c491a3c74ee4af974e6d800c62de",router_id="d23abc8d-2991-4a55-ba98-2aaea84cc72f",status="ACTIVE"} 1
openstack_neutron_floating_ip{floating_ip_address="172.24.4.42",floating_network_id="376da547-b977-4cfe-9cba-275c80debf57",id="898b198e-49f7-47d6-a7e1-53f626a548e6",project_id="4969c491a3c74ee4af974e6d800c62de",router_id="0303bf18-2c52-479c-bd68-e0ad712a1639",status="ACTIVE"} 1
# HELP openstack_neutron_floating_ips floating_ips
# TYPE openstack_neutron_floating_ips gauge
openstack_neutron_floating_ips 4
# HELP openstack_neutron_floating_ips_associated_not_active floating_ips_associated_not_active
# TYPE openstack_neutron_floating_ips_associated_not_active gauge
openstack_neutron_floating_ips_associated_not_active 1
# HELP openstack_neutron_l3_agent_of_router l3_agent_of_router
# TYPE openstack_neutron_l3_agent_of_router gauge
openstack_neutron_l3_agent_of_router{agent_admin_up="true",agent_alive="true",agent_host="dev-os-ctrl-02",ha_state="",l3_agent_id="ddbf087c-e38f-4a73-bcb3-c38f2a719a03",router_id="9daeb7dd-7e3f-4e44-8c42-c7a0e8c8a42f"} 1
openstack_neutron_l3_agent_of_router{agent_admin_up="true",agent_alive="true",agent_host="dev-os-ctrl-02",ha_state="",l3_agent_id="ddbf087c-e38f-4a73-bcb3-c38f2a719a03",router_id="f8a44de0-fc8e-45df-93c7-f79bf3b01c95"} 1
# HELP openstack_neutron_network network
# TYPE openstack_neutron_network gauge
openstack_neutron_network{id="d32019d3-bc6e-4319-9c1d-6722fc136a22",is_external="false",is_shared="false",name="net1",provider_network_type="vlan",provider_physical_network="public",provider_segmentation_id="3",status="ACTIVE",subnets="54d6f61d-db07-451c-9ab3-b9609b6b6f0b",tags="tag1,tag2",tenant_id="4fd44f30292945e481c7b8a0c8908869"} 0
openstack_neutron_network{id="db193ab3-96e3-4cb3-8fc5-05f4296d0324",is_external="false",is_shared="false",name="net2",provider_network_type="local",provider_physical_network="",provider_segmentation_id="",status="ACTIVE",subnets="08eae331-0402-425a-923c-34f7cfe39c1b",tags="tag1,tag2",tenant_id="26a7980765d0414dbc1fc1f88cdb7e6e"} 0
# HELP openstack_neutron_network_ip_availabilities_total network_ip_availabilities_total
# TYPE openstack_neutron_network_ip_availabilities_total gauge
openstack_neutron_network_ip_availabilities_total{cidr="10.0.0.0/24",ip_version="4",network_id="6801d9c8-20e6-4b27-945d-62499f00002e",network_name="private",project_id="d56d3b8dd6894a508cf41b96b522328c",subnet_name="private-subnet"} 253
openstack_neutron_network_ip_availabilities_total{cidr="172.24.4.0/24",ip_version="4",network_id="4cf895c9-c3d1-489e-b02e-59b5c8976809",network_name="public",project_id="1a02cc95f1734fcc9d3c753818f03002",subnet_name="public-subnet"} 253
openstack_neutron_network_ip_availabilities_total{cidr="2001:db8::/64",ip_version="6",network_id="4cf895c9-c3d1-489e-b02e-59b5c8976809",network_name="public",project_id="1a02cc95f1734fcc9d3c753818f03002",subnet_name="ipv6-public-subnet"} 1.8446744073709552e+19
openstack_neutron_network_ip_availabilities_total{cidr="fdbf:ac66:9be8::/64",ip_version="6",network_id="6801d9c8-20e6-4b27-945d-62499f00002e",network_name="private",project_id="d56d3b8dd6894a508cf41b96b522328c",subnet_name="ipv6-private-subnet"} 1.8446744073709552e+19
# HELP openstack_neutron_network_ip_availabilities_used network_ip_availabilities_used
# TYPE openstack_neutron_network_ip_availabilities_used gauge
openstack_neutron_network_ip_availabilities_used{cidr="10.0.0.0/24",ip_version="4",network_id="6801d9c8-20e6-4b27-945d-62499f00002e",network_name="private",project_id="d56d3b8dd6894a508cf41b96b522328c",subnet_name="private-subnet"} 2
openstack_neutron_network_ip_availabilities_used{cidr="172.24.4.0/24",ip_version="4",network_id="4cf895c9-c3d1-489e-b02e-59b5c8976809",network_name="public",project_id="1a02cc95f1734fcc9d3c753818f03002",subnet_name="public-subnet"} 1
openstack_neutron_network_ip_availabilities_used{cidr="2001:db8::/64",ip_version="6",network_id="4cf895c9-c3d1-489e-b02e-59b5c8976809",network_name="public",project_id="1a02cc95f1734fcc9d3c753818f03002",subnet_name="ipv6-public-subnet"} 1
openstack_neutron_network_ip_availabilities_used{cidr="fdbf:ac66:9be8::/64",ip_version="6",network_id="6801d9c8-20e6-4b27-945d-62499f00002e",network_name="private",project_id="d56d3b8dd6894a508cf41b96b522328c",subnet_name="ipv6-private-subnet"} 2
# HELP openstack_neutron_networks networks
# TYPE openstack_neutron_networks gauge
openstack_neutron_networks 2
# HELP openstack_neutron_port port
# TYPE openstack_neutron_port gauge
openstack_neutron_port{admin_state_up="true",binding_vif_type="",device_owner="network:router_gateway",fixed_ips="",mac_address="fa:16:3e:58:42:ed",network_id="70c1db1f-b701-45bd-96e0-a313ee3430b3",status="ACTIVE",uuid="d80b1a3b-4fc1-49f3-952e-1e2ab7081d8b"} 1
openstack_neutron_port{admin_state_up="true",binding_vif_type="",device_owner="network:router_interface",fixed_ips="10.0.0.1",mac_address="fa:16:3e:bb:3c:e4",network_id="f27aa545-cbdd-4907-b0c6-c9e8b039dcc2",status="ACTIVE",uuid="f71a6703-d6de-4be1-a91a-a570ede1d159"} 1
openstack_neutron_port{admin_state_up="true",binding_vif_type="ovs",device_owner="neutron:LOADBALANCERV2",fixed_ips="192.168.36.198,192.168.36.254,",mac_address="fa:16:3e:0b:14:fd",network_id="675c54a5-a9f3-4f5e-a0b4-e026b29c217b",status="N/A",uuid="f0b24508-eb48-4530-a38b-c042df147101"} 1
# HELP openstack_neutron_ports ports
# TYPE openstack_neutron_ports gauge
openstack_neutron_ports 3
# HELP openstack_neutron_ports_lb_not_active ports_lb_not_active
# TYPE openstack_neutron_ports_lb_not_active gauge
openstack_neutron_ports_lb_not_active 1
# HELP openstack_neutron_ports_no_ips ports_no_ips
# TYPE openstack_neutron_ports_no_ips gauge
openstack_neutron_ports_no_ips 1
# HELP openstack_neutron_router router
# TYPE openstack_neutron_router gauge
openstack_neutron_router{admin_state_up="true",external_network_id="78620e54-9ec2-4372-8b07-3ac2d02e0288",id="9daeb7dd-7e3f-4e44-8c42-c7a0e8c8a42f",name="router2",project_id="a2a651cc26974de98c9a1f9aa88eb2e6",status="N/A"} 1
openstack_neutron_router{admin_state_up="true",external_network_id="78620e54-9ec2-4372-8b07-3ac2d02e0288",id="f8a44de0-fc8e-45df-93c7-f79bf3b01c95",name="router1",project_id="a2a651cc26974de98c9a1f9aa88eb2e6",status="ACTIVE"} 1
# HELP openstack_neutron_routers routers
# TYPE openstack_neutron_routers gauge
openstack_neutron_routers 2
# HELP openstack_neutron_routers_not_active routers_not_active
# TYPE openstack_neutron_routers_not_active gauge
openstack_neutron_routers_not_active 1
# HELP openstack_neutron_security_groups security_groups
# TYPE openstack_neutron_security_groups gauge
openstack_neutron_security_groups 1
# HELP openstack_neutron_subnet subnet
# TYPE openstack_neutron_subnet gauge
openstack_neutron_subnet{cidr="10.0.0.0/24",dns_nameservers="",enable_dhcp="true",gateway_ip="10.0.0.1",id="08eae331-0402-425a-923c-34f7cfe39c1b",name="private-subnet",network_id="db193ab3-96e3-4cb3-8fc5-05f4296d0324",tags="tag1,tag2",tenant_id="26a7980765d0414dbc1fc1f88cdb7e6e"} 1
openstack_neutron_subnet{cidr="10.10.0.0/24",dns_nameservers="",enable_dhcp="true",gateway_ip="10.10.0.1",id="12769bb8-6c3c-11ec-8124-002b67875abf",name="pooled-subnet-ipv4",network_id="d32019d3-bc6e-4319-9c1d-6722fc136a22",tags="tag1,tag2",tenant_id="4fd44f30292945e481c7b8a0c8908869"} 1
openstack_neutron_subnet{cidr="192.0.0.0/8",dns_nameservers="",enable_dhcp="true",gateway_ip="192.0.0.1",id="54d6f61d-db07-451c-9ab3-b9609b6b6f0b",name="my_subnet",network_id="d32019d3-bc6e-4319-9c1d-6722fc136a22",tags="tag1,tag2",tenant_id="4fd44f30292945e481c7b8a0c8908869"} 1
openstack_neutron_subnet{cidr="2001:db8::/64",dns_nameservers="",enable_dhcp="true",gateway_ip="2001:db8::1",id="f73defec-6c43-11ec-a08b-002b67875abf",name="pooled-subnet-ipv6",network_id="d32019d3-bc6e-4319-9c1d-6722fc136a22",tags="tag1,tag2",tenant_id="4fd44f30292945e481c7b8a0c8908869"} 1
# HELP openstack_neutron_subnets subnets
# TYPE openstack_neutron_subnets gauge
openstack_neutron_subnets 4
# HELP openstack_neutron_subnets_free subnets_free
# TYPE openstack_neutron_subnets_free gauge
openstack_neutron_subnets_free{ip_version="4",prefix="10.10.0.0/21",prefix_length="24",project_id="9fadcee8aa7c40cdb2114fff7d569c08",subnet_pool_id="f49a1319-423a-4ee6-ba54-1d95a4f6cc68",subnet_pool_name="my-subnet-pool-ipv4"} 7
openstack_neutron_subnets_free{ip_version="4",prefix="10.10.0.0/21",prefix_length="25",project_id="9fadcee8aa7c40cdb2114fff7d569c08",subnet_pool_id="f49a1319-423a-4ee6-ba54-1d95a4f6cc68",subnet_pool_name="my-subnet-pool-ipv4"} 14
openstack_neutron_subnets_free{ip_version="4",prefix="10.10.0.0/21",prefix_length="26",project_id="9fadcee8aa7c40cdb2114fff7d569c08",subnet_pool_id="f49a1319-423a-4ee6-ba54-1d95a4f6cc68",subnet_pool_name="my-subnet-pool-ipv4"} 28
openstack_neutron_subnets_free{ip_version="6",prefix="2001:db8::/63",prefix_length="63",project_id="9fadcee8aa7c40cdb2114fff7d569c08",subnet_pool_id="03f761e6-eee0-43fc-a921-8acf64c14988",subnet_pool_name="my-subnet-pool-ipv6"} 0
openstack_neutron_subnets_free{ip_version="6",prefix="2001:db8::/63",prefix_length="64",project_id="9fadcee8aa7c40cdb2114fff7d569c08",subnet_pool_id="03f761e6-eee0-43fc-a921-8acf64c14988",subnet_pool_name="my-subnet-pool-ipv6"} 1
openstack_neutron_subnets_free{ip_version="6",prefix="2001:db8::/63",prefix_length="65",project_id="9fadcee8aa7c40cdb2114fff7d569c08",subnet_pool_id="03f761e6-eee0-43fc-a921-8acf64c14988",subnet_pool_name="my-subnet-pool-ipv6"} 2
# HELP openstack_neutron_subnets_total subnets_total
# TYPE openstack_neutron_subnets_total gauge
openstack_neutron_subnets_total{ip_version="4",prefix="10.10.0.0/21",prefix_length="24",project_id="9fadcee8aa7c40cdb2114fff7d569c08",subnet_pool_id="f49a1319-423a-4ee6-ba54-1d95a4f6cc68",subnet_pool_name="my-subnet-pool-ipv4"} 8
openstack_neutron_subnets_total{ip_version="4",prefix="10.10.0.0/21",prefix_length="25",project_id="9fadcee8aa7c40cdb2114fff7d569c08",subnet_pool_id="f49a1319-423a-4ee6-ba54-1d95a4f6cc68",subnet_pool_name="my-subnet-pool-ipv4"} 16
openstack_neutron_subnets_total{ip_version="4",prefix="10.10.0.0/21",prefix_length="26",project_id="9fadcee8aa7c40cdb2114fff7d569c08",subnet_pool_id="f49a1319-423a-4ee6-ba54-1d95a4f6cc68",subnet_pool_name="my-subnet-pool-ipv4"} 32
openstack_neutron_subnets_total{ip_version="6",prefix="2001:db8::/63",prefix_length="63",project_id="9fadcee8aa7c40cdb2114fff7d569c08",subnet_pool_id="03f761e6-eee0-43fc-a921-8acf64c14988",subnet_pool_name="my-subnet-pool-ipv6"} 1
openstack_neutron_subnets_total{ip_version="6",prefix="2001:db8::/63",prefix_length="64",project_id="9fadcee8aa7c40cdb2114fff7d569c08",subnet_pool_id="03f761e6-eee0-43fc-a921-8acf64c14988",subnet_pool_name="my-subnet-pool-ipv6"} 2
openstack_neutron_subnets_total{ip_version="6",prefix="2001:db8::/63",prefix_length="65",project_id="9fadcee8aa7c40cdb2114fff7d569c08",subnet_pool_id="03f761e6-eee0-43fc-a921-8acf64c14988",subnet_pool_name="my-subnet-pool-ipv6"} 4
# HELP openstack_neutron_subnets_used subnets_used
# TYPE openstack_neutron_subnets_used gauge
openstack_neutron_subnets_used{ip_version="4",prefix="10.10.0.0/21",prefix_length="24",project_id="9fadcee8aa7c40cdb2114fff7d569c08",subnet_pool_id="f49a1319-423a-4ee6-ba54-1d95a4f6cc68",subnet_pool_name="my-subnet-pool-ipv4"} 1
openstack_neutron_subnets_used{ip_version="4",prefix="10.10.0.0/21",prefix_length="25",project_id="9fadcee8aa7c40cdb2114fff7d569c08",subnet_pool_id="f49a1319-423a-4ee6-ba54-1d95a4f6cc68",subnet_pool_name="my-subnet-pool-ipv4"} 0
openstack_neutron_subnets_used{ip_version="4",prefix="10.10.0.0/21",prefix_length="26",project_id="9fadcee8aa7c40cdb2114fff7d569c08",subnet_pool_id="f49a1319-423a-4ee6-ba54-1d95a4f6cc68",subnet_pool_name="my-subnet-pool-ipv4"} 0
openstack_neutron_subnets_used{ip_version="6",prefix="2001:db8::/63",prefix_length="63",project_id="9fadcee8aa7c40cdb2114fff7d569c08",subnet_pool_id="03f761e6-eee0-43fc-a921-8acf64c14988",subnet_pool_name="my-subnet-pool-ipv6"} 0
openstack_neutron_subnets_used{ip_version="6",prefix="2001:db8::/63",prefix_length="64",project_id="9fadcee8aa7c40cdb2114fff7d569c08",subnet_pool_id="03f761e6-eee0-43fc-a921-8acf64c14988",subnet_pool_name="my-subnet-pool-ipv6"} 1
openstack_neutron_subnets_used{ip_version="6",prefix="2001:db8::/63",prefix_length="65",project_id="9fadcee8aa7c40cdb2114fff7d569c08",subnet_pool_id="03f761e6-eee0-43fc-a921-8acf64c14988",subnet_pool_name="my-subnet-pool-ipv6"} 0
# HELP openstack_neutron_up up
# TYPE openstack_neutron_up gauge
openstack_neutron_up 1
`

func (suite *NeutronTestSuite) TestNeutronExporter() {
	err := testutil.CollectAndCompare(*suite.Exporter, strings.NewReader(neutronExpectedUp))
	assert.NoError(suite.T(), err)
}
