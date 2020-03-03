package exporters

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

type NovaTestSuite struct {
	BaseOpenStackTestSuite
}

var novaExpectedUp = `
# HELP openstack_nova_agent_state agent_state
# TYPE openstack_nova_agent_state counter
openstack_nova_agent_state{adminState="disabled",disabledReason="test1",hostname="host1",id="1",service="nova-scheduler",zone="internal"} 1
openstack_nova_agent_state{adminState="disabled",disabledReason="test2",hostname="host1",id="2",service="nova-compute",zone="nova"} 1
openstack_nova_agent_state{adminState="disabled",disabledReason="test4",hostname="host2",id="4",service="nova-compute",zone="nova"} 0
openstack_nova_agent_state{adminState="enabled",disabledReason="",hostname="host2",id="3",service="nova-scheduler",zone="internal"} 0
# HELP openstack_nova_availability_zones availability_zones
# TYPE openstack_nova_availability_zones gauge
openstack_nova_availability_zones 1
# HELP openstack_nova_current_workload current_workload
# TYPE openstack_nova_current_workload gauge
openstack_nova_current_workload{aggregates="",availability_zone="",hostname="host1"} 0
# HELP openstack_nova_flavors flavors
# TYPE openstack_nova_flavors gauge
openstack_nova_flavors 7
# HELP openstack_nova_limits_memory_max limits_memory_max
# TYPE openstack_nova_limits_memory_max gauge
openstack_nova_limits_memory_max{tenant="admin"} 51200
openstack_nova_limits_memory_max{tenant="alt_demo"} 51200
openstack_nova_limits_memory_max{tenant="demo"} 51200
openstack_nova_limits_memory_max{tenant="invisible_to_admin"} 51200
openstack_nova_limits_memory_max{tenant="service"} 51200
openstack_nova_limits_memory_max{tenant="swifttenanttest1"} 51200
openstack_nova_limits_memory_max{tenant="swifttenanttest2"} 51200
openstack_nova_limits_memory_max{tenant="swifttenanttest4"} 51200
# HELP openstack_nova_limits_memory_used limits_memory_used
# TYPE openstack_nova_limits_memory_used gauge
openstack_nova_limits_memory_used{tenant="admin"} 0
openstack_nova_limits_memory_used{tenant="alt_demo"} 0
openstack_nova_limits_memory_used{tenant="demo"} 0
openstack_nova_limits_memory_used{tenant="invisible_to_admin"} 0
openstack_nova_limits_memory_used{tenant="service"} 0
openstack_nova_limits_memory_used{tenant="swifttenanttest1"} 0
openstack_nova_limits_memory_used{tenant="swifttenanttest2"} 0
openstack_nova_limits_memory_used{tenant="swifttenanttest4"} 0
# HELP openstack_nova_limits_vcpus_max limits_vcpus_max
# TYPE openstack_nova_limits_vcpus_max gauge
openstack_nova_limits_vcpus_max{tenant="admin"} 20
openstack_nova_limits_vcpus_max{tenant="alt_demo"} 20
openstack_nova_limits_vcpus_max{tenant="demo"} 20
openstack_nova_limits_vcpus_max{tenant="invisible_to_admin"} 20
openstack_nova_limits_vcpus_max{tenant="service"} 20
openstack_nova_limits_vcpus_max{tenant="swifttenanttest1"} 20
openstack_nova_limits_vcpus_max{tenant="swifttenanttest2"} 20
openstack_nova_limits_vcpus_max{tenant="swifttenanttest4"} 20
# HELP openstack_nova_limits_vcpus_used limits_vcpus_used
# TYPE openstack_nova_limits_vcpus_used gauge
openstack_nova_limits_vcpus_used{tenant="admin"} 0
openstack_nova_limits_vcpus_used{tenant="alt_demo"} 0
openstack_nova_limits_vcpus_used{tenant="demo"} 0
openstack_nova_limits_vcpus_used{tenant="invisible_to_admin"} 0
openstack_nova_limits_vcpus_used{tenant="service"} 0
openstack_nova_limits_vcpus_used{tenant="swifttenanttest1"} 0
openstack_nova_limits_vcpus_used{tenant="swifttenanttest2"} 0
openstack_nova_limits_vcpus_used{tenant="swifttenanttest4"} 0
# HELP openstack_nova_local_storage_available_bytes local_storage_available_bytes
# TYPE openstack_nova_local_storage_available_bytes gauge
openstack_nova_local_storage_available_bytes{aggregates="",availability_zone="",hostname="host1"} 1.103806595072e+12
# HELP openstack_nova_local_storage_used_bytes local_storage_used_bytes
# TYPE openstack_nova_local_storage_used_bytes gauge
openstack_nova_local_storage_used_bytes{aggregates="",availability_zone="",hostname="host1"} 0
# HELP openstack_nova_memory_available_bytes memory_available_bytes
# TYPE openstack_nova_memory_available_bytes gauge
openstack_nova_memory_available_bytes{aggregates="",availability_zone="",hostname="host1"} 8.589934592e+09
# HELP openstack_nova_memory_used_bytes memory_used_bytes
# TYPE openstack_nova_memory_used_bytes gauge
openstack_nova_memory_used_bytes{aggregates="",availability_zone="",hostname="host1"} 5.36870912e+08
# HELP openstack_nova_running_vms running_vms
# TYPE openstack_nova_running_vms gauge
openstack_nova_running_vms{aggregates="",availability_zone="",hostname="host1"} 0
# HELP openstack_nova_security_groups security_groups
# TYPE openstack_nova_security_groups gauge
openstack_nova_security_groups 1
# HELP openstack_nova_server_status server_status
# TYPE openstack_nova_server_status gauge
openstack_nova_server_status{address_ipv4="1.2.3.4",address_ipv6="80fe::",availability_zone="nova",flavor_id="<nil>",host_id="2091634baaccdc4c5a1d57069c833e402921df696b7f970791b12ec6",id="2ce4c5b3-2866-4972-93ce-77a2ea46a7f9",name="new-server-test",status="ACTIVE",tenant_id="6f70656e737461636b20342065766572",user_id="fake",uuid="2ce4c5b3-2866-4972-93ce-77a2ea46a7f9"} 0
# HELP openstack_nova_total_vms total_vms
# TYPE openstack_nova_total_vms gauge
openstack_nova_total_vms 1
# HELP openstack_nova_up up
# TYPE openstack_nova_up gauge
openstack_nova_up 1
# HELP openstack_nova_vcpus_available vcpus_available
# TYPE openstack_nova_vcpus_available gauge
openstack_nova_vcpus_available{aggregates="",availability_zone="",hostname="host1"} 2
# HELP openstack_nova_vcpus_used vcpus_used
# TYPE openstack_nova_vcpus_used gauge
openstack_nova_vcpus_used{aggregates="",availability_zone="",hostname="host1"} 0
`

var novaExpectedDown = `
# HELP openstack_nova_up up
# TYPE openstack_nova_up gauge
openstack_nova_up 0
`

func (suite *NovaTestSuite) TestNovaExporter() {
	err := testutil.CollectAndCompare(*suite.Exporter, strings.NewReader(novaExpectedUp))
	assert.NoError(suite.T(), err)
}

func (suite *NovaTestSuite) TestNovaExporterWithEndpointDown() {
	suite.teardownFixtures()
	defer suite.installFixtures()

	err := testutil.CollectAndCompare(*suite.Exporter, strings.NewReader(novaExpectedDown))
	assert.NoError(suite.T(), err)
}
