package exporters

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

type CinderTestSuite struct {
	BaseOpenStackTestSuite
}

var cinderExpectedUp = `
# HELP openstack_cinder_agent_state agent_state
# TYPE openstack_cinder_agent_state counter
openstack_cinder_agent_state{adminState="enabled",disabledReason="",hostname="devstack@lvmdriver-1",service="cinder-volume",zone="nova"} 1
openstack_cinder_agent_state{adminState="enabled",disabledReason="Test1",hostname="devstack",service="cinder-scheduler",zone="nova"} 1
openstack_cinder_agent_state{adminState="enabled",disabledReason="Test2",hostname="devstack",service="cinder-backup",zone="nova"} 1
# HELP openstack_cinder_snapshots snapshots
# TYPE openstack_cinder_snapshots gauge
openstack_cinder_snapshots 1
# HELP openstack_cinder_up up
# TYPE openstack_cinder_up gauge
openstack_cinder_up 1
# HELP openstack_cinder_volume_status volume_status
# TYPE openstack_cinder_volume_status gauge
openstack_cinder_volume_status{bootable="false",id="6edbc2f4-1507-44f8-ac0d-eed1d2608d38",name="test-volume-attachments",size="2",status="in-use",tenant_id="bab7d5c60cd041a0a36f7c4b6e1dd978",volume_type="lvmdriver-1"} 5
openstack_cinder_volume_status{bootable="true",id="173f7b48-c4c1-4e70-9acc-086b39073506",name="test-volume",size="1",status="available",tenant_id="bab7d5c60cd041a0a36f7c4b6e1dd978",volume_type="lvmdriver-1"} 1
# HELP openstack_cinder_volumes volumes
# TYPE openstack_cinder_volumes gauge
openstack_cinder_volumes 2
`

var cinderExpectedDown = `
# HELP openstack_cinder_up up
# TYPE openstack_cinder_up gauge
openstack_cinder_up 0
`

func (suite *CinderTestSuite) TestCinderExporter() {
	err := testutil.CollectAndCompare(*suite.Exporter, strings.NewReader(cinderExpectedUp))
	assert.NoError(suite.T(), err)
}

func (suite *CinderTestSuite) TestCinderExporterWithEndpointDown() {
	suite.teardownFixtures()
	defer suite.installFixtures()

	err := testutil.CollectAndCompare(*suite.Exporter, strings.NewReader(cinderExpectedDown))
	assert.NoError(suite.T(), err)
}
