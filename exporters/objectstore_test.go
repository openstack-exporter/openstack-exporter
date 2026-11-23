package exporters

import (
	"strings"

	"github.com/jarcoal/httpmock"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

type ObjectStoreTestSuite struct {
	BaseOpenStackTestSuite
}

var swiftExpectedUp = `
# HELP openstack_object_store_bytes bytes
# TYPE openstack_object_store_bytes gauge
openstack_object_store_bytes{container_name="centos9-appstream",project_id="0c4e939acacf4376bdcd1129f1a054ad"} 5.2570729217e+10
openstack_object_store_bytes{container_name="centos9-baseos",project_id="0c4e939acacf4376bdcd1129f1a054ad"} 3.481572133e+09
openstack_object_store_bytes{container_name="centos9-epel",project_id="0c4e939acacf4376bdcd1129f1a054ad"} 1.6001261302e+10
openstack_object_store_bytes{container_name="centos9-epel-next",project_id="0c4e939acacf4376bdcd1129f1a054ad"} 3.02234197e+08
# HELP openstack_object_store_objects objects
# TYPE openstack_object_store_objects gauge
openstack_object_store_objects{container_name="centos9-appstream",project_id="0c4e939acacf4376bdcd1129f1a054ad"} 22505
openstack_object_store_objects{container_name="centos9-baseos",project_id="0c4e939acacf4376bdcd1129f1a054ad"} 2931
openstack_object_store_objects{container_name="centos9-epel",project_id="0c4e939acacf4376bdcd1129f1a054ad"} 16785
openstack_object_store_objects{container_name="centos9-epel-next",project_id="0c4e939acacf4376bdcd1129f1a054ad"} 509
# HELP openstack_object_store_up up
# TYPE openstack_object_store_up gauge
openstack_object_store_up 1
`

func (suite *ObjectStoreTestSuite) TestObjectStoreExporter() {
	base := (*suite.Exporter).(*ObjectStoreExporter)
	endpoint := base.ClientV2.Endpoint
	err := testutil.CollectAndCompare(*suite.Exporter, strings.NewReader(swiftExpectedUp))
	assert.NoError(suite.T(), err)
	suite.Equal(endpoint, base.ClientV2.Endpoint)
}

func (suite *ObjectStoreTestSuite) TestScopedContainersWithBytesDisabled() {
	base := (*suite.Exporter).(*ObjectStoreExporter)
	config := base.ExporterConfig
	config.TenantID = "0c4e939acacf4376bdcd1129f1a054ad"
	config.DisabledMetrics = []string{"object_store-bytes"}
	client := *config.ClientV2
	client.Endpoint = suite.MakeURL("/object-store/v1/AUTH_"+config.TenantID, "")
	endpoint := client.Endpoint
	config.ClientV2 = &client
	exporter, err := NewObjectStoreExporter(&config, base.logger)
	suite.Require().NoError(err)
	expected := swiftExpectedUp[strings.Index(swiftExpectedUp, "# HELP openstack_object_store_objects"):]
	suite.NoError(testutil.CollectAndCompare(exporter, strings.NewReader(expected)))
	suite.Equal(endpoint, exporter.ClientV2.Endpoint)
	suite.Zero(httpmock.GetCallCountInfo()["GET "+suite.MakeURL("/identity/v3/projects", "")])
}
