package dag

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"scriptweaver/internal/core"
	"scriptweaver/internal/incremental"
)

// Mixed Cached and Executed Runs:
// Upstream A is ReuseCache (restored), downstream B is Execute and consumes A's artifact.
func TestExecutorSerial_IncrementalPlan_MixedCachedAndExecuted_RestorationFeedsDownstream(t *testing.T) {
	workDir := t.TempDir()

	cache := core.NewMemoryCache()
	coreRunner := core.NewRunner(workDir, cache)
	cacheRunner, err := NewCacheAwareRunner(coreRunner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	g, err := NewTaskGraph(
		[]core.Task{
			{
				Name:    "A",
				Run:     "printf 'A1' > a.txt",
				Outputs: []string{"a.txt"},
			},
			{
				Name:    "B",
				Inputs:  []string{"a.txt"},
				Run:     `IFS= read -r x < a.txt; printf '%sB' "$x" > b.txt`,
				Outputs: []string{"b.txt"},
			},
		},
		[]Edge{{From: "A", To: "B"}},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Run 1: populate cache (both execute).
	exec1, err := NewExecutor(g, cacheRunner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	res1, err := exec1.RunSerial(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res1.FinalState["A"] != TaskCompleted || res1.FinalState["B"] != TaskCompleted {
		t.Fatalf("expected completed on first run, got: %v", res1.FinalState)
	}

	// Delete artifacts to force restoration + execution to prove correctness.
	if err := os.Remove(filepath.Join(workDir, "a.txt")); err != nil {
		t.Fatalf("removing a.txt: %v", err)
	}
	if err := os.Remove(filepath.Join(workDir, "b.txt")); err != nil {
		t.Fatalf("removing b.txt: %v", err)
	}

	// Plan: A reused from cache, B executed.
	plan := &incremental.IncrementalPlan{
		Order: []string{"A", "B"},
		Decisions: map[string]incremental.NodeExecutionDecision{
			"A": incremental.DecisionReuseCache,
			"B": incremental.DecisionExecute,
		},
	}

	exec2, err := NewExecutor(g, cacheRunner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	exec2.Plan = plan

	res2, err := exec2.RunSerial(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res2.FinalState["A"] != TaskCompleted {
		t.Fatalf("expected A completed (restored), got %s", res2.FinalState["A"])
	}
	if res2.FinalState["B"] != TaskCompleted {
		t.Fatalf("expected B completed (executed), got %s", res2.FinalState["B"])
	}

	// Verify B could consume A's restored artifact.
	b, err := os.ReadFile(filepath.Join(workDir, "b.txt"))
	if err != nil {
		t.Fatalf("reading b.txt: %v", err)
	}
	if string(b) != "A1B" {
		t.Fatalf("unexpected B output: %q", b)
	}
}
