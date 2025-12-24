# Sprint-08 &mdash; Failure Recovery & Resume Semantics

## Objective

Introduce deterministic, resumable execution semantics that allow ScriptWeaver to recover from partial failures without re-executing completed work or corrupting state.

This sprint formalizes **what it means to fail, where execution may resume**, and **how retries behave deterministically**.

## Why This Sprint Exists

ScriptWeaver already has:

* Deterministic graph definitions
* Stable hashing
* A controlled workspace
* Trace and cache primitives

What is missing is **explicit failure semantics**.

Without this sprint:

* Failures force full re-runs
* CI/CD usage is weak
* Partial progress is indistinguishable from corruption

Sprint-08 turns ScriptWeaver into a **recoverable execution engine**.

## Non-Goals

* No plugin system
* No dynamic graph mutation
* No distributed execution
* No UX polish or CLI redesign

## High-Level Capabilities

* Resume execution from last valid checkpoint
* Deterministic retry behavior
* Explicit failure classification
* Workspace-encoded execution state

## Deliverables

* Frozen execution state model
* Resume eligibility rules
* Failure classification taxonomy
* Deterministic retry contract
* TDD coverage for all failure paths