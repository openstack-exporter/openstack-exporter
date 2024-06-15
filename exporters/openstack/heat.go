package openstack

import (
	"github.com/go-kit/log"
	"github.com/gophercloud/gophercloud/openstack/orchestration/v1/stacks"
	"github.com/gophercloud/gophercloud/pagination"
	"github.com/prometheus/client_golang/prometheus"
)

var stack_status = []string{
	"INIT_IN_PROGRESS",
	"INIT_FAILED",
	"INIT_COMPLETE",
	"CREATE_IN_PROGRESS",
	"CREATE_FAILED",
	"CREATE_COMPLETE",
	"DELETE_IN_PROGRESS",
	"DELETE_FAILED",
	"DELETE_COMPLETE",
	"UPDATE_IN_PROGRESS",
	"UPDATE_FAILED",
	"UPDATE_COMPLETE",
	"ROLLBACK_IN_PROGRESS",
	"ROLLBACK_FAILED",
	"ROLLBACK_COMPLETE",
	"SUSPEND_IN_PROGRESS",
	"SUSPEND_FAILED",
	"SUSPEND_COMPLETE",
	"RESUME_IN_PROGRESS",
	"RESUME_FAILED",
	"RESUME_COMPLETE",
	"ADOPT_IN_PROGRESS",
	"ADOPT_FAILED",
	"ADOPT_COMPLETE",
	"SNAPSHOT_IN_PROGRESS",
	"SNAPSHOT_FAILED",
	"SNAPSHOT_COMPLETE",
	"CHECK_IN_PROGRESS",
	"CHECK_FAILED",
	"CHECK_COMPLETE",
}

func mapHeatStatus(current string) int {
	for idx, status := range stack_status {
		if current == status {
			return idx
		}
	}
	return -1
}

type listedStack struct {
	ID      string
	Name    string `json:"stack_name"`
	Status  string `json:"stack_status"`
	Project string
}

// extractStacks extracts and returns a slice of listedStack. It is used while iterating
// over a stacks.List call.
func extractStacks(r pagination.Page) ([]listedStack, error) {
	var s struct {
		ListedStacks []listedStack `json:"stacks"`
	}
	err := (r.(stacks.StackPage)).ExtractInto(&s)
	return s.ListedStacks, err
}

type HeatExporter struct {
	BaseOpenStackExporter
}

var defaultHeatMetrics = []Metric{
	{Name: "stack_status", Labels: []string{"id", "name", "project_id", "status"}, Fn: ListAllStacks},
	{Name: "stack_status_counter", Labels: []string{"status"}, Fn: nil},
}

func NewHeatExporter(config *ExporterConfig, logger log.Logger) (*HeatExporter, error) {
	exporter := HeatExporter{
		BaseOpenStackExporter{
			Name:           "heat",
			ExporterConfig: *config,
			logger:         logger,
		},
	}

	for _, metric := range defaultHeatMetrics {
		if !exporter.isSlowMetric(&metric) {
			exporter.AddMetric(metric.Name, metric.Fn, metric.Labels, metric.DeprecatedVersion, nil)
		}
	}

	return &exporter, nil
}

func ListAllStacks(exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allStacks []listedStack
	allPagesStacks, err := stacks.List(exporter.Client, stacks.ListOpts{}).AllPages()
	if err != nil {
		return err
	}
	allStacks, err = extractStacks(allPagesStacks)
	if err != nil {
		return err
	}

	var stack_status_counter = make(map[string]int, len(server_status))
	for _, s := range stack_status {
		stack_status_counter[s] = 0
	}

	for _, stack := range allStacks {
		stack_status_counter[stack.Status]++
		// Stack status metrics
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["stack_status"].Metric,
			prometheus.GaugeValue, float64(mapHeatStatus(stack.Status)), stack.ID, stack.Name, stack.Project, stack.Status)
	}

	// Stack status counter metrics
	for status, count := range stack_status_counter {
		ch <- prometheus.MustNewConstMetric(
			exporter.Metrics["stack_status_counter"].Metric, prometheus.GaugeValue, float64(count), status)
	}

	return nil
}
