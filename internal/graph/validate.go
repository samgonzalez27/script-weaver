package graph

import (
	"fmt"
	"sort"
)

// Validate performs structural validation on a Graph.
// It checks for duplicate node IDs, dangling edges, self-referential edges,
// and cycles. Returns StructuralError on any violation.
func Validate(g *Graph) error {
	// Build node ID set and check for duplicates
	nodeIDs := make(map[string]bool, len(g.Nodes))
	// Sort nodes by ID first for deterministic duplicate detection
	sortedNodes := make([]Node, len(g.Nodes))
	copy(sortedNodes, g.Nodes)
	sort.Slice(sortedNodes, func(i, j int) bool {
		return sortedNodes[i].ID < sortedNodes[j].ID
	})

	for _, node := range sortedNodes {
		if nodeIDs[node.ID] {
			return &StructuralError{
				Kind: "duplicate_id",
				Msg:  fmt.Sprintf("duplicate node ID: %q", node.ID),
			}
		}
		nodeIDs[node.ID] = true
	}

	// Sort edges for deterministic error reporting
	sortedEdges := make([]Edge, len(g.Edges))
	copy(sortedEdges, g.Edges)
	sort.Slice(sortedEdges, func(i, j int) bool {
		if sortedEdges[i].From != sortedEdges[j].From {
			return sortedEdges[i].From < sortedEdges[j].From
		}
		return sortedEdges[i].To < sortedEdges[j].To
	})

	// Check for self-referential and dangling edges
	adjacency := make(map[string][]string)
	for _, edge := range sortedEdges {
		// Self-reference check
		if edge.From == edge.To {
			return &StructuralError{
				Kind: "self_reference",
				Msg:  fmt.Sprintf("self-referential edge: %q -> %q", edge.From, edge.To),
			}
		}
		// Dangling edge check - 'from' must exist
		if !nodeIDs[edge.From] {
			return &StructuralError{
				Kind: "dangling_edge",
				Msg:  fmt.Sprintf("edge references unknown node: %q", edge.From),
			}
		}
		// Dangling edge check - 'to' must exist
		if !nodeIDs[edge.To] {
			return &StructuralError{
				Kind: "dangling_edge",
				Msg:  fmt.Sprintf("edge references unknown node: %q", edge.To),
			}
		}
		adjacency[edge.From] = append(adjacency[edge.From], edge.To)
	}

	// Cycle detection using DFS with coloring
	// Colors: 0 = white (unvisited), 1 = gray (in progress), 2 = black (done)
	color := make(map[string]int)
	var path []string

	var dfs func(node string) error
	dfs = func(node string) error {
		color[node] = 1 // gray - in progress
		path = append(path, node)

		// Sort neighbors for deterministic traversal
		neighbors := adjacency[node]
		sort.Strings(neighbors)

		for _, neighbor := range neighbors {
			if color[neighbor] == 1 {
				// Found cycle - build cycle path
				cycleStart := -1
				for i, n := range path {
					if n == neighbor {
						cycleStart = i
						break
					}
				}
				cyclePath := append(path[cycleStart:], neighbor)
				return &StructuralError{
					Kind: "cycle",
					Msg:  fmt.Sprintf("cycle detected: %v", cyclePath),
				}
			}
			if color[neighbor] == 0 {
				if err := dfs(neighbor); err != nil {
					return err
				}
			}
		}

		path = path[:len(path)-1]
		color[node] = 2 // black - done
		return nil
	}

	// Get all node IDs sorted for deterministic traversal order
	allNodes := make([]string, 0, len(nodeIDs))
	for id := range nodeIDs {
		allNodes = append(allNodes, id)
	}
	sort.Strings(allNodes)

	for _, nodeID := range allNodes {
		if color[nodeID] == 0 {
			if err := dfs(nodeID); err != nil {
				return err
			}
		}
	}

	return nil
}
