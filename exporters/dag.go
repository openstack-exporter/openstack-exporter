package exporters

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Source is a data-fetching node in an exporter collection graph.
//
// Fetch writes into the per-scrape state S. Sources that run concurrently
// must write to disjoint fields of S. A Source that fails blocks every node
// that depends on it, directly or transitively; a Source that wants its
// dependents to run despite partial failures must absorb those failures
// itself (record them in S and return nil).
type Source[S any] struct {
	Name      string
	DependsOn []string
	Fetch     func(ctx context.Context, s *S) error
}

// Emitter converts scrape state into metrics once all of its Sources have
// completed successfully. Metrics lists the metric names the emitter may
// produce; PruneSchedule drops an emitter whose metrics are all disabled.
type Emitter[S any] struct {
	Name    string
	Metrics []string
	Sources []string
	Emit    func(ctx context.Context, s *S, ch chan<- prometheus.Metric) error
}

// Graph declares the data sources and metric emitters for an exporter.
type Graph[S any] struct {
	Sources  []Source[S]
	Emitters []Emitter[S]
}

type nodeKind int

const (
	nodeSource nodeKind = iota
	nodeEmitter
)

type node struct {
	kind  nodeKind
	index int
}

// Schedule is a validated, dependency-ordered execution plan for a subset of
// a Graph. Build one with Graph.ComputeSchedule or Graph.PruneSchedule and
// execute it with Run, once per scrape.
type Schedule[S any] struct {
	graph        *Graph[S]
	nodes        []node
	dependencies [][]int
	dependents   [][]int
	waves        [][]int
}

// ComputeSchedule validates the graph and returns a schedule of every node.
func (g *Graph[S]) ComputeSchedule() (Schedule[S], error) {
	liveSources := make([]bool, len(g.Sources))
	for i := range liveSources {
		liveSources[i] = true
	}

	liveEmitters := make([]bool, len(g.Emitters))
	for i := range liveEmitters {
		liveEmitters[i] = true
	}

	return g.computeSchedule(liveSources, liveEmitters)
}

// PruneSchedule returns the nodes required by enabled metrics: an emitter is
// kept when at least one of its metrics is enabled, and a source is kept when
// a live emitter needs it, directly or through other sources. The policy must
// contain every metric declared by an emitter so stale metric names fail fast.
func (g *Graph[S]) PruneSchedule(metricEnabled map[string]bool) (Schedule[S], error) {
	if _, err := g.ComputeSchedule(); err != nil {
		return Schedule[S]{}, err
	}

	liveEmitters := make([]bool, len(g.Emitters))
	for i, emitter := range g.Emitters {
		for _, metric := range emitter.Metrics {
			enabled, known := metricEnabled[metric]
			if !known {
				return Schedule[S]{}, fmt.Errorf("emitter %q declares unknown metric %q", emitter.Name, metric)
			}
			liveEmitters[i] = liveEmitters[i] || enabled
		}
	}

	sourceIndex := make(map[string]int, len(g.Sources))
	for i, source := range g.Sources {
		sourceIndex[source.Name] = i
	}

	liveSources := make([]bool, len(g.Sources))
	var markSource func(string)
	markSource = func(name string) {
		index, exists := sourceIndex[name]
		if !exists || liveSources[index] {
			return
		}

		liveSources[index] = true
		for _, dependency := range g.Sources[index].DependsOn {
			markSource(dependency)
		}
	}

	for i, emitter := range g.Emitters {
		if !liveEmitters[i] {
			continue
		}
		for _, source := range emitter.Sources {
			markSource(source)
		}
	}

	return g.computeSchedule(liveSources, liveEmitters)
}

func (g *Graph[S]) computeSchedule(liveSources, liveEmitters []bool) (Schedule[S], error) {
	if len(liveSources) != len(g.Sources) {
		return Schedule[S]{}, fmt.Errorf("source liveness length mismatch")
	}
	if len(liveEmitters) != len(g.Emitters) {
		return Schedule[S]{}, fmt.Errorf("emitter liveness length mismatch")
	}

	sourceIndex := make(map[string]int, len(g.Sources))
	for i, source := range g.Sources {
		if _, exists := sourceIndex[source.Name]; exists {
			return Schedule[S]{}, fmt.Errorf("duplicate source name: %q", source.Name)
		}
		sourceIndex[source.Name] = i
	}
	for _, source := range g.Sources {
		if err := checkReferences("source", source.Name, source.DependsOn, sourceIndex); err != nil {
			return Schedule[S]{}, err
		}
	}

	emitterNames := make(map[string]struct{}, len(g.Emitters))
	for _, emitter := range g.Emitters {
		if _, exists := emitterNames[emitter.Name]; exists {
			return Schedule[S]{}, fmt.Errorf("duplicate emitter name: %q", emitter.Name)
		}
		emitterNames[emitter.Name] = struct{}{}
		if err := checkReferences("emitter", emitter.Name, emitter.Sources, sourceIndex); err != nil {
			return Schedule[S]{}, err
		}
	}

	var nodes []node
	sourceNodes := make([]int, len(g.Sources))
	for i := range sourceNodes {
		sourceNodes[i] = -1
		if liveSources[i] {
			sourceNodes[i] = len(nodes)
			nodes = append(nodes, node{kind: nodeSource, index: i})
		}
	}

	emitterNodes := make([]int, len(g.Emitters))
	for i := range emitterNodes {
		emitterNodes[i] = -1
		if liveEmitters[i] {
			emitterNodes[i] = len(nodes)
			nodes = append(nodes, node{kind: nodeEmitter, index: i})
		}
	}

	dependencies := make([][]int, len(nodes))
	dependents := make([][]int, len(nodes))
	addDependency := func(node, dependency int) {
		if node < 0 || dependency < 0 {
			return
		}
		dependencies[node] = append(dependencies[node], dependency)
		dependents[dependency] = append(dependents[dependency], node)
	}

	for source, node := range sourceNodes {
		if node < 0 {
			continue
		}
		for _, dependency := range g.Sources[source].DependsOn {
			addDependency(node, sourceNodes[sourceIndex[dependency]])
		}
	}
	for emitter, node := range emitterNodes {
		if node < 0 {
			continue
		}
		for _, source := range g.Emitters[emitter].Sources {
			addDependency(node, sourceNodes[sourceIndex[source]])
		}
	}

	waves, err := scheduleWaves(dependencies, dependents)
	if err != nil {
		return Schedule[S]{}, err
	}

	return Schedule[S]{
		graph:        g,
		nodes:        nodes,
		dependencies: dependencies,
		dependents:   dependents,
		waves:        waves,
	}, nil
}

// checkReferences verifies that every referenced source exists and that no
// source is referenced twice, which would leave a node waiting forever.
func checkReferences(kind, name string, refs []string, sourceIndex map[string]int) error {
	seen := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		if _, exists := sourceIndex[ref]; !exists {
			return fmt.Errorf("%s %q depends on missing source %q", kind, name, ref)
		}
		if _, dup := seen[ref]; dup {
			return fmt.Errorf("%s %q depends on source %q twice", kind, name, ref)
		}
		seen[ref] = struct{}{}
	}
	return nil
}

// scheduleWaves groups nodes into topological waves and detects cycles. Run
// does not wait for whole waves; the grouping documents the topology for
// logs and tests.
func scheduleWaves(dependencies, dependents [][]int) ([][]int, error) {
	remaining := make([]int, len(dependencies))
	var ready []int
	for i := range dependencies {
		remaining[i] = len(dependencies[i])
		if remaining[i] == 0 {
			ready = append(ready, i)
		}
	}

	var waves [][]int
	visited := 0
	for len(ready) > 0 {
		wave := append([]int(nil), ready...)
		waves = append(waves, wave)
		visited += len(wave)

		var next []int
		for _, node := range wave {
			for _, dependent := range dependents[node] {
				remaining[dependent]--
				if remaining[dependent] == 0 {
					next = append(next, dependent)
				}
			}
		}
		ready = next
	}

	if visited != len(dependencies) {
		return nil, fmt.Errorf("cycle detected in DAG")
	}

	return waves, nil
}

// ---------------------------------------------------------------------------
// Introspection, for logging and tests
// ---------------------------------------------------------------------------

func (sched *Schedule[S]) nodeName(n node) string {
	switch n.kind {
	case nodeSource:
		return "source:" + sched.graph.Sources[n.index].Name
	case nodeEmitter:
		return "emitter:" + sched.graph.Emitters[n.index].Name
	default:
		return "unknown"
	}
}

// Waves returns the node names ("source:x", "emitter:y") grouped by
// topological wave. Execution is dependency-driven, so a node in a later wave
// starts as soon as its own dependencies complete.
func (sched *Schedule[S]) Waves() [][]string {
	waves := make([][]string, len(sched.waves))
	for i, wave := range sched.waves {
		waves[i] = make([]string, len(wave))
		for j, index := range wave {
			waves[i][j] = sched.nodeName(sched.nodes[index])
		}
	}
	return waves
}

// TotalSources returns the number of sources in the schedule.
func (sched *Schedule[S]) TotalSources() int {
	n := 0
	for _, nd := range sched.nodes {
		if nd.kind == nodeSource {
			n++
		}
	}
	return n
}

// PrunedSources returns the graph sources that are not part of the schedule.
func (sched *Schedule[S]) PrunedSources() []string {
	live := make(map[int]bool, len(sched.nodes))
	for _, nd := range sched.nodes {
		if nd.kind == nodeSource {
			live[nd.index] = true
		}
	}
	var pruned []string
	for i, source := range sched.graph.Sources {
		if !live[i] {
			pruned = append(pruned, source.Name)
		}
	}
	return pruned
}

// PrunedEmitters returns the graph emitters that are not part of the schedule.
func (sched *Schedule[S]) PrunedEmitters() []string {
	live := make(map[int]bool, len(sched.nodes))
	for _, nd := range sched.nodes {
		if nd.kind == nodeEmitter {
			live[nd.index] = true
		}
	}
	var pruned []string
	for i, emitter := range sched.graph.Emitters {
		if !live[i] {
			pruned = append(pruned, emitter.Name)
		}
	}
	return pruned
}

// ---------------------------------------------------------------------------
// Execution
// ---------------------------------------------------------------------------

// RunHooks receives completion events from Run. Hooks are invoked from the
// goroutine that ran the node, so they must be safe for concurrent use. All
// fields are optional.
type RunHooks struct {
	SourceDone  func(name string, elapsed time.Duration, err error)
	EmitterDone func(name string, elapsed time.Duration, err error)
}

// RunResult summarises one Run. A node is skipped when a dependency failed
// or was skipped, or when the context was cancelled before it started.
type RunResult struct {
	SourcesOK       int
	SourcesFailed   int
	SourcesSkipped  int
	EmittersOK      int
	EmittersFailed  int
	EmittersSkipped int
}

// Healthy reports whether at least one source fetched successfully, which is
// what the per-exporter "up" gauge reflects.
func (r RunResult) Healthy() bool {
	return r.SourcesOK > 0
}

type nodeStatus int

const (
	statusOK nodeStatus = iota
	statusFailed
	statusSkipped
)

type completion struct {
	index  int
	status nodeStatus
}

// Run executes the schedule against state, starting every node as soon as its
// dependencies have completed successfully. Sources run concurrently when
// independent. A node downstream of a failed source is skipped, so an emitter
// never observes partially fetched state. A panic inside Fetch or Emit is
// recovered and reported as that node's error.
func (sched *Schedule[S]) Run(ctx context.Context, state *S, ch chan<- prometheus.Metric, hooks RunHooks) RunResult {
	var result RunResult
	n := len(sched.nodes)
	if n == 0 {
		return result
	}

	remaining := make([]int, n)
	blocked := make([]bool, n)
	// Every node reports exactly once, so a buffer of n never blocks.
	completed := make(chan completion, n)
	var wg sync.WaitGroup

	start := func(i int) {
		if blocked[i] || ctx.Err() != nil {
			completed <- completion{index: i, status: statusSkipped}
			return
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			status := statusOK
			if err := sched.runNode(ctx, i, state, ch, hooks); err != nil {
				status = statusFailed
			}
			completed <- completion{index: i, status: status}
		}()
	}

	for i := range sched.nodes {
		remaining[i] = len(sched.dependencies[i])
		if remaining[i] == 0 {
			start(i)
		}
	}

	for done := 0; done < n; done++ {
		c := <-completed
		result.record(sched.nodes[c.index].kind, c.status)
		for _, dependent := range sched.dependents[c.index] {
			if c.status != statusOK {
				blocked[dependent] = true
			}
			remaining[dependent]--
			if remaining[dependent] == 0 {
				start(dependent)
			}
		}
	}
	wg.Wait()

	return result
}

func (r *RunResult) record(kind nodeKind, status nodeStatus) {
	switch {
	case kind == nodeSource && status == statusOK:
		r.SourcesOK++
	case kind == nodeSource && status == statusFailed:
		r.SourcesFailed++
	case kind == nodeSource:
		r.SourcesSkipped++
	case status == statusOK:
		r.EmittersOK++
	case status == statusFailed:
		r.EmittersFailed++
	default:
		r.EmittersSkipped++
	}
}

func (sched *Schedule[S]) runNode(ctx context.Context, i int, state *S, ch chan<- prometheus.Metric, hooks RunHooks) (err error) {
	nd := sched.nodes[i]
	started := time.Now()
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v\n%s", r, debug.Stack())
		}
		elapsed := time.Since(started)
		switch nd.kind {
		case nodeSource:
			if hooks.SourceDone != nil {
				hooks.SourceDone(sched.graph.Sources[nd.index].Name, elapsed, err)
			}
		case nodeEmitter:
			if hooks.EmitterDone != nil {
				hooks.EmitterDone(sched.graph.Emitters[nd.index].Name, elapsed, err)
			}
		}
	}()

	switch nd.kind {
	case nodeSource:
		if fetch := sched.graph.Sources[nd.index].Fetch; fetch != nil {
			return fetch(ctx, state)
		}
	case nodeEmitter:
		if emit := sched.graph.Emitters[nd.index].Emit; emit != nil {
			return emit(ctx, state, ch)
		}
	}
	return nil
}
