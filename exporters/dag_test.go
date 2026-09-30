package exporters

import (
	"reflect"
	"strings"
	"testing"
)

func TestComputeScheduleBuildsMixedDependencyWaves(t *testing.T) {
	type scrapeState struct{}
	graph := Graph[struct{}, scrapeState]{
		Sources: []Source[struct{}, scrapeState]{
			{Name: "a"},
			{Name: "b"},
			{Name: "c", DependsOn: []string{"a"}},
		},
		Emitters: []Emitter[struct{}, scrapeState]{
			{Name: "emitA", Metrics: []string{"a_metric"}, Sources: []string{"a"}},
			{Name: "emitC", Metrics: []string{"c_metric"}, Sources: []string{"c"}},
		},
	}

	schedule, err := graph.ComputeSchedule()
	if err != nil {
		t.Fatalf("ComputeSchedule() error = %v", err)
	}

	want := [][]string{
		{"source:a", "source:b"},
		{"source:c", "emitter:emitA"},
		{"emitter:emitC"},
	}
	if got := scheduleWaveNames(&graph, schedule); !reflect.DeepEqual(got, want) {
		t.Fatalf("waves = %#v, want %#v", got, want)
	}

	wantDependencies := [][]int{nil, nil, {0}, {0}, {2}}
	if !reflect.DeepEqual(schedule.dependencies, wantDependencies) {
		t.Fatalf("dependencies = %#v, want %#v", schedule.dependencies, wantDependencies)
	}
	wantDependents := [][]int{{2, 3}, nil, {4}, nil, nil}
	if !reflect.DeepEqual(schedule.dependents, wantDependents) {
		t.Fatalf("dependents = %#v, want %#v", schedule.dependents, wantDependents)
	}
}

func TestComputeScheduleRejectsInvalidGraph(t *testing.T) {
	type scrapeState struct{}
	tests := []struct {
		name      string
		graph     Graph[struct{}, scrapeState]
		wantError string
	}{
		{
			name: "duplicate source",
			graph: Graph[struct{}, scrapeState]{
				Sources: []Source[struct{}, scrapeState]{{Name: "a"}, {Name: "a"}},
			},
			wantError: `duplicate source name: "a"`,
		},
		{
			name: "duplicate emitter",
			graph: Graph[struct{}, scrapeState]{
				Emitters: []Emitter[struct{}, scrapeState]{{Name: "emit"}, {Name: "emit"}},
			},
			wantError: `duplicate emitter name: "emit"`,
		},
		{
			name: "missing source dependency",
			graph: Graph[struct{}, scrapeState]{
				Sources: []Source[struct{}, scrapeState]{{Name: "a", DependsOn: []string{"missing"}}},
			},
			wantError: `source "a" depends on missing source "missing"`,
		},
		{
			name: "missing emitter source",
			graph: Graph[struct{}, scrapeState]{
				Emitters: []Emitter[struct{}, scrapeState]{{Name: "emit", Sources: []string{"missing"}}},
			},
			wantError: `emitter "emit" depends on missing source "missing"`,
		},
		{
			name: "cycle",
			graph: Graph[struct{}, scrapeState]{
				Sources: []Source[struct{}, scrapeState]{
					{Name: "a", DependsOn: []string{"b"}},
					{Name: "b", DependsOn: []string{"a"}},
				},
			},
			wantError: "cycle detected in DAG",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := test.graph.ComputeSchedule()
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("ComputeSchedule() error = %v, want error containing %q", err, test.wantError)
			}
		})
	}
}

func TestPruneScheduleRecomputesDependencyGraph(t *testing.T) {
	type scrapeState struct{}
	graph := Graph[struct{}, scrapeState]{
		Sources: []Source[struct{}, scrapeState]{
			{Name: "a"},
			{Name: "b"},
			{Name: "c", DependsOn: []string{"b"}},
		},
		Emitters: []Emitter[struct{}, scrapeState]{
			{Name: "emitA", Metrics: []string{"a_metric"}, Sources: []string{"a"}},
			{Name: "emitC", Metrics: []string{"c_metric"}, Sources: []string{"c"}},
		},
	}

	schedule, err := graph.PruneSchedule(map[string]bool{
		"a_metric": true,
		"c_metric": false,
	})
	if err != nil {
		t.Fatalf("PruneSchedule() error = %v", err)
	}

	want := [][]string{{"source:a"}, {"emitter:emitA"}}
	if got := scheduleWaveNames(&graph, schedule); !reflect.DeepEqual(got, want) {
		t.Fatalf("pruned waves = %#v, want %#v", got, want)
	}
}

func TestPruneScheduleKeepsSharedDependencies(t *testing.T) {
	type scrapeState struct{}
	graph := Graph[struct{}, scrapeState]{
		Sources: []Source[struct{}, scrapeState]{
			{Name: "shared"},
			{Name: "left", DependsOn: []string{"shared"}},
			{Name: "right", DependsOn: []string{"shared"}},
		},
		Emitters: []Emitter[struct{}, scrapeState]{
			{Name: "emitLeft", Metrics: []string{"left_metric"}, Sources: []string{"left"}},
			{Name: "emitRight", Metrics: []string{"right_metric"}, Sources: []string{"right"}},
		},
	}

	schedule, err := graph.PruneSchedule(map[string]bool{
		"left_metric":  false,
		"right_metric": true,
	})
	if err != nil {
		t.Fatalf("PruneSchedule() error = %v", err)
	}

	want := [][]string{{"source:shared"}, {"source:right"}, {"emitter:emitRight"}}
	if got := scheduleWaveNames(&graph, schedule); !reflect.DeepEqual(got, want) {
		t.Fatalf("pruned waves = %#v, want %#v", got, want)
	}
}

func TestPruneScheduleKeepsEmitterWhenAnyMetricIsEnabled(t *testing.T) {
	type scrapeState struct{}
	graph := Graph[struct{}, scrapeState]{
		Sources: []Source[struct{}, scrapeState]{{Name: "data"}},
		Emitters: []Emitter[struct{}, scrapeState]{
			{Name: "emit", Metrics: []string{"enabled", "disabled"}, Sources: []string{"data"}},
		},
	}

	schedule, err := graph.PruneSchedule(map[string]bool{"enabled": true, "disabled": false})
	if err != nil {
		t.Fatalf("PruneSchedule() error = %v", err)
	}

	want := [][]string{{"source:data"}, {"emitter:emit"}}
	if got := scheduleWaveNames(&graph, schedule); !reflect.DeepEqual(got, want) {
		t.Fatalf("pruned waves = %#v, want %#v", got, want)
	}
}

func TestPruneScheduleRejectsUnknownMetric(t *testing.T) {
	type scrapeState struct{}
	graph := Graph[struct{}, scrapeState]{
		Sources: []Source[struct{}, scrapeState]{{Name: "a"}},
		Emitters: []Emitter[struct{}, scrapeState]{
			{Name: "emitA", Metrics: []string{"typo"}, Sources: []string{"a"}},
		},
	}

	_, err := graph.PruneSchedule(map[string]bool{"a_metric": true})
	if err == nil || !strings.Contains(err.Error(), `emitter "emitA" declares unknown metric "typo"`) {
		t.Fatalf("PruneSchedule() error = %v, want unknown metric error", err)
	}
}

func TestPruneScheduleRejectsCycleInPrunedSources(t *testing.T) {
	type scrapeState struct{}
	graph := Graph[struct{}, scrapeState]{
		Sources: []Source[struct{}, scrapeState]{
			{Name: "a", DependsOn: []string{"b"}},
			{Name: "b", DependsOn: []string{"a"}},
		},
		Emitters: []Emitter[struct{}, scrapeState]{
			{Name: "emit", Metrics: []string{"disabled"}, Sources: []string{"a"}},
		},
	}

	_, err := graph.PruneSchedule(map[string]bool{"disabled": false})
	if err == nil || !strings.Contains(err.Error(), "cycle detected in DAG") {
		t.Fatalf("PruneSchedule() error = %v, want cycle error", err)
	}
}

func TestPruneScheduleDropsEverythingWhenAllMetricsAreDisabled(t *testing.T) {
	type scrapeState struct{}
	graph := Graph[struct{}, scrapeState]{
		Sources: []Source[struct{}, scrapeState]{{Name: "data"}},
		Emitters: []Emitter[struct{}, scrapeState]{
			{Name: "emit", Metrics: []string{"metric"}, Sources: []string{"data"}},
		},
	}

	schedule, err := graph.PruneSchedule(map[string]bool{"metric": false})
	if err != nil {
		t.Fatalf("PruneSchedule() error = %v", err)
	}
	if len(schedule.nodes) != 0 || len(schedule.waves) != 0 {
		t.Fatalf("PruneSchedule() = %#v, want empty schedule", schedule)
	}
}

func scheduleWaveNames[E, S any](graph *Graph[E, S], schedule Schedule) [][]string {
	waves := make([][]string, len(schedule.waves))
	for i, wave := range schedule.waves {
		waves[i] = make([]string, len(wave))
		for j, nodeIndex := range wave {
			node := schedule.nodes[nodeIndex]
			switch node.kind {
			case scheduleSource:
				waves[i][j] = "source:" + graph.Sources[node.index].Name
			case scheduleEmitter:
				waves[i][j] = "emitter:" + graph.Emitters[node.index].Name
			}
		}
	}
	return waves
}
