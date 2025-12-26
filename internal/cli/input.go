package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

const (
	// Sprint-10 exit codes (spec.md)
	ExitSuccess         = 0
	ExitValidationError = 1
	ExitWorkspaceError  = 2
	ExitExecutionError  = 3
)

type Command string

const (
	CommandValidate Command = "validate"
	CommandRun      Command = "run"
	CommandResume   Command = "resume"
	CommandPlugins  Command = "plugins"
)

type ExecutionMode string

const (
	ExecutionModeClean       ExecutionMode = "clean"
	ExecutionModeIncremental ExecutionMode = "incremental"
)

type ValidateInvocation struct {
	GraphPath string
	Strict    bool
}

type RunInvocation struct {
	WorkDir      string
	GraphPath    string
	CacheDir     string
	OutputDir    string
	Mode         ExecutionMode
	Trace        bool
	PluginsAllow []string
}

type ResumeInvocation struct {
	WorkDir         string
	GraphPath       string
	PreviousRunID   string
	RetryFailedOnly bool
}

type PluginsInvocation struct {
	Subcommand string
}

// CLIInvocation is the canonical, parsed Sprint-10 invocation.
//
// It contains exactly one active subcommand configuration.
type CLIInvocation struct {
	Command  Command
	Validate ValidateInvocation
	Run      RunInvocation
	Resume   ResumeInvocation
	Plugins  PluginsInvocation
}

type InvocationError struct {
	ExitCode int
	Message  string
}

func (e *InvocationError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func invalidInvocationf(format string, args ...any) error {
	return &InvocationError{ExitCode: ExitValidationError, Message: fmt.Sprintf(format, args...)}
}

// ParseInvocation parses CLI flags into a canonical CLIInvocation.
func ParseInvocation(args []string) (CLIInvocation, error) {
	if len(args) == 0 {
		return CLIInvocation{}, invalidInvocationf("missing subcommand")
	}

	sub := strings.TrimSpace(args[0])
	rest := args[1:]

	switch Command(sub) {
	case CommandValidate:
		fs := flag.NewFlagSet("scriptweaver validate", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		var graphPath string
		var strict bool
		fs.StringVar(&graphPath, "graph", "", "Path to graph definition. Required.")
		fs.BoolVar(&strict, "strict", false, "Fail validation on warnings.")
		if err := fs.Parse(rest); err != nil {
			return CLIInvocation{}, invalidInvocationf("%v", err)
		}
		if fs.NArg() != 0 {
			return CLIInvocation{}, invalidInvocationf("unexpected positional arguments: %q", strings.Join(fs.Args(), " "))
		}
		if strings.TrimSpace(graphPath) == "" {
			return CLIInvocation{}, invalidInvocationf("--graph is required")
		}
		return CLIInvocation{Command: CommandValidate, Validate: ValidateInvocation{GraphPath: filepath.Clean(graphPath), Strict: strict}}, nil

	case CommandRun:
		fs := flag.NewFlagSet("scriptweaver run", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		var workDir string
		var graphPath string
		var cacheDir string
		var outputDir string
		var trace bool
		var mode string
		var pluginsCSV string

		fs.StringVar(&workDir, "workdir", "", "Workspace directory. Required.")
		fs.StringVar(&graphPath, "graph", "", "Graph source path. Required.")
		fs.StringVar(&cacheDir, "cache-dir", "", "Cache directory. Required.")
		fs.StringVar(&outputDir, "output-dir", "", "Output directory. Required.")
		fs.BoolVar(&trace, "trace", false, "Enable verbose execution tracing.")
		fs.StringVar(&mode, "mode", string(ExecutionModeClean), "Execution mode: clean|incremental")
		fs.StringVar(&pluginsCSV, "plugins", "", "Comma-separated allowlist of plugin IDs.")

		if err := fs.Parse(rest); err != nil {
			return CLIInvocation{}, invalidInvocationf("%v", err)
		}
		if fs.NArg() != 0 {
			return CLIInvocation{}, invalidInvocationf("unexpected positional arguments: %q", strings.Join(fs.Args(), " "))
		}
		workDirAbs, err := cleanAbsPath(workDir)
		if err != nil {
			return CLIInvocation{}, err
		}
		if strings.TrimSpace(graphPath) == "" {
			return CLIInvocation{}, invalidInvocationf("--graph is required")
		}
		if strings.TrimSpace(cacheDir) == "" {
			return CLIInvocation{}, invalidInvocationf("--cache-dir is required")
		}
		if strings.TrimSpace(outputDir) == "" {
			return CLIInvocation{}, invalidInvocationf("--output-dir is required")
		}

		parsedMode, err := parseExecutionMode(mode)
		if err != nil {
			return CLIInvocation{}, err
		}

		resolvedGraph, err := resolveUnderWorkDir(workDirAbs, graphPath)
		if err != nil {
			return CLIInvocation{}, err
		}
		resolvedCache, err := resolveUnderWorkDir(workDirAbs, cacheDir)
		if err != nil {
			return CLIInvocation{}, err
		}
		resolvedOutput, err := resolveUnderWorkDir(workDirAbs, outputDir)
		if err != nil {
			return CLIInvocation{}, err
		}

		return CLIInvocation{Command: CommandRun, Run: RunInvocation{
			WorkDir:      workDirAbs,
			GraphPath:    resolvedGraph,
			CacheDir:     resolvedCache,
			OutputDir:    resolvedOutput,
			Mode:         parsedMode,
			Trace:        trace,
			PluginsAllow: splitCSV(pluginsCSV),
		}}, nil

	case CommandResume:
		fs := flag.NewFlagSet("scriptweaver resume", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		var workDir string
		var graphPath string
		var previousRunID string
		var retryFailedOnly bool
		fs.StringVar(&workDir, "workdir", "", "Workspace directory. Required.")
		fs.StringVar(&graphPath, "graph", "", "Graph source path. Required.")
		fs.StringVar(&previousRunID, "previous-run-id", "", "Identifier of prior run. Required.")
		fs.BoolVar(&retryFailedOnly, "retry-failed-only", false, "Only re-execute failed work from prior run.")
		if err := fs.Parse(rest); err != nil {
			return CLIInvocation{}, invalidInvocationf("%v", err)
		}
		if fs.NArg() != 0 {
			return CLIInvocation{}, invalidInvocationf("unexpected positional arguments: %q", strings.Join(fs.Args(), " "))
		}
		workDirAbs, err := cleanAbsPath(workDir)
		if err != nil {
			return CLIInvocation{}, err
		}
		if strings.TrimSpace(graphPath) == "" {
			return CLIInvocation{}, invalidInvocationf("--graph is required")
		}
		if strings.TrimSpace(previousRunID) == "" {
			return CLIInvocation{}, invalidInvocationf("--previous-run-id is required")
		}
		resolvedGraph, err := resolveUnderWorkDir(workDirAbs, graphPath)
		if err != nil {
			return CLIInvocation{}, err
		}
		return CLIInvocation{Command: CommandResume, Resume: ResumeInvocation{
			WorkDir:         workDirAbs,
			GraphPath:       resolvedGraph,
			PreviousRunID:   strings.TrimSpace(previousRunID),
			RetryFailedOnly: retryFailedOnly,
		}}, nil

	case CommandPlugins:
		if len(rest) == 0 {
			return CLIInvocation{}, invalidInvocationf("missing plugins subcommand")
		}
		if len(rest) != 1 {
			return CLIInvocation{}, invalidInvocationf("unexpected positional arguments: %q", strings.Join(rest, " "))
		}
		sub2 := strings.TrimSpace(rest[0])
		if sub2 != "list" {
			return CLIInvocation{}, invalidInvocationf("unknown plugins subcommand %q", sub2)
		}
		return CLIInvocation{Command: CommandPlugins, Plugins: PluginsInvocation{Subcommand: sub2}}, nil
	default:
		return CLIInvocation{}, invalidInvocationf("unknown subcommand %q", sub)
	}
}

func parseExecutionMode(raw string) (ExecutionMode, error) {
	n := strings.ToLower(strings.TrimSpace(raw))
	switch ExecutionMode(n) {
	case ExecutionModeClean, ExecutionModeIncremental:
		return ExecutionMode(n), nil
	case "":
		return "", invalidInvocationf("--mode is required")
	default:
		return "", invalidInvocationf("invalid --mode %q (expected clean|incremental)", raw)
	}
}

func resolveUnderWorkDir(workDir, p string) (string, error) {
	if strings.TrimSpace(p) == "" {
		return "", invalidInvocationf("path must not be empty")
	}
	clean := filepath.Clean(p)
	if clean == "." {
		return "", invalidInvocationf("path must not be '.'")
	}

	// If absolute, accept as-is; it is still deterministic.
	// If relative, resolve under WorkDir.
	if filepath.IsAbs(clean) {
		return clean, nil
	}

	// WorkDir is required to be absolute, so Join does not consult process CWD.
	return filepath.Clean(filepath.Join(workDir, clean)), nil
}

func cleanAbsPath(p string) (string, error) {
	clean := filepath.Clean(strings.TrimSpace(p))
	if clean == "" {
		return "", invalidInvocationf("--workdir is required")
	}
	if filepath.IsAbs(clean) {
		return clean, nil
	}
	abs, err := filepath.Abs(clean)
	if err != nil {
		return "", fmt.Errorf("resolve --workdir: %w", err)
	}
	return abs, nil
}

func splitCSV(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, p := range parts {
		id := strings.TrimSpace(p)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// ExitCode extracts a semantic exit code from a ParseInvocation error.
// If the error is not a known invocation error, it returns ExitExecutionError.
func ExitCode(err error) int {
	var invErr *InvocationError
	if errors.As(err, &invErr) && invErr != nil {
		if invErr.ExitCode != 0 {
			return invErr.ExitCode
		}
		return ExitValidationError
	}
	if err == nil {
		return ExitSuccess
	}
	return ExitExecutionError
}
