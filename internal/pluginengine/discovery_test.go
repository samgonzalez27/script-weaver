package pluginengine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type captureLogger struct {
	lines []string
}

func (l *captureLogger) Printf(format string, args ...any) {
	l.lines = append(l.lines, fmt.Sprintf(format, args...))
}

func TestDiscoverAndRegister_NonRecursiveIgnoresSubSubdirectories(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	pluginDir := filepath.Join(root, "plugin1")
	if err := os.MkdirAll(filepath.Join(pluginDir, "nested"), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// Only nested has a manifest; pluginDir itself does not.
	if err := os.WriteFile(filepath.Join(pluginDir, "nested", "manifest.json"), []byte(`{
		"plugin_id": "p1",
		"version": "0.1.0",
		"hooks": ["BeforeRun"]
	}`), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	reg, errs := DiscoverAndRegister(root, nil)
	if len(errs) != 0 {
		t.Fatalf("errs = %#v, want none", errs)
	}
	if len(reg.Manifests) != 0 {
		t.Fatalf("got %d manifests, want 0", len(reg.Manifests))
	}
}

func TestDiscoverAndRegister_SkipsDirectoryMissingManifest(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "plugin-no-manifest"), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	reg, errs := DiscoverAndRegister(root, nil)
	if len(errs) != 0 {
		t.Fatalf("errs = %#v, want none", errs)
	}
	if len(reg.Manifests) != 0 {
		t.Fatalf("got %d manifests, want 0", len(reg.Manifests))
	}
}

func TestDiscoverAndRegister_DeterministicOrderByPluginID(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	// Directory names intentionally not correlated with plugin_id ordering.
	if err := os.MkdirAll(filepath.Join(root, "zzz"), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "aaa"), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "zzz", "manifest.json"), []byte(`{
		"plugin_id": "b",
		"version": "0.1.0",
		"hooks": ["BeforeRun"]
	}`), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "aaa", "manifest.json"), []byte(`{
		"plugin_id": "a",
		"version": "0.1.0",
		"hooks": ["BeforeRun"]
	}`), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	reg, errs := DiscoverAndRegister(root, nil)
	if len(errs) != 0 {
		t.Fatalf("errs = %#v, want none", errs)
	}
	if len(reg.Manifests) != 2 {
		t.Fatalf("got %d manifests, want 2", len(reg.Manifests))
	}
	if reg.Manifests[0].PluginID != "a" || reg.Manifests[1].PluginID != "b" {
		t.Fatalf("order = [%s %s], want [a b]", reg.Manifests[0].PluginID, reg.Manifests[1].PluginID)
	}
}

func TestDiscoverAndRegister_InvalidManifestLoggedAndSkipped(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "bad"), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "good"), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "bad", "manifest.json"), []byte(`{
		"plugin_id": "bad",
		"version": "0.1.0",
		"hooks": ["NotARealHook"]
	}`), 0o600); err != nil {
		t.Fatalf("write bad manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "good", "manifest.json"), []byte(`{
		"plugin_id": "good",
		"version": "0.1.0",
		"hooks": ["BeforeRun"]
	}`), 0o600); err != nil {
		t.Fatalf("write good manifest: %v", err)
	}

	log := &captureLogger{}
	reg, errs := DiscoverAndRegister(root, log)
	if len(reg.Manifests) != 1 || reg.Manifests[0].PluginID != "good" {
		t.Fatalf("manifests = %#v, want only 'good'", reg.Manifests)
	}
	if len(errs) == 0 {
		t.Fatalf("expected at least one error, got none")
	}
	joined := strings.Join(log.lines, "\n")
	if !strings.Contains(joined, "invalid plugin") {
		t.Fatalf("expected log to mention invalid plugin, got: %s", joined)
	}
}
