package exporters

import (
	"strings"

	"github.com/jarcoal/httpmock"
	"github.com/prometheus/client_golang/prometheus"

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
		name     string
		disabled []string
		empty    bool
	}{
		{name: "populated catalog"},
		{name: "empty catalog", empty: true},
		{name: "count disabled", disabled: []string{"glance-images"}},
		{name: "info disabled", disabled: []string{"glance-image_info"}},
	} {
		suite.Run(tc.name, func() {
			if tc.empty {
				httpmock.RegisterResponder("GET", suite.MakeURL("/glance/v2/images", ""),
					httpmock.NewJsonResponderOrPanic(200, map[string]interface{}{"images": []interface{}{}}))
			} else {
				suite.SetResponseFromFixture("GET", 200, suite.MakeURL("/glance/v2/images", ""), suite.FixturePath("glance_images"))
			}
			base := (*suite.Exporter).(*GlanceExporter)
			config := base.ExporterConfig
			config.DisabledMetrics = tc.disabled
			config.DisableSlowMetrics = true
			exporter, err := NewGlanceExporter(&config, base.logger)
			suite.Require().NoError(err)
			registry := prometheus.NewRegistry()
			suite.Require().NoError(registry.Register(exporter))
			families, err := registry.Gather()
			suite.Require().NoError(err)
			var count, infoSum float64
			var countSamples, infoSamples int
			for _, family := range families {
				switch family.GetName() {
				case "openstack_glance_images":
					countSamples = len(family.Metric)
					for _, metric := range family.Metric {
						count += metric.GetGauge().GetValue()
						suite.Empty(metric.Label)
					}
				case "openstack_glance_image_info":
					infoSamples = len(family.Metric)
					for _, metric := range family.Metric {
						infoSum += metric.GetGauge().GetValue()
						suite.Equal(float64(1), metric.GetGauge().GetValue())
					}
				}
			}
			wantCount := float64(2)
			if tc.empty {
				wantCount = 0
			}
			if exporter.MetricIsDisabled("images") {
				suite.Zero(countSamples)
			} else {
				suite.Equal(1, countSamples)
				suite.Equal(wantCount, count)
			}
			if exporter.MetricIsDisabled("image_info") {
				suite.Zero(infoSamples)
			} else {
				suite.Equal(int(wantCount), infoSamples)
				suite.Equal(wantCount, infoSum)
			}
		})
	}
}
