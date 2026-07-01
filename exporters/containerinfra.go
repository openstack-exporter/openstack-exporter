package exporters

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/gophercloud/gophercloud/v2/openstack/containerinfra/v1/clusters"
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	RegisterTypedExporter("container-infra", NewContainerInfraExporter)
}

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
	sched Schedule
	descs containerInfraDescs
}

type containerInfraDescs struct {
	TotalClusters  *prometheus.Desc `metric:"total_clusters"`
	ClusterMasters *prometheus.Desc `metric:"cluster_masters" labels:"uuid,name,stack_id,status,node_count,project_id"`
	ClusterNodes   *prometheus.Desc `metric:"cluster_nodes"   labels:"uuid,name,stack_id,status,master_count,project_id"`
	ClusterStatus  *prometheus.Desc `metric:"cluster_status"  labels:"uuid,name,stack_id,status,node_count,master_count,project_id"`
}

type containerInfraScrape struct {
	clusters []clusters.Cluster
}

var containerInfraGraph = Graph[*ContainerInfraExporter, containerInfraScrape]{
	Sources: []Source[*ContainerInfraExporter, containerInfraScrape]{
		{Name: "clusters", Fetch: (*ContainerInfraExporter).fetchClusters},
	},
	Emitters: []Emitter[*ContainerInfraExporter, containerInfraScrape]{
		{
			Name:    "clusters",
			Metrics: []string{"total_clusters", "cluster_masters", "cluster_nodes", "cluster_status"},
			Sources: []string{"clusters"},
			Emit:    (*ContainerInfraExporter).emitClusters,
		},
	},
}

func NewContainerInfraExporter(config *ExporterConfig, logger *slog.Logger) (*ContainerInfraExporter, error) {
	e := &ContainerInfraExporter{
		BaseOpenStackExporter: BaseOpenStackExporter{
			Name:           "container_infra",
			ExporterConfig: *config,
			logger:         logger,
		},
	}
	e.RegisterAndFillDescs(&e.descs)
	sched, err := containerInfraGraph.PruneSchedule(&e.BaseOpenStackExporter)
	if err != nil {
		return nil, err
	}
	e.sched = sched
	containerInfraGraph.LogDAG(&e.BaseOpenStackExporter, e.sched)
	return e, nil
}

func (e *ContainerInfraExporter) Collect(ch chan<- prometheus.Metric) {
	e.RunCollect(ch, e.sched, func(ch chan<- prometheus.Metric) int {
		s := new(containerInfraScrape)
		return runSchedule(e, &e.BaseOpenStackExporter, &containerInfraGraph, e.sched, s, ch)
	})
}

func (e *ContainerInfraExporter) fetchClusters(ctx context.Context, s *containerInfraScrape) error {
	allPages, err := clusters.List(e.ClientV2, clusters.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}
	s.clusters, err = clusters.ExtractClusters(allPages)
	return err
}

func (e *ContainerInfraExporter) emitClusters(ctx context.Context, s *containerInfraScrape, ch chan<- prometheus.Metric) error {
	emitGauge(ch, e.descs.TotalClusters, float64(len(s.clusters)))
	for _, c := range s.clusters {
		emitGauge(ch, e.descs.ClusterMasters, float64(c.MasterCount), c.UUID, c.Name, c.StackID, c.Status, strconv.Itoa(c.NodeCount), c.ProjectID)
		emitGauge(ch, e.descs.ClusterNodes, float64(c.NodeCount), c.UUID, c.Name, c.StackID, c.Status, strconv.Itoa(c.MasterCount), c.ProjectID)
		emitGauge(ch, e.descs.ClusterStatus, float64(mapClusterStatus(c.Status)), c.UUID, c.Name, c.StackID, c.Status,
			strconv.Itoa(c.NodeCount), strconv.Itoa(c.MasterCount), c.ProjectID)
	}
	return nil
}
