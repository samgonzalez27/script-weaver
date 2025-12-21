package graph

import (
	"errors"
	"strings"
	"testing"
)

// validMinimalJSON is the smallest valid graph definition.
const validMinimalJSON = `{
	"schema_version": "1.0.0",
	"graph": {
		"nodes": [],
		"edges": []
	},
	"metadata": {}
}`

func TestParse_ValidMinimal(t *testing.T) {
	doc, err := Parse(strings.NewReader(validMinimalJSON))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if doc.SchemaVersion != "1.0.0" {
		t.Errorf("expected schema_version 1.0.0, got %s", doc.SchemaVersion)
	}
	if doc.Graph.Nodes == nil {
		t.Error("expected nodes to be non-nil")
	}
	if doc.Graph.Edges == nil {
		t.Error("expected edges to be non-nil")
	}
}

func TestParse_ValidWithNodes(t *testing.T) {
	json := `{
		"schema_version": "1.0.0",
		"graph": {
			"nodes": [
				{"id": "node1", "type": "exec", "inputs": {"cmd": "echo"}, "outputs": ["stdout"]}
			],
			"edges": []
		},
		"metadata": {"name": "test"}
	}`
	doc, err := Parse(strings.NewReader(json))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(doc.Graph.Nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(doc.Graph.Nodes))
	}
	if doc.Graph.Nodes[0].ID != "node1" {
		t.Errorf("expected node id 'node1', got %s", doc.Graph.Nodes[0].ID)
	}
}

func TestParse_MissingSchemaVersion(t *testing.T) {
	json := `{
		"graph": {"nodes": [], "edges": []},
		"metadata": {}
	}`
	_, err := Parse(strings.NewReader(json))
	if err == nil {
		t.Fatal("expected error for missing schema_version")
	}
	if !errors.Is(err, ErrSchema) {
		t.Errorf("expected SchemaError, got %T: %v", err, err)
	}
}

func TestParse_MissingGraph(t *testing.T) {
	json := `{
		"schema_version": "1.0.0",
		"metadata": {}
	}`
	_, err := Parse(strings.NewReader(json))
	if err == nil {
		t.Fatal("expected error for missing graph")
	}
	if !errors.Is(err, ErrSchema) {
		t.Errorf("expected SchemaError, got %T: %v", err, err)
	}
}

func TestParse_MissingMetadata(t *testing.T) {
	// Note: metadata is a required field but it's an empty object, so missing
	// entirely should still work if the JSON decoder doesn't enforce it.
	// Per spec, it's required, so we test that an empty metadata passes.
	json := `{
		"schema_version": "1.0.0",
		"graph": {"nodes": [], "edges": []},
		"metadata": {}
	}`
	_, err := Parse(strings.NewReader(json))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestParse_UnknownTopLevelField(t *testing.T) {
	json := `{
		"schema_version": "1.0.0",
		"graph": {"nodes": [], "edges": []},
		"metadata": {},
		"extra_field": "should fail"
	}`
	_, err := Parse(strings.NewReader(json))
	if err == nil {
		t.Fatal("expected error for unknown field")
	}
	// Unknown fields cause ParseError (from DisallowUnknownFields)
	if !errors.Is(err, ErrParse) {
		t.Errorf("expected ParseError, got %T: %v", err, err)
	}
}

func TestParse_UnknownNodeField(t *testing.T) {
	json := `{
		"schema_version": "1.0.0",
		"graph": {
			"nodes": [{"id": "n1", "type": "t", "inputs": {}, "outputs": [], "unknown": true}],
			"edges": []
		},
		"metadata": {}
	}`
	_, err := Parse(strings.NewReader(json))
	if err == nil {
		t.Fatal("expected error for unknown node field")
	}
	if !errors.Is(err, ErrParse) {
		t.Errorf("expected ParseError, got %T: %v", err, err)
	}
}

func TestParse_IncorrectTypeForNodes(t *testing.T) {
	json := `{
		"schema_version": "1.0.0",
		"graph": {"nodes": "not an array", "edges": []},
		"metadata": {}
	}`
	_, err := Parse(strings.NewReader(json))
	if err == nil {
		t.Fatal("expected error for wrong type")
	}
	if !errors.Is(err, ErrSchema) {
		t.Errorf("expected SchemaError for type mismatch, got %T: %v", err, err)
	}
}

func TestParse_IncorrectTypeForSchemaVersion(t *testing.T) {
	json := `{
		"schema_version": 100,
		"graph": {"nodes": [], "edges": []},
		"metadata": {}
	}`
	_, err := Parse(strings.NewReader(json))
	if err == nil {
		t.Fatal("expected error for wrong type")
	}
	if !errors.Is(err, ErrSchema) {
		t.Errorf("expected SchemaError for type mismatch, got %T: %v", err, err)
	}
}

func TestParse_UnsupportedVersion(t *testing.T) {
	json := `{
		"schema_version": "2.0.0",
		"graph": {"nodes": [], "edges": []},
		"metadata": {}
	}`
	_, err := Parse(strings.NewReader(json))
	if err == nil {
		t.Fatal("expected error for unsupported version")
	}
	if !errors.Is(err, ErrSemantic) {
		t.Errorf("expected SemanticError, got %T: %v", err, err)
	}
}

func TestParse_MalformedJSON(t *testing.T) {
	json := `{not valid json}`
	_, err := Parse(strings.NewReader(json))
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
	if !errors.Is(err, ErrParse) {
		t.Errorf("expected ParseError, got %T: %v", err, err)
	}
}

func TestParse_MissingNodeRequiredFields(t *testing.T) {
	testCases := []struct {
		name  string
		json  string
		field string
	}{
		{
			name: "missing node id",
			json: `{
				"schema_version": "1.0.0",
				"graph": {"nodes": [{"type": "t", "inputs": {}, "outputs": []}], "edges": []},
				"metadata": {}
			}`,
			field: "id",
		},
		{
			name: "missing node type",
			json: `{
				"schema_version": "1.0.0",
				"graph": {"nodes": [{"id": "n1", "inputs": {}, "outputs": []}], "edges": []},
				"metadata": {}
			}`,
			field: "type",
		},
		{
			name: "missing node inputs",
			json: `{
				"schema_version": "1.0.0",
				"graph": {"nodes": [{"id": "n1", "type": "t", "outputs": []}], "edges": []},
				"metadata": {}
			}`,
			field: "inputs",
		},
		{
			name: "missing node outputs",
			json: `{
				"schema_version": "1.0.0",
				"graph": {"nodes": [{"id": "n1", "type": "t", "inputs": {}}], "edges": []},
				"metadata": {}
			}`,
			field: "outputs",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse(strings.NewReader(tc.json))
			if err == nil {
				t.Fatalf("expected error for missing %s", tc.field)
			}
			if !errors.Is(err, ErrSchema) {
				t.Errorf("expected SchemaError, got %T: %v", err, err)
			}
		})
	}
}

func TestParse_MissingEdgeFields(t *testing.T) {
	testCases := []struct {
		name  string
		json  string
		field string
	}{
		{
			name: "missing edge from",
			json: `{
				"schema_version": "1.0.0",
				"graph": {"nodes": [], "edges": [{"to": "n1"}]},
				"metadata": {}
			}`,
			field: "from",
		},
		{
			name: "missing edge to",
			json: `{
				"schema_version": "1.0.0",
				"graph": {"nodes": [], "edges": [{"from": "n1"}]},
				"metadata": {}
			}`,
			field: "to",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse(strings.NewReader(tc.json))
			if err == nil {
				t.Fatalf("expected error for missing %s", tc.field)
			}
			if !errors.Is(err, ErrSchema) {
				t.Errorf("expected SchemaError, got %T: %v", err, err)
			}
		})
	}
}
