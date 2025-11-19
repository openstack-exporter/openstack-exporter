package exporters

import (
	"context"
	"log/slog"

	"github.com/gophercloud/gophercloud/openstack/orchestration/v1/stacks"
	"github.com/gophercloud/gophercloud/pagination"
	"github.com/prometheus/client_golang/prometheus"
)

var knownStackStatuses = map[string]int{
	"INIT_IN_PROGRESS":     0,
	"INIT_FAILED":          1,
	"INIT_COMPLETE":        2,
	"CREATE_IN_PROGRESS":   3,
	"CREATE_FAILED":        4,
	"CREATE_COMPLETE":      5,
	"DELETE_IN_PROGRESS":   6,
	"DELETE_FAILED":        7,
	"DELETE_COMPLETE":      8,
	"UPDATE_IN_PROGRESS":   9,
	"UPDATE_FAILED":        10,
	"UPDATE_COMPLETE":      11,
	"ROLLBACK_IN_PROGRESS": 12,
	"ROLLBACK_FAILED":      13,
	"ROLLBACK_COMPLETE":    14,
	"SUSPEND_IN_PROGRESS":  15,
	"SUSPEND_FAILED":       16,
	"SUSPEND_COMPLETE":     17,
	"RESUME_IN_PROGRESS":   18,
	"RESUME_FAILED":        19,
	"RESUME_COMPLETE":      20,
	"ADOPT_IN_PROGRESS":    21,
	"ADOPT_FAILED":         22,
	"ADOPT_COMPLETE":       23,
	"SNAPSHOT_IN_PROGRESS": 24,
	"SNAPSHOT_FAILED":      25,
	"SNAPSHOT_COMPLETE":    26,
	"CHECK_IN_PROGRESS":    27,
	"CHECK_FAILED":         28,
	"CHECK_COMPLETE":       29,
}

func mapHeatStatus(current string) int {
	v, ok := knownStackStatuses[current]
	if !ok {
		return -1
	}
	return v
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

func NewHeatExporter(config *ExporterConfig, logger *slog.Logger) (*HeatExporter, error) {
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

func ListAllStacks(_ context.Context, exporter *BaseOpenStackExporter, ch chan<- prometheus.Metric) error {
	var allStacks []listedStack
	allPagesStacks, err := stacks.List(exporter.Client, stacks.ListOpts{}).AllPages()
	if err != nil {
		return err
	}
	allStacks, err = extractStacks(allPagesStacks)
	if err != nil {
		return err
	}

	var stackStatusCounter = make(map[string]int, len(knownStackStatuses))
	for k := range knownStackStatuses {
		stackStatusCounter[k] = 0
	}

	for _, stack := range allStacks {
		stackStatusCounter[stack.Status]++

		// Stack status metrics
		ch <- prometheus.MustNewConstMetric(exporter.Metrics["stack_status"].Metric,
			prometheus.GaugeValue, float64(mapHeatStatus(stack.Status)), stack.ID, stack.Name, stack.Project, stack.Status)
	}

	// Stack status counter metrics
	for status, count := range stackStatusCounter {
		ch <- prometheus.MustNewConstMetric(
			exporter.Metrics["stack_status_counter"].Metric, prometheus.GaugeValue, float64(count), status)
	}

	return nil
}
