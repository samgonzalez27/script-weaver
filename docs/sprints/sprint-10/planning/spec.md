# Sprint-10 — CLI Specification

## CLI Structure

The CLI introduces explicit subcommands:

- `scriptweaver validate`
- `scriptweaver run`
- `scriptweaver resume`
- `scriptweaver plugins`

No positional arguments outside subcommands are allowed.

## validate

Purpose: Validate graph definition without executing.

Required flags:
- `--graph`

Optional flags:
- `--strict` (default: `false`, fail on warnings)

Behavior:
- Parses and validates graph schema
- Exits non-zero on failure
- Produces no workspace side effects

## run

Purpose: Execute a graph in a workspace.

Required flags:
- `--workdir`
- `--graph`
- `--cache-dir`
- `--output-dir`

Optional flags:
- `--trace` (default: `false`)
- `--mode=clean|incremental` (default: `clean`)
- `--plugins` (default: none, comma-separated allowlist)

Behavior:
- Creates or validates workspace
- Executes graph deterministically
- Registers a new run

## resume

Purpose: Resume a previous run.

Required flags:
- `--workdir`
- `--previous-run-id`

Optional flags:
- `--retry-failed-only` (default: `false`)

Behavior:
- Loads prior run state
- Creates a new run linked via `previous_run_id`

## plugins

Purpose: Inspect plugin state.

Subcommands:
- `plugins list`

Behavior:
- Reads plugin manifests
- Displays enabled/disabled status
- No mutation of plugin files

## Exit Codes

- `0`: success
- `1`: validation error
- `2`: workspace error
- `3`: execution error
