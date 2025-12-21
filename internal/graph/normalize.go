package graph

import (
	"sort"
)

// Normalize transforms the graph into its canonical form.
// This ensures deterministic serialization and hash computation.
//
// Normalization rules:
//   - Nodes are sorted by id (lexicographically)
//   - Edges are sorted by from, then to
//   - Outputs in each node are sorted lexicographically
//   - Inputs map keys are sorted by encoding/json on marshal
//
// This function modifies the graph in place and returns it for chaining.
func (g *Graph) Normalize() *Graph {
	// Sort nodes by ID
	sort.Slice(g.Nodes, func(i, j int) bool {
		return g.Nodes[i].ID < g.Nodes[j].ID
	})

	// Sort outputs within each node
	for i := range g.Nodes {
		if g.Nodes[i].Outputs != nil {
			sort.Strings(g.Nodes[i].Outputs)
		}
	}

	// Sort edges by from, then to
	sort.Slice(g.Edges, func(i, j int) bool {
		if g.Edges[i].From != g.Edges[j].From {
			return g.Edges[i].From < g.Edges[j].From
		}
		return g.Edges[i].To < g.Edges[j].To
	})

	return g
}

// Normalized returns a normalized copy of the graph without modifying the original.
func (g *Graph) Normalized() *Graph {
	// Deep copy nodes
	nodes := make([]Node, len(g.Nodes))
	for i, n := range g.Nodes {
		// Copy inputs map
		inputs := make(map[string]any, len(n.Inputs))
		for k, v := range n.Inputs {
			inputs[k] = v
		}
		// Copy outputs slice
		outputs := make([]string, len(n.Outputs))
		copy(outputs, n.Outputs)

		nodes[i] = Node{
			ID:      n.ID,
			Type:    n.Type,
			Inputs:  inputs,
			Outputs: outputs,
		}
	}

	// Copy edges
	edges := make([]Edge, len(g.Edges))
	copy(edges, g.Edges)

	// Create copy and normalize
	copy := &Graph{
		Nodes: nodes,
		Edges: edges,
	}
	return copy.Normalize()
}
