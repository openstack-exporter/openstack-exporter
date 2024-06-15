package openstack

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

type ContainerInfraTestSuite struct {
	BaseOpenStackTestSuite
}

var containerInfraExpectedUp = `
# HELP openstack_container_infra_cluster_masters cluster_masters
# TYPE openstack_container_infra_cluster_masters gauge
openstack_container_infra_cluster_masters{name="k8s",node_count="1",project_id="0cbd49cbf76d405d9c86562e1d579bd3",stack_id="31c1ee6c-081e-4f39-9f0f-f1d87a7defa1",status="CREATE_FAILED",uuid="273c39d5-fa17-4372-b6b1-93a572de2cef"} 1
# HELP openstack_container_infra_cluster_nodes cluster_nodes
# TYPE openstack_container_infra_cluster_nodes gauge
openstack_container_infra_cluster_nodes{master_count="1",name="k8s",project_id="0cbd49cbf76d405d9c86562e1d579bd3",stack_id="31c1ee6c-081e-4f39-9f0f-f1d87a7defa1",status="CREATE_FAILED",uuid="273c39d5-fa17-4372-b6b1-93a572de2cef"} 1
# HELP openstack_container_infra_cluster_status cluster_status
# TYPE openstack_container_infra_cluster_status gauge
openstack_container_infra_cluster_status{master_count="1",name="k8s",node_count="1",project_id="0cbd49cbf76d405d9c86562e1d579bd3",stack_id="31c1ee6c-081e-4f39-9f0f-f1d87a7defa1",status="CREATE_FAILED",uuid="273c39d5-fa17-4372-b6b1-93a572de2cef"} 1
# HELP openstack_container_infra_total_clusters total_clusters
# TYPE openstack_container_infra_total_clusters gauge
openstack_container_infra_total_clusters 1
# HELP openstack_container_infra_up up
# TYPE openstack_container_infra_up gauge
openstack_container_infra_up 1
`

func (suite *ContainerInfraTestSuite) TestContainerInfraExporter() {
	err := testutil.CollectAndCompare(*suite.Exporter, strings.NewReader(containerInfraExpectedUp))
	assert.NoError(suite.T(), err)
}
