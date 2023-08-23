package exporters

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

type PlacementTestSuite struct {
	BaseOpenStackTestSuite
}

var placementExpected = `# HELP openstack_placement_local_storage_available_bytes local_storage_available_bytes
# TYPE openstack_placement_local_storage_available_bytes gauge
openstack_placement_local_storage_available_bytes{hostname="cmp-1-svr8204.localdomain"} 2.197949513728e+12
openstack_placement_local_storage_available_bytes{hostname="cmp-5-svr8208.localdomain"} 2.197949513728e+12
# HELP openstack_placement_local_storage_used_bytes local_storage_used_bytes
# TYPE openstack_placement_local_storage_used_bytes gauge
openstack_placement_local_storage_used_bytes{hostname="cmp-1-svr8204.localdomain"} 7.482906771456e+12
openstack_placement_local_storage_used_bytes{hostname="cmp-5-svr8208.localdomain"} 0
# HELP openstack_placement_memory_available_bytes memory_available_bytes
# TYPE openstack_placement_memory_available_bytes gauge
openstack_placement_memory_available_bytes{hostname="cmp-1-svr8204.localdomain"} 8.09969385472e+11
openstack_placement_memory_available_bytes{hostname="cmp-5-svr8208.localdomain"} 8.09969385472e+11
# HELP openstack_placement_memory_used_bytes memory_used_bytes
# TYPE openstack_placement_memory_used_bytes gauge
openstack_placement_memory_used_bytes{hostname="cmp-1-svr8204.localdomain"} 2.03948032e+09
openstack_placement_memory_used_bytes{hostname="cmp-5-svr8208.localdomain"} 0
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
# HELP openstack_placement_vcpus_available vcpus_available
# TYPE openstack_placement_vcpus_available gauge
openstack_placement_vcpus_available{hostname="cmp-1-svr8204.localdomain"} 96
# HELP openstack_placement_vcpus_used vcpus_used
# TYPE openstack_placement_vcpus_used gauge
openstack_placement_vcpus_used{hostname="cmp-1-svr8204.localdomain"} 10
`

func (suite *PlacementTestSuite) TestPlacementExporter() {
	err := testutil.CollectAndCompare(*suite.Exporter, strings.NewReader(placementExpected))
	assert.NoError(suite.T(), err)
}
