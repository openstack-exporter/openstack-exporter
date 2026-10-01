package exporters

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/jarcoal/httpmock"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

type DesignateTestSuite struct {
	BaseOpenStackTestSuite
}

var designateExpectedUp = `
# HELP openstack_designate_recordsets recordsets
# TYPE openstack_designate_recordsets gauge
openstack_designate_recordsets{tenant_id="4335d1f0-f793-11e2-b778-0800200c9a66",zone_id="a86dba58-0043-4cc6-a1bb-69d5e86f3ca3",zone_name="example.org."} 1
# HELP openstack_designate_recordsets_status recordsets_status
# TYPE openstack_designate_recordsets_status gauge
openstack_designate_recordsets_status{id="f7b10e9b-0cae-4a91-b162-562bc6096648",name="example.org.",status="PENDING",type="A",zone_id="a86dba58-0043-4cc6-a1bb-69d5e86f3ca3",zone_name="example.org."} 0
# HELP openstack_designate_up up
# TYPE openstack_designate_up gauge
openstack_designate_up 1
# HELP openstack_designate_zone_status zone_status
# TYPE openstack_designate_zone_status gauge
openstack_designate_zone_status{id="a86dba58-0043-4cc6-a1bb-69d5e86f3ca3",name="example.org.",status="ACTIVE",tenant_id="4335d1f0-f793-11e2-b778-0800200c9a66",type="PRIMARY"} 1
# HELP openstack_designate_zones zones
# TYPE openstack_designate_zones gauge
openstack_designate_zones 1
`

func (suite *DesignateTestSuite) TestDesignateExporter() {
	err := testutil.CollectAndCompare(*suite.Exporter, strings.NewReader(designateExpectedUp))
	assert.NoError(suite.T(), err)
}

func (suite *DesignateTestSuite) TestDesignateRecordsetPagination() {
	exporter := (*suite.Exporter).(*DesignateExporter)
	exporter.DesignateRecordsetLimit = 1

	httpmock.RegisterResponder("GET", suite.MakeURL("/designate/v2/zones", ""), httpmock.NewJsonResponderOrPanic(200, json.RawMessage(`{
  "zones": [
   {"id":"a", "name":"a.example.", "project_id":"project"},
   {"id":"b", "name":"b.example.", "project_id":"project"},
   {"id":"empty", "name":"empty.example.", "project_id":"project"}
  ]
 }`)))
	firstURL := suite.MakeURL("/designate/v2/recordsets?limit=1", "")
	nextURL := suite.MakeURL("/designate/v2/recordsets?limit=1&marker=first", "")
	httpmock.RegisterResponder("GET", firstURL, func(req *http.Request) (*http.Response, error) {
		suite.Equal("True", req.Header.Get("X-Auth-All-Projects"))
		response, err := httpmock.NewJsonResponse(200, map[string]any{
			"recordsets": []map[string]string{{"id": "first", "zone_id": "a", "status": "ACTIVE"}},
			"links":      map[string]string{"next": nextURL},
		})
		response.Request = req
		return response, err
	})
	httpmock.RegisterResponder("GET", nextURL, func(req *http.Request) (*http.Response, error) {
		suite.Equal("True", req.Header.Get("X-Auth-All-Projects"))
		response, err := httpmock.NewJsonResponse(200, json.RawMessage(`{
   "recordsets": [
    {"id":"second", "zone_id":"b", "status":"ACTIVE"},
    {"id":"third", "zone_id":"a", "status":"ACTIVE"}
   ], "links": {}
  }`))
		response.Request = req
		return response, err
	})

	err := testutil.CollectAndCompare(*suite.Exporter, strings.NewReader(`
# HELP openstack_designate_recordsets recordsets
# TYPE openstack_designate_recordsets gauge
openstack_designate_recordsets{tenant_id="project",zone_id="a",zone_name="a.example."} 2
openstack_designate_recordsets{tenant_id="project",zone_id="b",zone_name="b.example."} 1
openstack_designate_recordsets{tenant_id="project",zone_id="empty",zone_name="empty.example."} 0
# HELP openstack_designate_up up
# TYPE openstack_designate_up gauge
openstack_designate_up 1
`), "openstack_designate_recordsets", "openstack_designate_up")
	suite.Require().NoError(err)
	calls := httpmock.GetCallCountInfo()
	suite.Equal(1, calls["GET "+firstURL])
	suite.Equal(1, calls["GET "+nextURL])
}

func (suite *DesignateTestSuite) TestDesignateRecordsetError() {
	httpmock.RegisterResponder("GET", suite.MakeURL("/designate/v2/recordsets?limit=1000", ""), httpmock.NewStringResponder(500, ""))
	err := testutil.CollectAndCompare(*suite.Exporter, strings.NewReader(`
# HELP openstack_designate_up up
# TYPE openstack_designate_up gauge
openstack_designate_up 0
`), "openstack_designate_up")
	suite.Require().NoError(err)
}
