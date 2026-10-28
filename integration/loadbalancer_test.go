package integration

import (
	"testing"

	"github.com/openstack-exporter/openstack-exporter/integration/clients"
	"github.com/openstack-exporter/openstack-exporter/integration/funcs"
)

func TestLoadBalancerIntegration(t *testing.T) {
	clients.RequireLong(t)

	networkClient := funcs.NewNetworkClient(t)
	loadBalancerClient := funcs.NewLoadBalancerClient(t)
	network, _ := funcs.MustCreateNetwork(t, networkClient)
	subnet, _ := funcs.MustCreateSubnet(t, networkClient, network)
	loadBalancer := funcs.MustCreateLoadBalancer(t, loadBalancerClient, subnet.ID)
	listener := funcs.MustCreateListener(t, loadBalancerClient, loadBalancer.ID)

	cleanup := startExporter(t, "load-balancer")
	defer cleanup()

	metrics := scrapeLoggedMetrics(t, "after load balancer and listener create")

	t.Run("openstack_loadbalancer_up_metric", func(t *testing.T) {
		metrics.requireUp(t, "openstack_loadbalancer_up")
	})

	t.Run("loadbalancer_status_and_totals", func(t *testing.T) {
		metrics.requireMinValue(t, "openstack_loadbalancer_total_loadbalancers", nil, 1)
		metrics.requireMinValue(t, "openstack_loadbalancer_total_listeners", nil, 1)
		metrics.requireLabels(t, "openstack_loadbalancer_loadbalancer_status", labels{"id": loadBalancer.ID},
			"id", "name", "project_id", "operating_status", "provisioning_status", "provider", "vip_address")
	})

	t.Run("loadbalancer_stats_metrics", func(t *testing.T) {
		for _, metricName := range []string{
			"openstack_loadbalancer_stats_bytes_in",
			"openstack_loadbalancer_stats_bytes_out",
			"openstack_loadbalancer_stats_active_connections",
			"openstack_loadbalancer_stats_total_connections",
			"openstack_loadbalancer_stats_request_errors",
		} {
			metrics.requireLabels(t, metricName, labels{"id": loadBalancer.ID},
				"id", "name", "project_id", "operating_status", "provisioning_status", "provider", "vip_address")
		}
	})

	t.Run("listener_stats_metrics", func(t *testing.T) {
		for _, metricName := range []string{
			"openstack_loadbalancer_listener_stats_bytes_in",
			"openstack_loadbalancer_listener_stats_bytes_out",
			"openstack_loadbalancer_listener_stats_active_connections",
			"openstack_loadbalancer_listener_stats_total_connections",
			"openstack_loadbalancer_listener_stats_request_errors",
		} {
			metrics.requireLabels(t, metricName, labels{"id": listener.ID, "loadbalancer_id": loadBalancer.ID},
				"id", "name", "project_id", "operating_status", "provisioning_status", "protocol", "protocol_port", "loadbalancer_id")
		}
	})
}
