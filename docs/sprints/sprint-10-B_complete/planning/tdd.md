# Test-Driven Development Contracts

## 1. CLI Integrity Tests
These tests verify the CLI layer itself.

### Test: Clean Run Default
*   **Command**: `sw run --graph fixtures/basic.json --workdir ./tmp`
*   **Input**: Valid graph, clean workspace.
*   **Expectation**:
    *   Exit Code: 0
    *   Output: Contains "Execution succeeded"
    *   Artifacts: created in `./tmp`

### Test: Validation Failure
*   **Command**: `sw validate --graph fixtures/cyclic.json`
*   **Input**: Graph with a dependency cycle.
*   **Expectation**:
    *   Exit Code: 1
    *   Stderr: Contains "Cycle detected"

### Test: Unknown Flag Strictness
*   **Command**: `sw run --graph fixtures/basic.json --workdir ./tmp --random-flag`
*   **Expectation**:
    *   Exit Code: 2
    *   Stderr: Contains "unknown flag"

## 2. Canonical Feature Verification
These tests verify variables mapped from the specification.

### Test: Hash Stability (Sprint 06)
*   **Command**: `sw hash --graph fixtures/basic.json`
*   **Iteration 1 Output**: `HASH_A`
*   **Iteration 2 Output**: `HASH_A`
*   **Constraint**: Output must be identical across runs and environments. Changing `--workdir` must NOT change the hash.

### Test: Execution Recovery (Sprint 08)
*   **Setup**: Run `sw run ...` where a task fails. Capture Run ID `<RUN_ID>`.
*   **Command**: `sw run --graph ... --resume <RUN_ID>`
*   **Expectation**:
    *   Engine skips previously successful tasks.
    *   Trace indicates "Resuming from <RUN_ID>".

### Test: Plugin Loading (Sprint 09)
*   **Command**: `sw plugins list --plugin-dir ./plugins`
*   **Input**: Directory with 2 plugins [Alpha, Beta].
*   **Expectation**:
    *   Output contains exactly "Alpha" and "Beta" on separate lines.
    *   Order is alphabetical (Alpha then Beta).

## 3. Regression Suite
The existing engine test suite (`internal/engine/...`) MUST be invoked and PASS completely. The CLI implementation must not require any changes to existing engine tests.