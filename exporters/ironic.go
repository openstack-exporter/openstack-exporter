package exporters

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/gophercloud/gophercloud/v2/openstack/baremetal/v1/nodes"
	"github.com/openstack-exporter/openstack-exporter/utils"
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	RegisterTypedExporter("baremetal", NewIronicExporter)
}

const ironicLatestSupportedMicroversion = "1.90"

type IronicExporter struct {
	BaseOpenStackExporter
	sched Schedule
	descs ironicDescs
}

type ironicDescs struct {
	Node                   *prometheus.Desc `metric:"node"                   labels:"id,name,provision_state,power_state,maintenance,maintenance_reason,console_enabled,resource_class,deploy_kernel,deploy_ramdisk,retired,retired_reason"`
	NodeUpdatedAt          *prometheus.Desc `metric:"node_updated_at"         labels:"id,name,provision_state"`
	NodeProvisionUpdatedAt *prometheus.Desc `metric:"node_provision_updated_at" labels:"id,name,provision_state"`
}

type ironicScrape struct {
	nodes []nodes.Node
}

var ironicGraph = Graph[*IronicExporter, ironicScrape]{
	Sources: []Source[*IronicExporter, ironicScrape]{
		{Name: "nodes", Fetch: (*IronicExporter).fetchNodes},
	},
	Emitters: []Emitter[*IronicExporter, ironicScrape]{
		{
			Name:    "nodes",
			Metrics: []string{"node", "node_updated_at", "node_provision_updated_at"},
			Sources: []string{"nodes"},
			Emit:    (*IronicExporter).emitNodes,
		},
	},
}

func NewIronicExporter(config *ExporterConfig, logger *slog.Logger) (*IronicExporter, error) {
	ctx := context.TODO()
	if err := utils.SetupClientMicroversionV2(ctx, config.ClientV2, "OS_BAREMETAL_API_VERSION", ironicLatestSupportedMicroversion, logger); err != nil {
		return nil, err
	}
	e := &IronicExporter{
		BaseOpenStackExporter: BaseOpenStackExporter{
			Name:           "ironic",
			ExporterConfig: *config,
			logger:         logger,
		},
	}
	e.RegisterAndFillDescs(&e.descs)
	sched, err := ironicGraph.PruneSchedule(&e.BaseOpenStackExporter)
	if err != nil {
		return nil, err
	}
	e.sched = sched
	ironicGraph.LogDAG(&e.BaseOpenStackExporter, e.sched)
	return e, nil
}

func (e *IronicExporter) Collect(ch chan<- prometheus.Metric) {
	e.RunCollect(ch, e.sched, func(ch chan<- prometheus.Metric) int {
		s := new(ironicScrape)
		return runSchedule(e, &e.BaseOpenStackExporter, &ironicGraph, e.sched, s, ch)
	})
}

func (e *IronicExporter) fetchNodes(ctx context.Context, s *ironicScrape) error {
	allPages, err := nodes.ListDetail(e.ClientV2, nodes.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}
	s.nodes, err = nodes.ExtractNodes(allPages)
	return err
}

func (e *IronicExporter) emitNodes(ctx context.Context, s *ironicScrape, ch chan<- prometheus.Metric) error {
	for _, node := range s.nodes {
		deployKernel := getDriverInfoString(node.DriverInfo, "deploy_kernel")
		deployRamdisk := getDriverInfoString(node.DriverInfo, "deploy_ramdisk")
		emitGauge(ch, e.descs.Node, 1.0,
			node.UUID, node.Name, node.ProvisionState, node.PowerState,
			strconv.FormatBool(node.Maintenance), node.MaintenanceReason,
			strconv.FormatBool(node.ConsoleEnabled), node.ResourceClass,
			deployKernel, deployRamdisk,
			strconv.FormatBool(node.Retired), node.RetiredReason)
		if !node.UpdatedAt.IsZero() {
			emitGauge(ch, e.descs.NodeUpdatedAt,
				float64(node.UpdatedAt.Unix()), node.UUID, node.Name, node.ProvisionState)
		}
		if !node.ProvisionUpdatedAt.IsZero() {
			emitGauge(ch, e.descs.NodeProvisionUpdatedAt,
				float64(node.ProvisionUpdatedAt.Unix()), node.UUID, node.Name, node.ProvisionState)
		}
	}
	return nil
}

func getDriverInfoString(driverInfo map[string]any, key string) string {
	v, ok := driverInfo[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}
