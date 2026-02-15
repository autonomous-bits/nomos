// Package ast defines the Abstract Syntax Tree node types for the Nomos parser.
//
// The AST represents the syntactic structure of Nomos configuration files.
// All node types include source location information via SourceSpan for
// precise error reporting and tooling support.
package ast //nolint:revive // standard AST package name, no actual conflict

// SourceSpan represents a source code location with file name and position.
// All AST nodes embed or include a SourceSpan to enable precise diagnostics.
type SourceSpan struct {
	Filename  string `json:"filename"`
	StartLine int    `json:"start_line"`
	StartCol  int    `json:"start_col"`
	EndLine   int    `json:"end_line"`
	EndCol    int    `json:"end_col"`
}

// Node is the base interface for all AST nodes.
type Node interface {
	// Span returns the source location of this node.
	Span() SourceSpan
	node() // Marker method to ensure only AST types implement Node
}

// AST represents a complete parsed Nomos configuration file.
// It contains all top-level statements in source order.
type AST struct {
	Statements []Stmt     `json:"statements"`
	SourceSpan SourceSpan `json:"source_span"`
}

// Span implements Node for AST.
func (a *AST) Span() SourceSpan { return a.SourceSpan }
func (a *AST) node()            {}

// Stmt represents any statement in a Nomos file.
type Stmt interface {
	Node
	stmt() // Marker method
}

// SourceDecl represents a source provider declaration.
// Example: source:
//
//	alias: 'folder'
//	type: 'folder'
//	version: '1.0.0'  // Optional: semantic version
//	path: '../config'
type SourceDecl struct {
	Alias      string          `json:"alias"`
	Type       string          `json:"type"`
	Version    string          `json:"version"` // Semantic version or empty string for unversioned providers
	Config     map[string]Expr `json:"config"`  // Key-value configuration (excludes reserved fields: alias, type, version)
	SourceSpan SourceSpan      `json:"source_span"`
}

// Span implements Node for SourceDecl.
func (s *SourceDecl) Span() SourceSpan { return s.SourceSpan }
func (s *SourceDecl) node()            {}
func (s *SourceDecl) stmt()            {}

// SectionDecl represents a configuration section with key-value pairs.
// Example: config-section-name:
//
//	key1: value1
//	key2: value2
//
// For inline scalar values (e.g., region: "us-west-2"), the Value field is set
// and Entries is nil. For nested maps, Entries is populated and Value is nil.
// Exactly one of Value or Entries should be set (mutually exclusive).
type SectionDecl struct {
	Name       string          `json:"name"`
	Value      Expr            `json:"value,omitempty"`   // For inline scalar values (mutually exclusive with Entries)
	Entries    map[string]Expr `json:"entries,omitempty"` // For nested maps (mutually exclusive with Value)
	SourceSpan SourceSpan      `json:"source_span"`
}

// Span implements Node for SectionDecl.
func (s *SectionDecl) Span() SourceSpan { return s.SourceSpan }
func (s *SectionDecl) node()            {}
func (s *SectionDecl) stmt()            {}

// Expr represents expressions (currently minimal; expanded as needed).
type Expr interface {
	Node
	expr() // Marker method
}

// PathExpr represents a dotted path expression.
// Example: a.b.c
type PathExpr struct {
	Components []string   `json:"components"`
	SourceSpan SourceSpan `json:"source_span"`
}

// Span implements Node for PathExpr.
func (p *PathExpr) Span() SourceSpan { return p.SourceSpan }
func (p *PathExpr) node()            {}
func (p *PathExpr) expr()            {}

// IdentExpr represents a simple identifier.
type IdentExpr struct {
	Name       string     `json:"name"`
	SourceSpan SourceSpan `json:"source_span"`
}

// Span implements Node for IdentExpr.
func (i *IdentExpr) Span() SourceSpan { return i.SourceSpan }
func (i *IdentExpr) node()            {}
func (i *IdentExpr) expr()            {}

// StringLiteral represents a string literal value.
type StringLiteral struct {
	Value      string     `json:"value"`
	SourceSpan SourceSpan `json:"source_span"`
}

// Span implements Node for StringLiteral.
func (s *StringLiteral) Span() SourceSpan { return s.SourceSpan }
func (s *StringLiteral) node()            {}
func (s *StringLiteral) expr()            {}

// ReferenceExpr represents an inline reference expression.
// Example: @alias:path.to.value
//
// References are first-class values that can appear anywhere a value is expected.
// They consist of an alias (identifying the source provider instance) and a
// path describing what to resolve. Providers interpret the path segments.
type ReferenceExpr struct {
	Alias      string     `json:"alias"`       // Source provider instance alias
	Path       []string   `json:"path"`        // Path segments (may include "." for root)
	SourceSpan SourceSpan `json:"source_span"` // Precise source location
}

// Span implements Node for ReferenceExpr.
func (r *ReferenceExpr) Span() SourceSpan { return r.SourceSpan }
func (r *ReferenceExpr) node()            {}
func (r *ReferenceExpr) expr()            {}

// MapExpr represents a nested map/object literal.
// Example:
//
//	databases:
//	  primary:
//	    host: 'localhost'
//	    port: '5432'
//
// MapExpr enables nested configuration structures where values can themselves
// be maps, allowing for arbitrary depth nesting.
type MapExpr struct {
	Entries    map[string]Expr `json:"entries"`     // Nested key-value pairs
	SourceSpan SourceSpan      `json:"source_span"` // Precise source location
}

// Span implements Node for MapExpr.
func (m *MapExpr) Span() SourceSpan { return m.SourceSpan }
func (m *MapExpr) node()            {}
func (m *MapExpr) expr()            {}

// ListExpr represents a list/array expression in a Nomos configuration.
// Example:
//
//	servers:
//	  - web01
//	  - web02
//	  - web03
//
// ListExpr preserves item order and supports scalar values, nested lists,
// maps, and inline references. Empty lists are represented by [] at the value
// position.
type ListExpr struct {
	Elements   []Expr     `json:"elements"`    // Ordered list of expressions
	SourceSpan SourceSpan `json:"source_span"` // Precise source location
}

// Span implements Node for ListExpr.
func (l *ListExpr) Span() SourceSpan { return l.SourceSpan }
func (l *ListExpr) node()            {}
func (l *ListExpr) expr()            {}
