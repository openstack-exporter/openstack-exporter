package integration

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/openstack-exporter/openstack-exporter/integration/clients"
	"github.com/openstack-exporter/openstack-exporter/integration/funcs"
)

func TestComputeIntegration(t *testing.T) {
	clients.RequireLong(t)

	// Build a compute client
	computeClient, err := clients.NewComputeV2Client()
	if err != nil {
		t.Fatalf("Failed to build compute client: %v", err)
	}

	// Create a real VM
	server, err := funcs.CreateServer(t, computeClient)
	if err != nil {
		t.Fatalf("Could not create test server: %v", err)
	}
	defer funcs.DeleteServer(t, computeClient, server)

	// Helper to print body on failure
	failWithBody := func(t *testing.T, body string, msg string, args ...interface{}) {
		t.Helper()
		log.Printf("Metrics body:\n%s\n", body)
		t.Fatalf(msg, args...)
	}

	// Start exporter
	_, cleanup, err := startOpenStackExporter([]string{"compute"})
	if err != nil {
		t.Fatalf("Failed to start exporter: %v", err)
	}
	defer cleanup()

	metricsURL := "http://localhost:9180/metrics"

	// Allow Nova + exporter time to settle
	time.Sleep(10 * time.Second)

	_, bodyBytes, err := httpGetRetry(metricsURL, 10, t)
	if err != nil {
		t.Fatalf("Failed to fetch metrics: %v", err)
	}
	body := string(bodyBytes)
	t.Logf("Metrics response body:\n%s", body)

	t.Run("nova_server_status_metric_present", func(t *testing.T) {
		// Matches lines like:
		// openstack_nova_server_status{...,id="<uuid>",...} 0
		re := regexp.MustCompile(fmt.Sprintf(
			`openstack_nova_server_status\{[^}]*id="%s"`,
			server.ID,
		))
		if !re.MatchString(body) {
			failWithBody(t, body,
				"Expected server_status metric for server %s not found",
				server.ID,
			)
		}
	})

	t.Run("nova_total_vms_incremented", func(t *testing.T) {
		if !strings.Contains(body, "openstack_nova_total_vms") {
			failWithBody(t, body,
				"Metric openstack_nova_total_vms missing entirely",
			)
		}
		re := regexp.MustCompile(`openstack_nova_total_vms [0-9]+`)
		if !re.MatchString(body) {
			failWithBody(t, body,
				"openstack_nova_total_vms did not contain a numeric value",
			)
		}
	})

	t.Run("nova_server_local_gb_metric_present", func(t *testing.T) {
		// Matches lines like:
		// openstack_nova_server_local_gb{id="<uuid>",name="...",tenant_id="..."} 20
		re := regexp.MustCompile(fmt.Sprintf(
			`openstack_nova_server_local_gb\{[^}]*id="%s"`,
			server.ID,
		))
		if !re.MatchString(body) {
			failWithBody(t, body,
				"Expected server_local_gb metric for server %s not found",
				server.ID,
			)
		}
	})

	t.Run("nova_server_az_label_present", func(t *testing.T) {
		// Ensure the status metric for this server has both id and availability_zone labels.
		re := regexp.MustCompile(fmt.Sprintf(
			`openstack_nova_server_status\{[^}]*id="%s"[^}]*\}[^\n]*\n`,
			regexp.QuoteMeta(server.ID),
		))

		match := re.FindString(body)
		if match == "" || !strings.Contains(match, fmt.Sprintf(`availability_zone="%s"`, server.AvailabilityZone)) {
			failWithBody(t, body,
				"Expected AZ label '%s' for server %s not found",
				server.AvailabilityZone, server.ID,
			)
		}
	})

	t.Run("nova_quota_instances_admin_in_use_present", func(t *testing.T) {
		// Expect a non-zero in_use instances quota for admin tenant:
		// openstack_nova_quota_instances{tenant="admin",type="in_use"} <n> (n > 0)
		re := regexp.MustCompile(
			`openstack_nova_quota_instances\{tenant="admin",type="in_use"\} [1-9][0-9]*`,
		)
		if !re.MatchString(body) {
			failWithBody(t, body,
				"Expected non-zero quota_instances for admin tenant not found",
			)
		}
	})

	t.Run("nova_quota_key_pairs_admin_limit_present", func(t *testing.T) {
		// Expect exactly this line (based on your setup/fixtures):
		// openstack_nova_quota_key_pairs{tenant="admin",type="limit"} 100
		const expected = `openstack_nova_quota_key_pairs{tenant="admin",type="limit"} 100`
		if !strings.Contains(body, expected) {
			failWithBody(t, body,
				"Expected quota_key_pairs limit=100 for admin tenant not found",
			)
		}
	})
}

// Helper: GET with retries
func httpGetRetry(url string, max int, t *testing.T) (*http.Response, []byte, error) {
	for i := 0; i < max; i++ {
		resp, err := http.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err == nil {
				return resp, body, nil
			}
		}
		time.Sleep(1 * time.Second)
	}
	return nil, nil, fmt.Errorf("failed after %d retries", max)
}
