package exporters

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/gophercloud/gophercloud/v2/openstack/baremetal/apiversions"
	"github.com/gophercloud/gophercloud/v2/openstack/baremetal/v1/nodes"
	"github.com/prometheus/client_golang/prometheus"
)

// IronicExporter : extends BaseOpenStackExporter
type IronicExporter struct {
	BaseOpenStackExporter
}

var additionalLabelNameConstraintRe = regexp.MustCompile(`^([^_0-9][^_][a-zA-Z]|(?:_)[a-zA-Z0-9]|[a-zA-Z])[a-zA-Z0-9_]*$`)

var defaultIronicMetrics = []Metric{
	{Name: "node", Labels: []string{"id", "name", "provision_state", "power_state", "maintenance", "console_enabled", "resource_class", "deploy_kernel", "deploy_ramdisk", "retired", "retired_reason"}, Fn: ListNodes},
}

var nodeJSONFieldIndex map[string]int

func init() {
	t := reflect.TypeOf(nodes.Node{})
	nodeJSONFieldIndex = make(map[string]int)
	for i := 0; i < t.NumField(); i++ {
		tag := strings.Split(t.Field(i).Tag.Get("json"), ",")[0]
		if tag != "" && tag != "-" {
			nodeJSONFieldIndex[tag] = i
		}
	}
}

// NewIronicExporter : returns a pointer to IronicExporter
func NewIronicExporter(config *ExporterConfig, logger *slog.Logger) (*IronicExporter, error) {
	exporter := IronicExporter{
		BaseOpenStackExporter{
			Name:           "ironic",
			ExporterConfig: *config,
			logger:         logger,
		},
	}
	computedNodeMetricLabels, err := computeNodeMetricLabels(exporter.IronicAdditionalLabels)
	if err != nil {
		return nil, err
	}

	for _, metric := range defaultIronicMetrics {
		if exporter.isDeprecatedMetric(&metric) {
			continue
		}
		if !exporter.isSlowMetric(&metric) {
			if metric.Name == "node" {
				exporter.AddMetric(metric.Name, metric.Fn, computedNodeMetricLabels, metric.DeprecatedVersion, nil)
			} else {
				exporter.AddMetric(metric.Name, metric.Fn, metric.Labels, metric.DeprecatedVersion, nil)
			}

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

	additionalLabels := make([]string, 0)
	if exporter.IronicAdditionalLabels != "" {
		additionalLabels = strings.Split(exporter.IronicAdditionalLabels, ",")
	}

	for _, node := range allNodes {
		additionalLabelsValues := make([]string, len(additionalLabels))
		for i, label := range additionalLabels {
			additionalLabelsValues[i] = resolveNodeField(node, label)
		}

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

		labelValues := []string{node.UUID, node.Name, node.ProvisionState, node.PowerState, strconv.FormatBool(node.Maintenance),
			strconv.FormatBool(node.ConsoleEnabled), node.ResourceClass, deployKernel, deployRamdisk, strconv.FormatBool(node.Retired), node.RetiredReason}
		labelValues = append(labelValues, additionalLabelsValues...)

		ch <- prometheus.MustNewConstMetric(exporter.Metrics["node"].Metric,
			prometheus.GaugeValue, 1.0, labelValues...)
	}

	return nil
}

func computeNodeMetricLabels(additionalLabelsString string) ([]string, error) {
	var nodeMetric Metric

	for _, metric := range defaultIronicMetrics {
		if metric.Name == "node" {
			nodeMetric = metric
		}
	}

	if nodeMetric.Name == "" {
		return nil, fmt.Errorf("node metric not found")
	}

	nodeMetricComputedLabels := make([]string, len(nodeMetric.Labels))
	copy(nodeMetricComputedLabels, nodeMetric.Labels)

	additionalLabels := make([]string, 0)
	if additionalLabelsString != "" {
		// strings.Replace(exporter.IronicAdditionalLabels, ".", "_", -1) is done to convert labels like extra.rack_id to extra_rack_id for prometheus compatibility
		labels := strings.Split(additionalLabelsString, ",")
		for _, label := range labels {
			label = strings.ReplaceAll(label, ".", "_")
			if !additionalLabelNameConstraintRe.MatchString(label) {
				return nil, fmt.Errorf("label %s is not valid prometheus label name", label)
			}
			additionalLabels = append(additionalLabels, label)
		}
	} else {
		return nodeMetric.Labels, nil
	}

	for _, label := range additionalLabels {
		if slices.Contains(nodeMetric.Labels, label) {
			return nil, fmt.Errorf("label %s is already present in node metric labels", label)
		}
		nodeMetricComputedLabels = append(nodeMetricComputedLabels, label)
	}

	return nodeMetricComputedLabels, nil
}

// resolveNodeField resolves a dot-path against a Node.
// "conductor"     → node.Conductor (struct field by JSON tag)
// "extra.rack_id" → node.Extra["rack_id"] (map field by JSON tag, then map key)
// for support of deeply nested labels we can switch to using gjson instead of reflect
func resolveNodeField(node nodes.Node, path string) string {
	parts := strings.SplitN(path, ".", 2)

	idx, ok := nodeJSONFieldIndex[parts[0]]
	if !ok {
		return ""
	}

	fieldVal := reflect.ValueOf(node).Field(idx)

	if len(parts) == 1 {
		return fmt.Sprintf("%v", fieldVal.Interface())
	}

	// nested map access
	if fieldVal.Kind() != reflect.Map {
		return ""
	}
	mapVal := fieldVal.MapIndex(reflect.ValueOf(parts[1]))
	if !mapVal.IsValid() {
		return ""
	}
	if mapVal.Kind() == reflect.Interface {
		mapVal = mapVal.Elem()
	}
	return fmt.Sprintf("%v", mapVal.Interface())
}
