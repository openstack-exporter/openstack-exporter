package integration

import (
	"log"
	"testing"

	"github.com/openstack-exporter/openstack-exporter/integration/clients"
)

func TestPlacementResourceTraitsIntegration(t *testing.T) {
	clients.RequireLong(t)

	_, cleanup, err := startOpenStackExporter([]string{"placement"})
	if err != nil {
		t.Fatalf("Failed to start exporter: %v", err)
	}
	defer cleanup()

	_, bodyBytes, err := httpGetRetry(defaultMetricsURL, 10, t)
	if err != nil {
		t.Fatalf("Failed to fetch metrics: %v", err)
	}

	metricFamilies, err := parseMetrics(bodyBytes)
	if err != nil {
		log.Printf("Metrics body:\n%s", bodyBytes)
		t.Fatalf("Failed to parse metrics response: %v", err)
	}

	if samples := metricFamilies["openstack_placement_resource_traits"]; len(samples) == 0 {
		t.Fatalf("Expected openstack_placement_resource_traits in metrics response")
	}

	for _, sample := range metricFamilies["openstack_placement_resource_total"] {
		if _, ok := sample.labels["resource_traits"]; !ok {
			t.Fatalf("Expected openstack_placement_resource_total to include resource_traits: %#v", sample.labels)
		}
	}
}
