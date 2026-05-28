package integration

import (
	"regexp"
	"strings"
	"testing"

	"github.com/openstack-exporter/openstack-exporter/integration/clients"
)

func TestIdentityIntegration(t *testing.T) {
	clients.RequireLong(t)

	_, cleanup, err := startOpenStackExporter([]string{
		"identity",
	})
	if err != nil {
		t.Fatalf("Failed to start OpenStack exporter: %v", err)
	}
	defer cleanup()

	const maxTriesFetch = 10
	resp, body, err := httpGetRetry(defaultMetricsURL, maxTriesFetch, t)
	if err != nil {
		t.Fatalf("Failed to fetch metrics after multiple retries: %v", err)
	}

	bodyString := string(body)

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
			defaultMetricsURL,
			bodyString,
		)
	}

	t.Run("openstack_identity_up_metric", func(t *testing.T) {
		if !strings.Contains(bodyString, "openstack_identity_up") {
			logOnFailure(t)
			t.Fatalf("Metric %q not found in metrics response", "openstack_identity_up")
		}
		if !strings.Contains(bodyString, "openstack_identity_up 1") {
			logOnFailure(t)
			t.Error(
				"openstack_identity_up metric should have value 1 indicating service is up",
			)
		}
		if !strings.Contains(bodyString, "# HELP openstack_identity_up up") {
			logOnFailure(t)
			t.Error("Missing HELP comment for openstack_identity_up metric")
		}
		if !strings.Contains(bodyString, "# TYPE openstack_identity_up gauge") {
			logOnFailure(t)
			t.Error("Missing TYPE comment for openstack_identity_up metric")
		}
	})

	t.Run("openstack_identity_core_metrics_present", func(t *testing.T) {
		expected := []string{
			"# HELP openstack_identity_projects",
			"# HELP openstack_identity_users",
			"# HELP openstack_identity_domains",
		}
		foundAny := false
		for _, m := range expected {
			if strings.Contains(bodyString, m) {
				foundAny = true
				break
			}
		}
		if !foundAny {
			// Informational, but still useful to have the full body when this triggers.
			logOnFailure(t)
			t.Log(
				"Note: Expected Identity metrics HELP headers not found; Keystone may not be fully available",
			)
		}
	})

	// Regex-based specificity checks against project_info metric line
	t.Run("identity_project_info_line_format", func(t *testing.T) {
		lineRe := regexp.MustCompile(
			`(?m)^openstack_identity_project_info\{.*\} [0-9.e\+\-]+$`,
		)
		lines := lineRe.FindAllString(bodyString, -1)
		if len(lines) == 0 {
			logOnFailure(t)
			t.Fatalf(
				"No 'openstack_identity_project_info' lines found matching expected format",
			)
		}
		// Validate key labels exist within a matched line (order-independent)
		labelChecks := []*regexp.Regexp{
			regexp.MustCompile(`\bid="[^"]+"`),
			regexp.MustCompile(`\bname="[^"]+"`),
			regexp.MustCompile(`\bdomain_id="[^"]+"`),
			regexp.MustCompile(`\benabled="(?:true|false)"`),
			regexp.MustCompile(`\bparent_id="[^"]+"`),
		}
		matched := false
		for _, l := range lines {
			ok := true
			for _, re := range labelChecks {
				if !re.MatchString(l) {
					ok = false
					break
				}
			}
			if ok {
				matched = true
				break
			}
		}
		if !matched {
			logOnFailure(t)
			t.Errorf(
				"No 'openstack_identity_project_info' line contained required labels (id,name,domain_id,enabled,parent_id)",
			)
		}
	})
}
