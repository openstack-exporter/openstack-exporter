package utils

import (
	"fmt"
	"testing"

	assertpkg "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- ExtraLabelsFlag tests ----

func TestExtraLabelsFlag_Set_DynamicLabel(t *testing.T) {
	flg := new(ExtraLabelsFlag)
	require.NoError(t, flg.Set("baremetal.node:conductor"))

	spec := flg.Specs["baremetal"]["node"]
	assertpkg.Equal(t, []string{"conductor"}, spec.DynamicLabels)
	assertpkg.Equal(t, []string{"conductor"}, spec.DynamicFields)
	assertpkg.Empty(t, spec.StaticLabels)
}

func TestExtraLabelsFlag_Set_DotPathConvertsToUnderscore(t *testing.T) {
	flg := new(ExtraLabelsFlag)
	require.NoError(t, flg.Set("baremetal.node:extra.rack_id"))

	spec := flg.Specs["baremetal"]["node"]
	assertpkg.Equal(t, []string{"extra_rack_id"}, spec.DynamicLabels)
	assertpkg.Equal(t, []string{"extra.rack_id"}, spec.DynamicFields)
}

func TestExtraLabelsFlag_Set_StaticLabel(t *testing.T) {
	flg := new(ExtraLabelsFlag)
	require.NoError(t, flg.Set("baremetal.node:datacenter=dc1"))

	spec := flg.Specs["baremetal"]["node"]
	assertpkg.Empty(t, spec.DynamicLabels)
	assertpkg.Empty(t, spec.DynamicFields)
	assertpkg.Equal(t, map[string]string{"datacenter": "dc1"}, spec.StaticLabels)
}

func TestExtraLabelsFlag_Set_Mixed(t *testing.T) {
	flg := new(ExtraLabelsFlag)
	require.NoError(t, flg.Set("baremetal.node:conductor,extra.rack_id,env=production"))

	spec := flg.Specs["baremetal"]["node"]
	assertpkg.Equal(t, []string{"conductor", "extra_rack_id"}, spec.DynamicLabels)
	assertpkg.Equal(t, []string{"conductor", "extra.rack_id"}, spec.DynamicFields)
	assertpkg.Equal(t, map[string]string{"env": "production"}, spec.StaticLabels)
}

func TestExtraLabelsFlag_Set_StaticValueWithEquals(t *testing.T) {
	// Static label values that themselves contain '=' must be preserved
	flg := new(ExtraLabelsFlag)
	require.NoError(t, flg.Set("compute.server:tag=key=value"))

	spec := flg.Specs["compute"]["server"]
	assertpkg.Equal(t, map[string]string{"tag": "key=value"}, spec.StaticLabels)
}

func TestExtraLabelsFlag_Set_Cumulative(t *testing.T) {
	flg := new(ExtraLabelsFlag)
	require.NoError(t, flg.Set("baremetal.node:conductor"))
	require.NoError(t, flg.Set("compute.server:name"))

	assertpkg.Contains(t, flg.Specs, "baremetal")
	assertpkg.Contains(t, flg.Specs, "compute")
	assertpkg.Contains(t, flg.Specs["baremetal"], "node")
	assertpkg.Contains(t, flg.Specs["compute"], "server")
}

func TestExtraLabelsFlag_Set_EmptyStringIsNoop(t *testing.T) {
	flg := new(ExtraLabelsFlag)
	assertpkg.NoError(t, flg.Set(""))
	assertpkg.Empty(t, flg.Specs)
}

func TestExtraLabelsFlag_Set_Errors(t *testing.T) {
	tests := []struct {
		name  string
		input string
		errIs error
	}{
		{"missing colon", "baremetal.node", nil},
		{"missing dot in target", "baremetal:conductor", nil},
		{"empty labels after colon", "baremetal.node:", ErrMissingLabels},
		{"duplicate dynamic label", "baremetal.node:conductor,conductor", ErrLabelDup},
		{"dynamic then static same name", "baremetal.node:conductor,conductor=abc", ErrLabelDup},
		{"bad label name starts with digit", "baremetal.node:1bad", ErrLabelName},
		{"bad label name double underscore prefix", "baremetal.node:__reserved", ErrLabelName},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			flg := new(ExtraLabelsFlag)
			err := flg.Set(tc.input)
			assertpkg.Error(t, err)
			if tc.errIs != nil {
				assertpkg.ErrorIs(t, err, tc.errIs)
			}
		})
	}
}

func TestExtraLabelsFlag_IsCumulative(t *testing.T) {
	flg := new(ExtraLabelsFlag)
	assertpkg.True(t, flg.IsCumulative())
}

func TestExtraLabelsFlag_String(t *testing.T) {
	flg := new(ExtraLabelsFlag)
	assertpkg.NotEmpty(t, flg.String())
}

func TestExtraLabelsFlag_Extract(t *testing.T) {
	flg := new(ExtraLabelsFlag)
	require.NoError(t, flg.Set("baremetal.node:conductor,env=production"))
	require.NoError(t, flg.Set("compute.server:name"))

	t.Run("found", func(t *testing.T) {
		spec := flg.Extract("baremetal", "node")
		assertpkg.NotNil(t, spec)
		assertpkg.Equal(t, []string{"conductor"}, spec.DynamicLabels)
		assertpkg.Equal(t, map[string]string{"env": "production"}, spec.StaticLabels)
	})

	t.Run("wrong metric", func(t *testing.T) {
		assertpkg.Nil(t, flg.Extract("baremetal", "port"))
	})

	t.Run("wrong service", func(t *testing.T) {
		assertpkg.Nil(t, flg.Extract("network", "node"))
	})

	t.Run("nil receiver", func(t *testing.T) {
		var nilFlg *ExtraLabelsFlag
		assertpkg.Nil(t, nilFlg.Extract("baremetal", "node"))
	})
}

func TestExtraLabelsFlag_Extract_TableDriven(t *testing.T) {
	flg := new(ExtraLabelsFlag)
	require.NoError(t, flg.Set("baremetal.node:conductor"))

	tests := []struct {
		service string
		metric  string
		found   bool
	}{
		{"baremetal", "node", true},
		{"baremetal", "port", false},
		{"compute", "node", false},
		{"", "", false},
	}
	for _, tc := range tests {
		t.Run(fmt.Sprintf("%s.%s", tc.service, tc.metric), func(t *testing.T) {
			spec := flg.Extract(tc.service, tc.metric)
			if tc.found {
				assertpkg.NotNil(t, spec)
			} else {
				assertpkg.Nil(t, spec)
			}
		})
	}
}

func TestLabelMappingFlag_Set(t *testing.T) {
	assert := assertpkg.New(t)

	flg := new(LabelMappingFlag)

	// 1. Basic parsing
	err := flg.Set("server_group=group,severity,foobar=baz")
	assert.NoError(err)
	assert.Len(flg.Labels, 3)
	assert.Len(flg.Keys, 3)
	assert.Equal([]string{"server_group", "severity", "foobar"}, flg.Labels)
	assert.Equal([]string{"group", "severity", "baz"}, flg.Keys)

	// 2. Test cumulative
	err = flg.Set("test,notify_team=quux")
	assert.NoError(err)
	assert.Len(flg.Labels, 5)
	assert.Len(flg.Keys, 5)
	assert.Equal([]string{"server_group", "severity", "foobar", "test", "notify_team"}, flg.Labels)
	assert.Equal([]string{"group", "severity", "baz", "test", "quux"}, flg.Keys)

	// 3. Forbid label duplication
	err = flg.Set("test2,severity")
	assert.ErrorIs(err, ErrLabelDup)
	assert.EqualError(err, "duplicate label: severity")

	// 4. Check label name comply with prometheus requirements
	for _, badLabel := range []string{"Test Label", "__some_label", "1ee7"} {
		t.Run(badLabel, func(t *testing.T) {
			err = flg.Set(badLabel)
			assertpkg.ErrorIs(t, err, ErrLabelName)
			assertpkg.EqualError(t, err, fmt.Sprintf("bad label name: %s", badLabel))
		})
	}
}

func TestLabelMappingFlag_Extract(t *testing.T) {
	flg := new(LabelMappingFlag)
	err := flg.Set("server_group=group,severity,foobar=baz")
	require.NoError(t, err)

	testCases := []struct {
		name     string
		metadata map[string]string
		expected []string
	}{
		{"all", map[string]string{"group": "grp1", "severity": "critical", "baz": "lorem-ipsum"}, []string{"grp1", "critical", "lorem-ipsum"}},
		{"some", map[string]string{"group": "grp2"}, []string{"grp2", "", ""}},
		{"nil", nil, []string{"", "", ""}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			obtained := flg.Extract(tc.metadata)
			assertpkg.Equal(t, tc.expected, obtained)
		})
	}
}
