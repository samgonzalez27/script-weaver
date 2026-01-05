# Data Dictionary

## CLI Terms

### Invocation
A single execution of the `sw` binary. Contains arguments, flags, and environment context.

### Workdir
The root directory against which all relative paths in the Graph are resolved. Use of `--workdir` limits the blast radius of execution.

### Cache Dir
The directory used by the engine to store content-addressable artifacts. Controlled via `--cache-dir`.

## Engine Terms (Canonical)

### Task Graph
The immutable definition of work. Hashed into a `GraphHash`.
*   **Constraint**: The `GraphHash` is derived ONLY from the Task Graph content. CLI flags (`workdir`, `mode`, `output-dir`) are EXCLUDED from the graph hash.

### Run ID
A unique identifier for a specific execution attempt. Used for the `--resume` functionality.

### Trace
A deterministic log of execution decisions. Generated when `--trace` is active.

### Plugin
An external binary implementing the distinct Plugin Interface. Loaded from `--plugin-dir`.