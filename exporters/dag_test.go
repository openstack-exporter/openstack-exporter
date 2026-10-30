package exporters

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type scrapeState struct {
	a, b, c int
}

func TestComputeScheduleBuildsMixedDependencyWaves(t *testing.T) {
	graph := Graph[scrapeState]{
		Sources: []Source[scrapeState]{
			{Name: "a"},
			{Name: "b"},
			{Name: "c", DependsOn: []string{"a"}},
		},
		Emitters: []Emitter[scrapeState]{
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
	if got := schedule.Waves(); !reflect.DeepEqual(got, want) {
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
	if got := schedule.TotalSources(); got != 3 {
		t.Fatalf("TotalSources() = %d, want 3", got)
	}
	if pruned := schedule.PrunedSources(); len(pruned) != 0 {
		t.Fatalf("PrunedSources() = %v, want none", pruned)
	}
}

func TestComputeScheduleRejectsInvalidGraph(t *testing.T) {
	tests := []struct {
		name      string
		graph     Graph[scrapeState]
		wantError string
	}{
		{
			name: "duplicate source",
			graph: Graph[scrapeState]{
				Sources: []Source[scrapeState]{{Name: "a"}, {Name: "a"}},
			},
			wantError: `duplicate source name: "a"`,
		},
		{
			name: "duplicate emitter",
			graph: Graph[scrapeState]{
				Emitters: []Emitter[scrapeState]{{Name: "emit"}, {Name: "emit"}},
			},
			wantError: `duplicate emitter name: "emit"`,
		},
		{
			name: "missing source dependency",
			graph: Graph[scrapeState]{
				Sources: []Source[scrapeState]{{Name: "a", DependsOn: []string{"missing"}}},
			},
			wantError: `source "a" depends on missing source "missing"`,
		},
		{
			name: "missing emitter source",
			graph: Graph[scrapeState]{
				Emitters: []Emitter[scrapeState]{{Name: "emit", Sources: []string{"missing"}}},
			},
			wantError: `emitter "emit" depends on missing source "missing"`,
		},
		{
			name: "repeated source dependency",
			graph: Graph[scrapeState]{
				Sources: []Source[scrapeState]{{Name: "a"}, {Name: "b", DependsOn: []string{"a", "a"}}},
			},
			wantError: `source "b" depends on source "a" twice`,
		},
		{
			name: "repeated emitter source",
			graph: Graph[scrapeState]{
				Sources:  []Source[scrapeState]{{Name: "a"}},
				Emitters: []Emitter[scrapeState]{{Name: "emit", Sources: []string{"a", "a"}}},
			},
			wantError: `emitter "emit" depends on source "a" twice`,
		},
		{
			name: "cycle",
			graph: Graph[scrapeState]{
				Sources: []Source[scrapeState]{
					{Name: "a", DependsOn: []string{"b"}},
					{Name: "b", DependsOn: []string{"a"}},
				},
			},
			wantError: "cycle detected in DAG",
		},
		{
			name: "self dependency",
			graph: Graph[scrapeState]{
				Sources: []Source[scrapeState]{{Name: "a", DependsOn: []string{"a"}}},
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
	graph := Graph[scrapeState]{
		Sources: []Source[scrapeState]{
			{Name: "a"},
			{Name: "b"},
			{Name: "c", DependsOn: []string{"b"}},
		},
		Emitters: []Emitter[scrapeState]{
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
	if got := schedule.Waves(); !reflect.DeepEqual(got, want) {
		t.Fatalf("pruned waves = %#v, want %#v", got, want)
	}
	if got, want := schedule.PrunedSources(), []string{"b", "c"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("PrunedSources() = %v, want %v", got, want)
	}
	if got, want := schedule.PrunedEmitters(), []string{"emitC"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("PrunedEmitters() = %v, want %v", got, want)
	}
}

func TestPruneScheduleKeepsSharedDependencies(t *testing.T) {
	graph := Graph[scrapeState]{
		Sources: []Source[scrapeState]{
			{Name: "shared"},
			{Name: "left", DependsOn: []string{"shared"}},
			{Name: "right", DependsOn: []string{"shared"}},
		},
		Emitters: []Emitter[scrapeState]{
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
	if got := schedule.Waves(); !reflect.DeepEqual(got, want) {
		t.Fatalf("pruned waves = %#v, want %#v", got, want)
	}
}

func TestPruneScheduleKeepsEmitterWhenAnyMetricIsEnabled(t *testing.T) {
	graph := Graph[scrapeState]{
		Sources: []Source[scrapeState]{{Name: "data"}},
		Emitters: []Emitter[scrapeState]{
			{Name: "emit", Metrics: []string{"enabled", "disabled"}, Sources: []string{"data"}},
		},
	}

	schedule, err := graph.PruneSchedule(map[string]bool{"enabled": true, "disabled": false})
	if err != nil {
		t.Fatalf("PruneSchedule() error = %v", err)
	}

	want := [][]string{{"source:data"}, {"emitter:emit"}}
	if got := schedule.Waves(); !reflect.DeepEqual(got, want) {
		t.Fatalf("pruned waves = %#v, want %#v", got, want)
	}
}

func TestPruneScheduleRejectsUnknownMetric(t *testing.T) {
	graph := Graph[scrapeState]{
		Sources: []Source[scrapeState]{{Name: "a"}},
		Emitters: []Emitter[scrapeState]{
			{Name: "emitA", Metrics: []string{"typo"}, Sources: []string{"a"}},
		},
	}

	_, err := graph.PruneSchedule(map[string]bool{"a_metric": true})
	if err == nil || !strings.Contains(err.Error(), `emitter "emitA" declares unknown metric "typo"`) {
		t.Fatalf("PruneSchedule() error = %v, want unknown metric error", err)
	}
}

func TestPruneScheduleRejectsCycleInPrunedSources(t *testing.T) {
	graph := Graph[scrapeState]{
		Sources: []Source[scrapeState]{
			{Name: "a", DependsOn: []string{"b"}},
			{Name: "b", DependsOn: []string{"a"}},
		},
		Emitters: []Emitter[scrapeState]{
			{Name: "emit", Metrics: []string{"disabled"}, Sources: []string{"a"}},
		},
	}

	_, err := graph.PruneSchedule(map[string]bool{"disabled": false})
	if err == nil || !strings.Contains(err.Error(), "cycle detected in DAG") {
		t.Fatalf("PruneSchedule() error = %v, want cycle error", err)
	}
}

func TestPruneScheduleDropsEverythingWhenAllMetricsAreDisabled(t *testing.T) {
	graph := Graph[scrapeState]{
		Sources: []Source[scrapeState]{{Name: "data"}},
		Emitters: []Emitter[scrapeState]{
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

	result := schedule.Run(context.Background(), &scrapeState{}, nil, RunHooks{})
	if result != (RunResult{}) {
		t.Fatalf("Run() on empty schedule = %+v, want zero result", result)
	}
	if result.Healthy() {
		t.Fatal("Run() on empty schedule reported healthy")
	}
}

// drain returns a metric channel whose consumer discards everything.
func drain(t *testing.T) chan<- prometheus.Metric {
	t.Helper()
	ch := make(chan prometheus.Metric)
	go func() {
		for range ch {
		}
	}()
	t.Cleanup(func() { close(ch) })
	return ch
}

func TestRunStartsReadyNodesWithoutWaveBarrier(t *testing.T) {
	fastDone := make(chan struct{})
	allowSlow := make(chan struct{})
	emitted := make(chan struct{})

	graph := Graph[scrapeState]{
		Sources: []Source[scrapeState]{
			{
				Name: "fast",
				Fetch: func(context.Context, *scrapeState) error {
					close(fastDone)
					return nil
				},
			},
			{
				Name: "slow",
				Fetch: func(context.Context, *scrapeState) error {
					<-allowSlow
					return nil
				},
			},
		},
		Emitters: []Emitter[scrapeState]{
			{
				Name:    "fastEmitter",
				Metrics: []string{"fast_metric"},
				Sources: []string{"fast"},
				Emit: func(context.Context, *scrapeState, chan<- prometheus.Metric) error {
					close(emitted)
					return nil
				},
			},
		},
	}
	schedule, err := graph.ComputeSchedule()
	if err != nil {
		t.Fatalf("ComputeSchedule() error = %v", err)
	}

	done := make(chan RunResult, 1)
	go func() {
		done <- schedule.Run(context.Background(), &scrapeState{}, drain(t), RunHooks{})
	}()

	<-fastDone
	select {
	case <-emitted:
	case <-time.After(200 * time.Millisecond):
		close(allowSlow)
		t.Fatal("emitter did not start while unrelated source was still running")
	}

	close(allowSlow)
	want := RunResult{SourcesOK: 2, EmittersOK: 1}
	if got := <-done; got != want {
		t.Fatalf("Run() = %+v, want %+v", got, want)
	}
}

func TestRunSkipsEverythingDownstreamOfFailedSource(t *testing.T) {
	var mu sync.Mutex
	ran := map[string]bool{}
	mark := func(name string) {
		mu.Lock()
		defer mu.Unlock()
		ran[name] = true
	}
	fetchErr := errors.New("boom")

	graph := Graph[scrapeState]{
		Sources: []Source[scrapeState]{
			{Name: "broken", Fetch: func(context.Context, *scrapeState) error { mark("broken"); return fetchErr }},
			{Name: "fine", Fetch: func(context.Context, *scrapeState) error { mark("fine"); return nil }},
			{Name: "child", DependsOn: []string{"broken"}, Fetch: func(context.Context, *scrapeState) error { mark("child"); return nil }},
			{Name: "grandchild", DependsOn: []string{"child"}, Fetch: func(context.Context, *scrapeState) error { mark("grandchild"); return nil }},
		},
		Emitters: []Emitter[scrapeState]{
			{Name: "emitBroken", Metrics: []string{"m1"}, Sources: []string{"broken"},
				Emit: func(context.Context, *scrapeState, chan<- prometheus.Metric) error { mark("emitBroken"); return nil }},
			{Name: "emitFine", Metrics: []string{"m2"}, Sources: []string{"fine"},
				Emit: func(context.Context, *scrapeState, chan<- prometheus.Metric) error { mark("emitFine"); return nil }},
			{Name: "emitBoth", Metrics: []string{"m3"}, Sources: []string{"fine", "grandchild"},
				Emit: func(context.Context, *scrapeState, chan<- prometheus.Metric) error { mark("emitBoth"); return nil }},
		},
	}
	schedule, err := graph.ComputeSchedule()
	if err != nil {
		t.Fatalf("ComputeSchedule() error = %v", err)
	}

	var hookMu sync.Mutex
	hookErrs := map[string]error{}
	hooks := RunHooks{
		SourceDone: func(name string, _ time.Duration, err error) {
			hookMu.Lock()
			defer hookMu.Unlock()
			hookErrs[name] = err
		},
	}

	result := schedule.Run(context.Background(), &scrapeState{}, drain(t), hooks)

	want := RunResult{SourcesOK: 1, SourcesFailed: 1, SourcesSkipped: 2, EmittersOK: 1, EmittersSkipped: 2}
	if result != want {
		t.Fatalf("Run() = %+v, want %+v", result, want)
	}
	if !result.Healthy() {
		t.Fatal("Run() reported unhealthy although one source succeeded")
	}
	wantRan := map[string]bool{"broken": true, "fine": true, "emitFine": true}
	if !reflect.DeepEqual(ran, wantRan) {
		t.Fatalf("ran = %v, want %v", ran, wantRan)
	}
	if !errors.Is(hookErrs["broken"], fetchErr) {
		t.Fatalf("SourceDone(broken) err = %v, want %v", hookErrs["broken"], fetchErr)
	}
	if _, called := hookErrs["child"]; called {
		t.Fatal("SourceDone was called for a skipped source")
	}
}

func TestRunRecoversPanicsAsFailures(t *testing.T) {
	graph := Graph[scrapeState]{
		Sources: []Source[scrapeState]{
			{Name: "panicky", Fetch: func(context.Context, *scrapeState) error {
				panic("boom")
			}},
		},
		Emitters: []Emitter[scrapeState]{
			{Name: "emit", Metrics: []string{"m"}, Sources: []string{"panicky"},
				Emit: func(context.Context, *scrapeState, chan<- prometheus.Metric) error {
					t.Error("emitter ran after its source panicked")
					return nil
				}},
		},
	}
	schedule, err := graph.ComputeSchedule()
	if err != nil {
		t.Fatalf("ComputeSchedule() error = %v", err)
	}

	var gotErr error
	hooks := RunHooks{SourceDone: func(_ string, _ time.Duration, err error) { gotErr = err }}
	result := schedule.Run(context.Background(), &scrapeState{}, drain(t), hooks)

	want := RunResult{SourcesFailed: 1, EmittersSkipped: 1}
	if result != want {
		t.Fatalf("Run() = %+v, want %+v", result, want)
	}
	if gotErr == nil || !strings.Contains(gotErr.Error(), "panic: boom") {
		t.Fatalf("SourceDone err = %v, want recovered panic", gotErr)
	}
	if !strings.Contains(gotErr.Error(), "dag_test.go") {
		t.Fatalf("SourceDone err lacks a stack trace: %v", gotErr)
	}
}

func TestRunCountsEmitterErrorsWithoutBlocking(t *testing.T) {
	emitErr := errors.New("emit failed")
	graph := Graph[scrapeState]{
		Sources: []Source[scrapeState]{{Name: "a"}},
		Emitters: []Emitter[scrapeState]{
			{Name: "bad", Metrics: []string{"m1"}, Sources: []string{"a"},
				Emit: func(context.Context, *scrapeState, chan<- prometheus.Metric) error { return emitErr }},
			{Name: "good", Metrics: []string{"m2"}, Sources: []string{"a"},
				Emit: func(context.Context, *scrapeState, chan<- prometheus.Metric) error { return nil }},
		},
	}
	schedule, err := graph.ComputeSchedule()
	if err != nil {
		t.Fatalf("ComputeSchedule() error = %v", err)
	}

	var mu sync.Mutex
	seen := map[string]error{}
	hooks := RunHooks{EmitterDone: func(name string, _ time.Duration, err error) {
		mu.Lock()
		defer mu.Unlock()
		seen[name] = err
	}}
	result := schedule.Run(context.Background(), &scrapeState{}, drain(t), hooks)

	want := RunResult{SourcesOK: 1, EmittersOK: 1, EmittersFailed: 1}
	if result != want {
		t.Fatalf("Run() = %+v, want %+v", result, want)
	}
	if !errors.Is(seen["bad"], emitErr) || seen["good"] != nil {
		t.Fatalf("EmitterDone errors = %v", seen)
	}
}

func TestRunSkipsNodesAfterContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	graph := Graph[scrapeState]{
		Sources: []Source[scrapeState]{
			{Name: "first", Fetch: func(context.Context, *scrapeState) error {
				cancel()
				return nil
			}},
			{Name: "second", DependsOn: []string{"first"}, Fetch: func(context.Context, *scrapeState) error {
				t.Error("second source ran after cancellation")
				return nil
			}},
		},
		Emitters: []Emitter[scrapeState]{
			{Name: "emit", Metrics: []string{"m"}, Sources: []string{"second"}},
		},
	}
	schedule, err := graph.ComputeSchedule()
	if err != nil {
		t.Fatalf("ComputeSchedule() error = %v", err)
	}

	result := schedule.Run(ctx, &scrapeState{}, drain(t), RunHooks{})
	want := RunResult{SourcesOK: 1, SourcesSkipped: 1, EmittersSkipped: 1}
	if result != want {
		t.Fatalf("Run() = %+v, want %+v", result, want)
	}
}

func TestRunLetsIndependentSourcesWriteDisjointState(t *testing.T) {
	// Exercised under -race: concurrent sources touch different fields and
	// the emitter observes both after they complete.
	graph := Graph[scrapeState]{
		Sources: []Source[scrapeState]{
			{Name: "a", Fetch: func(_ context.Context, s *scrapeState) error { s.a = 1; return nil }},
			{Name: "b", Fetch: func(_ context.Context, s *scrapeState) error { s.b = 2; return nil }},
			{Name: "c", DependsOn: []string{"a", "b"}, Fetch: func(_ context.Context, s *scrapeState) error { s.c = s.a + s.b; return nil }},
		},
		Emitters: []Emitter[scrapeState]{
			{Name: "emit", Metrics: []string{"m"}, Sources: []string{"c"},
				Emit: func(_ context.Context, s *scrapeState, _ chan<- prometheus.Metric) error {
					if s.c != 3 {
						t.Errorf("emitter saw c = %d, want 3", s.c)
					}
					return nil
				}},
		},
	}
	schedule, err := graph.ComputeSchedule()
	if err != nil {
		t.Fatalf("ComputeSchedule() error = %v", err)
	}

	for i := 0; i < 50; i++ {
		result := schedule.Run(context.Background(), &scrapeState{}, drain(t), RunHooks{})
		if want := (RunResult{SourcesOK: 3, EmittersOK: 1}); result != want {
			t.Fatalf("Run() = %+v, want %+v", result, want)
		}
	}
}
