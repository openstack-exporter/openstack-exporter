package openstack

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

type HeatTestSuite struct {
	BaseOpenStackTestSuite
}

var heatExpectedUp = `
# HELP openstack_heat_stack_status stack_status
# TYPE openstack_heat_stack_status gauge
openstack_heat_stack_status{id="0009e826-5ad0-4310-994c-d3d2151eb6fd",name="demo-stack1",project_id="0cbd49cbf76d405d9c86562e1d579bd3",status="UPDATE_COMPLETE"} 11
openstack_heat_stack_status{id="00cb0780-c883-4964-89c3-b79d840b3cbf",name="demo-stack2",project_id="0cbd49cbf76d405d9c86562e1d579bd3",status="CREATE_COMPLETE"} 5
openstack_heat_stack_status{id="03438d56-3109-4881-b75e-c8eb83cb9985",name="demo-stack3",project_id="0cbd49cbf76d405d9c86562e1d579bd3",status="CREATE_FAILED"} 4
openstack_heat_stack_status{id="1128f6cf-589b-468c-8ba1-9ae7e3f24507",name="demo-stack4",project_id="0cbd49cbf76d405d9c86562e1d579bd3",status="UPDATE_FAILED"} 10
openstack_heat_stack_status{id="23f50926-d2ab-4e13-86ee-0c768f8ce426",name="demo-stack5",project_id="0cbd49cbf76d405d9c86562e1d579bd3",status="DELETE_IN_PROGRESS"} 6
openstack_heat_stack_status{id="24cb54d6-f060-41b6-b7ae-e4c149b35382",name="demo-stack6",project_id="0cbd49cbf76d405d9c86562e1d579bd3",status="DELETE_FAILED"} 7
# HELP openstack_heat_stack_status_counter stack_status_counter
# TYPE openstack_heat_stack_status_counter gauge
openstack_heat_stack_status_counter{status="ADOPT_COMPLETE"} 0
openstack_heat_stack_status_counter{status="ADOPT_FAILED"} 0
openstack_heat_stack_status_counter{status="ADOPT_IN_PROGRESS"} 0
openstack_heat_stack_status_counter{status="CHECK_COMPLETE"} 0
openstack_heat_stack_status_counter{status="CHECK_FAILED"} 0
openstack_heat_stack_status_counter{status="CHECK_IN_PROGRESS"} 0
openstack_heat_stack_status_counter{status="CREATE_COMPLETE"} 1
openstack_heat_stack_status_counter{status="CREATE_FAILED"} 1
openstack_heat_stack_status_counter{status="CREATE_IN_PROGRESS"} 0
openstack_heat_stack_status_counter{status="DELETE_COMPLETE"} 0
openstack_heat_stack_status_counter{status="DELETE_FAILED"} 1
openstack_heat_stack_status_counter{status="DELETE_IN_PROGRESS"} 1
openstack_heat_stack_status_counter{status="INIT_COMPLETE"} 0
openstack_heat_stack_status_counter{status="INIT_FAILED"} 0
openstack_heat_stack_status_counter{status="INIT_IN_PROGRESS"} 0
openstack_heat_stack_status_counter{status="RESUME_COMPLETE"} 0
openstack_heat_stack_status_counter{status="RESUME_FAILED"} 0
openstack_heat_stack_status_counter{status="RESUME_IN_PROGRESS"} 0
openstack_heat_stack_status_counter{status="ROLLBACK_COMPLETE"} 0
openstack_heat_stack_status_counter{status="ROLLBACK_FAILED"} 0
openstack_heat_stack_status_counter{status="ROLLBACK_IN_PROGRESS"} 0
openstack_heat_stack_status_counter{status="SNAPSHOT_COMPLETE"} 0
openstack_heat_stack_status_counter{status="SNAPSHOT_FAILED"} 0
openstack_heat_stack_status_counter{status="SNAPSHOT_IN_PROGRESS"} 0
openstack_heat_stack_status_counter{status="SUSPEND_COMPLETE"} 0
openstack_heat_stack_status_counter{status="SUSPEND_FAILED"} 0
openstack_heat_stack_status_counter{status="SUSPEND_IN_PROGRESS"} 0
openstack_heat_stack_status_counter{status="UPDATE_COMPLETE"} 1
openstack_heat_stack_status_counter{status="UPDATE_FAILED"} 1
openstack_heat_stack_status_counter{status="UPDATE_IN_PROGRESS"} 0
# HELP openstack_heat_up up
# TYPE openstack_heat_up gauge
openstack_heat_up 1
`

func (suite *HeatTestSuite) TestHeatExporter() {
	err := testutil.CollectAndCompare(*suite.Exporter, strings.NewReader(heatExpectedUp))
	assert.NoError(suite.T(), err)
}
