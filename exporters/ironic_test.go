package exporters

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud/v2/openstack/baremetal/v1/nodes"
	"github.com/jarcoal/httpmock"
	"github.com/openstack-exporter/openstack-exporter/utils"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

type IronicTestSuite struct {
	BaseOpenStackTestSuite
}

var ironicExpectedUp = `
# HELP openstack_ironic_node node
# TYPE openstack_ironic_node gauge
openstack_ironic_node{console_enabled="false",deploy_kernel="7ff5ef56-daaa-4256-9dd8-c3f1f9964ebc",deploy_ramdisk="e9c96d45-a4c8-4165-8753-9d8f32779e99",id="f50dcc35-4913-4667-a9fa-d130659c5661",maintenance="false",maintenance_reason="",name="r1-02",power_state="power off",provision_state="available",resource_class="baremetal",retired="true",retired_reason="No longer needed"} 1
openstack_ironic_node{console_enabled="true",deploy_kernel="7ff5ef56-daaa-4256-9dd8-c3f1f9964ebc",deploy_ramdisk="e9c96d45-a4c8-4165-8753-9d8f32779e99",id="0129d2fc-0e5c-4b5b-a73b-01844d913957",maintenance="false",maintenance_reason="",name="r1-04",power_state="power on",provision_state="active",resource_class="baremetal",retired="true",retired_reason="No longer needed"} 1
openstack_ironic_node{console_enabled="true",deploy_kernel="7ff5ef56-daaa-4256-9dd8-c3f1f9964ebc",deploy_ramdisk="e9c96d45-a4c8-4165-8753-9d8f32779e99",id="c9f98cc9-25e9-424e-8a89-002989054ec2",maintenance="true",maintenance_reason="Firmware upgrade",name="r1-05",power_state="power off",provision_state="available",resource_class="baremetal",retired="true",retired_reason="No longer needed"} 1
openstack_ironic_node{console_enabled="true",deploy_kernel="7ff5ef56-daaa-4256-9dd8-c3f1f9964ebc",deploy_ramdisk="e9c96d45-a4c8-4165-8753-9d8f32779e99",id="d381bea3-8768-4f12-a9b3-abf750ba918f",maintenance="false",maintenance_reason="",name="r1-03",power_state="power on",provision_state="active",resource_class="baremetal",retired="true",retired_reason="No longer needed"} 1
openstack_ironic_node{console_enabled="true",deploy_kernel="7ff5ef56-daaa-4256-9dd8-c3f1f9964ebc",deploy_ramdisk="e9c96d45-a4c8-4165-8753-9d8f32779e99",id="d5641882-f7e5-4b92-9423-7e8157586218",maintenance="true",maintenance_reason="",name="r1-01",power_state="power off",provision_state="error",resource_class="baremetal",retired="true",retired_reason="No longer needed"} 1
# HELP openstack_ironic_node_provision_updated_at node_provision_updated_at
# TYPE openstack_ironic_node_provision_updated_at gauge
openstack_ironic_node_provision_updated_at{id="0129d2fc-0e5c-4b5b-a73b-01844d913957",name="r1-04",provision_state="active"} 1.593544011e+09
openstack_ironic_node_provision_updated_at{id="c9f98cc9-25e9-424e-8a89-002989054ec2",name="r1-05",provision_state="available"} 1.562908443e+09
openstack_ironic_node_provision_updated_at{id="d381bea3-8768-4f12-a9b3-abf750ba918f",name="r1-03",provision_state="active"} 1.593747281e+09
openstack_ironic_node_provision_updated_at{id="d5641882-f7e5-4b92-9423-7e8157586218",name="r1-01",provision_state="error"} 1.594708597e+09
openstack_ironic_node_provision_updated_at{id="f50dcc35-4913-4667-a9fa-d130659c5661",name="r1-02",provision_state="available"} 1.594740492e+09
# HELP openstack_ironic_node_updated_at node_updated_at
# TYPE openstack_ironic_node_updated_at gauge
openstack_ironic_node_updated_at{id="0129d2fc-0e5c-4b5b-a73b-01844d913957",name="r1-04",provision_state="active"} 1.593544011e+09
openstack_ironic_node_updated_at{id="c9f98cc9-25e9-424e-8a89-002989054ec2",name="r1-05",provision_state="available"} 1.592845911e+09
openstack_ironic_node_updated_at{id="d381bea3-8768-4f12-a9b3-abf750ba918f",name="r1-03",provision_state="active"} 1.594162438e+09
openstack_ironic_node_updated_at{id="d5641882-f7e5-4b92-9423-7e8157586218",name="r1-01",provision_state="error"} 1.594708598e+09
openstack_ironic_node_updated_at{id="f50dcc35-4913-4667-a9fa-d130659c5661",name="r1-02",provision_state="available"} 1.594740494e+09
# HELP openstack_ironic_up up
# TYPE openstack_ironic_up gauge
openstack_ironic_up 1
`

func (suite *IronicTestSuite) TestIronicExporter() {
	err := testutil.CollectAndCompare(suite.Exporter, strings.NewReader(ironicExpectedUp), metricNamesFrom(ironicExpectedUp)...)
	assert.NoError(suite.T(), err)
}

func (suite *IronicTestSuite) TestIronicExporterFollowsMarkerPagination() {
	fixture, err := os.ReadFile(suite.FixturePath("ironic_nodes"))
	assert.NoError(suite.T(), err)

	var page map[string]any
	assert.NoError(suite.T(), json.Unmarshal(fixture, &page))
	nodesOnFixture := page["nodes"].([]any)
	firstPageNodes := make([]any, 0, ironicNodePageSize)
	for len(firstPageNodes) < ironicNodePageSize {
		firstPageNodes = append(firstPageNodes, nodesOnFixture...)
	}
	page["nodes"] = firstPageNodes
	marker := firstPageNodes[len(firstPageNodes)-1].(map[string]any)["uuid"].(string)

	var secondPage map[string]any
	assert.NoError(suite.T(), json.Unmarshal(fixture, &secondPage))

	firstURL := suite.MakeURL("/ironic/v1/nodes/detail?limit=1000&sort_dir=asc&sort_key=id", "")
	nextURL := suite.MakeURL("/ironic/v1/nodes/detail?limit=1000&marker="+marker+"&sort_dir=asc&sort_key=id", "")
	httpmock.RegisterResponder("GET", firstURL, httpmock.NewJsonResponderOrPanic(http.StatusOK, page))
	httpmock.RegisterResponder("GET", nextURL, httpmock.NewJsonResponderOrPanic(http.StatusOK, secondPage))

	exporter := suite.Exporter.(*IronicExporter)
	scrape := new(ironicScrape)
	err = exporter.fetchNodes(context.Background(), scrape)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), scrape.nodes, ironicNodePageSize+len(nodesOnFixture))
}

func ironicTestMapping(t *testing.T, mapping string) *utils.LabelMappingFlag {
	t.Helper()
	m := &utils.LabelMappingFlag{DeriveLabelFromLeaf: true}
	if err := m.Set(mapping); err != nil {
		t.Fatalf("Set(%q) error = %v", mapping, err)
	}
	return m
}

// The default mapping must reproduce exactly the labels the node metric
// carried when deploy_kernel and deploy_ramdisk were hardcoded.
func TestIronicDefaultNodeExtraLabels(t *testing.T) {
	m := NewIronicNodeExtraLabels()
	if want := []string{"deploy_kernel", "deploy_ramdisk"}; !reflect.DeepEqual(m.Labels, want) {
		t.Fatalf("default labels = %#v, want %#v", m.Labels, want)
	}
}

func TestIronicNodeExtraLabelValues(t *testing.T) {
	node := &nodes.Node{
		DriverInfo:   map[string]any{"deploy_kernel": "kernel-uuid", "ipmi_address": "10.10.0.101", "ipmi_password": "hunter2"},
		InstanceInfo: map[string]any{"image_source": "image-uuid", "memory_mb": float64(32768), "image_properties": map[string]any{"os_distro": "ubuntu"}},
		Extra:        map[string]any{"root_password": "hash"},
		Properties:   map[string]any{"cpu_arch": "x86_64"},
	}

	tests := []struct {
		name    string
		mapping string
		want    []string
	}{
		{"driver_info", "driver_info.deploy_kernel", []string{"kernel-uuid"}},
		{"across dictionaries", "driver_info.ipmi_address,instance_info.image_source,properties.cpu_arch",
			[]string{"10.10.0.101", "image-uuid", "x86_64"}},
		{"numeric value", "instance_info.memory_mb", []string{"32768"}},
		{"nested object is not rendered", "instance_info.image_properties", []string{""}},
		{"missing key", "driver_info.does_not_exist", []string{""}},
		{"renamed label", "bmc=driver_info.ipmi_address", []string{"10.10.0.101"}},
		{"credentials are redacted", "driver_info.ipmi_password,extra.root_password",
			[]string{utils.RedactedLabelValue, utils.RedactedLabelValue}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapping := ironicTestMapping(t, tt.mapping)
			if got := mapping.ExtractNestedAny(ironicNodeNestedMaps(node)...); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ExtractNestedAny() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

// An empty flag value must drop the labels entirely, which is how an operator
// opts out of the defaults.
func TestIronicNodeExtraLabelsCanBeCleared(t *testing.T) {
	mapping := ironicTestMapping(t, "")
	if got := mapping.ExtractNestedAny(ironicNodeNestedMaps(&nodes.Node{DriverInfo: map[string]any{"deploy_kernel": "k"}})...); len(got) != 0 {
		t.Fatalf("ExtractNestedAny() = %#v, want empty", got)
	}
}

func TestValidateNodeExtraLabels(t *testing.T) {
	for _, tt := range []struct {
		mapping string
		wantErr bool
	}{
		{"driver_info.deploy_kernel", false},
		{"instance_info.image_source", false},
		{"extra.foo", false},
		{"properties.cpu_arch", false},
		{"driver_internal_info.agent_version", false},
		{"lbl=bogus_dict.key", true},
		{"unqualified", true},
	} {
		m := &utils.LabelMappingFlag{DeriveLabelFromLeaf: true}
		if err := m.Set(tt.mapping); err != nil {
			t.Fatalf("Set(%q) error = %v", tt.mapping, err)
		}
		err := m.ValidateNestedKeys(ironicNodeInfoMaps...)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateNestedKeys(%q) error = %v, wantErr %v", tt.mapping, err, tt.wantErr)
		}
	}
}
