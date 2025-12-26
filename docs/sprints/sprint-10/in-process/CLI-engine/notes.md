
# Sprint-10 CLI Implementation Notes

## Source-of-truth paths

- The specification paths referenced in the sprint instructions used `cli-engine/` (lowercase).
- The repository uses `docs/sprints/sprint-10/in-process/CLI-engine/` (capitalized).
- Implementation uses the capitalized folder as the authoritative location.

## Exit code mapping

- Implemented exactly as in `spec.md`:
	- `0` success
	- `1` validation error
	- `2` workspace error
	- `3` execution error
- There is no separate "internal" exit code; panics and unexpected engine errors map to exit code `3`.

## Flag parsing + determinism

- Uses strict parsing via per-subcommand `flag.FlagSet` with `ContinueOnError` and suppressed flag package output.
- Unknown flags are fatal parse errors (exit code `1`).
- Positional arguments beyond the subcommand structure are rejected (exit code `1`).

## `validate`

- `--strict` is accepted and defaults to `false`.
- The current graph loader does not expose a warnings channel; therefore `--strict` does not change outcomes today.
- `validate` does not call workspace initialization and produces no `.scriptweaver` artifacts.

## `run --trace`

- Sprint-10 defines `--trace` as a boolean flag (no output path).
- Implementation emits the canonical trace JSON (already produced deterministically by the DAG executor) to **stdout/stderr boundary** as follows:
	- Trace JSON is written to `stderr` (one JSON document + newline) when `--trace=true`.
	- When execution panics or fails before a trace exists, an empty trace `{graphHash, events: []}` is emitted.

## `run --plugins`

- `--plugins` is parsed as a comma-separated allowlist of plugin IDs.
- Default behavior is "no plugins enabled": when the allowlist is empty, execution does not perform plugin discovery.
- Plugin execution is not modified; the CLI only controls discovery/enabling decisions.

## `resume`

- Spec-required behavior implemented:
	- Validates that `--graph` hash matches the stored `graph_hash` for `--previous-run-id`; mismatch returns exit code `1`.
	- Creates a new run linked via `previous_run_id`.
- `resume` does not accept `--cache-dir` or `--output-dir` per `spec.md`.
	- Output directory clearing is therefore skipped during `resume`.
	- When `--retry-failed-only=true`, resume attempts cache reuse using the canonical workspace cache at `<workdir>/.scriptweaver/cache`.
	- When `--retry-failed-only=false` (default), resume executes in `clean` mode.
- Rationale: the persistent run metadata does not store the original CLI `cache-dir`/`output-dir`, so resume must derive cache behavior from the `.scriptweaver` workspace.

## `plugins list`

- `plugins list` scans `<cwd>/.scriptweaver/plugins`.
- A plugin is reported as:
	- `enabled` if `manifest.json` parses and validates.
	- `disabled` if `manifest.json` exists but is invalid (includes error message).
- Output ordering is deterministic.

