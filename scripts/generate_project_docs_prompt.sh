#!/usr/bin/env bash
set -e

PROJECT_DIR="docs/project"

cat <<EOF
You are the documentation agent responsible for generating the global project documentation files for a deterministic developer automation engine.

Your task:
- Create the following markdown files in ${PROJECT_DIR}:
  - vision.md
  - constraints.md
  - architecture.md

These files must be authoritative and globally read-only for all sprints. They should **never be overwritten by individual sprints**, except during rare maintenance updates (e.g., every 10 sprints).

Context for the project:

- The project is a deterministic developer automation engine, designed to guarantee **reproducible, correct actions on code and development environments**.
- Unlike AI agents, linters, or CI/CD tools, this engine is deterministic by architecture:
  - Checksum-based state tracking
  - Rule-based correction patterns
  - Diff-based local previews and rollback snapshots
  - Predictable patch output
  - Version-pinned logic
- It operates as a **full action engine**:
  - Detects issues
  - Proposes deterministic corrective actions
  - Applies actions safely
  - Reruns validations and logs outcomes
- It integrates directly into developer workflows (VS Code, CLI, pre-commit hooks, CI/CD pipelines) without being a bolt-on.
- Key value propositions:
  - Eliminates flaky builds, environment drift, misconfigured setups
  - Ensures reproducibility of developer workflows
  - Provides enterprise-ready observability, audit logs, and policy enforcement
  - Serves as a "compiler for your development workflow"

Guidelines for the files:

1. vision.md
   - Clearly explain the project purpose
   - Highlight uniqueness, deterministic guarantees, and competitive edge
   - Summarize value to developers and teams

2. constraints.md
   - Define rules and limitations of the engine
   - Include determinism constraints, reproducibility guarantees, and operational rules
   - Specify what the engine will and will not handle (e.g., external non-deterministic APIs)

3. architecture.md
   - Provide a high-level overview of system design
   - Describe core modules and their responsibilities (CLI, DAG engine, incremental engine, trace engine, plugin engine, project integration, recovery, graph-contracts, etc.)
   - Define input/output contracts, workflow of actions, and logging/replay mechanisms
   - Include diagrams or structured bullet points for clarity

Output:
- Produce **direct markdown content** suitable to save into the files above
- Do **not reference individual sprints**
- Do **not perform review or validation**; focus solely on creation
- Content should be stable, authoritative, and reusable for all sprints

EOF
