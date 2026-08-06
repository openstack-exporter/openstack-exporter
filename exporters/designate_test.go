package exporters

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

type DesignateTestSuite struct {
	BaseOpenStackTestSuite
}

var designateExpectedUp = `
# HELP openstack_designate_recordsets recordsets
# TYPE openstack_designate_recordsets gauge
openstack_designate_recordsets{tenant_id="4335d1f0-f793-11e2-b778-0800200c9a66",zone_id="a86dba58-0043-4cc6-a1bb-69d5e86f3ca3",zone_name="example.org."} 0
# HELP openstack_designate_recordsets_status recordsets_status
# TYPE openstack_designate_recordsets_status gauge
openstack_designate_recordsets_status{id="f7b10e9b-0cae-4a91-b162-562bc6096648",name="example.org.",status="PENDING",type="A",zone_id="2150b1bf-dee2-4221-9d85-11f7886fb15f",zone_name="example.com."} 0
# HELP openstack_designate_up up
# TYPE openstack_designate_up gauge
openstack_designate_up 1
# HELP openstack_designate_zone_status zone_status
# TYPE openstack_designate_zone_status gauge
openstack_designate_zone_status{id="a86dba58-0043-4cc6-a1bb-69d5e86f3ca3",name="example.org.",status="ACTIVE",tenant_id="4335d1f0-f793-11e2-b778-0800200c9a66",type="PRIMARY"} 1
# HELP openstack_designate_zones zones
# TYPE openstack_designate_zones gauge
openstack_designate_zones 1
`

func (suite *DesignateTestSuite) TestDesignateExporter() {
	err := testutil.CollectAndCompare(*suite.Exporter, strings.NewReader(designateExpectedUp))
	assert.NoError(suite.T(), err)
}
