# Sprint-08 Summary: Failure Recovery & Resume Semantics

**Status**: CLOSED
**Date**: 2025-12-23

## Executive Summary

Sprint-08 successfully turned ScriptWeaver into a **recoverable execution engine**. We have formalized and frozen the "Resume Contract," which guarantees that execution can recover from partial failures without re-doing valid work and without corrupting the workspace.

## Frozen Contract

The following semantics are now immutable fundamental constraints of the system:

### 1. Resume = New Run
*   Resuming is **not** appending to an old run. It is a **new run** (fresh ID, fresh logs).
*   A resumable run MUST explictly link to a `previous_run_id`.
*   The system uses an "overlay" plan: it restores valid outputs from the previous run's checkpoints and executes only the remaining work.

### 2. Failure is Explicit
Four failure classes are now strictly defined and detectable:
1.  **Graph Failure**: Schema/structural issues (Never Resumable).
2.  **Workspace Failure**: Corruption/IO issues (Never Resumable).
3.  **Execution Failure**: Node process failure (Conditionally Resumable).
4.  **System Failure**: Crash/Panic (Resumable if checkpoints exist).

### 3. Checkpoint Atomicity
*   Checkpoints are written only after node success.
*   Writes are atomic (`fsync` + `rename`).
*   A checkpoint is only valid if:
    *   Node output is deterministic.
    *   Cache entry exists.
    *   Trace entry is complete.

## Capabilities Delivered

| Capability | Description | Status |
| :--- | :--- | :--- |
| **Clean Mode** | `scriptweaver run --mode=clean` ignores all state. | ✅ |
| **Incremental Mode** | `scriptweaver run` (default) auto-resumes if eligible. | ✅ |
| **Resume-Only Mode** | `scriptweaver run --mode=resume-only` fails if resume invalid. | ✅ |
| **Crash Recovery** | System can resume from a SIGTERM or power loss. | ✅ |
| **Run Linking** | Every run tracks its lineage via `previous_run_id`. | ✅ |

## Guarantees Enforced

1.  **Determinism**: Resume restores outputs *before* computing downstream hashes, ensuring hash stability.
2.  **Safety**: Invalidated or corrupted workspaces immediately force a full re-run (or failure in resume-only mode).
3.  **Bounded Retries**: A resume counts as a retry; retry counts are strictly incremented.

## Objectives Met

*   **Frozen execution state model**: JSON schema for Runs, Failures, and Checkpoints is live.
*   **Resume eligibility rules**: Enforced by the `ResumeEligibilityChecker`.
*   **Deterministic retry**: Verified by integration tests.
