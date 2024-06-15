package openstack

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

type GlanceTestSuite struct {
	BaseOpenStackTestSuite
}

var glanceExpectedUp = `
# HELP openstack_glance_image_bytes image_bytes
# TYPE openstack_glance_image_bytes gauge
openstack_glance_image_bytes{id="781b3762-9469-4cec-b58d-3349e5de4e9c",name="F17-x86_64-cfntools",tenant_id="5ef70662f8b34079a6eddb8da9d75fe8"} 4.76704768e+08
openstack_glance_image_bytes{id="1bea47ed-f6a9-463b-b423-14b9cca9ad27",name="cirros-0.3.2-x86_64-disk",tenant_id="5ef70662f8b34079a6eddb8da9d75fe8"} 1.3167616e+07
# HELP openstack_glance_image_created_at image_created_at
# TYPE openstack_glance_image_created_at gauge
openstack_glance_image_created_at{hidden="false",id="781b3762-9469-4cec-b58d-3349e5de4e9c",name="F17-x86_64-cfntools",status="active",tenant_id="5ef70662f8b34079a6eddb8da9d75fe8",visibility="public"} 1.414657419e+09
openstack_glance_image_created_at{hidden="false",id="1bea47ed-f6a9-463b-b423-14b9cca9ad27",name="cirros-0.3.2-x86_64-disk",status="active",tenant_id="5ef70662f8b34079a6eddb8da9d75fe8",visibility="public"} 1.415380026e+09
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
