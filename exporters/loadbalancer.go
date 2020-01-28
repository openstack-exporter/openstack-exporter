package exporters

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/loadbalancers"
	"github.com/prometheus/client_golang/prometheus"
)

var loadbalancer_status = []string{
	// Octavia API v2 entities have two status codes present in the response body.
	// The provisioning_status describes the lifecycle status of the entity while the operating_status provides the observed status of the entity.
	// Here we put operating_status in metrics value and provisioning_status in metrics label
	"ONLINE",     // Entity is operating normally. All pool members are healthy
	"DRAINING",   // The member is not accepting new connections
	"OFFLINE",    // Entity is administratively disabled
	"ERROR",      // The entity has failed. The member is failing it's health monitoring checks. All of the pool members are in ERROR
	"NO_MONITOR", // No health monitor is configured for this entity and it's status is unknown
}

func mapLoadbalancerStatus(current string) int {
	for idx, status := range loadbalancer_status {
		if current == status {
			return idx
		}
	}
	return -1
}

type LoadbalancerExporter struct {
	BaseOpenStackExporter
}

var defaultLoadbalancerMetrics = []Metric{
	{Name: "total_loadbalancers", Fn: ListAllLoadbalancers},
	{Name: "loadbalancer_status", Labels: []string{"id", "name", "project_id", "operating_status", "provisioning_status", "provider", "vip_address"}},
}

func NewLoadbalancerExporter(client *gophercloud.ServiceClient, prefix string, disabledMetrics []string) (*LoadbalancerExporter, error) {
	exporter := LoadbalancerExporter{
		BaseOpenStackExporter{
			Name:            "loadbalancer",
			Prefix:          prefix,
			Client:          client,
			DisabledMetrics: disabledMetrics,
		},
	}
	for _, metric := range defaultLoadbalancerMetrics {
		exporter.AddMetric(metric.Name, metric.Fn, metric.Labels, nil)
	}
	return &exporter, nil
}

func ListAllLoadbalancers(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allLoadbalancers []loadbalancers.LoadBalancer
	allPagesLoadbalancers, err := loadbalancers.List(exporter.Client, loadbalancers.ListOpts{}).AllPages()
	if err != nil {
		return err
	}
	allLoadbalancers, err = loadbalancers.ExtractLoadBalancers(allPagesLoadbalancers)
	if err != nil {
		return err
	}
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["total_loadbalancers"].Metric,
		prometheus.GaugeValue, float64(len(allLoadbalancers)))
	// Loadbalancer status metrics
	for _, loadbalancer := range allLoadbalancers {
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["loadbalancer_status"].Metric,
			prometheus.GaugeValue, float64(mapLoadbalancerStatus(loadbalancer.OperatingStatus)), loadbalancer.ID, loadbalancer.Name, loadbalancer.ProjectID,
			loadbalancer.OperatingStatus, loadbalancer.ProvisioningStatus, loadbalancer.Provider, loadbalancer.VipAddress)
	}
	return nil
}
