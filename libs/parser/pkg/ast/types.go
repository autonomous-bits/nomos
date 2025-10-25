// Package ast defines the Abstract Syntax Tree node types for the Nomos parser.
//
// The AST represents the syntactic structure of Nomos configuration files.
// All node types include source location information via SourceSpan for
// precise error reporting and tooling support.
package ast

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
//	path: '../config'
type SourceDecl struct {
	Alias      string            `json:"alias"`
	Type       string            `json:"type"`
	Config     map[string]string `json:"config"` // Key-value configuration
	SourceSpan SourceSpan        `json:"source_span"`
}

// Span implements Node for SourceDecl.
func (s *SourceDecl) Span() SourceSpan { return s.SourceSpan }
func (s *SourceDecl) node()            {}
func (s *SourceDecl) stmt()            {}

// ImportStmt represents an import statement.
// Example: import:folder:filename
type ImportStmt struct {
	Alias      string     `json:"alias"`
	Path       string     `json:"path,omitempty"` // Optional nested path
	SourceSpan SourceSpan `json:"source_span"`
}

// Span implements Node for ImportStmt.
func (i *ImportStmt) Span() SourceSpan { return i.SourceSpan }
func (i *ImportStmt) node()            {}
func (i *ImportStmt) stmt()            {}

// ReferenceStmt represents a reference statement.
// Example: reference:folder:config.key
type ReferenceStmt struct {
	Alias      string     `json:"alias"`
	Path       string     `json:"path"`
	SourceSpan SourceSpan `json:"source_span"`
}

// Span implements Node for ReferenceStmt.
func (r *ReferenceStmt) Span() SourceSpan { return r.SourceSpan }
func (r *ReferenceStmt) node()            {}
func (r *ReferenceStmt) stmt()            {}

// SectionDecl represents a configuration section with key-value pairs.
// Example: config-section-name:
//
//	key1: value1
//	key2: value2
type SectionDecl struct {
	Name       string            `json:"name"`
	Entries    map[string]string `json:"entries"`
	SourceSpan SourceSpan        `json:"source_span"`
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
