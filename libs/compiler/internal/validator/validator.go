package validator

import (
	"context"
	"fmt"
	"strings"

	"github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
)

// Options configures semantic validation behavior.
type Options struct {
	// RegisteredProviderAliases contains the list of valid provider aliases.
	RegisteredProviderAliases []string
}

// Validator performs semantic validation on compiled data.
type Validator struct {
	opts Options
}

// New creates a new Validator with the given options.
func New(opts Options) *Validator {
	return &Validator{
		opts: opts,
	}
}

// Validate performs semantic validation on the provided data.
// It checks for unresolved references and returns structured errors.
func (v *Validator) Validate(ctx context.Context, data any) error {
	return v.validateValue(ctx, data, "")
}

// validateValue recursively validates a single value.
func (v *Validator) validateValue(ctx context.Context, val any, path string) error {
	switch value := val.(type) {
	case *ast.ReferenceExpr:
		// Check if this is an unresolved reference
		// In the validation phase, references should have been resolved by the resolver
		// If we still see a ReferenceExpr, it's unresolved
		return v.checkReference(value)

	case map[string]any:
		// Recursively validate map entries
		for k, mapValue := range value {
			newPath := k
			if path != "" {
				newPath = path + "." + k
			}
			if err := v.validateValue(ctx, mapValue, newPath); err != nil {
				return err
			}
		}

	case []any:
		// Recursively validate slice elements
		for i, elem := range value {
			newPath := fmt.Sprintf("%s[%d]", path, i)
			if err := v.validateValue(ctx, elem, newPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// checkReference validates a reference expression.
func (v *Validator) checkReference(ref *ast.ReferenceExpr) error {
	// Check if the alias is registered
	if !v.isAliasRegistered(ref.Alias) {
		return &ErrUnresolvedReference{
			Alias:       ref.Alias,
			Path:        ref.Path,
			SourceSpan:  ref.SourceSpan,
			Suggestions: v.findSuggestions(ref.Alias),
		}
	}

	return nil
}

// isAliasRegistered checks if an alias is in the list of registered providers.
func (v *Validator) isAliasRegistered(alias string) bool {
	for _, registered := range v.opts.RegisteredProviderAliases {
		if registered == alias {
			return true
		}
	}
	return false
}

// findSuggestions uses fuzzy matching to suggest correct provider aliases.
func (v *Validator) findSuggestions(typo string) []string {
	var suggestions []string

	// Simple Levenshtein-like fuzzy matching
	// For each registered alias, calculate edit distance
	// Return aliases with distance <= 2
	for _, alias := range v.opts.RegisteredProviderAliases {
		distance := levenshteinDistance(typo, alias)
		if distance <= 2 {
			suggestions = append(suggestions, alias)
		}
	}

	return suggestions
}

// levenshteinDistance calculates the Levenshtein distance between two strings.
// This is used for fuzzy matching of provider aliases.
func levenshteinDistance(s1, s2 string) int {
	len1 := len(s1)
	len2 := len(s2)

	// Create a matrix to store distances
	matrix := make([][]int, len1+1)
	for i := range matrix {
		matrix[i] = make([]int, len2+1)
	}

	// Initialize first row and column
	for i := 0; i <= len1; i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len2; j++ {
		matrix[0][j] = j
	}

	// Fill matrix
	for i := 1; i <= len1; i++ {
		for j := 1; j <= len2; j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}

			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len1][len2]
}

// min returns the minimum of three integers.
func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// DependencyGraph represents a directed graph for cycle detection.
type DependencyGraph struct {
	nodes map[string]*GraphNode
}

// GraphNode represents a node in the dependency graph.
type GraphNode struct {
	ID          string
	SourceSpan  ast.SourceSpan
	Description string
	edges       []*GraphEdge
}

// GraphEdge represents an edge in the dependency graph.
type GraphEdge struct {
	To   *GraphNode
	Type EdgeType
}

// AddNode adds a node to the graph.
func (g *DependencyGraph) AddNode(node *GraphNode) {
	if g.nodes == nil {
		g.nodes = make(map[string]*GraphNode)
	}
	g.nodes[node.ID] = node
}

// AddEdge adds a directed edge between two nodes.
func (g *DependencyGraph) AddEdge(fromID, toID string, edgeType EdgeType) {
	fromNode := g.nodes[fromID]
	toNode := g.nodes[toID]

	if fromNode == nil || toNode == nil {
		return
	}

	fromNode.edges = append(fromNode.edges, &GraphEdge{
		To:   toNode,
		Type: edgeType,
	})
}

// DetectCycles performs cycle detection using depth-first search.
// Returns ErrCycleDetected if a cycle is found.
func (g *DependencyGraph) DetectCycles() error {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	var path []*GraphNode

	for id, node := range g.nodes {
		if !visited[id] {
			if cycle := g.dfs(node, visited, recStack, path); cycle != nil {
				return g.buildCycleError(cycle)
			}
		}
	}

	return nil
}

// dfs performs depth-first search to detect cycles.
// Returns the path forming the cycle if one is found.
func (g *DependencyGraph) dfs(node *GraphNode, visited, recStack map[string]bool, path []*GraphNode) []*GraphNode {
	visited[node.ID] = true
	recStack[node.ID] = true
	path = append(path, node)

	for _, edge := range node.edges {
		nextNode := edge.To

		if !visited[nextNode.ID] {
			if cycle := g.dfs(nextNode, visited, recStack, path); cycle != nil {
				return cycle
			}
		} else if recStack[nextNode.ID] {
			// Found a cycle: extract the cycle from path
			cycleStart := -1
			for i, n := range path {
				if n.ID == nextNode.ID {
					cycleStart = i
					break
				}
			}
			if cycleStart >= 0 {
				return append(path[cycleStart:], nextNode)
			}
		}
	}

	recStack[node.ID] = false
	return nil
}

// buildCycleError constructs an ErrCycleDetected from a cycle path.
func (g *DependencyGraph) buildCycleError(cyclePath []*GraphNode) error {
	chain := make([]CycleNode, len(cyclePath))

	for i, node := range cyclePath {
		edgeType := EdgeImport
		if i < len(cyclePath)-1 {
			// Determine edge type from node to next
			for _, edge := range node.edges {
				if edge.To.ID == cyclePath[i+1].ID {
					edgeType = edge.Type
					break
				}
			}
		}

		chain[i] = CycleNode{
			Type:        edgeType,
			SourceSpan:  node.SourceSpan,
			Description: node.Description,
		}
	}

	// Build descriptive message
	var parts []string
	for _, node := range cyclePath {
		parts = append(parts, node.Description)
	}
	message := fmt.Sprintf("cycle detected: %s", strings.Join(parts, " → "))

	return &ErrCycleDetected{
		Chain:   chain,
		Message: message,
	}
}
