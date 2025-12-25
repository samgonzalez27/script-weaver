package dag

import "context"

// LifecycleHooks provides optional synchronous hook points around execution.
//
// Hooks must be inert:
//   - must not panic
//   - should return quickly (they run inline with execution)
//
// The engine will continue regardless of hook failures; hook implementations
// are expected to log/report errors as appropriate.
//
// Note: Hook context is intentionally minimal to preserve isolation and avoid
// mutating core engine state.
type LifecycleHooks interface {
	BeforeRun(ctx context.Context)
	AfterRun(ctx context.Context)
	BeforeNode(ctx context.Context, taskID string)
	AfterNode(ctx context.Context, taskID string)
}
