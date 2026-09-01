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
# HELP openstack_cinder_backup_gb backup_gb
# TYPE openstack_cinder_backup_gb gauge
openstack_cinder_backup_gb{id="2ef47aee-8844-490c-804d-2a8efe561c65",incremental="false",name="test-volume-backup",snapshot_id="",status="available",tenant_id="bab7d5c60cd041a0a36f7c4b6e1dd978",volume_id="173f7b48-c4c1-4e70-9acc-086b39073506"} 1
openstack_cinder_backup_gb{id="4dbf0ec2-0b57-4669-9823-9f7c76f2b4f8",incremental="true",name="test-volume-backup-incremental",snapshot_id="b1323cda-8e4b-41c1-afc5-2fc791809c8c",status="available",tenant_id="bab7d5c60cd041a0a36f7c4b6e1dd978",volume_id="173f7b48-c4c1-4e70-9acc-086b39073506"} 1
# HELP openstack_cinder_backups backups
# TYPE openstack_cinder_backups gauge
openstack_cinder_backups 2
# HELP openstack_cinder_agent_state agent_state
# TYPE openstack_cinder_agent_state gauge
openstack_cinder_agent_state{adminState="enabled",disabledReason="",hostname="devstack@lvmdriver-1",service="cinder-volume",uuid="3649e0f6-de80-ab6e-4f1c-351042d2f7fe",zone="nova"} 1
openstack_cinder_agent_state{adminState="enabled",disabledReason="Test1",hostname="devstack",service="cinder-scheduler",uuid="3649e0f6-de80-ab6e-4f1c-351042d2f7fe",zone="nova"} 1
openstack_cinder_agent_state{adminState="enabled",disabledReason="Test2",hostname="devstack",service="cinder-backup",uuid="3649e0f6-de80-ab6e-4f1c-351042d2f7fe",zone="nova"} 1
# HELP openstack_cinder_limits_backup_max_gb limits_backup_max_gb
# TYPE openstack_cinder_limits_backup_max_gb gauge
openstack_cinder_limits_backup_max_gb{tenant="admin",tenant_id="0c4e939acacf4376bdcd1129f1a054ad"} 1000
openstack_cinder_limits_backup_max_gb{tenant="alt_demo",tenant_id="fdb8424c4e4f4c0ba32c52e2de3bd80e"} 1000
openstack_cinder_limits_backup_max_gb{tenant="demo",tenant_id="0cbd49cbf76d405d9c86562e1d579bd3"} 1000
openstack_cinder_limits_backup_max_gb{tenant="invisible_to_admin",tenant_id="5961c443439d4fcebe42643723755e9d"} 1000
openstack_cinder_limits_backup_max_gb{tenant="service",tenant_id="3d594eb0f04741069dbbb521635b21c7"} 1000
openstack_cinder_limits_backup_max_gb{tenant="swifttenanttest1",tenant_id="43ebde53fc314b1c9ea2b8c5dc744927"} 1000
openstack_cinder_limits_backup_max_gb{tenant="swifttenanttest2",tenant_id="2db68fed84324f29bb73130c6c2094fb"} 1000
openstack_cinder_limits_backup_max_gb{tenant="swifttenanttest4",tenant_id="4b1eb781a47440acb8af9850103e537f"} 1000
# HELP openstack_cinder_limits_backup_used_gb limits_backup_used_gb
# TYPE openstack_cinder_limits_backup_used_gb gauge
openstack_cinder_limits_backup_used_gb{tenant="admin",tenant_id="0c4e939acacf4376bdcd1129f1a054ad"} 0
openstack_cinder_limits_backup_used_gb{tenant="alt_demo",tenant_id="fdb8424c4e4f4c0ba32c52e2de3bd80e"} 0
openstack_cinder_limits_backup_used_gb{tenant="demo",tenant_id="0cbd49cbf76d405d9c86562e1d579bd3"} 0
openstack_cinder_limits_backup_used_gb{tenant="invisible_to_admin",tenant_id="5961c443439d4fcebe42643723755e9d"} 0
openstack_cinder_limits_backup_used_gb{tenant="service",tenant_id="3d594eb0f04741069dbbb521635b21c7"} 0
openstack_cinder_limits_backup_used_gb{tenant="swifttenanttest1",tenant_id="43ebde53fc314b1c9ea2b8c5dc744927"} 0
openstack_cinder_limits_backup_used_gb{tenant="swifttenanttest2",tenant_id="2db68fed84324f29bb73130c6c2094fb"} 0
openstack_cinder_limits_backup_used_gb{tenant="swifttenanttest4",tenant_id="4b1eb781a47440acb8af9850103e537f"} 0
# HELP openstack_cinder_limits_backups_max limits_backups_max
# TYPE openstack_cinder_limits_backups_max gauge
openstack_cinder_limits_backups_max{tenant="admin",tenant_id="0c4e939acacf4376bdcd1129f1a054ad"} 10
openstack_cinder_limits_backups_max{tenant="alt_demo",tenant_id="fdb8424c4e4f4c0ba32c52e2de3bd80e"} 10
openstack_cinder_limits_backups_max{tenant="demo",tenant_id="0cbd49cbf76d405d9c86562e1d579bd3"} 10
openstack_cinder_limits_backups_max{tenant="invisible_to_admin",tenant_id="5961c443439d4fcebe42643723755e9d"} 10
openstack_cinder_limits_backups_max{tenant="service",tenant_id="3d594eb0f04741069dbbb521635b21c7"} 10
openstack_cinder_limits_backups_max{tenant="swifttenanttest1",tenant_id="43ebde53fc314b1c9ea2b8c5dc744927"} 10
openstack_cinder_limits_backups_max{tenant="swifttenanttest2",tenant_id="2db68fed84324f29bb73130c6c2094fb"} 10
openstack_cinder_limits_backups_max{tenant="swifttenanttest4",tenant_id="4b1eb781a47440acb8af9850103e537f"} 10
# HELP openstack_cinder_limits_backups_used limits_backups_used
# TYPE openstack_cinder_limits_backups_used gauge
openstack_cinder_limits_backups_used{tenant="admin",tenant_id="0c4e939acacf4376bdcd1129f1a054ad"} 0
openstack_cinder_limits_backups_used{tenant="alt_demo",tenant_id="fdb8424c4e4f4c0ba32c52e2de3bd80e"} 0
openstack_cinder_limits_backups_used{tenant="demo",tenant_id="0cbd49cbf76d405d9c86562e1d579bd3"} 0
openstack_cinder_limits_backups_used{tenant="invisible_to_admin",tenant_id="5961c443439d4fcebe42643723755e9d"} 0
openstack_cinder_limits_backups_used{tenant="service",tenant_id="3d594eb0f04741069dbbb521635b21c7"} 0
openstack_cinder_limits_backups_used{tenant="swifttenanttest1",tenant_id="43ebde53fc314b1c9ea2b8c5dc744927"} 0
openstack_cinder_limits_backups_used{tenant="swifttenanttest2",tenant_id="2db68fed84324f29bb73130c6c2094fb"} 0
openstack_cinder_limits_backups_used{tenant="swifttenanttest4",tenant_id="4b1eb781a47440acb8af9850103e537f"} 0
# HELP openstack_cinder_limits_snapshots_max limits_snapshots_max
# TYPE openstack_cinder_limits_snapshots_max gauge
openstack_cinder_limits_snapshots_max{tenant="admin",tenant_id="0c4e939acacf4376bdcd1129f1a054ad"} 10
openstack_cinder_limits_snapshots_max{tenant="alt_demo",tenant_id="fdb8424c4e4f4c0ba32c52e2de3bd80e"} 10
openstack_cinder_limits_snapshots_max{tenant="demo",tenant_id="0cbd49cbf76d405d9c86562e1d579bd3"} 10
openstack_cinder_limits_snapshots_max{tenant="invisible_to_admin",tenant_id="5961c443439d4fcebe42643723755e9d"} 10
openstack_cinder_limits_snapshots_max{tenant="service",tenant_id="3d594eb0f04741069dbbb521635b21c7"} 10
openstack_cinder_limits_snapshots_max{tenant="swifttenanttest1",tenant_id="43ebde53fc314b1c9ea2b8c5dc744927"} 10
openstack_cinder_limits_snapshots_max{tenant="swifttenanttest2",tenant_id="2db68fed84324f29bb73130c6c2094fb"} 10
openstack_cinder_limits_snapshots_max{tenant="swifttenanttest4",tenant_id="4b1eb781a47440acb8af9850103e537f"} 10
# HELP openstack_cinder_limits_snapshots_used limits_snapshots_used
# TYPE openstack_cinder_limits_snapshots_used gauge
openstack_cinder_limits_snapshots_used{tenant="admin",tenant_id="0c4e939acacf4376bdcd1129f1a054ad"} 0
openstack_cinder_limits_snapshots_used{tenant="alt_demo",tenant_id="fdb8424c4e4f4c0ba32c52e2de3bd80e"} 0
openstack_cinder_limits_snapshots_used{tenant="demo",tenant_id="0cbd49cbf76d405d9c86562e1d579bd3"} 0
openstack_cinder_limits_snapshots_used{tenant="invisible_to_admin",tenant_id="5961c443439d4fcebe42643723755e9d"} 0
openstack_cinder_limits_snapshots_used{tenant="service",tenant_id="3d594eb0f04741069dbbb521635b21c7"} 0
openstack_cinder_limits_snapshots_used{tenant="swifttenanttest1",tenant_id="43ebde53fc314b1c9ea2b8c5dc744927"} 0
openstack_cinder_limits_snapshots_used{tenant="swifttenanttest2",tenant_id="2db68fed84324f29bb73130c6c2094fb"} 0
openstack_cinder_limits_snapshots_used{tenant="swifttenanttest4",tenant_id="4b1eb781a47440acb8af9850103e537f"} 0
# HELP openstack_cinder_limits_volume_max_gb limits_volume_max_gb
# TYPE openstack_cinder_limits_volume_max_gb gauge
openstack_cinder_limits_volume_max_gb{tenant="admin",tenant_id="0c4e939acacf4376bdcd1129f1a054ad"} 1000
openstack_cinder_limits_volume_max_gb{tenant="alt_demo",tenant_id="fdb8424c4e4f4c0ba32c52e2de3bd80e"} 1000
openstack_cinder_limits_volume_max_gb{tenant="demo",tenant_id="0cbd49cbf76d405d9c86562e1d579bd3"} 1000
openstack_cinder_limits_volume_max_gb{tenant="invisible_to_admin",tenant_id="5961c443439d4fcebe42643723755e9d"} 1000
openstack_cinder_limits_volume_max_gb{tenant="service",tenant_id="3d594eb0f04741069dbbb521635b21c7"} 1000
openstack_cinder_limits_volume_max_gb{tenant="swifttenanttest1",tenant_id="43ebde53fc314b1c9ea2b8c5dc744927"} 1000
openstack_cinder_limits_volume_max_gb{tenant="swifttenanttest2",tenant_id="2db68fed84324f29bb73130c6c2094fb"} 1000
openstack_cinder_limits_volume_max_gb{tenant="swifttenanttest4",tenant_id="4b1eb781a47440acb8af9850103e537f"} 1000
# HELP openstack_cinder_limits_volume_used_gb limits_volume_used_gb
# TYPE openstack_cinder_limits_volume_used_gb gauge
openstack_cinder_limits_volume_used_gb{tenant="admin",tenant_id="0c4e939acacf4376bdcd1129f1a054ad"} 0
openstack_cinder_limits_volume_used_gb{tenant="alt_demo",tenant_id="fdb8424c4e4f4c0ba32c52e2de3bd80e"} 0
openstack_cinder_limits_volume_used_gb{tenant="demo",tenant_id="0cbd49cbf76d405d9c86562e1d579bd3"} 0
openstack_cinder_limits_volume_used_gb{tenant="invisible_to_admin",tenant_id="5961c443439d4fcebe42643723755e9d"} 0
openstack_cinder_limits_volume_used_gb{tenant="service",tenant_id="3d594eb0f04741069dbbb521635b21c7"} 0
openstack_cinder_limits_volume_used_gb{tenant="swifttenanttest1",tenant_id="43ebde53fc314b1c9ea2b8c5dc744927"} 0
openstack_cinder_limits_volume_used_gb{tenant="swifttenanttest2",tenant_id="2db68fed84324f29bb73130c6c2094fb"} 0
openstack_cinder_limits_volume_used_gb{tenant="swifttenanttest4",tenant_id="4b1eb781a47440acb8af9850103e537f"} 0
# HELP openstack_cinder_limits_volumes_max limits_volumes_max
# TYPE openstack_cinder_limits_volumes_max gauge
openstack_cinder_limits_volumes_max{tenant="admin",tenant_id="0c4e939acacf4376bdcd1129f1a054ad"} 10
openstack_cinder_limits_volumes_max{tenant="alt_demo",tenant_id="fdb8424c4e4f4c0ba32c52e2de3bd80e"} 10
openstack_cinder_limits_volumes_max{tenant="demo",tenant_id="0cbd49cbf76d405d9c86562e1d579bd3"} 10
openstack_cinder_limits_volumes_max{tenant="invisible_to_admin",tenant_id="5961c443439d4fcebe42643723755e9d"} 10
openstack_cinder_limits_volumes_max{tenant="service",tenant_id="3d594eb0f04741069dbbb521635b21c7"} 10
openstack_cinder_limits_volumes_max{tenant="swifttenanttest1",tenant_id="43ebde53fc314b1c9ea2b8c5dc744927"} 10
openstack_cinder_limits_volumes_max{tenant="swifttenanttest2",tenant_id="2db68fed84324f29bb73130c6c2094fb"} 10
openstack_cinder_limits_volumes_max{tenant="swifttenanttest4",tenant_id="4b1eb781a47440acb8af9850103e537f"} 10
# HELP openstack_cinder_limits_volumes_used limits_volumes_used
# TYPE openstack_cinder_limits_volumes_used gauge
openstack_cinder_limits_volumes_used{tenant="admin",tenant_id="0c4e939acacf4376bdcd1129f1a054ad"} 0
openstack_cinder_limits_volumes_used{tenant="alt_demo",tenant_id="fdb8424c4e4f4c0ba32c52e2de3bd80e"} 0
openstack_cinder_limits_volumes_used{tenant="demo",tenant_id="0cbd49cbf76d405d9c86562e1d579bd3"} 0
openstack_cinder_limits_volumes_used{tenant="invisible_to_admin",tenant_id="5961c443439d4fcebe42643723755e9d"} 0
openstack_cinder_limits_volumes_used{tenant="service",tenant_id="3d594eb0f04741069dbbb521635b21c7"} 0
openstack_cinder_limits_volumes_used{tenant="swifttenanttest1",tenant_id="43ebde53fc314b1c9ea2b8c5dc744927"} 0
openstack_cinder_limits_volumes_used{tenant="swifttenanttest2",tenant_id="2db68fed84324f29bb73130c6c2094fb"} 0
openstack_cinder_limits_volumes_used{tenant="swifttenanttest4",tenant_id="4b1eb781a47440acb8af9850103e537f"} 0
# HELP openstack_cinder_pool_capacity_free_gb pool_capacity_free_gb
# TYPE openstack_cinder_pool_capacity_free_gb gauge
openstack_cinder_pool_capacity_free_gb{name="i666testhost@FastPool01",vendor_name="EMC",volume_backend_name="VNX_Pool"} 636.316
# HELP openstack_cinder_pool_capacity_total_gb pool_capacity_total_gb
# TYPE openstack_cinder_pool_capacity_total_gb gauge
openstack_cinder_pool_capacity_total_gb{name="i666testhost@FastPool01",vendor_name="EMC",volume_backend_name="VNX_Pool"} 1692.429
# HELP openstack_cinder_snapshot_gb snapshot_gb
# TYPE openstack_cinder_snapshot_gb gauge
openstack_cinder_snapshot_gb{id="b1323cda-8e4b-41c1-afc5-2fc791809c8c",name="test-volume-snapshot",status="available",tenant_id="bab7d5c60cd041a0a36f7c4b6e1dd978",volume_id="173f7b48-c4c1-4e70-9acc-086b39073506"} 1
# HELP openstack_cinder_snapshots snapshots
# TYPE openstack_cinder_snapshots gauge
openstack_cinder_snapshots 1
# HELP openstack_cinder_snapshots_by_tenant snapshots_by_tenant
# TYPE openstack_cinder_snapshots_by_tenant gauge
openstack_cinder_snapshots_by_tenant{tenant_id="bab7d5c60cd041a0a36f7c4b6e1dd978"} 1
# HELP openstack_cinder_up up
# TYPE openstack_cinder_up gauge
openstack_cinder_up 1
# HELP openstack_cinder_volume_gb volume_gb
# TYPE openstack_cinder_volume_gb gauge
openstack_cinder_volume_gb{availability_zone="nova",bootable="false",id="6edbc2f4-1507-44f8-ac0d-eed1d2608d38",name="test-volume-attachments",server_id="f4fda93b-06e0-4743-8117-bc8bcecd651b",status="in-use",tenant_id="bab7d5c60cd041a0a36f7c4b6e1dd978",user_id="32779452fcd34ae1a53a797ac8a1e064",volume_type="lvmdriver-1"} 2
openstack_cinder_volume_gb{availability_zone="nova",bootable="true",id="173f7b48-c4c1-4e70-9acc-086b39073506",name="test-volume",server_id="",status="available",tenant_id="bab7d5c60cd041a0a36f7c4b6e1dd978",user_id="32779452fcd34ae1a53a797ac8a1e064",volume_type="lvmdriver-1"} 1
# HELP openstack_cinder_volume_status volume_status
# TYPE openstack_cinder_volume_status gauge
openstack_cinder_volume_status{bootable="false",host="difleming@lvmdriver-1#lvmdriver-1",id="6edbc2f4-1507-44f8-ac0d-eed1d2608d38",name="test-volume-attachments",server_id="f4fda93b-06e0-4743-8117-bc8bcecd651b",size="2",status="in-use",tenant_id="bab7d5c60cd041a0a36f7c4b6e1dd978",volume_type="lvmdriver-1"} 5
openstack_cinder_volume_status{bootable="true",host="difleming@lvmdriver-1#lvmdriver-1",id="173f7b48-c4c1-4e70-9acc-086b39073506",name="test-volume",server_id="",size="1",status="available",tenant_id="bab7d5c60cd041a0a36f7c4b6e1dd978",volume_type="lvmdriver-1"} 1
# HELP openstack_cinder_volume_status_counter volume_status_counter
# TYPE openstack_cinder_volume_status_counter gauge
openstack_cinder_volume_status_counter{status="attaching"} 0
openstack_cinder_volume_status_counter{status="available"} 1
openstack_cinder_volume_status_counter{status="awaiting-transfer"} 0
openstack_cinder_volume_status_counter{status="backing-up"} 0
openstack_cinder_volume_status_counter{status="creating"} 0
openstack_cinder_volume_status_counter{status="deleting"} 0
openstack_cinder_volume_status_counter{status="detaching"} 0
openstack_cinder_volume_status_counter{status="downloading"} 0
openstack_cinder_volume_status_counter{status="error"} 0
openstack_cinder_volume_status_counter{status="error_backing-up"} 0
openstack_cinder_volume_status_counter{status="error_deleting"} 0
openstack_cinder_volume_status_counter{status="error_extending"} 0
openstack_cinder_volume_status_counter{status="error_restoring"} 0
openstack_cinder_volume_status_counter{status="extending"} 0
openstack_cinder_volume_status_counter{status="in-use"} 1
openstack_cinder_volume_status_counter{status="maintenance"} 0
openstack_cinder_volume_status_counter{status="reserved"} 0
openstack_cinder_volume_status_counter{status="restoring-backup"} 0
openstack_cinder_volume_status_counter{status="retyping"} 0
openstack_cinder_volume_status_counter{status="uploading"} 0
# HELP openstack_cinder_volume_type_quota_gigabytes volume_type_quota_gigabytes
# TYPE openstack_cinder_volume_type_quota_gigabytes gauge
openstack_cinder_volume_type_quota_gigabytes{tenant="admin",tenant_id="0c4e939acacf4376bdcd1129f1a054ad",volume_type="lvmdriver-1"} 1000
openstack_cinder_volume_type_quota_gigabytes{tenant="alt_demo",tenant_id="fdb8424c4e4f4c0ba32c52e2de3bd80e",volume_type="lvmdriver-1"} 1000
openstack_cinder_volume_type_quota_gigabytes{tenant="demo",tenant_id="0cbd49cbf76d405d9c86562e1d579bd3",volume_type="lvmdriver-1"} 1000
openstack_cinder_volume_type_quota_gigabytes{tenant="invisible_to_admin",tenant_id="5961c443439d4fcebe42643723755e9d",volume_type="lvmdriver-1"} 1000
openstack_cinder_volume_type_quota_gigabytes{tenant="service",tenant_id="3d594eb0f04741069dbbb521635b21c7",volume_type="lvmdriver-1"} 1000
openstack_cinder_volume_type_quota_gigabytes{tenant="swifttenanttest1",tenant_id="43ebde53fc314b1c9ea2b8c5dc744927",volume_type="lvmdriver-1"} 1000
openstack_cinder_volume_type_quota_gigabytes{tenant="swifttenanttest2",tenant_id="2db68fed84324f29bb73130c6c2094fb",volume_type="lvmdriver-1"} 1000
openstack_cinder_volume_type_quota_gigabytes{tenant="swifttenanttest4",tenant_id="4b1eb781a47440acb8af9850103e537f",volume_type="lvmdriver-1"} 1000
# HELP openstack_cinder_volumes volumes
# TYPE openstack_cinder_volumes gauge
openstack_cinder_volumes 2
`

func (suite *CinderTestSuite) TestCinderExporter() {
	err := testutil.CollectAndCompare(*suite.Exporter, strings.NewReader(cinderExpectedUp))
	assert.NoError(suite.T(), err)
}
