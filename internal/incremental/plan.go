package incremental

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"

	"scriptweaver/internal/core"
)

// NodeExecutionDecision represents the deterministic plan decision for a task.
//
// From docs/sprints/sprint-02/in-process/incremental-engine/tdd.md:
// decisions are strictly Execute or ReuseCache (no Skip state).
type NodeExecutionDecision string

const (
	DecisionExecute    NodeExecutionDecision = "Execute"
	DecisionReuseCache NodeExecutionDecision = "ReuseCache"
)

// IncrementalPlan maps every node name to a deterministic execution decision.
//
// Prohibition (spec): runtime-conditional skipping is unsupported; every node must have a decision.
type IncrementalPlan struct {
	// Order is the deterministic ordered task list for evaluation/serialization.
	// This overlays the static graph; it does not mutate graph structure.
	Order []string

	Decisions map[string]NodeExecutionDecision
}

// PlanningResult is the deterministic output of the incremental planning phase.
//
// It contains:
//   - Invalidation: per-task invalidation state (source of truth)
//   - Plan: per-task execution decision overlay
//
// The planning phase must not execute tasks; it is computed purely from snapshots and cache presence.
type PlanningResult struct {
	Invalidation InvalidationMap
	Plan         *IncrementalPlan
}

// SerializeDeterministic returns a deterministic byte representation of the plan.
//
// Determinism strategy:
//   - Serialize tasks in p.Order
//   - For each task: name + decision, length-prefixed
func (p *IncrementalPlan) SerializeDeterministic() []byte {
	if p == nil {
		return nil
	}

	h := sha256.New()

	writeField := func(data []byte) {
		length := uint64(len(data))
		lengthBytes := []byte{
			byte(length >> 56),
			byte(length >> 48),
			byte(length >> 40),
			byte(length >> 32),
			byte(length >> 24),
			byte(length >> 16),
			byte(length >> 8),
			byte(length),
		}
		h.Write(lengthBytes)
		h.Write(data)
	}

	// Order count + entries
	writeField([]byte{byte(len(p.Order))})
	for _, name := range p.Order {
		writeField([]byte(name))
		dec := p.Decisions[name]
		writeField([]byte(dec))
	}

	return h.Sum(nil)
}

// Hash returns a deterministic hex-encoded hash of the plan.
func (p *IncrementalPlan) Hash() string {
	bin := p.SerializeDeterministic()
	if len(bin) == 0 {
		return ""
	}
	return hex.EncodeToString(bin)
}

// BuildIncrementalPlan produces a NodeExecutionDecision for every node in graph.
//
// A node is ReuseCache IFF:
//   - it is NOT invalidated
//   - its TaskHash exists in the cache index
//   - all upstream dependencies are ReuseCache
//
// Otherwise it is Execute.
func BuildIncrementalPlan(graph *GraphSnapshot, invalidation InvalidationMap, cache core.Cache) (*IncrementalPlan, error) {
	plan := &IncrementalPlan{Decisions: make(map[string]NodeExecutionDecision)}
	if graph == nil || len(graph.Nodes) == 0 {
		return plan, nil
	}
	if cache == nil {
		return nil, fmt.Errorf("cache is nil")
	}

	// Canonical node list.
	names := make([]string, 0, len(graph.Nodes))
	for name := range graph.Nodes {
		names = append(names, name)
	}
	sort.Strings(names)

	// Build deterministic adjacency + indegrees.
	outgoing := make(map[string][]string, len(graph.Nodes))
	indeg := make(map[string]int, len(graph.Nodes))
	for _, name := range names {
		indeg[name] = 0
	}
	for _, name := range names {
		n := graph.Nodes[name]
		for _, parent := range normalizeStringSet(n.Upstream) {
			if _, exists := graph.Nodes[parent]; !exists {
				// Malformed graph snapshot; treat as structural problem by forcing execution.
				continue
			}
			outgoing[parent] = append(outgoing[parent], name)
			indeg[name]++
		}
	}
	for k := range outgoing {
		sort.Strings(outgoing[k])
	}

	order := topoOrder(names, outgoing, indeg)
	plan.Order = append([]string(nil), order...)

	for _, name := range order {
		n := graph.Nodes[name]

		inv := invalidation[name]
		if inv.Invalidated {
			plan.Decisions[name] = DecisionExecute
			continue
		}

		// Cache presence is required.
		if n.TaskHash == "" {
			plan.Decisions[name] = DecisionExecute
			continue
		}
		exists, err := cache.Has(core.TaskHash(n.TaskHash))
		if err != nil {
			return nil, fmt.Errorf("checking cache for %q: %w", name, err)
		}
		if !exists {
			plan.Decisions[name] = DecisionExecute
			continue
		}

		// All upstream dependencies must be ReuseCache.
		allUpstreamReuse := true
		for _, parent := range normalizeStringSet(n.Upstream) {
			if plan.Decisions[parent] != DecisionReuseCache {
				allUpstreamReuse = false
				break
			}
		}
		if allUpstreamReuse {
			plan.Decisions[name] = DecisionReuseCache
		} else {
			plan.Decisions[name] = DecisionExecute
		}
	}

	// Ensure every node has a decision (including any nodes not returned by topoOrder fallback).
	for _, name := range names {
		if _, ok := plan.Decisions[name]; !ok {
			plan.Decisions[name] = DecisionExecute
		}
	}

	// Ensure every node appears in Order deterministically.
	if len(plan.Order) != len(names) {
		plan.Order = append([]string(nil), names...)
		sort.Strings(plan.Order)
	}

	return plan, nil
}

// PlanIncremental computes the InvalidationMap and the IncrementalPlan for newGraph.
//
// Requirements (Sprint-04 invalidation engine):
//   - The InvalidationMap is the source of truth and must include an entry for every task in newGraph.
//   - Planning must not execute tasks.
//
// This function is a convenience integration point so callers do not need to manually stitch
// invalidation + plan building.
func PlanIncremental(oldGraph, newGraph *GraphSnapshot, cache core.Cache) (*PlanningResult, error) {
	inv := CalculateInvalidation(oldGraph, newGraph)
	plan, err := BuildIncrementalPlan(newGraph, inv, cache)
	if err != nil {
		return nil, err
	}
	return &PlanningResult{Invalidation: inv, Plan: plan}, nil
}
