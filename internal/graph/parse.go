package graph

import (
	"encoding/json"
	"fmt"
	"io"
)

// SupportedSchemaVersion is the only schema version this package supports.
const SupportedSchemaVersion = "1.0.0"

// Parse decodes a graph definition from JSON and validates it.
// It returns ParseError for malformed JSON, SchemaError for missing or
// invalid fields, and SemanticError for unsupported schema versions.
func Parse(r io.Reader) (*Document, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()

	var doc Document
	if err := dec.Decode(&doc); err != nil {
		// Check if this is a type error (wrong field type)
		if _, ok := err.(*json.UnmarshalTypeError); ok {
			return nil, &SchemaError{Msg: fmt.Sprintf("invalid field type: %v", err)}
		}
		// Check if this is an unknown field error
		if syntaxErr, ok := err.(*json.SyntaxError); ok {
			return nil, &ParseError{Msg: fmt.Sprintf("malformed JSON at offset %d", syntaxErr.Offset), Err: err}
		}
		// Unknown field errors from DisallowUnknownFields come as generic errors
		// containing "unknown field"
		return nil, &ParseError{Msg: err.Error(), Err: err}
	}

	// Validate required fields
	if err := validateRequired(&doc); err != nil {
		return nil, err
	}

	// Validate schema version
	if doc.SchemaVersion != SupportedSchemaVersion {
		return nil, &SemanticError{
			Msg: fmt.Sprintf("unsupported schema_version %q, expected %q", doc.SchemaVersion, SupportedSchemaVersion),
		}
	}

	return &doc, nil
}

// validateRequired checks that all required fields are present.
func validateRequired(doc *Document) error {
	if doc.SchemaVersion == "" {
		return &SchemaError{Field: "schema_version", Msg: "required field is missing"}
	}
	// Note: Graph and Metadata are structs, so they are always "present" after decode.
	// We need to validate their required sub-fields.
	if doc.Graph.Nodes == nil {
		return &SchemaError{Field: "graph.nodes", Msg: "required field is missing"}
	}
	if doc.Graph.Edges == nil {
		return &SchemaError{Field: "graph.edges", Msg: "required field is missing"}
	}
	// Validate each node has required fields
	for i, node := range doc.Graph.Nodes {
		if node.ID == "" {
			return &SchemaError{Field: fmt.Sprintf("graph.nodes[%d].id", i), Msg: "required field is missing"}
		}
		if node.Type == "" {
			return &SchemaError{Field: fmt.Sprintf("graph.nodes[%d].type", i), Msg: "required field is missing"}
		}
		if node.Inputs == nil {
			return &SchemaError{Field: fmt.Sprintf("graph.nodes[%d].inputs", i), Msg: "required field is missing"}
		}
		if node.Outputs == nil {
			return &SchemaError{Field: fmt.Sprintf("graph.nodes[%d].outputs", i), Msg: "required field is missing"}
		}
	}
	// Validate each edge has required fields
	for i, edge := range doc.Graph.Edges {
		if edge.From == "" {
			return &SchemaError{Field: fmt.Sprintf("graph.edges[%d].from", i), Msg: "required field is missing"}
		}
		if edge.To == "" {
			return &SchemaError{Field: fmt.Sprintf("graph.edges[%d].to", i), Msg: "required field is missing"}
		}
	}
	return nil
}
