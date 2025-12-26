# Sprint-10 — CLI Expansion & Integration

## Overview
Sprint-10 extends the ScriptWeaver CLI to expose and integrate functionality delivered in Sprints 06–09. This sprint does **not** change core engine semantics. It surfaces already-stabilized capabilities through a coherent, explicit, and user-safe CLI interface.

Primary goals:
- Make graph validation, workspace management, resume/retry, and plugins operable via CLI
- Preserve determinism and frozen contracts
- Avoid introducing new engine behavior

## Sprint Goal
Expose stabilized engine capabilities (graph contract, project integration, resume semantics, plugin system) through an expanded, explicit CLI interface suitable for local use and CI environments.

## In Scope
- CLI subcommands and flags for:
  - Graph validation (Sprint-06)
  - Project integration / workspace discovery (Sprint-07)
  - Run resume / retry selection (Sprint-08)
  - Plugin visibility and enable/disable controls (Sprint-09)
- Clear error messages and exit codes
- Backward compatibility with existing invocation style where feasible

## Out of Scope
- Dynamic plugin installation or marketplace
- Changes to graph hashing rules
- Changes to execution scheduling or concurrency

## Non-Goals
- No interactive TUI
- No remote execution features
