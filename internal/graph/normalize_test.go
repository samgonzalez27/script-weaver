package graph

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestNormalize_SortsNodesById(t *testing.T) {
	g := &Graph{
		Nodes: []Node{
			{ID: "z", Type: "t", Inputs: map[string]any{}, Outputs: []string{}},
			{ID: "a", Type: "t", Inputs: map[string]any{}, Outputs: []string{}},
			{ID: "m", Type: "t", Inputs: map[string]any{}, Outputs: []string{}},
		},
		Edges: []Edge{},
	}

	g.Normalize()

	expected := []string{"a", "m", "z"}
	for i, id := range expected {
		if g.Nodes[i].ID != id {
			t.Errorf("expected node %d to have id %q, got %q", i, id, g.Nodes[i].ID)
		}
	}
}

func TestNormalize_SortsEdgesByFromThenTo(t *testing.T) {
	g := &Graph{
		Nodes: []Node{
			{ID: "a", Type: "t", Inputs: map[string]any{}, Outputs: []string{}},
			{ID: "b", Type: "t", Inputs: map[string]any{}, Outputs: []string{}},
			{ID: "c", Type: "t", Inputs: map[string]any{}, Outputs: []string{}},
		},
		Edges: []Edge{
			{From: "b", To: "c"},
			{From: "a", To: "c"},
			{From: "a", To: "b"},
		},
	}

	g.Normalize()

	expected := []Edge{
		{From: "a", To: "b"},
		{From: "a", To: "c"},
		{From: "b", To: "c"},
	}
	for i, e := range expected {
		if g.Edges[i] != e {
			t.Errorf("expected edge %d to be %v, got %v", i, e, g.Edges[i])
		}
	}
}

func TestNormalize_SortsOutputs(t *testing.T) {
	g := &Graph{
		Nodes: []Node{
			{ID: "a", Type: "t", Inputs: map[string]any{}, Outputs: []string{"z", "a", "m"}},
		},
		Edges: []Edge{},
	}

	g.Normalize()

	expected := []string{"a", "m", "z"}
	for i, out := range expected {
		if g.Nodes[0].Outputs[i] != out {
			t.Errorf("expected output %d to be %q, got %q", i, out, g.Nodes[0].Outputs[i])
		}
	}
}

func TestNormalize_DifferentOrdersSameResult(t *testing.T) {
	// Graph 1: nodes in order z, a, m
	g1 := &Graph{
		Nodes: []Node{
			{ID: "z", Type: "exec", Inputs: map[string]any{"cmd": "echo"}, Outputs: []string{"stderr", "stdout"}},
			{ID: "a", Type: "exec", Inputs: map[string]any{"cmd": "ls"}, Outputs: []string{"out"}},
			{ID: "m", Type: "exec", Inputs: map[string]any{"cmd": "cat"}, Outputs: []string{}},
		},
		Edges: []Edge{
			{From: "z", To: "a"},
			{From: "a", To: "m"},
		},
	}

	// Graph 2: same nodes in different order (a, m, z), edges reversed
	g2 := &Graph{
		Nodes: []Node{
			{ID: "a", Type: "exec", Inputs: map[string]any{"cmd": "ls"}, Outputs: []string{"out"}},
			{ID: "m", Type: "exec", Inputs: map[string]any{"cmd": "cat"}, Outputs: []string{}},
			{ID: "z", Type: "exec", Inputs: map[string]any{"cmd": "echo"}, Outputs: []string{"stdout", "stderr"}},
		},
		Edges: []Edge{
			{From: "a", To: "m"},
			{From: "z", To: "a"},
		},
	}

	g1.Normalize()
	g2.Normalize()

	// Marshal both to JSON
	b1, err := json.Marshal(g1)
	if err != nil {
		t.Fatalf("failed to marshal g1: %v", err)
	}
	b2, err := json.Marshal(g2)
	if err != nil {
		t.Fatalf("failed to marshal g2: %v", err)
	}

	if !bytes.Equal(b1, b2) {
		t.Errorf("normalized graphs are not byte-identical:\ng1: %s\ng2: %s", b1, b2)
	}
}

func TestNormalize_ParsedGraphsIdentical(t *testing.T) {
	// Two JSON documents with same content, different ordering
	json1 := `{
		"schema_version": "1.0.0",
		"graph": {
			"nodes": [
				{"id": "b", "type": "t", "inputs": {"y": 2, "x": 1}, "outputs": ["out2", "out1"]},
				{"id": "a", "type": "t", "inputs": {"a": 1}, "outputs": []}
			],
			"edges": [{"from": "b", "to": "a"}]
		},
		"metadata": {}
	}`

	json2 := `{
		"schema_version": "1.0.0",
		"graph": {
			"nodes": [
				{"id": "a", "type": "t", "inputs": {"a": 1}, "outputs": []},
				{"id": "b", "type": "t", "inputs": {"x": 1, "y": 2}, "outputs": ["out1", "out2"]}
			],
			"edges": [{"from": "b", "to": "a"}]
		},
		"metadata": {}
	}`

	doc1, err := Parse(strings.NewReader(json1))
	if err != nil {
		t.Fatalf("failed to parse json1: %v", err)
	}
	doc2, err := Parse(strings.NewReader(json2))
	if err != nil {
		t.Fatalf("failed to parse json2: %v", err)
	}

	doc1.Graph.Normalize()
	doc2.Graph.Normalize()

	// Marshal graph portions
	b1, err := json.Marshal(doc1.Graph)
	if err != nil {
		t.Fatalf("failed to marshal doc1.Graph: %v", err)
	}
	b2, err := json.Marshal(doc2.Graph)
	if err != nil {
		t.Fatalf("failed to marshal doc2.Graph: %v", err)
	}

	if !bytes.Equal(b1, b2) {
		t.Errorf("parsed and normalized graphs are not byte-identical:\ng1: %s\ng2: %s", b1, b2)
	}
}

func TestNormalized_DoesNotModifyOriginal(t *testing.T) {
	g := &Graph{
		Nodes: []Node{
			{ID: "z", Type: "t", Inputs: map[string]any{}, Outputs: []string{"b", "a"}},
			{ID: "a", Type: "t", Inputs: map[string]any{}, Outputs: []string{}},
		},
		Edges: []Edge{},
	}

	// Get normalized copy
	norm := g.Normalized()

	// Original should be unchanged
	if g.Nodes[0].ID != "z" {
		t.Error("original graph was modified - first node ID changed")
	}
	if g.Nodes[0].Outputs[0] != "b" {
		t.Error("original graph was modified - outputs were sorted")
	}

	// Normalized copy should be sorted
	if norm.Nodes[0].ID != "a" {
		t.Error("normalized copy not sorted properly")
	}
}

func TestNormalize_EmptyGraph(t *testing.T) {
	g := &Graph{
		Nodes: []Node{},
		Edges: []Edge{},
	}

	g.Normalize()

	if len(g.Nodes) != 0 || len(g.Edges) != 0 {
		t.Error("empty graph should remain empty after normalization")
	}
}

func TestNormalize_InputsMapKeysSorted(t *testing.T) {
	// Go's encoding/json sorts map keys alphabetically
	g := &Graph{
		Nodes: []Node{
			{ID: "a", Type: "t", Inputs: map[string]any{"z": 1, "a": 2, "m": 3}, Outputs: []string{}},
		},
		Edges: []Edge{},
	}

	g.Normalize()

	b, err := json.Marshal(g)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Verify keys appear in sorted order in JSON
	jsonStr := string(b)
	aIdx := strings.Index(jsonStr, `"a":`)
	mIdx := strings.Index(jsonStr, `"m":`)
	zIdx := strings.Index(jsonStr, `"z":`)

	if !(aIdx < mIdx && mIdx < zIdx) {
		t.Errorf("map keys not sorted in JSON output: %s", jsonStr)
	}
}
