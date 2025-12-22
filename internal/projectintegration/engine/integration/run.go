package integration

import (
	"errors"
	"fmt"
	"strings"

	"scriptweaver/internal/projectintegration/engine/config"
	"scriptweaver/internal/projectintegration/engine/discovery"
	"scriptweaver/internal/projectintegration/engine/workspace"
)

// Result describes the resolved integration inputs for an execution.
type Result struct {
	ProjectRoot string
	Workspace   workspace.Workspace
	Config      config.Config
	GraphPath   string
}

// Run orchestrates the deterministic integration flow:
// Init Workspace -> Load Config -> Discover Graph.
//
// If projectRoot is empty, the working directory is used.
// If cliGraphPath is empty and config.json provides graph_path, that value is
// treated as the explicit graph path for discovery.
//
// If sandboxGuard is true, the orchestration verifies that no regular files
// outside .scriptweaver/ were added/removed/modified during the flow.
func Run(projectRoot, cliGraphPath string, sandboxGuard bool) (Result, error) {
	root := strings.TrimSpace(projectRoot)
	if root == "" {
		wd, err := workspace.DetectProjectRoot()
		if err != nil {
			return Result{}, err
		}
		root = wd
	}

	var before map[string]fileSnapshot
	if sandboxGuard {
		s, err := snapshotOutsideWorkspace(root)
		if err != nil {
			return Result{}, fmt.Errorf("sandbox snapshot(before): %w", err)
		}
		before = s
	}

	ws, err := workspace.EnsureWorkspace(root)
	if err != nil {
		return Result{}, &InvalidWorkspaceError{Err: err}
	}

	cfg, _, err := config.LoadOptional(root)
	if err != nil {
		return Result{}, &InvalidConfigError{Err: err}
	}

	explicit := strings.TrimSpace(cliGraphPath)
	if explicit == "" && strings.TrimSpace(cfg.GraphPath) != "" {
		explicit = cfg.GraphPath
	}

	graphPath, err := discovery.Discover(root, explicit)
	if err != nil {
		switch {
		case errors.Is(err, discovery.ErrAmbiguousGraphs):
			return Result{}, &AmbiguousGraphError{Err: err}
		case errors.Is(err, discovery.ErrNoGraphFound):
			return Result{}, &GraphNotFoundError{Err: err}
		default:
			return Result{}, err
		}
	}

	if sandboxGuard {
		after, err := snapshotOutsideWorkspace(root)
		if err != nil {
			return Result{}, fmt.Errorf("sandbox snapshot(after): %w", err)
		}
		if d := diffSnapshots(before, after); d != "" {
			return Result{}, &SandboxViolationError{Details: d}
		}
	}

	return Result{ProjectRoot: root, Workspace: ws, Config: cfg, GraphPath: graphPath}, nil
}
