package incremental

import (
	"sort"
)

// InvalidationReason enumerates the cause of task invalidation.
//
// From docs/sprints/sprint-02/in-process/incremental-engine/spec.md:
// A task must be invalidated if any of the following change:
//   - Input content
//   - Declared input set
//   - Environment variables
//   - Command string
//   - Declared outputs
//   - Upstream dependency identity or result
//
// And invalidation propagates strictly downstream.
type InvalidationReason string

const (
	ReasonNone InvalidationReason = ""

	ReasonInputChanged          InvalidationReason = "InputChanged"
	ReasonDeclaredInputsChanged InvalidationReason = "DeclaredInputsChanged"
	ReasonEnvChanged            InvalidationReason = "EnvChanged"
	ReasonCommandChanged        InvalidationReason = "CommandChanged"
	ReasonOutputsChanged        InvalidationReason = "OutputsChanged"

	ReasonGraphStructureChanged InvalidationReason = "GraphStructureChanged"
	ReasonDependencyInvalidated InvalidationReason = "DependencyInvalidated"
)

// NodeSnapshot captures the minimal identity inputs required to decide whether a node
// is unchanged or invalidated.
//
// This intentionally keeps "input content" distinct from "declared inputs".
// The incremental engine can compute InputHash from resolved file contents, while
// DeclaredInputs records the task's declared input set.
type NodeSnapshot struct {
	Name string

	// TaskHash is the deterministic execution/cache identity for the node.
	// It is used by incremental planning to check cache presence.
	TaskHash string

	// DeclaredInputs is the task's declared input set (paths/globs).
	// It is treated as a set for identity.
	DeclaredInputs []string

	// InputHash is a deterministic summary of resolved input content.
	// Any change must invalidate the node.
	InputHash string

	// Env is the task's declared environment variable map.
	Env map[string]string

	// Command is the task's command string.
	Command string

	// Outputs is the task's declared outputs.
	// It is treated as a set for identity.
	Outputs []string

	// Upstream is the list of direct dependency node names.
	// It is treated as a set for identity.
	Upstream []string
}

// GraphSnapshot represents the minimal information needed to compute an incremental invalidation plan.
//
// Nodes are addressed by stable node name.
type GraphSnapshot struct {
	Nodes map[string]NodeSnapshot
}

// InvalidationEntry is the per-node invalidation decision.
//
// If Invalidated is true, Reason is required.
type InvalidationEntry struct {
	Invalidated bool
	Reason      InvalidationReason
}

// InvalidationMap maps node name -> invalidation decision.
//
// It includes entries for every node in newGraph.
type InvalidationMap map[string]InvalidationEntry

// CalculateInvalidation computes which nodes in newGraph are invalidated relative to oldGraph.
//
// Invalidation is strictly transitive: if A is invalidated, every downstream dependent of A
// in the new graph is invalidated as well.
func CalculateInvalidation(oldGraph, newGraph *GraphSnapshot) InvalidationMap {
	result := make(InvalidationMap)
	if newGraph == nil || len(newGraph.Nodes) == 0 {
		return result
	}

	oldNodes := map[string]NodeSnapshot{}
	if oldGraph != nil && oldGraph.Nodes != nil {
		oldNodes = oldGraph.Nodes
	}

	// Canonical node name list.
	names := make([]string, 0, len(newGraph.Nodes))
	for name := range newGraph.Nodes {
		names = append(names, name)
	}
	sort.Strings(names)

	// Build deterministic adjacency + indegrees from the new graph.
	outgoing := make(map[string][]string, len(newGraph.Nodes))
	indeg := make(map[string]int, len(newGraph.Nodes))
	for _, name := range names {
		indeg[name] = 0
	}
	for _, name := range names {
		n := newGraph.Nodes[name]
		for _, parent := range normalizeStringSet(n.Upstream) {
			// A missing upstream dependency is considered a graph-structure change for this node.
			if _, exists := newGraph.Nodes[parent]; !exists {
				continue
			}
			outgoing[parent] = append(outgoing[parent], name)
			indeg[name]++
		}
	}
	for k := range outgoing {
		sort.Strings(outgoing[k])
	}

	// Deterministic topological order (lexical tie-break).
	topo := topoOrder(names, outgoing, indeg)

	// 1) Seed direct invalidations.
	for _, name := range topo {
		newNode := newGraph.Nodes[name]
		oldNode, existed := oldNodes[name]

		if !existed {
			result[name] = InvalidationEntry{Invalidated: true, Reason: ReasonGraphStructureChanged}
			continue
		}

		// Subgraph invalidation rules, in the order listed by spec.md.
		if newNode.InputHash != oldNode.InputHash {
			result[name] = InvalidationEntry{Invalidated: true, Reason: ReasonInputChanged}
			continue
		}
		if !equalStringSet(newNode.DeclaredInputs, oldNode.DeclaredInputs) {
			result[name] = InvalidationEntry{Invalidated: true, Reason: ReasonDeclaredInputsChanged}
			continue
		}
		if !equalStringMap(newNode.Env, oldNode.Env) {
			result[name] = InvalidationEntry{Invalidated: true, Reason: ReasonEnvChanged}
			continue
		}
		if newNode.Command != oldNode.Command {
			result[name] = InvalidationEntry{Invalidated: true, Reason: ReasonCommandChanged}
			continue
		}
		if !equalStringSet(newNode.Outputs, oldNode.Outputs) {
			result[name] = InvalidationEntry{Invalidated: true, Reason: ReasonOutputsChanged}
			continue
		}

		// Upstream dependency identity (direct parents) is compared as a set.
		if !equalStringSet(newNode.Upstream, oldNode.Upstream) {
			result[name] = InvalidationEntry{Invalidated: true, Reason: ReasonGraphStructureChanged}
			continue
		}

		// Default: unchanged.
		result[name] = InvalidationEntry{Invalidated: false, Reason: ReasonNone}
	}

	// 2) Propagate invalidation strictly downstream.
	for _, name := range topo {
		entry := result[name]
		if entry.Invalidated {
			continue
		}
		newNode := newGraph.Nodes[name]
		for _, parent := range normalizeStringSet(newNode.Upstream) {
			parentEntry, ok := result[parent]
			if ok && parentEntry.Invalidated {
				result[name] = InvalidationEntry{Invalidated: true, Reason: ReasonDependencyInvalidated}
				break
			}
			// If the upstream dependency is missing in newGraph, this is a structural change.
			if _, exists := newGraph.Nodes[parent]; !exists {
				result[name] = InvalidationEntry{Invalidated: true, Reason: ReasonGraphStructureChanged}
				break
			}
		}
	}

	return result
}

func normalizeStringSet(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	out = append(out, in...)
	sort.Strings(out)
	// Deduplicate.
	j := 0
	for i := 0; i < len(out); i++ {
		if i == 0 || out[i] != out[i-1] {
			out[j] = out[i]
			j++
		}
	}
	return out[:j]
}

func equalStringSet(a, b []string) bool {
	aa := normalizeStringSet(a)
	bb := normalizeStringSet(b)
	if len(aa) != len(bb) {
		return false
	}
	for i := range aa {
		if aa[i] != bb[i] {
			return false
		}
	}
	return true
}

func equalStringMap(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, av := range a {
		bv, ok := b[k]
		if !ok {
			return false
		}
		if av != bv {
			return false
		}
	}
	return true
}

func topoOrder(names []string, outgoing map[string][]string, indeg map[string]int) []string {
	// Work on a copy.
	ind := make(map[string]int, len(indeg))
	for k, v := range indeg {
		ind[k] = v
	}

	ready := make([]string, 0, len(names))
	for _, n := range names {
		if ind[n] == 0 {
			ready = append(ready, n)
		}
	}
	sort.Strings(ready)

	order := make([]string, 0, len(names))
	for len(ready) > 0 {
		n := ready[0]
		ready = ready[1:]
		order = append(order, n)

		for _, m := range outgoing[n] {
			ind[m]--
			if ind[m] == 0 {
				// Insert m into ready keeping it sorted.
				idx := sort.SearchStrings(ready, m)
				ready = append(ready, "")
				copy(ready[idx+1:], ready[idx:])
				ready[idx] = m
			}
		}
	}

	// If we couldn't order everything (cycle or malformed upstream), fall back to lexical.
	if len(order) != len(names) {
		fallback := make([]string, len(names))
		copy(fallback, names)
		sort.Strings(fallback)
		return fallback
	}
	return order
}
