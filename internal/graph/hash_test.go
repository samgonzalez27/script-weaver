package graph

import (
	"strings"
	"testing"
)

// --- Hash Stability Tests ---

func TestComputeHash_SameGraphSameHash(t *testing.T) {
	g := &Graph{
		Nodes: []Node{
			{ID: "a", Type: "exec", Inputs: map[string]any{"cmd": "echo"}, Outputs: []string{"out"}},
		},
		Edges: []Edge{},
	}

	hash1, err := ComputeHash(g)
	if err != nil {
		t.Fatalf("first hash failed: %v", err)
	}

	hash2, err := ComputeHash(g)
	if err != nil {
		t.Fatalf("second hash failed: %v", err)
	}

	if hash1 != hash2 {
		t.Errorf("same graph produced different hashes: %s vs %s", hash1, hash2)
	}
}

func TestComputeHash_ReorderedJSONSameHash(t *testing.T) {
	// Parse two JSON documents with different ordering
	json1 := `{
		"schema_version": "1.0.0",
		"graph": {
			"nodes": [
				{"id": "z", "type": "t", "inputs": {"b": 2, "a": 1}, "outputs": ["y", "x"]},
				{"id": "a", "type": "t", "inputs": {}, "outputs": []}
			],
			"edges": [{"from": "z", "to": "a"}]
		},
		"metadata": {"name": "test1"}
	}`

	json2 := `{
		"schema_version": "1.0.0",
		"graph": {
			"nodes": [
				{"id": "a", "type": "t", "inputs": {}, "outputs": []},
				{"id": "z", "type": "t", "inputs": {"a": 1, "b": 2}, "outputs": ["x", "y"]}
			],
			"edges": [{"from": "z", "to": "a"}]
		},
		"metadata": {"name": "test2"}
	}`

	doc1, err := Parse(strings.NewReader(json1))
	if err != nil {
		t.Fatalf("parse json1 failed: %v", err)
	}
	doc2, err := Parse(strings.NewReader(json2))
	if err != nil {
		t.Fatalf("parse json2 failed: %v", err)
	}

	hash1, err := ComputeHash(&doc1.Graph)
	if err != nil {
		t.Fatalf("hash1 failed: %v", err)
	}
	hash2, err := ComputeHash(&doc2.Graph)
	if err != nil {
		t.Fatalf("hash2 failed: %v", err)
	}

	if hash1 != hash2 {
		t.Errorf("reordered JSON produced different hashes: %s vs %s", hash1, hash2)
	}
}

func TestComputeHash_WhitespaceOnlyChangesSameHash(t *testing.T) {
	// Compact JSON
	json1 := `{"schema_version":"1.0.0","graph":{"nodes":[{"id":"a","type":"t","inputs":{},"outputs":[]}],"edges":[]},"metadata":{}}`

	// Same with lots of whitespace
	json2 := `{
		"schema_version": "1.0.0",
		"graph": {
			"nodes": [
				{
					"id": "a",
					"type": "t",
					"inputs": {},
					"outputs": []
				}
			],
			"edges": []
		},
		"metadata": {}
	}`

	doc1, err := Parse(strings.NewReader(json1))
	if err != nil {
		t.Fatalf("parse json1 failed: %v", err)
	}
	doc2, err := Parse(strings.NewReader(json2))
	if err != nil {
		t.Fatalf("parse json2 failed: %v", err)
	}

	hash1, _ := ComputeHash(&doc1.Graph)
	hash2, _ := ComputeHash(&doc2.Graph)

	if hash1 != hash2 {
		t.Errorf("whitespace-only changes affected hash: %s vs %s", hash1, hash2)
	}
}

func TestComputeHash_MetadataChangeSameHash(t *testing.T) {
	json1 := `{
		"schema_version": "1.0.0",
		"graph": {"nodes": [{"id": "a", "type": "t", "inputs": {}, "outputs": []}], "edges": []},
		"metadata": {"name": "Original Name", "description": "Original"}
	}`

	json2 := `{
		"schema_version": "1.0.0",
		"graph": {"nodes": [{"id": "a", "type": "t", "inputs": {}, "outputs": []}], "edges": []},
		"metadata": {"name": "Different Name", "description": "Different", "labels": ["new"]}
	}`

	doc1, _ := Parse(strings.NewReader(json1))
	doc2, _ := Parse(strings.NewReader(json2))

	hash1, _ := ComputeHash(&doc1.Graph)
	hash2, _ := ComputeHash(&doc2.Graph)

	if hash1 != hash2 {
		t.Errorf("metadata change affected hash: %s vs %s", hash1, hash2)
	}
}

// --- Hash Sensitivity Tests ---

func TestComputeHash_NodeInputChangeDifferentHash(t *testing.T) {
	g1 := &Graph{
		Nodes: []Node{
			{ID: "a", Type: "exec", Inputs: map[string]any{"cmd": "echo"}, Outputs: []string{}},
		},
		Edges: []Edge{},
	}

	g2 := &Graph{
		Nodes: []Node{
			{ID: "a", Type: "exec", Inputs: map[string]any{"cmd": "cat"}, Outputs: []string{}},
		},
		Edges: []Edge{},
	}

	hash1, _ := ComputeHash(g1)
	hash2, _ := ComputeHash(g2)

	if hash1 == hash2 {
		t.Error("different node inputs should produce different hash")
	}
}

func TestComputeHash_EdgeChangeDifferentHash(t *testing.T) {
	g1 := &Graph{
		Nodes: []Node{
			{ID: "a", Type: "t", Inputs: map[string]any{}, Outputs: []string{}},
			{ID: "b", Type: "t", Inputs: map[string]any{}, Outputs: []string{}},
		},
		Edges: []Edge{{From: "a", To: "b"}},
	}

	g2 := &Graph{
		Nodes: []Node{
			{ID: "a", Type: "t", Inputs: map[string]any{}, Outputs: []string{}},
			{ID: "b", Type: "t", Inputs: map[string]any{}, Outputs: []string{}},
		},
		Edges: []Edge{{From: "b", To: "a"}},
	}

	hash1, _ := ComputeHash(g1)
	hash2, _ := ComputeHash(g2)

	if hash1 == hash2 {
		t.Error("different edges should produce different hash")
	}
}

func TestComputeHash_SemanticChangeDifferentHash(t *testing.T) {
	g1 := &Graph{
		Nodes: []Node{
			{ID: "a", Type: "exec", Inputs: map[string]any{}, Outputs: []string{}},
		},
		Edges: []Edge{},
	}

	g2 := &Graph{
		Nodes: []Node{
			{ID: "a", Type: "shell", Inputs: map[string]any{}, Outputs: []string{}},
		},
		Edges: []Edge{},
	}

	hash1, _ := ComputeHash(g1)
	hash2, _ := ComputeHash(g2)

	if hash1 == hash2 {
		t.Error("different node type should produce different hash")
	}
}

func TestComputeHash_NodeAddedDifferentHash(t *testing.T) {
	g1 := &Graph{
		Nodes: []Node{
			{ID: "a", Type: "t", Inputs: map[string]any{}, Outputs: []string{}},
		},
		Edges: []Edge{},
	}

	g2 := &Graph{
		Nodes: []Node{
			{ID: "a", Type: "t", Inputs: map[string]any{}, Outputs: []string{}},
			{ID: "b", Type: "t", Inputs: map[string]any{}, Outputs: []string{}},
		},
		Edges: []Edge{},
	}

	hash1, _ := ComputeHash(g1)
	hash2, _ := ComputeHash(g2)

	if hash1 == hash2 {
		t.Error("added node should produce different hash")
	}
}

func TestComputeHash_OutputChangeDifferentHash(t *testing.T) {
	g1 := &Graph{
		Nodes: []Node{
			{ID: "a", Type: "t", Inputs: map[string]any{}, Outputs: []string{"out1"}},
		},
		Edges: []Edge{},
	}

	g2 := &Graph{
		Nodes: []Node{
			{ID: "a", Type: "t", Inputs: map[string]any{}, Outputs: []string{"out2"}},
		},
		Edges: []Edge{},
	}

	hash1, _ := ComputeHash(g1)
	hash2, _ := ComputeHash(g2)

	if hash1 == hash2 {
		t.Error("different outputs should produce different hash")
	}
}

func TestComputeHash_EmptyGraph(t *testing.T) {
	g := &Graph{
		Nodes: []Node{},
		Edges: []Edge{},
	}

	hash, err := ComputeHash(g)
	if err != nil {
		t.Fatalf("hash failed: %v", err)
	}

	// Hash should be deterministic
	if len(hash) != 64 { // SHA-256 hex = 64 chars
		t.Errorf("expected 64 char hex hash, got %d chars", len(hash))
	}

	// Verify it's the same on subsequent calls
	hash2, _ := ComputeHash(g)
	if hash != hash2 {
		t.Error("empty graph hash not stable")
	}
}

func TestComputeHash_DoesNotModifyOriginal(t *testing.T) {
	g := &Graph{
		Nodes: []Node{
			{ID: "z", Type: "t", Inputs: map[string]any{}, Outputs: []string{"b", "a"}},
			{ID: "a", Type: "t", Inputs: map[string]any{}, Outputs: []string{}},
		},
		Edges: []Edge{},
	}

	_, err := ComputeHash(g)
	if err != nil {
		t.Fatalf("hash failed: %v", err)
	}

	// Original should be unchanged
	if g.Nodes[0].ID != "z" {
		t.Error("original graph was modified - node order changed")
	}
	if g.Nodes[0].Outputs[0] != "b" {
		t.Error("original graph was modified - outputs sorted")
	}
}
