package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParse_AllowsGraphPathOnly(t *testing.T) {
	cfg, err := Parse([]byte(`{"graph_path":"graphs/main.graph.json"}`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cfg.GraphPath != "graphs/main.graph.json" {
		t.Fatalf("GraphPath = %q", cfg.GraphPath)
	}
}

func TestParse_RejectsUnknownField(t *testing.T) {
	if _, err := Parse([]byte(`{"graph_path":"x","extra":true}`)); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestParse_RejectsWorkspacePath(t *testing.T) {
	errText := mustErrText(t, func() error {
		_, err := Parse([]byte(`{"workspace_path":"/tmp/sw"}`))
		return err
	})
	if errText == "" {
		t.Fatalf("expected non-empty error")
	}
}

func TestParse_RejectsSemanticOverrides(t *testing.T) {
	if _, err := Parse([]byte(`{"semantic_overrides":{}}`)); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestParse_RejectsNonStringGraphPath(t *testing.T) {
	if _, err := Parse([]byte(`{"graph_path":123}`)); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestParse_RejectsEmptyGraphPath(t *testing.T) {
	if _, err := Parse([]byte(`{"graph_path":"   "}`)); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestLoadOptional_MissingConfigIsNotAnError(t *testing.T) {
	root := t.TempDir()
	cfg, ok, err := LoadOptional(root)
	if err != nil {
		t.Fatalf("LoadOptional: %v", err)
	}
	if ok {
		t.Fatalf("ok = true, want false")
	}
	if cfg.GraphPath != "" {
		t.Fatalf("GraphPath = %q, want empty", cfg.GraphPath)
	}
}

func TestLoadOptional_LoadsOnlyFromScriptweaverDir(t *testing.T) {
	root := t.TempDir()

	// Create a config at the project root; it must be ignored.
	if err := os.WriteFile(filepath.Join(root, "config.json"), []byte(`{"graph_path":"graphs/root.graph.json"}`), 0o644); err != nil {
		t.Fatalf("WriteFile root config: %v", err)
	}

	cfg, ok, err := LoadOptional(root)
	if err != nil {
		t.Fatalf("LoadOptional: %v", err)
	}
	if ok {
		t.Fatalf("ok = true, want false")
	}
	if cfg.GraphPath != "" {
		t.Fatalf("GraphPath = %q, want empty", cfg.GraphPath)
	}

	// Now create the correct location.
	if err := os.MkdirAll(filepath.Join(root, ".scriptweaver"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".scriptweaver", "config.json"), []byte(`{"graph_path":"graphs/correct.graph.json"}`), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, ok, err = LoadOptional(root)
	if err != nil {
		t.Fatalf("LoadOptional: %v", err)
	}
	if !ok {
		t.Fatalf("ok = false, want true")
	}
	if cfg.GraphPath != "graphs/correct.graph.json" {
		t.Fatalf("GraphPath = %q", cfg.GraphPath)
	}
}

func mustErrText(t *testing.T, fn func() error) string {
	t.Helper()
	err := fn()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	return err.Error()
}
