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
	"os"
	"path"
	"testing"
)

const baseFixturePath = "./fixtures"
const cloudName = "test.cloud"

type BaseOpenStackTestSuite struct {
	suite.Suite
	ServiceName string
	Prefix      string
	Exporter    *OpenStackExporter
	Recorder    *httptest.ResponseRecorder
}

func (suite *BaseOpenStackTestSuite) StartMetricsHandler() {
	router := mux.NewRouter()
	router.Handle("/metrics", promhttp.Handler())
	req, _ := http.NewRequest("GET", "/metrics", nil)
	suite.Recorder = httptest.NewRecorder()
	router.ServeHTTP(suite.Recorder, req)
}

func (suite *BaseOpenStackTestSuite) SetResponseFromFixture(method string, statusCode int, url string, file string) {
	httpmock.RegisterResponder(method, url, func(request *http.Request) (*http.Response, error) {
		data, _ := ioutil.ReadFile(file)
		return &http.Response{
			Body: ioutil.NopCloser(bytes.NewReader(data)),
			Header: http.Header{
				"Content-Type":    []string{"application/json"},
				"X-Subject-Token": []string{"1234"},
			},
			StatusCode: statusCode,
			Request:    request,
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
	httpmock.Activate()
	suite.Prefix = "openstack"
	suite.SetResponseFromFixture("POST", 201,
		suite.MakeURL("/v3/auth/tokens", "35357"),
		suite.FixturePath("tokens"))
}

func (suite *BaseOpenStackTestSuite) TearDownSuite() {
	defer httpmock.DeactivateAndReset()
}

func (suite *BaseOpenStackTestSuite) SetupTest() {
	os.Setenv("OS_CLIENT_CONFIG_FILE", path.Join(baseFixturePath, "test_config.yaml"))
	exporter, err := EnableExporter(suite.ServiceName, suite.Prefix, cloudName)
	if err != nil {
		panic(err)
	}
	suite.Exporter = exporter
}

func (suite *BaseOpenStackTestSuite) TearDownTest() {
	prometheus.Unregister(*suite.Exporter)
}

type NovaTestSuite struct {
	BaseOpenStackTestSuite
}

func (suite *NovaTestSuite) TestNovaExporter() {
	suite.SetResponseFromFixture("GET", 200,
		suite.MakeURL("/compute/", ""),
		suite.FixturePath("nova_api_discovery"))

	suite.SetResponseFromFixture("GET", 200,
		suite.MakeURL("/compute/os-services", ""),
		suite.FixturePath("nova_os_services"),
	)
	suite.SetResponseFromFixture("GET", 200,
		suite.MakeURL("/compute/os-hypervisors/detail", ""),
		suite.FixturePath("nova_os_hypervisors"),
	)
	suite.SetResponseFromFixture("GET", 200,
		suite.MakeURL("/compute/flavors/detail", ""),
		suite.FixturePath("nova_os_flavors"),
	)
	suite.SetResponseFromFixture("GET", 200,
		suite.MakeURL("/compute/os-availability-zone", ""),
		suite.FixturePath("nova_os_availability_zones"),
	)
	suite.SetResponseFromFixture("GET", 200,
		suite.MakeURL("/compute/os-security-groups", ""),
		suite.FixturePath("nova_os_security_groups"),
	)
	suite.SetResponseFromFixture("GET", 200,
		suite.MakeURL("/compute/servers/detail?all_tenants=true", ""),
		suite.FixturePath("nova_os_servers"),
	)

	suite.StartMetricsHandler()

	for _, metric := range defaultNovaMetrics {
		suite.Contains(suite.Recorder.Body.String(), "nova_"+metric.Name)
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
	suite.SetResponseFromFixture("GET", 200,
		suite.MakeURL("/neutron/v2.0/ports", ""),
		suite.FixturePath("neutron_ports"),
	)

	suite.StartMetricsHandler()

	//Check that all the default metrics are contained on the response
	for _, metric := range defaultNeutronMetrics {
		suite.Contains(suite.Recorder.Body.String(), "neutron_"+metric.Name)
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
		suite.MakeURL("/glance/v2/images", ""),
		suite.FixturePath("glance_images"),
	)

	suite.StartMetricsHandler()

	//Check that all the default metrics are contained on the response
	for _, metric := range defaultGlanceMetrics {
		suite.Contains(suite.Recorder.Body.String(), "glance_"+metric.Name)
	}
}

type CinderTestSuite struct {
	BaseOpenStackTestSuite
}

func (suite *CinderTestSuite) TestCinderExporter() {
	suite.SetResponseFromFixture("GET", 200,
		suite.MakeURL("/volumes/", ""),
		suite.FixturePath("cinder_api_discovery"),
	)
	suite.SetResponseFromFixture("GET", 200,
		suite.MakeURL("/volumes/volumes/detail?all_tenants=true", ""),
		suite.FixturePath("cinder_volumes"),
	)

	suite.SetResponseFromFixture("GET", 200,
		suite.MakeURL("/volumes/snapshots", ""),
		suite.FixturePath("cinder_snapshots"),
	)

	suite.SetResponseFromFixture("GET", 200,
		suite.MakeURL("/volumes/os-services", ""),
		suite.FixturePath("cinder_os_services"),
	)
	suite.StartMetricsHandler()

	//Check that all the default metrics are contained on the response
	for _, metric := range defaultCinderMetrics {
		suite.Contains(suite.Recorder.Body.String(), "cinder_"+metric.Name)
	}
}

func TestOpenStackSuites(t *testing.T) {
	suite.Run(t, &CinderTestSuite{BaseOpenStackTestSuite: BaseOpenStackTestSuite{ServiceName: "volume"}})
	suite.Run(t, &NovaTestSuite{BaseOpenStackTestSuite: BaseOpenStackTestSuite{ServiceName: "compute"}})
	suite.Run(t, &NeutronTestSuite{BaseOpenStackTestSuite: BaseOpenStackTestSuite{ServiceName: "network"}})
	suite.Run(t, &GlanceTestSuite{BaseOpenStackTestSuite: BaseOpenStackTestSuite{ServiceName: "image"}})
}
