package exporters

import (
	"log/slog"
	"os"
	"path"
	"strings"

	"github.com/jarcoal/httpmock"
	"github.com/openstack-exporter/openstack-exporter/utils"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type IronicTestSuite struct {
	BaseOpenStackTestSuite
}

var ironicExpectedUp = `
# HELP openstack_ironic_node node
# TYPE openstack_ironic_node gauge
openstack_ironic_node{console_enabled="false",deploy_kernel="7ff5ef56-daaa-4256-9dd8-c3f1f9964ebc",deploy_ramdisk="e9c96d45-a4c8-4165-8753-9d8f32779e99",id="f50dcc35-4913-4667-a9fa-d130659c5661",maintenance="false",name="r1-02",power_state="power off",provision_state="available",resource_class="baremetal",retired="true",retired_reason="No longer needed"} 1
openstack_ironic_node{console_enabled="true",deploy_kernel="7ff5ef56-daaa-4256-9dd8-c3f1f9964ebc",deploy_ramdisk="e9c96d45-a4c8-4165-8753-9d8f32779e99",id="0129d2fc-0e5c-4b5b-a73b-01844d913957",maintenance="false",name="r1-04",power_state="power on",provision_state="active",resource_class="baremetal",retired="true",retired_reason="No longer needed"} 1
openstack_ironic_node{console_enabled="true",deploy_kernel="7ff5ef56-daaa-4256-9dd8-c3f1f9964ebc",deploy_ramdisk="e9c96d45-a4c8-4165-8753-9d8f32779e99",id="c9f98cc9-25e9-424e-8a89-002989054ec2",maintenance="true",name="r1-05",power_state="power off",provision_state="available",resource_class="baremetal",retired="true",retired_reason="No longer needed"} 1
openstack_ironic_node{console_enabled="true",deploy_kernel="7ff5ef56-daaa-4256-9dd8-c3f1f9964ebc",deploy_ramdisk="e9c96d45-a4c8-4165-8753-9d8f32779e99",id="d381bea3-8768-4f12-a9b3-abf750ba918f",maintenance="false",name="r1-03",power_state="power on",provision_state="active",resource_class="baremetal",retired="true",retired_reason="No longer needed"} 1
openstack_ironic_node{console_enabled="true",deploy_kernel="7ff5ef56-daaa-4256-9dd8-c3f1f9964ebc",deploy_ramdisk="e9c96d45-a4c8-4165-8753-9d8f32779e99",id="d5641882-f7e5-4b92-9423-7e8157586218",maintenance="true",name="r1-01",power_state="power off",provision_state="error",resource_class="baremetal",retired="true",retired_reason="No longer needed"} 1
# HELP openstack_ironic_up up
# TYPE openstack_ironic_up gauge
openstack_ironic_up 1
`

func (suite *IronicTestSuite) TestIronicExporter() {
	err := testutil.CollectAndCompare(*suite.Exporter, strings.NewReader(ironicExpectedUp))
	assert.NoError(suite.T(), err)
}

// IronicExtraLabelsTestSuite verifies that --extra-labels adds dynamic and
// static labels to the ironic node metric.
type IronicExtraLabelsTestSuite struct {
	BaseOpenStackTestSuite
}

func (suite *IronicExtraLabelsTestSuite) SetupTest() {
	httpmock.Activate()
	suite.Prefix = "openstack"
	suite.ServiceName = "baremetal"

	suite.teardownFixtures()
	suite.installFixtures()

	os.Setenv("OS_CLIENT_CONFIG_FILE", path.Join(baseFixturePath, "test_config.yaml"))

	novaMetadataMapping := new(utils.LabelMappingFlag)
	extraLabels := new(utils.ExtraLabelsFlag)
	// Add "conductor" as a dynamic label and "env=test" as a static label.
	// The service key must match the internal exporter name ("ironic"), which
	// is the same token used in metric names like openstack_ironic_node.
	require.NoError(suite.T(), extraLabels.Set("ironic.node:conductor,env=test"))

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))
	exporter, err := NewExporter("baremetal", suite.Prefix, cloudName, []string{}, "public", false, false, false, false, "", "", novaMetadataMapping, extraLabels, func() (string, error) {
		return DEFAULT_UUID, nil
	}, logger)
	require.NoError(suite.T(), err)
	suite.Exporter = &exporter
}

// ironicExtraLabelsExpected contains the expected openstack_ironic_node output
// when extra-labels=baremetal.node:conductor,env=test is active.
// Labels are sorted alphabetically: conductor, console_enabled, deploy_kernel,
// deploy_ramdisk, env, id, maintenance, name, power_state, provision_state,
// resource_class, retired, retired_reason.
var ironicExtraLabelsExpected = `
# HELP openstack_ironic_node node
# TYPE openstack_ironic_node gauge
openstack_ironic_node{conductor="https://ironic01.openstack.example.org",console_enabled="false",deploy_kernel="7ff5ef56-daaa-4256-9dd8-c3f1f9964ebc",deploy_ramdisk="e9c96d45-a4c8-4165-8753-9d8f32779e99",env="test",id="f50dcc35-4913-4667-a9fa-d130659c5661",maintenance="false",name="r1-02",power_state="power off",provision_state="available",resource_class="baremetal",retired="true",retired_reason="No longer needed"} 1
openstack_ironic_node{conductor="https://ironic01.openstack.example.org",console_enabled="true",deploy_kernel="7ff5ef56-daaa-4256-9dd8-c3f1f9964ebc",deploy_ramdisk="e9c96d45-a4c8-4165-8753-9d8f32779e99",env="test",id="0129d2fc-0e5c-4b5b-a73b-01844d913957",maintenance="false",name="r1-04",power_state="power on",provision_state="active",resource_class="baremetal",retired="true",retired_reason="No longer needed"} 1
openstack_ironic_node{conductor="https://ironic01.openstack.example.org",console_enabled="true",deploy_kernel="7ff5ef56-daaa-4256-9dd8-c3f1f9964ebc",deploy_ramdisk="e9c96d45-a4c8-4165-8753-9d8f32779e99",env="test",id="c9f98cc9-25e9-424e-8a89-002989054ec2",maintenance="true",name="r1-05",power_state="power off",provision_state="available",resource_class="baremetal",retired="true",retired_reason="No longer needed"} 1
openstack_ironic_node{conductor="https://ironic01.openstack.example.org",console_enabled="true",deploy_kernel="7ff5ef56-daaa-4256-9dd8-c3f1f9964ebc",deploy_ramdisk="e9c96d45-a4c8-4165-8753-9d8f32779e99",env="test",id="d381bea3-8768-4f12-a9b3-abf750ba918f",maintenance="false",name="r1-03",power_state="power on",provision_state="active",resource_class="baremetal",retired="true",retired_reason="No longer needed"} 1
openstack_ironic_node{conductor="https://ironic01.openstack.example.org",console_enabled="true",deploy_kernel="7ff5ef56-daaa-4256-9dd8-c3f1f9964ebc",deploy_ramdisk="e9c96d45-a4c8-4165-8753-9d8f32779e99",env="test",id="d5641882-f7e5-4b92-9423-7e8157586218",maintenance="true",name="r1-01",power_state="power off",provision_state="error",resource_class="baremetal",retired="true",retired_reason="No longer needed"} 1
`

func (suite *IronicExtraLabelsTestSuite) TestIronicExporterWithExtraLabels() {
	err := testutil.CollectAndCompare(*suite.Exporter, strings.NewReader(ironicExtraLabelsExpected), "openstack_ironic_node")
	assert.NoError(suite.T(), err)
}
