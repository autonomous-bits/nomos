// Package converter provides functionality to convert parsed AST to runtime data structures.
package converter

import (
	"fmt"
	"strings"

	"github.com/autonomous-bits/nomos/libs/compiler/internal/models"
	"github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
)

const OrderedEntriesKey = "__nomos_ordered_entries__"

// OrderedEntry tracks ordered map entries including spread references.
// Spread entries have Spread=true and an empty Key.
type OrderedEntry struct {
	Key    string
	Value  any
	Spread bool
}

// ASTToData converts an AST into a map suitable for merging and composition.
// It extracts all SectionDecl statements and converts their entries to map[string]any.
// Returns the data map and any errors encountered during conversion.
func ASTToData(tree *ast.AST) (map[string]any, error) {
	if tree == nil {
		return make(map[string]any), nil
	}

	result := make(map[string]any)
	rootOrdered := make([]OrderedEntry, 0)
	hasRootSpread := false

	for _, stmt := range tree.Statements {
		// Currently we only process SectionDecl statements
		// Other statement types (source, import, reference) are handled elsewhere
		switch node := stmt.(type) {
		case *ast.SectionDecl:
			// Convert the section to a nested map entry
			sectionData, err := sectionToData(node)
			if err != nil {
				return nil, fmt.Errorf("failed to convert section %q: %w", node.Name, err)
			}

			// Add section to result
			result[node.Name] = sectionData
			rootOrdered = append(rootOrdered, OrderedEntry{
				Key:   node.Name,
				Value: sectionData,
			})

		case *ast.SpreadStmt:
			isSpread := shouldSpread(node.Reference)
			hasRootSpread = true
			rootOrdered = append(rootOrdered, OrderedEntry{
				Value:  node.Reference,
				Spread: isSpread,
			})
		default:
			continue
		}
	}

	if hasRootSpread {
		result[OrderedEntriesKey] = rootOrdered
	}

	return result, nil
}

// sectionToData converts a SectionDecl to either a scalar value or map[string]any.
// If the section has an inline Value (e.g., region: "us-west-2"), returns the scalar.
// If the section has Entries (nested map), returns map[string]any.
func sectionToData(section *ast.SectionDecl) (any, error) {
	// Check for inline scalar value first
	if section.Value != nil {
		return exprToValue(section.Value)
	}

	// Otherwise, process as nested map
	return mapEntriesToData(section.Entries)
}

// exprToValue converts an AST expression to a runtime value.
func exprToValue(expr ast.Expr) (any, error) {
	if expr == nil {
		return nil, fmt.Errorf("nil expression")
	}

	switch e := expr.(type) {
	case *ast.StringLiteral:
		// Check if it's a variable reference (starts with "var.")
		if strings.HasPrefix(e.Value, "var.") {
			parts := strings.Split(e.Value, ".")
			if len(parts) > 1 {
				return &ast.ReferenceExpr{
					Alias:      "var",
					Path:       parts[1:],
					SourceSpan: e.SourceSpan,
				}, nil
			}
		}
		return e.Value, nil

	case *ast.ReferenceExpr:
		// ReferenceExpr nodes are kept as-is for later resolution
		// The resolver will handle these
		return e, nil

	case *ast.MapExpr:
		// MapExpr represents a nested map structure
		// Recursively convert all nested entries
		return mapEntriesToData(e.Entries)

	case *ast.ListExpr:
		// ListExpr represents an ordered list of values
		result := make([]any, 0, len(e.Elements))
		for idx, element := range e.Elements {
			val, err := exprToValue(element)
			if err != nil {
				return nil, fmt.Errorf("failed to convert list element %d: %w", idx, err)
			}
			result = append(result, val)
		}
		return result, nil

	case *ast.PathExpr:
		// Check if it's a variable reference (starts with "var")
		if len(e.Components) > 1 && e.Components[0] == "var" {
			return &ast.ReferenceExpr{
				Alias:      "var",
				Path:       e.Components[1:],
				SourceSpan: e.SourceSpan,
			}, nil
		}

		// PathExpr might represent a structured value
		// For now, convert to string representation
		return pathExprToString(e), nil

	case *ast.IdentExpr:
		// Identifiers become strings
		return e.Name, nil

	case *ast.MarkedExpr:
		// Marked expressions become Secrets
		val, err := exprToValue(e.Expr)
		if err != nil {
			return nil, err
		}
		return models.Secret{Value: val}, nil

	default:
		return nil, fmt.Errorf("unsupported expression type: %T", expr)
	}
}

func mapEntriesToData(entries []ast.MapEntry) (map[string]any, error) {
	result := make(map[string]any, len(entries))
	ordered := make([]OrderedEntry, 0, len(entries))
	hasSpread := false

	for _, entry := range entries {
		value, err := exprToValue(entry.Value)
		if err != nil {
			return nil, fmt.Errorf("failed to convert key %q: %w", entry.Key, err)
		}

		isSpread := entry.Spread
		if isSpread {
			if ref, ok := value.(*ast.ReferenceExpr); ok {
				isSpread = shouldSpread(ref)
			}
		}

		ordered = append(ordered, OrderedEntry{
			Key:    entry.Key,
			Value:  value,
			Spread: isSpread,
		})
		if isSpread {
			hasSpread = true
			continue
		}
		result[entry.Key] = value
	}

	if hasSpread {
		result[OrderedEntriesKey] = ordered
	}

	return result, nil
}

// shouldSpread determines if a reference should be treated as a spread.
// A reference is a spread if its path contains a wildcard "*".
func shouldSpread(ref *ast.ReferenceExpr) bool {
	for _, segment := range ref.Path {
		if segment == "*" {
			return true
		}
	}
	return false
}

// pathExprToString converts a PathExpr to a dot-separated string.
func pathExprToString(p *ast.PathExpr) string {
	if len(p.Components) == 0 {
		return ""
	}
	result := p.Components[0]
	for i := 1; i < len(p.Components); i++ {
		result += "." + p.Components[i]
	}
	return result
}
