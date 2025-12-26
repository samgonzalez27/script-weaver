# Sprint-10 Completion Summary

**Sprint Goal:** Expose stabilized engine capabilities through an expanded, explicit CLI interface suitable for local use and CI environments.

**Status:** APPROVED
**Frozen:** Yes
**Closed:** Yes

## Deliverables

### CLI Expansion
The following commands have been implemented and verified against `spec.md`:

- **`scriptweaver validate`**
  - Flags: `--graph` (required), `--strict` (default: false)
  - Behavior: Validates graph schema without side effects.

- **`scriptweaver run`**
  - Flags:
    - `--workdir`, `--graph`, `--cache-dir`, `--output-dir` (required)
    - `--trace` (default: false)
    - `--mode=clean|incremental` (default: clean)
    - `--plugins` (default: none)
  - Behavior: Executes graphs deterministically, creating a new run record.

- **`scriptweaver resume`**
  - Flags:
    - `--workdir`, `--graph`, `--previous-run-id` (required)
    - `--retry-failed-only` (default: false)
  - Behavior: Links to a prior run. Enforces graph hash matching between the provided graph and the previous run (introduced as a spec correction to handle missing persistence).

- **`scriptweaver plugins list`**
  - Behavior: Lists discovered plugins and their enabled/disabled status deterministically without mutation.

## Scope & Behavior
- **Engine Integrity:** No new engine behavior or graph hashing changes were introduced.
- **Determinism:** CLI flag parsing is strict, and flag order does not affect execution.
- **Exit Codes:** Strictly mapped to 0 (Success), 1 (Validation), 2 (Workspace), 3 (Execution).

## Spec Corrections
One spec correction was required and verified:
- `resume` command now requires `--graph` argument to facilitate graph loading, as the engine does not persist the source graph definition. Validation ensures the provided graph matches the hash of the previous run.

Sprint-10 is now closed.
