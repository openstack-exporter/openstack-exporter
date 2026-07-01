package exporters

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"log/slog"

	"github.com/jarcoal/httpmock"
	"github.com/openstack-exporter/openstack-exporter/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/suite"
)

// metricNamesFrom extracts the unique metric family names from an expected
// prometheus text-format string. Pass the result to testutil.CollectAndCompare
// so only those families are compared, ignoring new instrumentation metrics.
var metricNameRe = regexp.MustCompile(`(?m)^# HELP (\S+)`)

func metricNamesFrom(expected string) []string {
	var names []string
	for _, m := range metricNameRe.FindAllStringSubmatch(expected, -1) {
		names = append(names, m[1])
	}
	return names
}

const baseFixturePath = "./fixtures"
const cloudName = "test.cloud"

type BaseOpenStackTestSuite struct {
	suite.Suite
	ServiceName string
	Prefix      string
	Exporter    OpenStackExporter
}

func (suite *BaseOpenStackTestSuite) SetResponseFromFixture(method string, statusCode int, url string, file string) {
	data, _ := os.ReadFile(file)
	response := &http.Response{
		Body: httpmock.NewRespBodyFromBytes(data),
		Header: http.Header{
			"Content-Type":    []string{"application/json"},
			"X-Subject-Token": []string{"1234"},
		},
		StatusCode: statusCode,
	}

	responder := httpmock.ResponderFromResponse(response).Times(2)
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
	"/container-infra/":              "container_infra_api_discovery",
	"/container-infra/clusters":      "container_infra_clusters",
	"/compute/":                      "nova_api_discovery",
	"/compute/v2.1/":                 "nova_api_v2.1",
	"/compute/os-services":           "nova_os_services",
	"/compute/os-hypervisors/detail": "nova_os_hypervisors",
	"/compute/flavors/detail":        "nova_os_flavors",
	"/compute/os-availability-zone":  "nova_os_availability_zones",
	"/compute/os-security-groups":    "nova_os_security_groups",
	"/compute/os-aggregates":         "nova_os_aggregates",
	"/compute/limits?tenant_id=0c4e939acacf4376bdcd1129f1a054ad": "nova_os_limits",
	"/compute/limits?tenant_id=0cbd49cbf76d405d9c86562e1d579bd3": "nova_os_limits",
	"/compute/limits?tenant_id=2db68fed84324f29bb73130c6c2094fb": "nova_os_limits",
	"/compute/limits?tenant_id=3d594eb0f04741069dbbb521635b21c7": "nova_os_limits",
	"/compute/limits?tenant_id=43ebde53fc314b1c9ea2b8c5dc744927": "nova_os_limits",
	"/compute/limits?tenant_id=4b1eb781a47440acb8af9850103e537f": "nova_os_limits",
	"/compute/limits?tenant_id=5961c443439d4fcebe42643723755e9d": "nova_os_limits",
	"/compute/limits?tenant_id=fdb8424c4e4f4c0ba32c52e2de3bd80e": "nova_os_limits",
	"/compute/servers/detail?all_tenants=true":                   "nova_os_servers",
	"/compute/os-simple-tenant-usage?detailed=1":                 "nova_os_simple_tenant_usage",
	"/glance/":          "glance_api_discovery",
	"/glance/v2/images": "glance_images",
	"/gnocchi/v1/metric?marker=5e9b3ee0-aee1-4461-8849-3f4ae5e30d8d": "gnocchi_empty",
	"/gnocchi/v1/metric":                         "gnocchi_metric",
	"/gnocchi/v1/status":                         "gnocchi_status",
	"/gnocchi/v1/status?details=true":            "gnocchi_status",
	"/identity/v3/projects":                      "identity_projects",
	"/identity/v3/domains":                       "identity_domains",
	"/identity/v3/users":                         "identity_users",
	"/identity/v3/groups":                        "identity_groups",
	"/identity/v3/regions":                       "identity_regions",
	"/neutron/":                                  "neutron_api_discovery",
	"/neutron/v2.0/floatingips":                  "neutron_floating_ips",
	"/neutron/v2.0/agents":                       "neutron_agents",
	"/neutron/v2.0/networks":                     "neutron_networks",
	"/neutron/v2.0/security-groups":              "neutron_security_groups",
	"/neutron/v2.0/subnets":                      "neutron_subnets",
	"/neutron/v2.0/subnetpools":                  "neutron_subnet_pools",
	"/neutron/v2.0/ports":                        "neutron_ports",
	"/neutron/v2.0/network-ip-availabilities":    "neutron_network_ip_availabilities",
	"/neutron/v2.0/routers":                      "neutron_routers",
	"/neutron/v2.0/agents?binary=ovn-controller": "neutron_ovn_controller_agents",
	"/neutron/v2.0/routers/f8a44de0-fc8e-45df-93c7-f79bf3b01c95/l3-agents": "neutron_routers_l3_agents",
	"/neutron/v2.0/routers/9daeb7dd-7e3f-4e44-8c42-c7a0e8c8a42f/l3-agents": "neutron_routers_l3_agents",
	"/neutron/v2.0/extensions/vpnaas":                                      "neutron_vpnaas_extension",
	"/neutron/v2.0/vpn/endpoint-groups":                                    "neutron_vpn_endpoint_groups",
	"/neutron/v2.0/vpn/ipsecpolicies":                                      "neutron_ipsecpolicies",
	"/neutron/v2.0/vpn/vpnservices":                                        "neutron_vpnservices",
	"/neutron/v2.0/vpn/ipsec-site-connections":                             "neutron_ipsec_site_connections",
	"/neutron/v2.0/vpn/ikepolicies":                                        "neutron_ikepolicies",
	"/loadbalancer/":                                                       "loadbalancer_api_discovery",
	"/loadbalancer/v2.0/lbaas/loadbalancers":                               "loadbalancer_loadbalancers",
	"/loadbalancer/v2.0/octavia/amphorae":                                  "loadbalancer_amphorae",
	"/loadbalancer/v2.0/lbaas/pools":                                       "loadbalancer_pools",
	"/ironic/":                                                             "ironic_api_discovery",
	"/ironic/v1":                                                           "ironic_v1",
	"/ironic/v1/nodes":                                                     "ironic_nodes",
	"/ironic/v1/nodes/detail":                                              "ironic_nodes",
	"/volumes":                                                             "cinder_api_discovery",
	"/volumes/":                                                            "cinder_api_discovery",
	"/volumes/volumes/detail?all_tenants=true":                             "cinder_volumes",
	"/volumes/snapshots":                                                   "cinder_snapshots",
	"/volumes/os-services":                                                 "cinder_os_services",
	"/volumes/scheduler-stats/get_pools?detail=true":                       "cinder_scheduler_stats_pools",
	"/volumes/os-quota-sets/0c4e939acacf4376bdcd1129f1a054ad?usage=true": "cinder_os_quota_sets_usage",
	"/volumes/os-quota-sets/0cbd49cbf76d405d9c86562e1d579bd3?usage=true": "cinder_os_quota_sets_usage",
	"/volumes/os-quota-sets/2db68fed84324f29bb73130c6c2094fb?usage=true": "cinder_os_quota_sets_usage",
	"/volumes/os-quota-sets/3d594eb0f04741069dbbb521635b21c7?usage=true": "cinder_os_quota_sets_usage",
	"/volumes/os-quota-sets/43ebde53fc314b1c9ea2b8c5dc744927?usage=true": "cinder_os_quota_sets_usage",
	"/volumes/os-quota-sets/4b1eb781a47440acb8af9850103e537f?usage=true": "cinder_os_quota_sets_usage",
	"/volumes/os-quota-sets/5961c443439d4fcebe42643723755e9d?usage=true": "cinder_os_quota_sets_usage",
	"/volumes/os-quota-sets/fdb8424c4e4f4c0ba32c52e2de3bd80e?usage=true": "cinder_os_quota_sets_usage",
	"/volumes/os-quota-sets/0c4e939acacf4376bdcd1129f1a054ad":            "cinder_os_quota_sets",
	"/volumes/os-quota-sets/0cbd49cbf76d405d9c86562e1d579bd3":            "cinder_os_quota_sets",
	"/volumes/os-quota-sets/2db68fed84324f29bb73130c6c2094fb":            "cinder_os_quota_sets",
	"/volumes/os-quota-sets/3d594eb0f04741069dbbb521635b21c7":            "cinder_os_quota_sets",
	"/volumes/os-quota-sets/43ebde53fc314b1c9ea2b8c5dc744927":            "cinder_os_quota_sets",
	"/volumes/os-quota-sets/4b1eb781a47440acb8af9850103e537f":            "cinder_os_quota_sets",
	"/volumes/os-quota-sets/5961c443439d4fcebe42643723755e9d":            "cinder_os_quota_sets",
	"/volumes/os-quota-sets/fdb8424c4e4f4c0ba32c52e2de3bd80e":            "cinder_os_quota_sets",
	"/designate/":         "designate_api_discovery",
	"/designate/v2/zones": "designate_zones",
	"/designate/v2/zones/a86dba58-0043-4cc6-a1bb-69d5e86f3ca3/recordsets": "designate_recordsets",
	"/database/": "trove_api_discovery",
	"/database/mgmt/instances?include_clustered=False&deleted=False": "trove_instances",
	"/orchestration/":               "heat_api_discovery",
	"/orchestration/stacks":         "heat_stacks",
	"/placement/":                   "placement_api_discovery",
	"/placement/resource_providers": "resource_providers",
	"/placement/resource_providers/b985be15-99bf-4baf-9ef7-3ef166cd7f31/inventories": "resource_provider_1_inventory",
	"/placement/resource_providers/328c9f0a-5a3c-4ad6-9347-689eb7632d7b/inventories": "resource_provider_2_inventory",
	"/placement/resource_providers/b985be15-99bf-4baf-9ef7-3ef166cd7f31/usages":      "resource_provider_1_usage",
	"/placement/resource_providers/328c9f0a-5a3c-4ad6-9347-689eb7632d7b/usages":      "resource_provider_2_usage",
	"/placement/resource_providers/b985be15-99bf-4baf-9ef7-3ef166cd7f31/allocations": "resource_provider_1_allocations",
	"/placement/resource_providers/328c9f0a-5a3c-4ad6-9347-689eb7632d7b/allocations": "resource_provider_2_allocations",
	"/compute/os-quota-sets/0c4e939acacf4376bdcd1129f1a054ad/detail":                 "nova_quotas_1_usage",
	"/compute/os-quota-sets/0cbd49cbf76d405d9c86562e1d579bd3/detail":                 "nova_quotas_1_usage",
	"/compute/os-quota-sets/2db68fed84324f29bb73130c6c2094fb/detail":                 "nova_quotas_1_usage",
	"/compute/os-quota-sets/3d594eb0f04741069dbbb521635b21c7/detail":                 "nova_quotas_1_usage",
	"/compute/os-quota-sets/43ebde53fc314b1c9ea2b8c5dc744927/detail":                 "nova_quotas_1_usage",
	"/compute/os-quota-sets/5961c443439d4fcebe42643723755e9d/detail":                 "nova_quotas_1_usage",
	"/compute/os-quota-sets/fdb8424c4e4f4c0ba32c52e2de3bd80e/detail":                 "nova_quotas_1_usage",
	"/compute/os-quota-sets/4b1eb781a47440acb8af9850103e537f/detail":                 "nova_quotas_1_usage",
	"/neutron/v2.0/quotas/0c4e939acacf4376bdcd1129f1a054ad/details.json":             "neutron_quotas_1_usage",
	"/neutron/v2.0/quotas/0cbd49cbf76d405d9c86562e1d579bd3/details.json":             "neutron_quotas_1_usage",
	"/neutron/v2.0/quotas/2db68fed84324f29bb73130c6c2094fb/details.json":             "neutron_quotas_1_usage",
	"/neutron/v2.0/quotas/3d594eb0f04741069dbbb521635b21c7/details.json":             "neutron_quotas_1_usage",
	"/neutron/v2.0/quotas/43ebde53fc314b1c9ea2b8c5dc744927/details.json":             "neutron_quotas_1_usage",
	"/neutron/v2.0/quotas/5961c443439d4fcebe42643723755e9d/details.json":             "neutron_quotas_1_usage",
	"/neutron/v2.0/quotas/fdb8424c4e4f4c0ba32c52e2de3bd80e/details.json":             "neutron_quotas_1_usage",
	"/neutron/v2.0/quotas/4b1eb781a47440acb8af9850103e537f/details.json":             "neutron_quotas_1_usage",
	"/shares/v2/shares/detail?all_tenants=true":                                      "manila_shares",
	"/object-store/": "swift_list", // NOTE: /v1/AUTH_%(tenant_id)s
	"/object-store/?marker=centos9-epel-next": "swift_empty",
}

const DEFAULT_UUID = "3649e0f6-de80-ab6e-4f1c-351042d2f7fe"

func (suite *BaseOpenStackTestSuite) SetupTest() {
	httpmock.Activate()
	suite.Prefix = "openstack"

	suite.teardownFixtures()
	suite.installFixtures()

	os.Setenv("OS_CLIENT_CONFIG_FILE", path.Join(baseFixturePath, "test_config.yaml"))

	novaMetadataMapping := new(utils.LabelMappingFlag)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))
	opts := ExporterOptions{
		Cloud:                    cloudName,
		Prefix:                   suite.Prefix,
		DisabledMetrics:          []string{},
		EndpointType:             "public",
		NovaMetadataMapping:      novaMetadataMapping,
		DnsConcurrentCount:       10,
		APIDetailConcurrentCount: 10,
		PlacementConcurrentCount: 10,
		UUIDGenFunc: func() (string, error) {
			return DEFAULT_UUID, nil
		},
	}
	exporter, err := NewExporter(suite.ServiceName, opts, logger)

	if err != nil {
		suite.Require().NoError(err)
	}
	suite.Exporter = exporter
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

func TestMetricOptionsUseExporterNames(t *testing.T) {
	exporter := BaseOpenStackExporter{
		Name: "nova",
		ExporterConfig: ExporterConfig{
			ServiceName: "compute",
			ExporterOptions: ExporterOptions{
				DisabledMetrics: []string{"nova-total_vms"},
				EnabledMetrics:  []string{"nova-limits_vcpus_max"},
			},
		},
	}

	if got := exporter.qualifiedMetricName("total_vms"); got != "nova-total_vms" {
		t.Fatalf("qualifiedMetricName() = %q, want %q", got, "nova-total_vms")
	}
	if exporter.IsMetricEnabled("total_vms") {
		t.Fatal("legacy nova-* disable key did not disable the metric")
	}
	if !exporter.isExplicitlyEnabled("limits_vcpus_max") {
		t.Fatal("legacy nova-* enable key did not enable the metric")
	}
}

func TestSourceFetchDurationMetricIsOptIn(t *testing.T) {
	disabled := BaseOpenStackExporter{
		Name: "nova",
		ExporterConfig: ExporterConfig{
			ExporterOptions: ExporterOptions{Prefix: "openstack"},
		},
	}
	disabled.RegisterAndFillDescs(&struct{}{})
	if disabled.sourceFetchDuration != nil {
		t.Fatal("source fetch duration metric was created without CollectTime")
	}

	enabled := BaseOpenStackExporter{
		Name: "nova",
		ExporterConfig: ExporterConfig{
			ExporterOptions: ExporterOptions{Prefix: "openstack", CollectTime: true},
		},
	}
	enabled.RegisterAndFillDescs(&struct{}{})
	enabled.observeSourceFetchDuration("servers", time.Second)

	expected := `
# HELP openstack_exporter_source_fetch_duration_seconds Duration of source fetches in seconds.
# TYPE openstack_exporter_source_fetch_duration_seconds histogram
openstack_exporter_source_fetch_duration_seconds_bucket{service="nova",source="servers",le="0.1"} 0
openstack_exporter_source_fetch_duration_seconds_bucket{service="nova",source="servers",le="0.5"} 0
openstack_exporter_source_fetch_duration_seconds_bucket{service="nova",source="servers",le="1"} 1
openstack_exporter_source_fetch_duration_seconds_bucket{service="nova",source="servers",le="2"} 1
openstack_exporter_source_fetch_duration_seconds_bucket{service="nova",source="servers",le="5"} 1
openstack_exporter_source_fetch_duration_seconds_bucket{service="nova",source="servers",le="10"} 1
openstack_exporter_source_fetch_duration_seconds_bucket{service="nova",source="servers",le="30"} 1
openstack_exporter_source_fetch_duration_seconds_bucket{service="nova",source="servers",le="60"} 1
openstack_exporter_source_fetch_duration_seconds_bucket{service="nova",source="servers",le="+Inf"} 1
openstack_exporter_source_fetch_duration_seconds_sum{service="nova",source="servers"} 1
openstack_exporter_source_fetch_duration_seconds_count{service="nova",source="servers"} 1
`
	if err := testutil.CollectAndCompare(enabled.sourceFetchDuration, strings.NewReader(expected)); err != nil {
		t.Fatal(err)
	}
}

func TestSourceFetchDurationRegistersMultipleServices(t *testing.T) {
	registry := prometheus.NewPedanticRegistry()
	for _, service := range []string{"nova", "cinder"} {
		exporter := BaseOpenStackExporter{
			Name: service,
			ExporterConfig: ExporterConfig{
				ExporterOptions: ExporterOptions{Prefix: "openstack", CollectTime: true},
			},
		}
		exporter.RegisterAndFillDescs(&struct{}{})
		if err := registry.Register(exporter.sourceFetchDuration); err != nil {
			t.Fatalf("Register(%s source duration) error = %v", service, err)
		}
		exporter.observeSourceFetchDuration("servers", time.Second)
	}

	mfs, err := registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, mf := range mfs {
		if mf.GetName() == "openstack_exporter_source_fetch_duration_seconds" {
			if got := len(mf.Metric); got != 2 {
				t.Fatalf("source duration series = %d, want 2", got)
			}
			return
		}
	}
	t.Fatal("source duration metric family was not gathered")
}

func TestComputeScheduleBuildsMixedDependencyWaves(t *testing.T) {
	type scrape struct{}
	graph := Graph[struct{}, scrape]{
		Sources: []Source[struct{}, scrape]{
			{Name: "a"},
			{Name: "b"},
			{Name: "c", DependsOn: []string{"a"}},
		},
		Emitters: []Emitter[struct{}, scrape]{
			{Name: "emitA", Metrics: []string{"a_metric"}, Sources: []string{"a"}},
			{Name: "emitC", Metrics: []string{"c_metric"}, Sources: []string{"c"}},
		},
	}

	sched, err := graph.ComputeSchedule()
	if err != nil {
		t.Fatalf("ComputeSchedule() error = %v", err)
	}

	got := scheduleWaveNames(&graph, sched)
	want := [][]string{
		{"source:a", "source:b"},
		{"source:c[deps:a]", "emitter:emitA[sources:a]"},
		{"emitter:emitC[sources:c]"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("waves = %#v, want %#v", got, want)
	}
}

func TestPruneScheduleRecomputesDependencyGraph(t *testing.T) {
	type scrape struct{}
	graph := Graph[struct{}, scrape]{
		Sources: []Source[struct{}, scrape]{
			{Name: "a"},
			{Name: "b"},
			{Name: "c", DependsOn: []string{"b"}},
		},
		Emitters: []Emitter[struct{}, scrape]{
			{Name: "emitA", Metrics: []string{"a_metric"}, Sources: []string{"a"}},
			{Name: "emitC", Metrics: []string{"c_metric"}, Sources: []string{"c"}},
		},
	}
	base := &BaseOpenStackExporter{
		Name: "test",
		ExporterConfig: ExporterConfig{
			ExporterOptions: ExporterOptions{
				DisabledMetrics: []string{"test-c_metric"},
			},
		},
		logger: slog.New(slog.NewTextHandler(os.Stdout, nil)),
	}

	pruned, err := graph.PruneSchedule(base)
	if err != nil {
		t.Fatalf("PruneSchedule() error = %v", err)
	}
	got := scheduleWaveNames(&graph, pruned)
	want := [][]string{
		{"source:a"},
		{"emitter:emitA[sources:a]"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("pruned waves = %#v, want %#v", got, want)
	}
	if gotSources := pruned.TotalSources(); gotSources != 1 {
		t.Fatalf("TotalSources() = %d, want 1", gotSources)
	}
}

func TestRunScheduleStartsReadyNodesWithoutWaveBarrier(t *testing.T) {
	type scrape struct{}
	fastDone := make(chan struct{})
	allowSlow := make(chan struct{})
	emitted := make(chan struct{})

	graph := Graph[struct{}, scrape]{
		Sources: []Source[struct{}, scrape]{
			{
				Name: "fast",
				Fetch: func(struct{}, context.Context, *scrape) error {
					close(fastDone)
					return nil
				},
			},
			{
				Name: "slow",
				Fetch: func(struct{}, context.Context, *scrape) error {
					<-allowSlow
					return nil
				},
			},
		},
		Emitters: []Emitter[struct{}, scrape]{
			{
				Name:    "fastEmitter",
				Metrics: []string{"fast_metric"},
				Sources: []string{"fast"},
				Emit: func(struct{}, context.Context, *scrape, chan<- prometheus.Metric) error {
					close(emitted)
					return nil
				},
			},
		},
	}
	sched, err := graph.ComputeSchedule()
	if err != nil {
		t.Fatalf("ComputeSchedule() error = %v", err)
	}
	base := &BaseOpenStackExporter{
		Name:   "test",
		logger: slog.New(slog.NewTextHandler(os.Stdout, nil)),
	}

	done := make(chan int, 1)
	go func() {
		done <- runSchedule(struct{}{}, base, &graph, sched, new(scrape), make(chan prometheus.Metric))
	}()

	<-fastDone
	select {
	case <-emitted:
	case <-time.After(200 * time.Millisecond):
		close(allowSlow)
		t.Fatal("emitter did not start while unrelated source was still running")
	}

	close(allowSlow)
	if failures := <-done; failures != 0 {
		t.Fatalf("runSchedule() failures = %d, want 0", failures)
	}
}

func scheduleWaveNames[E, S any](graph *Graph[E, S], sched Schedule) [][]string {
	out := make([][]string, len(sched.waves))
	for i, wave := range sched.waves {
		out[i] = make([]string, len(wave))
		for j, nodeIdx := range wave {
			out[i][j] = graph.nodeLogName(sched.nodes[nodeIdx])
		}
	}
	return out
}

func TestOpenStackSuites(t *testing.T) {
	suite.Run(t, &CinderTestSuite{BaseOpenStackTestSuite: BaseOpenStackTestSuite{ServiceName: "volume"}})
	suite.Run(t, &NovaTestSuite{BaseOpenStackTestSuite: BaseOpenStackTestSuite{ServiceName: "compute"}})
	suite.Run(t, &NeutronTestSuite{BaseOpenStackTestSuite: BaseOpenStackTestSuite{ServiceName: "network"}})
	suite.Run(t, &LoadbalancerTestSuite{BaseOpenStackTestSuite: BaseOpenStackTestSuite{ServiceName: "load-balancer"}})
	suite.Run(t, &GlanceTestSuite{BaseOpenStackTestSuite: BaseOpenStackTestSuite{ServiceName: "image"}})
	suite.Run(t, &ContainerInfraTestSuite{BaseOpenStackTestSuite: BaseOpenStackTestSuite{ServiceName: "container-infra"}})
	suite.Run(t, &DesignateTestSuite{BaseOpenStackTestSuite: BaseOpenStackTestSuite{ServiceName: "dns"}})
	suite.Run(t, &IronicTestSuite{BaseOpenStackTestSuite: BaseOpenStackTestSuite{ServiceName: "baremetal"}})
	suite.Run(t, &GnocchiTestSuite{BaseOpenStackTestSuite: BaseOpenStackTestSuite{ServiceName: "gnocchi"}})
	suite.Run(t, &KeystoneTestSuite{BaseOpenStackTestSuite: BaseOpenStackTestSuite{ServiceName: "identity"}})
	suite.Run(t, &TroveTestSuite{BaseOpenStackTestSuite: BaseOpenStackTestSuite{ServiceName: "database"}})
	suite.Run(t, &HeatTestSuite{BaseOpenStackTestSuite: BaseOpenStackTestSuite{ServiceName: "orchestration"}})
	suite.Run(t, &PlacementTestSuite{BaseOpenStackTestSuite: BaseOpenStackTestSuite{ServiceName: "placement"}})
	suite.Run(t, &ManilaTestSuite{BaseOpenStackTestSuite: BaseOpenStackTestSuite{ServiceName: "sharev2"}})
	suite.Run(t, &ObjectStoreTestSuite{BaseOpenStackTestSuite: BaseOpenStackTestSuite{ServiceName: "object-store"}})
}
