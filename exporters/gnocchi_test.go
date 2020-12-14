package exporters

// This test does not work because httpmock cannot emulate gnocchi api (/v1/metric) Pagination properly.
// Refer: https://gnocchi.xyz/rest.html#pagination

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

type GnocchiTestSuite struct {
	BaseOpenStackTestSuite
}

var gnocchiExpectedUp = `
# HELP openstack_gnocchi_status_measures_to_process status_measures_to_process
# TYPE openstack_gnocchi_status_measures_to_process gauge
openstack_gnocchi_status_measures_to_process 0
# HELP openstack_gnocchi_status_metric_having_measures_to_process status_metric_having_measures_to_process
# TYPE openstack_gnocchi_status_metric_having_measures_to_process gauge
openstack_gnocchi_status_metric_having_measures_to_process 0
# HELP openstack_gnocchi_status_metricd_processors status_metricd_processors
# TYPE openstack_gnocchi_status_metricd_processors gauge
openstack_gnocchi_status_metricd_processors 0
# HELP openstack_gnocchi_up up
# TYPE openstack_gnocchi_up gauge
openstack_gnocchi_up 1
`

func (suite *GnocchiTestSuite) TestGnocchiExporter() {
	err := testutil.CollectAndCompare(*suite.Exporter, strings.NewReader(gnocchiExpectedUp))
	assert.NoError(suite.T(), err)
}
