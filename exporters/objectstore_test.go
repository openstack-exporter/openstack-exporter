package exporters

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

type ObjectStoreTestSuite struct {
	BaseOpenStackTestSuite
}

var swiftExpectedUp = `
# HELP openstack_object_store_bytes bytes
# TYPE openstack_object_store_bytes gauge
openstack_object_store_bytes{container_name="centos9-appstream"} 5.2570729217e+10
openstack_object_store_bytes{container_name="centos9-baseos"} 3.481572133e+09
openstack_object_store_bytes{container_name="centos9-epel"} 1.6001261302e+10
openstack_object_store_bytes{container_name="centos9-epel-next"} 3.02234197e+08
# HELP openstack_object_store_objects objects
# TYPE openstack_object_store_objects gauge
openstack_object_store_objects{container_name="centos9-appstream"} 22505
openstack_object_store_objects{container_name="centos9-baseos"} 2931
openstack_object_store_objects{container_name="centos9-epel"} 16785
openstack_object_store_objects{container_name="centos9-epel-next"} 509
# HELP openstack_object_store_up up
# TYPE openstack_object_store_up gauge
openstack_object_store_up 1
`

func (suite *ObjectStoreTestSuite) TestObjectStoreExporter() {
	err := testutil.CollectAndCompare(*suite.Exporter, strings.NewReader(swiftExpectedUp))
	assert.NoError(suite.T(), err)
}
