package main

import (
	"net/http/httptest"
	"testing"

	"github.com/openstack-exporter/openstack-exporter/exporters"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetAutoServicesState(t *testing.T) {
	serviceStates := map[string]serviceState{
		"compute": serviceEnabled,
		"network": serviceAuto,
		"image":   serviceAuto,
	}

	setAutoServicesState(serviceStates, serviceDisabled)
	assert.Equal(t, serviceEnabled, serviceStates["compute"])
	assert.Equal(t, serviceDisabled, serviceStates["network"])
	assert.Equal(t, serviceDisabled, serviceStates["image"])
}

func TestApplyAutodetection(t *testing.T) {
	serviceStates := map[string]serviceState{
		"compute": serviceEnabled,
		"network": serviceAuto,
		"image":   serviceAuto,
		"dns":     serviceDisabled,
	}

	applyAutodetection(serviceStates, []string{"network"})
	assert.Equal(t, serviceEnabled, serviceStates["compute"])
	assert.Equal(t, serviceEnabled, serviceStates["network"])
	assert.Equal(t, serviceDisabled, serviceStates["image"])
	assert.Equal(t, serviceDisabled, serviceStates["dns"])
}

func TestGetEnabledServicesFromStates(t *testing.T) {
	serviceStates := make(map[string]serviceState, len(exporters.SupportedExporters))
	for _, service := range exporters.SupportedExporters {
		serviceStates[service] = serviceDisabled
	}
	serviceStates["compute"] = serviceEnabled
	serviceStates["network"] = serviceEnabled

	services := getEnabledServicesFromStates(serviceStates)
	assert.ElementsMatch(t, []string{"compute", "network"}, services)
}

func TestSelectServicesForRequest(t *testing.T) {
	configured := []string{"compute", "network", "image"}
	tests := []struct {
		name      string
		url       string
		expected  []string
		errSubstr string
	}{
		{
			name:     "filters with include and exclude",
			url:      "/probe?include_services=compute,image&exclude_services=image",
			expected: []string{"compute"},
		},
		{
			name:     "normalizes spaces and duplicates",
			url:      "/probe?include_services=compute,%20compute%20,network",
			expected: []string{"compute", "network"},
		},
		{
			name:      "rejects invalid include service",
			url:       "/probe?include_services=compute,bad",
			errSubstr: "invalid include_services",
		},
		{
			name:      "rejects invalid exclude service",
			url:       "/probe?exclude_services=bad",
			errSubstr: "invalid exclude_services",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.url, nil)
			services, err := selectServicesForRequest(configured, req)

			if tc.errSubstr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errSubstr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expected, services)
		})
	}
}
