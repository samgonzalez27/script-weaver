package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"scriptweaver/internal/dag"
	"scriptweaver/internal/pluginengine"
)

type pluginDiscoveryStubExecutor struct{}

func (pluginDiscoveryStubExecutor) Run(context.Context, *dag.TaskGraph, dag.TaskRunner) (*dag.GraphResult, error) {
	return &dag.GraphResult{FinalState: dag.ExecutionState{}}, nil
}

func TestExecute_DiscoversPluginsFromWorkDirDefaultRoot(t *testing.T) {
	workDir := t.TempDir()

	// Create minimal workspace + graph.
	if err := os.MkdirAll(filepath.Join(workDir, ".scriptweaver"), 0o755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}
	graphPath := filepath.Join(workDir, "graph.json")
	// Minimal graph: one task.
	graphJSON := `{"tasks":[{"name":"t1","run":"echo ok"}],"edges":[]}`
	if err := os.WriteFile(graphPath, []byte(graphJSON), 0o644); err != nil {
		t.Fatalf("write graph: %v", err)
	}

	inv := CLIInvocation{
		Command: CommandRun,
		Run: RunInvocation{
			GraphPath:    graphPath,
			WorkDir:      workDir,
			CacheDir:     filepath.Join(workDir, "cache"),
			OutputDir:    filepath.Join(workDir, "out"),
			Mode:         ExecutionModeClean,
			PluginsAllow: []string{"p1"},
		},
	}

	// Override discovery function to capture the root it is called with.
	old := discoverPlugins
	t.Cleanup(func() { discoverPlugins = old })

	var gotRoot string
	discoverPlugins = func(root string, _ pluginengine.Logger) (pluginengine.Registry, []error) {
		gotRoot = root
		return pluginengine.Registry{ByID: map[string]pluginengine.PluginManifest{}}, nil
	}

	// Use a deterministic stub executor; we only assert plugin discovery integration.
	res, err := ExecuteWithExecutor(context.Background(), inv, pluginDiscoveryStubExecutor{})
	if err != nil {
		t.Fatalf("ExecuteWithExecutor error: %v", err)
	}
	if res.ExitCode != ExitSuccess {
		t.Fatalf("unexpected exit code: %d", res.ExitCode)
	}

	wantRoot := filepath.Join(workDir, pluginengine.DefaultPluginsRoot)
	if gotRoot != wantRoot {
		t.Fatalf("discoverPlugins root = %q, want %q", gotRoot, wantRoot)
	}
}

func TestExecute_Default_NoPluginsEnabled_DoesNotDiscover(t *testing.T) {
	workDir := t.TempDir()

	// Create minimal workspace + graph.
	if err := os.MkdirAll(filepath.Join(workDir, ".scriptweaver"), 0o755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}
	graphPath := filepath.Join(workDir, "graph.json")
	graphJSON := `{"tasks":[{"name":"t1","run":"echo ok"}],"edges":[]}`
	if err := os.WriteFile(graphPath, []byte(graphJSON), 0o644); err != nil {
		t.Fatalf("write graph: %v", err)
	}

	inv := CLIInvocation{
		Command: CommandRun,
		Run: RunInvocation{
			GraphPath:    graphPath,
			WorkDir:      workDir,
			CacheDir:     filepath.Join(workDir, "cache"),
			OutputDir:    filepath.Join(workDir, "out"),
			Mode:         ExecutionModeClean,
			Trace:        false,
			PluginsAllow: nil,
		},
	}

	old := discoverPlugins
	t.Cleanup(func() { discoverPlugins = old })

	called := false
	discoverPlugins = func(string, pluginengine.Logger) (pluginengine.Registry, []error) {
		called = true
		return pluginengine.Registry{ByID: map[string]pluginengine.PluginManifest{}}, nil
	}

	res, err := ExecuteWithExecutor(context.Background(), inv, pluginDiscoveryStubExecutor{})
	if err != nil {
		t.Fatalf("ExecuteWithExecutor error: %v", err)
	}
	if res.ExitCode != ExitSuccess {
		t.Fatalf("unexpected exit code: %d", res.ExitCode)
	}
	if called {
		t.Fatalf("expected plugin discovery not to run")
	}
}
