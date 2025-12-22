package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureWorkspace_CreatesStructureWhenMissing(t *testing.T) {
	root := t.TempDir()

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})
	if err := os.Chdir(root); err != nil {
		t.Fatalf("Chdir: %v", err)
	}

	ws, err := EnsureWorkspace("")
	if err != nil {
		t.Fatalf("EnsureWorkspace: %v", err)
	}
	if ws.ProjectRoot != root {
		t.Fatalf("ProjectRoot = %q, want %q", ws.ProjectRoot, root)
	}

	mustBeDir(t, filepath.Join(root, ".scriptweaver"))
	mustBeDir(t, filepath.Join(root, ".scriptweaver", "cache"))
	mustBeDir(t, filepath.Join(root, ".scriptweaver", "runs"))
	mustBeDir(t, filepath.Join(root, ".scriptweaver", "logs"))
}

func TestEnsureWorkspace_CreatesStructureRelativeToExplicitRoot(t *testing.T) {
	root := t.TempDir()

	ws, err := EnsureWorkspace(root)
	if err != nil {
		t.Fatalf("EnsureWorkspace: %v", err)
	}
	if ws.ProjectRoot != root {
		t.Fatalf("ProjectRoot = %q, want %q", ws.ProjectRoot, root)
	}

	mustBeDir(t, filepath.Join(root, ".scriptweaver"))
	mustBeDir(t, filepath.Join(root, ".scriptweaver", "cache"))
	mustBeDir(t, filepath.Join(root, ".scriptweaver", "runs"))
	mustBeDir(t, filepath.Join(root, ".scriptweaver", "logs"))
}

func TestEnsureWorkspace_AllowsOptionalConfigJSON(t *testing.T) {
	root := t.TempDir()
	workspaceDir := filepath.Join(root, ".scriptweaver")
	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspaceDir, "config.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := EnsureWorkspace(root); err != nil {
		t.Fatalf("EnsureWorkspace: %v", err)
	}
}

func TestEnsureWorkspace_AllowsOptionalGraphsDir(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".scriptweaver", "graphs"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	if _, err := EnsureWorkspace(root); err != nil {
		t.Fatalf("EnsureWorkspace: %v", err)
	}
}

func TestEnsureWorkspace_RejectsUnauthorizedEntries(t *testing.T) {
	root := t.TempDir()
	workspaceDir := filepath.Join(root, ".scriptweaver")
	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspaceDir, "evil.txt"), []byte("nope"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := EnsureWorkspace(root); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestEnsureWorkspace_RejectsRequiredDirNameAsFile(t *testing.T) {
	root := t.TempDir()
	workspaceDir := filepath.Join(root, ".scriptweaver")
	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspaceDir, "cache"), []byte("not a dir"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := EnsureWorkspace(root); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestEnsureWorkspace_RejectsWorkspacePathCollision(t *testing.T) {
	root := t.TempDir()
	workspacePath := filepath.Join(root, ".scriptweaver")
	if err := os.WriteFile(workspacePath, []byte("not a dir"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := EnsureWorkspace(root); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func mustBeDir(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat %s: %v", path, err)
	}
	if !info.IsDir() {
		t.Fatalf("%s is not a dir", path)
	}
}
