package openstack

import (
	"strconv"

	"github.com/go-kit/log"
	"github.com/gophercloud/gophercloud/openstack/containerinfra/v1/clusters"
	"github.com/prometheus/client_golang/prometheus"
)

var cluster_status = []string{
	"CREATE_COMPLETE",
	"CREATE_FAILED",
	"CREATE_IN_PROGRESS",
	"UPDATE_IN_PROGRESS",
	"UPDATE_FAILED",
	"UPDATE_COMPLETE",
	"DELETE_IN_PROGRESS",
	"DELETE_FAILED",
	"DELETE_COMPLETE",
	"RESUME_COMPLETE",
	"RESUME_FAILED",
	"RESTORE_COMPLETE",
	"ROLLBACK_IN_PROGRESS",
	"ROLLBACK_FAILED",
	"ROLLBACK_COMPLETE",
	"SNAPSHOT_COMPLETE",
	"CHECK_COMPLETE",
	"ADOPT_COMPLETE",
}

func mapClusterStatus(current string) int {
	for idx, status := range cluster_status {
		if current == status {
			return idx
		}
	}
	return -1
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

func NewContainerInfraExporter(config *ExporterConfig, logger log.Logger) (*ContainerInfraExporter, error) {
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

func ListAllClusters(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allClusters []clusters.Cluster
	allPagesClusters, err := clusters.List(exporter.Client, clusters.ListOpts{}).AllPages()
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
