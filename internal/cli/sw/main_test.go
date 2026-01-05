package sw

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	d := wd
	for {
		if _, err := os.Stat(filepath.Join(d, "go.mod")); err == nil {
			return d
		}
		parent := filepath.Dir(d)
		if parent == d {
			t.Fatalf("could not find repo root from %q", wd)
		}
		d = parent
	}
}

func TestRun_CleanDefault_Succeeds(t *testing.T) {
	root := repoRoot(t)
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	workdir := t.TempDir()

	var out, errBuf bytes.Buffer
	exit := Main([]string{"run", "--graph", "fixtures/basic.json", "--workdir", workdir}, &out, &errBuf)
	if exit != ExitSuccess {
		t.Fatalf("exit=%d stderr=%q", exit, errBuf.String())
	}
	if !strings.Contains(out.String(), "Execution succeeded") {
		t.Fatalf("stdout=%q", out.String())
	}
}

func TestValidate_Cycle_FailsWithExit1(t *testing.T) {
	root := repoRoot(t)
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	var out, errBuf bytes.Buffer
	exit := Main([]string{"validate", "--graph", "fixtures/cyclic.json"}, &out, &errBuf)
	if exit != ExitValidationError {
		t.Fatalf("exit=%d stderr=%q", exit, errBuf.String())
	}
	if !strings.Contains(errBuf.String(), "Cycle detected") {
		t.Fatalf("stderr=%q", errBuf.String())
	}
}

func TestRun_UnknownFlag_StrictExit2(t *testing.T) {
	root := repoRoot(t)
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	workdir := t.TempDir()
	var out, errBuf bytes.Buffer
	exit := Main([]string{"run", "--graph", "fixtures/basic.json", "--workdir", workdir, "--random-flag"}, &out, &errBuf)
	if exit != ExitArgOrSystemError {
		t.Fatalf("exit=%d stderr=%q", exit, errBuf.String())
	}
	if !strings.Contains(strings.ToLower(errBuf.String()), "unknown flag") {
		t.Fatalf("stderr=%q", errBuf.String())
	}
}

func TestHash_Stable_IgnoresWorkdirFlag(t *testing.T) {
	root := repoRoot(t)
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	var out1, err1 bytes.Buffer
	exit1 := Main([]string{"hash", "--graph", "fixtures/basic.json"}, &out1, &err1)
	if exit1 != ExitSuccess {
		t.Fatalf("exit=%d stderr=%q", exit1, err1.String())
	}

	var out2, err2 bytes.Buffer
	exit2 := Main([]string{"hash", "--graph", "fixtures/basic.json", "--workdir", t.TempDir()}, &out2, &err2)
	if exit2 != ExitSuccess {
		t.Fatalf("exit=%d stderr=%q", exit2, err2.String())
	}

	if strings.TrimSpace(out1.String()) != strings.TrimSpace(out2.String()) {
		t.Fatalf("hash mismatch: %q vs %q", out1.String(), out2.String())
	}
}

func TestPluginsList_OutputSortedPlaintext(t *testing.T) {
	root := repoRoot(t)
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	pluginDir := t.TempDir()
	// Create two plugin dirs with manifests.
	alpha := filepath.Join(pluginDir, "alpha")
	beta := filepath.Join(pluginDir, "beta")
	if err := os.MkdirAll(alpha, 0o755); err != nil {
		t.Fatalf("mkdir alpha: %v", err)
	}
	if err := os.MkdirAll(beta, 0o755); err != nil {
		t.Fatalf("mkdir beta: %v", err)
	}
	if err := os.WriteFile(filepath.Join(alpha, "manifest.json"), []byte(`{"plugin_id":"Alpha","version":"0.0.0","hooks":["BeforeRun"],"description":"alpha"}`), 0o644); err != nil {
		t.Fatalf("write alpha manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(beta, "manifest.json"), []byte(`{"plugin_id":"Beta","version":"0.0.0","hooks":["BeforeRun"],"description":"beta"}`), 0o644); err != nil {
		t.Fatalf("write beta manifest: %v", err)
	}

	var out, errBuf bytes.Buffer
	exit := Main([]string{"plugins", "list", "--plugin-dir", pluginDir}, &out, &errBuf)
	if exit != ExitSuccess {
		t.Fatalf("exit=%d stderr=%q", exit, errBuf.String())
	}
	got := strings.TrimSpace(out.String())
	if got != "Alpha\nBeta" {
		t.Fatalf("stdout=%q", out.String())
	}
}
