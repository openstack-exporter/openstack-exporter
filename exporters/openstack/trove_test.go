package openstack

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

type TroveTestSuite struct {
	BaseOpenStackTestSuite
}

var troveExpectedUp = `
# HELP openstack_trove_instance_status instance_status
# TYPE openstack_trove_instance_status gauge
openstack_trove_instance_status{datastore_type="mysql",datastore_version="5.7",health_status="available",id="0cef87c6-bd23-4f6b-8458-a393c39486d8",name="mysql1",region="RegionOne",status="ACTIVE",tenant_id="0cbd49cbf76d405d9c86562e1d579bd3"} 2
# HELP openstack_trove_instance_volume_size_gb instance_volume_size_gb
# TYPE openstack_trove_instance_volume_size_gb gauge
openstack_trove_instance_volume_size_gb{datastore_type="mysql",datastore_version="5.7",health_status="available",id="0cef87c6-bd23-4f6b-8458-a393c39486d8",name="mysql1",region="RegionOne",status="ACTIVE",tenant_id="0cbd49cbf76d405d9c86562e1d579bd3"} 20
# HELP openstack_trove_instance_volume_used_gb instance_volume_used_gb
# TYPE openstack_trove_instance_volume_used_gb gauge
openstack_trove_instance_volume_used_gb{datastore_type="mysql",datastore_version="5.7",health_status="available",id="0cef87c6-bd23-4f6b-8458-a393c39486d8",name="mysql1",region="RegionOne",status="ACTIVE",tenant_id="0cbd49cbf76d405d9c86562e1d579bd3"} 0.4
# HELP openstack_trove_total_instances total_instances
# TYPE openstack_trove_total_instances gauge
openstack_trove_total_instances 1
# HELP openstack_trove_up up
# TYPE openstack_trove_up gauge
openstack_trove_up 1
`

func (suite *TroveTestSuite) TestTroveExporter() {
	err := testutil.CollectAndCompare(*suite.Exporter, strings.NewReader(troveExpectedUp))
	assert.NoError(suite.T(), err)
}
