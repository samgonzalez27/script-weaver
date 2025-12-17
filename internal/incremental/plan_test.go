package incremental

import (
	"testing"

	"scriptweaver/internal/core"
)

func TestBuildIncrementalPlan_IncrementalNoOpGraph_AllReuseCache(t *testing.T) {
	oldGraph := &GraphSnapshot{
		Nodes: map[string]NodeSnapshot{
			"A": {
				Name:           "A",
				TaskHash:       "hash-A",
				DeclaredInputs: []string{"a.txt"},
				InputHash:      "input-A",
				Env:            map[string]string{"K": "V"},
				Command:        "echo A",
				Outputs:        []string{"a.out"},
				Upstream:       nil,
			},
			"B": {
				Name:           "B",
				TaskHash:       "hash-B",
				DeclaredInputs: []string{"b.txt"},
				InputHash:      "input-B",
				Env:            map[string]string{"K": "V"},
				Command:        "echo B",
				Outputs:        []string{"b.out"},
				Upstream:       []string{"A"},
			},
			"C": {
				Name:           "C",
				TaskHash:       "hash-C",
				DeclaredInputs: []string{"c.txt"},
				InputHash:      "input-C",
				Env:            map[string]string{"K": "V"},
				Command:        "echo C",
				Outputs:        []string{"c.out"},
				Upstream:       []string{"B"},
			},
		},
	}

	// Unchanged graph: newGraph identical to oldGraph.
	newGraph := &GraphSnapshot{Nodes: map[string]NodeSnapshot{}}
	for k, v := range oldGraph.Nodes {
		newGraph.Nodes[k] = v
	}

	inv := CalculateInvalidation(oldGraph, newGraph)

	cache := core.NewMemoryCache()
	for _, n := range newGraph.Nodes {
		if err := cache.Put(&core.CacheEntry{Hash: core.TaskHash(n.TaskHash)}); err != nil {
			t.Fatalf("failed to seed cache for %q: %v", n.Name, err)
		}
	}

	plan, err := BuildIncrementalPlan(newGraph, inv, cache)
	if err != nil {
		t.Fatalf("BuildIncrementalPlan failed: %v", err)
	}
	if len(plan.Order) != len(newGraph.Nodes) {
		t.Fatalf("expected plan.Order to include all nodes")
	}

	for name := range newGraph.Nodes {
		if plan.Decisions[name] != DecisionReuseCache {
			t.Fatalf("expected %q decision %q, got %q", name, DecisionReuseCache, plan.Decisions[name])
		}
	}
}
