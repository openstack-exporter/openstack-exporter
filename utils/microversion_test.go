package utils

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/stretchr/testify/require"
)

func TestIsMicroversionAtLeast(t *testing.T) {
	tests := []struct {
		name     string
		current  string
		required string
		want     bool
	}{
		{name: "same version", current: "2.46", required: "2.46", want: true},
		{name: "higher minor", current: "2.50", required: "2.46", want: true},
		{name: "lower minor", current: "2.32", required: "2.33", want: false},
		{name: "higher major", current: "3.1", required: "2.99", want: true},
		{name: "lower major", current: "1.99", required: "2.1", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := IsMicroversionAtLeast(tt.current, tt.required)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestIsMicroversionAtLeastInvalidVersion(t *testing.T) {
	_, err := IsMicroversionAtLeast("invalid", "2.46")
	require.Error(t, err)

	_, err = IsMicroversionAtLeast("2.46", "invalid")
	require.Error(t, err)
}

func TestSetupClientMicroversionV2UsesDefaultWhenSupported(t *testing.T) {
	client := newMicroversionTestClient(t, `{
		"version": {
			"id": "v2.1",
			"status": "CURRENT",
			"version": "2.90",
			"min_version": "2.1"
		}
	}`)

	err := SetupClientMicroversionV2(context.Background(), client, "OS_TEST_API_VERSION", "2.50", slog.Default())

	require.NoError(t, err)
	require.Equal(t, "2.50", client.Microversion)
}

func TestSetupClientMicroversionV2UsesDetectedMaximumWithoutDefault(t *testing.T) {
	client := newMicroversionTestClient(t, `{
		"version": {
			"id": "v3.0",
			"status": "CURRENT",
			"version": "3.71",
			"min_version": "3.0"
		}
	}`)

	err := SetupClientMicroversionV2(context.Background(), client, "OS_TEST_API_VERSION", "", slog.Default())

	require.NoError(t, err)
	require.Equal(t, "3.71", client.Microversion)
}

func TestSetupClientMicroversionV2UsesEnvironmentOverride(t *testing.T) {
	t.Setenv("OS_TEST_API_VERSION", "2.42")
	client := newMicroversionTestClient(t, `{
		"version": {
			"id": "v2.1",
			"status": "CURRENT",
			"version": "2.90",
			"min_version": "2.1"
		}
	}`)

	err := SetupClientMicroversionV2(context.Background(), client, "OS_TEST_API_VERSION", "2.50", slog.Default())

	require.NoError(t, err)
	require.Equal(t, "2.42", client.Microversion)
}

func TestSetupClientMicroversionV2RejectsUnsupportedEnvironmentOverride(t *testing.T) {
	t.Setenv("OS_TEST_API_VERSION", "2.99")
	client := newMicroversionTestClient(t, `{
		"version": {
			"id": "v2.1",
			"status": "CURRENT",
			"version": "2.90",
			"min_version": "2.1"
		}
	}`)

	err := SetupClientMicroversionV2(context.Background(), client, "OS_TEST_API_VERSION", "2.50", slog.Default())

	require.Error(t, err)
}

func TestSetupClientMicroversionV2SkipsUnsupportedDiscoveryWithoutOverride(t *testing.T) {
	client := newMicroversionTestClient(t, `{
		"version": {
			"id": "v2.0",
			"status": "CURRENT"
		}
	}`)

	err := SetupClientMicroversionV2(context.Background(), client, "OS_TEST_API_VERSION", "", slog.Default())

	require.NoError(t, err)
	require.Empty(t, client.Microversion)
}

func TestSetupClientMicroversionV2RejectsUnsupportedDiscoveryWithOverride(t *testing.T) {
	t.Setenv("OS_TEST_API_VERSION", "2.42")
	client := newMicroversionTestClient(t, `{
		"version": {
			"id": "v2.0",
			"status": "CURRENT"
		}
	}`)

	err := SetupClientMicroversionV2(context.Background(), client, "OS_TEST_API_VERSION", "", slog.Default())

	require.Error(t, err)
}

func newMicroversionTestClient(t *testing.T, discoveryResponse string) *gophercloud.ServiceClient {
	t.Helper()

	return &gophercloud.ServiceClient{
		ProviderClient: &gophercloud.ProviderClient{
			HTTPClient: http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(bytes.NewBufferString(discoveryResponse)),
					Request:    req,
				}, nil
			})},
		},
		Endpoint: "http://example.test/",
		Type:     "test",
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
