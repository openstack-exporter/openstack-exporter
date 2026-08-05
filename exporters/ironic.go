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

var defaultIronicNodeLabels = []string{"id", "name", "provision_state", "power_state", "maintenance", "maintenance_reason", "console_enabled", "resource_class", "retired", "retired_reason"}
var ironicDriverInfoLabels = []string{"deploy_kernel", "deploy_ramdisk"}

// IronicExporter : extends BaseOpenStackExporter
type IronicExporter struct {
	BaseOpenStackExporter
}

var defaultIronicMetrics = []Metric{
	{Name: "node", Labels: defaultIronicNodeLabels, Fn: ListNodes},
	{Name: "node_updated_at", Labels: []string{"id", "name", "provision_state"}, Fn: nil},
	{Name: "node_provision_updated_at", Labels: []string{"id", "name", "provision_state"}, Fn: nil},
}

// NewIronicExporter : returns a pointer to IronicExporter
func NewIronicExporter(config *ExporterConfig, logger *slog.Logger) (*IronicExporter, error) {
	ctx := context.TODO()

	// NOTE(Sharpz7) Gophercloud V2 adds this new field ResourceBase.
	// For whatever reason, it adds a v1 field to the URL,
	// so it sends requests to /v1/v1 if left unfixed.
	//config.ClientV2.ResourceBase = config.ClientV2.Endpoint

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
			labels := metric.Labels
			if metric.Name == "node" && config.EnableIronicDriverInfo {
				labels = append(append([]string{}, defaultIronicNodeLabels...), ironicDriverInfoLabels...)
			}
			exporter.AddMetric(metric.Name, metric.Fn, labels, metric.DeprecatedVersion, nil)
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
		labelValues := []string{
			node.UUID,
			node.Name,
			node.ProvisionState,
			node.PowerState,
			strconv.FormatBool(node.Maintenance),
			node.MaintenanceReason,
			strconv.FormatBool(node.ConsoleEnabled),
			node.ResourceClass,
			strconv.FormatBool(node.Retired),
			node.RetiredReason,
		}
		if exporter.EnableIronicDriverInfo {
			labelValues = append(labelValues,
				getDriverInfoString(node.DriverInfo, "deploy_kernel"),
				getDriverInfoString(node.DriverInfo, "deploy_ramdisk"),
			)
		}

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["node"].Metric,
			prometheus.GaugeValue, 1.0, labelValues...)

		if !node.UpdatedAt.IsZero() {
			ch <- prometheus.MustNewConstMetric(
				exporter.Metrics["node_updated_at"].Metric,
				prometheus.GaugeValue,
				float64(node.UpdatedAt.Unix()),
				node.UUID,
				node.Name,
				node.ProvisionState,
			)
		}

		if !node.ProvisionUpdatedAt.IsZero() {
			ch <- prometheus.MustNewConstMetric(
				exporter.Metrics["node_provision_updated_at"].Metric,
				prometheus.GaugeValue,
				float64(node.ProvisionUpdatedAt.Unix()),
				node.UUID,
				node.Name,
				node.ProvisionState,
			)
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
