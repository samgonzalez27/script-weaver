package dag

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"scriptweaver/internal/core"
	"scriptweaver/internal/incremental"
)

// TaskRunner executes a single task.
//
// The executor treats non-zero exit codes as failures via the returned exitCode.
// A non-nil error indicates an infrastructure/runtime error (e.g. inability to start a process).
//
// This interface is intentionally minimal for Prompt 4; later prompts can extend
// the result with artifacts/logs/cache signals.
type TaskRunner interface {
	// Probe checks whether the task can be satisfied from cache.
	// If cached is true, result must be non-nil and FromCache must be true.
	Probe(ctx context.Context, task core.Task) (result *NodeResult, cached bool, err error)

	Run(ctx context.Context, task core.Task) (*NodeResult, error)
}

// Executor executes a TaskGraph deterministically.
//
// In Prompt 4 we implement serial execution; the struct is designed so that
// parallel dispatch can be added without rewriting the core state/scheduling logic.
type Executor struct {
	Graph  *TaskGraph
	Runner TaskRunner

	// Plan overlays the static graph with deterministic incremental decisions.
	// If nil, the executor uses Runner.Probe to decide cache reuse.
	Plan *incremental.IncrementalPlan

	mu    sync.Mutex
	state ExecutionState
}

// NewExecutor creates an executor with all nodes initialized to PENDING.
func NewExecutor(g *TaskGraph, runner TaskRunner) (*Executor, error) {
	if g == nil {
		return nil, fmt.Errorf("nil graph")
	}
	if runner == nil {
		return nil, fmt.Errorf("nil runner")
	}

	state := make(ExecutionState, len(g.nodes))
	for _, n := range g.nodes {
		state[n.Name] = TaskPending
	}

	return &Executor{Graph: g, Runner: runner, state: state}, nil
}

// StateSnapshot returns a copy of the current execution state.
func (e *Executor) StateSnapshot() ExecutionState {
	e.mu.Lock()
	defer e.mu.Unlock()

	cp := make(ExecutionState, len(e.state))
	for k, v := range e.state {
		cp[k] = v
	}
	return cp
}

// RunSerial executes the graph in serial mode.
//
// Determinism:
//   - All state mutations are guarded by a single mutex.
//   - The scheduler is polled deterministically.
//   - The next task chosen is always the first element of the scheduler's ordered list.
func (e *Executor) RunSerial(ctx context.Context) (*GraphResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	order := make([]string, 0, len(e.Graph.nodes))
	taskHashes := make(map[string]core.TaskHash, len(e.Graph.nodes))
	stdout := make(map[string][]byte, len(e.Graph.nodes))
	stderr := make(map[string][]byte, len(e.Graph.nodes))
	exitCodes := make(map[string]int, len(e.Graph.nodes))

	for {
		// 1) Lock state + 2) poll scheduler
		e.mu.Lock()
		ready := GetReadyTasks(e.Graph, e.state)

		if len(ready) == 0 {
			// No runnable tasks: either we are finished, or deadlocked due to inconsistent state.
			allTerminal := true
			for _, st := range e.state {
				if !IsTerminal(st) {
					allTerminal = false
					break
				}
			}
			e.mu.Unlock()

			if allTerminal {
				final := e.StateSnapshot()
				return &GraphResult{
					GraphHash:      e.Graph.Hash(),
					FinalState:     final,
					ExecutionOrder: order,
					TaskHashes:     taskHashes,
					Stdout:         stdout,
					Stderr:         stderr,
					ExitCode:       exitCodes,
				}, nil
			}
			return nil, fmt.Errorf("no ready tasks but graph not finished")
		}

		next := ready[0]
		task := e.Graph.nodesByName[next].Task

		// Incremental plan mode: obey the precomputed decision overlay.
		if e.Plan != nil {
			decision := e.Plan.Decisions[next]
			if decision == incremental.DecisionReuseCache {
				// Treat restoration as a deterministic "run" step so failures propagate via Sprint-01 rules.
				if err := Transition(e.state, next, TaskPending, TaskRunning); err != nil {
					e.mu.Unlock()
					return nil, err
				}
				e.mu.Unlock()

				restoreRunner, ok := e.Runner.(interface {
					Restore(ctx context.Context, task core.Task) (*NodeResult, error)
				})
				if !ok {
					return nil, fmt.Errorf("runner does not support Restore for incremental plan execution")
				}

				res, err := restoreRunner.Restore(ctx, task)
				if err != nil {
					// Cached restoration failure is treated as a task failure (not an executor fatal error).
					e.mu.Lock()
					order = append(order, next)
					stderr[next] = []byte(err.Error())
					exitCodes[next] = 1
					if ferr := FailAndPropagate(e.Graph, e.state, next); ferr != nil {
						e.mu.Unlock()
						return nil, ferr
					}
					e.mu.Unlock()
					continue
				}
				if res == nil {
					e.mu.Lock()
					order = append(order, next)
					stderr[next] = []byte("nil restore result")
					exitCodes[next] = 1
					if ferr := FailAndPropagate(e.Graph, e.state, next); ferr != nil {
						e.mu.Unlock()
						return nil, ferr
					}
					e.mu.Unlock()
					continue
				}

				e.mu.Lock()
				order = append(order, next)
				taskHashes[next] = res.Hash
				stdout[next] = res.Stdout
				stderr[next] = res.Stderr
				exitCodes[next] = res.ExitCode

				if res.ExitCode == 0 {
					if err := Transition(e.state, next, TaskRunning, TaskCompleted); err != nil {
						e.mu.Unlock()
						return nil, err
					}
					e.mu.Unlock()
					continue
				}
				if err := FailAndPropagate(e.Graph, e.state, next); err != nil {
					e.mu.Unlock()
					return nil, err
				}
				e.mu.Unlock()
				continue
			}

			// DecisionExecute: do not probe cache. Always execute.
			if decision == incremental.DecisionExecute {
				if err := Transition(e.state, next, TaskPending, TaskRunning); err != nil {
					e.mu.Unlock()
					return nil, err
				}
				e.mu.Unlock()

				runRes, err := e.Runner.Run(ctx, task)
				if err != nil {
					return nil, fmt.Errorf("executing %q: %w", next, err)
				}
				if runRes == nil {
					return nil, fmt.Errorf("executing %q: nil result", next)
				}

				e.mu.Lock()
				order = append(order, next)
				taskHashes[next] = runRes.Hash
				stdout[next] = runRes.Stdout
				stderr[next] = runRes.Stderr
				exitCodes[next] = runRes.ExitCode

				if runRes.ExitCode == 0 {
					if err := Transition(e.state, next, TaskRunning, TaskCompleted); err != nil {
						e.mu.Unlock()
						return nil, err
					}
					e.mu.Unlock()
					continue
				}
				if err := FailAndPropagate(e.Graph, e.state, next); err != nil {
					e.mu.Unlock()
					return nil, err
				}
				e.mu.Unlock()
				continue
			}
		}

		// Default mode: probe cache on-the-fly.
		probeRes, cached, err := e.Runner.Probe(ctx, task)
		if err != nil {
			e.mu.Unlock()
			return nil, fmt.Errorf("probing cache for %q: %w", next, err)
		}
		if cached {
			if probeRes == nil {
				e.mu.Unlock()
				return nil, fmt.Errorf("probing cache for %q: nil result", next)
			}
			if err := Transition(e.state, next, TaskPending, TaskCached); err != nil {
				e.mu.Unlock()
				return nil, err
			}
			taskHashes[next] = probeRes.Hash
			stdout[next] = probeRes.Stdout
			stderr[next] = probeRes.Stderr
			exitCodes[next] = probeRes.ExitCode
			e.mu.Unlock()
			continue
		}

		if err := Transition(e.state, next, TaskPending, TaskRunning); err != nil {
			e.mu.Unlock()
			return nil, err
		}
		e.mu.Unlock()

		// 3) execute task (outside lock)
		runRes, err := e.Runner.Run(ctx, task)
		if err != nil {
			return nil, fmt.Errorf("executing %q: %w", next, err)
		}
		if runRes == nil {
			return nil, fmt.Errorf("executing %q: nil result", next)
		}

		// 4) update state (under lock)
		e.mu.Lock()
		order = append(order, next)
		taskHashes[next] = runRes.Hash
		stdout[next] = runRes.Stdout
		stderr[next] = runRes.Stderr
		exitCodes[next] = runRes.ExitCode

		if runRes.ExitCode == 0 {
			if err := Transition(e.state, next, TaskRunning, TaskCompleted); err != nil {
				e.mu.Unlock()
				return nil, err
			}
			e.mu.Unlock()
			continue
		}

		// Failure: mark failed and propagate skipped.
		if err := FailAndPropagate(e.Graph, e.state, next); err != nil {
			e.mu.Unlock()
			return nil, err
		}
		e.mu.Unlock()
	}
}

type workItem struct {
	name string
	task core.Task

	// reuseCache indicates the incremental plan decision for this task.
	reuseCache bool
}

type workResult struct {
	name   string
	result *NodeResult
	err    error
}

// RunParallel executes the graph using up to `concurrency` workers.
//
// Determinism strategy:
//   - Depth-staged dispatch: tasks are dispatched in increasing topological depth.
//   - Within the same depth: lexical order by task name.
//
// All state reads/writes are synchronized by e.mu. Task execution happens outside the lock.
func (e *Executor) RunParallel(ctx context.Context, concurrency int) (*GraphResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if concurrency <= 0 {
		return nil, fmt.Errorf("concurrency must be > 0")
	}

	maxDepth := 0
	for _, d := range e.Graph.depth {
		if d > maxDepth {
			maxDepth = d
		}
	}

	byDepth := make([][]string, maxDepth+1)
	for _, n := range e.Graph.nodes {
		byDepth[e.Graph.depth[n.canonicalIndex]] = append(byDepth[e.Graph.depth[n.canonicalIndex]], n.Name)
	}
	for d := range byDepth {
		sort.Strings(byDepth[d])
	}

	workCh := make(chan workItem, concurrency)
	doneCh := make(chan workResult, concurrency)

	var wg sync.WaitGroup
	var stopOnce sync.Once
	stopWorkers := func() {
		stopOnce.Do(func() {
			close(workCh)
			wg.Wait()
		})
	}
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for w := range workCh {
				if w.reuseCache {
					restoreRunner, ok := e.Runner.(interface {
						Restore(ctx context.Context, task core.Task) (*NodeResult, error)
					})
					if !ok {
						doneCh <- workResult{name: w.name, result: &NodeResult{ExitCode: 1, Stderr: []byte("runner does not support Restore")}, err: nil}
						continue
					}
					res, err := restoreRunner.Restore(ctx, w.task)
					if err != nil {
						// Treat restoration failure as a task failure (exit code != 0), not a fatal executor error.
						res = &NodeResult{ExitCode: 1, Stderr: []byte(err.Error())}
						err = nil
					}
					doneCh <- workResult{name: w.name, result: res, err: err}
					continue
				}

				res, err := e.Runner.Run(ctx, w.task)
				doneCh <- workResult{name: w.name, result: res, err: err}
			}
		}()
	}

	order := make([]string, 0, len(e.Graph.nodes))
	taskHashes := make(map[string]core.TaskHash, len(e.Graph.nodes))
	stdout := make(map[string][]byte, len(e.Graph.nodes))
	stderr := make(map[string][]byte, len(e.Graph.nodes))
	exitCodes := make(map[string]int, len(e.Graph.nodes))
	inFlight := 0

	// Helper: check dependency success for a node index.
	depsSatisfied := func(idx int) bool {
		for _, p := range e.Graph.incoming[idx] {
			pst := e.state[e.Graph.nodes[p].Name]
			if !IsSuccessful(pst) {
				return false
			}
		}
		return true
	}

	// Coordinator loop: stage by depth.
	for depth := 0; depth <= maxDepth; depth++ {
		names := byDepth[depth]
		nextToStart := 0

		for {
			// Dispatch as many tasks as possible for this depth.
			e.mu.Lock()
			for inFlight < concurrency && nextToStart < len(names) {
				name := names[nextToStart]
				node := e.Graph.nodesByName[name]
				st := e.state[name]

				// Already terminal (e.g., skipped by earlier failure) => never execute.
				if IsTerminal(st) {
					nextToStart++
					continue
				}
				if st != TaskPending {
					e.mu.Unlock()
					stopWorkers()
					return nil, fmt.Errorf("unexpected non-pending state for %q: %s", name, st)
				}
				if !depsSatisfied(node.canonicalIndex) {
					e.mu.Unlock()
					stopWorkers()
					return nil, fmt.Errorf("task %q at depth %d is pending but dependencies are not successful", name, depth)
				}

				// Incremental plan mode: do not probe cache; schedule based on decision.
				reuseCache := false
				if e.Plan != nil {
					reuseCache = (e.Plan.Decisions[name] == incremental.DecisionReuseCache)
				} else {
					res, cached, err := e.Runner.Probe(ctx, node.Task)
					if err != nil {
						e.mu.Unlock()
						stopWorkers()
						return nil, fmt.Errorf("probing cache for %q: %w", name, err)
					}
					if cached {
						if res == nil {
							e.mu.Unlock()
							stopWorkers()
							return nil, fmt.Errorf("probing cache for %q: nil result", name)
						}
						if err := Transition(e.state, name, TaskPending, TaskCached); err != nil {
							e.mu.Unlock()
							stopWorkers()
							return nil, err
						}
						taskHashes[name] = res.Hash
						stdout[name] = res.Stdout
						stderr[name] = res.Stderr
						exitCodes[name] = res.ExitCode
						nextToStart++
						continue
					}
				}

				if err := Transition(e.state, name, TaskPending, TaskRunning); err != nil {
					e.mu.Unlock()
					stopWorkers()
					return nil, err
				}
				order = append(order, name)
				inFlight++
				nextToStart++
				workCh <- workItem{name: name, task: node.Task, reuseCache: reuseCache}
			}

			// Are we done with this depth stage?
			stageDone := (nextToStart >= len(names) && inFlight == 0)
			e.mu.Unlock()
			if stageDone {
				break
			}

			// Wait for at least one completion or context cancellation.
			select {
			case <-ctx.Done():
				stopWorkers()
				return nil, fmt.Errorf("execution cancelled: %w", ctx.Err())
			case r := <-doneCh:
				if r.err != nil {
					stopWorkers()
					return nil, fmt.Errorf("executing %q: %w", r.name, r.err)
				}
				if r.result == nil {
					stopWorkers()
					return nil, fmt.Errorf("executing %q: nil result", r.name)
				}

				e.mu.Lock()
				cur := e.state[r.name]
				if cur != TaskRunning {
					e.mu.Unlock()
					stopWorkers()
					return nil, fmt.Errorf("completion for %q but state is %s", r.name, cur)
				}

				// Record result data.
				taskHashes[r.name] = r.result.Hash
				stdout[r.name] = r.result.Stdout
				stderr[r.name] = r.result.Stderr
				exitCodes[r.name] = r.result.ExitCode

				if r.result.ExitCode == 0 {
					if err := Transition(e.state, r.name, TaskRunning, TaskCompleted); err != nil {
						e.mu.Unlock()
						stopWorkers()
						return nil, err
					}
				} else {
					if err := FailAndPropagate(e.Graph, e.state, r.name); err != nil {
						e.mu.Unlock()
						stopWorkers()
						return nil, err
					}
				}
				inFlight--
				e.mu.Unlock()
			}
		}
	}

	stopWorkers()

	final := e.StateSnapshot()
	return &GraphResult{
		GraphHash:      e.Graph.Hash(),
		FinalState:     final,
		ExecutionOrder: order,
		TaskHashes:     taskHashes,
		Stdout:         stdout,
		Stderr:         stderr,
		ExitCode:       exitCodes,
	}, nil
}
