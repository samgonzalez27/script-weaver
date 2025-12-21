package graph

import (
	"errors"
	"fmt"
)

// Sentinel errors for programmatic error checking via errors.Is().
var (
	// ErrParse indicates malformed JSON or encoding issues.
	ErrParse = errors.New("parse error")

	// ErrSchema indicates schema violations: missing fields, wrong types, unknown fields.
	ErrSchema = errors.New("schema error")

	// ErrStructural indicates structural violations: cycles, duplicate IDs, dangling edges.
	ErrStructural = errors.New("structural error")

	// ErrSemantic indicates semantic violations: invalid version, logic violations.
	ErrSemantic = errors.New("semantic error")
)

// ParseError represents a failure to parse the graph JSON.
// Wraps ErrParse for errors.Is() compatibility.
type ParseError struct {
	Msg string // Deterministic error message
	Err error  // Optional underlying error (e.g., from json.Unmarshal)
}

func (e *ParseError) Error() string {
	if e == nil {
		return ""
	}
	if e.Msg == "" {
		return ErrParse.Error()
	}
	return fmt.Sprintf("%s: %s", ErrParse.Error(), e.Msg)
}

func (e *ParseError) Unwrap() error { return ErrParse }

// SchemaError represents a schema validation failure.
// Wraps ErrSchema for errors.Is() compatibility.
type SchemaError struct {
	Field string // The field that caused the error (if applicable)
	Msg   string // Deterministic error message
}

func (e *SchemaError) Error() string {
	if e == nil {
		return ""
	}
	if e.Field != "" {
		return fmt.Sprintf("%s: %s: %s", ErrSchema.Error(), e.Field, e.Msg)
	}
	if e.Msg == "" {
		return ErrSchema.Error()
	}
	return fmt.Sprintf("%s: %s", ErrSchema.Error(), e.Msg)
}

func (e *SchemaError) Unwrap() error { return ErrSchema }

// StructuralError represents a structural validation failure.
// Wraps ErrStructural for errors.Is() compatibility.
type StructuralError struct {
	Kind string // Type of structural issue: "cycle", "duplicate_id", "dangling_edge"
	Msg  string // Deterministic error message
}

func (e *StructuralError) Error() string {
	if e == nil {
		return ""
	}
	if e.Msg == "" {
		return ErrStructural.Error()
	}
	return fmt.Sprintf("%s: %s", ErrStructural.Error(), e.Msg)
}

func (e *StructuralError) Unwrap() error { return ErrStructural }

// SemanticError represents a semantic validation failure.
// Wraps ErrSemantic for errors.Is() compatibility.
type SemanticError struct {
	Msg string // Deterministic error message
}

func (e *SemanticError) Error() string {
	if e == nil {
		return ""
	}
	if e.Msg == "" {
		return ErrSemantic.Error()
	}
	return fmt.Sprintf("%s: %s", ErrSemantic.Error(), e.Msg)
}

func (e *SemanticError) Unwrap() error { return ErrSemantic }
