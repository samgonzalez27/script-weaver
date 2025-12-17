package incremental

import "testing"

func TestCalculateInvalidation_CascadingDependencyChain(t *testing.T) {
	oldGraph := &GraphSnapshot{
		Nodes: map[string]NodeSnapshot{
			"A": {
				Name:           "A",
				DeclaredInputs: []string{"a.txt"},
				InputHash:      "old-input-hash-A",
				Env:            map[string]string{"K": "V"},
				Command:        "echo A",
				Outputs:        []string{"a.out"},
				Upstream:       nil,
			},
			"B": {
				Name:           "B",
				DeclaredInputs: []string{"b.txt"},
				InputHash:      "input-hash-B",
				Env:            map[string]string{"K": "V"},
				Command:        "echo B",
				Outputs:        []string{"b.out"},
				Upstream:       []string{"A"},
			},
			"C": {
				Name:           "C",
				DeclaredInputs: []string{"c.txt"},
				InputHash:      "input-hash-C",
				Env:            map[string]string{"K": "V"},
				Command:        "echo C",
				Outputs:        []string{"c.out"},
				Upstream:       []string{"B"},
			},
		},
	}

	newGraph := &GraphSnapshot{
		Nodes: map[string]NodeSnapshot{
			"A": {
				Name:           "A",
				DeclaredInputs: []string{"a.txt"},
				InputHash:      "new-input-hash-A", // Simulate input content change.
				Env:            map[string]string{"K": "V"},
				Command:        "echo A",
				Outputs:        []string{"a.out"},
				Upstream:       nil,
			},
			"B": {
				Name:           "B",
				DeclaredInputs: []string{"b.txt"},
				InputHash:      "input-hash-B",
				Env:            map[string]string{"K": "V"},
				Command:        "echo B",
				Outputs:        []string{"b.out"},
				Upstream:       []string{"A"},
			},
			"C": {
				Name:           "C",
				DeclaredInputs: []string{"c.txt"},
				InputHash:      "input-hash-C",
				Env:            map[string]string{"K": "V"},
				Command:        "echo C",
				Outputs:        []string{"c.out"},
				Upstream:       []string{"B"},
			},
		},
	}

	inv := CalculateInvalidation(oldGraph, newGraph)

	a := inv["A"]
	if !a.Invalidated {
		t.Fatalf("expected A invalidated")
	}
	if a.Reason != ReasonInputChanged {
		t.Fatalf("expected A reason %q, got %q", ReasonInputChanged, a.Reason)
	}

	b := inv["B"]
	if !b.Invalidated {
		t.Fatalf("expected B invalidated")
	}
	if b.Reason != ReasonDependencyInvalidated {
		t.Fatalf("expected B reason %q, got %q", ReasonDependencyInvalidated, b.Reason)
	}

	c := inv["C"]
	if !c.Invalidated {
		t.Fatalf("expected C invalidated")
	}
	if c.Reason != ReasonDependencyInvalidated {
		t.Fatalf("expected C reason %q, got %q", ReasonDependencyInvalidated, c.Reason)
	}
}
