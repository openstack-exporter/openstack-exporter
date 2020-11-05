package exporters

import (
	"strconv"

	"github.com/gophercloud/gophercloud/openstack/baremetal/apiversions"
	"github.com/gophercloud/gophercloud/openstack/baremetal/v1/nodes"
	"github.com/prometheus/client_golang/prometheus"
)

// IronicExporter : extends BaseOpenStackExporter
type IronicExporter struct {
	BaseOpenStackExporter
}

var defaultIronicMetrics = []Metric{
	{Name: "node", Labels: []string{"id", "name", "provision_state", "power_state", "maintenance", "console_enabled", "resource_class"}, Fn: ListNodes},
}

// NewIronicExporter : returns a pointer to IronicExporter
func NewIronicExporter(config *ExporterConfig) (*IronicExporter, error) {
	exporter := IronicExporter{
		BaseOpenStackExporter{
			Name:           "ironic",
			ExporterConfig: *config,
		},
	}

	for _, metric := range defaultIronicMetrics {
		exporter.AddMetric(metric.Name, metric.Fn, metric.Labels, nil)
	}

	// Set Microversion workaround
	microversion, err := apiversions.Get(config.Client, "v1").Extract()
	if err == nil {
		exporter.Client.Microversion = microversion.Version
	}

	return &exporter, nil
}

// ListNodes : list nodes
func ListNodes(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	allPagesNodes, err := nodes.ListDetail(exporter.Client, nodes.ListOpts{}).AllPages()
	if err != nil {
		return err
	}

	allNodes, err := nodes.ExtractNodes(allPagesNodes)
	if err != nil {
		return err
	}

	for _, node := range allNodes {
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["node"].Metric,
			prometheus.GaugeValue, 1.0, node.UUID, node.Name, node.ProvisionState, node.PowerState,
			strconv.FormatBool(node.Maintenance), strconv.FormatBool(node.ConsoleEnabled), node.ResourceClass)
	}

	return nil
}
