package graph

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// getTestdataPath returns the absolute path to the testdata directory.
func getTestdataPath() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata")
}

func loadFixture(t *testing.T, name string) *Document {
	t.Helper()
	path := filepath.Join(getTestdataPath(), name)
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("failed to open fixture %s: %v", name, err)
	}
	defer f.Close()

	doc, err := Parse(f)
	if err != nil {
		t.Fatalf("failed to parse fixture %s: %v", name, err)
	}
	return doc
}

// --- Golden Fixture Tests ---

// Locked hashes for regression testing.
// These hashes are the contract - if they change, it's a breaking change.
const (
	// Hash of minimal.graph.json (empty nodes/edges)
	MinimalGraphHash = "a461bf77bc4e4d732f7afc121c70e7f70ed8bf225a082a4e01951d1eb6b5c278"

	// Hash of maximal.graph.json (complex pipeline)
	MaximalGraphHash = "87f41d22ad26e2102bfd37bf69cc45866886873cb2292563efe800ab5f92fc9a"
)

func TestGolden_MinimalGraph_ParseAndValidate(t *testing.T) {
	doc := loadFixture(t, "minimal.graph.json")

	if doc.SchemaVersion != "1.0.0" {
		t.Errorf("expected schema_version 1.0.0, got %s", doc.SchemaVersion)
	}

	if err := Validate(&doc.Graph); err != nil {
		t.Errorf("minimal graph should be valid, got error: %v", err)
	}
}

func TestGolden_MinimalGraph_HashLocked(t *testing.T) {
	doc := loadFixture(t, "minimal.graph.json")

	hash, err := ComputeHash(&doc.Graph)
	if err != nil {
		t.Fatalf("failed to compute hash: %v", err)
	}

	if hash != MinimalGraphHash {
		t.Errorf("minimal graph hash changed!\nexpected: %s\ngot:      %s\nThis is a BREAKING CHANGE if intentional.", MinimalGraphHash, hash)
	}
}

func TestGolden_MaximalGraph_ParseAndValidate(t *testing.T) {
	doc := loadFixture(t, "maximal.graph.json")

	if doc.SchemaVersion != "1.0.0" {
		t.Errorf("expected schema_version 1.0.0, got %s", doc.SchemaVersion)
	}

	if err := Validate(&doc.Graph); err != nil {
		t.Errorf("maximal graph should be valid, got error: %v", err)
	}

	// Verify structure
	if len(doc.Graph.Nodes) != 4 {
		t.Errorf("expected 4 nodes, got %d", len(doc.Graph.Nodes))
	}
	if len(doc.Graph.Edges) != 3 {
		t.Errorf("expected 3 edges, got %d", len(doc.Graph.Edges))
	}
}

func TestGolden_MaximalGraph_HashLocked(t *testing.T) {
	doc := loadFixture(t, "maximal.graph.json")

	hash, err := ComputeHash(&doc.Graph)
	if err != nil {
		t.Fatalf("failed to compute hash: %v", err)
	}

	if hash != MaximalGraphHash {
		t.Errorf("maximal graph hash changed!\nexpected: %s\ngot:      %s\nThis is a BREAKING CHANGE if intentional.", MaximalGraphHash, hash)
	}
}

func TestGolden_CyclicGraph_FailsValidation(t *testing.T) {
	doc := loadFixture(t, "cyclic.graph.json")

	err := Validate(&doc.Graph)
	if err == nil {
		t.Fatal("cyclic graph should fail validation")
	}

	if !errors.Is(err, ErrStructural) {
		t.Errorf("expected StructuralError, got %T: %v", err, err)
	}

	se, ok := err.(*StructuralError)
	if !ok {
		t.Fatalf("expected *StructuralError, got %T", err)
	}
	if se.Kind != "cycle" {
		t.Errorf("expected error kind 'cycle', got %q", se.Kind)
	}
}

func TestGolden_DuplicateIdGraph_FailsValidation(t *testing.T) {
	doc := loadFixture(t, "duplicate_id.graph.json")

	err := Validate(&doc.Graph)
	if err == nil {
		t.Fatal("duplicate ID graph should fail validation")
	}

	if !errors.Is(err, ErrStructural) {
		t.Errorf("expected StructuralError, got %T: %v", err, err)
	}

	se, ok := err.(*StructuralError)
	if !ok {
		t.Fatalf("expected *StructuralError, got %T", err)
	}
	if se.Kind != "duplicate_id" {
		t.Errorf("expected error kind 'duplicate_id', got %q", se.Kind)
	}
}

// --- Hash Stability Across Metadata Changes ---

func TestGolden_MaximalGraph_MetadataDoesNotAffectHash(t *testing.T) {
	doc := loadFixture(t, "maximal.graph.json")

	// Compute hash with original metadata
	hash1, _ := ComputeHash(&doc.Graph)

	// Modify metadata
	doc.Metadata.Name = "Completely Different Name"
	doc.Metadata.Description = "This description was changed"
	doc.Metadata.Labels = []string{"different", "labels"}

	// Hash should be identical
	hash2, _ := ComputeHash(&doc.Graph)

	if hash1 != hash2 {
		t.Errorf("metadata change affected hash!\nbefore: %s\nafter:  %s", hash1, hash2)
	}
}
