package exporters

import (
	"context"
	"fmt"
	"runtime/debug"
	"sort"
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
//
// DependsOn names other sources, or XOR groups of sources, that must have
// completed before Fetch runs. Requires and XorGroup are described on Graph.
type Source[S any] struct {
	Name      string
	XorGroup  string
	Requires  []Requirement
	DependsOn []string
	Fetch     func(ctx context.Context, s *S) error
}

// Emitter converts scrape state into metrics once all of its Sources have
// completed successfully. Metrics lists the metric names the emitter may
// produce and must not be empty; PruneSchedule drops an emitter whose
// metrics are all disabled. Sources names sources or XOR groups of sources.
type Emitter[S any] struct {
	Name     string
	XorGroup string
	Requires []Requirement
	Metrics  []string
	Sources  []string
	Emit     func(ctx context.Context, s *S, ch chan<- prometheus.Metric) error
}

// Graph declares the data sources and metric emitters for an exporter.
//
// A node may declare Requires, conditions on the Capabilities the exporter
// found when it bound to a cloud. A node whose requirements do not hold is
// left out of the schedule, together with everything that depends on it.
//
// Nodes of one kind that are alternative implementations of the same thing
// share an XorGroup. Dependents name the group, never a member, and exactly
// one member is scheduled: among the members whose requirements hold, the
// one declaring the most requirements. A member with no requirements is the
// default and is taken only when nothing more specific holds. Two candidates
// with the same number of requirements are an error, so overlapping ranges
// must be made disjoint with Below. Declaration order carries no meaning.
// Members of an emitter group must declare the same metrics, so the group
// has exactly one publisher for each of them. The choice is made once, when
// the schedule is computed, and never at scrape time.
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
	chosen       map[string]string
}

// ComputeSchedule validates the graph and returns a schedule of every node
// that needs no capabilities: nodes with requirements are left out and XOR
// groups resolve to their default member.
func (g *Graph[S]) ComputeSchedule() (Schedule[S], error) {
	return g.resolve(nil, nil)
}

// PruneSchedule returns the nodes required by enabled metrics among those
// whose requirements hold against caps: an emitter is kept when at least one
// of its metrics is enabled, and a source is kept when a live emitter needs
// it, directly or through other sources. The policy must contain every
// metric declared by an emitter so stale metric names fail fast. A nil caps
// is an empty capability set.
func (g *Graph[S]) PruneSchedule(metricEnabled map[string]bool, caps *Capabilities) (Schedule[S], error) {
	if metricEnabled == nil {
		metricEnabled = map[string]bool{}
	}
	return g.resolve(metricEnabled, caps)
}

// graphIndex is the static structure of a validated graph. It carries no
// type parameter so the checks can live in plain methods.
type graphIndex struct {
	sourceByName  map[string]int
	sourceGrouped []bool
	sourceGroups  map[string][]int
	emitterGroups map[string][]int
	sourceDeps    [][]string
}

// validate checks everything that does not depend on capabilities or policy:
// names, group shapes, references and cycles.
func (g *Graph[S]) validate() (graphIndex, error) {
	idx := graphIndex{
		sourceByName:  make(map[string]int, len(g.Sources)),
		sourceGrouped: make([]bool, len(g.Sources)),
		sourceGroups:  make(map[string][]int),
		emitterGroups: make(map[string][]int),
		sourceDeps:    make([][]string, len(g.Sources)),
	}

	for i, source := range g.Sources {
		if _, exists := idx.sourceByName[source.Name]; exists {
			return idx, fmt.Errorf("duplicate source name: %q", source.Name)
		}
		idx.sourceByName[source.Name] = i
		idx.sourceDeps[i] = source.DependsOn
		if source.XorGroup != "" {
			idx.sourceGrouped[i] = true
			idx.sourceGroups[source.XorGroup] = append(idx.sourceGroups[source.XorGroup], i)
		}
	}

	emitterNames := make(map[string]struct{}, len(g.Emitters))
	for i, emitter := range g.Emitters {
		if _, exists := emitterNames[emitter.Name]; exists {
			return idx, fmt.Errorf("duplicate emitter name: %q", emitter.Name)
		}
		emitterNames[emitter.Name] = struct{}{}
		if len(emitter.Metrics) == 0 {
			return idx, fmt.Errorf("emitter %q declares no metrics", emitter.Name)
		}
		if emitter.XorGroup != "" {
			idx.emitterGroups[emitter.XorGroup] = append(idx.emitterGroups[emitter.XorGroup], i)
		}
	}

	for group, members := range idx.sourceGroups {
		if _, clash := idx.sourceByName[group]; clash {
			return idx, fmt.Errorf("xor group %q has the same name as a source", group)
		}
		if _, clash := idx.emitterGroups[group]; clash {
			return idx, fmt.Errorf("xor group %q mixes sources and emitters", group)
		}
		defaults := 0
		for _, i := range members {
			if len(g.Sources[i].Requires) == 0 {
				defaults++
			}
		}
		if defaults > 1 {
			return idx, fmt.Errorf("xor group %q has %d members without requirements", group, defaults)
		}
	}
	for group, members := range idx.emitterGroups {
		if _, clash := idx.sourceByName[group]; clash {
			return idx, fmt.Errorf("xor group %q has the same name as a source", group)
		}
		defaults := 0
		for _, i := range members {
			if len(g.Emitters[i].Requires) == 0 {
				defaults++
			}
			if !sameMetrics(g.Emitters[members[0]].Metrics, g.Emitters[i].Metrics) {
				return idx, fmt.Errorf("xor group %q: emitters %q and %q declare different metrics",
					group, g.Emitters[members[0]].Name, g.Emitters[i].Name)
			}
		}
		if defaults > 1 {
			return idx, fmt.Errorf("xor group %q has %d members without requirements", group, defaults)
		}
	}

	for _, source := range g.Sources {
		if err := idx.checkReferences("source", source.Name, source.DependsOn); err != nil {
			return idx, err
		}
	}
	for _, emitter := range g.Emitters {
		if err := idx.checkReferences("emitter", emitter.Name, emitter.Sources); err != nil {
			return idx, err
		}
	}

	return idx, idx.checkCycles()
}

// checkReferences verifies that every reference names an ungrouped source or
// a source group, and that nothing is referenced twice, which would leave a
// node waiting forever.
func (idx graphIndex) checkReferences(kind, name string, refs []string) error {
	seen := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		if _, dup := seen[ref]; dup {
			return fmt.Errorf("%s %q depends on source %q twice", kind, name, ref)
		}
		seen[ref] = struct{}{}
		if _, isGroup := idx.sourceGroups[ref]; isGroup {
			continue
		}
		index, exists := idx.sourceByName[ref]
		if !exists {
			return fmt.Errorf("%s %q depends on missing source %q", kind, name, ref)
		}
		if idx.sourceGrouped[index] {
			return fmt.Errorf("%s %q depends on xor group member %q; depend on the group instead", kind, name, ref)
		}
	}
	return nil
}

// checkCycles runs the wave scheduler over the static source topology, with
// every XOR group collapsed into one vertex carrying all of its members'
// dependencies. Emitters cannot take part in a cycle: nothing depends on them.
func (idx graphIndex) checkCycles() error {
	vertexOf := make(map[string]int)
	var vertices []string
	vertex := func(name string) int {
		if v, ok := vertexOf[name]; ok {
			return v
		}
		vertexOf[name] = len(vertices)
		vertices = append(vertices, name)
		return vertexOf[name]
	}
	// Walk sources by index so every vertex exists before edges are added.
	sourceNames := make([]string, len(idx.sourceByName))
	for name, i := range idx.sourceByName {
		sourceNames[i] = name
	}
	groupOf := make([]string, len(sourceNames))
	for group, members := range idx.sourceGroups {
		for _, i := range members {
			groupOf[i] = group
		}
	}
	from := make([]int, len(sourceNames))
	for i := range sourceNames {
		if idx.sourceGrouped[i] {
			from[i] = vertex(groupOf[i])
		} else {
			from[i] = vertex(sourceNames[i])
		}
	}

	dependencies := make([][]int, len(vertices))
	dependents := make([][]int, len(vertices))
	for i, deps := range idx.sourceDeps {
		for _, ref := range deps {
			to := vertex(ref)
			dependencies[from[i]] = append(dependencies[from[i]], to)
			dependents[to] = append(dependents[to], from[i])
		}
	}
	_, err := scheduleWaves(dependencies, dependents)
	return err
}

func sameMetrics(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	as := append([]string(nil), a...)
	bs := append([]string(nil), b...)
	sort.Strings(as)
	sort.Strings(bs)
	for i := range as {
		if as[i] != bs[i] {
			return false
		}
	}
	return true
}

// pickMember applies the XOR rule to one group: the candidate with the most
// requirements wins, a tie is an error, no candidate yields -1.
func pickMember(group string, members []int, requires func(int) []Requirement, name func(int) string, caps *Capabilities) (int, error) {
	best, bestLen := -1, -1
	tie := -1
	for _, i := range members {
		reqs := requires(i)
		if !satisfied(reqs, caps) {
			continue
		}
		switch {
		case len(reqs) > bestLen:
			best, bestLen, tie = i, len(reqs), -1
		case len(reqs) == bestLen:
			tie = i
		}
	}
	if tie >= 0 {
		return -1, fmt.Errorf("xor group %q: members %q and %q both match with %d requirements (%s)",
			group, name(best), name(tie), bestLen, requirementsString(requires(best)))
	}
	return best, nil
}

// resolve validates the graph, applies capabilities and XOR choices, prunes
// by metric policy when one is given, and builds the execution plan.
func (g *Graph[S]) resolve(metricEnabled map[string]bool, caps *Capabilities) (Schedule[S], error) {
	idx, err := g.validate()
	if err != nil {
		return Schedule[S]{}, err
	}

	chosen := make(map[string]string)

	// target maps every reference (ungrouped source name or group name) to
	// the source index that will serve it, or -1 when nothing can.
	target := make(map[string]int, len(g.Sources))
	for i, source := range g.Sources {
		if idx.sourceGrouped[i] {
			continue
		}
		target[source.Name] = -1
		if satisfied(source.Requires, caps) {
			target[source.Name] = i
		}
	}
	for group, members := range idx.sourceGroups {
		pick, err := pickMember(group, members,
			func(i int) []Requirement { return g.Sources[i].Requires },
			func(i int) string { return g.Sources[i].Name }, caps)
		if err != nil {
			return Schedule[S]{}, err
		}
		target[group] = pick
		if pick >= 0 {
			chosen[group] = g.Sources[pick].Name
		}
	}

	// A source is available when it was selected and every reference it
	// makes resolves to an available source. The static cycle check makes
	// the recursion finite.
	sourceAvail := make([]int8, len(g.Sources)) // 0 unknown, 1 yes, -1 no
	var available func(ref string) bool
	available = func(ref string) bool {
		i, ok := target[ref]
		if !ok || i < 0 {
			return false
		}
		if sourceAvail[i] != 0 {
			return sourceAvail[i] > 0
		}
		sourceAvail[i] = 1
		for _, dep := range g.Sources[i].DependsOn {
			if !available(dep) {
				sourceAvail[i] = -1
				break
			}
		}
		return sourceAvail[i] > 0
	}

	emitterAvail := make([]bool, len(g.Emitters))
	for i, emitter := range g.Emitters {
		if emitter.XorGroup != "" || !satisfied(emitter.Requires, caps) {
			continue
		}
		emitterAvail[i] = true
	}
	for group, members := range idx.emitterGroups {
		pick, err := pickMember(group, members,
			func(i int) []Requirement { return g.Emitters[i].Requires },
			func(i int) string { return g.Emitters[i].Name }, caps)
		if err != nil {
			return Schedule[S]{}, err
		}
		if pick >= 0 {
			emitterAvail[pick] = true
			chosen[group] = g.Emitters[pick].Name
		}
	}
	for i, emitter := range g.Emitters {
		if !emitterAvail[i] {
			continue
		}
		for _, ref := range emitter.Sources {
			if !available(ref) {
				emitterAvail[i] = false
				break
			}
		}
	}

	liveEmitters := make([]bool, len(g.Emitters))
	for i, emitter := range g.Emitters {
		enabled := metricEnabled == nil
		for _, metric := range emitter.Metrics {
			if metricEnabled == nil {
				break
			}
			on, known := metricEnabled[metric]
			if !known {
				return Schedule[S]{}, fmt.Errorf("emitter %q declares unknown metric %q", emitter.Name, metric)
			}
			enabled = enabled || on
		}
		liveEmitters[i] = emitterAvail[i] && enabled
	}

	// Without a policy every available source is scheduled, whether or not an
	// emitter needs it. With one, only sources that live emitters reach.
	liveSources := make([]bool, len(g.Sources))
	if metricEnabled == nil {
		for i, source := range g.Sources {
			ref := source.Name
			if idx.sourceGrouped[i] {
				ref = source.XorGroup
			}
			liveSources[i] = target[ref] == i && available(ref)
		}
	}
	var markSource func(ref string)
	markSource = func(ref string) {
		i := target[ref]
		if liveSources[i] {
			return
		}
		liveSources[i] = true
		for _, dep := range g.Sources[i].DependsOn {
			markSource(dep)
		}
	}
	for i, emitter := range g.Emitters {
		if !liveEmitters[i] {
			continue
		}
		for _, ref := range emitter.Sources {
			markSource(ref)
		}
	}

	sched, err := g.build(liveSources, liveEmitters, target)
	if err != nil {
		return Schedule[S]{}, err
	}
	sched.chosen = chosen
	return sched, nil
}

// build assembles the execution plan for the live nodes, with every
// reference already resolved to a source index by target.
func (g *Graph[S]) build(liveSources, liveEmitters []bool, target map[string]int) (Schedule[S], error) {
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
	link := func(n int, ref string) error {
		source := target[ref]
		if source < 0 || sourceNodes[source] < 0 {
			return fmt.Errorf("internal: live node depends on dead reference %q", ref)
		}
		dependency := sourceNodes[source]
		dependencies[n] = append(dependencies[n], dependency)
		dependents[dependency] = append(dependents[dependency], n)
		return nil
	}

	for source, n := range sourceNodes {
		if n < 0 {
			continue
		}
		for _, ref := range g.Sources[source].DependsOn {
			if err := link(n, ref); err != nil {
				return Schedule[S]{}, err
			}
		}
	}
	for emitter, n := range emitterNodes {
		if n < 0 {
			continue
		}
		for _, ref := range g.Emitters[emitter].Sources {
			if err := link(n, ref); err != nil {
				return Schedule[S]{}, err
			}
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

// Chosen returns, for every XOR group that resolved, the name of the member
// that was scheduled. Groups with no satisfied member are absent.
func (sched *Schedule[S]) Chosen() map[string]string {
	out := make(map[string]string, len(sched.chosen))
	for group, member := range sched.chosen {
		out[group] = member
	}
	return out
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

// Healthy reports whether the run is in a good state, which is what the
// per-exporter "up" gauge reflects: at least one source fetched successfully,
// or no source ran at all. The latter covers a schedule whose sources were
// all pruned because every metric is disabled; nothing failed, so the
// exporter is still up.
func (r RunResult) Healthy() bool {
	if r.SourcesOK > 0 {
		return true
	}
	return r.SourcesFailed == 0 && r.SourcesSkipped == 0
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
