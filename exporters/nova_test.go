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
# HELP openstack_nova_flavor flavor
# TYPE openstack_nova_flavor gauge
openstack_nova_flavor{disk="0",id="1",is_public="true",name="m1.tiny",ram="512",vcpus="1"} 1
openstack_nova_flavor{disk="0",id="2",is_public="true",name="m1.small",ram="2048",vcpus="1"} 1
openstack_nova_flavor{disk="0",id="3",is_public="true",name="m1.medium",ram="4096",vcpus="2"} 1
openstack_nova_flavor{disk="0",id="4",is_public="true",name="m1.large",ram="8192",vcpus="4"} 1
openstack_nova_flavor{disk="0",id="5",is_public="true",name="m1.xlarge",ram="16384",vcpus="8"} 1
openstack_nova_flavor{disk="0",id="6",is_public="true",name="m1.tiny.specs",ram="512",vcpus="1"} 1
openstack_nova_flavor{disk="0",id="7",is_public="true",name="m1.small.description",ram="2048",vcpus="1"} 1
openstack_nova_flavor{disk="0",id="8",is_public="false",name="m1.tiny.private",ram="512",vcpus="1"} 1
# HELP openstack_nova_flavors flavors
# TYPE openstack_nova_flavors gauge
openstack_nova_flavors 8
# HELP openstack_nova_free_disk_bytes free_disk_bytes
# TYPE openstack_nova_free_disk_bytes gauge
openstack_nova_free_disk_bytes{aggregates="",availability_zone="",hostname="host1"} 1.103806595072e+12
# HELP openstack_nova_limits_instances_max limits_instances_max
# TYPE openstack_nova_limits_instances_max gauge
openstack_nova_limits_instances_max{tenant="admin",tenant_id="0c4e939acacf4376bdcd1129f1a054ad"} 10
openstack_nova_limits_instances_max{tenant="alt_demo",tenant_id="fdb8424c4e4f4c0ba32c52e2de3bd80e"} 10
openstack_nova_limits_instances_max{tenant="demo",tenant_id="0cbd49cbf76d405d9c86562e1d579bd3"} 10
openstack_nova_limits_instances_max{tenant="invisible_to_admin",tenant_id="5961c443439d4fcebe42643723755e9d"} 10
openstack_nova_limits_instances_max{tenant="service",tenant_id="3d594eb0f04741069dbbb521635b21c7"} 10
openstack_nova_limits_instances_max{tenant="swifttenanttest1",tenant_id="43ebde53fc314b1c9ea2b8c5dc744927"} 10
openstack_nova_limits_instances_max{tenant="swifttenanttest2",tenant_id="2db68fed84324f29bb73130c6c2094fb"} 10
openstack_nova_limits_instances_max{tenant="swifttenanttest4",tenant_id="4b1eb781a47440acb8af9850103e537f"} 10
# HELP openstack_nova_limits_instances_used limits_instances_used
# TYPE openstack_nova_limits_instances_used gauge
openstack_nova_limits_instances_used{tenant="admin",tenant_id="0c4e939acacf4376bdcd1129f1a054ad"} 0
openstack_nova_limits_instances_used{tenant="alt_demo",tenant_id="fdb8424c4e4f4c0ba32c52e2de3bd80e"} 0
openstack_nova_limits_instances_used{tenant="demo",tenant_id="0cbd49cbf76d405d9c86562e1d579bd3"} 0
openstack_nova_limits_instances_used{tenant="invisible_to_admin",tenant_id="5961c443439d4fcebe42643723755e9d"} 0
openstack_nova_limits_instances_used{tenant="service",tenant_id="3d594eb0f04741069dbbb521635b21c7"} 0
openstack_nova_limits_instances_used{tenant="swifttenanttest1",tenant_id="43ebde53fc314b1c9ea2b8c5dc744927"} 0
openstack_nova_limits_instances_used{tenant="swifttenanttest2",tenant_id="2db68fed84324f29bb73130c6c2094fb"} 0
openstack_nova_limits_instances_used{tenant="swifttenanttest4",tenant_id="4b1eb781a47440acb8af9850103e537f"} 0
# HELP openstack_nova_limits_memory_max limits_memory_max
# TYPE openstack_nova_limits_memory_max gauge
openstack_nova_limits_memory_max{tenant="admin",tenant_id="0c4e939acacf4376bdcd1129f1a054ad"} 51200
openstack_nova_limits_memory_max{tenant="alt_demo",tenant_id="fdb8424c4e4f4c0ba32c52e2de3bd80e"} 51200
openstack_nova_limits_memory_max{tenant="demo",tenant_id="0cbd49cbf76d405d9c86562e1d579bd3"} 51200
openstack_nova_limits_memory_max{tenant="invisible_to_admin",tenant_id="5961c443439d4fcebe42643723755e9d"} 51200
openstack_nova_limits_memory_max{tenant="service",tenant_id="3d594eb0f04741069dbbb521635b21c7"} 51200
openstack_nova_limits_memory_max{tenant="swifttenanttest1",tenant_id="43ebde53fc314b1c9ea2b8c5dc744927"} 51200
openstack_nova_limits_memory_max{tenant="swifttenanttest2",tenant_id="2db68fed84324f29bb73130c6c2094fb"} 51200
openstack_nova_limits_memory_max{tenant="swifttenanttest4",tenant_id="4b1eb781a47440acb8af9850103e537f"} 51200
# HELP openstack_nova_limits_memory_used limits_memory_used
# TYPE openstack_nova_limits_memory_used gauge
openstack_nova_limits_memory_used{tenant="admin",tenant_id="0c4e939acacf4376bdcd1129f1a054ad"} 0
openstack_nova_limits_memory_used{tenant="alt_demo",tenant_id="fdb8424c4e4f4c0ba32c52e2de3bd80e"} 0
openstack_nova_limits_memory_used{tenant="demo",tenant_id="0cbd49cbf76d405d9c86562e1d579bd3"} 0
openstack_nova_limits_memory_used{tenant="invisible_to_admin",tenant_id="5961c443439d4fcebe42643723755e9d"} 0
openstack_nova_limits_memory_used{tenant="service",tenant_id="3d594eb0f04741069dbbb521635b21c7"} 0
openstack_nova_limits_memory_used{tenant="swifttenanttest1",tenant_id="43ebde53fc314b1c9ea2b8c5dc744927"} 0
openstack_nova_limits_memory_used{tenant="swifttenanttest2",tenant_id="2db68fed84324f29bb73130c6c2094fb"} 0
openstack_nova_limits_memory_used{tenant="swifttenanttest4",tenant_id="4b1eb781a47440acb8af9850103e537f"} 0
# HELP openstack_nova_limits_vcpus_max limits_vcpus_max
# TYPE openstack_nova_limits_vcpus_max gauge
openstack_nova_limits_vcpus_max{tenant="admin",tenant_id="0c4e939acacf4376bdcd1129f1a054ad"} 20
openstack_nova_limits_vcpus_max{tenant="alt_demo",tenant_id="fdb8424c4e4f4c0ba32c52e2de3bd80e"} 20
openstack_nova_limits_vcpus_max{tenant="demo",tenant_id="0cbd49cbf76d405d9c86562e1d579bd3"} 20
openstack_nova_limits_vcpus_max{tenant="invisible_to_admin",tenant_id="5961c443439d4fcebe42643723755e9d"} 20
openstack_nova_limits_vcpus_max{tenant="service",tenant_id="3d594eb0f04741069dbbb521635b21c7"} 20
openstack_nova_limits_vcpus_max{tenant="swifttenanttest1",tenant_id="43ebde53fc314b1c9ea2b8c5dc744927"} 20
openstack_nova_limits_vcpus_max{tenant="swifttenanttest2",tenant_id="2db68fed84324f29bb73130c6c2094fb"} 20
openstack_nova_limits_vcpus_max{tenant="swifttenanttest4",tenant_id="4b1eb781a47440acb8af9850103e537f"} 20
# HELP openstack_nova_limits_vcpus_used limits_vcpus_used
# TYPE openstack_nova_limits_vcpus_used gauge
openstack_nova_limits_vcpus_used{tenant="admin",tenant_id="0c4e939acacf4376bdcd1129f1a054ad"} 0
openstack_nova_limits_vcpus_used{tenant="alt_demo",tenant_id="fdb8424c4e4f4c0ba32c52e2de3bd80e"} 0
openstack_nova_limits_vcpus_used{tenant="demo",tenant_id="0cbd49cbf76d405d9c86562e1d579bd3"} 0
openstack_nova_limits_vcpus_used{tenant="invisible_to_admin",tenant_id="5961c443439d4fcebe42643723755e9d"} 0
openstack_nova_limits_vcpus_used{tenant="service",tenant_id="3d594eb0f04741069dbbb521635b21c7"} 0
openstack_nova_limits_vcpus_used{tenant="swifttenanttest1",tenant_id="43ebde53fc314b1c9ea2b8c5dc744927"} 0
openstack_nova_limits_vcpus_used{tenant="swifttenanttest2",tenant_id="2db68fed84324f29bb73130c6c2094fb"} 0
openstack_nova_limits_vcpus_used{tenant="swifttenanttest4",tenant_id="4b1eb781a47440acb8af9850103e537f"} 0
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
# HELP openstack_nova_server_local_gb server_local_gb
# TYPE openstack_nova_server_local_gb gauge
openstack_nova_server_local_gb{id="27bb2854-b06a-48f5-ab4e-139817b8b8ff",name="openstack-monitoring-0",tenant_id="110f6313d2d346b4aa90eabe4970b62a"} 10
openstack_nova_server_local_gb{id="2dbdf831-4ffa-485b-8020-216655fb5c7d",name="openstack-monitoring-3",tenant_id="110f6313d2d346b4aa90eabe4970b62a"} 10
openstack_nova_server_local_gb{id="6c773231-6532-447d-b651-9e0d1518b31d",name="openstack-monitoring-1",tenant_id="110f6313d2d346b4aa90eabe4970b62a"} 10
openstack_nova_server_local_gb{id="f99bb4a3-90ff-46fa-b8ec-2ef6ac1f3b7d",name="openstack-monitoring-2-prod-zone",tenant_id="110f6313d2d346b4aa90eabe4970b62a"} 10
# HELP openstack_nova_server_status server_status
# TYPE openstack_nova_server_status gauge
openstack_nova_server_status{My_Server_Name="Apache1",address_ipv4="1.2.3.4",address_ipv6="80fe::",availability_zone="nova",flavor_id="1",host_id="2091634baaccdc4c5a1d57069c833e402921df696b7f970791b12ec6",hypervisor_hostname="fake-mini",id="2ce4c5b3-2866-4972-93ce-77a2ea46a7f9",instance_libvirt="instance-00000001",name="new-server-test",status="ACTIVE",tenant_id="6f70656e737461636b20342065766572",user_id="fake",uuid="2ce4c5b3-2866-4972-93ce-77a2ea46a7f9"} 0
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

func (suite *NovaTestSuite) TestNovaExporter() {
	err := testutil.CollectAndCompare(*suite.Exporter, strings.NewReader(novaExpectedUp))
	assert.NoError(suite.T(), err)
}
