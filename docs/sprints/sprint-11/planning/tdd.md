# Sprint 11 TDD & Validation Plan

## 1. Engine Facade Tests (`internal/core`)

### Unit Tests
- **`TestEngine_Initialization`**: Verify that creating an Engine with valid config succeeds and wires up sub-components (Cache, etc.) correctly.
- **`TestEngine_Run_Delegation`**: Mock the internal DAG executor and verify that `Engine.Run` correctly passes parameters and returns results.
- **`TestEngine_LoadGraph`**: Verify it correctly parses valid graph files and returns errors for invalid ones.

### Integration Tests
- **`TestEngine_EndToEnd`**: Create a real Engine instance with a temporary file cache. Run a simple 3-node graph. Verify all tasks execute.

## 2. CLI Regression Tests (`internal/cli`)

Since we are gutting the CLI's internal logic, we must ensure existing CLI tests pass.
- **`cli_test.go`**: Ensure `run` command still works exactly as before from the user's perspective.
- **`executor_test.go`**: These tests might need to be moved to `internal/core` if they were testing logic that is now in the Facade. If they test CLI flag parsing, they stay.

## 3. Resume Logic Tests

### Scenario: Headless Resume
1.  **Setup:** Create a graph `test_graph.yaml` that fails at Task B.
2.  **Action:** Run `scriptweaver run test_graph.yaml`. It fails and prints a Run ID.
3.  **Action:** Delete the local reference to the graph path in the test harness (simulate user forgetting).
4.  **Action:** Run `scriptweaver resume <run-id>` (WITHOUT `--graph`).
5.  **Assertion:** The engine locates `test_graph.yaml` from the metadata and resumes execution.
6.  **Assertion:** Task B is retried.

### Scenario: Resume with Moved Graph (Error Case)
1.  **Setup:** Run and fail as above.
2.  **Action:** Move `test_graph.yaml` to `test_graph_moved.yaml`.
3.  **Action:** Run `scriptweaver resume <run-id>`.
4.  **Assertion:** Command fails with a clear error: "Original graph file not found at ...; use --graph to specify new location."

## 4. Trace Output Tests

### Scenario: Trace to File
1.  **Action:** Run `scriptweaver run graph.yaml --trace-out=trace.json`.
2.  **Assertion:** `trace.json` exists and contains valid JSON trace events.
3.  **Assertion:** Stderr does NOT contain the trace JSON (or contains it only if `--trace` was also explicitly passed, depending on final spec decision - assume `--trace-out` implies tracing enabled).
