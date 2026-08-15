package exporters

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	gophercloudv2 "github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/baremetal/v1/nodes"
	"github.com/openstack-exporter/openstack-exporter/utils"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	ironicLatestSupportedMicroversion = "1.90"
	// The detailed node endpoint enforces a positive limit and may omit
	// nodes_links even where its server-side default truncates the result, so
	// AllPages has nothing to follow and silently returns a partial node list.
	// listAllNodes follows the marker itself instead.
	ironicNodePageSize = 1000
)

// IronicExporter : extends BaseOpenStackExporter
type IronicExporter struct {
	BaseOpenStackExporter
}

var defaultIronicMetrics = []Metric{
	{Name: "node", Labels: []string{"id", "name", "provision_state", "power_state", "maintenance", "maintenance_reason", "conductor_group", "traits", "instance_uuid", "lessee", "last_error", "serial_number", "console_enabled", "resource_class", "deploy_kernel", "deploy_ramdisk", "retired", "retired_reason"}, Fn: ListNodes},
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
			exporter.AddMetric(metric.Name, metric.Fn, metric.Labels, metric.DeprecatedVersion, nil)
		}
	}

	return &exporter, nil
}

// listAllNodes returns every node, paging on an explicit marker under a stable
// sort so that deployments with more nodes than the API returns in one response
// are collected completely.
func listAllNodes(ctx context.Context, client *gophercloudv2.ServiceClient) ([]nodes.Node, error) {
	var allNodes []nodes.Node
	var marker string

	for {
		allPagesNodes, err := nodes.ListDetail(client, nodes.ListOpts{
			Limit:   ironicNodePageSize,
			Marker:  marker,
			SortKey: "id",
			SortDir: "asc",
		}).AllPages(ctx)
		if err != nil {
			return nil, err
		}

		page, err := nodes.ExtractNodes(allPagesNodes)
		if err != nil {
			return nil, err
		}

		allNodes = append(allNodes, page...)
		if len(page) < ironicNodePageSize {
			return allNodes, nil
		}

		marker = page[len(page)-1].UUID
		if marker == "" {
			return nil, fmt.Errorf("ironic: node page of %d entries ends without a UUID", ironicNodePageSize)
		}
	}
}

// ListNodes : list nodes
func ListNodes(ctx context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	allNodes, err := listAllNodes(ctx, exporter.ClientV2)
	if err != nil {
		return err
	}

	for _, node := range allNodes {
		deployKernel := getDriverInfoString(node.DriverInfo, "deploy_kernel")
		deployRamdisk := getDriverInfoString(node.DriverInfo, "deploy_ramdisk")
		serialNumber := getNestedExtraString(node.Extra, "system_vendor", "serial_number")

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["node"].Metric,
			prometheus.GaugeValue, 1.0, node.UUID, node.Name, node.ProvisionState, node.PowerState,
			strconv.FormatBool(node.Maintenance), sanitizeMetricString(node.MaintenanceReason), node.ConductorGroup, strings.Join(node.Traits, " "),
			node.InstanceUUID, node.Lessee, sanitizeMetricString(node.LastError), serialNumber, strconv.FormatBool(node.ConsoleEnabled), node.ResourceClass,
			deployKernel, deployRamdisk, strconv.FormatBool(node.Retired), node.RetiredReason)

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

func getExtraString(extra map[string]any, key string) string {
	if extra == nil {
		return ""
	}

	value, ok := extra[key]
	if !ok {
		return ""
	}

	s, ok := value.(string)
	if !ok {
		return ""
	}

	return s
}

func getNestedExtraString(extra map[string]any, key string, nestedKey string) string {
	if extra == nil {
		return ""
	}

	nested, ok := extra[key].(map[string]any)
	if !ok {
		return ""
	}

	return getExtraString(nested, nestedKey)
}

func sanitizeMetricString(value string) string {
	return strings.TrimSpace(strings.ReplaceAll(value, "\n", " "))
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
