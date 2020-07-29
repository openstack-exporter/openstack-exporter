package exporters

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
openstack_neutron_agent_state{adminState="up",hostname="agenthost1",id="840d5d68-5759-4e9e-812f-f3bd19214c7f",service="neutron-dhcp-agent"} 1
openstack_neutron_agent_state{adminState="up",hostname="agenthost1",id="a09b81fc-5a42-46d3-a306-1a5d122a7787",service="neutron-l3-agent"} 1
openstack_neutron_agent_state{adminState="up",hostname="agenthost1",id="2bf84eaf-d869-49cc-8401-cbbca5177e59",service="neutron-lbaasv2-agent"} 1
openstack_neutron_agent_state{adminState="up",hostname="agenthost1",id="c876c9f7-1058-4b9b-90ed-20fb3f905ec4",service="neutron-metadata-agent"} 1
openstack_neutron_agent_state{adminState="up",hostname="agenthost1",id="04c62b91-b799-48b7-9cd5-2982db6df9c6",service="neutron-openvswitch-agent"} 1
# HELP openstack_neutron_floating_ips floating_ips
# TYPE openstack_neutron_floating_ips gauge
openstack_neutron_floating_ips 4
# HELP openstack_neutron_floating_ips_associated_not_active floating_ips_associated_not_active
# TYPE openstack_neutron_floating_ips_associated_not_active gauge
openstack_neutron_floating_ips_associated_not_active 1
# HELP openstack_neutron_loadbalancers loadbalancers
# TYPE openstack_neutron_loadbalancers gauge
openstack_neutron_loadbalancers 2
# HELP openstack_neutron_loadbalancers_not_active loadbalancers_not_active
# TYPE openstack_neutron_loadbalancers_not_active gauge
openstack_neutron_loadbalancers_not_active 0
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
openstack_neutron_networks 0
# HELP openstack_neutron_port port
# TYPE openstack_neutron_port gauge
openstack_neutron_port{binding_vif_type="",device_owner="network:router_gateway",mac_address="fa:16:3e:58:42:ed",network_id="70c1db1f-b701-45bd-96e0-a313ee3430b3",status="ACTIVE",uuid="d80b1a3b-4fc1-49f3-952e-1e2ab7081d8b"} 1
openstack_neutron_port{binding_vif_type="",device_owner="network:router_interface",mac_address="fa:16:3e:bb:3c:e4",network_id="f27aa545-cbdd-4907-b0c6-c9e8b039dcc2",status="ACTIVE",uuid="f71a6703-d6de-4be1-a91a-a570ede1d159"} 1
openstack_neutron_port{binding_vif_type="ovs",device_owner="neutron:LOADBALANCERV2",mac_address="fa:16:3e:0b:14:fd",network_id="675c54a5-a9f3-4f5e-a0b4-e026b29c217b",status="N/A",uuid="f0b24508-eb48-4530-a38b-c042df147101"} 1
# HELP openstack_neutron_ports ports
# TYPE openstack_neutron_ports gauge
openstack_neutron_ports 3
# HELP openstack_neutron_ports_no_ips ports_no_ips
# TYPE openstack_neutron_ports_no_ips gauge
openstack_neutron_ports_no_ips 1
# HELP openstack_neutron_ports_lb_not_active ports_lb_not_active
# TYPE openstack_neutron_ports_lb_not_active gauge
openstack_neutron_ports_lb_not_active 1
# HELP openstack_neutron_routers routers
# TYPE openstack_neutron_routers gauge
openstack_neutron_routers 0
# HELP openstack_neutron_routers_not_active routers_not_active
# TYPE openstack_neutron_routers_not_active gauge
openstack_neutron_routers_not_active 0
# HELP openstack_neutron_security_groups security_groups
# TYPE openstack_neutron_security_groups gauge
openstack_neutron_security_groups 1
# HELP openstack_neutron_subnets subnets
# TYPE openstack_neutron_subnets gauge
openstack_neutron_subnets 2
# HELP openstack_neutron_up up
# TYPE openstack_neutron_up gauge
openstack_neutron_up 1
`

var neutronExpectedDown = `
# HELP openstack_neutron_up up
# TYPE openstack_neutron_up gauge
openstack_neutron_up 0
`

func (suite *NeutronTestSuite) TestNeutronExporter() {
	err := testutil.CollectAndCompare(*suite.Exporter, strings.NewReader(neutronExpectedUp))
	assert.NoError(suite.T(), err)
}

func (suite *NeutronTestSuite) TestNeutronExporterWithEndpointDown() {
	suite.teardownFixtures()
	defer suite.installFixtures()

	err := testutil.CollectAndCompare(*suite.Exporter, strings.NewReader(neutronExpectedDown))
	assert.NoError(suite.T(), err)
}
