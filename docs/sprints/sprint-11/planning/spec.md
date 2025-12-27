# Sprint 11 Specifications

## 1. Engine Facade Architecture

### Problem
Currently, `internal/cli/executor.go` is responsible for:
- Instantiating the `Cache`.
- Setting up the `FailureRecorder`.
- Configuring the `TraceEmitter`.
- Wiring these into the `dag.Executor`.
- Handling the execution lifecycle.

This violates the separation of concerns; the CLI should only handle flag parsing and UI output.

### Solution: `internal/core.Engine`
Introduce a new `Engine` struct in `internal/core` (or a new `internal/engine` package if `core` is too crowded, but `core` is preferred for now) that acts as the single entry point for workflow operations.

#### Responsibilities
- **Initialization:** Accepts configuration (cache paths, concurrency settings, etc.) and wires up internal components.
- **Graph Loading:** Handles reading, parsing, and validating the graph file.
- **Execution:** Runs the workflow, managing the `dag.Executor`, `FailureRecorder`, and `TraceEmitter` internally.
- **Resumption:** Handles loading previous state and resuming execution.

#### Interface Sketch
```go
type Engine interface {
    // LoadGraph parses and validates a graph from a file.
    LoadGraph(path string) (*dag.Graph, error)

    // Run executes a graph with the given options.
    Run(ctx context.Context, graph *dag.Graph, opts RunOptions) (*RunResult, error)

    // Resume continues a previous run.
    Resume(ctx context.Context, runID string) (*RunResult, error)
}
```

## 2. Headless Resume

### Problem
When a run fails, the user must run:
`scriptweaver resume <run-id> --graph <path-to-original-graph>`

If the user forgets the graph path, they cannot resume.

### Solution
1.  **Update Run Metadata:** The `internal/core/runner.go` (or equivalent state tracker) must save the absolute path of the graph file used in the run metadata (`run.json` or similar).
2.  **Update Resume Logic:**
    - If `--graph` is provided, use it (override).
    - If `--graph` is NOT provided, look up the path in the run metadata.
    - If the file at that path is missing or changed (hash check optional but good), error out.

## 3. Trace Output Flexibility

### Problem
`--trace` currently dumps JSON to stderr. This is hard to capture cleanly if the CLI also prints logs to stderr.

### Solution
- Add `--trace-out=<file>` flag.
- If set, the `TraceEmitter` should write to the specified file instead of (or in addition to) the default stream.
- If `--trace` is used without `--trace-out`, behavior remains unchanged (stderr).
