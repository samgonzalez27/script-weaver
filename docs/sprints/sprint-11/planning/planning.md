# Sprint 11 Planning: Engine Refactor & Facade

## Sprint Goal
**Refactor the core execution logic into a unified "Engine Facade" to decouple the CLI from low-level component wiring, improve maintainability, and enable "headless" workflow resumption.**

## Context & Motivation
During Sprint 10, it was observed that the CLI package (`internal/cli`) was manually wiring up complex components (FailureRecorder, TraceEmitter, Cache, GraphExecutor). This coupling makes the CLI fragile and difficult to test. Additionally, the current `resume` functionality requires users to re-specify the graph file, which is a usability friction point.

This sprint focuses on architectural cleanup and usability improvements by introducing a proper `Engine` abstraction in `internal/core`.

## Deliverables
1.  **Engine Facade (`internal/core`):** A new high-level API that encapsulates component wiring.
2.  **CLI Refactor:** Update `internal/cli` to consume the new Engine Facade instead of manual wiring.
3.  **Headless Resume:** Persist graph source information in run metadata to allow `resume` without arguments.
4.  **Trace Output Control:** Support writing trace logs to a specific file via `--trace-out`.

## Out of Scope
- New plugin system features (deferred to future sprints).
- Major changes to the DAG scheduling algorithm itself (refactor is structural, not behavioral).
- `validate --strict` warning bubbling (deferred).

## Timeline
- **Week 1:** Core Facade implementation and CLI migration.
- **Week 2:** Resume enhancements, Trace output, and regression testing.
