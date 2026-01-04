# Project Constraints

These constraints are binding across all sprints unless explicitly revised.

## Determinism

- All execution decisions must be derivable from declared inputs
- No hidden state, timestamps, randomness, or environment leakage
- Hashes and traces must be reproducible

## Explicit Contracts

- Engines operate only on declared inputs and outputs
- Planning, execution, caching, recovery, and plugins are isolated concerns
- CLI is an interface layer only; it must not introduce behavior

## Safety Over Convenience

- Fail fast on invalid graphs, workspaces, or state
- Unknown flags, malformed inputs, or ambiguous state are fatal errors
- Partial success is not silently tolerated

## Recoverability

- Failures must be recordable
- Resumption must never re-execute completed tasks
- Recovery must not depend on undocumented state

## Documentation-Driven Development

- Specs define behavior
- Tests enforce contracts
- Notes document decisions and contradictions
