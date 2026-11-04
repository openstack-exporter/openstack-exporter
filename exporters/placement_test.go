package exporters

import (
	"net/http"
	"os"
	"strings"

	"github.com/jarcoal/httpmock"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

type PlacementTestSuite struct {
	BaseOpenStackTestSuite
}

var placementExpected = `
# HELP openstack_placement_resource_allocation_ratio resource_allocation_ratio
# TYPE openstack_placement_resource_allocation_ratio gauge
openstack_placement_resource_allocation_ratio{hostname="cmp-1-svr8204.localdomain",resource_traits="CUSTOM_HW_FPGA_CLASS1,CUSTOM_HW_FPGA_CLASS3",resourcetype="DISK_GB"} 1.2000000476837158
openstack_placement_resource_allocation_ratio{hostname="cmp-1-svr8204.localdomain",resource_traits="CUSTOM_HW_FPGA_CLASS1,CUSTOM_HW_FPGA_CLASS3",resourcetype="MEMORY_MB"} 1.2999999523162842
openstack_placement_resource_allocation_ratio{hostname="cmp-1-svr8204.localdomain",resource_traits="CUSTOM_HW_FPGA_CLASS1,CUSTOM_HW_FPGA_CLASS3",resourcetype="VCPU"} 3
openstack_placement_resource_allocation_ratio{hostname="cmp-5-svr8208.localdomain",resource_traits="CUSTOM_GPU",resourcetype="DISK_GB"} 1.2000000476837158
openstack_placement_resource_allocation_ratio{hostname="cmp-5-svr8208.localdomain",resource_traits="CUSTOM_GPU",resourcetype="MEMORY_MB"} 1
openstack_placement_resource_allocation_ratio{hostname="cmp-5-svr8208.localdomain",resource_traits="CUSTOM_GPU",resourcetype="PCPU"} 1
# HELP openstack_placement_resource_generation resource_generation
# TYPE openstack_placement_resource_generation gauge
openstack_placement_resource_generation{hostname="cmp-1-svr8204.localdomain",resource_traits="CUSTOM_HW_FPGA_CLASS1,CUSTOM_HW_FPGA_CLASS3",resourcetype="DISK_GB"} 20
openstack_placement_resource_generation{hostname="cmp-1-svr8204.localdomain",resource_traits="CUSTOM_HW_FPGA_CLASS1,CUSTOM_HW_FPGA_CLASS3",resourcetype="MEMORY_MB"} 20
openstack_placement_resource_generation{hostname="cmp-1-svr8204.localdomain",resource_traits="CUSTOM_HW_FPGA_CLASS1,CUSTOM_HW_FPGA_CLASS3",resourcetype="VCPU"} 20
openstack_placement_resource_generation{hostname="cmp-5-svr8208.localdomain",resource_traits="CUSTOM_GPU",resourcetype="DISK_GB"} 12
openstack_placement_resource_generation{hostname="cmp-5-svr8208.localdomain",resource_traits="CUSTOM_GPU",resourcetype="MEMORY_MB"} 12
openstack_placement_resource_generation{hostname="cmp-5-svr8208.localdomain",resource_traits="CUSTOM_GPU",resourcetype="PCPU"} 12
# HELP openstack_placement_resource_provider_allocations resource_provider_allocations
# TYPE openstack_placement_resource_provider_allocations gauge
openstack_placement_resource_provider_allocations{hostname="cmp-1-svr8204.localdomain",resourcetype="DISK_GB",uuid="a0b15655-e674-4e63-aa64-cde2f5de4402"} 40
openstack_placement_resource_provider_allocations{hostname="cmp-1-svr8204.localdomain",resourcetype="MEMORY_MB",uuid="a0b15655-e674-4e63-aa64-cde2f5de4402"} 4096
openstack_placement_resource_provider_allocations{hostname="cmp-1-svr8204.localdomain",resourcetype="VCPU",uuid="a0b15655-e674-4e63-aa64-cde2f5de4402"} 2
openstack_placement_resource_provider_allocations{hostname="cmp-1-svr8204.localdomain",resourcetype="DISK_GB",uuid="b3c94dec-88e6-4e6a-9a82-7f10a81b5a5e"} 80
openstack_placement_resource_provider_allocations{hostname="cmp-1-svr8204.localdomain",resourcetype="MEMORY_MB",uuid="b3c94dec-88e6-4e6a-9a82-7f10a81b5a5e"} 8192
openstack_placement_resource_provider_allocations{hostname="cmp-1-svr8204.localdomain",resourcetype="VCPU",uuid="b3c94dec-88e6-4e6a-9a82-7f10a81b5a5e"} 4
# HELP openstack_placement_resource_reserved resource_reserved
# TYPE openstack_placement_resource_reserved gauge
openstack_placement_resource_reserved{hostname="cmp-1-svr8204.localdomain",resource_traits="CUSTOM_HW_FPGA_CLASS1,CUSTOM_HW_FPGA_CLASS3",resourcetype="DISK_GB"} 0
openstack_placement_resource_reserved{hostname="cmp-1-svr8204.localdomain",resource_traits="CUSTOM_HW_FPGA_CLASS1,CUSTOM_HW_FPGA_CLASS3",resourcetype="MEMORY_MB"} 8192
openstack_placement_resource_reserved{hostname="cmp-1-svr8204.localdomain",resource_traits="CUSTOM_HW_FPGA_CLASS1,CUSTOM_HW_FPGA_CLASS3",resourcetype="VCPU"} 0
openstack_placement_resource_reserved{hostname="cmp-5-svr8208.localdomain",resource_traits="CUSTOM_GPU",resourcetype="DISK_GB"} 0
openstack_placement_resource_reserved{hostname="cmp-5-svr8208.localdomain",resource_traits="CUSTOM_GPU",resourcetype="MEMORY_MB"} 8192
openstack_placement_resource_reserved{hostname="cmp-5-svr8208.localdomain",resource_traits="CUSTOM_GPU",resourcetype="PCPU"} 0
# HELP openstack_placement_resource_total resource_total
# TYPE openstack_placement_resource_total gauge
openstack_placement_resource_total{hostname="cmp-1-svr8204.localdomain",resource_traits="CUSTOM_HW_FPGA_CLASS1,CUSTOM_HW_FPGA_CLASS3",resourcetype="DISK_GB"} 2047
openstack_placement_resource_total{hostname="cmp-1-svr8204.localdomain",resource_traits="CUSTOM_HW_FPGA_CLASS1,CUSTOM_HW_FPGA_CLASS3",resourcetype="MEMORY_MB"} 772447
openstack_placement_resource_total{hostname="cmp-1-svr8204.localdomain",resource_traits="CUSTOM_HW_FPGA_CLASS1,CUSTOM_HW_FPGA_CLASS3",resourcetype="VCPU"} 96
openstack_placement_resource_total{hostname="cmp-5-svr8208.localdomain",resource_traits="CUSTOM_GPU",resourcetype="DISK_GB"} 2047
openstack_placement_resource_total{hostname="cmp-5-svr8208.localdomain",resource_traits="CUSTOM_GPU",resourcetype="MEMORY_MB"} 772447
openstack_placement_resource_total{hostname="cmp-5-svr8208.localdomain",resource_traits="CUSTOM_GPU",resourcetype="PCPU"} 96
# HELP openstack_placement_resource_traits resource_traits
# TYPE openstack_placement_resource_traits gauge
openstack_placement_resource_traits{hostname="cmp-1-svr8204.localdomain",resource_traits="CUSTOM_HW_FPGA_CLASS1,CUSTOM_HW_FPGA_CLASS3"} 1
openstack_placement_resource_traits{hostname="cmp-5-svr8208.localdomain",resource_traits="CUSTOM_GPU"} 1
# HELP openstack_placement_resource_usage resource_usage
# TYPE openstack_placement_resource_usage gauge
openstack_placement_resource_usage{hostname="cmp-1-svr8204.localdomain",resource_traits="CUSTOM_HW_FPGA_CLASS1,CUSTOM_HW_FPGA_CLASS3",resourcetype="DISK_GB"} 6969
openstack_placement_resource_usage{hostname="cmp-1-svr8204.localdomain",resource_traits="CUSTOM_HW_FPGA_CLASS1,CUSTOM_HW_FPGA_CLASS3",resourcetype="MEMORY_MB"} 1945
openstack_placement_resource_usage{hostname="cmp-1-svr8204.localdomain",resource_traits="CUSTOM_HW_FPGA_CLASS1,CUSTOM_HW_FPGA_CLASS3",resourcetype="VCPU"} 10
openstack_placement_resource_usage{hostname="cmp-5-svr8208.localdomain",resource_traits="CUSTOM_GPU",resourcetype="DISK_GB"} 0
openstack_placement_resource_usage{hostname="cmp-5-svr8208.localdomain",resource_traits="CUSTOM_GPU",resourcetype="MEMORY_MB"} 0
openstack_placement_resource_usage{hostname="cmp-5-svr8208.localdomain",resource_traits="CUSTOM_GPU",resourcetype="PCPU"} 0
# HELP openstack_placement_up up
# TYPE openstack_placement_up gauge
openstack_placement_up 1
`

func (suite *PlacementTestSuite) TestPlacementExporter() {
	// Real Placement rejects traits requests using the default microversion 1.0.
	for _, fixture := range []struct{ uuid, name string }{
		{"b985be15-99bf-4baf-9ef7-3ef166cd7f31", "resource_provider_1_traits"},
		{"328c9f0a-5a3c-4ad6-9347-689eb7632d7b", "resource_provider_2_traits"},
	} {
		data, err := os.ReadFile(suite.FixturePath(fixture.name))
		suite.Require().NoError(err)
		httpmock.RegisterResponder("GET", suite.MakeURL("/placement/resource_providers/"+fixture.uuid+"/traits", ""), func(req *http.Request) (*http.Response, error) {
			suite.Equal("placement 1.6", req.Header.Get("OpenStack-API-Version"))
			return httpmock.NewStringResponse(http.StatusOK, string(data)), nil
		})
	}
	err := testutil.CollectAndCompare(*suite.Exporter, strings.NewReader(placementExpected))
	assert.NoError(suite.T(), err)
}

func (suite *PlacementTestSuite) TestDisabledPlacementMetrics() {
	suite.SetResponseFromFixture("GET", 200, suite.MakeURL("/placement/", ""), suite.FixturePath("placement_api_discovery"))
	config := (*suite.Exporter).(*PlacementExporter).ExporterConfig
	config.DisabledMetrics = []string{"placement-resource_traits", "placement-resource_usage"}
	exporter, err := NewPlacementExporter(&config, (*suite.Exporter).(*PlacementExporter).logger)
	suite.Require().NoError(err)
	count := testutil.CollectAndCount(exporter, "openstack_placement_resource_traits", "openstack_placement_resource_usage")
	suite.Zero(count)
	suite.NoError(testutil.CollectAndCompare(exporter, strings.NewReader(placementExpected),
		"openstack_placement_resource_total", "openstack_placement_up"))
}

func (suite *PlacementTestSuite) TestLegacyPlacementMicroversion() {
	data, err := os.ReadFile(suite.FixturePath("placement_api_discovery"))
	suite.Require().NoError(err)
	httpmock.RegisterResponder("GET", suite.MakeURL("/placement/", ""), httpmock.NewStringResponder(200, strings.ReplaceAll(string(data), "1.39", "1.0")))
	suite.checkPlacementWithoutTraits()
}

func (suite *PlacementTestSuite) TestPlacementMicroversionOverride() {
	suite.T().Setenv("OS_PLACEMENT_API_VERSION", "1.0")
	suite.SetResponseFromFixture("GET", 200, suite.MakeURL("/placement/", ""), suite.FixturePath("placement_api_discovery"))
	suite.checkPlacementWithoutTraits()
}

func (suite *PlacementTestSuite) checkPlacementWithoutTraits() {
	config := (*suite.Exporter).(*PlacementExporter).ExporterConfig
	exporter, err := NewPlacementExporter(&config, (*suite.Exporter).(*PlacementExporter).logger)
	suite.Require().NoError(err)
	suite.Equal("1.0", exporter.ClientV2.Microversion)
	for _, uuid := range []string{"b985be15-99bf-4baf-9ef7-3ef166cd7f31", "328c9f0a-5a3c-4ad6-9347-689eb7632d7b"} {
		httpmock.RegisterResponder("GET", suite.MakeURL("/placement/resource_providers/"+uuid+"/traits", ""), func(req *http.Request) (*http.Response, error) {
			suite.Fail("traits endpoint must not be requested below microversion 1.6")
			return httpmock.NewStringResponse(404, ""), nil
		})
	}
	expected := strings.NewReplacer("CUSTOM_HW_FPGA_CLASS1,CUSTOM_HW_FPGA_CLASS3", "", "CUSTOM_GPU", "").Replace(placementExpected)
	suite.NoError(testutil.CollectAndCompare(exporter, strings.NewReader(expected)))
}
