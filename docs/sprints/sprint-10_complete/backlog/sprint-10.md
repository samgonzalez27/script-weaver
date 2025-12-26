# Sprint-10 Backlog & Deferred Work

This document captures items deferred during Sprint-10 to preserve scope, as well as structural observations to inform Sprint-11.

## Deferred Items

### 1. Resume Persistence Gaps
- **Context:** The `resume` command currently forces the user to re-supply the `--graph` path because the engine does not persist the graph source path or definition in the run metadata.
- **Impact:** This creates a slightly ergonomic friction (user must remember which graph file produced run X).
- **Status:** Resolved via spec correction (requiring `--graph`) for Sprint-10.
- **Future:** Sprint-11 should consider whether run metadata should persist the graph source path or a copy of the graph definition to enable "headless" resume.

### 2. Trace Output Flexibility
- **Context:** The `--trace` flag is currently a simple boolean that emits JSON to stderr.
- **Impact:** Useful for CI, but less flexible for local debugging where a file output might be preferred without shell redirection.
- **Future:** Consider adding `--trace-out=<path>`.

### 3. Warnings visibility in `validate`
- **Context:** `validate --strict` is implemented, but the current graph loader does not expose a warnings channel, rendering the flag effectively a no-op regarding behavior change (it always succeeds if no errors).
- **Future:** Loop back to the core engine team to bubble up schema warnings (e.g., deprecated fields) so `--strict` can be fully utilized.

## Structural Observations

### CLI / Engine Boundary
- The `GraphExecutor` interface in the CLI package proved effective for testing but highlighted that the CLI is manually wiring up several components (`FailureRecorder`, `TraceEmitter`, `Cache`).
- **Note:** As the engine features grow, this wiring code in `executor.go` (CLI layer) may become fragile. Consider a higher-level "Engine Facade" in `internal/core` to centralize this wiring for all consumers (CLI, Server, TUI).

### Plugin Discovery
- Plugin discovery logic is currently invoked directly by the CLI. If a future "Project" entity is introduced, plugin discovery should likely be owned by the Project loader, not the CLI entrypoint.
