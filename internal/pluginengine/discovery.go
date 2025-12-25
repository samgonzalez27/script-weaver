package pluginengine

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

const DefaultPluginsRoot = ".scriptweaver/plugins"

// Logger is the minimal logging interface used by the plugin engine.
// It is satisfied by *log.Logger and many test doubles.
type Logger interface {
	Printf(format string, args ...any)
}

type nopLogger struct{}

func (nopLogger) Printf(string, ...any) {}

func loggerOrNop(l Logger) Logger {
	if l == nil {
		return nopLogger{}
	}
	return l
}

// Registry stores successfully loaded plugin manifests.
// Order is deterministic (sorted by plugin_id).
type Registry struct {
	Manifests []PluginManifest
	ByID      map[string]PluginManifest
}

// DiscoverAndRegister scans a plugins root directory for plugin subdirectories
// (non-recursive), loads/validates manifests, and registers them.
//
// Behavior:
//   - If root does not exist: returns empty registry, no errors.
//   - Directories missing manifest.json are skipped.
//   - Invalid manifests are skipped with logged errors.
//   - Duplicate plugin IDs are rejected (later entries skipped).
//   - Final registry order is deterministic by plugin_id.
func DiscoverAndRegister(root string, log Logger) (Registry, []error) {
	log = loggerOrNop(log)

	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return Registry{ByID: map[string]PluginManifest{}}, nil
		}
		log.Printf("pluginengine: failed to read plugins root %q: %v", root, err)
		return Registry{ByID: map[string]PluginManifest{}}, []error{err}
	}

	// Deterministic discovery: sort directory entries by name.
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	reg := Registry{ByID: make(map[string]PluginManifest)}
	var errs []error

	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		pluginDir := filepath.Join(root, ent.Name())
		manifestPath := filepath.Join(pluginDir, "manifest.json")

		if _, statErr := os.Stat(manifestPath); statErr != nil {
			if os.IsNotExist(statErr) {
				// Explicitly skip directories without a manifest.json.
				continue
			}
			err := fmt.Errorf("stat manifest.json in %q: %w", pluginDir, statErr)
			log.Printf("pluginengine: %v", err)
			errs = append(errs, err)
			continue
		}

		m, loadErr := LoadPluginManifestFile(manifestPath)
		if loadErr != nil {
			log.Printf("pluginengine: invalid plugin in %q: %v", pluginDir, loadErr)
			errs = append(errs, loadErr)
			continue
		}

		if _, exists := reg.ByID[m.PluginID]; exists {
			err := fmt.Errorf("%w: %s", ErrDuplicatePluginID, m.PluginID)
			log.Printf("pluginengine: %v", err)
			errs = append(errs, err)
			continue
		}
		reg.ByID[m.PluginID] = m
	}

	reg.Manifests = make([]PluginManifest, 0, len(reg.ByID))
	for _, m := range reg.ByID {
		reg.Manifests = append(reg.Manifests, m)
	}
	// Deterministic execution/registration order: sort by plugin_id.
	sort.Slice(reg.Manifests, func(i, j int) bool { return reg.Manifests[i].PluginID < reg.Manifests[j].PluginID })

	return reg, errs
}
