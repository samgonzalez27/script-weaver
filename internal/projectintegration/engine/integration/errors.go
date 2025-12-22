package integration

import "fmt"

// InvalidWorkspaceError indicates that the reserved .scriptweaver workspace
// exists but does not conform to the allowed structure.
type InvalidWorkspaceError struct {
	Err error
}

func (e *InvalidWorkspaceError) Error() string {
	if e == nil || e.Err == nil {
		return "invalid workspace"
	}
	return fmt.Sprintf("invalid workspace: %v", e.Err)
}

func (e *InvalidWorkspaceError) Unwrap() error { return e.Err }

// InvalidConfigError indicates that .scriptweaver/config.json is present but
// invalid.
type InvalidConfigError struct {
	Err error
}

func (e *InvalidConfigError) Error() string {
	if e == nil || e.Err == nil {
		return "invalid config"
	}
	return fmt.Sprintf("invalid config: %v", e.Err)
}

func (e *InvalidConfigError) Unwrap() error { return e.Err }

// AmbiguousGraphError indicates that discovery found multiple graph candidates
// at the same precedence level.
type AmbiguousGraphError struct {
	Err error
}

func (e *AmbiguousGraphError) Error() string {
	if e == nil || e.Err == nil {
		return "ambiguous graph"
	}
	return fmt.Sprintf("ambiguous graph: %v", e.Err)
}

func (e *AmbiguousGraphError) Unwrap() error { return e.Err }

// GraphNotFoundError indicates no graph was found via deterministic discovery.
type GraphNotFoundError struct {
	Err error
}

func (e *GraphNotFoundError) Error() string {
	if e == nil || e.Err == nil {
		return "graph not found"
	}
	return fmt.Sprintf("graph not found: %v", e.Err)
}

func (e *GraphNotFoundError) Unwrap() error { return e.Err }

// SandboxViolationError indicates the sandbox guard detected a write outside
// .scriptweaver/ during orchestration.
type SandboxViolationError struct {
	Details string
}

func (e *SandboxViolationError) Error() string {
	if e == nil || e.Details == "" {
		return "sandbox violation"
	}
	return "sandbox violation: " + e.Details
}
