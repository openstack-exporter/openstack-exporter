package exporters

import (
	"log/slog"
	"strconv"
	"time"

	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/amphorae"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/listeners"
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

type LoadbalancerExporter struct {
	BaseOpenStackExporter
}

var loadbalancerMetricLabels = []string{"id", "name", "project_id", "operating_status", "provisioning_status", "provider", "vip_address"}

var listenerStatsMetricLabels = []string{"id", "name", "project_id", "operating_status", "provisioning_status", "protocol", "protocol_port", "loadbalancer_id"}

var defaultLoadbalancerMetrics = []Metric{
	{Name: "total_loadbalancers", Fn: ListAllLoadbalancers},
	{Name: "loadbalancer_status", Labels: loadbalancerMetricLabels},
	{Name: "stats_bytes_in", Labels: loadbalancerMetricLabels, Slow: true},
	{Name: "stats_bytes_out", Labels: loadbalancerMetricLabels, Slow: true},
	{Name: "stats_active_connections", Labels: loadbalancerMetricLabels, Slow: true},
	{Name: "stats_total_connections", Labels: loadbalancerMetricLabels, Slow: true},
	{Name: "stats_request_errors", Labels: loadbalancerMetricLabels, Slow: true},
	{Name: "total_listeners", Fn: ListAllListeners},
	{Name: "listener_stats_bytes_in", Labels: listenerStatsMetricLabels, Slow: true},
	{Name: "listener_stats_bytes_out", Labels: listenerStatsMetricLabels, Slow: true},
	{Name: "listener_stats_active_connections", Labels: listenerStatsMetricLabels, Slow: true},
	{Name: "listener_stats_total_connections", Labels: listenerStatsMetricLabels, Slow: true},
	{Name: "listener_stats_request_errors", Labels: listenerStatsMetricLabels, Slow: true},
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
		if !exporter.isSlowMetric(&metric) {
			exporter.AddMetric(metric.Name, metric.Fn, metric.Labels, metric.DeprecatedVersion, nil)
		}
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

		// Loadbalancer stats metrics (only if enabled)
		if _, hasStatsMetrics := exporter.Metrics["stats_bytes_in"]; hasStatsMetrics {
			stats, err := loadbalancers.GetStats(exporter.Client, loadbalancer.ID).Extract()
			if err != nil {
				exporter.logger.Warn("failed to get loadbalancer stats", "id", loadbalancer.ID, "error", err)
				continue
			}

			labelValues := []string{loadbalancer.ID, loadbalancer.Name, loadbalancer.ProjectID,
				loadbalancer.OperatingStatus, loadbalancer.ProvisioningStatus, loadbalancer.Provider, loadbalancer.VipAddress}

			ch <- prometheus.MustNewConstMetric(exporter.Metrics["stats_bytes_in"].Metric,
				prometheus.GaugeValue, float64(stats.BytesIn), labelValues...)
			ch <- prometheus.MustNewConstMetric(exporter.Metrics["stats_bytes_out"].Metric,
				prometheus.GaugeValue, float64(stats.BytesOut), labelValues...)
			ch <- prometheus.MustNewConstMetric(exporter.Metrics["stats_active_connections"].Metric,
				prometheus.GaugeValue, float64(stats.ActiveConnections), labelValues...)
			ch <- prometheus.MustNewConstMetric(exporter.Metrics["stats_total_connections"].Metric,
				prometheus.GaugeValue, float64(stats.TotalConnections), labelValues...)
			ch <- prometheus.MustNewConstMetric(exporter.Metrics["stats_request_errors"].Metric,
				prometheus.GaugeValue, float64(stats.RequestErrors), labelValues...)
		}
	}
	return nil
}

func listenerLbsLabels(lbs []listeners.LoadBalancerID) string {
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

func ListAllListeners(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allListeners []listeners.Listener
	allPagesListeners, err := listeners.List(exporter.Client, listeners.ListOpts{}).AllPages()
	if err != nil {
		return err
	}
	allListeners, err = listeners.ExtractListeners(allPagesListeners)
	if err != nil {
		return err
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["total_listeners"].Metric,
		prometheus.GaugeValue, float64(len(allListeners)))

	// Listener stats metrics (only if enabled)
	if _, hasStatsMetrics := exporter.Metrics["listener_stats_bytes_in"]; hasStatsMetrics {
		for _, listener := range allListeners {
			stats, err := listeners.GetStats(exporter.Client, listener.ID).Extract()
			if err != nil {
				exporter.logger.Warn("failed to get listener stats", "id", listener.ID, "error", err)
				continue
			}

			labelValues := []string{listener.ID, listener.Name, listener.ProjectID,
				listener.OperatingStatus, listener.ProvisioningStatus, listener.Protocol,
				strconv.Itoa(listener.ProtocolPort), listenerLbsLabels(listener.Loadbalancers)}

			ch <- prometheus.MustNewConstMetric(exporter.Metrics["listener_stats_bytes_in"].Metric,
				prometheus.GaugeValue, float64(stats.BytesIn), labelValues...)
			ch <- prometheus.MustNewConstMetric(exporter.Metrics["listener_stats_bytes_out"].Metric,
				prometheus.GaugeValue, float64(stats.BytesOut), labelValues...)
			ch <- prometheus.MustNewConstMetric(exporter.Metrics["listener_stats_active_connections"].Metric,
				prometheus.GaugeValue, float64(stats.ActiveConnections), labelValues...)
			ch <- prometheus.MustNewConstMetric(exporter.Metrics["listener_stats_total_connections"].Metric,
				prometheus.GaugeValue, float64(stats.TotalConnections), labelValues...)
			ch <- prometheus.MustNewConstMetric(exporter.Metrics["listener_stats_request_errors"].Metric,
				prometheus.GaugeValue, float64(stats.RequestErrors), labelValues...)
		}
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
	// Loadbalancer status metrics
	for _, amphora := range allAmphorae {
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["amphora_status"].Metric,
			prometheus.GaugeValue, float64(mapAmphoraStatus(amphora.Status)), amphora.ID, amphora.LoadbalancerID, amphora.ComputeID, amphora.Status,
			amphora.Role, amphora.LBNetworkIP, amphora.HAIP, amphora.CertExpiration.Format(time.RFC3339))
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
	for _, pool := range allPools {
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["pool_status"].Metric,
			prometheus.GaugeValue, float64(mapPoolStatus(pool.ProvisioningStatus)), pool.ID, pool.ProvisioningStatus, pool.Name,
			lbsLabels(pool.Loadbalancers), pool.Protocol, pool.LBMethod, pool.OperatingStatus, pool.ProjectID)
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
