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
				Emitters: []Emitter[scrapeState]{
					{Name: "emit", Metrics: []string{"m"}},
					{Name: "emit", Metrics: []string{"m"}},
				},
			},
			wantError: `duplicate emitter name: "emit"`,
		},
		{
			name: "emitter without metrics",
			graph: Graph[scrapeState]{
				Sources:  []Source[scrapeState]{{Name: "a"}},
				Emitters: []Emitter[scrapeState]{{Name: "emit", Sources: []string{"a"}}},
			},
			wantError: `emitter "emit" declares no metrics`,
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
				Emitters: []Emitter[scrapeState]{{Name: "emit", Metrics: []string{"m"}, Sources: []string{"missing"}}},
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
				Emitters: []Emitter[scrapeState]{{Name: "emit", Metrics: []string{"m"}, Sources: []string{"a", "a"}}},
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
	}, nil)
	if err != nil {
		t.Fatalf("PruneSchedule(, nil) error = %v", err)
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
	}, nil)
	if err != nil {
		t.Fatalf("PruneSchedule(, nil) error = %v", err)
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

	schedule, err := graph.PruneSchedule(map[string]bool{"enabled": true, "disabled": false}, nil)
	if err != nil {
		t.Fatalf("PruneSchedule(, nil) error = %v", err)
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

	_, err := graph.PruneSchedule(map[string]bool{"a_metric": true}, nil)
	if err == nil || !strings.Contains(err.Error(), `emitter "emitA" declares unknown metric "typo"`) {
		t.Fatalf("PruneSchedule(, nil) error = %v, want unknown metric error", err)
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

	_, err := graph.PruneSchedule(map[string]bool{"disabled": false}, nil)
	if err == nil || !strings.Contains(err.Error(), "cycle detected in DAG") {
		t.Fatalf("PruneSchedule(, nil) error = %v, want cycle error", err)
	}
}

func TestPruneScheduleDropsEverythingWhenAllMetricsAreDisabled(t *testing.T) {
	graph := Graph[scrapeState]{
		Sources: []Source[scrapeState]{{Name: "data"}},
		Emitters: []Emitter[scrapeState]{
			{Name: "emit", Metrics: []string{"metric"}, Sources: []string{"data"}},
		},
	}

	schedule, err := graph.PruneSchedule(map[string]bool{"metric": false}, nil)
	if err != nil {
		t.Fatalf("PruneSchedule(, nil) error = %v", err)
	}
	if len(schedule.nodes) != 0 || len(schedule.waves) != 0 {
		t.Fatalf("PruneSchedule(, nil) = %#v, want empty schedule", schedule)
	}

	result := schedule.Run(context.Background(), &scrapeState{}, nil, RunHooks{})
	if result != (RunResult{}) {
		t.Fatalf("Run() on empty schedule = %+v, want zero result", result)
	}
	if !result.Healthy() {
		t.Fatal("Run() on empty schedule reported unhealthy, want healthy: nothing failed")
	}
}

func TestRunResultHealthy(t *testing.T) {
	tests := []struct {
		name   string
		result RunResult
		want   bool
	}{
		{name: "no sources ran", result: RunResult{}, want: true},
		{name: "no sources, emitter ran", result: RunResult{EmittersOK: 1}, want: true},
		{name: "one source ok", result: RunResult{SourcesOK: 1}, want: true},
		{name: "one ok one failed", result: RunResult{SourcesOK: 1, SourcesFailed: 1}, want: true},
		{name: "only failures", result: RunResult{SourcesFailed: 1}, want: false},
		{name: "only skipped", result: RunResult{SourcesSkipped: 1}, want: false},
		{name: "failed and skipped", result: RunResult{SourcesFailed: 1, SourcesSkipped: 2, EmittersSkipped: 1}, want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.result.Healthy(); got != test.want {
				t.Fatalf("%+v.Healthy() = %v, want %v", test.result, got, test.want)
			}
		})
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

// ---------------------------------------------------------------------------
// Requirements and XOR groups
// ---------------------------------------------------------------------------

const testAxis Feature = "test.microversion"

func capsAt(major, minor int) *Capabilities {
	var caps Capabilities
	caps.SetVersion(testAxis, Version{major, minor})
	return &caps
}

// xorGraph has a three-way source group covering disjoint version ranges
// and a dependent emitter.
func xorGraph() Graph[scrapeState] {
	return Graph[scrapeState]{
		Sources: []Source[scrapeState]{
			{Name: "a_new", XorGroup: "a", Requires: []Requirement{AtLeast{testAxis, Version{2, 47}}}},
			{Name: "a_mid", XorGroup: "a", Requires: []Requirement{AtLeast{testAxis, Version{2, 33}}, Below{testAxis, Version{2, 47}}}},
			{Name: "a_old", XorGroup: "a"},
			{Name: "b", DependsOn: []string{"a"}},
		},
		Emitters: []Emitter[scrapeState]{
			{Name: "emit", Metrics: []string{"m"}, Sources: []string{"a", "b"}},
		},
	}
}

func TestComputeScheduleRejectsInvalidXorGroups(t *testing.T) {
	tests := []struct {
		name      string
		graph     Graph[scrapeState]
		wantError string
	}{
		{
			name: "group named like a source",
			graph: Graph[scrapeState]{
				Sources: []Source[scrapeState]{
					{Name: "a"},
					{Name: "a_v2", XorGroup: "a"},
				},
			},
			wantError: `xor group "a" has the same name as a source`,
		},
		{
			name: "emitter group named like a source",
			graph: Graph[scrapeState]{
				Sources: []Source[scrapeState]{{Name: "a"}},
				Emitters: []Emitter[scrapeState]{
					{Name: "e1", XorGroup: "a", Metrics: []string{"m"}},
				},
			},
			wantError: `xor group "a" has the same name as a source`,
		},
		{
			name: "group mixing sources and emitters",
			graph: Graph[scrapeState]{
				Sources:  []Source[scrapeState]{{Name: "s", XorGroup: "g"}},
				Emitters: []Emitter[scrapeState]{{Name: "e", XorGroup: "g", Metrics: []string{"m"}}},
			},
			wantError: `xor group "g" mixes sources and emitters`,
		},
		{
			name: "two default sources",
			graph: Graph[scrapeState]{
				Sources: []Source[scrapeState]{
					{Name: "s1", XorGroup: "g"},
					{Name: "s2", XorGroup: "g"},
				},
			},
			wantError: `xor group "g" has 2 members without requirements`,
		},
		{
			name: "two default emitters",
			graph: Graph[scrapeState]{
				Emitters: []Emitter[scrapeState]{
					{Name: "e1", XorGroup: "g", Metrics: []string{"m"}},
					{Name: "e2", XorGroup: "g", Metrics: []string{"m"}},
				},
			},
			wantError: `xor group "g" has 2 members without requirements`,
		},
		{
			name: "emitter members with different metrics",
			graph: Graph[scrapeState]{
				Emitters: []Emitter[scrapeState]{
					{Name: "e1", XorGroup: "g", Metrics: []string{"m", "n"}},
					{Name: "e2", XorGroup: "g", Requires: []Requirement{Has("x")}, Metrics: []string{"m"}},
				},
			},
			wantError: `xor group "g": emitters "e1" and "e2" declare different metrics`,
		},
		{
			name: "dependency on a group member",
			graph: Graph[scrapeState]{
				Sources: []Source[scrapeState]{
					{Name: "s1", XorGroup: "g"},
					{Name: "b", DependsOn: []string{"s1"}},
				},
			},
			wantError: `source "b" depends on xor group member "s1"; depend on the group instead`,
		},
		{
			name: "emitter on a group member",
			graph: Graph[scrapeState]{
				Sources:  []Source[scrapeState]{{Name: "s1", XorGroup: "g"}},
				Emitters: []Emitter[scrapeState]{{Name: "e", Metrics: []string{"m"}, Sources: []string{"s1"}}},
			},
			wantError: `emitter "e" depends on xor group member "s1"; depend on the group instead`,
		},
		{
			name: "cycle through a group",
			graph: Graph[scrapeState]{
				Sources: []Source[scrapeState]{
					{Name: "s1", XorGroup: "g", DependsOn: []string{"b"}},
					{Name: "s2", XorGroup: "g", Requires: []Requirement{Has("x")}},
					{Name: "b", DependsOn: []string{"g"}},
				},
			},
			wantError: "cycle detected in DAG",
		},
		{
			name: "group referenced twice",
			graph: Graph[scrapeState]{
				Sources:  []Source[scrapeState]{{Name: "s1", XorGroup: "g"}},
				Emitters: []Emitter[scrapeState]{{Name: "e", Metrics: []string{"m"}, Sources: []string{"g", "g"}}},
			},
			wantError: `emitter "e" depends on source "g" twice`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.graph.ComputeSchedule()
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("ComputeSchedule() error = %v, want containing %q", err, tt.wantError)
			}
		})
	}
}

func TestResolveXorGroupPicksMostSpecificMember(t *testing.T) {
	graph := xorGraph()
	policy := map[string]bool{"m": true}

	tests := []struct {
		name string
		caps *Capabilities
		want string
	}{
		{name: "newest range", caps: capsAt(2, 50), want: "a_new"},
		{name: "exact lower bound of newest", caps: capsAt(2, 47), want: "a_new"},
		{name: "middle range", caps: capsAt(2, 40), want: "a_mid"},
		{name: "below every range", caps: capsAt(2, 10), want: "a_old"},
		{name: "axis unset", caps: &Capabilities{}, want: "a_old"},
		{name: "nil capabilities", caps: nil, want: "a_old"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sched, err := graph.PruneSchedule(policy, tt.caps)
			if err != nil {
				t.Fatalf("PruneSchedule() error = %v", err)
			}
			if got := sched.Chosen(); !reflect.DeepEqual(got, map[string]string{"a": tt.want}) {
				t.Fatalf("Chosen() = %v, want a=%s", got, tt.want)
			}
			want := [][]string{{"source:" + tt.want}, {"source:b"}, {"emitter:emit"}}
			if got := sched.Waves(); !reflect.DeepEqual(got, want) {
				t.Fatalf("waves = %#v, want %#v", got, want)
			}
			if got := sched.TotalSources(); got != 2 {
				t.Fatalf("TotalSources() = %d, want 2", got)
			}
		})
	}

	// ComputeSchedule has no capabilities, so the default wins there too.
	sched, err := graph.ComputeSchedule()
	if err != nil {
		t.Fatalf("ComputeSchedule() error = %v", err)
	}
	if got := sched.Chosen()["a"]; got != "a_old" {
		t.Fatalf("ComputeSchedule chose %q, want a_old", got)
	}
}

func TestResolveXorGroupRejectsTie(t *testing.T) {
	graph := Graph[scrapeState]{
		Sources: []Source[scrapeState]{
			{Name: "s33", XorGroup: "g", Requires: []Requirement{AtLeast{testAxis, Version{2, 33}}}},
			{Name: "s47", XorGroup: "g", Requires: []Requirement{AtLeast{testAxis, Version{2, 47}}}},
		},
	}

	if _, err := graph.PruneSchedule(map[string]bool{}, capsAt(2, 40)); err != nil {
		t.Fatalf("PruneSchedule() at 2.40 error = %v, want nil: only s33 matches", err)
	}

	_, err := graph.PruneSchedule(map[string]bool{}, capsAt(2, 50))
	want := `xor group "g": members "s33" and "s47" both match with 1 requirements`
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("PruneSchedule() at 2.50 error = %v, want containing %q", err, want)
	}
}

func TestResolveUnsatisfiedGroupPrunesDependents(t *testing.T) {
	graph := Graph[scrapeState]{
		Sources: []Source[scrapeState]{
			{Name: "s_new", XorGroup: "g", Requires: []Requirement{AtLeast{testAxis, Version{2, 47}}}},
			{Name: "s_ext", XorGroup: "g", Requires: []Requirement{Has("ext")}},
			{Name: "b", DependsOn: []string{"g"}},
			{Name: "c"},
		},
		Emitters: []Emitter[scrapeState]{
			{Name: "emitB", Metrics: []string{"mb"}, Sources: []string{"b"}},
			{Name: "emitC", Metrics: []string{"mc"}, Sources: []string{"c"}},
		},
	}
	policy := map[string]bool{"mb": true, "mc": true}

	sched, err := graph.PruneSchedule(policy, capsAt(2, 10))
	if err != nil {
		t.Fatalf("PruneSchedule() error = %v", err)
	}
	if got := sched.Chosen(); len(got) != 0 {
		t.Fatalf("Chosen() = %v, want empty", got)
	}
	if got, want := sched.Waves(), [][]string{{"source:c"}, {"emitter:emitC"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("waves = %#v, want %#v", got, want)
	}
	if got, want := sched.PrunedSources(), []string{"s_new", "s_ext", "b"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("PrunedSources() = %v, want %v", got, want)
	}
	if got, want := sched.PrunedEmitters(), []string{"emitB"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("PrunedEmitters() = %v, want %v", got, want)
	}

	var caps Capabilities
	caps.Set("ext")
	sched, err = graph.PruneSchedule(policy, &caps)
	if err != nil {
		t.Fatalf("PruneSchedule() with ext error = %v", err)
	}
	if got := sched.Chosen()["g"]; got != "s_ext" {
		t.Fatalf("Chosen() = %v, want g=s_ext", sched.Chosen())
	}
	if got := sched.PrunedEmitters(); len(got) != 0 {
		t.Fatalf("PrunedEmitters() = %v, want none", got)
	}
}

func TestResolveRequirementsDropUngroupedNodeAndDependents(t *testing.T) {
	graph := Graph[scrapeState]{
		Sources: []Source[scrapeState]{
			{Name: "a", Requires: []Requirement{Has("ext")}},
			{Name: "b", DependsOn: []string{"a"}},
		},
		Emitters: []Emitter[scrapeState]{
			{Name: "emitA", Metrics: []string{"ma"}, Sources: []string{"a"}},
			{Name: "emitB", Metrics: []string{"mb"}, Sources: []string{"b"}},
			{Name: "emitOpt", Metrics: []string{"mo"}, Requires: []Requirement{Has("opt")}},
		},
	}

	sched, err := graph.ComputeSchedule()
	if err != nil {
		t.Fatalf("ComputeSchedule() error = %v", err)
	}
	if got := sched.Waves(); len(got) != 0 {
		t.Fatalf("waves = %#v, want none: nothing is satisfiable without capabilities", got)
	}
	if got, want := sched.PrunedEmitters(), []string{"emitA", "emitB", "emitOpt"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("PrunedEmitters() = %v, want %v", got, want)
	}

	var caps Capabilities
	caps.Set("ext")
	sched, err = graph.PruneSchedule(map[string]bool{"ma": true, "mb": true, "mo": true}, &caps)
	if err != nil {
		t.Fatalf("PruneSchedule() error = %v", err)
	}
	want := [][]string{{"source:a"}, {"source:b", "emitter:emitA"}, {"emitter:emitB"}}
	if got := sched.Waves(); !reflect.DeepEqual(got, want) {
		t.Fatalf("waves = %#v, want %#v", got, want)
	}
	if got, want := sched.PrunedEmitters(), []string{"emitOpt"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("PrunedEmitters() = %v, want %v", got, want)
	}
}

// TestResolveEmitterXorGroup models Nova's server status: the newer member
// needs an extra source that only exists from a given microversion.
func TestResolveEmitterXorGroup(t *testing.T) {
	graph := Graph[scrapeState]{
		Sources: []Source[scrapeState]{
			{Name: "servers"},
			{Name: "flavor_map", Requires: []Requirement{AtLeast{testAxis, Version{2, 47}}}},
		},
		Emitters: []Emitter[scrapeState]{
			{Name: "status_mapped", XorGroup: "status", Requires: []Requirement{AtLeast{testAxis, Version{2, 47}}},
				Metrics: []string{"status"}, Sources: []string{"servers", "flavor_map"}},
			{Name: "status_plain", XorGroup: "status", Metrics: []string{"status"}, Sources: []string{"servers"}},
		},
	}

	sched, err := graph.PruneSchedule(map[string]bool{"status": true}, capsAt(2, 50))
	if err != nil {
		t.Fatalf("PruneSchedule() error = %v", err)
	}
	if got := sched.Chosen()["status"]; got != "status_mapped" {
		t.Fatalf("Chosen() = %v, want status=status_mapped", sched.Chosen())
	}
	want := [][]string{{"source:servers", "source:flavor_map"}, {"emitter:status_mapped"}}
	if got := sched.Waves(); !reflect.DeepEqual(got, want) {
		t.Fatalf("waves = %#v, want %#v", got, want)
	}

	sched, err = graph.PruneSchedule(map[string]bool{"status": true}, capsAt(2, 10))
	if err != nil {
		t.Fatalf("PruneSchedule() error = %v", err)
	}
	if got := sched.Chosen()["status"]; got != "status_plain" {
		t.Fatalf("Chosen() = %v, want status=status_plain", sched.Chosen())
	}
	want = [][]string{{"source:servers"}, {"emitter:status_plain"}}
	if got := sched.Waves(); !reflect.DeepEqual(got, want) {
		t.Fatalf("waves = %#v, want %#v", got, want)
	}

	// A disabled metric drops the chosen member like any other emitter, and
	// with it the sources only it needed.
	sched, err = graph.PruneSchedule(map[string]bool{"status": false}, capsAt(2, 50))
	if err != nil {
		t.Fatalf("PruneSchedule() error = %v", err)
	}
	if got := sched.Waves(); len(got) != 0 {
		t.Fatalf("waves = %#v, want none", got)
	}
}

func TestRunExecutesOnlyChosenXorMember(t *testing.T) {
	var mu sync.Mutex
	var fetched []string
	record := func(name string) func(context.Context, *scrapeState) error {
		return func(context.Context, *scrapeState) error {
			mu.Lock()
			defer mu.Unlock()
			fetched = append(fetched, name)
			return nil
		}
	}

	graph := xorGraph()
	for i := range graph.Sources {
		graph.Sources[i].Fetch = record(graph.Sources[i].Name)
	}
	graph.Sources[3].Fetch = func(_ context.Context, s *scrapeState) error {
		mu.Lock()
		defer mu.Unlock()
		// b must run after the chosen member of a, never in parallel with it.
		if len(fetched) != 1 {
			return errors.New("b ran before the group member completed")
		}
		fetched = append(fetched, "b")
		return nil
	}

	sched, err := graph.PruneSchedule(map[string]bool{"m": true}, capsAt(2, 40))
	if err != nil {
		t.Fatalf("PruneSchedule() error = %v", err)
	}
	result := sched.Run(context.Background(), new(scrapeState), drain(t), RunHooks{})
	want := RunResult{SourcesOK: 2, EmittersOK: 1}
	if result != want {
		t.Fatalf("Run() = %+v, want %+v", result, want)
	}
	if got := []string{"a_mid", "b"}; !reflect.DeepEqual(fetched, got) {
		t.Fatalf("fetched = %v, want %v", fetched, got)
	}
}
