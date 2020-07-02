package exporters

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/suite"
)

const baseFixturePath = "./fixtures"
const cloudName = "test.cloud"

type BaseOpenStackTestSuite struct {
	suite.Suite
	ServiceName string
	Prefix      string
	Exporter    *OpenStackExporter
}

func (suite *BaseOpenStackTestSuite) SetResponseFromFixture(method string, statusCode int, url string, file string) {
	data, _ := ioutil.ReadFile(file)
	response := &http.Response{
		Body: ioutil.NopCloser(bytes.NewReader(data)),
		Header: http.Header{
			"Content-Type":    []string{"application/json"},
			"X-Subject-Token": []string{"1234"},
		},
		StatusCode: statusCode,
	}

	responder := httpmock.ResponderFromResponse(response).Once()
	httpmock.RegisterResponder(method, url, responder)
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

var fixtures map[string]string = map[string]string{
	"/container-infra/clusters":                                  "container_infra_clusters",
	"/compute/":                                                  "nova_api_discovery",
	"/compute/os-services":                                       "nova_os_services",
	"/compute/os-hypervisors/detail":                             "nova_os_hypervisors",
	"/compute/flavors/detail":                                    "nova_os_flavors",
	"/compute/os-availability-zone":                              "nova_os_availability_zones",
	"/compute/os-security-groups":                                "nova_os_security_groups",
	"/compute/os-aggregates":                                     "nova_os_aggregates",
	"/compute/limits?tenant_id=0c4e939acacf4376bdcd1129f1a054ad": "nova_os_limits",
	"/compute/limits?tenant_id=0cbd49cbf76d405d9c86562e1d579bd3": "nova_os_limits",
	"/compute/limits?tenant_id=2db68fed84324f29bb73130c6c2094fb": "nova_os_limits",
	"/compute/limits?tenant_id=3d594eb0f04741069dbbb521635b21c7": "nova_os_limits",
	"/compute/limits?tenant_id=43ebde53fc314b1c9ea2b8c5dc744927": "nova_os_limits",
	"/compute/limits?tenant_id=4b1eb781a47440acb8af9850103e537f": "nova_os_limits",
	"/compute/limits?tenant_id=5961c443439d4fcebe42643723755e9d": "nova_os_limits",
	"/compute/limits?tenant_id=fdb8424c4e4f4c0ba32c52e2de3bd80e": "nova_os_limits",
	"/compute/servers/detail?all_tenants=true":                   "nova_os_servers",
	"/glance/":                                       "glance_api_discovery",
	"/glance/v2/images":                              "glance_images",
	"/identity/v3/projects":                          "identity_projects",
	"/neutron/":                                      "neutron_api_discovery",
	"/neutron/v2.0/floatingips":                      "neutron_floating_ips",
	"/neutron/v2.0/agents":                           "neutron_agents",
	"/neutron/v2.0/networks":                         "neutron_networks",
	"/neutron/v2.0/security-groups":                  "neutron_security_groups",
	"/neutron/v2.0/subnets":                          "neutron_subnets",
	"/neutron/v2.0/ports":                            "neutron_ports",
	"/neutron/v2.0/network-ip-availabilities":        "neutron_network_ip_availabilities",
	"/neutron/v2.0/routers":                          "neutron_routers",
	"/neutron/v2.0/lbaas/loadbalancers":              "neutron_loadbalancers",
	"/volumes":                                       "cinder_api_discovery",
	"/volumes/volumes/detail?all_tenants=true":       "cinder_volumes",
	"/volumes/snapshots":                             "cinder_snapshots",
	"/volumes/os-services":                           "cinder_os_services",
	"/volumes/scheduler-stats/get_pools?detail=true": "cinder_scheduler_stats_pools",
	"/volumes/os-quota-sets/0c4e939acacf4376bdcd1129f1a054ad?usage=true": "cinder_os_quota_sets",
	"/volumes/os-quota-sets/0cbd49cbf76d405d9c86562e1d579bd3?usage=true": "cinder_os_quota_sets",
	"/volumes/os-quota-sets/2db68fed84324f29bb73130c6c2094fb?usage=true": "cinder_os_quota_sets",
	"/volumes/os-quota-sets/3d594eb0f04741069dbbb521635b21c7?usage=true": "cinder_os_quota_sets",
	"/volumes/os-quota-sets/43ebde53fc314b1c9ea2b8c5dc744927?usage=true": "cinder_os_quota_sets",
	"/volumes/os-quota-sets/4b1eb781a47440acb8af9850103e537f?usage=true": "cinder_os_quota_sets",
	"/volumes/os-quota-sets/5961c443439d4fcebe42643723755e9d?usage=true": "cinder_os_quota_sets",
	"/volumes/os-quota-sets/fdb8424c4e4f4c0ba32c52e2de3bd80e?usage=true": "cinder_os_quota_sets",
	"/designate/v2/zones": "designate_zones",
	"/designate/v2/zones/a86dba58-0043-4cc6-a1bb-69d5e86f3ca3/recordsets": "designate_recordsets",
}

func (suite *BaseOpenStackTestSuite) SetupTest() {
	httpmock.Activate()
	suite.Prefix = "openstack"

	suite.teardownFixtures()
	suite.installFixtures()

	os.Setenv("OS_CLIENT_CONFIG_FILE", path.Join(baseFixturePath, "test_config.yaml"))

	exporter, err := NewExporter(suite.ServiceName, suite.Prefix, cloudName, []string{}, "public", false)
	if err != nil {
		panic(err)
	}
	suite.Exporter = &exporter
}

func (suite *BaseOpenStackTestSuite) teardownFixtures() {
	httpmock.Reset()
	suite.SetResponseFromFixture("POST", 201,
		suite.MakeURL("/v3/auth/tokens", "35357"),
		suite.FixturePath("tokens"),
	)
}

func (suite *BaseOpenStackTestSuite) installFixtures() {
	for path, fixture := range fixtures {
		suite.SetResponseFromFixture("GET", 200,
			suite.MakeURL(path, ""),
			suite.FixturePath(fixture),
		)
	}

	// NOTE(mnaser): The following makes sure that all requests are mocked
	//               and any un-mocked requests will fail to ensure we have
	//               full coverage.
	httpmock.RegisterNoResponder(
		func(req *http.Request) (*http.Response, error) {
			msg := fmt.Sprintf("Unmocked request: %s", req.URL.RequestURI())
			suite.T().Error(errors.New(msg))
			return httpmock.NewStringResponse(500, ""), nil
		},
	)
}

func (suite *BaseOpenStackTestSuite) TearDownTest() {
	defer httpmock.DeactivateAndReset()
}

func TestOpenStackSuites(t *testing.T) {
	suite.Run(t, &CinderTestSuite{BaseOpenStackTestSuite: BaseOpenStackTestSuite{ServiceName: "volume"}})
	suite.Run(t, &NovaTestSuite{BaseOpenStackTestSuite: BaseOpenStackTestSuite{ServiceName: "compute"}})
	suite.Run(t, &NeutronTestSuite{BaseOpenStackTestSuite: BaseOpenStackTestSuite{ServiceName: "network"}})
	suite.Run(t, &GlanceTestSuite{BaseOpenStackTestSuite: BaseOpenStackTestSuite{ServiceName: "image"}})
	suite.Run(t, &ContainerInfraTestSuite{BaseOpenStackTestSuite: BaseOpenStackTestSuite{ServiceName: "container-infra"}})
	suite.Run(t, &DesignateTestSuite{BaseOpenStackTestSuite: BaseOpenStackTestSuite{ServiceName: "dns"}})
}
