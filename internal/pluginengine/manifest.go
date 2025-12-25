package pluginengine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// PluginManifest is defined by the Sprint-09 Data Dictionary.
// JSON field mapping must remain stable.
type PluginManifest struct {
	PluginID     string   `json:"plugin_id"`
	Version      string   `json:"version"`
	Hooks        []string `json:"hooks"`
	Description  string   `json:"description"`
}

// RuntimePluginState is defined by the Sprint-09 Data Dictionary.
// JSON tags are included for consistent field mapping, although this is runtime-only state.
type RuntimePluginState struct {
	PluginID   string `json:"plugin_id"`
	Enabled    bool   `json:"enabled"`
	LoadError  string `json:"load_error"`
}

// SupportedHooks returns the set of allowed hook names.
func SupportedHooks() map[string]struct{} {
	return map[string]struct{}{
		"BeforeRun":  {},
		"AfterRun":   {},
		"BeforeNode": {},
		"AfterNode":  {},
	}
}

func ValidatePluginManifest(m PluginManifest) error {
	if m.PluginID == "" {
		return fmt.Errorf("%w: %w", ErrManifestInvalid, ErrMissingPluginID)
	}
	if m.Version == "" {
		return fmt.Errorf("%w: %w", ErrManifestInvalid, ErrMissingVersion)
	}
	if m.Hooks == nil {
		return fmt.Errorf("%w: %w", ErrManifestInvalid, ErrMissingHooks)
	}
	if len(m.Hooks) == 0 {
		return fmt.Errorf("%w: %w", ErrManifestInvalid, ErrEmptyHooks)
	}

	supported := SupportedHooks()
	for _, hook := range m.Hooks {
		if _, ok := supported[hook]; !ok {
			return fmt.Errorf("%w: %w: %s", ErrManifestInvalid, ErrUnsupportedHook, hook)
		}
	}

	return nil
}

func ParsePluginManifestJSON(r io.Reader) (PluginManifest, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()

	var m PluginManifest
	if err := dec.Decode(&m); err != nil {
		return PluginManifest{}, fmt.Errorf("%w: %w", ErrManifestMalformed, err)
	}
	// Ensure there is no trailing junk.
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		if err == nil {
			return PluginManifest{}, fmt.Errorf("%w: trailing data", ErrManifestMalformed)
		}
		return PluginManifest{}, fmt.Errorf("%w: %w", ErrManifestMalformed, err)
	}

	if err := ValidatePluginManifest(m); err != nil {
		return PluginManifest{}, err
	}
	return m, nil
}

func ParsePluginManifestBytes(data []byte) (PluginManifest, error) {
	return ParsePluginManifestJSON(bytes.NewReader(data))
}

func LoadPluginManifestFile(path string) (PluginManifest, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return PluginManifest{}, fmt.Errorf("manifest not found: %w", err)
		}
		return PluginManifest{}, err
	}
	defer f.Close()

	return ParsePluginManifestJSON(f)
}

func LoadPluginManifestDir(pluginDir string) (PluginManifest, error) {
	return LoadPluginManifestFile(filepath.Join(pluginDir, "manifest.json"))
}
