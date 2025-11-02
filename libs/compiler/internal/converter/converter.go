// Package converter provides functionality to convert parsed AST to runtime data structures.
package converter

import (
	"fmt"

	"github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
)

// ASTToData converts an AST into a map suitable for merging and composition.
// It extracts all SectionDecl statements and converts their entries to map[string]any.
// Returns the data map and any errors encountered during conversion.
func ASTToData(tree *ast.AST) (map[string]any, error) {
	if tree == nil {
		return make(map[string]any), nil
	}

	result := make(map[string]any)

	for _, stmt := range tree.Statements {
		// Currently we only process SectionDecl statements
		// Other statement types (source, import, reference) are handled elsewhere
		section, isSection := stmt.(*ast.SectionDecl)
		if !isSection {
			continue
		}

		// Convert the section to a nested map entry
		sectionData, err := sectionToData(section)
		if err != nil {
			return nil, fmt.Errorf("failed to convert section %q: %w", section.Name, err)
		}

		// Add section to result
		result[section.Name] = sectionData
	}

	return result, nil
}

// sectionToData converts a SectionDecl to map[string]any.
func sectionToData(section *ast.SectionDecl) (map[string]any, error) {
	result := make(map[string]any, len(section.Entries))

	for key, expr := range section.Entries {
		value, err := exprToValue(expr)
		if err != nil {
			return nil, fmt.Errorf("failed to convert key %q: %w", key, err)
		}
		result[key] = value
	}

	return result, nil
}

// exprToValue converts an AST expression to a runtime value.
func exprToValue(expr ast.Expr) (any, error) {
	switch e := expr.(type) {
	case *ast.StringLiteral:
		return e.Value, nil

	case *ast.ReferenceExpr:
		// ReferenceExpr nodes are kept as-is for later resolution
		// The resolver will handle these
		return e, nil

	case *ast.MapExpr:
		// MapExpr represents a nested map structure
		// Recursively convert all nested entries
		result := make(map[string]any, len(e.Entries))
		for k, v := range e.Entries {
			val, err := exprToValue(v)
			if err != nil {
				return nil, fmt.Errorf("failed to convert nested key %q: %w", k, err)
			}
			result[k] = val
		}
		return result, nil

	case *ast.PathExpr:
		// PathExpr might represent a structured value
		// For now, convert to string representation
		return pathExprToString(e), nil

	case *ast.IdentExpr:
		// Identifiers become strings
		return e.Name, nil

	default:
		return nil, fmt.Errorf("unsupported expression type: %T", expr)
	}
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
