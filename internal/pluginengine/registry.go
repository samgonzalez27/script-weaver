package pluginengine

import "fmt"

// RegisterManifests validates manifests and rejects duplicate plugin IDs.
// It returns a map keyed by plugin_id.
func RegisterManifests(manifests []PluginManifest) (map[string]PluginManifest, error) {
	byID := make(map[string]PluginManifest, len(manifests))
	for _, m := range manifests {
		if err := ValidatePluginManifest(m); err != nil {
			return nil, err
		}
		if _, exists := byID[m.PluginID]; exists {
			return nil, fmt.Errorf("%w: %s", ErrDuplicatePluginID, m.PluginID)
		}
		byID[m.PluginID] = m
	}
	return byID, nil
}
