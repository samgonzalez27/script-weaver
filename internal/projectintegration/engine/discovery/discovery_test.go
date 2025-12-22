package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

const validMinimalGraphJSON = `{
	"schema_version": "1.0.0",
	"graph": {"nodes": [], "edges": []},
	"metadata": {}
}`

func TestDiscover_ExplicitPathWins(t *testing.T) {
	root := t.TempDir()

	// Put a valid graph in graphs/ too; explicit must win.
	mustWrite(t, filepath.Join(root, "graphs", "a.json"), validMinimalGraphJSON)
	mustWrite(t, filepath.Join(root, "explicit.json"), validMinimalGraphJSON)

	p, err := Discover(root, "explicit.json")
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	want := filepath.Join(root, "explicit.json")
	if p != want {
		t.Fatalf("path = %q, want %q", p, want)
	}
}

func TestDiscover_GraphsDirAtRoot(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "graphs", "only.json"), validMinimalGraphJSON)

	p, err := Discover(root, "")
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	want := filepath.Join(root, "graphs", "only.json")
	if p != want {
		t.Fatalf("path = %q, want %q", p, want)
	}
}

func TestDiscover_ScriptweaverGraphsFallback(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, ".scriptweaver", "graphs", "only.json"), validMinimalGraphJSON)

	p, err := Discover(root, "")
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	want := filepath.Join(root, ".scriptweaver", "graphs", "only.json")
	if p != want {
		t.Fatalf("path = %q, want %q", p, want)
	}
}

func TestDiscover_AmbiguousGraphsDirFails(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "graphs", "b.json"), validMinimalGraphJSON)
	mustWrite(t, filepath.Join(root, "graphs", "a.json"), validMinimalGraphJSON)

	if _, err := Discover(root, ""); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestDiscover_AmbiguousScriptweaverGraphsDirFails(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, ".scriptweaver", "graphs", "b.json"), validMinimalGraphJSON)
	mustWrite(t, filepath.Join(root, ".scriptweaver", "graphs", "a.json"), validMinimalGraphJSON)

	if _, err := Discover(root, ""); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestDiscover_InvalidGraphFails(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "graphs", "bad.json"), `{"nope":true}`)

	if _, err := Discover(root, ""); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}
