// Package graph provides types and functions for parsing, validating,
// and working with ScriptWeaver graph definitions.
//
// Graphs are declarative execution plans encoded as JSON. This package
// implements strict validation phases:
//
//   - Parse: JSON decoding and encoding validation
//   - Schema: Required fields, types, and unknown field rejection
//   - Structural: DAG validation, duplicate IDs, dangling edges
//   - Semantic: Version compatibility, logic rules
//
// All validation errors are categorized into distinct error types
// that can be checked programmatically using errors.Is().
package graph
