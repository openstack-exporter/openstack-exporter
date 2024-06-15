package openstack

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
openstack_placement_resource_allocation_ratio{hostname="cmp-1-svr8204.localdomain",resourcetype="DISK_GB"} 1.2000000476837158
openstack_placement_resource_allocation_ratio{hostname="cmp-1-svr8204.localdomain",resourcetype="MEMORY_MB"} 1.2999999523162842
openstack_placement_resource_allocation_ratio{hostname="cmp-1-svr8204.localdomain",resourcetype="VCPU"} 3
openstack_placement_resource_allocation_ratio{hostname="cmp-5-svr8208.localdomain",resourcetype="DISK_GB"} 1.2000000476837158
openstack_placement_resource_allocation_ratio{hostname="cmp-5-svr8208.localdomain",resourcetype="MEMORY_MB"} 1
openstack_placement_resource_allocation_ratio{hostname="cmp-5-svr8208.localdomain",resourcetype="PCPU"} 1
# HELP openstack_placement_resource_reserved resource_reserved
# TYPE openstack_placement_resource_reserved gauge
openstack_placement_resource_reserved{hostname="cmp-1-svr8204.localdomain",resourcetype="DISK_GB"} 0
openstack_placement_resource_reserved{hostname="cmp-1-svr8204.localdomain",resourcetype="MEMORY_MB"} 8192
openstack_placement_resource_reserved{hostname="cmp-1-svr8204.localdomain",resourcetype="VCPU"} 0
openstack_placement_resource_reserved{hostname="cmp-5-svr8208.localdomain",resourcetype="DISK_GB"} 0
openstack_placement_resource_reserved{hostname="cmp-5-svr8208.localdomain",resourcetype="MEMORY_MB"} 8192
openstack_placement_resource_reserved{hostname="cmp-5-svr8208.localdomain",resourcetype="PCPU"} 0
# HELP openstack_placement_resource_total resource_total
# TYPE openstack_placement_resource_total gauge
openstack_placement_resource_total{hostname="cmp-1-svr8204.localdomain",resourcetype="DISK_GB"} 2047
openstack_placement_resource_total{hostname="cmp-1-svr8204.localdomain",resourcetype="MEMORY_MB"} 772447
openstack_placement_resource_total{hostname="cmp-1-svr8204.localdomain",resourcetype="VCPU"} 96
openstack_placement_resource_total{hostname="cmp-5-svr8208.localdomain",resourcetype="DISK_GB"} 2047
openstack_placement_resource_total{hostname="cmp-5-svr8208.localdomain",resourcetype="MEMORY_MB"} 772447
openstack_placement_resource_total{hostname="cmp-5-svr8208.localdomain",resourcetype="PCPU"} 96
# HELP openstack_placement_resource_usage resource_usage
# TYPE openstack_placement_resource_usage gauge
openstack_placement_resource_usage{hostname="cmp-1-svr8204.localdomain",resourcetype="DISK_GB"} 6969
openstack_placement_resource_usage{hostname="cmp-1-svr8204.localdomain",resourcetype="MEMORY_MB"} 1945
openstack_placement_resource_usage{hostname="cmp-1-svr8204.localdomain",resourcetype="VCPU"} 10
openstack_placement_resource_usage{hostname="cmp-5-svr8208.localdomain",resourcetype="DISK_GB"} 0
openstack_placement_resource_usage{hostname="cmp-5-svr8208.localdomain",resourcetype="MEMORY_MB"} 0
openstack_placement_resource_usage{hostname="cmp-5-svr8208.localdomain",resourcetype="PCPU"} 0
# HELP openstack_placement_up up
# TYPE openstack_placement_up gauge
openstack_placement_up 1
`

func (suite *PlacementTestSuite) TestPlacementExporter() {
	err := testutil.CollectAndCompare(*suite.Exporter, strings.NewReader(placementExpected))
	assert.NoError(suite.T(), err)
}
