package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"scriptweaver/internal/core"
	"scriptweaver/internal/dag"
	"scriptweaver/internal/graph"
	"scriptweaver/internal/incremental"
	"scriptweaver/internal/pluginengine"
	"scriptweaver/internal/projectintegration/engine/workspace"
	"scriptweaver/internal/recovery/state"
	"scriptweaver/internal/trace"
)

var discoverPlugins = pluginengine.DiscoverAndRegister

// GraphExecutor is the minimal engine interface the CLI wires into.
//
// This allows the CLI to prove exit-code mapping (including panic) in tests
// without depending on specific executor internals.
type GraphExecutor interface {
	Run(ctx context.Context, graph *dag.TaskGraph, runner dag.TaskRunner) (*dag.GraphResult, error)
}

type defaultGraphExecutor struct{}

func (defaultGraphExecutor) Run(ctx context.Context, graph *dag.TaskGraph, runner dag.TaskRunner) (*dag.GraphResult, error) {
	exec, err := dag.NewExecutor(graph, runner)
	if err != nil {
		return nil, err
	}
	return exec.RunSerial(ctx)
}

type cliGraphExecutor struct {
	Plan     *incremental.IncrementalPlan
	Observer dag.NodeObserver
}

func (c cliGraphExecutor) Run(ctx context.Context, graph *dag.TaskGraph, runner dag.TaskRunner) (*dag.GraphResult, error) {
	exec, err := dag.NewExecutor(graph, runner)
	if err != nil {
		return nil, err
	}
	exec.Plan = c.Plan
	exec.Observer = c.Observer
	return exec.RunSerial(ctx)
}

type CLIResult struct {
	ExitCode    int
	GraphResult *dag.GraphResult
}

// Execute is the default entrypoint for running a canonical invocation.
func Execute(ctx context.Context, inv CLIInvocation) (CLIResult, error) {
	return ExecuteWithExecutor(ctx, inv, defaultGraphExecutor{})
}

// Execute maps a canonical CLIInvocation to engine execution.
//
// Responsibilities:
//   - Prepare OutputDir using the Overwrite policy (no stale files).
//   - Select cache strategy based on ExecutionMode.
//   - Initialize trace output before execution and finalize after execution,
//     even on panic/failure.
//   - Translate engine outcomes to semantic exit codes.

func ExecuteWithExecutor(ctx context.Context, inv CLIInvocation, executor GraphExecutor) (res CLIResult, execErr error) {
	switch inv.Command {
	case CommandValidate:
		return executeValidate(inv.Validate)
	case CommandRun:
		return executeRun(ctx, inv.Run, executor)
	case CommandResume:
		return executeResume(ctx, inv.Resume, executor)
	case CommandPlugins:
		return executePlugins(inv.Plugins)
	default:
		return CLIResult{ExitCode: ExitValidationError}, fmt.Errorf("unknown command: %q", inv.Command)
	}
}

func executeValidate(inv ValidateInvocation) (CLIResult, error) {
	_ = inv.Strict // Sprint-10: strict is accepted; graph loader currently has no warnings channel.
	if strings.TrimSpace(inv.GraphPath) == "" {
		return CLIResult{ExitCode: ExitValidationError}, fmt.Errorf("--graph is required")
	}
	_, err := LoadGraphFromFile(inv.GraphPath)
	if err != nil {
		return CLIResult{ExitCode: ExitValidationError}, err
	}
	return CLIResult{ExitCode: ExitSuccess}, nil
}

type execInvocation struct {
	WorkDir          string
	GraphPath        string
	CacheDir         string
	OutputDir        string
	Mode             ExecutionMode
	Trace            bool
	PluginsAllowlist []string

	IsResume        bool
	PreviousRunID   string
	RetryFailedOnly bool
}

func executeRun(ctx context.Context, inv RunInvocation, executor GraphExecutor) (CLIResult, error) {
	ei := execInvocation{
		WorkDir:          inv.WorkDir,
		GraphPath:        inv.GraphPath,
		CacheDir:         inv.CacheDir,
		OutputDir:        inv.OutputDir,
		Mode:             inv.Mode,
		Trace:            inv.Trace,
		PluginsAllowlist: inv.PluginsAllow,
		IsResume:         false,
	}
	return executeGraph(ctx, ei, executor)
}

func executeResume(ctx context.Context, inv ResumeInvocation, executor GraphExecutor) (CLIResult, error) {
	ei := execInvocation{
		WorkDir:         inv.WorkDir,
		GraphPath:       inv.GraphPath,
		IsResume:        true,
		PreviousRunID:   inv.PreviousRunID,
		RetryFailedOnly: inv.RetryFailedOnly,
	}
	// Sprint-10 spec does not define cache-dir/output-dir flags for resume.
	// We therefore avoid output-dir clearing and, if retry-failed-only is true,
	// reuse the canonical workspace cache at <workdir>/.scriptweaver/cache.
	if ei.RetryFailedOnly {
		ei.Mode = ExecutionModeIncremental
		ei.CacheDir = filepath.Join(ei.WorkDir, ".scriptweaver", "cache")
	} else {
		ei.Mode = ExecutionModeClean
	}
	return executeGraph(ctx, ei, executor)
}

func executePlugins(inv PluginsInvocation) (CLIResult, error) {
	if inv.Subcommand != "list" {
		return CLIResult{ExitCode: ExitValidationError}, fmt.Errorf("unknown plugins subcommand %q", inv.Subcommand)
	}
	root, err := os.Getwd()
	if err != nil {
		return CLIResult{ExitCode: ExitWorkspaceError}, fmt.Errorf("detect workdir: %w", err)
	}
	pluginsRoot := filepath.Join(root, pluginengine.DefaultPluginsRoot)
	entries, err := listPluginStates(pluginsRoot)
	if err != nil {
		return CLIResult{ExitCode: ExitWorkspaceError}, err
	}
	for _, e := range entries {
		fmt.Fprintln(os.Stdout, e)
	}
	return CLIResult{ExitCode: ExitSuccess}, nil
}

func executeGraph(ctx context.Context, inv execInvocation, executor GraphExecutor) (res CLIResult, execErr error) {
	res.ExitCode = ExitExecutionError
	if executor == nil {
		return res, fmt.Errorf("nil executor")
	}

	// Initialize recovery store as early as possible so failures can be recorded.
	st, _ := state.NewStore(inv.WorkDir)
	rec := &state.FailureRecorder{Store: st}
	runID, _ := rec.NewRunID()

	// Best-effort: validate/init .scriptweaver workspace; even if this fails,
	// we still attempt to record a WorkspaceFailure.
	_, wsErr := workspace.EnsureWorkspace(inv.WorkDir)
	if wsErr != nil {
		if runID != "" {
			_ = rec.StartRun(state.Run{RunID: runID, GraphHash: "", StartTime: time.Now().UTC(), Mode: state.ExecutionMode(inv.Mode), RetryCount: 0, Status: "failed", PreviousRunID: nil})
			_ = rec.RecordFailure(runID, &state.WorkspaceFailureError{Code: "WorkspaceInvalid", Message: wsErr.Error(), Cause: wsErr})
		}
		res.ExitCode = ExitWorkspaceError
		return res, wsErr
	}

	// Plugin discovery is deterministic and non-recursive.
	// Sprint-10: default behavior is no plugins enabled; therefore we only
	// perform discovery during execution if an allowlist was explicitly provided.
	if len(inv.PluginsAllowlist) > 0 {
		pluginsRoot := filepath.Join(inv.WorkDir, pluginengine.DefaultPluginsRoot)
		pluginLog := log.New(os.Stderr, "", 0)
		_, _ = discoverPlugins(pluginsRoot, pluginLog)
	}

	graphObj, graphHash, err := loadGraphAndHash(inv.GraphPath)
	if err != nil {
		if runID != "" {
			_ = rec.StartRun(state.Run{RunID: runID, GraphHash: "", StartTime: time.Now().UTC(), Mode: state.ExecutionMode(inv.Mode), RetryCount: 0, Status: "failed", PreviousRunID: nil})
			var se *graph.SchemaError
			var ste *graph.StructuralError
			switch {
			case errors.As(err, &se):
				_ = rec.RecordFailure(runID, &state.GraphFailureError{Code: "SchemaViolation", Message: err.Error(), Cause: err})
			case errors.As(err, &ste):
				_ = rec.RecordFailure(runID, &state.GraphFailureError{Code: "StructuralInvalidity", Message: err.Error(), Cause: err})
			default:
				_ = rec.RecordFailure(runID, &state.GraphFailureError{Code: "GraphLoadError", Message: err.Error(), Cause: err})
			}
		}
		res.ExitCode = ExitValidationError
		return res, err
	}

	// Resume mode: validate previous run and graph hash up-front.
	var previousRunID *string
	retryCount := 0
	if inv.IsResume {
		if strings.TrimSpace(inv.PreviousRunID) == "" {
			res.ExitCode = ExitValidationError
			return res, fmt.Errorf("--previous-run-id is required")
		}
		prev, perr := st.LoadRun(inv.PreviousRunID)
		if perr != nil {
			res.ExitCode = ExitValidationError
			return res, fmt.Errorf("previous run not found: %w", perr)
		}
		if prev.GraphHash != graphHash {
			res.ExitCode = ExitValidationError
			return res, fmt.Errorf("graph hash mismatch for previous run")
		}
		id := inv.PreviousRunID
		previousRunID = &id
		retryCount = prev.RetryCount + 1
	}

	traceEmitter := newTraceEmitter(inv.Trace, os.Stderr, graphHash)
	defer func() {
		_ = traceEmitter.Finalize(res.GraphResult)
	}()

	if strings.TrimSpace(inv.OutputDir) != "" {
		if err := prepareOutputDir(inv.OutputDir); err != nil {
			if runID != "" {
				_ = rec.RecordFailure(runID, &state.WorkspaceFailureError{Code: "OutputDir", Message: err.Error(), Cause: err})
			}
			res.ExitCode = ExitWorkspaceError
			return res, err
		}
	}

	cache, err := cacheForMode(inv.Mode, inv.CacheDir)
	if err != nil {
		if runID != "" {
			_ = rec.RecordFailure(runID, &state.WorkspaceFailureError{Code: "CacheDir", Message: err.Error(), Cause: err})
		}
		res.ExitCode = ExitWorkspaceError
		return res, err
	}

	runner := core.NewRunner(inv.WorkDir, cache)
	cacheRunner, err := dag.NewCacheAwareRunner(runner)
	if err != nil {
		res.ExitCode = ExitExecutionError
		return res, err
	}

	// Create a checkpoint observer. Checkpoints are only meaningful for incremental/resume.
	var obs dag.NodeObserver
	if runID != "" && inv.Mode == ExecutionModeIncremental {
		validator := &state.CheckpointValidator{Store: st, Cache: cache, Harvester: core.NewHarvester(inv.WorkDir)}
		obs = checkpointObserver{RunID: runID, Validator: validator}
	}

	// Resume planning (resume only): best-effort attempt to reuse prior work.
	var executorToUse GraphExecutor = executor
	var resumePlan *incremental.IncrementalPlan
	if inv.IsResume && inv.RetryFailedOnly {
		if _, ferr := st.LoadFailure(inv.PreviousRunID); ferr != nil {
			res.ExitCode = ExitValidationError
			return res, fmt.Errorf("previous run has no recorded failure")
		}
		checkpoints, cerr := st.LoadAllCheckpoints(inv.PreviousRunID)
		if cerr != nil {
			res.ExitCode = ExitWorkspaceError
			return res, cerr
		}
		if len(checkpoints) == 0 {
			res.ExitCode = ExitValidationError
			return res, fmt.Errorf("previous run has no checkpoints")
		}
		plan, _, _, _, perr := buildResumePlan(ctx, graphObj, runner, cacheRunner, cache, checkpoints)
		if perr != nil {
			res.ExitCode = ExitWorkspaceError
			return res, perr
		}
		resumePlan = plan
	}

	// Record the run metadata now that we know GraphHash and any run linkage.
	if runID != "" {
		_ = rec.StartRun(state.Run{RunID: runID, GraphHash: graphHash, StartTime: time.Now().UTC(), Mode: state.ExecutionMode(inv.Mode), RetryCount: retryCount, Status: "running", PreviousRunID: previousRunID})
	}

	defer func() {
		if r := recover(); r != nil {
			res.ExitCode = ExitExecutionError
			res.GraphResult = nil
			execErr = fmt.Errorf("panic: %v", r)
			if runID != "" {
				_ = rec.RecordFailure(runID, &state.SystemFailureError{Code: "Panic", Message: fmt.Sprintf("panic: %v", r), Cause: execErr})
			}
		}
	}()

	// If the caller provided the default executor, always run through the CLI-owned executor
	// so we can attach checkpoint observer.
	if _, ok := executor.(defaultGraphExecutor); ok {
		executorToUse = cliGraphExecutor{Plan: resumePlan, Observer: obs}
	}

	gr, err := executorToUse.Run(ctx, graphObj, cacheRunner)
	if err != nil {
		if runID != "" {
			_ = rec.RecordFailure(runID, &state.SystemFailureError{Code: "EngineError", Message: err.Error(), Cause: err})
		}
		res.ExitCode = ExitExecutionError
		return res, err
	}
	res.GraphResult = gr
	res.ExitCode = translateGraphResultToExitCode(gr)
	if res.ExitCode == ExitExecutionError && runID != "" {
		// Deterministically choose a representative failed node.
		failed := firstFailedNode(gr)
		_ = rec.RecordFailure(runID, &state.ExecutionFailureError{NodeID: failed, Code: "NodeFailed", Message: fmt.Sprintf("node %s failed", failed)})
	}
	return res, nil
}

type checkpointObserver struct {
	RunID     string
	Validator *state.CheckpointValidator
}

func (o checkpointObserver) OnTaskTerminal(task core.Task, result *dag.NodeResult, traceEvents []trace.TraceEvent) error {
	if o.RunID == "" {
		return fmt.Errorf("checkpoint observer: run id is empty")
	}
	if o.Validator == nil {
		return fmt.Errorf("checkpoint observer: validator is nil")
	}
	if result == nil {
		return fmt.Errorf("checkpoint observer: nil result")
	}
	if result.ExitCode != 0 {
		return nil
	}
	if task.Name == "" {
		return fmt.Errorf("checkpoint observer: task name is empty")
	}
	_, err := o.Validator.CreateAndSave(state.CheckpointInput{
		RunID:           o.RunID,
		NodeID:          task.Name,
		When:            time.Now().UTC(),
		TaskHash:        result.Hash,
		DeclaredOutputs: task.Outputs,
		ExitCode:        result.ExitCode,
		FromCache:       result.FromCache,
		TraceEvents:     traceEvents,
	})
	return err
}

func detectPreviousRunID(st *state.Store, graphHash string) (string, error) {
	if st == nil {
		return "", fmt.Errorf("nil store")
	}
	if graphHash == "" {
		return "", fmt.Errorf("graph hash is empty")
	}
	ids, err := st.ListRunIDs()
	if err != nil {
		return "", err
	}
	// Resume is only meaningful after a non-successful termination.
	// Prefer the most recent run with matching graph hash that has a persisted failure.
	var bestID string
	var bestTime time.Time
	for _, id := range ids {
		r, err := st.LoadRun(id)
		if err != nil {
			continue
		}
		if r.GraphHash != graphHash {
			continue
		}
		if _, ferr := st.LoadFailure(id); ferr != nil {
			continue
		}
		if bestID == "" || r.StartTime.After(bestTime) || (r.StartTime.Equal(bestTime) && r.RunID < bestID) {
			bestID = r.RunID
			bestTime = r.StartTime
		}
	}
	return bestID, nil
}

func buildResumePlan(ctx context.Context, g *dag.TaskGraph, runner *core.Runner, restoreRunner interface {
	Restore(ctx context.Context, task core.Task) (*dag.NodeResult, error)
}, cache core.Cache, checkpoints map[string]state.Checkpoint) (*incremental.IncrementalPlan, string, *incremental.GraphSnapshot, incremental.InvalidationMap, error) {
	if g == nil {
		return nil, "", nil, nil, fmt.Errorf("nil graph")
	}
	if runner == nil {
		return nil, "", nil, nil, fmt.Errorf("nil runner")
	}
	if cache == nil {
		return nil, "", nil, nil, fmt.Errorf("nil cache")
	}

	order := g.TopologicalOrder()
	upstream := make(map[string][]string, len(order))
	for _, e := range g.Edges() {
		upstream[e.To] = append(upstream[e.To], e.From)
	}
	for k := range upstream {
		sort.Strings(upstream[k])
	}

	invMap := make(incremental.InvalidationMap, len(order))
	snap := &incremental.GraphSnapshot{Nodes: make(map[string]incremental.NodeSnapshot, len(order))}

	computedHash := make(map[string]core.TaskHash, len(order))
	canReuse := make(map[string]bool, len(order))
	restored := make(map[string]bool, len(order))

	plan := &incremental.IncrementalPlan{Order: append([]string(nil), order...), Decisions: make(map[string]incremental.NodeExecutionDecision, len(order))}
	for _, name := range order {
		n, _ := g.Node(name)
		// Populate snapshot for eligibility checks (only Upstream is used today).
		snap.Nodes[name] = incremental.NodeSnapshot{Name: name, Upstream: append([]string(nil), upstream[name]...)}

		// If we plan to reuse upstream tasks, restore their outputs before hashing this task's inputs.
		for _, p := range upstream[name] {
			if plan.Decisions[p] != incremental.DecisionReuseCache {
				continue
			}
			if restored[p] {
				continue
			}
			if restoreRunner == nil {
				return nil, "", nil, nil, fmt.Errorf("restore runner is required to build resume plan after output dir was cleared")
			}
			pn, _ := g.Node(p)
			res, err := restoreRunner.Restore(ctx, pn.Task)
			if err != nil {
				return nil, "", nil, nil, err
			}
			if res == nil || res.ExitCode != 0 {
				return nil, "", nil, nil, fmt.Errorf("restoring %q for resume plan failed", p)
			}
			restored[p] = true
		}

		h, err := computeTaskHash(runner, n.Task)
		if err != nil {
			return nil, "", nil, nil, err
		}
		computedHash[name] = h

		cp, ok := checkpoints[name]
		if !ok || !cp.Valid {
			invMap[name] = incremental.InvalidationEntry{Invalidated: false, Reasons: nil}
			canReuse[name] = false
			plan.Decisions[name] = incremental.DecisionExecute
			continue
		}
		// Checkpoint invalidation marker: task hash mismatch.
		invalidated := false
		if len(cp.CacheKeys) == 0 || cp.CacheKeys[0] == "" {
			invalidated = true
		} else if cp.CacheKeys[0] != h.String() {
			invalidated = true
		}
		invMap[name] = incremental.InvalidationEntry{Invalidated: invalidated, Reasons: nil}
		if invalidated {
			canReuse[name] = false
			plan.Decisions[name] = incremental.DecisionExecute
			continue
		}
		exists, err := cache.Has(h)
		if err != nil {
			return nil, "", nil, nil, err
		}
		if !exists {
			return nil, "", nil, nil, fmt.Errorf("cache entry missing for checkpointed task %q", name)
		}
		canReuse[name] = true

		allUpstreamReuse := true
		for _, p := range upstream[name] {
			if plan.Decisions[p] != incremental.DecisionReuseCache {
				allUpstreamReuse = false
				break
			}
		}
		if allUpstreamReuse {
			plan.Decisions[name] = incremental.DecisionReuseCache
			if !restored[name] {
				if restoreRunner == nil {
					return nil, "", nil, nil, fmt.Errorf("restore runner is required to build resume plan after output dir was cleared")
				}
				res, err := restoreRunner.Restore(ctx, n.Task)
				if err != nil {
					return nil, "", nil, nil, err
				}
				if res == nil || res.ExitCode != 0 {
					return nil, "", nil, nil, fmt.Errorf("restoring %q for resume plan failed", name)
				}
				restored[name] = true
			}
		} else {
			plan.Decisions[name] = incremental.DecisionExecute
		}
	}

	checkpointNode := ""
	for _, name := range order {
		if plan.Decisions[name] == incremental.DecisionReuseCache {
			checkpointNode = name
			continue
		}
		break
	}
	if checkpointNode == "" {
		return nil, "", snap, invMap, nil
	}
	return plan, checkpointNode, snap, invMap, nil
}

func computeTaskHash(r *core.Runner, task core.Task) (core.TaskHash, error) {
	if r == nil {
		return "", fmt.Errorf("nil runner")
	}
	inputSet, err := r.Resolver.Resolve(task.Inputs)
	if err != nil {
		return "", fmt.Errorf("resolving inputs: %w", err)
	}
	hashInput := core.HashInput{Inputs: inputSet, Command: task.Run, Env: task.Env, Outputs: task.Outputs, WorkingDir: r.WorkingDir}
	return r.Hasher.ComputeHash(hashInput), nil
}

func firstFailedNode(gr *dag.GraphResult) string {
	if gr == nil || len(gr.FinalState) == 0 {
		return ""
	}
	names := make([]string, 0, len(gr.FinalState))
	for n := range gr.FinalState {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		if gr.FinalState[n] == dag.TaskFailed {
			return n
		}
	}
	return ""
}

func translateGraphResultToExitCode(gr *dag.GraphResult) int {
	if gr == nil {
		return ExitExecutionError
	}
	for _, st := range gr.FinalState {
		if st == dag.TaskFailed {
			return ExitExecutionError
		}
	}
	return ExitSuccess
}

func cacheForMode(mode ExecutionMode, cacheDir string) (core.Cache, error) {
	switch mode {
	case ExecutionModeIncremental:
		if cacheDir == "" {
			return nil, fmt.Errorf("cache dir is empty")
		}
		if err := os.MkdirAll(cacheDir, 0o755); err != nil {
			return nil, fmt.Errorf("create cache dir: %w", err)
		}
		return core.NewFileCache(cacheDir), nil
	case ExecutionModeClean:
		return noCache{}, nil
	default:
		return nil, fmt.Errorf("unknown execution mode: %q", mode)
	}
}

type noCache struct{}

func (noCache) Has(core.TaskHash) (bool, error)             { return false, nil }
func (noCache) Get(core.TaskHash) (*core.CacheEntry, error) { return nil, nil }
func (noCache) Put(*core.CacheEntry) error                  { return nil }

func prepareOutputDir(dir string) error {
	if dir == "" {
		return fmt.Errorf("output dir is empty")
	}
	clean := filepath.Clean(dir)
	if clean == "/" {
		return fmt.Errorf("refusing to operate on output dir '/' ")
	}
	info, err := os.Stat(clean)
	if err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(clean, 0o755)
		}
		return fmt.Errorf("stat output dir: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("output dir is not a directory: %s", clean)
	}
	entries, err := os.ReadDir(clean)
	if err != nil {
		return fmt.Errorf("read output dir: %w", err)
	}
	for _, e := range entries {
		p := filepath.Join(clean, e.Name())
		if err := os.RemoveAll(p); err != nil {
			return fmt.Errorf("clear output dir: %w", err)
		}
	}
	return nil
}

func loadGraphAndHash(path string) (*dag.TaskGraph, string, error) {
	g, err := LoadGraphFromFile(path)
	if err != nil {
		return nil, "", err
	}
	return g, g.Hash().String(), nil
}

type traceFileWriter struct {
	enabled   bool
	w         io.Writer
	graphHash string
}

func newTraceEmitter(enabled bool, w io.Writer, graphHash string) *traceFileWriter {
	if !enabled {
		return &traceFileWriter{enabled: false}
	}
	if w == nil {
		w = io.Discard
	}
	return &traceFileWriter{enabled: true, w: w, graphHash: graphHash}
}

func (w *traceFileWriter) Finalize(gr *dag.GraphResult) error {
	if w == nil || !w.enabled {
		return nil
	}
	// Emit canonical JSON trace bytes if available; otherwise, emit an empty trace.
	if gr != nil && len(gr.TraceBytes) > 0 {
		_, err := w.w.Write(append(gr.TraceBytes, '\n'))
		return err
	}
	b, err := trace.ExecutionTrace{GraphHash: w.graphHash, Events: nil}.CanonicalJSON()
	if err != nil {
		return err
	}
	_, err = w.w.Write(append(b, '\n'))
	return err
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	tmp, err := os.CreateTemp(dir, base+".tmp.*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}()

	if _, err := tmp.Write(data); err != nil {
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		return err
	}
	_ = tmp.Sync() // best-effort durability
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
