package integration

import (
	"bytes"
	"log"
	"math"
	"testing"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"

	"github.com/openstack-exporter/openstack-exporter/integration/clients"
	"github.com/openstack-exporter/openstack-exporter/integration/funcs"
)

type metricSample struct {
	labels map[string]string
	value  float64
}

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

	_, bodyBytes, err := httpGetRetry(defaultMetricsURL, 10, t)
	if err != nil {
		t.Fatalf("Failed to fetch metrics: %v", err)
	}
	body := string(bodyBytes)
	t.Logf("Metrics response body:\n%s", body)

	metricFamilies, err := parseMetrics(bodyBytes)
	if err != nil {
		failWithBody(t, body, "Failed to parse metrics response: %v", err)
	}

	t.Run("nova_server_status_metric_present", func(t *testing.T) {
		if _, ok := findMetric(metricFamilies, "openstack_nova_server_status", map[string]string{
			"id": server.ID,
		}); !ok {
			failWithBody(t, body,
				"Expected server_status metric for server %s not found",
				server.ID,
			)
		}
	})

	t.Run("nova_total_vms_incremented", func(t *testing.T) {
		sample, ok := findMetric(metricFamilies, "openstack_nova_total_vms", nil)
		if !ok {
			failWithBody(t, body,
				"Metric openstack_nova_total_vms missing entirely",
			)
		}
		if sample.value < 1 {
			failWithBody(t, body,
				"Expected openstack_nova_total_vms to be at least 1, got %v",
				sample.value,
			)
		}
	})

	t.Run("nova_server_local_gb_metric_present", func(t *testing.T) {
		if _, ok := findMetric(metricFamilies, "openstack_nova_server_local_gb", map[string]string{
			"id": server.ID,
		}); !ok {
			failWithBody(t, body,
				"Expected server_local_gb metric for server %s not found",
				server.ID,
			)
		}
	})

	t.Run("nova_server_az_label_present", func(t *testing.T) {
		if _, ok := findMetric(metricFamilies, "openstack_nova_server_status", map[string]string{
			"id":                server.ID,
			"availability_zone": server.AvailabilityZone,
		}); !ok {
			failWithBody(t, body,
				"Expected AZ label '%s' for server %s not found",
				server.AvailabilityZone, server.ID,
			)
		}
	})

	t.Run("nova_server_status_labels_present", func(t *testing.T) {
		sample, ok := findMetric(metricFamilies, "openstack_nova_server_status", map[string]string{
			"id":     server.ID,
			"name":   server.Name,
			"status": "ACTIVE",
			"uuid":   server.ID,
		})
		if !ok {
			failWithBody(t, body,
				"Expected server_status labels for active server %s not found",
				server.ID,
			)
		}
		for _, label := range []string{"flavor_id", "tenant_id", "user_id", "host_id", "hypervisor_hostname"} {
			if sample.labels[label] == "" {
				failWithBody(t, body,
					"Expected server_status metric for server %s to include non-empty %s label",
					server.ID,
					label,
				)
			}
		}
	})

	t.Run("nova_quota_instances_admin_in_use_present", func(t *testing.T) {
		sample, ok := findMetric(metricFamilies, "openstack_nova_quota_instances", map[string]string{
			"tenant": "admin",
			"type":   "in_use",
		})
		if !ok {
			failWithBody(t, body,
				"Expected non-zero quota_instances for admin tenant not found",
			)
		}
		if sample.value < 1 {
			failWithBody(t, body,
				"Expected non-zero quota_instances for admin tenant, got %v",
				sample.value,
			)
		}
		if sample.labels["tenant_id"] == "" {
			failWithBody(t, body,
				"Expected quota_instances for admin tenant to include tenant_id label",
			)
		}
	})

	t.Run("nova_quota_key_pairs_admin_limit_present", func(t *testing.T) {
		sample, ok := findMetric(metricFamilies, "openstack_nova_quota_key_pairs", map[string]string{
			"tenant": "admin",
			"type":   "limit",
		})
		if !ok {
			failWithBody(t, body,
				"Expected quota_key_pairs limit for admin tenant not found",
			)
		}
		if sample.value < 1 {
			failWithBody(t, body,
				"Expected positive quota_key_pairs limit for admin tenant, got %v",
				sample.value,
			)
		}
		if sample.labels["tenant_id"] == "" {
			failWithBody(t, body,
				"Expected quota_key_pairs limit for admin tenant to include tenant_id label",
			)
		}
	})

	t.Run("nova_quota_admin_usage_metrics_present", func(t *testing.T) {
		for _, tc := range []struct {
			name      string
			metric    string
			quotaType string
			minValue  float64
		}{
			{name: "cores_in_use", metric: "openstack_nova_quota_cores", quotaType: "in_use", minValue: 1},
			{name: "ram_in_use", metric: "openstack_nova_quota_ram", quotaType: "in_use", minValue: 1},
			{name: "instances_limit", metric: "openstack_nova_quota_instances", quotaType: "limit", minValue: 1},
		} {
			t.Run(tc.name, func(t *testing.T) {
				sample, ok := findMetric(metricFamilies, tc.metric, map[string]string{
					"tenant": "admin",
					"type":   tc.quotaType,
				})
				if !ok {
					failWithBody(t, body,
						"Expected %s metric for admin tenant with type %s not found",
						tc.metric,
						tc.quotaType,
					)
				}
				if sample.labels["tenant_id"] == "" {
					failWithBody(t, body,
						"Expected %s metric for admin tenant to include tenant_id label",
						tc.metric,
					)
				}
				if sample.value < tc.minValue {
					failWithBody(t, body,
						"Expected %s metric for admin tenant with type %s to be at least %v, got %v",
						tc.metric,
						tc.quotaType,
						tc.minValue,
						sample.value,
					)
				}
			})
		}
	})
}

func parseMetrics(body []byte) (map[string][]metricSample, error) {
	parser := expfmt.TextParser{}
	metricFamilies, err := parser.TextToMetricFamilies(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	samples := make(map[string][]metricSample, len(metricFamilies))
	for name, family := range metricFamilies {
		for _, metric := range family.GetMetric() {
			value, ok := metricValue(metric)
			if !ok {
				continue
			}
			sample := metricSample{
				labels: make(map[string]string, len(metric.GetLabel())),
				value:  value,
			}
			for _, label := range metric.GetLabel() {
				sample.labels[label.GetName()] = label.GetValue()
			}
			samples[name] = append(samples[name], sample)
		}
	}

	return samples, nil
}

func metricValue(metric *dto.Metric) (float64, bool) {
	switch {
	case metric.GetGauge() != nil:
		return metric.GetGauge().GetValue(), true
	case metric.GetCounter() != nil:
		return metric.GetCounter().GetValue(), true
	case metric.GetUntyped() != nil:
		return metric.GetUntyped().GetValue(), true
	default:
		return math.NaN(), false
	}
}

func findMetric(metricFamilies map[string][]metricSample, name string, labels map[string]string) (metricSample, bool) {
	for _, sample := range metricFamilies[name] {
		if labelsMatch(sample.labels, labels) {
			return sample, true
		}
	}
	return metricSample{}, false
}

func labelsMatch(got, want map[string]string) bool {
	for name, value := range want {
		if got[name] != value {
			return false
		}
	}
	return true
}
