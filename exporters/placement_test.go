package exporters

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

type PlacementTestSuite struct {
	BaseOpenStackTestSuite
}

var placementExpected = `
# HELP openstack_placement_resource_allocation_ratio resource_allocation_ratio
# TYPE openstack_placement_resource_allocation_ratio gauge
openstack_placement_resource_allocation_ratio{hostname="cmp-1-svr8204.localdomain",resourcetype="DISK_GB",zone=""} 1.2000000476837158
openstack_placement_resource_allocation_ratio{hostname="cmp-1-svr8204.localdomain",resourcetype="MEMORY_MB",zone=""} 1.2999999523162842
openstack_placement_resource_allocation_ratio{hostname="cmp-1-svr8204.localdomain",resourcetype="VCPU",zone=""} 3
openstack_placement_resource_allocation_ratio{hostname="cmp-5-svr8208.localdomain",resourcetype="DISK_GB",zone=""} 1.2000000476837158
openstack_placement_resource_allocation_ratio{hostname="cmp-5-svr8208.localdomain",resourcetype="MEMORY_MB",zone=""} 1
openstack_placement_resource_allocation_ratio{hostname="cmp-5-svr8208.localdomain",resourcetype="PCPU",zone=""} 1
# HELP openstack_placement_resource_generation resource_generation
# TYPE openstack_placement_resource_generation gauge
openstack_placement_resource_generation{hostname="cmp-1-svr8204.localdomain",resourcetype="DISK_GB",zone=""} 20
openstack_placement_resource_generation{hostname="cmp-1-svr8204.localdomain",resourcetype="MEMORY_MB",zone=""} 20
openstack_placement_resource_generation{hostname="cmp-1-svr8204.localdomain",resourcetype="VCPU",zone=""} 20
openstack_placement_resource_generation{hostname="cmp-5-svr8208.localdomain",resourcetype="DISK_GB",zone=""} 12
openstack_placement_resource_generation{hostname="cmp-5-svr8208.localdomain",resourcetype="MEMORY_MB",zone=""} 12
openstack_placement_resource_generation{hostname="cmp-5-svr8208.localdomain",resourcetype="PCPU",zone=""} 12
# HELP openstack_placement_resource_provider_allocations resource_provider_allocations
# TYPE openstack_placement_resource_provider_allocations gauge
openstack_placement_resource_provider_allocations{hostname="cmp-1-svr8204.localdomain",resourcetype="DISK_GB",uuid="a0b15655-e674-4e63-aa64-cde2f5de4402",zone=""} 40
openstack_placement_resource_provider_allocations{hostname="cmp-1-svr8204.localdomain",resourcetype="MEMORY_MB",uuid="a0b15655-e674-4e63-aa64-cde2f5de4402",zone=""} 4096
openstack_placement_resource_provider_allocations{hostname="cmp-1-svr8204.localdomain",resourcetype="VCPU",uuid="a0b15655-e674-4e63-aa64-cde2f5de4402",zone=""} 2
openstack_placement_resource_provider_allocations{hostname="cmp-1-svr8204.localdomain",resourcetype="DISK_GB",uuid="b3c94dec-88e6-4e6a-9a82-7f10a81b5a5e",zone=""} 80
openstack_placement_resource_provider_allocations{hostname="cmp-1-svr8204.localdomain",resourcetype="MEMORY_MB",uuid="b3c94dec-88e6-4e6a-9a82-7f10a81b5a5e",zone=""} 8192
openstack_placement_resource_provider_allocations{hostname="cmp-1-svr8204.localdomain",resourcetype="VCPU",uuid="b3c94dec-88e6-4e6a-9a82-7f10a81b5a5e",zone=""} 4
# HELP openstack_placement_resource_reserved resource_reserved
# TYPE openstack_placement_resource_reserved gauge
openstack_placement_resource_reserved{hostname="cmp-1-svr8204.localdomain",resourcetype="DISK_GB",zone=""} 0
openstack_placement_resource_reserved{hostname="cmp-1-svr8204.localdomain",resourcetype="MEMORY_MB",zone=""} 8192
openstack_placement_resource_reserved{hostname="cmp-1-svr8204.localdomain",resourcetype="VCPU",zone=""} 0
openstack_placement_resource_reserved{hostname="cmp-5-svr8208.localdomain",resourcetype="DISK_GB",zone=""} 0
openstack_placement_resource_reserved{hostname="cmp-5-svr8208.localdomain",resourcetype="MEMORY_MB",zone=""} 8192
openstack_placement_resource_reserved{hostname="cmp-5-svr8208.localdomain",resourcetype="PCPU",zone=""} 0
# HELP openstack_placement_resource_total resource_total
# TYPE openstack_placement_resource_total gauge
openstack_placement_resource_total{hostname="cmp-1-svr8204.localdomain",resourcetype="DISK_GB",zone=""} 2047
openstack_placement_resource_total{hostname="cmp-1-svr8204.localdomain",resourcetype="MEMORY_MB",zone=""} 772447
openstack_placement_resource_total{hostname="cmp-1-svr8204.localdomain",resourcetype="VCPU",zone=""} 96
openstack_placement_resource_total{hostname="cmp-5-svr8208.localdomain",resourcetype="DISK_GB",zone=""} 2047
openstack_placement_resource_total{hostname="cmp-5-svr8208.localdomain",resourcetype="MEMORY_MB",zone=""} 772447
openstack_placement_resource_total{hostname="cmp-5-svr8208.localdomain",resourcetype="PCPU",zone=""} 96
# HELP openstack_placement_resource_usage resource_usage
# TYPE openstack_placement_resource_usage gauge
openstack_placement_resource_usage{hostname="cmp-1-svr8204.localdomain",resourcetype="DISK_GB",zone=""} 6969
openstack_placement_resource_usage{hostname="cmp-1-svr8204.localdomain",resourcetype="MEMORY_MB",zone=""} 1945
openstack_placement_resource_usage{hostname="cmp-1-svr8204.localdomain",resourcetype="VCPU",zone=""} 10
openstack_placement_resource_usage{hostname="cmp-5-svr8208.localdomain",resourcetype="DISK_GB",zone=""} 0
openstack_placement_resource_usage{hostname="cmp-5-svr8208.localdomain",resourcetype="MEMORY_MB",zone=""} 0
openstack_placement_resource_usage{hostname="cmp-5-svr8208.localdomain",resourcetype="PCPU",zone=""} 0
# HELP openstack_placement_up up
# TYPE openstack_placement_up gauge
openstack_placement_up 1
`

func (suite *PlacementTestSuite) TestPlacementExporter() {
	err := testutil.CollectAndCompare(*suite.Exporter, strings.NewReader(placementExpected))
	assert.NoError(suite.T(), err)
}
