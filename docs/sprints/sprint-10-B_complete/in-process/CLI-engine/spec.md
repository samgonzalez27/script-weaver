# Specification

## 1. CLI Command Set
The CLI supports ONLY the following commands. No aliases, no hidden commands.

### `sw run`
Executes a task graph deterministically.

*   **Required Flags**:
    *   `--graph <path>`: Path to the graph definition file (JSON/YAML).
    *   `--workdir <path>`: Root directory for execution context.

*   **Optional Flags**:
    *   `--cache-dir <path>`: Directory for deterministic artifact caching. Default: `.sw/cache`.
    *   `--output-dir <path>`: Directory for execution outputs. Default: `.sw/output`.
    *   `--resume <run-id>`: ID of a previous run to resume (Sprint 08 Recovery).
    *   `--plugin-dir <path>`: Directory containing compiled plugins (Sprint 09).
    *   `--trace`: Enable deterministic trace logging (Sprint 03).
    *   `--mode <clean|incremental>`: Execution strategy. Default: `incremental`.

### `sw validate`
Validates graph schema and semantic integrity without execution.

*   **Required Flags**:
    *   `--graph <path>`: Path to the graph definition file.

*   **Behavior**:
    *   Performs structural validation.
    *   Performs cycle detection.
    *   No side effects on filesystem (except logging if configured).

### `sw hash`
Computes and prints the canonical structural hash of the graph.

*   **Required Flags**:
    *   `--graph <path>`: Path to the graph definition file.

*   **Behavior**:
    *   Output is the raw hex string of the hash.
    *   **Crucial**: CLI flags (like workspace or cache dir) MUST NOT affect the graph hash.

*   **Optional Flags**:
    *   `--workdir <path>`: Root directory for execution context. Accepted for consistency but **explicitly ignored** for hash computation.

### `sw plugins`
Manage and inspect plugins.

*   **Subcommands**:
    *   `list`: List available plugins in the strictly defined order.
        *   **Optional Flags**:
            *   `--plugin-dir <path>`: Directory containing compiled plugins. Default: None. 
        *   **Output Format**:
            *   Plain text, one plugin name per line.
            *   Sorted alphabetically (lexicographical usage order).
            *   No headers, no versions, only the plugin ID/Name.

## 2. Exit Codes
The CLI must return these exact codes:

*   **0**: Success (Execution completed, Validation passed).
*   **1**: Validation Error (Invalid graph, Cycle detected).
*   **2**: System Error (Missing workspace, Permission denied, Invalid arguments).
*   **3**: Execution Failure (One or more tasks failed).
*   **4**: Plugin Error (Failed to load, Incompatible interface).

## 3. Canonical Feature Mapping
The CLI explicitly exposes engine features defined in previous sprints:

| Engine Feature | Sprint | CLI Manifestation |
| :--- | :--- | :--- |
| **Incremental Execution** | Sprint 02/04 | `sw run --mode incremental` |
| **Deterministic Tracing** | Sprint 03 | `sw run --trace` |
| **Graph Schema Validation** | Sprint 06 | `sw validate --graph <file>` |
| **Execution Recovery** | Sprint 08 | `sw run --resume <id>` |
| **Plugin Architecture** | Sprint 09 | `sw run --plugin-dir <dir>` |
| **Graph Hashing** | Sprint 06 | `sw hash --graph <file>` |

## 4. Determinism Contracts
*   **Input Stability**: `sw run` with identical inputs (graph content, workspace content) must produce identical outputs and traces.
*   **Hash Stability**: `sw hash` must depend ONLY on the graph content.
*   **Flag Independence**: Flag parsing must be strict. Unknown flags result in Exit Code 2.
