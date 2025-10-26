// Package validator provides semantic validation for Nomos compilation.
package validator

import (
	"fmt"
	"strings"

	"github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
)

// ErrUnresolvedReference indicates a reference could not be resolved during semantic validation.
type ErrUnresolvedReference struct {
	// Alias is the provider alias that could not be resolved.
	Alias string

	// Path is the reference path that could not be resolved.
	Path []string

	// SourceSpan identifies where the unresolved reference appears in source.
	SourceSpan ast.SourceSpan

	// Suggestions contains possible corrections based on fuzzy matching.
	Suggestions []string
}

// Error implements the error interface.
func (e *ErrUnresolvedReference) Error() string {
	msg := fmt.Sprintf("unresolved reference %q:%v at %s:%d:%d",
		e.Alias,
		e.Path,
		e.SourceSpan.Filename,
		e.SourceSpan.StartLine,
		e.SourceSpan.StartCol,
	)

	if len(e.Suggestions) > 0 {
		msg += fmt.Sprintf(" (did you mean %q?)", e.Suggestions[0])
	}

	return msg
}

// ErrCycleDetected indicates a circular dependency was detected during semantic validation.
type ErrCycleDetected struct {
	// Chain contains the ordered sequence of SourceSpans forming the cycle.
	Chain []CycleNode

	// Message provides a human-readable description of the cycle.
	Message string
}

// CycleNode represents a single node in a detected dependency cycle.
type CycleNode struct {
	// Type indicates whether this is an import or reference edge.
	Type EdgeType

	// SourceSpan identifies the source location of this edge.
	SourceSpan ast.SourceSpan

	// Description provides additional context (e.g., import path, reference alias).
	Description string
}

// EdgeType indicates the type of dependency edge in a cycle.
type EdgeType string

const (
	// EdgeImport represents an import statement dependency.
	EdgeImport EdgeType = "import"

	// EdgeReference represents a reference expression dependency.
	EdgeReference EdgeType = "reference"
)

// Error implements the error interface.
func (e *ErrCycleDetected) Error() string {
	if e.Message != "" {
		return e.Message
	}

	var parts []string
	for i, node := range e.Chain {
		arrow := " → "
		if i == len(e.Chain)-1 {
			arrow = " → (cycle)"
		}

		parts = append(parts, fmt.Sprintf("%s at %s:%d:%d%s",
			node.Description,
			node.SourceSpan.Filename,
			node.SourceSpan.StartLine,
			node.SourceSpan.StartCol,
			arrow,
		))
	}

	return "cycle detected: " + strings.Join(parts, "")
}

// Is implements error matching for errors.Is.
func (e *ErrUnresolvedReference) Is(target error) bool {
	_, ok := target.(*ErrUnresolvedReference)
	return ok
}

// Is implements error matching for errors.Is.
func (e *ErrCycleDetected) Is(target error) bool {
	_, ok := target.(*ErrCycleDetected)
	return ok
}
