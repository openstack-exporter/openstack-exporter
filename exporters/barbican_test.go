package exporters

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

type BarbicanTestSuite struct {
	BaseOpenStackTestSuite
}

var barbicanExpectedUp = `
# HELP openstack_barbican_containers containers
# TYPE openstack_barbican_containers gauge
openstack_barbican_containers 1
# HELP openstack_barbican_orders orders
# TYPE openstack_barbican_orders gauge
openstack_barbican_orders{status="ACTIVE"} 1
openstack_barbican_orders{status="ERROR"} 1
openstack_barbican_orders{status="PENDING"} 0
# HELP openstack_barbican_secrets secrets
# TYPE openstack_barbican_secrets gauge
openstack_barbican_secrets 2
# HELP openstack_barbican_up up
# TYPE openstack_barbican_up gauge
openstack_barbican_up 1
`

func (suite *BarbicanTestSuite) TestBarbicanExporter() {
	err := testutil.CollectAndCompare(*suite.Exporter, strings.NewReader(barbicanExpectedUp))
	assert.NoError(suite.T(), err)
}
