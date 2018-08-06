package main

import (
	"bytes"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/jarcoal/httpmock"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/suite"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

const testCloudConfig = `
clouds:
 test.cloud:
   region_name: RegionOne
   identity_api_version: 3
   identity_interface: internal
   auth: 
     username: 'admin'
     password: 'admin'
     project_name: 'admin'
     project_domain_name: 'Default'
     user_domain_name: 'Default'
     auth_url: 'http://test.cloud:35357/v3'
`
const baseFixturePath = "./fixtures"
const cloudName = "test.cloud"

type BaseOpenStackTestSuite struct {
	suite.Suite
	Config      *Cloud
	ServiceName string
	Exporter    *OpenStackExporter
}

func (suite *BaseOpenStackTestSuite) SetResponseFromFixture(method string, statusCode int, url string, file string) {
	httpmock.RegisterResponder(method, url, func(request *http.Request) (*http.Response, error) {
		data, _ := ioutil.ReadFile(file)
		return &http.Response{
			Body:       ioutil.NopCloser(bytes.NewReader(data)),
			Header:     http.Header{"X-Subject-Token": []string{"1234"}},
			StatusCode: statusCode,
		}, nil
	})
}

func (suite *BaseOpenStackTestSuite) MakeURL(resource string, port string) string {
	if port != "" {
		return fmt.Sprintf("http://%s:%s%s", cloudName, port, resource)
	}
	return fmt.Sprintf("http://%s%s", cloudName, resource)
}

func (suite *BaseOpenStackTestSuite) FixturePath(name string) string {
	return fmt.Sprintf("%s/%s", baseFixturePath, name+".json")
}

func (suite *BaseOpenStackTestSuite) SetupSuite() {
	config, _ := NewCloudConfigFromByteArray([]byte(testCloudConfig))
	cloudConfig, _ := config.GetByName(cloudName)

	httpmock.Activate()
	suite.Config = cloudConfig
	suite.SetResponseFromFixture("POST", 201,
		suite.MakeURL("/v3/auth/tokens", "35357"),
		suite.FixturePath("tokens"))
}

func (suite *BaseOpenStackTestSuite) TearDownSuite() {
	defer httpmock.DeactivateAndReset()
}

func (suite *BaseOpenStackTestSuite) SetupTest() {
	exporter, _ := EnableExporter(suite.ServiceName, suite.Config)
	suite.Exporter = exporter
}

func (suite *BaseOpenStackTestSuite) TearDownTest() {
	prometheus.Unregister(*suite.Exporter)
}

func (suite *BaseOpenStackTestSuite) TestConfig() {
	suite.NotNil(suite.Config)
}

type NovaTestSuite struct {
	BaseOpenStackTestSuite
}

func (suite *NovaTestSuite) TestNovaExporter() {
	suite.SetResponseFromFixture("GET", 200,
		suite.MakeURL("/compute/", ""),
		suite.FixturePath("nova_api_discovery"))

	suite.SetResponseFromFixture("GET", 200,
		suite.MakeURL("/compute/v2.1/os-services", ""),
		suite.FixturePath("nova_os_services"),
	)
	suite.SetResponseFromFixture("GET", 200,
		suite.MakeURL("/compute/v2.1/os-hypervisors/detail", ""),
		suite.FixturePath("nova_os_hypervisors"),
	)
	suite.SetResponseFromFixture("GET", 200,
		suite.MakeURL("/compute/v2.1/flavors", ""),
		suite.FixturePath("nova_os_flavors"),
	)
	suite.SetResponseFromFixture("GET", 200,
		suite.MakeURL("/compute/v2.1/os-availability-zone", ""),
		suite.FixturePath("nova_os_availability_zones"),
	)
	suite.SetResponseFromFixture("GET", 200,
		suite.MakeURL("/compute/v2.1/os-security-groups", ""),
		suite.FixturePath("nova_os_security_groups"),
	)
	suite.SetResponseFromFixture("GET", 200,
		suite.MakeURL("/compute/v2.1/servers?all_tenants=1", ""),
		suite.FixturePath("nova_os_servers"),
	)

	router := mux.NewRouter()
	router.Handle("/metrics", promhttp.Handler())

	req, _ := http.NewRequest("GET", "/metrics", nil)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	//Check that all the default metrics are contained on the response
	for _, metric := range defaultNovaMetrics {
		suite.Contains(res.Body.String(), "nova_"+metric.Name)
	}
}

type NeutronTestSuite struct {
	BaseOpenStackTestSuite
}

func (suite *NeutronTestSuite) TestNeutronExporter() {

	suite.SetResponseFromFixture("GET", 200,
		suite.MakeURL("/neutron/", ""),
		suite.FixturePath("neutron_api_discovery"),
	)
	suite.SetResponseFromFixture("GET", 200,
		suite.MakeURL("/neutron/v2.0/floatingips", ""),
		suite.FixturePath("neutron_floating_ips"),
	)
	suite.SetResponseFromFixture("GET", 200,
		suite.MakeURL("/neutron/v2.0/agents", ""),
		suite.FixturePath("neutron_agents"),
	)
	suite.SetResponseFromFixture("GET", 200,
		suite.MakeURL("/neutron/v2.0/networks", ""),
		suite.FixturePath("neutron_networks"),
	)
	suite.SetResponseFromFixture("GET", 200,
		suite.MakeURL("/neutron/v2.0/security-groups", ""),
		suite.FixturePath("neutron_security_groups"),
	)
	suite.SetResponseFromFixture("GET", 200,
		suite.MakeURL("/neutron/v2.0/subnets", ""),
		suite.FixturePath("neutron_subnets"),
	)

	router := mux.NewRouter()
	router.Handle("/metrics", promhttp.Handler())

	req, _ := http.NewRequest("GET", "/metrics", nil)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	//Check that all the default metrics are contained on the response
	for _, metric := range defaultNeutronMetrics {
		suite.Contains(res.Body.String(), "neutron_"+metric.Name)
	}
}

type GlanceTestSuite struct {
	BaseOpenStackTestSuite
}

func (suite *GlanceTestSuite) TestGlanceExporter() {

	suite.SetResponseFromFixture("GET", 200,
		suite.MakeURL("/glance/", ""),
		suite.FixturePath("glance_api_discovery"),
	)
	suite.SetResponseFromFixture("GET", 200,
		suite.MakeURL("/glance/v2//images", ""),
		suite.FixturePath("glance_images"),
	)

	router := mux.NewRouter()
	router.Handle("/metrics", promhttp.Handler())
	req, _ := http.NewRequest("GET", "/metrics", nil)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	//Check that all the default metrics are contained on the response
	for _, metric := range defaultGlanceMetrics {
		suite.Contains(res.Body.String(), "glance_"+metric.Name)
	}
}

//type CinderTestSuite struct {
//	BaseOpenStackTestSuite
//}
//
//func (suite *CinderTestSuite) TestCinderExporter() {
//
//	router := mux.NewRouter()
//	router.Handle("/metrics", promhttp.Handler())
//	req, _ := http.NewRequest("GET", "/metrics", nil)
//	res := httptest.NewRecorder()
//	router.ServeHTTP(res, req)
//
//	//Check that all the default metrics are contained on the response
//	for _, metric := range defaultCinderMetrics {
//		suite.Contains(res.Body.String(), "cinder_" + metric.Name)
//	}
//}
//

func TestOpenStackSuites(t *testing.T) {
	suite.Run(t, &NovaTestSuite{BaseOpenStackTestSuite: BaseOpenStackTestSuite{ServiceName: "compute"}})
	suite.Run(t, &NeutronTestSuite{BaseOpenStackTestSuite: BaseOpenStackTestSuite{ServiceName: "network"}})
	suite.Run(t, &GlanceTestSuite{BaseOpenStackTestSuite: BaseOpenStackTestSuite{ServiceName: "image"}})
}
