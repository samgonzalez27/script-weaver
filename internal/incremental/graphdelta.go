package incremental

import "sort"

// GraphDelta represents the difference between two graph executions.
//
// From docs/sprints/sprint-02/planning/data-dictionary.md.
type GraphDelta struct {
	AddedNodes    []string
	RemovedNodes  []string
	ModifiedNodes []string
}

// CalculateGraphDelta computes a deterministic delta between oldGraph and newGraph.
//
// Nodes are identified by name. A node is considered modified if it exists in both graphs
// but its NodeSnapshot differs.
func CalculateGraphDelta(oldGraph, newGraph *GraphSnapshot) GraphDelta {
	var delta GraphDelta

	oldNodes := map[string]NodeSnapshot{}
	if oldGraph != nil && oldGraph.Nodes != nil {
		oldNodes = oldGraph.Nodes
	}
	newNodes := map[string]NodeSnapshot{}
	if newGraph != nil && newGraph.Nodes != nil {
		newNodes = newGraph.Nodes
	}

	// Added/modified
	for name, nn := range newNodes {
		on, ok := oldNodes[name]
		if !ok {
			delta.AddedNodes = append(delta.AddedNodes, name)
			continue
		}
		if !equalNodeSnapshot(on, nn) {
			delta.ModifiedNodes = append(delta.ModifiedNodes, name)
		}
	}

	// Removed
	for name := range oldNodes {
		if _, ok := newNodes[name]; !ok {
			delta.RemovedNodes = append(delta.RemovedNodes, name)
		}
	}

	sort.Strings(delta.AddedNodes)
	sort.Strings(delta.RemovedNodes)
	sort.Strings(delta.ModifiedNodes)

	return delta
}

func equalNodeSnapshot(a, b NodeSnapshot) bool {
	if a.Name != b.Name {
		return false
	}
	if a.TaskHash != b.TaskHash {
		return false
	}
	if a.InputHash != b.InputHash {
		return false
	}
	if a.Command != b.Command {
		return false
	}
	if !equalStringSet(a.DeclaredInputs, b.DeclaredInputs) {
		return false
	}
	if !equalStringSet(a.Outputs, b.Outputs) {
		return false
	}
	if !equalStringSet(a.Upstream, b.Upstream) {
		return false
	}
	if !equalStringMap(a.Env, b.Env) {
		return false
	}
	return true
}
