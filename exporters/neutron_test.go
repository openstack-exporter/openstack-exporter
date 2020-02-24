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
openstack_neutron_agent_state{adminState="up",hostname="agenthost1",service="neutron-dhcp-agent"} 1
openstack_neutron_agent_state{adminState="up",hostname="agenthost1",service="neutron-l3-agent"} 1
openstack_neutron_agent_state{adminState="up",hostname="agenthost1",service="neutron-lbaasv2-agent"} 1
openstack_neutron_agent_state{adminState="up",hostname="agenthost1",service="neutron-metadata-agent"} 1
openstack_neutron_agent_state{adminState="up",hostname="agenthost1",service="neutron-openvswitch-agent"} 1
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
