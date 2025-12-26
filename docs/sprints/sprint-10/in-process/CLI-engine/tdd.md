# Sprint-10 — CLI Test Plan

## CLI Validation Tests

- Running without subcommand fails
- Unknown subcommand fails
- Missing required flags fail with stable error messages

## validate

- Valid graph exits 0
- Invalid graph exits 1
- No workspace artifacts created
- Default behavior: warnings do not cause failure (strict=false)

## run

- Clean run creates workspace
- Incremental run reuses cache
- Plugin allowlist respected
- Default behavior: mode=clean
- Default behavior: no plugins enabled

## resume

- Resume requires valid previous_run_id and graph
- Resume fails if graph hash differs from previous run (exit code 1)
- New run links to prior run
- Resume without prior run fails
- Default behavior: retry-failed-only=false (resumes all pending)

## plugins

- Lists discovered plugins deterministically
- Disabled plugins are marked
