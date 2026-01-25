package integration

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/openstack-exporter/openstack-exporter/integration/clients"
)

func TestImagesIntegration(t *testing.T) {
	clients.RequireLong(t)

	_, cleanup, err := startOpenStackExporter([]string{
		"image",
	})
	if err != nil {
		t.Fatalf("Failed to start OpenStack exporter: %v", err)
	}
	defer cleanup()

	metricsURL := "http://localhost:9180/metrics"

	fetchMetrics := func(
		url string,
		maxTries int,
	) (resp *http.Response, body []byte, err error) {
		for i := 0; i < maxTries; i++ {
			resp, err = http.Get(url)
			if err == nil && resp.StatusCode == http.StatusOK {
				defer resp.Body.Close()
				body, err = io.ReadAll(resp.Body)
				if err == nil {
					return resp, body, nil
				}
				t.Logf(
					"Attempt %d: Failed to read response body: %v",
					i+1,
					err,
				)
			} else {
				var statusCode int
				if resp != nil {
					statusCode = resp.StatusCode
				}
				t.Logf(
					"Attempt %d: Failed to get metrics, status code: %d, error: %v",
					i+1,
					statusCode,
					err,
				)
			}
			if resp != nil && resp.Body != nil {
				resp.Body.Close()
			}
			time.Sleep(1 * time.Second)
		}
		if err != nil {
			return nil, nil, fmt.Errorf(
				"failed to get metrics after %d retries: %w",
				maxTries,
				err,
			)
		}
		return nil, nil, fmt.Errorf(
			"failed to get metrics after %d retries, but the error is nil (this should not happen)",
			maxTries,
		)
	}

	time.Sleep(10 * time.Second)

	const maxTriesFetch = 10
	resp, body, err := fetchMetrics(metricsURL, maxTriesFetch)
	if err != nil {
		// No body to print here (fetch failed before we had a body).
		t.Fatalf("Failed to fetch metrics after multiple retries: %v", err)
	}

	bodyString := string(body)
	// t.Logf("Metrics response body:\n%s", bodyString)

	// Helper to always dump status, endpoint, and full body on failure paths.
	logOnFailure := func(t *testing.T) {
		t.Helper()
		statusCode := 0
		if resp != nil {
			statusCode = resp.StatusCode
		}
		t.Logf(
			"\nStatus Code: %d\nMetrics Endpoint: %s\nResponse Body:\n%s\n",
			statusCode,
			metricsURL,
			bodyString,
		)
	}

	t.Run("openstack_glance_up_metric", func(t *testing.T) {
		if !strings.Contains(bodyString, "openstack_glance_up") {
			logOnFailure(t)
			t.Fatalf(
				"Metric %q not found in metrics response",
				"openstack_glance_up",
			)
		}
		if !strings.Contains(bodyString, "openstack_glance_up 1") {
			logOnFailure(t)
			t.Error(
				"openstack_glance_up metric should have value 1 indicating service is up",
			)
		}
		if !strings.Contains(bodyString, "# HELP openstack_glance_up up") {
			logOnFailure(t)
			t.Error("Missing HELP comment for openstack_glance_up metric")
		}
		if !strings.Contains(bodyString, "# TYPE openstack_glance_up gauge") {
			logOnFailure(t)
			t.Error("Missing TYPE comment for openstack_glance_up metric")
		}
	})

	t.Run("openstack_glance_core_metrics_present", func(t *testing.T) {
		expected := []string{
			"# HELP openstack_glance_images",
			"# HELP openstack_glance_image_bytes",
			"# HELP openstack_glance_image_created_at",
		}
		foundAny := false
		for _, m := range expected {
			if strings.Contains(bodyString, m) {
				foundAny = true
				break
			}
		}
		if !foundAny {
			// Informational, but full body is useful when this triggers.
			logOnFailure(t)
			t.Log(
				"Note: Expected Glance metrics HELP headers not found; Glance may not be fully available",
			)
		}
	})

	// Regex-based specificity checks against actual metric lines
	t.Run("glance_image_bytes_line_format", func(t *testing.T) {
		re := regexp.MustCompile(
			`(?m)^openstack_glance_image_bytes\{id="[^"]+",name="[^"]+",tenant_id="[^"]+"\} [0-9.e\+\-]+$`,
		)
		if !re.MatchString(bodyString) {
			logOnFailure(t)
			t.Errorf(
				"No 'openstack_glance_image_bytes' line matched expected format",
			)
		}
	})

	t.Run("glance_image_created_at_line_format", func(t *testing.T) {
		re := regexp.MustCompile(
			`(?m)^openstack_glance_image_created_at\{hidden="(?:true|false)",id="[^"]+",name="[^"]+",status="[^"]+",tenant_id="[^"]+",visibility="[^"]+"\} [0-9.e\+\-]+$`,
		)
		if !re.MatchString(bodyString) {
			logOnFailure(t)
			t.Errorf(
				"No 'openstack_glance_image_created_at' line matched expected format with labels hidden,id,name,status,tenant_id,visibility",
			)
		}
	})
}
