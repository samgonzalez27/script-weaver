# Sprint 10-B Planning

## Objective
Deliver a canonical, production-ready CLI for ScriptWeaver that strictly exposes the features of the deterministic engine (Sprints 06-09) without introducing new behaviors, inconsistent abstractions, or nondeterminism. The implementation must adhere to the Minimal Surface Area philosophy of Sprint 05.

## Scope (Strict)

### In-Scope
*   **CLI Implementation**: A thin adapter layer mapping user intent to engine calls.
*   **Canonical Features**: Exposure of Graph validation, Execution (Clean/Incremental), Recovery (Sprint 08), Plugin Hooks (Sprint 09), and Tracing (Sprint 03).
*   **Professional Go Structure**:
    *   `cmd/sw/`: Main entrypoint.
    *   `internal/cli/`: Flag parsing, validation, and engine orchestration.
    *   `internal/engine/`: Existing canonical engine (MUST NOT BE MODIFIED).

### Out-Scope
*   New engine features or heuristics.
*   Modifications to graph hashing or determinism.
*   Interactive modes or "watch" commands.
*   Any features from discarded Sprints 10-11 in the `experimental` folder.

## Constraints
1.  **Determinism**: Output must be byte-for-byte identical for identical inputs.
2.  **No Logic Leakage**: No business logic in `cmd/sw`. All orchestration in `internal/cli`.
3.  **Engine Sanctity**: `internal/engine` is strictly read-only.
4.  **Zero Ambiguity**: Flags, defaults, and exit codes are binding as defined in `spec.md`.

## Deliverables
1.  `cmd/sw` binary.
2.  Passing test suite ensuring CLI-to-Engine fidelity.
3.  Verification that no existing engine tests are broken.

## Acceptance Criteria
*   CLI commands `sw run`, `sw validate`, `sw hash`, `sw plugins` implemented exactly as specified.
*   All canonical features map to working CLI flags.
*   Exit codes match `spec.md` definitions.
*   Project structure follows `cmd/sw`, `internal/cli`, `internal/engine`.