package exporters

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/amphorae"
	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/listeners"
	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/loadbalancers"
	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/pools"
	"github.com/prometheus/client_golang/prometheus"
)

// Octavia API v2 entities have two status codes present in the response body.
// The provisioning_status describes the lifecycle status of the entity while the operating_status provides the observed status of the entity.
// Here we put operating_status in metrics value and provisioning_status in metrics label
var knownLoadbalancerStatuses = map[string]int{
	"ONLINE":     0, // Entity is operating normally. All pool members are healthy
	"DRAINING":   1, // The member is not accepting new connections
	"OFFLINE":    2, // Entity is administratively disabled
	"ERROR":      3, // The entity has failed. The member is failing it's health monitoring checks. All of the pool members are in ERROR
	"NO_MONITOR": 4, // No health monitor is configured for this entity and it's status is unknown
}

// The status of the amphora. One of: BOOTING, ALLOCATED, READY, PENDING_CREATE, PENDING_DELETE, DELETED, ERROR.
var knownAmphoraStatuses = map[string]int{
	"BOOTING":        0,
	"ALLOCATED":      1,
	"READY":          2,
	"PENDING_CREATE": 3,
	"PENDING_DELETE": 4,
	"DELETED":        5,
	"ERROR":          6,
}

// Loadbalancer pool provisioning status. One of: ACTIVE, DELETED, ERROR, PENDING_CREATE, PENDING_UPDATE, PENDING_DELETE.
var knownPoolStatuses = map[string]int{
	"ACTIVE":         0,
	"DELETED":        1,
	"ERROR":          2,
	"PENDING_CREATE": 3,
	"PENDING_UPDATE": 4,
	"PENDING_DELETE": 5,
}

func mapLoadbalancerStatus(current string) int {
	return mapStatus(knownLoadbalancerStatuses, current)
}

func mapAmphoraStatus(current string) int {
	return mapStatus(knownAmphoraStatuses, current)
}

func mapPoolStatus(current string) int {
	return mapStatus(knownPoolStatuses, current)
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

func ListAllLoadbalancers(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allLoadbalancers []loadbalancers.LoadBalancer
	allPagesLoadbalancers, err := loadbalancers.List(exporter.ClientV2, loadbalancers.ListOpts{}).AllPages(ctx)
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
			stats, err := loadbalancers.GetStats(ctx, exporter.ClientV2, loadbalancer.ID).Extract()
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

func ListAllListeners(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allListeners []listeners.Listener
	allPagesListeners, err := listeners.List(exporter.ClientV2, listeners.ListOpts{}).AllPages(ctx)
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
			stats, err := listeners.GetStats(ctx, exporter.ClientV2, listener.ID).Extract()
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

func ListAllAmphorae(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allAmphorae []amphorae.Amphora
	allPagesAmphorae, err := amphorae.List(exporter.ClientV2, amphorae.ListOpts{}).AllPages(ctx)
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

func ListAllPools(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allPools []pools.Pool
	allPagesPools, err := pools.List(exporter.ClientV2, pools.ListOpts{}).AllPages(ctx)
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
		if i != 0 {
			label += ","
		}
		label += l.ID
	}

	return label
}
