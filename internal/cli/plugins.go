package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"scriptweaver/internal/pluginengine"
)

// listPluginStates scans plugin directories and returns deterministic, human-readable
// status lines.
//
// Sprint-10 contract:
// - No mutation of plugin files.
// - Deterministic ordering.
//
// Interpretation:
// - A plugin is "enabled" if its manifest.json parses and validates.
// - A plugin is "disabled" if manifest.json exists but is invalid.
func listPluginStates(pluginsRoot string) ([]string, error) {
	entries, err := os.ReadDir(pluginsRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read plugins root: %w", err)
	}

	type row struct {
		sortKey string
		line    string
	}
	rows := make([]row, 0, len(entries))

	// Deterministic traversal by directory name.
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		pluginDir := filepath.Join(pluginsRoot, ent.Name())
		manifestPath := filepath.Join(pluginDir, "manifest.json")
		if _, statErr := os.Stat(manifestPath); statErr != nil {
			// Skip directories with no manifest.json (matches discovery behavior).
			continue
		}

		m, loadErr := pluginengine.LoadPluginManifestFile(manifestPath)
		if loadErr != nil {
			dir := ent.Name()
			msg := strings.TrimSpace(loadErr.Error())
			rows = append(rows, row{sortKey: "~" + dir, line: fmt.Sprintf("%s disabled %s", dir, msg)})
			continue
		}
		rows = append(rows, row{sortKey: m.PluginID, line: fmt.Sprintf("%s enabled", m.PluginID)})
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].sortKey < rows[j].sortKey })
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.line)
	}
	return out, nil
}
