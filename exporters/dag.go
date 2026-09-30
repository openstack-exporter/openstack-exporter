package exporters

import (
	"context"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

// Source is a data-fetching node in an exporter collection graph.
type Source[E, S any] struct {
	Name      string
	DependsOn []string
	Fetch     func(E, context.Context, *S) error
}

// Emitter emits metrics after all of its source dependencies complete.
type Emitter[E, S any] struct {
	Name    string
	Metrics []string
	Sources []string
	Emit    func(E, context.Context, *S, chan<- prometheus.Metric) error
}

// Graph declares the data sources and metric emitters for an exporter.
type Graph[E, S any] struct {
	Sources  []Source[E, S]
	Emitters []Emitter[E, S]
}

type scheduleNodeKind int

const (
	scheduleSource scheduleNodeKind = iota
	scheduleEmitter
)

type scheduleNode struct {
	kind  scheduleNodeKind
	index int
}

// Schedule is a topologically sorted execution plan for a Graph.
type Schedule struct {
	nodes        []scheduleNode
	dependencies [][]int
	dependents   [][]int
	waves        [][]int
}

// ComputeSchedule validates the graph and returns its complete schedule.
func (g *Graph[E, S]) ComputeSchedule() (Schedule, error) {
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

// PruneSchedule returns the nodes required by enabled metrics. The policy must
// contain every metric declared by an emitter so stale metric names fail fast.
func (g *Graph[E, S]) PruneSchedule(metricEnabled map[string]bool) (Schedule, error) {
	if _, err := g.ComputeSchedule(); err != nil {
		return Schedule{}, err
	}

	liveEmitters := make([]bool, len(g.Emitters))
	for i, emitter := range g.Emitters {
		for _, metric := range emitter.Metrics {
			enabled, known := metricEnabled[metric]
			if !known {
				return Schedule{}, fmt.Errorf("emitter %q declares unknown metric %q", emitter.Name, metric)
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

func (g *Graph[E, S]) computeSchedule(liveSources, liveEmitters []bool) (Schedule, error) {
	if len(liveSources) != len(g.Sources) {
		return Schedule{}, fmt.Errorf("source liveness length mismatch")
	}
	if len(liveEmitters) != len(g.Emitters) {
		return Schedule{}, fmt.Errorf("emitter liveness length mismatch")
	}

	sourceIndex := make(map[string]int, len(g.Sources))
	for i, source := range g.Sources {
		if _, exists := sourceIndex[source.Name]; exists {
			return Schedule{}, fmt.Errorf("duplicate source name: %q", source.Name)
		}
		sourceIndex[source.Name] = i
	}
	for _, source := range g.Sources {
		for _, dependency := range source.DependsOn {
			if _, exists := sourceIndex[dependency]; !exists {
				return Schedule{}, fmt.Errorf("source %q depends on missing source %q", source.Name, dependency)
			}
		}
	}

	emitterNames := make(map[string]struct{}, len(g.Emitters))
	for _, emitter := range g.Emitters {
		if _, exists := emitterNames[emitter.Name]; exists {
			return Schedule{}, fmt.Errorf("duplicate emitter name: %q", emitter.Name)
		}
		emitterNames[emitter.Name] = struct{}{}
		for _, source := range emitter.Sources {
			if _, exists := sourceIndex[source]; !exists {
				return Schedule{}, fmt.Errorf("emitter %q depends on missing source %q", emitter.Name, source)
			}
		}
	}

	var nodes []scheduleNode
	sourceNodes := make([]int, len(g.Sources))
	for i := range sourceNodes {
		sourceNodes[i] = -1
		if liveSources[i] {
			sourceNodes[i] = len(nodes)
			nodes = append(nodes, scheduleNode{kind: scheduleSource, index: i})
		}
	}

	emitterNodes := make([]int, len(g.Emitters))
	for i := range emitterNodes {
		emitterNodes[i] = -1
		if liveEmitters[i] {
			emitterNodes[i] = len(nodes)
			nodes = append(nodes, scheduleNode{kind: scheduleEmitter, index: i})
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
		return Schedule{}, err
	}

	return Schedule{
		nodes:        nodes,
		dependencies: dependencies,
		dependents:   dependents,
		waves:        waves,
	}, nil
}

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
