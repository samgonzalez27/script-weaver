## ExecutionTrace

Represents the canonical trace of a graph execution.

Includes:

* GraphHash
* Ordered list of TraceEvents

## TraceEvent

Represents a single logical event in execution.

Includes:

* EventType
* TaskID (if applicable)
* Reason (if applicable)

## EventType

Enumerates possible trace events.

Examples:

* TaskInvalidated
* TaskExecuted
* TaskCached
* TaskSkipped
* FailurePropagated
* ArtifactRestored

## TraceHash

A deterministic hash computed from the canonical execution trace.

## TraceEquivalence

Defines equivalence conditions between two execution traces based on:

* Structural identity
* Event ordering
* Event content