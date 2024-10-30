package exporters

import (
	"context"
	"strconv"

	"github.com/go-kit/log"
	"github.com/gophercloud/gophercloud/v2/openstack/baremetal/apiversions"
	"github.com/gophercloud/gophercloud/v2/openstack/baremetal/v1/nodes"
	"github.com/prometheus/client_golang/prometheus"
)

// IronicExporter : extends BaseOpenStackExporter
type IronicExporter struct {
	BaseOpenStackExporter
}

var defaultIronicMetrics = []Metric{
	{Name: "node", Labels: []string{"id", "name", "provision_state", "power_state", "maintenance", "console_enabled", "resource_class", "deploy_kernel", "deploy_ramdisk", "retired", "retired_reason"}, Fn: ListNodes},
}

// NewIronicExporter : returns a pointer to IronicExporter
func NewIronicExporter(config *ExporterConfig, logger log.Logger) (*IronicExporter, error) {
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

	// NOTE(Sharpz7) Gophercloud V2 adds this new field ResourceBase.
	// For whatever reason, it adds a v1 field to the URL,
	// so it sends requests to /v1/v1 if left unfixed.
	config.ClientV2.ResourceBase = config.ClientV2.Endpoint

	// Set Microversion workaround
	microversion, err := apiversions.Get(context.TODO(), config.ClientV2, "v1").Extract()
	if err == nil {
		config.ClientV2.Microversion = microversion.Version
		config.Client.Microversion = microversion.Version
	}

	return &exporter, nil
}

// ListNodes : list nodes
func ListNodes(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	allPagesNodes, err := nodes.ListDetail(exporter.ClientV2, nodes.ListOpts{}).AllPages(context.TODO())
	if err != nil {
		return err
	}

	allNodes, err := nodes.ExtractNodes(allPagesNodes)
	if err != nil {
		return err
	}

	for _, node := range allNodes {
		var deployKernel, deployRamdisk string

		if value, found := node.DriverInfo["deploy_kernel"]; found {
			if kernelValue, ok := value.(string); ok {
				deployKernel = kernelValue
			}
		}

		if value, found := node.DriverInfo["deploy_ramdisk"]; found {
			if ramdiskValue, ok := value.(string); ok {
				deployRamdisk = ramdiskValue
			}
		}

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["node"].Metric,
			prometheus.GaugeValue, 1.0, node.UUID, node.Name, node.ProvisionState, node.PowerState,
			strconv.FormatBool(node.Maintenance), strconv.FormatBool(node.ConsoleEnabled), node.ResourceClass,
			deployKernel, deployRamdisk, strconv.FormatBool(node.Retired), node.RetiredReason)
	}

	return nil
}
