package exporters

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

type IronicTestSuite struct {
	BaseOpenStackTestSuite
}

var ironicExpectedUp = `
# HELP openstack_ironic_node node
# TYPE openstack_ironic_node gauge
openstack_ironic_node{console_enabled="false",deploy_kernel="7ff5ef56-daaa-4256-9dd8-c3f1f9964ebc",deploy_ramdisk="e9c96d45-a4c8-4165-8753-9d8f32779e99",id="f50dcc35-4913-4667-a9fa-d130659c5661",maintenance="false",maintenance_reason="",name="r1-02",power_state="power off",provision_state="available",resource_class="baremetal",retired="true",retired_reason="No longer needed"} 1
openstack_ironic_node{console_enabled="true",deploy_kernel="7ff5ef56-daaa-4256-9dd8-c3f1f9964ebc",deploy_ramdisk="e9c96d45-a4c8-4165-8753-9d8f32779e99",id="0129d2fc-0e5c-4b5b-a73b-01844d913957",maintenance="false",maintenance_reason="",name="r1-04",power_state="power on",provision_state="active",resource_class="baremetal",retired="true",retired_reason="No longer needed"} 1
openstack_ironic_node{console_enabled="true",deploy_kernel="7ff5ef56-daaa-4256-9dd8-c3f1f9964ebc",deploy_ramdisk="e9c96d45-a4c8-4165-8753-9d8f32779e99",id="c9f98cc9-25e9-424e-8a89-002989054ec2",maintenance="true",maintenance_reason="Firmware upgrade",name="r1-05",power_state="power off",provision_state="available",resource_class="baremetal",retired="true",retired_reason="No longer needed"} 1
openstack_ironic_node{console_enabled="true",deploy_kernel="7ff5ef56-daaa-4256-9dd8-c3f1f9964ebc",deploy_ramdisk="e9c96d45-a4c8-4165-8753-9d8f32779e99",id="d381bea3-8768-4f12-a9b3-abf750ba918f",maintenance="false",maintenance_reason="",name="r1-03",power_state="power on",provision_state="active",resource_class="baremetal",retired="true",retired_reason="No longer needed"} 1
openstack_ironic_node{console_enabled="true",deploy_kernel="7ff5ef56-daaa-4256-9dd8-c3f1f9964ebc",deploy_ramdisk="e9c96d45-a4c8-4165-8753-9d8f32779e99",id="d5641882-f7e5-4b92-9423-7e8157586218",maintenance="true",maintenance_reason="",name="r1-01",power_state="power off",provision_state="error",resource_class="baremetal",retired="true",retired_reason="No longer needed"} 1
# HELP openstack_ironic_node_provision_updated_at node_provision_updated_at
# TYPE openstack_ironic_node_provision_updated_at gauge
openstack_ironic_node_provision_updated_at{id="0129d2fc-0e5c-4b5b-a73b-01844d913957",name="r1-04",provision_state="active"} 1.593544011e+09
openstack_ironic_node_provision_updated_at{id="c9f98cc9-25e9-424e-8a89-002989054ec2",name="r1-05",provision_state="available"} 1.562908443e+09
openstack_ironic_node_provision_updated_at{id="d381bea3-8768-4f12-a9b3-abf750ba918f",name="r1-03",provision_state="active"} 1.593747281e+09
openstack_ironic_node_provision_updated_at{id="d5641882-f7e5-4b92-9423-7e8157586218",name="r1-01",provision_state="error"} 1.594708597e+09
openstack_ironic_node_provision_updated_at{id="f50dcc35-4913-4667-a9fa-d130659c5661",name="r1-02",provision_state="available"} 1.594740492e+09
# HELP openstack_ironic_node_updated_at node_updated_at
# TYPE openstack_ironic_node_updated_at gauge
openstack_ironic_node_updated_at{id="0129d2fc-0e5c-4b5b-a73b-01844d913957",name="r1-04",provision_state="active"} 1.593544011e+09
openstack_ironic_node_updated_at{id="c9f98cc9-25e9-424e-8a89-002989054ec2",name="r1-05",provision_state="available"} 1.592845911e+09
openstack_ironic_node_updated_at{id="d381bea3-8768-4f12-a9b3-abf750ba918f",name="r1-03",provision_state="active"} 1.594162438e+09
openstack_ironic_node_updated_at{id="d5641882-f7e5-4b92-9423-7e8157586218",name="r1-01",provision_state="error"} 1.594708598e+09
openstack_ironic_node_updated_at{id="f50dcc35-4913-4667-a9fa-d130659c5661",name="r1-02",provision_state="available"} 1.594740494e+09
# HELP openstack_ironic_up up
# TYPE openstack_ironic_up gauge
openstack_ironic_up 1
`

func (suite *IronicTestSuite) TestIronicExporter() {
	err := testutil.CollectAndCompare(*suite.Exporter, strings.NewReader(ironicExpectedUp))
	assert.NoError(suite.T(), err)
}
