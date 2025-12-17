
## Notes — Deterministic Trace Engine (Sprint 03)

### Scope and Authority

This design and implementation is derived strictly from:

- `docs/sprints/sprint-03/in-process/trace-engine/spec.md`
- `docs/sprints/sprint-03/in-process/trace-engine/tdd.md`

No assumptions were imported from other sprints.

### Canonical Trace Model

Implemented a minimal, rigid model in `internal/trace`:

- `ExecutionTrace`
	- `GraphHash` (string)
	- `Events` (ordered list of `TraceEvent`)

- `TraceEvent`
	- `Kind` (stable string discriminator)
	- `TaskID` (task/node identifier)
	- `Reason` (stable reason code; optional)
	- `CauseTaskID` (upstream task identifier; optional)
	- `Artifacts` (sorted list of stable artifact identifiers; optional)

This satisfies the spec’s constraints:

- Captures graph identity and an ordered list of events.
- Events represent logical decisions/transitions (not implementation details).
- Excludes runtime-dependent values: timestamps, pointers, map iteration order, wall-clock durations.

### Cache vs Execute (Incremental & Cache State Representation)

#### Distinction

Sprint-03 requires the trace to model the distinction between:

- `TaskExecuted`: the task performed fresh work (its command was executed)
- `TaskCached`: the task was **not executed** because cached results were reused

The trace records these as logical decisions, not a runtime timeline.

#### “Why was this task not executed?”

The requirement says the trace must explicitly show why a task was not executed (e.g., cache hit or invalidation reason).

Within this sprint’s authority, the DAG executor has access to:

- cache-hit decisions (`Runner.Probe`)
- incremental decisions (`IncrementalPlan.Decisions`)

but it does not currently receive a per-node invalidation reason map at execution time. Therefore, the implementation explains non-execution via stable `reason` codes on `TaskCached`:

- `reason = "CacheHit"` when cache is reused due to a deterministic probe hit
- `reason = "PlannedReuseCache"` when cache reuse is chosen by an incremental plan

Skip propagation is also explicit and stable:

- `TaskSkipped` uses `reason = "UpstreamFailed"` and `causeTaskId = <failing task>`

### Deterministic Failure Propagation (Failures + Skips)

#### Required Behavior

- If task A fails and task B depends on A, the trace must include a `TaskSkipped` event for B.
- The set and order of skipped tasks must be identical across repeated runs, independent of concurrency and “early cancellation”.

#### Implementation Strategy

- On each committed task failure, emit `TaskFailed` for the failing task.
- The executor’s failure propagation is deterministic (reachability traversal in canonical index order).
- `TaskSkipped` events are **deferred** and emitted at the end from a derived map of `skippedTask -> causeTask`.

Deferring skip event emission avoids locking-in a non-deterministic cause when multiple failures happen concurrently.

#### Race-to-Failure Edge Case

If two distinct branches fail concurrently and both could explain why a downstream node is skipped, the trace must resolve this to a stable representation.

This implementation resolves “race to failure” deterministically by choosing:

- `causeTaskId = min( failing upstream task IDs that imply the skip )`

This makes the skip cause independent of which failure is observed first.

The trace may still contain multiple `TaskFailed` events (one per failing task), which is fine; canonical ordering guarantees stable event ordering.

If future sprint documents provide an explicit invalidation reason source at execution time, `TaskInvalidated` can be emitted with that reason.

#### Deterministic “Restoring from Cache” Events

The trace must represent restoration events without runtime-dependent details (like how many files had to be rewritten).

On successful cache reuse the trace emits a deterministic pair:

- `TaskCached` (with a stable `reason` as above)
- `TaskArtifactsRestored` (logical “artifacts ensured/restored”) with a stable `reason`:
	- `reason = "CacheReplay"` for probe-hit replay
	- `reason = "CacheRestore"` for incremental restore

Critically:

- `TaskExecuted` is never emitted for cached reuse.
- No artifact counts are recorded, because counts can vary with workspace state and would break trace stability.

### Deterministic Event Ordering

#### Problem

Parallel execution emits events in a non-deterministic time order. If the trace were appended “as events happen”, then:

- `Trace(ParallelRun)` would differ from `Trace(SerialRun)`
- repeated parallel runs could differ due to scheduling jitter

This violates the TDD requirement: byte-for-byte identical traces for observationally equivalent runs.

#### Requirement

Define and implement a **canonical ordering** that:

- does not depend on completion order
- is stable under concurrency
- provides a strict total order so serialization is uniquely determined

The sprint spec explicitly states:

- “Trace events must be ordered canonically.”
- “Ordering must be independent of execution timing or concurrency.”
- “Concurrent events must be sorted by Task ID to force a canonical order.”

#### Sorting Strategy Implemented

The trace engine canonicalizes by sorting **all events** using a lexicographic key:

1. `TaskID` (primary; lexicographic)
2. `Kind` precedence (fixed table, e.g. Invalidated < Restored < Cached < Executed < Failed < Skipped)
3. `Reason` (lexicographic)
4. `CauseTaskID` (lexicographic)
5. `Artifacts` (lexicographic slice compare after sorting the slice)

Additionally:

- `Artifacts` are sorted.
- Empty `Artifacts` is normalized to `nil` so empty-vs-absent cannot diverge.

This intentionally treats the trace as an **observational log** (spec: “Trace Inertness”) whose ordering is a canonicalization function over the set/multiset of events, rather than a real-time timeline.

#### Why This Guarantees a Strict Total Order

Let each event be mapped to a tuple key $K(e)$ with components listed above.

- Each component is a value from a totally ordered set:
	- strings under lexicographic order
	- integers via the fixed kind precedence table
	- slices ordered by lexicographic comparison after deterministic sorting
- Lexicographic order over tuples of totally ordered components is itself a total order.

Therefore, for any two events $e_1, e_2$:

- either $K(e_1) < K(e_2)$, $K(e_1) = K(e_2)$, or $K(e_1) > K(e_2)$

Using a stable sort on these keys yields a deterministic sequence for any given multiset of events.

#### Relationship to DAG Partial Orders

The DAG induces a partial order on *task execution feasibility*, but the sprint-03 spec does not require the trace’s event list to reflect causal/topological ordering—only that it is canonical and concurrency-independent.

This ordering strategy is a deterministic linearization that is valid for **all possible partial orders** because:

- it does not depend on which tasks happened to run first
- it is defined without referencing runtime timing or scheduling
- it yields a unique order for a fixed set of logical outcomes

If a future spec requires topological alignment (e.g., “parents must appear before children”), the ordering key can be extended with a deterministic topological rank component (e.g., depth + TaskID). That requirement is not present in the current sprint documents, so the current implementation stays minimal.

### Canonical Serialization

To enforce byte-for-byte stability, encoding uses custom JSON marshalling:

- fixed field order (`graphHash`, then `events`; for events: `kind`, then optional fields)
- omission of absent optional fields
- deterministic handling of `Artifacts` as described above

`ExecutionTrace.CanonicalJSON()` canonicalizes a copy then marshals it.

### TraceHash

The spec’s equivalence rules include `TraceHash`. Implemented:

- `ComputeTraceHash(canonicalBytes)` = `sha256(canonicalBytes)` as hex.
- `ExecutionTrace.Hash()` hashes the canonical JSON encoding (`ExecutionTrace.CanonicalJSON()`) via `ComputeTraceHash`.

#### Canonical-Order Coverage

Because the hash is computed over the canonical encoding (which itself is produced after canonical sorting + normalization), the hash covers the canonical sorted order of events, not event insertion order.

#### Equivalence Statement (What “iff” Means Here)

Within this design, two traces are treated as **semantically equivalent** iff their canonical encodings are byte-for-byte identical. Under that definition:

- `CanonicalBytes(A) == CanonicalBytes(B)` iff A and B are semantically equivalent
- `TraceHash(A) == TraceHash(B)` if A and B are semantically equivalent

The strict “iff” claim for the hash itself depends on the (standard) assumption of no hash collisions for the chosen hash function. Using sha256 makes collisions infeasible in practice and stable across architectures/compilers.

### Tests

Unit tests in `internal/trace/trace_test.go` assert:

- insertion order differences canonicalize to identical bytes
- sorting by `TaskID` is enforced
- artifacts are sorted and empty artifacts are omitted
- trace hashing is deterministic

### Final Verification Harness (Sprint-03 TDD)

Implemented the final determinism checks as integration tests in `internal/dag/trace_determinism_integration_test.go`:

- Parallelism equivalence: run the same graph with `parallelism=1` vs `parallelism=8` and assert `TraceHash` (and canonical bytes) are identical.
- Incremental stability: run a graph once to populate cache, then run in incremental reuse mode twice without changing inputs and assert the incremental run’s trace is stable across repeats.
- Delay insensitivity: inject an artificial delay into one task (then into a different task) and assert trace bytes/hash are unaffected.

These tests collectively exercise the sprint requirements that traces are:

- independent of concurrency timing
- independent of task completion order
- stable across repeated equivalent executions

### Trace Engine Integration (Core Execution Loop)

#### Where Tracing Lives

Trace generation is integrated at the DAG execution layer in `internal/dag`:

- Serial execution: `(*dag.Executor).RunSerial`
- Parallel execution: `(*dag.Executor).RunParallel`

This is the “core execution loop” for graph runs and is the correct place to emit **logical decisions** such as cached reuse, execution, failure, and skip propagation.

The produced canonical trace bytes and hash are surfaced on the deterministic run summary:

- `dag.GraphResult.TraceBytes`
- `dag.GraphResult.TraceHash`

#### Consistent Event Emission Points

The key integration requirement is that event emission must not depend on “order of completion” or timing differences between clean/incremental/replay modes.

This implementation emits events at **state-decision commit points**, not at runtime wall-clock occurrences:

- `TaskCached`
	- Default (clean/replay path): emitted exactly when the executor commits `PENDING -> CACHED` after a deterministic cache probe.
	- Incremental plan reuse: emitted exactly when the executor commits the plan decision to run a restore step (before the restore), so serial and parallel runs share the same emission point.

- `TaskArtifactsRestored`
	- Emitted when cached artifacts are successfully ensured (probe-hit replay) or restored (incremental plan restore).

- `TaskExecuted`
	- Emitted only for fresh work when a task’s successful result (`exitCode == 0`) is committed and the state transition to `COMPLETED` is applied.

- `TaskFailed`
	- Emitted when a task’s failure is committed (exit code non-zero) and failure propagation is applied.

- `TaskSkipped`
	- Emitted for each downstream node that transitions `PENDING -> SKIPPED` during deterministic failure propagation.
	- Each skip event includes `causeTaskId = <failing task>` so failure propagation is explicit and stable.

These emission points are consistent across:

- clean runs (cache probe + execute)
- replay runs (cache probe hit)
- incremental runs (precomputed plan Execute/ReuseCache)
- serial vs parallel dispatch

#### High-Concurrency Collection Without Races

Parallel execution uses worker goroutines to run tasks, but **trace recording is performed only by the coordinator goroutine** (the same goroutine that owns and mutates the executor state under `e.mu`).

This ensures:

- no data races for trace event collection
- no additional lock contention in the hot worker paths
- no changes to scheduling or dispatch order due to tracing

The only shared data structure used for trace collection is `trace.Recorder`, but in practice it is written from a single goroutine in both serial and parallel runs.

#### Trace Inertness (Safety Argument)

The spec requires tracing to be observational only.

This integration enforces inertness as follows:

- Recording cannot fail the run:
	- Event recording has no return value.
	- All recording uses `trace.SafeRecord`, which swallows panics.
	- Final trace serialization/hashing errors are treated as non-fatal: `TraceBytes`/`TraceHash` are left empty rather than failing the build.

- Recording cannot modify task definitions or execution behavior:
	- Trace events are constructed from already-computed stable identifiers (task name, event kind, optional stable fields).
	- No mutation of tasks, cache, scheduler, or state machine logic is performed by tracing.

- Recording cannot introduce runtime-dependent ordering:
	- Even though events are recorded as they are committed, the final trace is canonicalized (sorted) before serialization.
	- Therefore concurrency interleavings cannot change trace byte output.

#### Known Limitation (By Sprint-03 Authority)

The trace spec mentions “invalidation reasons”, but the DAG executor currently receives only an `IncrementalPlan` decision (Execute vs ReuseCache) and does not receive a per-node invalidation reason.

Accordingly, this integration records cache/execution/skip/failure/restoration decisions but does not emit a dedicated `TaskInvalidated` event yet.
Adding that would require an explicit reason source in the sprint-03 trace-engine documents (or a reason map passed into the executor).

