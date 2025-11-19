package exporters

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/gophercloud/gophercloud/v2/openstack/baremetal/v1/nodes"
	"github.com/openstack-exporter/openstack-exporter/utils"
	"github.com/prometheus/client_golang/prometheus"
)

const ironicLatestSupportedMicroversion = "1.90"

// IronicExporter : extends BaseOpenStackExporter
type IronicExporter struct {
	BaseOpenStackExporter
}

var defaultIronicMetrics = []Metric{
	{Name: "node", Labels: []string{"id", "name", "provision_state", "power_state", "maintenance", "console_enabled", "resource_class", "deploy_kernel", "deploy_ramdisk", "retired", "retired_reason"}, Fn: ListNodes},
}

// NewIronicExporter : returns a pointer to IronicExporter
func NewIronicExporter(config *ExporterConfig, logger *slog.Logger) (*IronicExporter, error) {
	ctx := context.TODO()

	// NOTE(Sharpz7) Gophercloud V2 adds this new field ResourceBase.
	// For whatever reason, it adds a v1 field to the URL,
	// so it sends requests to /v1/v1 if left unfixed.
	config.ClientV2.ResourceBase = config.ClientV2.Endpoint

	err := utils.SetupClientMicroversionV2(ctx, config.ClientV2, "OS_BAREMETAL_API_VERSION", ironicLatestSupportedMicroversion, logger)
	if err != nil {
		return nil, err
	}

	exporter := IronicExporter{
		BaseOpenStackExporter{
			Name:           "ironic",
			ExporterConfig: *config,
			logger:         logger,
		},
	}

	for _, metric := range defaultIronicMetrics {
		if exporter.isDeprecatedMetric(&metric) {
			continue
		}
		if !exporter.isSlowMetric(&metric) {
			exporter.AddMetric(metric.Name, metric.Fn, metric.Labels, metric.DeprecatedVersion, nil)
		}
	}

	return &exporter, nil
}

// ListNodes : list nodes
func ListNodes(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	allPagesNodes, err := nodes.ListDetail(exporter.ClientV2, nodes.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}

	allNodes, err := nodes.ExtractNodes(allPagesNodes)
	if err != nil {
		return err
	}

	for _, node := range allNodes {
		deployKernel := getDriverInfoString(node.DriverInfo, "deploy_kernel")
		deployRamdisk := getDriverInfoString(node.DriverInfo, "deploy_ramdisk")

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["node"].Metric,
			prometheus.GaugeValue, 1.0, node.UUID, node.Name, node.ProvisionState, node.PowerState,
			strconv.FormatBool(node.Maintenance), strconv.FormatBool(node.ConsoleEnabled), node.ResourceClass,
			deployKernel, deployRamdisk, strconv.FormatBool(node.Retired), node.RetiredReason)
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
