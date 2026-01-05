# Sprint 10-B Backlog

## 1. Objective
Deliver a canonical, production-ready CLI for ScriptWeaver (`sw`) that strictly exposes the features of the deterministic engine (Incremental Execution, Tracing, Recovery, Plugin Architecture) without introducing new abstractions, interactive modes, or nondeterminism.

## 2. Scope

### In-Scope
*   **Canonical Binary**: `cmd/sw`
*   **Commands**: `run`, `validate`, `hash`, `plugins`
*   **Features**:
    *   Incremental execution (default) and Clean mode
    *   Deterministic tracing (`--trace`)
    *   Execution recovery (`--resume`)
    *   Plugin discovery (`--plugin-dir`)
    *   Graph hashing (flag-independent)
*   **Structure**: Rigid separation between `cmd/sw` (entrypoint), `internal/cli` (orchestration), and `internal/engine` (read-only).

### Out-Scope
*   Interactive mode or watch commands
*   Modifications to `internal/engine`
*   New engine features not already present in Sprint 09
*   Parallel execution (CLI uses serial executor)

## 3. Constraints
*   **Determinism**: `sw hash` must depend *only* on the graph content. `--workdir` must be ignored for hashing.
*   **Strictness**: Unknown flags must result in Exit Code 2. No fuzzy matching.
*   **Safety**: Engine code must remain untouched. The CLI is an adapter only.
*   **Fidelity**: Test contracts in `tdd.md` are binding.

## 4. Key Deliverables
1.  Compiled `sw` binary.
2.  Integration test suite verifying CLI contracts.
3.  Verification report confirming no regression of engine behavior.

## 5. References
*   [Specification](../planning/spec.md)
*   [TDD Contracts](../planning/tdd.md)
*   [Planning Doc](../planning/planning.md)
*   [Data Dictionary](../planning/data-dictionary.md)
