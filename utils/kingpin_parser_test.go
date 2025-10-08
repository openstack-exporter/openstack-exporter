package utils

import (
	"fmt"
	"testing"

	assertpkg "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
