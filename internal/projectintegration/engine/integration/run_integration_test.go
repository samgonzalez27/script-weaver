package integration

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

func TestRun_ZeroConfigCleanRepo(t *testing.T) {
	root := t.TempDir()

	// Simulate a "clean repo" with a single graph in graphs/.
	mustWrite(t, filepath.Join(root, "graphs", "only.json"), validMinimalGraphJSON)

	res, err := Run(root, "", true)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.GraphPath != filepath.Join(root, "graphs", "only.json") {
		t.Fatalf("GraphPath = %q", res.GraphPath)
	}

	// Zero-config creates the workspace structure.
	mustBeDir(t, filepath.Join(root, ".scriptweaver"))
	mustBeDir(t, filepath.Join(root, ".scriptweaver", "cache"))
	mustBeDir(t, filepath.Join(root, ".scriptweaver", "runs"))
	mustBeDir(t, filepath.Join(root, ".scriptweaver", "logs"))
}

func TestRun_Isolation_UserFilesUntouched(t *testing.T) {
	root := t.TempDir()

	// User file outside .scriptweaver.
	userPath := filepath.Join(root, "README.md")
	mustWrite(t, userPath, "hello")
	beforeHash := mustHash(t, userPath)

	mustWrite(t, filepath.Join(root, "graphs", "only.json"), validMinimalGraphJSON)

	_, err := Run(root, "", true)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	afterHash := mustHash(t, userPath)
	if beforeHash != afterHash {
		t.Fatalf("user file hash changed")
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

func mustHash(t *testing.T, path string) string {
	t.Helper()
	h, err := hashFile(path)
	if err != nil {
		t.Fatalf("hashFile: %v", err)
	}
	return h
}
