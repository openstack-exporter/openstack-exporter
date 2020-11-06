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
openstack_ironic_node{console_enabled="false",id="f50dcc35-4913-4667-a9fa-d130659c5661",maintenance="false",name="r1-02",power_state="power off",provision_state="available",resource_class="baremetal"} 1
openstack_ironic_node{console_enabled="true",id="0129d2fc-0e5c-4b5b-a73b-01844d913957",maintenance="false",name="r1-04",power_state="power on",provision_state="active",resource_class="baremetal"} 1
openstack_ironic_node{console_enabled="true",id="c9f98cc9-25e9-424e-8a89-002989054ec2",maintenance="true",name="r1-05",power_state="power off",provision_state="available",resource_class="baremetal"} 1
openstack_ironic_node{console_enabled="true",id="d381bea3-8768-4f12-a9b3-abf750ba918f",maintenance="false",name="r1-03",power_state="power on",provision_state="active",resource_class="baremetal"} 1
openstack_ironic_node{console_enabled="true",id="d5641882-f7e5-4b92-9423-7e8157586218",maintenance="true",name="r1-01",power_state="power off",provision_state="error",resource_class="baremetal"} 1
# HELP openstack_ironic_up up
# TYPE openstack_ironic_up gauge
openstack_ironic_up 1
`

var ironicExpectedDown = `
# HELP openstack_ironic_up up
# TYPE openstack_ironic_up gauge
openstack_ironic_up 0
`

func (suite *IronicTestSuite) TestIronicExporter() {
	err := testutil.CollectAndCompare(*suite.Exporter, strings.NewReader(ironicExpectedUp))
	assert.NoError(suite.T(), err)
}

func (suite *IronicTestSuite) TestIronicExporterWithEndpointDown() {
	suite.teardownFixtures()
	defer suite.installFixtures()

	err := testutil.CollectAndCompare(*suite.Exporter, strings.NewReader(ironicExpectedDown))
	assert.NoError(suite.T(), err)
}
