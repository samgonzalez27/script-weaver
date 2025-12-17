package incremental

import (
	"bytes"
	"testing"
)

func TestCalculateInvalidation_SingleTaskInputChanged(t *testing.T) {
	oldGraph := &GraphSnapshot{Nodes: map[string]NodeSnapshot{
		"A": {Name: "A", DeclaredInputs: []string{"a.txt"}, InputHash: "old", Env: map[string]string{"K": "V"}, Command: "echo A", Outputs: []string{"a.out"}},
	}}
	newGraph := &GraphSnapshot{Nodes: map[string]NodeSnapshot{
		"A": {Name: "A", DeclaredInputs: []string{"a.txt"}, InputHash: "new", Env: map[string]string{"K": "V"}, Command: "echo A", Outputs: []string{"a.out"}},
	}}

	inv := CalculateInvalidation(oldGraph, newGraph)
	a := inv["A"]
	if !a.Invalidated {
		t.Fatalf("expected A invalidated")
	}
	if len(a.Reasons) != 1 {
		t.Fatalf("expected 1 reason, got %d", len(a.Reasons))
	}
	if a.Reasons[0].Type != ReasonTypeInputChanged {
		t.Fatalf("expected reason %q, got %q", ReasonTypeInputChanged, a.Reasons[0].Type)
	}
}

func TestCalculateInvalidation_CascadingDependencyChain_RootCausePropagates(t *testing.T) {
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
	if len(a.Reasons) != 1 || a.Reasons[0].Type != ReasonTypeInputChanged {
		t.Fatalf("expected A reasons [InputChanged], got %#v", a.Reasons)
	}

	b := inv["B"]
	if !b.Invalidated {
		t.Fatalf("expected B invalidated")
	}
	if len(b.Reasons) != 1 || b.Reasons[0].Type != ReasonTypeDependencyInvalidated || b.Reasons[0].SourceTaskID != "A" {
		t.Fatalf("expected B reasons [DependencyInvalidated(A)], got %#v", b.Reasons)
	}

	c := inv["C"]
	if !c.Invalidated {
		t.Fatalf("expected C invalidated")
	}
	if len(c.Reasons) != 1 || c.Reasons[0].Type != ReasonTypeDependencyInvalidated || c.Reasons[0].SourceTaskID != "A" {
		t.Fatalf("expected C reasons [DependencyInvalidated(A)], got %#v", c.Reasons)
	}
}

func TestCalculateInvalidation_MultipleDependencyInvalidation_SortedBySourceTaskID(t *testing.T) {
	oldGraph := &GraphSnapshot{Nodes: map[string]NodeSnapshot{
		"A": {Name: "A", DeclaredInputs: []string{"a.txt"}, InputHash: "oldA"},
		"B": {Name: "B", DeclaredInputs: []string{"b.txt"}, InputHash: "oldB"},
		"C": {Name: "C", DeclaredInputs: []string{"c.txt"}, InputHash: "oldC", Upstream: []string{"A", "B"}},
	}}
	newGraph := &GraphSnapshot{Nodes: map[string]NodeSnapshot{
		"A": {Name: "A", DeclaredInputs: []string{"a.txt"}, InputHash: "newA"},
		"B": {Name: "B", DeclaredInputs: []string{"b.txt"}, InputHash: "newB"},
		"C": {Name: "C", DeclaredInputs: []string{"c.txt"}, InputHash: "oldC", Upstream: []string{"B", "A"}},
	}}

	inv := CalculateInvalidation(oldGraph, newGraph)
	c := inv["C"]
	if !c.Invalidated {
		t.Fatalf("expected C invalidated")
	}
	if len(c.Reasons) != 2 {
		t.Fatalf("expected 2 reasons, got %#v", c.Reasons)
	}
	if c.Reasons[0].Type != ReasonTypeDependencyInvalidated || c.Reasons[0].SourceTaskID != "A" {
		t.Fatalf("expected first reason DependencyInvalidated(A), got %#v", c.Reasons[0])
	}
	if c.Reasons[1].Type != ReasonTypeDependencyInvalidated || c.Reasons[1].SourceTaskID != "B" {
		t.Fatalf("expected second reason DependencyInvalidated(B), got %#v", c.Reasons[1])
	}
}

func TestCalculateInvalidation_MixedReasons_DeterministicOrder(t *testing.T) {
	oldGraph := &GraphSnapshot{Nodes: map[string]NodeSnapshot{
		"A": {Name: "A", DeclaredInputs: []string{"a.txt"}, InputHash: "old", Env: map[string]string{"K": "V"}},
	}}
	newGraph := &GraphSnapshot{Nodes: map[string]NodeSnapshot{
		"A": {Name: "A", DeclaredInputs: []string{"a.txt"}, InputHash: "new", Env: map[string]string{"K": "CHANGED"}},
	}}

	inv := CalculateInvalidation(oldGraph, newGraph)
	a := inv["A"]
	if !a.Invalidated {
		t.Fatalf("expected A invalidated")
	}
	if len(a.Reasons) != 2 {
		t.Fatalf("expected 2 reasons, got %#v", a.Reasons)
	}
	if a.Reasons[0].Type != ReasonTypeInputChanged {
		t.Fatalf("expected reasons[0] InputChanged, got %#v", a.Reasons[0])
	}
	if a.Reasons[1].Type != ReasonTypeEnvChanged {
		t.Fatalf("expected reasons[1] EnvChanged, got %#v", a.Reasons[1])
	}
}

func TestCalculateInvalidation_CascadingChain_WithIndependentMidFailure_ReferencesRootCauses(t *testing.T) {
	// A -> B -> C.
	// A is directly invalidated.
	// B is invalidated both because A is invalidated AND because B's own input changes.
	// C must reference root causes A and B (not just "B failed").
	oldGraph := &GraphSnapshot{Nodes: map[string]NodeSnapshot{
		"A": {Name: "A", DeclaredInputs: []string{"a.txt"}, InputHash: "oldA"},
		"B": {Name: "B", DeclaredInputs: []string{"b.txt"}, InputHash: "oldB", Upstream: []string{"A"}},
		"C": {Name: "C", DeclaredInputs: []string{"c.txt"}, InputHash: "oldC", Upstream: []string{"B"}},
	}}
	newGraph := &GraphSnapshot{Nodes: map[string]NodeSnapshot{
		"A": {Name: "A", DeclaredInputs: []string{"a.txt"}, InputHash: "newA"}, // direct invalidation
		"B": {Name: "B", DeclaredInputs: []string{"b.txt"}, InputHash: "newB", Upstream: []string{"A"}}, // independent direct invalidation
		"C": {Name: "C", DeclaredInputs: []string{"c.txt"}, InputHash: "oldC", Upstream: []string{"B"}},
	}}

	inv := CalculateInvalidation(oldGraph, newGraph)

	a := inv["A"]
	if !a.Invalidated || len(a.Reasons) != 1 || a.Reasons[0].Type != ReasonTypeInputChanged {
		t.Fatalf("expected A invalidated with [InputChanged], got %#v", a)
	}

	b := inv["B"]
	if !b.Invalidated {
		t.Fatalf("expected B invalidated")
	}
	// B has at least InputChanged directly; it may also include DependencyInvalidated(A).
	if len(b.Reasons) == 0 {
		t.Fatalf("expected B to have reasons")
	}

	c := inv["C"]
	if !c.Invalidated {
		t.Fatalf("expected C invalidated")
	}
	// C must reference both root causes: A (original upstream) and B (because B has an independent direct invalidation).
	if len(c.Reasons) != 2 {
		t.Fatalf("expected C reasons [DependencyInvalidated(A), DependencyInvalidated(B)], got %#v", c.Reasons)
	}
	if c.Reasons[0].Type != ReasonTypeDependencyInvalidated || c.Reasons[0].SourceTaskID != "A" {
		t.Fatalf("expected C reasons[0] DependencyInvalidated(A), got %#v", c.Reasons[0])
	}
	if c.Reasons[1].Type != ReasonTypeDependencyInvalidated || c.Reasons[1].SourceTaskID != "B" {
		t.Fatalf("expected C reasons[1] DependencyInvalidated(B), got %#v", c.Reasons[1])
	}
}

func TestCalculateInvalidation_Aggregation_LocalAndMultipleDependencyReasons_DeterministicOrder(t *testing.T) {
	// C depends on A and B, and C also has a local invalidation.
	// Expected deterministic reason order: local reasons first by type order, then dependency reasons sorted by SourceTaskID.
	oldGraph := &GraphSnapshot{Nodes: map[string]NodeSnapshot{
		"A": {Name: "A", DeclaredInputs: []string{"a.txt"}, InputHash: "oldA"},
		"B": {Name: "B", DeclaredInputs: []string{"b.txt"}, InputHash: "oldB"},
		"C": {Name: "C", DeclaredInputs: []string{"c.txt"}, InputHash: "oldC", Upstream: []string{"A", "B"}},
	}}
	newGraph := &GraphSnapshot{Nodes: map[string]NodeSnapshot{
		"A": {Name: "A", DeclaredInputs: []string{"a.txt"}, InputHash: "newA"},
		"B": {Name: "B", DeclaredInputs: []string{"b.txt"}, InputHash: "newB"},
		// C has local InputChanged, and upstream order is intentionally reversed to ensure ordering is not derived from creation order.
		"C": {Name: "C", DeclaredInputs: []string{"c.txt"}, InputHash: "newC", Upstream: []string{"B", "A"}},
	}}

	inv := CalculateInvalidation(oldGraph, newGraph)
	c := inv["C"]
	if !c.Invalidated {
		t.Fatalf("expected C invalidated")
	}
	if len(c.Reasons) != 3 {
		t.Fatalf("expected 3 reasons, got %#v", c.Reasons)
	}
	if c.Reasons[0].Type != ReasonTypeInputChanged {
		t.Fatalf("expected reasons[0] InputChanged, got %#v", c.Reasons[0])
	}
	if c.Reasons[1].Type != ReasonTypeDependencyInvalidated || c.Reasons[1].SourceTaskID != "A" {
		t.Fatalf("expected reasons[1] DependencyInvalidated(A), got %#v", c.Reasons[1])
	}
	if c.Reasons[2].Type != ReasonTypeDependencyInvalidated || c.Reasons[2].SourceTaskID != "B" {
		t.Fatalf("expected reasons[2] DependencyInvalidated(B), got %#v", c.Reasons[2])
	}
}

func TestInvalidationReason_DeterministicSerialization_IgnoresCreationOrder(t *testing.T) {
	r1 := InvalidationReason{
		Type: ReasonTypeGraphStructureChanged,
		Details: []InvalidationDetail{
			{Key: "Upstream", Value: "missing"},
			{Key: "DeclaredInputs", Value: "changed"},
		},
	}
	r2 := InvalidationReason{
		Type: ReasonTypeGraphStructureChanged,
		Details: []InvalidationDetail{
			{Key: "DeclaredInputs", Value: "changed"},
			{Key: "Upstream", Value: "missing"},
		},
	}

	b1, err := r1.MarshalBinary()
	if err != nil {
		t.Fatalf("marshal r1: %v", err)
	}
	b2, err := r2.MarshalBinary()
	if err != nil {
		t.Fatalf("marshal r2: %v", err)
	}
	if !bytes.Equal(b1, b2) {
		t.Fatalf("expected identical bytes for same logical reason")
	}

	rs1 := InvalidationReasons{r1, InvalidationReason{Type: ReasonTypeEnvChanged}}
	rs2 := InvalidationReasons{InvalidationReason{Type: ReasonTypeEnvChanged}, r2}

	s1, err := rs1.MarshalBinary()
	if err != nil {
		t.Fatalf("marshal rs1: %v", err)
	}
	s2, err := rs2.MarshalBinary()
	if err != nil {
		t.Fatalf("marshal rs2: %v", err)
	}
	if !bytes.Equal(s1, s2) {
		t.Fatalf("expected identical bytes for same logical reason set")
	}
}

func TestInvalidationMap_DeterministicSerialization_IgnoresMapOrder(t *testing.T) {
	m1 := InvalidationMap{}
	m1["B"] = InvalidationEntry{Invalidated: true, Reasons: InvalidationReasons{InvalidationReason{Type: ReasonTypeEnvChanged}}}
	m1["A"] = InvalidationEntry{Invalidated: true, Reasons: InvalidationReasons{InvalidationReason{Type: ReasonTypeInputChanged}}}

	m2 := InvalidationMap{}
	m2["A"] = InvalidationEntry{Invalidated: true, Reasons: InvalidationReasons{InvalidationReason{Type: ReasonTypeInputChanged}}}
	m2["B"] = InvalidationEntry{Invalidated: true, Reasons: InvalidationReasons{InvalidationReason{Type: ReasonTypeEnvChanged}}}

	b1, err := m1.MarshalBinary()
	if err != nil {
		t.Fatalf("marshal m1: %v", err)
	}
	b2, err := m2.MarshalBinary()
	if err != nil {
		t.Fatalf("marshal m2: %v", err)
	}
	if !bytes.Equal(b1, b2) {
		t.Fatalf("expected identical bytes for maps with same content")
	}
}
