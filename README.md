# ScriptWeaver

ScriptWeaver is a **deterministic task execution engine** designed to make complex build, data, and automation workflows **predictable, auditable, and replayable**.

Unlike general-purpose task runners, ScriptWeaver prioritizes correctness and reproducibility above all else. It ensures that identical inputs always produce identical outputs, enabling safe content-addressable caching and reliable failure recovery.

## Current Status

**Active Sprint: 10-B (Completed)**

The core engine is fully implemented, featuring incremental execution, DAG resolution, deterministic recovery, plugin hooks, and a canonical CLI.

## Key Features

- **Strict Determinism**: Tasks run in isolated environments. Inputs, outputs, and environment variables are explicitly controlled.
- **Incremental Execution**: Only re-executes tasks when inputs change. Uses content hashing rather than timestamps.
- **Execution Recovery**: Automatically resume failed workflows from the last successful checkpoint (`--resume`).
- **Deterministic Tracing**: Produces a byte-for-byte reproducible JSON trace of every execution decision.
- **Plugin System**: lifecycle hooks to extend behavior without modifying the core engine.
- **Canonical CLI**: A strict, minimal command-line interface `sw`.

## Installation

Prerequisites: Go 1.22+

```bash
git clone https://github.com/samgonzalez27/script-weaver.git
cd script-weaver
go build -o sw ./cmd/sw
```

## Usage

ScriptWeaver uses a strict CLI (`sw`). All paths must be explicit.

### Run a Graph
Execute tasks defined in a graph file.

```bash
./sw run --graph ./graphs/build.json --workdir $(pwd)
```

**Flags**:
- `--graph <path>`: (Required) Path to graph definition.
- `--workdir <path>`: (Required) Absolute root directory for execution.
- `--mode <clean|incremental>`: Execution strategy (default: `incremental`).
- `--resume <run-id>`: Resume a specific failed run ID.
- `--trace`: Enable deterministic trace logging.
- `--plugin-dir <path>`: Load plugins from directory.

### Validate a Graph
Check schema and cycle detection without running tasks.

```bash
./sw validate --graph ./graphs/build.json
```

### Compute Graph Hash
Print the canonical structural hash of the graph.

```bash
./sw hash --graph ./graphs/build.json
```

### Manage Plugins
List available plugins in deterministic order.

```bash
./sw plugins list --plugin-dir ./plugins
```

## Project Structure

```
script-weaver/
├── cmd/sw/               # Canonical CLI entrypoint
├── internal/
│   ├── cli/              # CLI orchestration and logic
│   ├── engine/           # The core deterministic engine (Read-Only)
│   ├── dag/              # Graph processing and scheduling
│   ├── pluginengine/     # Plugin discovery and hook execution
│   └── recovery/         # State management and failure recording
├── docs/sprints/         # Detailed planning and summary docs
└── go.mod
```

## Sprint History

- **Sprint 10-B**: Canonical CLI Implementation (`sw run`, `sw validate`, `sw hash`).
- **Sprint 09**: Plugin System foundations and lifecycle hooks.
- **Sprint 08**: Execution Recovery and state persistence.
- **Sprint 06**: Graph Hashing and structural validation.
- **Sprint 05**: "Minimal Surface Area" philosophy adopted.
- **Sprint 02-04**: Incremental execution and DAG engine.
- **Sprint 00-01**: Project foundation.

## Documentation

Detailed architectural notes, specifications, and sprint summaries are located in `docs/sprints/`. See `docs/project/` for high-level Vision and Constraints.

## License

MIT License.
