package utils

import (
	"errors"
	"fmt"
	"reflect"
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

func TestNewLabelMappingFlagIsEmptyNotNil(t *testing.T) {
	flg := NewLabelMappingFlag()
	if flg.Labels == nil || flg.Keys == nil {
		t.Fatal("NewLabelMappingFlag() returned nil slices")
	}
	if len(flg.Labels) != 0 || len(flg.Keys) != 0 {
		t.Fatalf("NewLabelMappingFlag() is not empty: %v %v", flg.Labels, flg.Keys)
	}
	if got := flg.ExtractAny(nil); len(got) != 0 {
		t.Fatalf("ExtractAny(nil) = %v, want empty", got)
	}
}

func TestLabelMappingFlagExtractAny(t *testing.T) {
	flg := new(LabelMappingFlag)
	if err := flg.Set("str=a_string,num=a_number,big=a_big_number,flag=a_bool,gone=missing,nested=an_object,list=an_array,null=a_null"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	got := flg.ExtractAny(map[string]any{
		"a_string":     "root",
		"a_number":     float64(30101),
		"a_big_number": float64(10737418240),
		"a_bool":       false,
		"an_object":    map[string]any{"os_distro": "ubuntu"},
		"an_array":     []any{"a", "b"},
		"a_null":       nil,
	})

	want := []string{"root", "30101", "10737418240", "false", "", "", "", ""}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ExtractAny() = %#v, want %#v", got, want)
	}
}

func TestQualifiedLabelMappingDerivesLabelFromLeaf(t *testing.T) {
	flg := &LabelMappingFlag{DeriveLabelFromLeaf: true}
	if err := flg.Set("driver_info.deploy_kernel,bmc=driver_info.redfish_address"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	if want := []string{"deploy_kernel", "bmc"}; !reflect.DeepEqual(flg.Labels, want) {
		t.Fatalf("Labels = %#v, want %#v", flg.Labels, want)
	}
	if want := []string{"driver_info.deploy_kernel", "driver_info.redfish_address"}; !reflect.DeepEqual(flg.Keys, want) {
		t.Fatalf("Keys = %#v, want %#v", flg.Keys, want)
	}
}

// Two qualified keys sharing a leaf would silently produce one label shadowing
// the other, so the existing duplicate check must reject them.
func TestQualifiedLabelMappingRejectsAmbiguousLeaves(t *testing.T) {
	flg := &LabelMappingFlag{DeriveLabelFromLeaf: true}
	err := flg.Set("instance_info.local_gb,properties.local_gb")
	if !errors.Is(err, ErrLabelDup) {
		t.Fatalf("Set() error = %v, want %v", err, ErrLabelDup)
	}
}

func TestIsSensitive(t *testing.T) {
	tests := []struct {
		key  string
		want bool
	}{
		{"ipmi_password", true},
		{"redfish_password", true},
		{"snmp_priv_password", true},
		{"root_password", true},
		{"password", true},
		{"driver_info.ipmi_password", true},
		{"extra.root_password", true},
		{"api_secret", true},
		{"auth_token", true},
		{"private_key", true},
		{"configdrive", true},
		{"instance_info.image_url", true},
		// Legitimate fields that an unanchored pattern would wrongly redact.
		{"password_updated_at", false},
		{"secret_count", false},
		{"deploy_kernel", false},
		{"ipmi_address", false},
		{"image_source", false},
		{"public_key", false},
	}

	for _, tt := range tests {
		if got := isSensitive(tt.key, tt.key); got != tt.want {
			t.Errorf("isSensitive(%q) = %v, want %v", tt.key, got, tt.want)
		}
	}
}

// A credential must stay redacted even when the operator gives it an
// innocuous-looking label.
func TestSensitiveDetectionChecksLabelAndKey(t *testing.T) {
	flg := &LabelMappingFlag{DeriveLabelFromLeaf: true}
	if err := flg.Set("bmc=driver_info.ipmi_password"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if !flg.Sensitive[0] {
		t.Fatal("credential hidden behind a neutral label was not detected")
	}
}

func TestExtractRedactsSensitiveValues(t *testing.T) {
	flg := new(LabelMappingFlag)
	if err := flg.Set("addr=ipmi_address,pw=ipmi_password,unset=redfish_password"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	got := flg.ExtractAny(map[string]any{
		"ipmi_address":  "10.10.0.101",
		"ipmi_password": "hunter2",
	})
	want := []string{"10.10.0.101", RedactedLabelValue, ""}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ExtractAny() = %#v, want %#v", got, want)
	}

	strGot := flg.Extract(map[string]string{
		"ipmi_address":  "10.10.0.101",
		"ipmi_password": "hunter2",
	})
	if !reflect.DeepEqual(strGot, want) {
		t.Fatalf("Extract() = %#v, want %#v", strGot, want)
	}

	if labels := flg.SensitiveLabels(); !reflect.DeepEqual(labels, []string{"pw", "unset"}) {
		t.Fatalf("SensitiveLabels() = %#v, want [pw unset]", labels)
	}
}

// Sensitive must stay index-aligned with Labels and Keys when a cumulative
// flag is given more than once.
func TestSensitiveStaysAlignedAcrossRepeatedSet(t *testing.T) {
	flg := new(LabelMappingFlag)
	if err := flg.Set("addr=ipmi_address"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if err := flg.Set("pw=ipmi_password"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if len(flg.Sensitive) != len(flg.Keys) {
		t.Fatalf("Sensitive/Keys length mismatch: %d vs %d", len(flg.Sensitive), len(flg.Keys))
	}
	if flg.Sensitive[0] || !flg.Sensitive[1] {
		t.Fatalf("Sensitive misaligned: %v", flg.Sensitive)
	}
}

func TestValidateAgainstRejectsCollidingLabels(t *testing.T) {
	flg := new(LabelMappingFlag)
	if err := flg.Set("status=some_key"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	if err := flg.ValidateAgainst([]string{"id", "status"}); !errors.Is(err, ErrLabelDup) {
		t.Fatalf("ValidateAgainst() error = %v, want %v", err, ErrLabelDup)
	}
	if err := flg.ValidateAgainst([]string{"id", "name"}); err != nil {
		t.Fatalf("ValidateAgainst() unexpected error = %v", err)
	}
}
