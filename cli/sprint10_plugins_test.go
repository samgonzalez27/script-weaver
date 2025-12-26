package cli_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	icl "scriptweaver/internal/cli"
)

func TestPluginsList_Deterministic_DisabledMarked(t *testing.T) {
	root := t.TempDir()
	pluginsRoot := filepath.Join(root, ".scriptweaver", "plugins")

	// Valid plugin.
	if err := os.MkdirAll(filepath.Join(pluginsRoot, "pA"), 0o755); err != nil {
		t.Fatalf("mkdir pA: %v", err)
	}
	manifestA := `{"plugin_id":"a","version":"1.0.0","hooks":["BeforeRun"],"description":"ok"}`
	if err := os.WriteFile(filepath.Join(pluginsRoot, "pA", "manifest.json"), []byte(manifestA), 0o644); err != nil {
		t.Fatalf("write manifestA: %v", err)
	}

	// Invalid plugin (missing hooks field).
	if err := os.MkdirAll(filepath.Join(pluginsRoot, "pB"), 0o755); err != nil {
		t.Fatalf("mkdir pB: %v", err)
	}
	manifestB := `{"plugin_id":"b","version":"1.0.0","description":"bad"}`
	if err := os.WriteFile(filepath.Join(pluginsRoot, "pB", "manifest.json"), []byte(manifestB), 0o644); err != nil {
		t.Fatalf("write manifestB: %v", err)
	}

	oldCwd, _ := os.Getwd()
	_ = os.Chdir(root)
	t.Cleanup(func() { _ = os.Chdir(oldCwd) })

	stdout := captureStdout(t, func() {
		res, err := icl.Run(context.Background(), []string{"plugins", "list"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.ExitCode != icl.ExitSuccess {
			t.Fatalf("expected exit 0 got %d", res.ExitCode)
		}
	})

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, got %q", stdout)
	}
	if !strings.HasPrefix(lines[0], "a enabled") {
		t.Fatalf("expected enabled plugin first, got %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "pB disabled") {
		t.Fatalf("expected disabled plugin marked, got %q", lines[1])
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = old
	b, err := io.ReadAll(r)
	_ = r.Close()
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	return string(b)
}
