package exporters

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

type GlanceTestSuite struct {
	BaseOpenStackTestSuite
}

var glanceExpectedUp = `
# HELP openstack_glance_images images
# TYPE openstack_glance_images gauge
openstack_glance_images 2
# HELP openstack_glance_up up
# TYPE openstack_glance_up gauge
openstack_glance_up 1
`

func (suite *GlanceTestSuite) TestGlanceExporter() {
	err := testutil.CollectAndCompare(*suite.Exporter, strings.NewReader(glanceExpectedUp))
	assert.NoError(suite.T(), err)
}
