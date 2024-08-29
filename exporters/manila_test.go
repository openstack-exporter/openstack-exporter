package exporters

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

type ManilaTestSuite struct {
	BaseOpenStackTestSuite
}

var manilaExpectedUp = `
# HELP openstack_sharev2_share_gb share_gb
# TYPE openstack_sharev2_share_gb gauge
openstack_sharev2_share_gb{availability_zone="az1",id="4be93e2e-ffff-ffff-ffff-603e3ec2a5d6",name="share-test",share_proto="NFS",share_type="az1",share_type_name="",status="available"} 1
# HELP openstack_sharev2_share_status share_status
# TYPE openstack_sharev2_share_status gauge
openstack_sharev2_share_status{id="4be93e2e-ffff-ffff-ffff-603e3ec2a5d6",name="share-test",share_proto="NFS",share_type="az1",share_type_name="",size="1",status="available"} 1
# HELP openstack_sharev2_share_status_counter share_status_counter
# TYPE openstack_sharev2_share_status_counter gauge
openstack_sharev2_share_status_counter{status="available"} 1
openstack_sharev2_share_status_counter{status="creating"} 0
openstack_sharev2_share_status_counter{status="deleting"} 0
openstack_sharev2_share_status_counter{status="error"} 0
openstack_sharev2_share_status_counter{status="error_deleting"} 0
openstack_sharev2_share_status_counter{status="extending"} 0
openstack_sharev2_share_status_counter{status="inactive"} 0
openstack_sharev2_share_status_counter{status="managing"} 0
openstack_sharev2_share_status_counter{status="migrating"} 0
openstack_sharev2_share_status_counter{status="migration_error"} 0
openstack_sharev2_share_status_counter{status="restoring"} 0
openstack_sharev2_share_status_counter{status="reverting"} 0
openstack_sharev2_share_status_counter{status="reverting_error"} 0
openstack_sharev2_share_status_counter{status="reverting_to_snapshot"} 0
openstack_sharev2_share_status_counter{status="shrinking"} 0
openstack_sharev2_share_status_counter{status="shrinking_error"} 0
openstack_sharev2_share_status_counter{status="soft_deleting"} 0
openstack_sharev2_share_status_counter{status="unmanaging"} 0
openstack_sharev2_share_status_counter{status="updating"} 0
# HELP openstack_sharev2_shares_counter shares_counter
# TYPE openstack_sharev2_shares_counter gauge
openstack_sharev2_shares_counter 1
# HELP openstack_sharev2_up up
# TYPE openstack_sharev2_up gauge
openstack_sharev2_up 1
`

func (suite *ManilaTestSuite) TestManilaExporter() {
	err := testutil.CollectAndCompare(*suite.Exporter, strings.NewReader(manilaExpectedUp))
	assert.NoError(suite.T(), err)
}
