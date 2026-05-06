package exporters

import (
	"testing"

	"github.com/openstack-exporter/openstack-exporter/utils"
	"github.com/prometheus/client_golang/prometheus"
	assertpkg "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sampleObj is a simple struct used to exercise resolveField without
// depending on any gophercloud type.
type sampleObj struct {
	Name   string                 `json:"name"`
	Status string                 `json:"status"`
	Extra  map[string]any `json:"extra"`
}

func TestResolveField_SimpleField(t *testing.T) {
	obj := sampleObj{Name: "foo", Status: "active"}
	assertpkg.Equal(t, "foo", resolveField(obj, "name"))
	assertpkg.Equal(t, "active", resolveField(obj, "status"))
}

func TestResolveField_NestedMapField(t *testing.T) {
	obj := sampleObj{Extra: map[string]any{"rack_id": "rack-42"}}
	assertpkg.Equal(t, "rack-42", resolveField(obj, "extra.rack_id"))
}

func TestResolveField_MissingTopLevelField(t *testing.T) {
	obj := sampleObj{Name: "foo"}
	assertpkg.Equal(t, "", resolveField(obj, "nonexistent"))
}

func TestResolveField_MissingMapKey(t *testing.T) {
	obj := sampleObj{Extra: map[string]any{}}
	assertpkg.Equal(t, "", resolveField(obj, "extra.missing_key"))
}

func TestResolveField_NilMap(t *testing.T) {
	obj := sampleObj{} // Extra is nil
	assertpkg.Equal(t, "", resolveField(obj, "extra.rack_id"))
}

func TestResolveField_NestedAccessOnNonMap(t *testing.T) {
	// Attempting nested access on a scalar field returns empty string
	obj := sampleObj{Name: "foo"}
	assertpkg.Equal(t, "", resolveField(obj, "name.subfield"))
}

func TestResolveField_IndexCachedOnSecondCall(t *testing.T) {
	// Call twice to exercise the cache hit path
	obj := sampleObj{Name: "cached"}
	assertpkg.Equal(t, "cached", resolveField(obj, "name"))
	assertpkg.Equal(t, "cached", resolveField(obj, "name"))
}

func TestComputeMetricLabels_NoExtraLabels(t *testing.T) {
	metric := Metric{Name: "node", Labels: []string{"id", "name"}}
	extraLabels := new(utils.ExtraLabelsFlag)

	labels := computeMetricLabels("baremetal", metric, extraLabels)
	assertpkg.Equal(t, []string{"id", "name"}, labels)
}

func TestComputeMetricLabels_WithDynamicLabels(t *testing.T) {
	metric := Metric{Name: "node", Labels: []string{"id", "name"}}
	extraLabels := new(utils.ExtraLabelsFlag)
	require.NoError(t, extraLabels.Set("baremetal.node:conductor,conductor_group"))

	labels := computeMetricLabels("baremetal", metric, extraLabels)
	assertpkg.Equal(t, []string{"id", "name", "conductor", "conductor_group"}, labels)
}

func TestComputeMetricLabels_NilExtraLabels(t *testing.T) {
	metric := Metric{Name: "node", Labels: []string{"id", "name"}}

	labels := computeMetricLabels("baremetal", metric, nil)
	assertpkg.Equal(t, []string{"id", "name"}, labels)
}

func TestComputeMetricLabels_DotPathLabelName(t *testing.T) {
	metric := Metric{Name: "node", Labels: []string{"id"}}
	extraLabels := new(utils.ExtraLabelsFlag)
	require.NoError(t, extraLabels.Set("baremetal.node:extra.rack_id"))

	labels := computeMetricLabels("baremetal", metric, extraLabels)
	// dot is converted to underscore in the label name
	assertpkg.Equal(t, []string{"id", "extra_rack_id"}, labels)
}

func TestComputeMetricLabels_DoesNotDuplicateExistingLabel(t *testing.T) {
	metric := Metric{Name: "node", Labels: []string{"id", "name", "conductor"}}
	extraLabels := new(utils.ExtraLabelsFlag)
	require.NoError(t, extraLabels.Set("baremetal.node:conductor"))

	labels := computeMetricLabels("baremetal", metric, extraLabels)
	count := 0
	for _, l := range labels {
		if l == "conductor" {
			count++
		}
	}
	assertpkg.Equal(t, 1, count, "conductor label must not be duplicated")
}

func TestComputeMetricLabels_WrongServiceNoEffect(t *testing.T) {
	metric := Metric{Name: "node", Labels: []string{"id"}}
	extraLabels := new(utils.ExtraLabelsFlag)
	require.NoError(t, extraLabels.Set("compute.server:name"))

	// extra-labels registered for a different service should not appear
	labels := computeMetricLabels("baremetal", metric, extraLabels)
	assertpkg.Equal(t, []string{"id"}, labels)
}

func TestComputeConstantLabels_WithStaticLabels(t *testing.T) {
	metric := Metric{Name: "node", Labels: []string{"id"}}
	extraLabels := new(utils.ExtraLabelsFlag)
	require.NoError(t, extraLabels.Set("baremetal.node:env=production,region=us-east"))

	constLabels := computeConstantLabels("baremetal", metric, extraLabels)
	assertpkg.Equal(t, prometheus.Labels{"env": "production", "region": "us-east"}, constLabels)
}

func TestComputeConstantLabels_OnlyDynamicLabelsReturnsNil(t *testing.T) {
	metric := Metric{Name: "node", Labels: []string{"id"}}
	extraLabels := new(utils.ExtraLabelsFlag)
	require.NoError(t, extraLabels.Set("baremetal.node:conductor"))

	constLabels := computeConstantLabels("baremetal", metric, extraLabels)
	assertpkg.Nil(t, constLabels)
}

func TestComputeConstantLabels_NoExtraLabels(t *testing.T) {
	metric := Metric{Name: "node", Labels: []string{"id"}}
	extraLabels := new(utils.ExtraLabelsFlag)

	constLabels := computeConstantLabels("baremetal", metric, extraLabels)
	assertpkg.Nil(t, constLabels)
}

func TestComputeConstantLabels_NilExtraLabels(t *testing.T) {
	metric := Metric{Name: "node", Labels: []string{"id"}}

	constLabels := computeConstantLabels("baremetal", metric, nil)
	assertpkg.Nil(t, constLabels)
}

func TestResolveExtraLabelValues_DynamicAndNestedFields(t *testing.T) {
	obj := sampleObj{
		Name:  "foo",
		Extra: map[string]any{"rack_id": "rack-42"},
	}
	spec := &utils.ExtraLabelSpec{
		DynamicFields: []string{"name", "extra.rack_id"},
		DynamicLabels: []string{"name", "extra_rack_id"},
		StaticLabels:  map[string]string{},
	}

	values := resolveExtraLabelValues(obj, spec)
	assertpkg.Equal(t, []string{"foo", "rack-42"}, values)
}

func TestResolveExtraLabelValues_NilSpec(t *testing.T) {
	obj := sampleObj{Name: "foo"}
	assertpkg.Nil(t, resolveExtraLabelValues(obj, nil))
}

func TestResolveExtraLabelValues_MissingField(t *testing.T) {
	obj := sampleObj{Name: "foo"}
	spec := &utils.ExtraLabelSpec{
		DynamicFields: []string{"nonexistent"},
		DynamicLabels: []string{"nonexistent"},
		StaticLabels:  map[string]string{},
	}

	values := resolveExtraLabelValues(obj, spec)
	assertpkg.Equal(t, []string{""}, values)
}
