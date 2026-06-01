package integration

import (
	"log"
	"testing"

	"github.com/openstack-exporter/openstack-exporter/integration/clients"
	"github.com/openstack-exporter/openstack-exporter/integration/funcs"
)

func TestNetworkingIntegration(t *testing.T) {
	clients.RequireLong(t)

	cleanup := startNetworkExporter(t)
	defer cleanup()

	metricFamilies, body := scrapeNetworkMetrics(t, "")

	t.Run("openstack_neutron_up_metric", func(t *testing.T) {
		sample, ok := findMetric(metricFamilies, "openstack_neutron_up", nil)
		if !ok {
			failNetworkingWithBody(t, body,
				"Metric %q not found in metrics response",
				"openstack_neutron_up",
			)
		}
		if sample.value != 1 {
			failNetworkingWithBody(t, body,
				"openstack_neutron_up metric should have value 1 indicating service is up, got %v",
				sample.value,
			)
		}
	})

	t.Run("openstack_neutron_core_metrics_present", func(t *testing.T) {
		expected := []string{
			"openstack_neutron_networks",
			"openstack_neutron_ports",
			"openstack_neutron_subnets",
			"openstack_neutron_router",
		}
		foundAny := false
		for _, m := range expected {
			if _, ok := metricFamilies[m]; ok {
				foundAny = true
				break
			}
		}
		if !foundAny {
			failNetworkingWithBody(t, body,
				"Expected Neutron core metrics not found; Neutron may not be fully available",
			)
		}
	})

	t.Run("neutron_network_labels_present", func(t *testing.T) {
		for _, sample := range metricFamilies["openstack_neutron_network"] {
			if sample.labels["id"] != "" &&
				sample.labels["name"] != "" &&
				sample.labels["is_external"] != "" &&
				sample.labels["is_shared"] != "" {
				if _, ok := sample.labels["provider_network_type"]; ok {
					return
				}
			}
		}
		failNetworkingWithBody(t, body,
			"No 'openstack_neutron_network' metric contained required labels (id,name,is_external,is_shared,provider_network_type)",
		)
	})
}

func TestNetworkingNetworkCreateDeleteUpdatesExporterMetrics(t *testing.T) {
	clients.RequireLong(t)

	networkClient, err := clients.NewNetworkV2Client()
	if err != nil {
		t.Fatalf("Failed to build network client: %v", err)
	}

	cleanup := startNetworkExporter(t)
	defer cleanup()

	network, err := funcs.CreateNetwork(t, networkClient)
	if err != nil {
		t.Fatalf("Could not create test network: %v", err)
	}

	metricFamilies, body := scrapeNetworkMetrics(t, "after network create")

	if _, ok := findMetric(metricFamilies, "openstack_neutron_network", map[string]string{
		"id":   network.ID,
		"name": network.Name,
	}); !ok {
		funcs.DeleteNetwork(t, networkClient, network)
		failNetworkingWithBody(t, body,
			"Expected network metric for created network %s not found",
			network.ID,
		)
	}

	funcs.DeleteNetwork(t, networkClient, network)

	metricFamilies, body = scrapeNetworkMetrics(t, "after network delete")

	if _, ok := findMetric(metricFamilies, "openstack_neutron_network", map[string]string{
		"id": network.ID,
	}); ok {
		failNetworkingWithBody(t, body,
			"Expected network metric for deleted network %s to disappear",
			network.ID,
		)
	}
}

func TestNetworkingSubnetCreateDeleteUpdatesExporterMetrics(t *testing.T) {
	clients.RequireLong(t)

	networkClient, err := clients.NewNetworkV2Client()
	if err != nil {
		t.Fatalf("Failed to build network client: %v", err)
	}

	cleanup := startNetworkExporter(t)
	defer cleanup()

	network, err := funcs.CreateNetwork(t, networkClient)
	if err != nil {
		t.Fatalf("Could not create test network: %v", err)
	}
	networkDeleted := false
	t.Cleanup(func() {
		if !networkDeleted {
			funcs.DeleteNetwork(t, networkClient, network)
		}
	})

	subnet, err := funcs.CreateSubnet(t, networkClient, network)
	if err != nil {
		t.Fatalf("Could not create test subnet: %v", err)
	}
	subnetDeleted := false
	t.Cleanup(func() {
		if !subnetDeleted {
			funcs.DeleteSubnet(t, networkClient, subnet)
		}
	})

	metricFamilies, body := scrapeNetworkMetrics(t, "after subnet create")
	if _, ok := findMetric(metricFamilies, "openstack_neutron_subnet", map[string]string{
		"id":         subnet.ID,
		"name":       subnet.Name,
		"network_id": network.ID,
		"cidr":       subnet.CIDR,
	}); !ok {
		failNetworkingWithBody(t, body,
			"Expected subnet metric for created subnet %s not found",
			subnet.ID,
		)
	}

	funcs.DeleteSubnet(t, networkClient, subnet)
	subnetDeleted = true

	metricFamilies, body = scrapeNetworkMetrics(t, "after subnet delete")
	if _, ok := findMetric(metricFamilies, "openstack_neutron_subnet", map[string]string{
		"id": subnet.ID,
	}); ok {
		failNetworkingWithBody(t, body,
			"Expected subnet metric for deleted subnet %s to disappear",
			subnet.ID,
		)
	}

	funcs.DeleteNetwork(t, networkClient, network)
	networkDeleted = true
}

func TestNetworkingPortCreateDeleteUpdatesExporterMetrics(t *testing.T) {
	clients.RequireLong(t)

	networkClient, err := clients.NewNetworkV2Client()
	if err != nil {
		t.Fatalf("Failed to build network client: %v", err)
	}

	cleanup := startNetworkExporter(t)
	defer cleanup()

	network, err := funcs.CreateNetwork(t, networkClient)
	if err != nil {
		t.Fatalf("Could not create test network: %v", err)
	}
	networkDeleted := false
	t.Cleanup(func() {
		if !networkDeleted {
			funcs.DeleteNetwork(t, networkClient, network)
		}
	})

	port, err := funcs.CreatePort(t, networkClient, network)
	if err != nil {
		t.Fatalf("Could not create test port: %v", err)
	}
	portDeleted := false
	t.Cleanup(func() {
		if !portDeleted {
			funcs.DeletePort(t, networkClient, port)
		}
	})

	metricFamilies, body := scrapeNetworkMetrics(t, "after port create")
	if _, ok := findMetric(metricFamilies, "openstack_neutron_port", map[string]string{
		"uuid":        port.ID,
		"network_id":  network.ID,
		"mac_address": port.MACAddress,
	}); !ok {
		failNetworkingWithBody(t, body,
			"Expected port metric for created port %s not found",
			port.ID,
		)
	}

	funcs.DeletePort(t, networkClient, port)
	portDeleted = true

	metricFamilies, body = scrapeNetworkMetrics(t, "after port delete")
	if _, ok := findMetric(metricFamilies, "openstack_neutron_port", map[string]string{
		"uuid": port.ID,
	}); ok {
		failNetworkingWithBody(t, body,
			"Expected port metric for deleted port %s to disappear",
			port.ID,
		)
	}

	funcs.DeleteNetwork(t, networkClient, network)
	networkDeleted = true
}

func TestNetworkingIPAvailabilityIncludesCreatedSubnet(t *testing.T) {
	clients.RequireLong(t)

	networkClient, err := clients.NewNetworkV2Client()
	if err != nil {
		t.Fatalf("Failed to build network client: %v", err)
	}

	cleanup := startNetworkExporter(t)
	defer cleanup()

	network, err := funcs.CreateNetwork(t, networkClient)
	if err != nil {
		t.Fatalf("Could not create test network: %v", err)
	}
	networkDeleted := false
	t.Cleanup(func() {
		if !networkDeleted {
			funcs.DeleteNetwork(t, networkClient, network)
		}
	})

	subnet, err := funcs.CreateSubnet(t, networkClient, network)
	if err != nil {
		t.Fatalf("Could not create test subnet: %v", err)
	}
	subnetDeleted := false
	t.Cleanup(func() {
		if !subnetDeleted {
			funcs.DeleteSubnet(t, networkClient, subnet)
		}
	})

	metricFamilies, body := scrapeNetworkMetrics(t, "after subnet create")
	if _, ok := findMetric(metricFamilies, "openstack_neutron_network_ip_availabilities_total", map[string]string{
		"network_id":   network.ID,
		"network_name": network.Name,
		"subnet_name":  subnet.Name,
		"cidr":         subnet.CIDR,
		"ip_version":   "4",
	}); !ok {
		failNetworkingWithBody(t, body,
			"Expected network IP availability total metric for subnet %s not found",
			subnet.ID,
		)
	}

	if _, ok := findMetric(metricFamilies, "openstack_neutron_network_ip_availabilities_used", map[string]string{
		"network_id":   network.ID,
		"network_name": network.Name,
		"subnet_name":  subnet.Name,
		"cidr":         subnet.CIDR,
		"ip_version":   "4",
	}); !ok {
		failNetworkingWithBody(t, body,
			"Expected network IP availability used metric for subnet %s not found",
			subnet.ID,
		)
	}

	funcs.DeleteSubnet(t, networkClient, subnet)
	subnetDeleted = true
	funcs.DeleteNetwork(t, networkClient, network)
	networkDeleted = true
}

func TestNetworkingQuotaMetricsHaveExpectedLabels(t *testing.T) {
	clients.RequireLong(t)

	cleanup := startNetworkExporter(t)
	defer cleanup()

	metricFamilies, body := scrapeNetworkMetrics(t, "")
	for _, metricName := range []string{
		"openstack_neutron_quota_network",
		"openstack_neutron_quota_subnet",
		"openstack_neutron_quota_port",
		"openstack_neutron_quota_router",
		"openstack_neutron_quota_floatingip",
		"openstack_neutron_quota_security_group",
		"openstack_neutron_quota_security_group_rule",
	} {
		sample, ok := findMetric(metricFamilies, metricName, map[string]string{
			"type": "limit",
		})
		if !ok {
			failNetworkingWithBody(t, body,
				"Expected %s metric with type=limit not found",
				metricName,
			)
		}
		for _, label := range []string{"tenant", "tenant_id", "type"} {
			if sample.labels[label] == "" {
				failNetworkingWithBody(t, body,
					"Expected %s metric to include non-empty %s label",
					metricName,
					label,
				)
			}
		}
	}
}

func startNetworkExporter(t *testing.T) func() {
	t.Helper()

	_, cleanup, err := startOpenStackExporter([]string{"network"})
	if err != nil {
		t.Fatalf("Failed to start exporter: %v", err)
	}
	return cleanup
}

func scrapeNetworkMetrics(t *testing.T, context string) (map[string][]metricSample, string) {
	t.Helper()

	_, bodyBytes, err := httpGetRetry(defaultMetricsURL, 10, t)
	if err != nil {
		if context == "" {
			t.Fatalf("Failed to fetch metrics: %v", err)
		}
		t.Fatalf("Failed to fetch metrics %s: %v", context, err)
	}

	body := string(bodyBytes)
	metricFamilies, err := parseMetrics(bodyBytes)
	if err != nil {
		if context == "" {
			failNetworkingWithBody(t, body, "Failed to parse metrics response: %v", err)
		}
		failNetworkingWithBody(t, body, "Failed to parse metrics response %s: %v", context, err)
	}

	return metricFamilies, body
}

func failNetworkingWithBody(t *testing.T, body string, msg string, args ...interface{}) {
	t.Helper()
	log.Printf("Metrics body:\n%s\n", body)
	t.Fatalf(msg, args...)
}
