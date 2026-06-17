package exporters

import (
	"context"
	"log/slog"

	"github.com/gophercloud/gophercloud/v2/openstack/orchestration/v1/stacks"
	"github.com/gophercloud/gophercloud/v2/pagination"
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	RegisterTypedExporter("orchestration", NewHeatExporter)
}

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
	return mapStatus(knownStackStatuses, current)
}

type listedStack struct {
	ID      string
	Name    string `json:"stack_name"`
	Status  string `json:"stack_status"`
	Project string
}

func extractStacks(r pagination.Page) ([]listedStack, error) {
	var s struct {
		ListedStacks []listedStack `json:"stacks"`
	}
	err := (r.(stacks.StackPage)).ExtractInto(&s)
	return s.ListedStacks, err
}

type HeatExporter struct {
	BaseOpenStackExporter
	sched Schedule
	descs heatDescs
}

type heatDescs struct {
	StackStatus        *prometheus.Desc `metric:"stack_status"         labels:"id,name,project_id,status"`
	StackStatusCounter *prometheus.Desc `metric:"stack_status_counter" labels:"status"`
}

type heatScrape struct {
	stacks []listedStack
}

var heatGraph = Graph[*HeatExporter, heatScrape]{
	Sources: []Source[*HeatExporter, heatScrape]{
		{Name: "stacks", Fetch: (*HeatExporter).fetchStacks},
	},
	Emitters: []Emitter[*HeatExporter, heatScrape]{
		{
			Name:    "stacks",
			Metrics: []string{"stack_status", "stack_status_counter"},
			Sources: []string{"stacks"},
			Emit:    (*HeatExporter).emitStacks,
		},
	},
}

func NewHeatExporter(config *ExporterConfig, logger *slog.Logger) (*HeatExporter, error) {
	e := &HeatExporter{
		BaseOpenStackExporter: BaseOpenStackExporter{
			Name:           "heat",
			ExporterConfig: *config,
			logger:         logger,
		},
	}
	e.RegisterAndFillDescs(&e.descs)
	sched, err := heatGraph.PruneSchedule(&e.BaseOpenStackExporter)
	if err != nil {
		return nil, err
	}
	e.sched = sched
	heatGraph.LogDAG(&e.BaseOpenStackExporter, e.sched)
	return e, nil
}

func (e *HeatExporter) Collect(ch chan<- prometheus.Metric) {
	e.RunCollect(ch, e.sched, func(ch chan<- prometheus.Metric) int {
		s := new(heatScrape)
		return runSchedule(e, &e.BaseOpenStackExporter, &heatGraph, e.sched, s, ch)
	})
}

func (e *HeatExporter) fetchStacks(ctx context.Context, s *heatScrape) error {
	allPages, err := stacks.List(e.ClientV2, stacks.ListOpts{}).AllPages(ctx)
	if err != nil {
		return err
	}
	s.stacks, err = extractStacks(allPages)
	return err
}

func (e *HeatExporter) emitStacks(ctx context.Context, s *heatScrape, ch chan<- prometheus.Metric) error {
	counter := make(map[string]int, len(knownStackStatuses))
	for k := range knownStackStatuses {
		counter[k] = 0
	}
	for _, stack := range s.stacks {
		emitGauge(ch, e.descs.StackStatus, float64(mapHeatStatus(stack.Status)), stack.ID, stack.Name, stack.Project, stack.Status)
		counter[stack.Status]++
	}
	for status, count := range counter {
		emitGauge(ch, e.descs.StackStatusCounter, float64(count), status)
	}
	return nil
}
