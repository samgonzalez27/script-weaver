package sw

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"scriptweaver/internal/cli"
	"scriptweaver/internal/dag"
	"scriptweaver/internal/pluginengine"
)

const (
	ExitSuccess          = 0
	ExitValidationError  = 1
	ExitArgOrSystemError = 2
	ExitExecutionFailure = 3
	ExitPluginError      = 4
)

// Main is the canonical entrypoint for the `sw` CLI.
// args should exclude argv[0].
func Main(args []string, stdout, stderr io.Writer) int {
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}

	if len(args) == 0 {
		fmt.Fprintln(stderr, "missing command (expected: run|validate|hash|plugins)")
		return ExitArgOrSystemError
	}

	switch args[0] {
	case "help", "-h", "--help":
		printHelp(stdout)
		return ExitSuccess
	case "run":
		return cmdRun(args[1:], stdout, stderr)
	case "validate":
		return cmdValidate(args[1:], stdout, stderr)
	case "hash":
		return cmdHash(args[1:], stdout, stderr)
	case "plugins":
		return cmdPlugins(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n", args[0])
		return ExitArgOrSystemError
	}
}

func printHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  sw run --graph <path> --workdir <path> [--cache-dir <path>] [--output-dir <path>] [--resume <run-id>] [--plugin-dir <path>] [--trace] [--mode <clean|incremental>]")
	fmt.Fprintln(w, "  sw validate --graph <path>")
	fmt.Fprintln(w, "  sw hash --graph <path> [--workdir <path>]")
	fmt.Fprintln(w, "  sw plugins list [--plugin-dir <path>]")
}

type strictFlagSet struct {
	fs *flag.FlagSet
}

func newStrictFlagSet(command string) *strictFlagSet {
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(&bytes.Buffer{}) // discard default usage output
	return &strictFlagSet{fs: fs}
}

func (s *strictFlagSet) parse(args []string, stderr io.Writer) error {
	if err := s.fs.Parse(args); err != nil {
		msg := err.Error()
		if strings.Contains(msg, "flag provided but not defined") {
			fmt.Fprintln(stderr, "unknown flag")
		} else {
			fmt.Fprintln(stderr, msg)
		}
		return err
	}
	if s.fs.NArg() != 0 {
		fmt.Fprintf(stderr, "unexpected positional arguments: %q\n", strings.Join(s.fs.Args(), " "))
		return fmt.Errorf("unexpected positional arguments")
	}
	return nil
}

func absFromCWD(p string) (string, error) {
	if strings.TrimSpace(p) == "" {
		return "", fmt.Errorf("path is empty")
	}
	clean := filepath.Clean(p)
	if filepath.IsAbs(clean) {
		return clean, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Clean(filepath.Join(cwd, clean)), nil
}

func absUnderWorkdir(workdirAbs, p string) (string, error) {
	if strings.TrimSpace(p) == "" {
		return "", fmt.Errorf("path is empty")
	}
	clean := filepath.Clean(p)
	if filepath.IsAbs(clean) {
		return clean, nil
	}
	return filepath.Clean(filepath.Join(workdirAbs, clean)), nil
}

func isGraphValidationErr(err error) bool {
	if err == nil {
		return false
	}
	var ge *dag.GraphError
	if errors.As(err, &ge) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "parse graph") || strings.Contains(msg, "invalid task graph") || strings.Contains(msg, "cycle")
}

func isSystemPathErr(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, os.ErrNotExist) || errors.Is(err, os.ErrPermission)
}

func cmdRun(args []string, stdout, stderr io.Writer) int {
	s := newStrictFlagSet("sw run")

	var graphPath string
	var workdir string
	var cacheDir string
	var outputDir string
	var resumeID string
	var pluginDir string
	var trace bool
	var mode string

	s.fs.StringVar(&graphPath, "graph", "", "Path to the graph definition file")
	s.fs.StringVar(&workdir, "workdir", "", "Root directory for execution context")
	s.fs.StringVar(&cacheDir, "cache-dir", ".sw/cache", "Directory for deterministic artifact caching")
	s.fs.StringVar(&outputDir, "output-dir", ".sw/output", "Directory for execution outputs")
	s.fs.StringVar(&resumeID, "resume", "", "ID of a previous run to resume")
	s.fs.StringVar(&pluginDir, "plugin-dir", "", "Directory containing compiled plugins")
	s.fs.BoolVar(&trace, "trace", false, "Enable deterministic trace logging")
	s.fs.StringVar(&mode, "mode", "incremental", "Execution strategy: clean|incremental")

	if err := s.parse(args, stderr); err != nil {
		return ExitArgOrSystemError
	}
	if strings.TrimSpace(graphPath) == "" {
		fmt.Fprintln(stderr, "--graph is required")
		return ExitArgOrSystemError
	}
	if strings.TrimSpace(workdir) == "" {
		fmt.Fprintln(stderr, "--workdir is required")
		return ExitArgOrSystemError
	}

	absWorkdir, err := absFromCWD(workdir)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return ExitArgOrSystemError
	}
	absGraph, err := absFromCWD(graphPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return ExitArgOrSystemError
	}

	cacheAbs, err := absUnderWorkdir(absWorkdir, cacheDir)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return ExitArgOrSystemError
	}
	outAbs, err := absUnderWorkdir(absWorkdir, outputDir)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return ExitArgOrSystemError
	}

	var execMode cli.ExecutionMode
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "clean":
		if strings.TrimSpace(resumeID) != "" {
			fmt.Fprintln(stderr, "--resume is not compatible with --mode clean")
			return ExitArgOrSystemError
		}
		execMode = cli.ExecutionModeClean
	case "incremental", "":
		execMode = cli.ExecutionModeIncremental
	default:
		fmt.Fprintf(stderr, "invalid --mode %q (expected clean|incremental)\n", mode)
		return ExitArgOrSystemError
	}

	if strings.TrimSpace(pluginDir) != "" {
		absPluginDir, err := absFromCWD(pluginDir)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return ExitArgOrSystemError
		}
		pluginLog := log.New(stderr, "", 0)
		_, errs := pluginengine.DiscoverAndRegister(absPluginDir, pluginLog)
		if len(errs) > 0 {
			fmt.Fprintln(stderr, "plugin error")
			return ExitPluginError
		}
	}

	inv := cli.CLIInvocation{
		GraphPath:     absGraph,
		WorkDir:       absWorkdir,
		CacheDir:      cacheAbs,
		OutputDir:     outAbs,
		ExecutionMode: execMode,
		ResumeRunID:   strings.TrimSpace(resumeID),
	}
	if trace {
		inv.Trace = cli.TraceConfig{Enabled: true, Path: filepath.Join(outAbs, "trace.json")}
	}

	res, execErr := cli.Execute(context.Background(), inv)
	if execErr != nil {
		if isGraphValidationErr(execErr) {
			if errors.Is(execErr, dag.ErrCycleFound) || strings.Contains(strings.ToLower(execErr.Error()), "cycle") {
				fmt.Fprintln(stderr, "Cycle detected")
			} else {
				fmt.Fprintln(stderr, execErr)
			}
			return ExitValidationError
		}
		fmt.Fprintln(stderr, execErr)
		return ExitArgOrSystemError
	}

	switch res.ExitCode {
	case cli.ExitSuccess:
		fmt.Fprintln(stdout, "Execution succeeded")
		return ExitSuccess
	case cli.ExitGraphFailure:
		fmt.Fprintln(stderr, "Execution failed")
		return ExitExecutionFailure
	case cli.ExitInvalidInvocation:
		return ExitArgOrSystemError
	default:
		return ExitArgOrSystemError
	}
}

func cmdValidate(args []string, stdout, stderr io.Writer) int {
	s := newStrictFlagSet("sw validate")
	var graphPath string
	s.fs.StringVar(&graphPath, "graph", "", "Path to the graph definition file")
	if err := s.parse(args, stderr); err != nil {
		return ExitArgOrSystemError
	}
	if strings.TrimSpace(graphPath) == "" {
		fmt.Fprintln(stderr, "--graph is required")
		return ExitArgOrSystemError
	}

	absGraph, err := absFromCWD(graphPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return ExitArgOrSystemError
	}

	_, err = cli.LoadGraphFromFile(absGraph)
	if err == nil {
		return ExitSuccess
	}
	if isSystemPathErr(err) {
		fmt.Fprintln(stderr, err)
		return ExitArgOrSystemError
	}
	if errors.Is(err, dag.ErrCycleFound) || strings.Contains(strings.ToLower(err.Error()), "cycle") {
		fmt.Fprintln(stderr, "Cycle detected")
		return ExitValidationError
	}
	fmt.Fprintln(stderr, err)
	return ExitValidationError
}

func cmdHash(args []string, stdout, stderr io.Writer) int {
	s := newStrictFlagSet("sw hash")
	var graphPath string
	var _workdir string
	s.fs.StringVar(&graphPath, "graph", "", "Path to the graph definition file")
	s.fs.StringVar(&_workdir, "workdir", "", "Accepted but ignored")
	if err := s.parse(args, stderr); err != nil {
		return ExitArgOrSystemError
	}
	if strings.TrimSpace(graphPath) == "" {
		fmt.Fprintln(stderr, "--graph is required")
		return ExitArgOrSystemError
	}

	absGraph, err := absFromCWD(graphPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return ExitArgOrSystemError
	}

	g, err := cli.LoadGraphFromFile(absGraph)
	if err != nil {
		if isSystemPathErr(err) {
			fmt.Fprintln(stderr, err)
			return ExitArgOrSystemError
		}
		fmt.Fprintln(stderr, err)
		return ExitValidationError
	}
	fmt.Fprintln(stdout, g.Hash().String())
	return ExitSuccess
}

func cmdPlugins(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "missing plugins subcommand (expected: list)")
		return ExitArgOrSystemError
	}
	switch args[0] {
	case "list":
		return cmdPluginsList(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown plugins subcommand: %s\n", args[0])
		return ExitArgOrSystemError
	}
}

func cmdPluginsList(args []string, stdout, stderr io.Writer) int {
	s := newStrictFlagSet("sw plugins list")
	var pluginDir string
	s.fs.StringVar(&pluginDir, "plugin-dir", "", "Directory containing compiled plugins")
	if err := s.parse(args, stderr); err != nil {
		return ExitArgOrSystemError
	}
	if strings.TrimSpace(pluginDir) == "" {
		return ExitSuccess
	}

	absPluginDir, err := absFromCWD(pluginDir)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return ExitArgOrSystemError
	}

	reg, errs := pluginengine.DiscoverAndRegister(absPluginDir, log.New(stderr, "", 0))
	if len(errs) > 0 {
		fmt.Fprintln(stderr, "plugin error")
		return ExitPluginError
	}

	ids := make([]string, 0, len(reg.Manifests))
	for _, m := range reg.Manifests {
		ids = append(ids, m.PluginID)
	}
	sort.Strings(ids)
	for _, id := range ids {
		fmt.Fprintln(stdout, id)
	}
	return ExitSuccess
}
