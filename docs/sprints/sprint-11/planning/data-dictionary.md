# Sprint 11 Data Dictionary

## New Structures

### `EngineConfig`
Configuration object for initializing the `Engine` facade.

| Field | Type | Description |
|-------|------|-------------|
| `CacheDir` | `string` | Path to the artifact cache directory. |
| `WorkDir` | `string` | Base working directory for execution. |
| `Concurrency` | `int` | Max parallel tasks. |
| `TraceOutput` | `io.Writer` | Destination for trace logs (file or stderr). |

### `RunOptions`
Options passed to `Engine.Run`.

| Field | Type | Description |
|-------|------|-------------|
| `Force` | `bool` | If true, ignore cache and force re-execution. |
| `Strict` | `bool` | If true, fail on warnings. |
| `Tags` | `[]string` | Filter tasks by tags. |

## Modified Structures

### `RunMetadata` (Persisted State)
The JSON structure stored in `.scriptweaver/runs/<id>/meta.json`.

| Field | Type | Description |
|-------|------|-------------|
| `ID` | `string` | Unique Run ID. |
| `StartTime` | `time.Time` | When the run started. |
| `Status` | `string` | `pending`, `success`, `failed`. |
| **`GraphPath`** | **`string`** | **(NEW) Absolute path to the graph file used for this run.** |
| **`GraphHash`** | **`string`** | **(NEW) Content hash of the graph file (for integrity check).** |

## Terminology

- **Facade:** A design pattern used here to provide a simplified interface to the complex subsystem of the execution engine.
- **Headless Resume:** Resuming a workflow run without needing to explicitly provide all original arguments (like the graph path), as the engine recalls them from persisted metadata.
