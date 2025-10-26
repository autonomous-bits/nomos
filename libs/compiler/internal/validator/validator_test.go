package validator

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
)

// TestValidate_UnresolvedReference tests detection of unresolved references.
func TestValidate_UnresolvedReference(t *testing.T) {
	// Arrange: Create an AST with an unresolved reference
	unresolvedRef := &ast.ReferenceExpr{
		Alias: "nonexistent",
		Path:  []string{"database", "host"},
		SourceSpan: ast.SourceSpan{
			Filename:  "test.csl",
			StartLine: 5,
			StartCol:  9,
			EndLine:   5,
			EndCol:    35,
		},
	}

	data := map[string]any{
		"database": map[string]any{
			"host": unresolvedRef, // Unresolved reference
			"port": 5432,
		},
	}

	registeredAliases := []string{"file", "env", "vault"} // Available providers

	// Act: Validate the data
	validator := New(Options{
		RegisteredProviderAliases: registeredAliases,
	})

	err := validator.Validate(context.Background(), data)

	// Assert: Expect ErrUnresolvedReference
	if err == nil {
		t.Fatal("expected error for unresolved reference, got nil")
	}

	var unresolvedErr *ErrUnresolvedReference
	if !errors.As(err, &unresolvedErr) {
		t.Fatalf("expected *ErrUnresolvedReference, got %T: %v", err, err)
	}

	// Check error details
	if unresolvedErr.Alias != "nonexistent" {
		t.Errorf("expected alias %q, got %q", "nonexistent", unresolvedErr.Alias)
	}

	if len(unresolvedErr.Path) != 2 || unresolvedErr.Path[0] != "database" || unresolvedErr.Path[1] != "host" {
		t.Errorf("expected path [database host], got %v", unresolvedErr.Path)
	}

	// Check that error message includes source location
	errMsg := err.Error()
	if !strings.Contains(errMsg, "test.csl") {
		t.Errorf("error message should contain filename, got: %s", errMsg)
	}

	if !strings.Contains(errMsg, "5:9") {
		t.Errorf("error message should contain line:col, got: %s", errMsg)
	}
}

// TestValidate_UnresolvedReference_WithSuggestions tests fuzzy matching suggestions.
func TestValidate_UnresolvedReference_WithSuggestions(t *testing.T) {
	// Arrange: Create an AST with a typo in the alias
	unresolvedRef := &ast.ReferenceExpr{
		Alias: "fil", // Typo of "file"
		Path:  []string{"config", "value"},
		SourceSpan: ast.SourceSpan{
			Filename:  "test.csl",
			StartLine: 3,
			StartCol:  10,
			EndLine:   3,
			EndCol:    25,
		},
	}

	data := map[string]any{
		"config": map[string]any{
			"value": unresolvedRef,
		},
	}

	registeredAliases := []string{"file", "env", "vault"}

	// Act: Validate
	validator := New(Options{
		RegisteredProviderAliases: registeredAliases,
	})

	err := validator.Validate(context.Background(), data)

	// Assert: Expect suggestion for "file"
	if err == nil {
		t.Fatal("expected error for unresolved reference, got nil")
	}

	var unresolvedErr *ErrUnresolvedReference
	if !errors.As(err, &unresolvedErr) {
		t.Fatalf("expected *ErrUnresolvedReference, got %T", err)
	}

	if len(unresolvedErr.Suggestions) == 0 {
		t.Error("expected suggestions, got none")
	}

	// Check that "file" is suggested
	if !contains(unresolvedErr.Suggestions, "file") {
		t.Errorf("expected suggestion %q, got %v", "file", unresolvedErr.Suggestions)
	}

	// Check error message includes suggestion
	errMsg := err.Error()
	if !strings.Contains(errMsg, "did you mean") {
		t.Errorf("error message should include suggestion, got: %s", errMsg)
	}
}

// TestValidate_CycleDetection_Imports tests cycle detection for import statements.
func TestValidate_CycleDetection_Imports(t *testing.T) {
	// Note: This is a simplified test. Full cycle detection requires
	// tracking imports across files, which happens during compilation.
	// For now, we'll test the graph-based cycle detection algorithm directly.

	t.Skip("TODO: Implement after cycle detection graph builder is complete")
}

// TestDetectCycles_SimpleGraph tests the cycle detection algorithm on a simple graph.
func TestDetectCycles_SimpleGraph(t *testing.T) {
	// Arrange: Build a graph with a cycle: A → B → C → A
	graph := &DependencyGraph{
		nodes: make(map[string]*GraphNode),
	}

	nodeA := &GraphNode{
		ID: "A",
		SourceSpan: ast.SourceSpan{
			Filename:  "a.csl",
			StartLine: 1,
			StartCol:  1,
			EndLine:   1,
			EndCol:    10,
		},
		Description: "file A",
	}

	nodeB := &GraphNode{
		ID: "B",
		SourceSpan: ast.SourceSpan{
			Filename:  "b.csl",
			StartLine: 1,
			StartCol:  1,
			EndLine:   1,
			EndCol:    10,
		},
		Description: "file B",
	}

	nodeC := &GraphNode{
		ID: "C",
		SourceSpan: ast.SourceSpan{
			Filename:  "c.csl",
			StartLine: 1,
			StartCol:  1,
			EndLine:   1,
			EndCol:    10,
		},
		Description: "file C",
	}

	graph.AddNode(nodeA)
	graph.AddNode(nodeB)
	graph.AddNode(nodeC)

	graph.AddEdge("A", "B", EdgeImport)
	graph.AddEdge("B", "C", EdgeImport)
	graph.AddEdge("C", "A", EdgeImport) // Creates cycle

	// Act: Detect cycles
	err := graph.DetectCycles()

	// Assert: Expect ErrCycleDetected
	if err == nil {
		t.Fatal("expected cycle detection error, got nil")
	}

	var cycleErr *ErrCycleDetected
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected *ErrCycleDetected, got %T: %v", err, err)
	}

	// Check cycle chain length (should be A → B → C → A, so 4 nodes)
	if len(cycleErr.Chain) < 3 {
		t.Errorf("expected at least 3 nodes in cycle chain, got %d", len(cycleErr.Chain))
	}

	// Check error message
	errMsg := err.Error()
	if !strings.Contains(errMsg, "cycle detected") {
		t.Errorf("error message should mention cycle, got: %s", errMsg)
	}
}

// TestDetectCycles_NoCycle tests that graphs without cycles pass validation.
func TestDetectCycles_NoCycle(t *testing.T) {
	// Arrange: Build a DAG: A → B, A → C, B → C
	graph := &DependencyGraph{
		nodes: make(map[string]*GraphNode),
	}

	nodeA := &GraphNode{ID: "A", Description: "A"}
	nodeB := &GraphNode{ID: "B", Description: "B"}
	nodeC := &GraphNode{ID: "C", Description: "C"}

	graph.AddNode(nodeA)
	graph.AddNode(nodeB)
	graph.AddNode(nodeC)

	graph.AddEdge("A", "B", EdgeImport)
	graph.AddEdge("A", "C", EdgeImport)
	graph.AddEdge("B", "C", EdgeImport)

	// Act: Detect cycles
	err := graph.DetectCycles()

	// Assert: Expect no error
	if err != nil {
		t.Errorf("expected no cycle, got error: %v", err)
	}
}

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
