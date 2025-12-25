package pluginengine

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPluginManifestDir_ValidManifestParses(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "manifest.json")
	content := `{
		"plugin_id": "logging-plugin",
		"version": "0.1.0",
		"hooks": ["BeforeRun", "AfterRun"],
		"description": "test plugin"
	}`
	if err := os.WriteFile(manifestPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write manifest.json: %v", err)
	}

	m, err := LoadPluginManifestDir(dir)
	if err != nil {
		t.Fatalf("LoadPluginManifestDir() error = %v", err)
	}
	if m.PluginID != "logging-plugin" {
		t.Fatalf("PluginID = %q, want %q", m.PluginID, "logging-plugin")
	}
	if m.Version != "0.1.0" {
		t.Fatalf("Version = %q, want %q", m.Version, "0.1.0")
	}
	if len(m.Hooks) != 2 || m.Hooks[0] != "BeforeRun" || m.Hooks[1] != "AfterRun" {
		t.Fatalf("Hooks = %#v, want [BeforeRun AfterRun]", m.Hooks)
	}
	if m.Description != "test plugin" {
		t.Fatalf("Description = %q, want %q", m.Description, "test plugin")
	}
}

func TestRegisterManifests_RejectsDuplicatePluginIDs(t *testing.T) {
	t.Parallel()

	m1 := PluginManifest{PluginID: "dup", Version: "0.1.0", Hooks: []string{"BeforeRun"}}
	m2 := PluginManifest{PluginID: "dup", Version: "0.2.0", Hooks: []string{"AfterRun"}}
	_, err := RegisterManifests([]PluginManifest{m1, m2})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, ErrDuplicatePluginID) {
		t.Fatalf("error = %v, want errors.Is(ErrDuplicatePluginID)", err)
	}
}

func TestValidatePluginManifest_RejectsUnsupportedHooks(t *testing.T) {
	t.Parallel()

	m := PluginManifest{PluginID: "p1", Version: "0.1.0", Hooks: []string{"NotARealHook"}}
	err := ValidatePluginManifest(m)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, ErrUnsupportedHook) {
		t.Fatalf("error = %v, want errors.Is(ErrUnsupportedHook)", err)
	}
}

func TestLoadPluginManifestDir_MissingManifestReturnsError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	_, err := LoadPluginManifestDir(dir)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, ErrManifestNotFound) {
		t.Fatalf("error = %v, want errors.Is(ErrManifestNotFound)", err)
	}
}

func TestLoadPluginManifestDir_MalformedManifestReturnsError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(manifestPath, []byte("{\n"), 0o600); err != nil {
		t.Fatalf("write manifest.json: %v", err)
	}

	_, err := LoadPluginManifestDir(dir)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, ErrManifestMalformed) {
		t.Fatalf("error = %v, want errors.Is(ErrManifestMalformed)", err)
	}
}
