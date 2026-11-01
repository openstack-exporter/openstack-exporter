package main

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/require"
)

func TestProbePreservesHealthyMetricsOnServiceFailure(t *testing.T) {
	for _, stalled := range []bool{false, true} {
		t.Run(fmt.Sprintf("stalled=%t", stalled), func(t *testing.T) {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()
			t.Setenv("OS_CLIENT_CONFIG_FILE", "exporters/fixtures/test_config.yaml")
			t.Setenv("OS_COMPUTE_API_VERSION", "2.87")
			oldTimeout, oldPrefix, oldCache, oldDisabled := *collectTimeout, *prefix, *cacheEnable, *disabledMetrics
			*collectTimeout = 25 * time.Millisecond
			t.Cleanup(func() {
				*collectTimeout, *prefix, *cacheEnable, *disabledMetrics = oldTimeout, oldPrefix, oldCache, oldDisabled
			})
			*prefix = "openstack"
			*cacheEnable = false
			*disabledMetrics = []string{"nova-running_vms", "nova-security_groups"}
			var logs bytes.Buffer
			httpmock.RegisterNoResponder(func(r *http.Request) (response *http.Response, err error) {
				defer func() {
					if response != nil {
						response.Request = r
					}
				}()
				if fixture, ok := map[string]string{"/compute/": "nova_api_discovery", "/compute/v2.1/": "nova_api_v2.1", "/glance/": "glance_api_discovery"}[r.URL.Path]; ok {
					data, err := os.ReadFile("exporters/fixtures/" + fixture + ".json")
					require.NoError(t, err)
					response := httpmock.NewStringResponse(200, strings.ReplaceAll(string(data), `"version": "2.14"`, `"version": "2.95"`))
					response.Header.Set("Content-Type", "application/json")
					return response, nil
				}
				if r.Method == "POST" && r.URL.Path == "/v3/auth/tokens" {
					data, err := os.ReadFile("exporters/fixtures/tokens.json")
					require.NoError(t, err)
					response := httpmock.NewStringResponse(201, string(data))
					response.Header.Set("X-Subject-Token", "test-token")
					response.Header.Set("Content-Type", "application/json")
					return response, nil
				}
				if r.URL.Path == "/glance/v2/images" {
					return httpmock.NewJsonResponse(200, map[string]any{"images": []any{}})
				}
				if stalled {
					<-r.Context().Done()
					return nil, r.Context().Err()
				}
				return httpmock.NewJsonResponse(500, map[string]any{"error": "Nova unavailable"})
			})
			handler := probeHandler([]string{"compute", "image"}, slog.New(slog.NewTextHandler(&logs, nil)))
			request := httptest.NewRequest("GET", "/probe?cloud=test.cloud", nil)
			response := httptest.NewRecorder()
			start := time.Now()
			handler(response, request)
			require.Equal(t, 200, response.Code, logs.String()+response.Body.String())
			require.Contains(t, response.Body.String(), "openstack_glance_images 0", logs.String())
			require.Contains(t, response.Body.String(), "openstack_nova_up 0", logs.String())
			require.True(t, strings.Contains(logs.String(), "Failed to collect metric"))
			require.Less(t, time.Since(start), time.Second)
			if stalled {
				require.Contains(t, logs.String(), "context deadline exceeded")
			}
		})
	}
}
