package compiler

import (
	"errors"
	"fmt"
	"strings"

	"github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
)

// ReferenceMode distinguishes the three resolution strategies for references.
//
// The mode cannot be determined purely from syntax; it requires introspecting
// the resolved data to determine if the path leads to a map or scalar.
type ReferenceMode int

const (
	// PropertyMode represents a single property value resolution.
	// Example: @config:app:server.port resolves to a scalar value.
	PropertyMode ReferenceMode = iota

	// MapMode represents specific nested map resolution.
	// Example: @config:app:database.pool resolves to a map of properties.
	MapMode

	// RootMode represents everything from a provider path.
	// Example: @base:. includes all properties at the provider root.
	RootMode
)

// String returns a human-readable representation of the ReferenceMode.
func (m ReferenceMode) String() string {
	switch m {
	case PropertyMode:
		return "Property"
	case MapMode:
		return "Map"
	case RootMode:
		return "Root"
	default:
		return "Unknown"
	}
}

// ResolvedReference represents the result of resolving a reference expression during compilation.
//
// The Mode field determines which of Value or Entries is populated:
//   - PropertyMode: Value contains the single resolved value, Entries is nil
//   - MapMode/RootMode: Entries contains the resolved map, Value is nil
type ResolvedReference struct {
	Mode    ReferenceMode       // Resolution mode (determines which field is populated)
	Value   ast.Expr            // For PropertyMode: the single resolved value
	Entries map[string]ast.Expr // For Root/MapMode: resolved entries to merge
}

// ResolutionContext tracks active resolution stack to detect circular references.
//
// The compiler maintains a context throughout the reference resolution process,
// pushing paths as they begin resolution and popping them when complete.
// If a path appears twice in the stack, a circular reference is detected.
type ResolutionContext struct {
	Stack []PathRef // Stack of paths currently being resolved
}

// PathRef identifies a unique reference path being resolved.
type PathRef struct {
	Alias string // Provider instance alias
	Path  string // Joined path segments
}

// String returns a human-readable representation of the PathRef.
func (r PathRef) String() string {
	return fmt.Sprintf("%s:%s", r.Alias, r.Path)
}

// Push adds a path to the resolution stack.
//
// Returns an error if the path already exists in the stack (circular reference detected).
// The error message includes the full cycle path for debugging.
func (ctx *ResolutionContext) Push(alias string, path []string) error {
	ref := PathRef{Alias: alias, Path: pathKey(path)}

	// Check for cycle
	for _, existing := range ctx.Stack {
		if existing == ref {
			return fmt.Errorf("%w: %s", ErrCircularReference, ctx.formatCycle(ref))
		}
	}

	ctx.Stack = append(ctx.Stack, ref)
	return nil
}

// Pop removes the most recent path from the resolution stack.
//
// This should be called with defer after a successful Push to ensure cleanup:
//
//	if err := ctx.Push(alias, path); err != nil {
//	    return err
//	}
//	defer ctx.Pop()
func (ctx *ResolutionContext) Pop() {
	if len(ctx.Stack) > 0 {
		ctx.Stack = ctx.Stack[:len(ctx.Stack)-1]
	}
}

// formatCycle creates a human-readable cycle path for error messages.
//
// Example: "base:app → base:common → base:app"
func (ctx *ResolutionContext) formatCycle(ref PathRef) string {
	parts := make([]string, 0, len(ctx.Stack)+1)
	for _, r := range ctx.Stack {
		parts = append(parts, r.String())
	}
	parts = append(parts, ref.String())
	return strings.Join(parts, " → ")
}

func pathKey(path []string) string {
	if len(path) == 0 {
		return "."
	}
	return strings.Join(path, ":")
}

// Sentinel errors for reference resolution failures.
var (
	// ErrAliasNotFound indicates a source alias is not configured.
	ErrAliasNotFound = errors.New("alias not found")

	// ErrPathNotFound indicates a provider cannot resolve the path.
	ErrPathNotFound = errors.New("path not found")

	// ErrPropertyPathInvalid indicates a property path does not exist in the data.
	ErrPropertyPathInvalid = errors.New("property path invalid")

	// ErrCircularReference indicates a cycle was detected in the resolution chain.
	ErrCircularReference = errors.New("circular reference detected")
)

// DetermineReferenceMode determines the resolution mode for a reference based on
// its path and the structure of the resolved data.
//
// Mode determination logic:
//   - Empty path → RootMode (entire data)
//   - Path leads to map → MapMode (nested map resolution)
//   - Path leads to scalar → PropertyMode (single value resolution)
func DetermineReferenceMode(ref *ast.ReferenceExpr, resourceData map[string]any) ReferenceMode {
	// Empty path means root reference
	if len(ref.Path) == 0 || (len(ref.Path) == 1 && ref.Path[0] == ".") {
		return RootMode
	}

	// Navigate to the value at the path
	value, err := navigatePath(resourceData, ref.Path)
	if err != nil {
		// If navigation fails, default to PropertyMode
		// The error will be caught during actual resolution
		return PropertyMode
	}

	// Determine mode based on the resolved value type
	if _, isMap := value.(map[string]any); isMap {
		return MapMode
	}

	return PropertyMode
}

// ResolveReference resolves a reference expression given provider data.
// Returns a ResolvedReference containing the resolved value(s) and metadata.
// The resCtx parameter tracks the resolution stack for circular reference detection.
func ResolveReference(ref *ast.ReferenceExpr, resourceData map[string]any, resCtx *ResolutionContext) (*ResolvedReference, error) {
	// Detect circular references by tracking this path in the resolution stack
	if err := resCtx.Push(ref.Alias, ref.Path); err != nil {
		// T089: Include source span in circular reference errors
		return nil, formatReferenceError(ref, "", err)
	}
	defer resCtx.Pop()

	// Determine the resolution mode
	mode := DetermineReferenceMode(ref, resourceData)

	// Create result structure
	resolved := &ResolvedReference{
		Mode: mode,
	}

	switch mode {
	case RootMode:
		// Convert entire provider data to AST expressions
		// T084: Empty provider data is valid - returns empty map
		entries, err := convertToASTExprs(resourceData)
		if err != nil {
			// T087: Wrap provider errors with full context (alias, path, operation)
			// T089: Include source span
			return nil, formatReferenceError(ref, "root",
				fmt.Errorf("failed to convert root data for alias %q: %w",
					ref.Alias, err))
		}
		resolved.Entries = entries

	case MapMode:
		// Navigate to the map at the path
		value, err := navigatePath(resourceData, ref.Path)
		if err != nil {
			// T087: Wrap errors with alias, path context
			// T089: Include source span
			return nil, formatReferenceError(ref, strings.Join(ref.Path, "."),
				fmt.Errorf("failed to navigate in alias %q: %w",
					ref.Alias, err))
		}

		// Type assert to map
		mapValue, ok := value.(map[string]any)
		if !ok {
			// T089: Include source span for type mismatch
			return nil, formatReferenceError(ref, strings.Join(ref.Path, "."),
				fmt.Errorf("expected map at alias %q path %q, got %T",
					ref.Alias, strings.Join(ref.Path, "."), value))
		}

		// Convert map to AST expressions
		entries, err := convertToASTExprs(mapValue)
		if err != nil {
			// T087: Include alias and path in conversion errors
			// T089: Include source span
			return nil, formatReferenceError(ref, strings.Join(ref.Path, "."),
				fmt.Errorf("failed to convert map data for alias %q: %w",
					ref.Alias, err))
		}
		resolved.Entries = entries

	case PropertyMode:
		// Navigate to the scalar value at the path
		value, err := navigatePath(resourceData, ref.Path)
		if err != nil {
			// T087: Full context for navigation errors
			// T089: Include source span
			return nil, formatReferenceError(ref, strings.Join(ref.Path, "."),
				fmt.Errorf("failed to navigate in alias %q: %w",
					ref.Alias, err))
		}

		// Convert to AST expression
		expr, err := valueToASTExpr(value)
		if err != nil {
			// T087: Include operation context
			// T089: Include source span
			return nil, formatReferenceError(ref, strings.Join(ref.Path, "."),
				fmt.Errorf("failed to convert value for alias %q: %w",
					ref.Alias, err))
		}
		resolved.Value = expr
	}

	return resolved, nil
}

// formatReferenceError wraps an error with source location information.
// T089: Ensures all reference resolution errors include filename and line number for precise error reporting.
//
// Parameters:
//   - ref: The reference expression being resolved
//   - path: The path being accessed (may be empty for root or partial path on error)
//   - err: The underlying error to wrap
//
// Returns an error with format: "resolving @alias:path at filename:line: error message"
func formatReferenceError(ref *ast.ReferenceExpr, path string, err error) error {
	// Build reference description
	refDesc := fmt.Sprintf("@%s:%s", ref.Alias, pathKey(ref.Path))
	if path != "" {
		refDesc = fmt.Sprintf("@%s:%s", ref.Alias, path)
	}

	// Include source location for precise error reporting
	return fmt.Errorf("resolving %s at %s:%d: %w",
		refDesc,
		ref.SourceSpan.Filename, ref.SourceSpan.StartLine,
		err)
}

// navigatePath navigates through nested maps following the path segments.
// Returns the value at the end of the path, or an error if navigation fails.
//
// T088: Error messages include available keys for actionable feedback.
// Note: The final value CAN be a scalar (forgiving), but cannot navigate THROUGH a scalar.
func navigatePath(data map[string]any, path []string) (any, error) {
	if len(path) == 0 {
		return data, nil
	}

	current := any(data)

	for i, segment := range path {
		currentMap, ok := current.(map[string]any)
		if !ok {
			// Current value is not a map - cannot navigate through scalar
			// This check catches attempts to navigate through scalars like: "version.major" when version="1.2.3"
			return nil, fmt.Errorf("path segment %d is not a map (got %T, remaining path: %v): %w",
				i, current, path[i:], ErrPropertyPathInvalid)
		}

		value, exists := currentMap[segment]
		if !exists {
			// T088: Collect available keys for helpful error message with actionable information
			availableKeys := make([]string, 0, len(currentMap))
			for k := range currentMap {
				availableKeys = append(availableKeys, k)
			}
			return nil, fmt.Errorf("path segment %q not found (available keys: %v): %w",
				segment, availableKeys, ErrPropertyPathInvalid)
		}

		current = value
	}

	// Return final value (forgiving: can be scalar or map)
	return current, nil
}

// convertToASTExprs converts a map[string]any to map[string]ast.Expr.
func convertToASTExprs(data map[string]any) (map[string]ast.Expr, error) {
	result := make(map[string]ast.Expr, len(data))
	for key, value := range data {
		expr, err := valueToASTExpr(value)
		if err != nil {
			return nil, fmt.Errorf("failed to convert key %q: %w", key, err)
		}
		result[key] = expr
	}
	return result, nil
}

// valueToASTExpr converts a runtime value to an AST expression.
func valueToASTExpr(value any) (ast.Expr, error) {
	if value == nil {
		return &ast.StringLiteral{Value: ""}, nil
	}

	switch v := value.(type) {
	case string:
		return &ast.StringLiteral{Value: v}, nil

	case int:
		return &ast.StringLiteral{Value: fmt.Sprintf("%d", v)}, nil

	case int64:
		return &ast.StringLiteral{Value: fmt.Sprintf("%d", v)}, nil

	case float64:
		return &ast.StringLiteral{Value: fmt.Sprintf("%g", v)}, nil

	case bool:
		return &ast.StringLiteral{Value: fmt.Sprintf("%t", v)}, nil

	case map[string]any:
		// Recursively convert nested map
		entries, err := convertToASTExprs(v)
		if err != nil {
			return nil, err
		}
		return &ast.MapExpr{Entries: entries}, nil

	case []any:
		// Convert list to ListExpr
		elements := make([]ast.Expr, 0, len(v))
		for i, elem := range v {
			expr, err := valueToASTExpr(elem)
			if err != nil {
				return nil, fmt.Errorf("failed to convert list element %d: %w", i, err)
			}
			elements = append(elements, expr)
		}
		return &ast.ListExpr{Elements: elements}, nil

	case *ast.ReferenceExpr:
		// Already an AST expression - pass through
		return v, nil

	default:
		// For other types, convert to string representation
		return &ast.StringLiteral{Value: fmt.Sprintf("%v", v)}, nil
	}
}
