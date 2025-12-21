package graph

import (
	"errors"
	"testing"
)

func TestValidate_ValidEmptyGraph(t *testing.T) {
	g := &Graph{
		Nodes: []Node{},
		Edges: []Edge{},
	}
	if err := Validate(g); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidate_ValidDAG(t *testing.T) {
	g := &Graph{
		Nodes: []Node{
			{ID: "a", Type: "t", Inputs: map[string]any{}, Outputs: []string{}},
			{ID: "b", Type: "t", Inputs: map[string]any{}, Outputs: []string{}},
			{ID: "c", Type: "t", Inputs: map[string]any{}, Outputs: []string{}},
		},
		Edges: []Edge{
			{From: "a", To: "b"},
			{From: "b", To: "c"},
		},
	}
	if err := Validate(g); err != nil {
		t.Fatalf("expected no error for valid DAG, got %v", err)
	}
}

func TestValidate_DuplicateNodeIDs(t *testing.T) {
	g := &Graph{
		Nodes: []Node{
			{ID: "node1", Type: "t", Inputs: map[string]any{}, Outputs: []string{}},
			{ID: "node1", Type: "t", Inputs: map[string]any{}, Outputs: []string{}},
		},
		Edges: []Edge{},
	}
	err := Validate(g)
	if err == nil {
		t.Fatal("expected error for duplicate node IDs")
	}
	if !errors.Is(err, ErrStructural) {
		t.Errorf("expected StructuralError, got %T: %v", err, err)
	}
	se, ok := err.(*StructuralError)
	if !ok {
		t.Fatalf("expected *StructuralError, got %T", err)
	}
	if se.Kind != "duplicate_id" {
		t.Errorf("expected Kind 'duplicate_id', got %q", se.Kind)
	}
}

func TestValidate_DanglingEdgeFromUnknown(t *testing.T) {
	g := &Graph{
		Nodes: []Node{
			{ID: "a", Type: "t", Inputs: map[string]any{}, Outputs: []string{}},
		},
		Edges: []Edge{
			{From: "unknown", To: "a"},
		},
	}
	err := Validate(g)
	if err == nil {
		t.Fatal("expected error for dangling edge")
	}
	if !errors.Is(err, ErrStructural) {
		t.Errorf("expected StructuralError, got %T: %v", err, err)
	}
	se, ok := err.(*StructuralError)
	if !ok {
		t.Fatalf("expected *StructuralError, got %T", err)
	}
	if se.Kind != "dangling_edge" {
		t.Errorf("expected Kind 'dangling_edge', got %q", se.Kind)
	}
}

func TestValidate_DanglingEdgeToUnknown(t *testing.T) {
	g := &Graph{
		Nodes: []Node{
			{ID: "a", Type: "t", Inputs: map[string]any{}, Outputs: []string{}},
		},
		Edges: []Edge{
			{From: "a", To: "unknown"},
		},
	}
	err := Validate(g)
	if err == nil {
		t.Fatal("expected error for dangling edge")
	}
	if !errors.Is(err, ErrStructural) {
		t.Errorf("expected StructuralError, got %T: %v", err, err)
	}
	se, ok := err.(*StructuralError)
	if !ok {
		t.Fatalf("expected *StructuralError, got %T", err)
	}
	if se.Kind != "dangling_edge" {
		t.Errorf("expected Kind 'dangling_edge', got %q", se.Kind)
	}
}

func TestValidate_SelfReferentialEdge(t *testing.T) {
	g := &Graph{
		Nodes: []Node{
			{ID: "a", Type: "t", Inputs: map[string]any{}, Outputs: []string{}},
		},
		Edges: []Edge{
			{From: "a", To: "a"},
		},
	}
	err := Validate(g)
	if err == nil {
		t.Fatal("expected error for self-referential edge")
	}
	if !errors.Is(err, ErrStructural) {
		t.Errorf("expected StructuralError, got %T: %v", err, err)
	}
	se, ok := err.(*StructuralError)
	if !ok {
		t.Fatalf("expected *StructuralError, got %T", err)
	}
	if se.Kind != "self_reference" {
		t.Errorf("expected Kind 'self_reference', got %q", se.Kind)
	}
}

func TestValidate_SimpleCycle(t *testing.T) {
	g := &Graph{
		Nodes: []Node{
			{ID: "a", Type: "t", Inputs: map[string]any{}, Outputs: []string{}},
			{ID: "b", Type: "t", Inputs: map[string]any{}, Outputs: []string{}},
		},
		Edges: []Edge{
			{From: "a", To: "b"},
			{From: "b", To: "a"},
		},
	}
	err := Validate(g)
	if err == nil {
		t.Fatal("expected error for cyclic graph")
	}
	if !errors.Is(err, ErrStructural) {
		t.Errorf("expected StructuralError, got %T: %v", err, err)
	}
	se, ok := err.(*StructuralError)
	if !ok {
		t.Fatalf("expected *StructuralError, got %T", err)
	}
	if se.Kind != "cycle" {
		t.Errorf("expected Kind 'cycle', got %q", se.Kind)
	}
}

func TestValidate_LongerCycle(t *testing.T) {
	g := &Graph{
		Nodes: []Node{
			{ID: "a", Type: "t", Inputs: map[string]any{}, Outputs: []string{}},
			{ID: "b", Type: "t", Inputs: map[string]any{}, Outputs: []string{}},
			{ID: "c", Type: "t", Inputs: map[string]any{}, Outputs: []string{}},
		},
		Edges: []Edge{
			{From: "a", To: "b"},
			{From: "b", To: "c"},
			{From: "c", To: "a"},
		},
	}
	err := Validate(g)
	if err == nil {
		t.Fatal("expected error for cyclic graph")
	}
	if !errors.Is(err, ErrStructural) {
		t.Errorf("expected StructuralError, got %T: %v", err, err)
	}
	se, ok := err.(*StructuralError)
	if !ok {
		t.Fatalf("expected *StructuralError, got %T", err)
	}
	if se.Kind != "cycle" {
		t.Errorf("expected Kind 'cycle', got %q", se.Kind)
	}
}

func TestValidate_DisconnectedComponents(t *testing.T) {
	// Two separate valid subgraphs - should pass
	g := &Graph{
		Nodes: []Node{
			{ID: "a", Type: "t", Inputs: map[string]any{}, Outputs: []string{}},
			{ID: "b", Type: "t", Inputs: map[string]any{}, Outputs: []string{}},
			{ID: "x", Type: "t", Inputs: map[string]any{}, Outputs: []string{}},
			{ID: "y", Type: "t", Inputs: map[string]any{}, Outputs: []string{}},
		},
		Edges: []Edge{
			{From: "a", To: "b"},
			{From: "x", To: "y"},
		},
	}
	if err := Validate(g); err != nil {
		t.Fatalf("expected no error for disconnected DAG, got %v", err)
	}
}

func TestValidate_DiamondDAG(t *testing.T) {
	// Diamond shape: a -> b, a -> c, b -> d, c -> d
	g := &Graph{
		Nodes: []Node{
			{ID: "a", Type: "t", Inputs: map[string]any{}, Outputs: []string{}},
			{ID: "b", Type: "t", Inputs: map[string]any{}, Outputs: []string{}},
			{ID: "c", Type: "t", Inputs: map[string]any{}, Outputs: []string{}},
			{ID: "d", Type: "t", Inputs: map[string]any{}, Outputs: []string{}},
		},
		Edges: []Edge{
			{From: "a", To: "b"},
			{From: "a", To: "c"},
			{From: "b", To: "d"},
			{From: "c", To: "d"},
		},
	}
	if err := Validate(g); err != nil {
		t.Fatalf("expected no error for diamond DAG, got %v", err)
	}
}

func TestValidate_DeterministicErrorOrder(t *testing.T) {
	// With multiple duplicates, should always report the first one lexicographically
	g := &Graph{
		Nodes: []Node{
			{ID: "z", Type: "t", Inputs: map[string]any{}, Outputs: []string{}},
			{ID: "a", Type: "t", Inputs: map[string]any{}, Outputs: []string{}},
			{ID: "a", Type: "t", Inputs: map[string]any{}, Outputs: []string{}},
			{ID: "z", Type: "t", Inputs: map[string]any{}, Outputs: []string{}},
		},
		Edges: []Edge{},
	}
	err := Validate(g)
	if err == nil {
		t.Fatal("expected error for duplicate node IDs")
	}
	se := err.(*StructuralError)
	// Should report 'a' first since nodes are sorted
	expected := `duplicate node ID: "a"`
	if se.Msg != expected {
		t.Errorf("expected deterministic error %q, got %q", expected, se.Msg)
	}
}
