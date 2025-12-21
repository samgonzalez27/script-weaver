package graph

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// ComputeHash computes a stable, deterministic hash of the graph.
//
// The hash is computed from the normalized JSON representation of the
// graph object only (nodes and edges). Metadata and schema_version are
// explicitly excluded per spec.
//
// The hash is stable across:
//   - Different JSON formatting/whitespace
//   - Different field ordering in source JSON
//   - Metadata changes
//
// The hash changes when:
//   - Node content changes (id, type, inputs, outputs)
//   - Edge content changes (from, to)
//   - Nodes or edges are added/removed
func ComputeHash(g *Graph) (string, error) {
	// Create a normalized copy to avoid modifying the original
	normalized := g.Normalized()

	// Marshal to compact JSON (no whitespace)
	// Note: encoding/json sorts map keys alphabetically
	data, err := json.Marshal(normalized)
	if err != nil {
		return "", &ParseError{Msg: "failed to serialize graph for hashing", Err: err}
	}

	// Compute SHA-256
	hash := sha256.Sum256(data)

	// Return as hex string
	return hex.EncodeToString(hash[:]), nil
}

// ComputeHashBytes returns the raw SHA-256 hash bytes.
func ComputeHashBytes(g *Graph) ([32]byte, error) {
	normalized := g.Normalized()

	data, err := json.Marshal(normalized)
	if err != nil {
		return [32]byte{}, &ParseError{Msg: "failed to serialize graph for hashing", Err: err}
	}

	return sha256.Sum256(data), nil
}
