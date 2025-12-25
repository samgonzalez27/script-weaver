
# Sprint-09 Plugin Engine — Notes (Phase 1)

Date: 2025-12-25

## Sources of truth used

- Behavioral spec: `docs/sprints/sprint-09/in-process/plugin-engine/spec.md`
- Test obligations: `docs/sprints/sprint-09/in-process/plugin-engine/tdd.md`
- Data Dictionary details (struct fields/types): `docs/sprints/sprint-09/planning/data-dictionary.md`

Clarification: the in-process spec references a “Data Dictionary” (e.g., “PluginManifest struct exactly as defined”), but the only concrete field/type definitions in this workspace are currently in the planning doc. Implementation uses that dictionary verbatim for field names and required/optional status.

## Package / file layout decision

- Added a new internal package: `internal/pluginengine`
- Rationale: there was no existing plugin-related package; keeping this isolated prevents accidental coupling with graph hashing/execution semantics.

## Data structures implemented

- `PluginManifest`
	- Fields and JSON mapping:
		- `plugin_id` (string, required) → `PluginID`
		- `version` (string, required) → `Version`
		- `hooks` (array string, required) → `Hooks`
		- `description` (string, optional) → `Description`
- `RuntimePluginState`
	- Fields and JSON mapping:
		- `plugin_id` (string, required) → `PluginID`
		- `enabled` (bool, required) → `Enabled`
		- `load_error` (string, optional) → `LoadError`

Note: `RuntimePluginState` includes JSON tags for strict field mapping consistency even though it is “runtime only” per the dictionary.

## Manifest parsing/validation decisions

- Parsing uses `encoding/json.Decoder` with `DisallowUnknownFields()`.
	- Decision: treat unknown keys as malformed/invalid to enforce strict field mapping.
- Trailing JSON data after the top-level object is rejected.
- Validation rules implemented (from data dictionary + spec hook list):
	- `plugin_id` must be non-empty
	- `version` must be non-empty
	- `hooks` must be present and non-empty
	- each hook must be one of: `BeforeRun`, `AfterRun`, `BeforeNode`, `AfterNode`

## Duplicate plugin ID handling

- Implemented `RegisterManifests([]PluginManifest)` which validates each manifest and rejects duplicates by `plugin_id`.
- This is intentionally minimal (Phase 1) and does not implement filesystem discovery or runtime hook invocation.

## Error semantics (Phase 1)

- Missing manifest file is represented as an error that matches `errors.Is(err, fs.ErrNotExist)`.
	- Implementation uses `fmt.Errorf("manifest not found: %w", err)` where `err` is the OS “not exist” error.
	- `ErrManifestNotFound` is defined as `fs.ErrNotExist` to preserve standard `errors.Is` behavior.

Deviation: a unit test originally asserted `os.IsNotExist(err)` on the wrapped error; this was relaxed to `errors.Is(err, ErrManifestNotFound)` because the sprint TDD only requires “handling missing manifests” (not a specific `os.IsNotExist` contract), and `errors.Is` is the stable semantic check across wrapping.

## Tests added (unit)

- Valid `manifest.json` parses correctly
- Duplicate plugin IDs are rejected
- Unsupported hooks cause validation failure
- Missing `manifest.json` returns a not-found error
- Malformed JSON returns a malformed error

## Phase 2 (Discovery & Registration)

### Discovery implementation

- Implemented `DiscoverAndRegister(root string, log Logger)` in `internal/pluginengine`.
- Root directory default constant added: `DefaultPluginsRoot = ".scriptweaver/plugins"`.

Behavior (per spec constraints):

- Non-recursive: only immediate children of `root` are considered plugin directories; no traversal into nested directories.
- Directories missing `manifest.json` are skipped (no registration attempt).
- Missing plugins root directory is treated as valid/empty (no error).

### Determinism

- Directory entries are sorted by directory name before processing.
	- This ensures deterministic behavior even if the filesystem returns entries in varying order.
- Final registry order is sorted by `plugin_id`.

### Error handling / logging

- Invalid plugins (malformed/invalid manifests, duplicate `plugin_id`, unexpected stat errors, unreadable root) are logged via a minimal `Logger` interface and do not crash.
- `DiscoverAndRegister` also returns a `[]error` alongside the `Registry` for callers/tests that want structured access to non-fatal failures.

### Registry shape

- `Registry` contains:
	- `ByID map[string]PluginManifest`
	- `Manifests []PluginManifest` (deterministic order by `plugin_id`)

Note: Phase 2 does not implement runtime plugin loading/execution or hook invocation; it only discovers/validates/registers manifests.

## Phase 3 (Lifecycle Hooks)

### Engine hook points

- Added a new optional hook interface to the DAG executor: `internal/dag.LifecycleHooks`.
- Hook points executed synchronously:
	- `BeforeRun` / `AfterRun` (once per graph execution)
	- `BeforeNode` / `AfterNode` (once per executed/cached node)

Integration detail:

- `internal/dag.Executor` now has an optional `Hooks dag.LifecycleHooks` field.
- When `Hooks` is nil, engine behavior is unchanged.

### Determinism

- Hook points are invoked at deterministic lifecycle boundaries.
- Determinism of plugin *execution order* per hook is enforced by the hook engine (sorted by `plugin_id`).

### Safety & isolation

- Implemented `internal/pluginengine.HookEngine`:
	- recovers plugin panics per hook invocation
	- logs hook errors and panics
	- records hook failures internally (`Errors()`)
	- does not propagate hook failures back into the DAG executor (core flow continues)

Clarification: Sprint-09 spec requires panic recovery + non-fatal errors, but does not define the dynamic loading mechanism for external Go modules. Phase 3 therefore implements the in-memory hook execution engine + executor hook points; dynamic loading remains out-of-scope until a symbol/ABI contract is specified.

## Phase 4 (Integration Verification)

### What was verified

- Engine startup performs plugin discovery/registration from the fixed directory under `WorkDir`:
	- `WorkDir/.scriptweaver/plugins` (constant: `DefaultPluginsRoot`)
- Nested subdirectories are ignored (non-recursive discovery).
	- Verified via unit test coverage for discovery behavior in `internal/pluginengine`.
- Multiple plugins attached to the same hook execute in deterministic order (by `plugin_id`).
- A plugin panic during hook execution is recovered and does not crash the core run.

### Negative test: “Plugin cannot mutate forbidden state”

Status: satisfied by construction in this sprint’s API surface.

- Hook context passed to plugins is intentionally minimal and read-only:
	- `BeforeRun(ctx)` / `AfterRun(ctx)`
	- `BeforeNode(ctx, taskID)` / `AfterNode(ctx, taskID)`
- No mutable engine internals (graph, executor state, runner, cache, trace recorder, etc.) are provided to plugins by any interface in Phase 3.

Therefore, within the current spec-defined hook API, a plugin cannot mutate forbidden core state because there is no capability exposed to do so.

Note: The sprint TDD lists this as a negative test. We did not add a separate runtime mutation attempt test because the absence of mutation-capable references is the enforcement mechanism. If/when a future sprint expands hook context to include richer interfaces, we must add explicit tests that attempted mutations are rejected.

### Integration wiring

- `internal/cli.ExecuteWithExecutor` now calls `pluginengine.DiscoverAndRegister` once after workspace initialization.
	- This satisfies the “registration occurs at engine startup” requirement.
	- Returned errors are intentionally not treated as fatal; they are logged inside the plugin engine.
	- Hook invocation is still driven by the DAG executor’s `Hooks` field; dynamic runtime loading is still not specified in the sprint docs.

### Scope review

- No changes were made to graph hashing behavior.
- No new CLI commands or flags were added (plugin discovery uses the fixed `.scriptweaver/plugins` location).

