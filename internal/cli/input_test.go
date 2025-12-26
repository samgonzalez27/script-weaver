package cli

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseInvocation_NoSubcommandFails(t *testing.T) {
	_, err := ParseInvocation(nil)
	if err == nil {
		t.Fatalf("expected error")
	}
	if ExitCode(err) != ExitValidationError {
		t.Fatalf("expected exit %d got %d", ExitValidationError, ExitCode(err))
	}
}

func TestParseInvocation_UnknownSubcommandFails(t *testing.T) {
	_, err := ParseInvocation([]string{"nope"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if ExitCode(err) != ExitValidationError {
		t.Fatalf("expected exit %d got %d", ExitValidationError, ExitCode(err))
	}
}

func TestParseInvocation_Run_DeterministicAndResolvesRelativeUnderWorkDir(t *testing.T) {
	workDir := t.TempDir()
	args := []string{
		"run",
		"--workdir", workDir,
		"--graph", "graphs/../graph.json",
		"--cache-dir", "./cache/..//cache",
		"--output-dir", "out/./",
		"--mode", "incremental",
		"--trace",
		"--plugins", "p2,p1,p1",
	}

	inv1, err := ParseInvocation(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	inv2, err := ParseInvocation(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(inv1, inv2) {
		t.Fatalf("expected deterministic parse")
	}

	if inv1.Command != CommandRun {
		t.Fatalf("expected command %q got %q", CommandRun, inv1.Command)
	}

	if inv1.Run.WorkDir != filepath.Clean(workDir) {
		t.Fatalf("workdir not canonicalized: %q", inv1.Run.WorkDir)
	}
	if inv1.Run.GraphPath != filepath.Join(workDir, "graph.json") {
		t.Fatalf("graph path not resolved: %q", inv1.Run.GraphPath)
	}
	if inv1.Run.CacheDir != filepath.Join(workDir, "cache") {
		t.Fatalf("cache dir not resolved: %q", inv1.Run.CacheDir)
	}
	if inv1.Run.OutputDir != filepath.Join(workDir, "out") {
		t.Fatalf("output dir not resolved: %q", inv1.Run.OutputDir)
	}
	if inv1.Run.Mode != ExecutionModeIncremental {
		t.Fatalf("expected mode %q got %q", ExecutionModeIncremental, inv1.Run.Mode)
	}
	if !inv1.Run.Trace {
		t.Fatalf("expected trace enabled")
	}
	if want := []string{"p2", "p1"}; !reflect.DeepEqual(inv1.Run.PluginsAllow, want) {
		t.Fatalf("plugins parsed = %#v, want %#v", inv1.Run.PluginsAllow, want)
	}
}

func TestParseInvocation_Run_Defaults(t *testing.T) {
	workDir := t.TempDir()
	inv, err := ParseInvocation([]string{
		"run",
		"--workdir", workDir,
		"--graph", "g.json",
		"--cache-dir", "cache",
		"--output-dir", "out",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inv.Run.Mode != ExecutionModeClean {
		t.Fatalf("expected default mode clean, got %q", inv.Run.Mode)
	}
	if inv.Run.Trace {
		t.Fatalf("expected default trace=false")
	}
	if inv.Run.PluginsAllow != nil {
		t.Fatalf("expected default plugins allowlist empty, got %#v", inv.Run.PluginsAllow)
	}
}

func TestParseInvocation_Validate_DefaultStrictFalse(t *testing.T) {
	inv, err := ParseInvocation([]string{"validate", "--graph", "g.json"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inv.Command != CommandValidate {
		t.Fatalf("expected command %q got %q", CommandValidate, inv.Command)
	}
	if inv.Validate.Strict {
		t.Fatalf("expected strict default false")
	}
}

func TestParseInvocation_Resume_RequiresPreviousRunID(t *testing.T) {
	workDir := t.TempDir()
	_, err := ParseInvocation([]string{"resume", "--workdir", workDir, "--graph", "g.json"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if ExitCode(err) != ExitValidationError {
		t.Fatalf("expected exit %d got %d", ExitValidationError, ExitCode(err))
	}
}
