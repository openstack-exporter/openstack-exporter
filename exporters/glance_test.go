package exporters

import (
	"strings"

	"github.com/jarcoal/httpmock"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

type GlanceTestSuite struct {
	BaseOpenStackTestSuite
}

var glanceExpectedUp = `
# HELP openstack_glance_image_bytes image_bytes
# TYPE openstack_glance_image_bytes gauge
openstack_glance_image_bytes{id="781b3762-9469-4cec-b58d-3349e5de4e9c",image_type="snapshot",name="F17-x86_64-cfntools",tenant_id="5ef70662f8b34079a6eddb8da9d75fe8"} 4.76704768e+08
openstack_glance_image_bytes{id="1bea47ed-f6a9-463b-b423-14b9cca9ad27",image_type="image",name="cirros-0.3.2-x86_64-disk",tenant_id="5ef70662f8b34079a6eddb8da9d75fe8"} 1.3167616e+07
# HELP openstack_glance_image_created_at image_created_at
# TYPE openstack_glance_image_created_at gauge
openstack_glance_image_created_at{hidden="false",id="781b3762-9469-4cec-b58d-3349e5de4e9c",image_type="snapshot",name="F17-x86_64-cfntools",status="active",tenant_id="5ef70662f8b34079a6eddb8da9d75fe8",visibility="public"} 1.414657419e+09
openstack_glance_image_created_at{hidden="false",id="1bea47ed-f6a9-463b-b423-14b9cca9ad27",image_type="image",name="cirros-0.3.2-x86_64-disk",status="active",tenant_id="5ef70662f8b34079a6eddb8da9d75fe8",visibility="public"} 1.415380026e+09
# HELP openstack_glance_image_info image_info
# TYPE openstack_glance_image_info gauge
openstack_glance_image_info{disk_format="qcow2",id="1bea47ed-f6a9-463b-b423-14b9cca9ad27",name="cirros-0.3.2-x86_64-disk",project_id="5ef70662f8b34079a6eddb8da9d75fe8",status="active",tags=""} 1
openstack_glance_image_info{disk_format="qcow2",id="781b3762-9469-4cec-b58d-3349e5de4e9c",name="F17-x86_64-cfntools",project_id="5ef70662f8b34079a6eddb8da9d75fe8",status="active",tags=""} 1
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

func (suite *GlanceTestSuite) TestImageCountAndInfo() {
	for _, tc := range []struct {
		name, disabled, count string
		empty                 bool
		info                  int
	}{
		{"populated catalog", "", "2", false, 2},
		{"empty catalog", "", "0", true, 0},
		{"count disabled", "glance-images", "", false, 2},
		{"info disabled", "glance-image_info", "2", false, 0},
	} {
		suite.Run(tc.name, func() {
			if tc.empty {
				httpmock.RegisterResponder("GET", suite.MakeURL("/glance/v2/images", ""),
					httpmock.NewJsonResponderOrPanic(200, map[string]any{"images": []any{}}))
			} else {
				suite.SetResponseFromFixture("GET", 200, suite.MakeURL("/glance/v2/images", ""), suite.FixturePath("glance_images"))
			}
			base := (*suite.Exporter).(*GlanceExporter)
			config := base.ExporterConfig
			config.DisabledMetrics = []string{tc.disabled}
			config.DisableSlowMetrics = true
			exporter, err := NewGlanceExporter(&config, base.logger)
			suite.Require().NoError(err)
			expected := ""
			if tc.count != "" {
				expected = "# HELP openstack_glance_images images\n# TYPE openstack_glance_images gauge\nopenstack_glance_images " + tc.count + "\n"
			}
			suite.NoError(testutil.CollectAndCompare(exporter, strings.NewReader(expected), "openstack_glance_images"))
			count := testutil.CollectAndCount(exporter, "openstack_glance_image_info")
			suite.Equal(tc.info, count)
		})
	}
}
