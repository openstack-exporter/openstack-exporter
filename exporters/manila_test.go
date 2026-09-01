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
# HELP openstack_sharev2_limits_shares_max_gb limits_shares_max_gb
# TYPE openstack_sharev2_limits_shares_max_gb gauge
openstack_sharev2_limits_shares_max_gb{tenant="admin",tenant_id="0c4e939acacf4376bdcd1129f1a054ad"} 1000
openstack_sharev2_limits_shares_max_gb{tenant="alt_demo",tenant_id="fdb8424c4e4f4c0ba32c52e2de3bd80e"} 1000
openstack_sharev2_limits_shares_max_gb{tenant="demo",tenant_id="0cbd49cbf76d405d9c86562e1d579bd3"} 1000
openstack_sharev2_limits_shares_max_gb{tenant="invisible_to_admin",tenant_id="5961c443439d4fcebe42643723755e9d"} 1000
openstack_sharev2_limits_shares_max_gb{tenant="service",tenant_id="3d594eb0f04741069dbbb521635b21c7"} 1000
openstack_sharev2_limits_shares_max_gb{tenant="swifttenanttest1",tenant_id="43ebde53fc314b1c9ea2b8c5dc744927"} 1000
openstack_sharev2_limits_shares_max_gb{tenant="swifttenanttest2",tenant_id="2db68fed84324f29bb73130c6c2094fb"} 1000
openstack_sharev2_limits_shares_max_gb{tenant="swifttenanttest4",tenant_id="4b1eb781a47440acb8af9850103e537f"} 1000
# HELP openstack_sharev2_limits_shares_max_instances limits_shares_max_instances
# TYPE openstack_sharev2_limits_shares_max_instances gauge
openstack_sharev2_limits_shares_max_instances{tenant="admin",tenant_id="0c4e939acacf4376bdcd1129f1a054ad"} 50
openstack_sharev2_limits_shares_max_instances{tenant="alt_demo",tenant_id="fdb8424c4e4f4c0ba32c52e2de3bd80e"} 50
openstack_sharev2_limits_shares_max_instances{tenant="demo",tenant_id="0cbd49cbf76d405d9c86562e1d579bd3"} 50
openstack_sharev2_limits_shares_max_instances{tenant="invisible_to_admin",tenant_id="5961c443439d4fcebe42643723755e9d"} 50
openstack_sharev2_limits_shares_max_instances{tenant="service",tenant_id="3d594eb0f04741069dbbb521635b21c7"} 50
openstack_sharev2_limits_shares_max_instances{tenant="swifttenanttest1",tenant_id="43ebde53fc314b1c9ea2b8c5dc744927"} 50
openstack_sharev2_limits_shares_max_instances{tenant="swifttenanttest2",tenant_id="2db68fed84324f29bb73130c6c2094fb"} 50
openstack_sharev2_limits_shares_max_instances{tenant="swifttenanttest4",tenant_id="4b1eb781a47440acb8af9850103e537f"} 50
# HELP openstack_sharev2_limits_shares_used_gb limits_shares_used_gb
# TYPE openstack_sharev2_limits_shares_used_gb gauge
openstack_sharev2_limits_shares_used_gb{tenant="admin",tenant_id="0c4e939acacf4376bdcd1129f1a054ad"} 0
openstack_sharev2_limits_shares_used_gb{tenant="alt_demo",tenant_id="fdb8424c4e4f4c0ba32c52e2de3bd80e"} 0
openstack_sharev2_limits_shares_used_gb{tenant="demo",tenant_id="0cbd49cbf76d405d9c86562e1d579bd3"} 0
openstack_sharev2_limits_shares_used_gb{tenant="invisible_to_admin",tenant_id="5961c443439d4fcebe42643723755e9d"} 0
openstack_sharev2_limits_shares_used_gb{tenant="service",tenant_id="3d594eb0f04741069dbbb521635b21c7"} 0
openstack_sharev2_limits_shares_used_gb{tenant="swifttenanttest1",tenant_id="43ebde53fc314b1c9ea2b8c5dc744927"} 0
openstack_sharev2_limits_shares_used_gb{tenant="swifttenanttest2",tenant_id="2db68fed84324f29bb73130c6c2094fb"} 0
openstack_sharev2_limits_shares_used_gb{tenant="swifttenanttest4",tenant_id="4b1eb781a47440acb8af9850103e537f"} 0
# HELP openstack_sharev2_limits_shares_used_instances limits_shares_used_instances
# TYPE openstack_sharev2_limits_shares_used_instances gauge
openstack_sharev2_limits_shares_used_instances{tenant="admin",tenant_id="0c4e939acacf4376bdcd1129f1a054ad"} 0
openstack_sharev2_limits_shares_used_instances{tenant="alt_demo",tenant_id="fdb8424c4e4f4c0ba32c52e2de3bd80e"} 0
openstack_sharev2_limits_shares_used_instances{tenant="demo",tenant_id="0cbd49cbf76d405d9c86562e1d579bd3"} 0
openstack_sharev2_limits_shares_used_instances{tenant="invisible_to_admin",tenant_id="5961c443439d4fcebe42643723755e9d"} 0
openstack_sharev2_limits_shares_used_instances{tenant="service",tenant_id="3d594eb0f04741069dbbb521635b21c7"} 0
openstack_sharev2_limits_shares_used_instances{tenant="swifttenanttest1",tenant_id="43ebde53fc314b1c9ea2b8c5dc744927"} 0
openstack_sharev2_limits_shares_used_instances{tenant="swifttenanttest2",tenant_id="2db68fed84324f29bb73130c6c2094fb"} 0
openstack_sharev2_limits_shares_used_instances{tenant="swifttenanttest4",tenant_id="4b1eb781a47440acb8af9850103e537f"} 0
# HELP openstack_sharev2_share_gb share_gb
# TYPE openstack_sharev2_share_gb gauge
openstack_sharev2_share_gb{availability_zone="az1",id="4be93e2e-ffff-ffff-ffff-603e3ec2a5d6",name="share-test",project_id="ffff8fa0ca1a468db8ad00970c1effff",share_proto="NFS",share_type="az1",share_type_name="",status="available"} 1
# HELP openstack_sharev2_share_status share_status
# TYPE openstack_sharev2_share_status gauge
openstack_sharev2_share_status{id="4be93e2e-ffff-ffff-ffff-603e3ec2a5d6",name="share-test",project_id="ffff8fa0ca1a468db8ad00970c1effff",share_proto="NFS",share_type="az1",share_type_name="",size="1",status="available"} 1
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
