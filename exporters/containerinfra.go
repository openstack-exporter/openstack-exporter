package exporters

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/gophercloud/gophercloud/v2/openstack/containerinfra/v1/clusters"
	"github.com/prometheus/client_golang/prometheus"
)

var knownClusterStatuses = map[string]int{
	"CREATE_COMPLETE":      0,
	"CREATE_FAILED":        1,
	"CREATE_IN_PROGRESS":   2,
	"UPDATE_IN_PROGRESS":   3,
	"UPDATE_FAILED":        4,
	"UPDATE_COMPLETE":      5,
	"DELETE_IN_PROGRESS":   6,
	"DELETE_FAILED":        7,
	"DELETE_COMPLETE":      8,
	"RESUME_COMPLETE":      9,
	"RESUME_FAILED":        10,
	"RESTORE_COMPLETE":     11,
	"ROLLBACK_IN_PROGRESS": 12,
	"ROLLBACK_FAILED":      13,
	"ROLLBACK_COMPLETE":    14,
	"SNAPSHOT_COMPLETE":    15,
	"CHECK_COMPLETE":       16,
	"ADOPT_COMPLETE":       17,
}

func mapClusterStatus(current string) int {
	return mapStatus(knownClusterStatuses, current)
}

type ContainerInfraExporter struct {
	BaseOpenStackExporter
}

var defaultContainerInfraMetrics = []Metric{
	{Name: "total_clusters", Fn: ListAllClusters},
	{Name: "cluster_masters", Labels: []string{"uuid", "name", "stack_id", "status", "node_count", "project_id"}, Fn: nil},
	{Name: "cluster_nodes", Labels: []string{"uuid", "name", "stack_id", "status", "master_count", "project_id"}, Fn: nil},
	{Name: "cluster_status", Labels: []string{"uuid", "name", "stack_id", "status", "node_count", "master_count", "project_id"}, Fn: nil},
}

func NewContainerInfraExporter(config *ExporterConfig, logger *slog.Logger) (*ContainerInfraExporter, error) {
	exporter := ContainerInfraExporter{
		BaseOpenStackExporter{
			Name:           "container_infra",
			ExporterConfig: *config,
			logger:         logger,
		},
	}

	for _, metric := range defaultContainerInfraMetrics {
		if exporter.isDeprecatedMetric(&metric) {
			continue
		}
		if !exporter.isSlowMetric(&metric) {
			exporter.AddMetric(metric.Name, metric.Fn, metric.Labels, metric.DeprecatedVersion, nil)
		}
	}

	return &exporter, nil
}

func ListAllClusters(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allClusters []clusters.Cluster
	allPagesClusters, err := clusters.List(exporter.ClientV2, clusters.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}

	allClusters, err = clusters.ExtractClusters(allPagesClusters)
	if err != nil {
		return err
	}

	ch <- prometheus.MustNewConstMetric(exporter.Metrics["total_clusters"].Metric,
		prometheus.GaugeValue, float64(len(allClusters)))

	// Cluster status metrics
	for _, cluster := range allClusters {
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["cluster_masters"].Metric,
			prometheus.GaugeValue, float64(cluster.MasterCount), cluster.UUID, cluster.Name,
			cluster.StackID, cluster.Status, strconv.Itoa(cluster.NodeCount), cluster.ProjectID)
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["cluster_nodes"].Metric,
			prometheus.GaugeValue, float64(cluster.NodeCount), cluster.UUID, cluster.Name,
			cluster.StackID, cluster.Status, strconv.Itoa(cluster.MasterCount), cluster.ProjectID)
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["cluster_status"].Metric,
			prometheus.GaugeValue, float64(mapClusterStatus(cluster.Status)), cluster.UUID, cluster.Name,
			cluster.StackID, cluster.Status, strconv.Itoa(cluster.NodeCount), strconv.Itoa(cluster.MasterCount), cluster.ProjectID)
	}

	return nil
}
