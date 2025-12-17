package incremental

import (
	"testing"
)

func TestIncrementalPlan_HashIsDeterministic(t *testing.T) {
	// Same logical plan, but decisions inserted in different map orders.
	p1 := &IncrementalPlan{
		Order: []string{"A", "B", "C"},
		Decisions: map[string]NodeExecutionDecision{
			"A": DecisionReuseCache,
			"B": DecisionExecute,
			"C": DecisionReuseCache,
		},
	}

	p2 := &IncrementalPlan{
		Order: []string{"A", "B", "C"},
		Decisions: map[string]NodeExecutionDecision{
			"C": DecisionReuseCache,
			"A": DecisionReuseCache,
			"B": DecisionExecute,
		},
	}

	h1 := p1.Hash()
	h2 := p2.Hash()
	if h1 == "" || h2 == "" {
		t.Fatalf("expected non-empty plan hashes")
	}
	if h1 != h2 {
		t.Fatalf("expected deterministic plan hash, got %q != %q", h1, h2)
	}
}
