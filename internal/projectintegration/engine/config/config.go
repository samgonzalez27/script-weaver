package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config is the integration-specific configuration loaded from
// <projectRoot>/.scriptweaver/config.json.
//
// Strictness: Only graph_path is permitted. Any other field causes an error.
//
// Determinism: No environment variables and no global config locations are used.
// The only config location is .scriptweaver/config.json under the project root.
type Config struct {
	GraphPath string
}

var (
	ErrInvalidConfig = errors.New("invalid integration config")
)

// Parse parses and validates integration config JSON.
//
// Allowed fields:
// - graph_path (string, non-empty)
//
// Rejected fields (explicit):
// - workspace_path
// - semantic_overrides
//
// Any unknown field is rejected.
func Parse(data []byte) (Config, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return Config{}, fmt.Errorf("%w: parse json: %v", ErrInvalidConfig, err)
	}

	var cfg Config
	for key, value := range raw {
		switch key {
		case "graph_path":
			var s string
			if err := json.Unmarshal(value, &s); err != nil {
				return Config{}, fmt.Errorf("%w: graph_path must be a string", ErrInvalidConfig)
			}
			s = strings.TrimSpace(s)
			if s == "" {
				return Config{}, fmt.Errorf("%w: graph_path must be non-empty", ErrInvalidConfig)
			}
			cfg.GraphPath = s
		case "workspace_path":
			return Config{}, fmt.Errorf("%w: workspace_path is not permitted", ErrInvalidConfig)
		case "semantic_overrides":
			return Config{}, fmt.Errorf("%w: semantic_overrides are not permitted", ErrInvalidConfig)
		default:
			return Config{}, fmt.Errorf("%w: unknown field %q", ErrInvalidConfig, key)
		}
	}

	return cfg, nil
}

// LoadOptional loads .scriptweaver/config.json from the given project root.
//
// If the config file is missing, it returns (Config{}, false, nil).
// If present, it parses strictly and returns (cfg, true, nil) or an error.
func LoadOptional(projectRoot string) (Config, bool, error) {
	if strings.TrimSpace(projectRoot) == "" {
		return Config{}, false, fmt.Errorf("%w: project root is required", ErrInvalidConfig)
	}

	path := filepath.Join(projectRoot, ".scriptweaver", "config.json")
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, false, nil
		}
		return Config{}, false, fmt.Errorf("read config: %w", err)
	}

	cfg, err := Parse(b)
	if err != nil {
		return Config{}, true, err
	}
	return cfg, true, nil
}
