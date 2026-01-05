
# CLI ↔ Engine Notes (Sprint 10-B)

## Scope implemented
- Canonical binary: `cmd/sw`.
- Canonical CLI package: `internal/cli/sw`.
- Commands implemented (and only these): `run`, `validate`, `hash`, `plugins list`.

## Strict flag parsing
- Each command uses its own `flag.FlagSet` with `ContinueOnError`.
- Unknown flags are treated as fatal argument errors and mapped to exit code **2**.
- Positional arguments are not allowed (extra args after flags fail with exit code **2**).

## Exit codes (canonical `sw`)
- `0` success
- `1` validation error (invalid graph, cycle)
- `2` system/argument error (missing required flags, unknown flags, I/O, permission)
- `3` execution failure (one or more tasks failed)
- `4` plugin error (plugin discovery/manifest problems)

### Mapping from internal orchestration layer
The underlying orchestration layer in `internal/cli` has its own internal exit codes.
The `sw` CLI (`internal/cli/sw`) translates these to the canonical Sprint 10-B exit codes.

## Determinism contracts
- `sw hash` is computed from graph content only by calling `LoadGraphFromFile(...).Hash()`.
- `sw hash --workdir` is accepted for consistency but is **ignored** for hash computation.
- `sw run` resolves `--cache-dir` and `--output-dir` under `--workdir` when relative.

## Trace behavior
- Spec defines `--trace` as a boolean.
- When enabled, the CLI writes a deterministic trace file to `<output-dir>/trace.json`.

## Resume behavior
- Spec defines `--resume <run-id>`.
- The orchestration layer accepts an optional `ResumeRunID` and uses it for resume planning.
- `--resume` is rejected in `--mode clean`.

## Plugins
- `sw plugins list` prints plugin IDs, one per line, sorted lexicographically.
- `--plugin-dir` is optional; an empty/unspecified dir results in empty output and success.
- Any plugin manifest/discovery errors are mapped to exit code **4**.

## Repository cleanup (canonical structure)
- Removed obsolete CLI entrypoint `cmd/scriptweaver/` (Sprint 10-B: `cmd/sw` is the only CLI binary entrypoint).
- Removed stray root-level `scriptweaver` artifact file (built binary; not source).
- Removed redundant/corrupted `internal/cli/sw/main.go` (implementation lives in `internal/cli/sw/cli.go`).
