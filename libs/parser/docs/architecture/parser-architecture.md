# Nomos Parser Architecture

**Module:** `libs/parser`  
**Author:** Architecture Review  
**Date:** October 26, 2025  
**Status:** Living Document

## Executive Summary

The Nomos Parser is a hand-written recursive descent parser that converts Nomos configuration source files (.csl) into a well-defined Abstract Syntax Tree (AST). It is designed to be concurrent-safe, performant, and provide precise error diagnostics with source location tracking.

The parser is intentionally focused on **syntax analysis only**—semantic validation (import resolution, reference checking, type validation) is delegated to the compiler module.

## Table of Contents

1. [Overview](#overview)
2. [Architecture Components](#architecture-components)
3. [Data Flow](#data-flow)
4. [Key Design Decisions](#key-design-decisions)
5. [AST Structure](#ast-structure)
6. [Error Handling](#error-handling)
7. [Performance Characteristics](#performance-characteristics)
8. [Testing Strategy](#testing-strategy)
9. [Future Considerations](#future-considerations)

## Overview

### Purpose

The parser serves as the front-end of the Nomos toolchain, transforming textual configuration scripts into a structured representation that can be analyzed and compiled.

### Design Goals

1. **Correctness**: Accurately parse valid Nomos syntax and reject invalid input with clear errors
2. **Performance**: Handle files up to 1MB efficiently with minimal memory allocations
3. **Concurrency Safety**: Allow multiple goroutines to parse simultaneously without data races
4. **Precise Diagnostics**: Provide file, line, and column information for all errors with context snippets
5. **Stability**: Maintain a stable public AST that downstream consumers can rely on

### Non-Goals

- **Semantic Validation**: The parser does not resolve imports, validate references, or check types
- **Optimization**: The parser does not optimize or transform the AST
- **Code Generation**: Output generation is handled by the compiler module

## Architecture Components

The parser is organized into four main layers:

```
┌─────────────────────────────────────────────┐
│           Public API Layer                  │
│  (parser.go - ParseFile, Parse functions)   │
└─────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────┐
│         Parser Implementation               │
│  (parser.go - parseStatement methods)       │
└─────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────┐
│           Scanner/Lexer                     │
│  (internal/scanner - tokenization)          │
└─────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────┐
│         AST Data Structures                 │
│  (pkg/ast - node types, SourceSpan)         │
└─────────────────────────────────────────────┘
```

### 1. Public API Layer (`parser.go`)

**Responsibilities:**
- Provide simple, ergonomic entry points for parsing
- Handle file I/O for filesystem-based parsing
- Manage parser instance lifecycle

**Key Functions:**
```go
// Package-level convenience functions
func ParseFile(path string) (*ast.AST, error)
func Parse(r io.Reader, filename string) (*ast.AST, error)

// Parser instance methods (for reuse/pooling)
type Parser struct{}
func NewParser(opts ...ParserOption) *Parser
func (p *Parser) ParseFile(path string) (*ast.AST, error)
func (p *Parser) Parse(r io.Reader, filename string) (*ast.AST, error)
```

**Design Notes:**
- Stateless parser instances enable concurrent usage
- Filename parameter is always required for error reporting
- I/O errors are wrapped as `ParseError` with `IOError` kind

### 2. Parser Implementation (`parser.go`)

**Responsibilities:**
- Orchestrate the parsing process
- Implement grammar rules as parsing methods
- Build AST nodes with correct source spans
- Handle syntax errors with context

**Key Methods:**
```go
func (p *Parser) parseStatements(s *scanner.Scanner, sourceText string) ([]ast.Stmt, error)
func (p *Parser) parseStatement(s *scanner.Scanner, sourceText string) (ast.Stmt, error)
func (p *Parser) parseSourceDecl(s *scanner.Scanner, startLine, startCol int, sourceText string) (*ast.SourceDecl, error)
func (p *Parser) parseImportStmt(s *scanner.Scanner, startLine, startCol int, sourceText string) (*ast.ImportStmt, error)
func (p *Parser) parseSectionDecl(s *scanner.Scanner, startLine, startCol int, sourceText string) (*ast.SectionDecl, error)
func (p *Parser) parseConfigBlock(s *scanner.Scanner) (map[string]ast.Expr, error)
func (p *Parser) parseValueExpr(s *scanner.Scanner, startLine, startCol int) (ast.Expr, error)
```

**Parsing Strategy:**
- **Recursive Descent**: Each grammar rule is implemented as a method
- **Single-Pass**: The parser makes one pass through the input
- **Lookahead**: Uses `PeekToken()` and `PeekChar()` for decision-making
- **Error Recovery**: Currently stops at first error (fail-fast approach)

### 3. Scanner/Lexer (`internal/scanner/scanner.go`)

**Responsibilities:**
- Character-level navigation through source text
- Track line and column positions
- Tokenize identifiers, keywords, and values
- Handle UTF-8 encoding correctly

**Key Type:**
```go
type Scanner struct {
    input     string  // Source text
    filename  string  // For error messages
    pos       int     // Current byte position
    line      int     // Current line (1-indexed)
    col       int     // Current column (1-indexed)
    lineStart int     // Byte position of line start
}
```

**Key Operations:**
```go
// Navigation
func (s *Scanner) PeekChar() rune
func (s *Scanner) Advance()
func (s *Scanner) IsEOF() bool

// Whitespace handling
func (s *Scanner) SkipWhitespace()
func (s *Scanner) SkipToNextLine()
func (s *Scanner) IsIndented() bool

// Token reading
func (s *Scanner) PeekToken() string
func (s *Scanner) ConsumeToken() string
func (s *Scanner) ReadIdentifier() string
func (s *Scanner) ReadPath() string
func (s *Scanner) ReadValue() string

// Position tracking
func (s *Scanner) Line() int
func (s *Scanner) Column() int
func (s *Scanner) Pos() int
```

**Design Notes:**
- Scanner is stateful but short-lived (one per parse operation)
- UTF-8 aware: uses `unicode/utf8` for multi-byte character handling
- Position tracking is 1-indexed (matches editor conventions)
- Indentation detection is significant (Nomos uses indentation for block structure)

### 4. AST Data Structures (`pkg/ast/types.go`)

**Responsibilities:**
- Define the abstract syntax tree node types
- Ensure all nodes have source location information
- Provide a stable, documented API for AST consumers

**Node Hierarchy:**
```
Node (interface)
├── AST (root node)
├── Stmt (statement interface)
│   ├── SourceDecl
│   ├── ImportStmt
│   ├── ReferenceStmt (deprecated - parse error)
│   └── SectionDecl
└── Expr (expression interface)
    ├── StringLiteral
    ├── ReferenceExpr (inline references)
    ├── PathExpr
    └── IdentExpr
```

**Core Types:**

```go
// Base types
type SourceSpan struct {
    Filename  string
    StartLine int  // 1-indexed
    StartCol  int  // 1-indexed
    EndLine   int  // 1-indexed, inclusive
    EndCol    int  // 1-indexed, inclusive
}

type Node interface {
    Span() SourceSpan
    node() // marker method
}

// Root node
type AST struct {
    Statements []Stmt
    SourceSpan SourceSpan
}

// Statements
type SourceDecl struct {
    Alias      string
    Type       string
    Config     map[string]Expr  // Key-value configuration
    SourceSpan SourceSpan
}

type ImportStmt struct {
    Alias      string
    Path       string  // Optional nested path
    SourceSpan SourceSpan
}

type SectionDecl struct {
    Name       string
    Entries    map[string]Expr  // Key-value pairs
    SourceSpan SourceSpan
}

// Expressions
type StringLiteral struct {
    Value      string
    SourceSpan SourceSpan
}

type ReferenceExpr struct {
    Alias      string    // Source alias
    Path       []string  // Dotted path components
    SourceSpan SourceSpan
}
```

**Design Notes:**
- All nodes implement the `Node` interface via `Span()` method
- Marker methods (`node()`, `stmt()`, `expr()`) prevent external types from implementing interfaces
- `SourceSpan` is inclusive on both start and end (editor-friendly)
- Expression values in sections/sources can be either `StringLiteral` or `ReferenceExpr`

## Data Flow

### High-Level Flow

```
┌──────────────┐
│  .csl File   │
│  or Reader   │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ Parse/       │
│ ParseFile    │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  io.ReadAll  │ ──────┐
└──────┬───────┘       │
       │               │
       ▼               │ (source text preserved
┌──────────────┐       │  for error snippets)
│   Scanner    │       │
│     New()    │◄──────┘
└──────┬───────┘
       │
       ▼
┌──────────────────────────────────────┐
│  parseStatements loop                │
│    └─► parseStatement                │
│          ├─► parseSourceDecl         │
│          ├─► parseImportStmt         │
│          ├─► parseSectionDecl        │
│          │     └─► parseConfigBlock  │
│          │           └─► parseValueExpr
│          └─► (error on reference)    │
└──────┬───────────────────────────────┘
       │
       ▼
┌──────────────┐
│   AST with   │
│  Statements  │
└──────────────┘
```

### Detailed Parsing Flow

1. **Initialization**
   - Read entire input into memory
   - Create Scanner with source text and filename
   - Preserve source text for error formatting

2. **Statement Loop** (`parseStatements`)
   - Skip whitespace and empty lines
   - For each non-empty line, call `parseStatement`
   - Accumulate statements into slice

3. **Statement Dispatch** (`parseStatement`)
   - Peek at first token
   - Dispatch to appropriate parser method:
     - `"source"` → `parseSourceDecl`
     - `"import"` → `parseImportStmt`
     - `"reference"` → Error (deprecated)
     - Other → `parseSectionDecl`

4. **Source Declaration** (`parseSourceDecl`)
   ```
   source:
       alias: 'folder'
       type: 'folder'
       path: './config'
   ```
   - Consume `source` keyword
   - Expect `:`
   - Parse config block (indented key-value pairs)
   - Extract and validate `alias` field (required)
   - Extract `type` field (optional)
   - Return `SourceDecl` node

5. **Import Statement** (`parseImportStmt`)
   ```
   import:alias:optional.path
   ```
   - Consume `import` keyword
   - Expect `:`
   - Read alias (required)
   - Optionally read `:` and path
   - Return `ImportStmt` node

6. **Section Declaration** (`parseSectionDecl`)
   ```
   section-name:
       key1: value1
       key2: reference:alias:path.to.value
   ```
   - Read section name
   - Expect `:`
   - Parse config block
   - Return `SectionDecl` node

7. **Config Block** (`parseConfigBlock`)
   - Loop while lines are indented
   - For each indented line:
     - Read key identifier
     - Expect `:`
     - Parse value expression
     - Store in map
   - Return map of entries

8. **Value Expression** (`parseValueExpr`)
   - Read value until end of line
   - Check if starts with `reference:`
     - Yes → Parse as `ReferenceExpr` (split by `:` and `.`)
     - No → Parse as `StringLiteral` (strip quotes)
   - Return expression node with source span

9. **AST Construction**
   - Collect all statements
   - Create AST node with full source span
   - Return to caller

## Key Design Decisions

### 1. Hand-Written vs. Generated Parser

**Decision:** Hand-written recursive descent parser

**Rationale:**
- Nomos grammar is relatively simple (4 statement types)
- Full control over error messages and recovery
- No external dependencies or build steps
- Easier to debug and maintain
- Better performance (no reflection, minimal allocations)

**Trade-offs:**
- More code to write initially
- Grammar changes require manual updates
- No automatic grammar validation

### 2. Single-Pass Parsing

**Decision:** One-pass parsing with no backtracking

**Rationale:**
- Nomos grammar is LL(1) - deterministic with one-token lookahead
- Simplifies implementation
- Better performance (linear time complexity)
- Easier to reason about

**Trade-offs:**
- Limited error recovery (currently stops at first error)
- Cannot handle ambiguous grammars

### 3. Indentation-Based Blocks

**Decision:** Use indentation to denote configuration blocks

**Rationale:**
- Matches Python/YAML style (familiar to users)
- Reduces visual noise (no braces)
- Natural for configuration files
- Enforces consistent formatting

**Implementation:**
- `Scanner.IsIndented()` checks if line starts with whitespace
- `parseConfigBlock()` loops while `IsIndented()` returns true
- Any whitespace at start of line counts as indented

### 4. Inline References as Expressions

**Decision:** References are first-class expression values, not statements

**Syntax:**
```
# Correct (inline reference as value)
cidr: reference:network:vpc.cidr

# Error (top-level reference - deprecated)
reference:network:vpc.cidr
```

**Rationale:**
- Clearer semantics: references are values, not imports
- More flexible: can be used anywhere a value is expected
- Simpler AST: no special reference resolution statements
- Explicit: references clearly indicate where values come from

**Migration:**
- Top-level `reference:` statements produce parse errors with migration guidance
- Parser enforces new syntax from the start

### 5. Error Handling Strategy

**Decision:** Fail-fast with rich error context

**Approach:**
- Stop at first error (no error recovery)
- Provide file, line, column for all errors
- Generate context snippets showing error location
- Include caret (`^`) pointing to exact position

**Rationale:**
- Simplifies parser logic (no error recovery complexity)
- First error is usually most important
- Rich context helps users fix issues quickly
- Configuration files are typically small (fixing and re-parsing is fast)

**Example Error:**
```
config.csl:5:12: invalid syntax: expected ':' after key
   5 |     vpc_cidr reference:network:vpc.cidr
     |             ^
```

### 6. UTF-8 Support

**Decision:** Full UTF-8 support throughout parser

**Implementation:**
- Use `unicode/utf8` package for decoding
- Track positions in runes, not bytes
- Handle multi-byte characters in identifiers and values

**Rationale:**
- Modern applications need Unicode support
- Configuration may include non-ASCII characters
- Go's standard library makes this straightforward

### 7. Stateless Parser Instances

**Decision:** Parser type has no mutable state

```go
type Parser struct {
    // Currently empty - may add options in future
}
```

**Rationale:**
- Enables concurrent parsing (multiple goroutines can share instance)
- Simplifies testing and reasoning
- State is localized to Scanner (which is short-lived)

**Benefits:**
- Thread-safe without locks
- Can pool parser instances
- Easier to reason about correctness

### 8. Source Span Tracking

**Decision:** All AST nodes include precise source location

**Design:**
```go
type SourceSpan struct {
    Filename  string
    StartLine int  // 1-indexed
    StartCol  int  // 1-indexed
    EndLine   int  // 1-indexed, inclusive
    EndCol    int  // 1-indexed, inclusive
}
```

**Rationale:**
- Essential for error reporting
- Enables IDE features (jump to definition, hover info)
- Supports future tooling (formatters, linters)
- Relatively low overhead (5 fields per node)

**Convention:**
- 1-indexed (matches editors and user expectations)
- Inclusive on both ends (end line/col points to last character)

## Error Handling

### Error Types

The parser defines a structured error type with semantic categories:

```go
type ParseErrorKind int

const (
    LexError     ParseErrorKind = iota  // Tokenization errors
    SyntaxError                          // Grammar violations
    IOError                              // File I/O failures
)

type ParseError struct {
    kind     ParseErrorKind
    filename string
    line     int
    col      int
    message  string
    snippet  string  // Context with caret
}
```

### Error Creation

Errors are created with full context:

```go
err := NewParseError(
    SyntaxError, 
    s.Filename(), 
    s.Line(), 
    s.Column(), 
    "expected ':' after key",
)
err.SetSnippet(generateSnippetFromSource(sourceText, s.Line(), s.Column()))
return nil, err
```

### Error Formatting

The `FormatParseError` function provides rich, human-readable output:

```go
func FormatParseError(err error, sourceText string) string
```

**Output Format:**
```
file:line:col: message
  line-1 | context before
  line   | error line
         | ^
  line+1 | context after
```

**Example:**
```
config.csl:12:5: invalid syntax: expected ':' after key
  11 |     region: 'us-west-2'
  12 |     cidr reference:network:vpc.cidr
     |          ^
  13 | 
```

### Error Recovery

**Current Approach:** None (fail-fast)

**Rationale:**
- Configuration files are typically small
- First error is usually root cause
- Fast iteration (fix and re-parse)
- Simpler implementation

**Future Enhancement:**
- Could add limited recovery for IDE use cases
- Continue parsing to find multiple errors
- Would require careful design to avoid cascading errors

## AST Structure

### Design Principles

1. **Minimal but Complete**: Include only what's needed for compilation
2. **Stable**: Changes to AST should be rare and well-communicated
3. **Self-Documenting**: Clear types and field names
4. **Location-Aware**: All nodes know where they came from

### Node Categories

#### Statements (top-level constructs)

1. **SourceDecl** - Source provider configuration
   ```go
   type SourceDecl struct {
       Alias      string              // Provider alias
       Type       string              // Provider type (optional)
       Config     map[string]Expr     // Configuration key-values
       SourceSpan SourceSpan
   }
   ```

2. **ImportStmt** - Import from source
   ```go
   type ImportStmt struct {
       Alias      string    // Source alias
       Path       string    // Optional nested path
       SourceSpan SourceSpan
   }
   ```

3. **SectionDecl** - Configuration section
   ```go
   type SectionDecl struct {
       Name       string              // Section name
       Entries    map[string]Expr     // Key-value pairs
       SourceSpan SourceSpan
   }
   ```

#### Expressions (values)

1. **StringLiteral** - Literal string value
   ```go
   type StringLiteral struct {
       Value      string
       SourceSpan SourceSpan
   }
   ```

2. **ReferenceExpr** - Reference to external value
   ```go
   type ReferenceExpr struct {
       Alias      string      // Source/import alias
       Path       []string    // Dotted path components
       SourceSpan SourceSpan
   }
   ```

### AST Traversal

**Current State:** Manual traversal by compiler

**Example:**
```go
for _, stmt := range ast.Statements {
    switch s := stmt.(type) {
    case *ast.SourceDecl:
        // Process source
    case *ast.ImportStmt:
        // Process import
    case *ast.SectionDecl:
        // Process section
    }
}
```

**Future:** Could add Visitor pattern for more complex traversals

### AST Immutability

**Philosophy:** AST is immutable after parsing

**Benefits:**
- Safe to share across goroutines
- Easier to reason about
- Compiler can cache ASTs

**Note:** Current implementation doesn't enforce this (maps are mutable), but convention is to treat AST as read-only.

## Performance Characteristics

### Benchmarks

From `parser_bench_test.go`:

```
BenchmarkParseFile/empty-10              3256 ns/op      928 B/op   19 allocs/op
BenchmarkParseFile/small-10              5623 ns/op     2272 B/op   45 allocs/op
BenchmarkParseFile/medium-10            32847 ns/op    12784 B/op  258 allocs/op
BenchmarkParseFile/large-10            329845 ns/op   128560 B/op 2579 allocs/op
BenchmarkParseFile/xlarge-10          3298567 ns/op  1285360 B/op 25790 allocs/op
```

### Performance Characteristics

- **Time Complexity:** O(n) where n is file size in bytes
- **Space Complexity:** O(n) for AST storage
- **Allocation Rate:** ~10 allocations per statement
- **Throughput:** ~300 KB/s on typical hardware

### Optimization Strategies

1. **Pre-allocate Maps:** Known-size maps could be pre-allocated
2. **String Interning:** Repeated identifiers could be interned
3. **Scanner Pooling:** Reuse scanner buffers
4. **AST Pooling:** Pool AST nodes for high-throughput scenarios

**Current Status:** Premature optimization avoided; current performance is acceptable for expected workloads (files < 100KB).

## Testing Strategy

### Test Categories

1. **Unit Tests** (`*_test.go` in packages)
   - Scanner methods
   - AST node creation
   - Error formatting

2. **API Tests** (`test/parser_api_test.go`)
   - Public API contracts
   - Basic parsing scenarios
   - Error handling

3. **Grammar Tests** (`test/parser_grammar_test.go`)
   - Each grammar construct
   - Boundary cases
   - Invalid syntax

4. **Golden Tests** (`test/golden_test.go`)
   - Parse `.csl` files from `testdata/fixtures/`
   - Compare AST (as JSON) to `.json` golden files
   - Ensures AST stability

5. **Error Golden Tests** (`test/golden_errors_test.go`)
   - Parse invalid `.csl` files from `testdata/errors/`
   - Compare error output to `.error.json` golden files
   - Ensures error message stability

6. **Integration Tests** (`test/integration/`)
   - Concurrency safety
   - Large file handling
   - Edge cases

### Test Data Organization

```
testdata/
├── fixtures/           # Valid .csl files
│   ├── simple.csl
│   ├── complex_config.csl
│   └── ...
├── golden/            # Expected AST outputs
│   ├── simple.csl.json
│   ├── complex_config.csl.json
│   └── errors/       # Expected error outputs
│       └── *.error.json
└── errors/           # Invalid .csl files
    ├── invalid_character.csl
    ├── unterminated_string.csl
    └── ...
```

### Coverage Goals

- **Line Coverage:** > 90%
- **Branch Coverage:** > 85%
- **Critical Paths:** 100% (error handling, main parse loop)

### Testing Best Practices

1. **Table-Driven Tests:** Use for similar test cases
2. **Golden Files:** Keep AST structure documented via examples
3. **Error Messages:** Test exact error text (ensures quality)
4. **Concurrency:** Explicit concurrency tests (not just race detector)

## Future Considerations

### Potential Enhancements

1. **Error Recovery**
   - Continue parsing after errors
   - Collect multiple errors in one pass
   - Useful for IDE integration

2. **Streaming Parser**
   - Parse large files without loading into memory
   - Trade-off: more complex implementation

3. **AST Visitor Pattern**
   - Standardized traversal interface
   - Easier for compiler and tools

4. **Comments in AST**
   - Preserve comments for documentation tools
   - Formatters could preserve original formatting

5. **Position-to-Node Mapping**
   - Fast lookup: given position, find AST node
   - Enables "jump to definition" in editors

6. **Incremental Parsing**
   - Re-parse only changed sections
   - Performance boost for IDE scenarios

7. **Parser Directives**
   - Allow parser configuration in source files
   - Example: `# parser: strict-indentation`

### Language Evolution

If Nomos grammar expands, consider:

1. **Expression Types**
   - Numbers, booleans, arrays, maps
   - Complex expressions (arithmetic, concatenation)

2. **Conditionals**
   - Environment-based configuration
   - Feature flags

3. **Functions/Macros**
   - Code reuse within configurations
   - Template-like capabilities

4. **Type Annotations**
   - Explicit type declarations
   - Earlier error detection

### Tooling Opportunities

1. **Parser as a Service**
   - HTTP API for parsing
   - Useful for web-based editors

2. **Language Server Protocol**
   - Full IDE integration
   - Real-time diagnostics, completions

3. **Formatter**
   - Canonical formatting
   - Integrate with parser to preserve structure

4. **Linter**
   - Style checks beyond syntax
   - Best practice enforcement

## Conclusion

The Nomos parser is a well-designed, hand-crafted component that successfully balances simplicity, performance, and usability. Its focus on precise error reporting and stable AST structure makes it an excellent foundation for the Nomos toolchain.

**Key Strengths:**
- Clear, maintainable code
- Excellent error diagnostics
- Concurrent-safe design
- Comprehensive test coverage

**Areas for Future Work:**
- Error recovery for multi-error reporting
- Performance optimizations for very large files
- Enhanced tooling integration

The architecture is well-suited for the current requirements and has clear paths for future evolution as the Nomos language and ecosystem grow.
