package graph

// Document represents the top-level structure of a graph definition file.
// All three required fields must be present: schema_version, graph, and metadata.
type Document struct {
	SchemaVersion string   `json:"schema_version"`
	Graph         Graph    `json:"graph"`
	Metadata      Metadata `json:"metadata"`
}

// Graph defines the execution structure with nodes and edges.
type Graph struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

// Node represents a single execution unit in the graph.
type Node struct {
	ID      string         `json:"id"`
	Type    string         `json:"type"`
	Inputs  map[string]any `json:"inputs"`
	Outputs []string       `json:"outputs"`
}

// Edge defines a directed dependency between two nodes.
type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// Metadata contains non-execution information about the graph.
// All fields are optional.
type Metadata struct {
	Name        string   `json:"name,omitempty"`
	Description string   `json:"description,omitempty"`
	Labels      []string `json:"labels,omitempty"`
}
