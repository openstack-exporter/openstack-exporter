package exporters

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strconv"

	"github.com/gophercloud/gophercloud/v2/openstack/baremetal/v1/nodes"
	"github.com/openstack-exporter/openstack-exporter/utils"
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	RegisterTypedExporter("baremetal", NewIronicExporter)
}

const (
	ironicLatestSupportedMicroversion = "1.90"
	// The Ironic detailed-node endpoint enforces a positive limit and may omit
	// nodes_links even when its server-side default truncates at 1,000 nodes.
	// fetchNodes follows the marker explicitly instead.
	ironicNodePageSize = 1000
)

type IronicExporter struct {
	BaseOpenStackExporter
	sched Schedule
	descs ironicDescs
	// nodeDesc has dynamic labels (base + NodeExtraLabels mapping).
	nodeDesc *prometheus.Desc
}

type ironicDescs struct {
	// node: stored directly in IronicExporter.nodeDesc
	NodeUpdatedAt          *prometheus.Desc `metric:"node_updated_at"         labels:"id,name,provision_state"`
	NodeProvisionUpdatedAt *prometheus.Desc `metric:"node_provision_updated_at" labels:"id,name,provision_state"`
}

// ironicNodeBaseLabels are the fixed labels of the node metric, before any
// operator-configured labels are appended.
var ironicNodeBaseLabels = []string{"id", "name", "provision_state", "power_state",
	"maintenance", "maintenance_reason", "console_enabled", "resource_class",
	"retired", "retired_reason"}

// DefaultIronicNodeExtraLabels reproduces the deploy_kernel and deploy_ramdisk
// labels that openstack_ironic_node carried before the mapping existed, so the
// default output is unchanged and operators who do not want these labels can
// now drop them by passing an empty value.
const DefaultIronicNodeExtraLabels = "driver_info.deploy_kernel,driver_info.deploy_ramdisk"

// NewIronicNodeExtraLabels returns the default node label mapping, for callers
// that build ExporterOptions without going through the CLI flag.
func NewIronicNodeExtraLabels() *utils.LabelMappingFlag {
	m := &utils.LabelMappingFlag{DeriveLabelFromLeaf: true}
	if err := m.Set(DefaultIronicNodeExtraLabels); err != nil {
		panic(err) // constant input, cannot fail
	}
	return m
}

// ironicNodeInfoMaps are the node dictionaries whose keys may be mapped to
// labels. A mapping key is qualified with one of these names, for example
// `driver_info.deploy_kernel`.
var ironicNodeInfoMaps = []string{"driver_info", "instance_info", "extra", "properties", "driver_internal_info"}

// IronicNodeInfoMaps returns the node dictionaries whose keys can be mapped to
// labels, for use in the CLI flag help.
func IronicNodeInfoMaps() []string {
	return slices.Clone(ironicNodeInfoMaps)
}

// ironicNodeNestedMaps exposes the five allowlisted node dictionaries to the
// generic label mapping helper. It deliberately contains no lookup or
// redaction logic: LabelMappingFlag owns both.
func ironicNodeNestedMaps(node *nodes.Node) []utils.NestedMap {
	return []utils.NestedMap{
		{Name: "driver_info", Values: node.DriverInfo},
		{Name: "instance_info", Values: node.InstanceInfo},
		{Name: "extra", Values: node.Extra},
		{Name: "properties", Values: node.Properties},
		{Name: "driver_internal_info", Values: node.DriverInternalInfo},
	}
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
	if err := config.IronicNodeExtraLabels.ValidateNestedKeys(ironicNodeInfoMaps...); err != nil {
		return nil, err
	}
	if config.IronicNodeExtraLabels != nil {
		if err := config.IronicNodeExtraLabels.ValidateAgainst(ironicNodeBaseLabels); err != nil {
			return nil, fmt.Errorf("ironic.node-extra-labels: %w", err)
		}
	}

	// node carries operator-configured labels, so its descriptor cannot come
	// from a struct tag and is declared dynamically.
	sched, err := SetupExporter(&e.BaseOpenStackExporter, &e.descs, &ironicGraph,
		WithDynamicMetrics([]string{"node"}, func(base *BaseOpenStackExporter) {
			if !base.IsMetricEnabled("node") {
				return
			}
			labels := slices.Clone(ironicNodeBaseLabels)
			if config.IronicNodeExtraLabels != nil {
				labels = append(labels, config.IronicNodeExtraLabels.Labels...)
			}
			e.nodeDesc = prometheus.NewDesc(
				prometheus.BuildFQName(base.GetName(), "", "node"),
				"node", labels, nil)
			base.RegisterDesc(e.nodeDesc)
		}))
	if err != nil {
		return nil, err
	}
	e.sched = sched
	return e, nil
}

func (e *IronicExporter) Collect(ch chan<- prometheus.Metric) {
	e.RunCollect(ch, e.sched, func(ch chan<- prometheus.Metric) int {
		s := new(ironicScrape)
		return runSchedule(e, &e.BaseOpenStackExporter, &ironicGraph, e.sched, s, ch)
	})
}

func (e *IronicExporter) fetchNodes(ctx context.Context, s *ironicScrape) error {
	var marker string
	for {
		allPages, err := nodes.ListDetail(e.ClientV2, nodes.ListOpts{
			Limit:   ironicNodePageSize,
			Marker:  marker,
			SortKey: "id",
			SortDir: "asc",
		}).AllPages(ctx)
		if err != nil {
			return err
		}

		page, err := nodes.ExtractNodes(allPages)
		if err != nil {
			return err
		}
		s.nodes = append(s.nodes, page...)
		if len(page) < ironicNodePageSize {
			return nil
		}

		marker = page[len(page)-1].UUID
		if marker == "" {
			return fmt.Errorf("ironic: node page of %d entries ends without a UUID", ironicNodePageSize)
		}
	}
}

func (e *IronicExporter) emitNodes(ctx context.Context, s *ironicScrape, ch chan<- prometheus.Metric) error {
	for _, node := range s.nodes {
		labelValues := []string{
			node.UUID, node.Name, node.ProvisionState, node.PowerState,
			strconv.FormatBool(node.Maintenance), node.MaintenanceReason,
			strconv.FormatBool(node.ConsoleEnabled), node.ResourceClass,
			strconv.FormatBool(node.Retired), node.RetiredReason,
		}
		labelValues = append(labelValues, e.IronicNodeExtraLabels.ExtractNestedAny(ironicNodeNestedMaps(&node)...)...)
		emitGauge(ch, e.nodeDesc, 1.0, labelValues...)
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
