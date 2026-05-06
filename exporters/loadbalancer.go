package exporters

import (
	"log/slog"
	"time"

	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/amphorae"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/loadbalancers"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/pools"
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

var amphora_status = []string{
	// The status of the amphora. One of: BOOTING, ALLOCATED, READY, PENDING_CREATE, PENDING_DELETE, DELETED, ERROR.
	"BOOTING",
	"ALLOCATED",
	"READY",
	"PENDING_CREATE",
	"PENDING_DELETE",
	"DELETED",
	"ERROR",
}

var pool_status = []string{
	// Loadbalancer pool provisioning status. One of: ACTIVE, DELETED, ERROR, PENDING_CREATE, PENDING_UPDATE, PENDING_DELETE.
	"ACTIVE",
	"DELETED",
	"ERROR",
	"PENDING_CREATE",
	"PENDING_UPDATE",
	"PENDING_DELETE",
}

func mapLoadbalancerStatus(current string) int {
	for idx, status := range loadbalancer_status {
		if current == status {
			return idx
		}
	}
	return -1
}

func mapAmphoraStatus(current string) int {

	for idx, status := range amphora_status {
		if current == status {
			return idx
		}
	}
	return -1
}

func mapPoolStatus(current string) int {

	for idx, status := range pool_status {
		if current == status {
			return idx
		}
	}
	return -1
}

const LOADBALANCER_SERVICE string = "loadbalancer"

type LoadbalancerExporter struct {
	BaseOpenStackExporter
}

var defaultLoadbalancerMetrics = []Metric{
	{Name: "total_loadbalancers", Fn: ListAllLoadbalancers},
	{Name: "loadbalancer_status", Labels: []string{"id", "name", "project_id", "operating_status", "provisioning_status", "provider", "vip_address"}},
	{Name: "total_amphorae", Fn: ListAllAmphorae},
	{Name: "amphora_status", Labels: []string{"id", "loadbalancer_id", "compute_id", "status", "role", "lb_network_ip", "ha_ip", "cert_expiration"}},
	{Name: "total_pools", Fn: ListAllPools},
	{Name: "pool_status", Labels: []string{"id", "provisioning_status", "name", "loadbalancers", "protocol", "lb_algorithm", "operating_status", "project_id"}},
}

func NewLoadbalancerExporter(config *ExporterConfig, logger *slog.Logger) (*LoadbalancerExporter, error) {
	exporter := LoadbalancerExporter{
		BaseOpenStackExporter{
			Name:           "loadbalancer",
			ExporterConfig: *config,
			logger:         logger,
		},
	}
	for _, metric := range defaultLoadbalancerMetrics {
		labels := computeMetricLabels(LOADBALANCER_SERVICE, metric, exporter.ExtraLabels)
		constLabels := computeConstantLabels(LOADBALANCER_SERVICE, metric, exporter.ExtraLabels)
		exporter.AddMetric(metric.Name, metric.Fn, labels, metric.DeprecatedVersion, constLabels)
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
	lbStatusSpec := exporter.ExtraLabels.Extract(LOADBALANCER_SERVICE, "loadbalancer_status")
	// Loadbalancer status metrics
	for _, loadbalancer := range allLoadbalancers {
		extraValues := resolveExtraLabelValues(loadbalancer, lbStatusSpec)
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["loadbalancer_status"].Metric,
			prometheus.GaugeValue, float64(mapLoadbalancerStatus(loadbalancer.OperatingStatus)), append([]string{loadbalancer.ID, loadbalancer.Name, loadbalancer.ProjectID,
				loadbalancer.OperatingStatus, loadbalancer.ProvisioningStatus, loadbalancer.Provider, loadbalancer.VipAddress}, extraValues...)...)
	}
	return nil
}

func ListAllAmphorae(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allAmphorae []amphorae.Amphora
	allPagesAmphorae, err := amphorae.List(exporter.Client, amphorae.ListOpts{}).AllPages()
	if err != nil {
		return err
	}
	allAmphorae, err = amphorae.ExtractAmphorae(allPagesAmphorae)
	if err != nil {
		return err
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["total_amphorae"].Metric,
		prometheus.GaugeValue, float64(len(allAmphorae)))
	amphoraStatusSpec := exporter.ExtraLabels.Extract(LOADBALANCER_SERVICE, "amphora_status")
	// Loadbalancer status metrics
	for _, amphora := range allAmphorae {
		extraValues := resolveExtraLabelValues(amphora, amphoraStatusSpec)
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["amphora_status"].Metric,
			prometheus.GaugeValue, float64(mapAmphoraStatus(amphora.Status)), append([]string{amphora.ID, amphora.LoadbalancerID, amphora.ComputeID, amphora.Status,
				amphora.Role, amphora.LBNetworkIP, amphora.HAIP, amphora.CertExpiration.Format(time.RFC3339)}, extraValues...)...)
	}
	return nil
}

func ListAllPools(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allPools []pools.Pool
	allPagesPools, err := pools.List(exporter.Client, pools.ListOpts{}).AllPages()
	if err != nil {
		return err
	}
	allPools, err = pools.ExtractPools(allPagesPools)
	if err != nil {
		return err
	}
	ch <- prometheus.MustNewConstMetric(exporter.Metrics["total_pools"].Metric,
		prometheus.GaugeValue, float64(len(allPools)))
	poolStatusSpec := exporter.ExtraLabels.Extract(LOADBALANCER_SERVICE, "pool_status")
	for _, pool := range allPools {
		extraValues := resolveExtraLabelValues(pool, poolStatusSpec)
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["pool_status"].Metric,
			prometheus.GaugeValue, float64(mapPoolStatus(pool.ProvisioningStatus)), append([]string{pool.ID, pool.ProvisioningStatus, pool.Name,
				lbsLabels(pool.Loadbalancers), pool.Protocol, pool.LBMethod, pool.OperatingStatus, pool.ProjectID}, extraValues...)...)
	}
	return nil
}

func lbsLabels(lbs []pools.LoadBalancerID) string {
	label := ""
	for i, l := range lbs {
		if i == 0 {
			label += l.ID
		} else {
			label += "," + l.ID
		}
	}
	return label
}
