# Architecture Overview &mdash; ScriptWeaver (Sprint-09 Baseline)

## Core Flow

1. **Graph Definition**

    - User provides a task graph
    - Graph is parsed, validated, normalized, and hashed

2. **Planning**

    - Incremental engine determines which tasks are eligible to run
    - Invalidated tasks are explicitly identified

3. **Execution**

    - DAG executor runs tasks deterministically
    - Execution order is derived solely from the graph and plan
    - Artifacts are produced relative to the workspace

4. **Caching**

    - Outputs are cached by content hash
    - Cached artifacts may be restored instead or re-executed

5. **Tracing**

    - Every execution produces a deterministic trace
    - Traces explain *why* tasks ran or were skipped

6. **Failure Recovery**

    - Failures are recorded with task-level granularity
    - Completed work is preserved
    - Resume eligibility is computed, not assumed

7. **Plugin Engine (Spring-09)**

    - Plugins may hook into defined lifecycle points
    - Plugins observe execution; they do not alter determinism
    - Plugin discovery and execution are explicit and controlled

## Layering

- `graph` &mdash; parsing, normalization, validation
- `dag` &mdash; execution scheduling and lifecycle
- `incremental` &mdash; invalidation and planning
- `core` &mdash; orchestration and artifact handling
- `trace` &mdash; deterministic execution logging
- `recovery` &mdash; failure recording and resume eligibility
- `pluginengine` &mdash; lifecycle extension points
- `cli` &mdash; thin interface layer (no behavior)

## Non-Goals

- Interactive workflows
- Dynamic graph mutation
- Implicit state inference
- Magic defaults or heuristics