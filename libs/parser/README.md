# Nomos Parser Library

A robust, concurrent-safe parser for the Nomos configuration scripting language (.csl files).

## Features

- **Concurrency-safe**: Parser instances can be used concurrently without data races
- **Precise error reporting**: All errors include file location and context snippets
- **Performance optimized**: Benchmarked for files up to 1MB with minimal allocations
- **Well-tested**: Comprehensive unit, integration, and concurrency tests
- **Stable AST**: Documented Abstract Syntax Tree with source location tracking

## What it achieves

The parser converts Nomos source text into a well-defined abstract syntax tree (AST). It is intentionally focused on syntax; semantics (imports, providers, references) are handled by the compiler.

- Tokenize and parse Nomos language constructs with helpful, deterministic errors.
- Produce an AST with stable public types for consumption by the compiler.
- Preserve source locations (file/line/column) on nodes for diagnostics.
- Keep runtime dependencies minimal and avoid global state.

## Relationship to other projects

- `libs/compiler` depends on the parser to obtain the AST used for analysis and code generation of the snapshot.
- The CLI does not import the parser directly.

```
CLI -> Compiler -> Parser
```

## Public API

### Basic Usage

```go
package main

import (
    "fmt"
    "log"

    "github.com/autonomous-bits/nomos/libs/parser"
)

func main() {
    // Parse a file from disk
    ast, err := parser.ParseFile("config.csl")
    if err != nil {
        log.Fatalf("Parse error: %v", err)
    }

    fmt.Printf("Parsed %d statements\n", len(ast.Statements))
}
```

### Parsing from a Reader

```go
package main

import (
    "log"
    "strings"

    "github.com/autonomous-bits/nomos/libs/parser"
)

func main() {
    input := `source:
	alias: 'folder'
	type:  'folder'
	path:  './config'
`
    
    reader := strings.NewReader(input)
    ast, err := parser.Parse(reader, "inline.csl")
    if err != nil {
        log.Fatalf("Parse error: %v", err)
    }

    // Process AST...
}
```

### Using Parser Instances (for pooling/reuse)

```go
package main

import (
    "sync"

    "github.com/autonomous-bits/nomos/libs/parser"
)

var parserPool = sync.Pool{
    New: func() interface{} {
        return parser.NewParser()
    },
}

func parseWithPooling(content string) error {
    p := parserPool.Get().(*parser.Parser)
    defer parserPool.Put(p)
    
    reader := strings.NewReader(content)
    _, err := p.Parse(reader, "pooled.csl")
    return err
}
```

## Workspace Setup

For local development in the Nomos monorepo, use the workspace sync script:

```bash
# From repository root
./tools/scripts/work-sync.sh
```

This will create or update the `go.work` file with all required modules (apps/command-line, libs/compiler, libs/parser).

func parseWithPool(filename string) (*parser.AST, error) {
    p := parserPool.Get().(*parser.Parser)
    defer parserPool.Put(p)

    return p.ParseFile(filename)
}
```

### Working with AST Nodes

The parser produces a strongly-typed AST with nodes representing different Nomos constructs:

```go
// Access statements in the AST
for _, stmt := range ast.Statements {
    switch s := stmt.(type) {
    case *ast.SourceDecl:
        fmt.Printf("Source: alias=%s, type=%s\n", s.Alias, s.Type)
    case *ast.ImportStmt:
        fmt.Printf("Import: alias=%s, path=%s\n", s.Alias, s.Path)
    case *ast.ReferenceStmt:
        fmt.Printf("Reference: alias=%s, path=%s\n", s.Alias, s.Path)
    case *ast.SectionDecl:
        fmt.Printf("Section: name=%s, entries=%d\n", s.Name, len(s.Entries))
    }
}
```

All AST nodes include source location information via `SourceSpan`:

```go
span := stmt.Span()
fmt.Printf("Location: %s:%d:%d\n", span.Filename, span.StartLine, span.StartCol)
```

## Error Handling

The parser provides deterministic, human-friendly error messages with precise source location information.

### Error Types

Parse errors are returned as `*parser.ParseError`, which includes:

- **Filename**: Source file name
- **Line** and **Column**: Error position (1-indexed)
- **Message**: Descriptive error message
- **Kind**: Error category (`LexError`, `SyntaxError`, or `IOError`)
- **Span**: Source location as `ast.SourceSpan`

### Error Kinds

```go
const (
    LexError     ParseErrorKind = iota  // Lexical/tokenization error
    SyntaxError                          // Grammar/syntax error
    IOError                              // File I/O error
)
```

### Basic Error Handling

```go
ast, err := parser.ParseFile("config.csl")
if err != nil {
    // Check if it's a parse error
    if parseErr, ok := err.(*parser.ParseError); ok {
        fmt.Printf("Parse error at %s:%d:%d: %s\n",
            parseErr.Filename(),
            parseErr.Line(),
            parseErr.Column(),
            parseErr.Message())
    } else {
        log.Fatal(err)
    }
}
```

### Programmatic Error Inspection

Parse errors can be inspected programmatically for custom handling:

```go
_, err := parser.ParseFile("config.csl")
if err != nil {
    if parseErr, ok := err.(*parser.ParseError); ok {
        switch parseErr.Kind() {
        case parser.SyntaxError:
            // Handle syntax error
            log.Printf("Syntax error: %s", parseErr.Message())
        case parser.LexError:
            // Handle lexical error
            log.Printf("Lexical error: %s", parseErr.Message())
        case parser.IOError:
            // Handle I/O error
            log.Printf("I/O error: %s", parseErr.Message())
        }
        
        // Access source location
        span := parseErr.Span()
        fmt.Printf("Error at %s:%d:%d\n", 
            span.Filename, span.StartLine, span.StartCol)
    }
}
```

### User-Friendly Error Formatting

Use `FormatParseError` to generate human-readable error messages with context snippets and caret markers:

```go
sourceText := "source:\n\talias: 'test'\nmy-section\n\tkey: 'value'"
_, err := parser.Parse(strings.NewReader(sourceText), "app.csl")
if err != nil {
    formatted := parser.FormatParseError(err, sourceText)
    fmt.Println(formatted)
}
```

Output:
```
app.csl:3:1: invalid syntax: expected ':' after identifier 'my-section'
   2 | 	alias: 'test'
   3 | my-section
   4 | 	key: 'value'
     | ^
```

The formatted error includes:
- **Machine-parseable prefix**: `file:line:col: message`
- **Context lines**: 1-3 lines showing the error location
- **Caret marker** (`^`): Points to the exact error position

**Note:** The caret position is rune-aware and correctly handles multi-byte UTF-8 characters.

### Error Formatting Best Practices

1. **Always provide source text** to `FormatParseError` for best error messages
2. **Log or display formatted errors** to end users for clarity
3. **Use programmatic inspection** when you need to handle different error types differently

```go
// Example: Log formatted error and handle programmatically
_, err := parser.ParseFile("config.csl")
if err != nil {
    if parseErr, ok := err.(*parser.ParseError); ok {
        // User-facing error
        sourceText, _ := os.ReadFile("config.csl")
        log.Println(parser.FormatParseError(parseErr, string(sourceText)))
        
        // Programmatic handling
        if parseErr.Kind() == parser.IOError {
            // Retry or suggest file path correction
        }
    }
}
```

All AST nodes include source span information for precise error reporting:

## Validation

The parser performs syntax validation during parsing to catch common errors early:

### Validated Syntax Errors

1. **Missing Colons After Keywords**
   - `source`, `import`, and `reference` keywords must be followed by `:`
   - Example: `source alias: 'test'` → Error: expected `:` after `source`

2. **Empty or Missing Aliases**
   - `source` declarations require a non-empty `alias` field
   - `import` statements require an alias
   - `reference` statements require both alias and path
   - Example: `source: alias: ''` → Error: non-empty alias required

3. **Incomplete Statements**
   - `reference` statements must include both alias and path separated by `:`
   - Example: `reference:my-alias` → Error: expected `:` after alias

4. **Unterminated Strings**
   - String literals must be properly terminated with matching quotes
   - Example: `key: 'value` → Error: unterminated string (missing closing ')

5. **Invalid Key Characters**
   - Key identifiers cannot start with special characters like `@`, `!`, `#`, etc.
   - Example: `@invalid: value` → Error: expected identifier for key

### Validation Examples

```go
// Valid syntax
validSource := `
source:
	alias: 'my-source'
	type: 'folder'
`

// Invalid: missing colon after keyword
invalidSource1 := `source
	alias: 'test'
`

// Invalid: empty alias
invalidSource2 := `
source:
	alias: ''
	type: 'test'
`

// Invalid: unterminated string
invalidSource3 := `
source:
	alias: 'unterminated
	type: 'test'
`

// All invalid examples will return *ParseError with specific validation messages
```

### Validation Limitations

Some validations are intentionally not performed:

- **Duplicate Keys**: Currently not validated (would require scope-aware detection for nested structures)
- **Indentation Consistency**: Non-indented content after a section simply terminates the section (results in empty section, which is valid)
- **Unknown Statements**: Unknown identifiers are treated as section declarations (by design)

```go
package main

import (
    "fmt"
    "log"

    "github.com/autonomous-bits/nomos/libs/parser"
    "github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
)

func main() {
    result, err := parser.ParseFile("config.csl")
    if err != nil {
        log.Fatal(err)
    }

    // Iterate over statements
    for i, stmt := range result.Statements {
        span := stmt.Span()
        fmt.Printf("Statement %d: %s:%d:%d-%d:%d\n",
            i,
            span.Filename,
            span.StartLine, span.StartCol,
            span.EndLine, span.EndCol,
        )

        // Type switch on statement kind
        switch s := stmt.(type) {
        case *ast.SourceDecl:
            fmt.Printf("  Source: alias=%s type=%s\n", s.Alias, s.Type)
        case *ast.ImportStmt:
            fmt.Printf("  Import: alias=%s path=%s\n", s.Alias, s.Path)
        case *ast.ReferenceStmt:
            fmt.Printf("  Reference: alias=%s path=%s\n", s.Alias, s.Path)
        case *ast.SectionDecl:
            fmt.Printf("  Section: name=%s entries=%d\n", s.Name, len(s.Entries))
        }
    }
}
```

## AST Types

The parser exports the following stable AST node types:

### Core Types

- **`AST`**: Root node containing all statements and source span
- **`SourceSpan`**: Source location with filename, line, and column information
- **`Node`**: Base interface for all AST nodes with `Span()` method

### Statement Types

- **`SourceDecl`**: Source provider declaration with alias, type, and configuration
- **`ImportStmt`**: Import statement with alias and optional path
- **`ReferenceStmt`**: Reference statement with alias and path
- **`SectionDecl`**: Configuration section with name and key-value entries

### Expression Types

- **`PathExpr`**: Dotted path expression (e.g., `config.key.value`)
- **`IdentExpr`**: Simple identifier
- **`StringLiteral`**: String literal value

All types include JSON tags for deterministic serialization, which is useful for testing and tooling.

## Error Handling

Parse errors include precise location information:

```go
ast, err := parser.ParseFile("bad.csl")
if err != nil {
    // Error message includes filename, line, and column
    // Example: "config.csl:5:10: invalid syntax: expected ':' after identifier 'foo'"
    fmt.Println(err)
}
```

## Language sketch (keywords)

From the repository README, Nomos supports the following surface:
- `source` — declare a source provider with alias and type
- `import` — import configuration from a source, optionally at a nested path
- `reference` — reference a specific value from a source

An illustrative (non-normative) EBNF sketch:

```
File        = { Stmt } .
Stmt        = SourceDecl | ImportStmt | ReferenceStmt .
SourceDecl  = "source" Ident TypeName SourceConfig .
ImportStmt  = "import" ":" Alias [ ":" Path ] .
ReferenceStmt = "reference" ":" Alias ":" Path .
Alias       = Ident .
Path        = Ident { "." Ident } .
```

Concrete token forms (as used in scripts):
- `import:{alias}` or `import:{alias}:{path_to_map}`
- `reference:{alias}:{path_to_property}`

Example configuration block:

```
source:
	alias: 'folder'
	type:  'folder'
	path:  '../config'

import:folder:filename

config-section-name:
	key1: value1
	key2: value2
```

## Error behavior

- Deterministic messages with a short description, file, line/column, and a small source snippet when feasible.
- No I/O beyond reading the provided input; no network access.

## Testing strategy

- Unit tests for each grammar rule and edge case.
- `testdata/` with positive and negative examples.
- Golden tests for AST rendering or token streams when useful.

## Versioning

- Tagged as `libs/parser/vX.Y.Z`. Follow semantic versioning for public AST/type changes.

## Development

### Running Tests

The parser includes comprehensive tests covering all grammar constructs, edge cases, and error scenarios.

```bash
# Run all tests
make test

# Run tests with verbose output
make test-verbose

# Run tests with race detector
make test-race

# Generate coverage report
make test-coverage
```

After running `make test-coverage`, open `coverage.html` in your browser to view detailed coverage information.

### Golden Tests

Golden tests verify that the parser produces consistent AST output. Golden files are stored in `testdata/golden/` as canonical JSON.

**Updating golden files** (only after reviewing and approving AST changes):

```bash
make update-golden
```

**Important:** Review all changes to golden files carefully before committing, as they define the expected behavior of the parser.

### Test Organization

```
libs/parser/
├── test/
│   ├── parser_api_test.go          # Public API tests
│   ├── parser_grammar_test.go      # Grammar construct tests
│   ├── golden_test.go              # Golden tests for positive cases
│   ├── golden_errors_test.go       # Golden tests for error cases
│   ├── errors_test.go              # Error model tests
│   ├── sourcespan_test.go          # Source span tests
│   └── integration/
│       └── parse_error_test.go     # Integration tests
└── testdata/
    ├── fixtures/                   # Valid .csl test files
    │   ├── simple.csl
    │   ├── complex_config.csl
    │   ├── unicode.csl
    │   └── negative/               # Invalid files for error testing
    ├── golden/                     # Expected AST JSON outputs
    │   ├── simple.csl.json
    │   └── errors/                 # Expected error outputs
    └── errors/                     # Error case fixtures
        ├── invalid_character.csl
        └── missing_colon_after_keyword.csl
```

### Benchmarking

Run performance benchmarks to ensure the parser meets performance targets:

```bash
make bench
```

### Linting

Ensure code quality by running the linter:

```bash
make lint
```

Or if you don't have `golangci-lint` installed:

```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
make lint
```

### Continuous Integration

The parser has automated CI checks that run on every pull request:

- **Tests**: All unit, integration, and golden tests
- **Coverage**: Minimum 80% code coverage required
- **Linting**: golangci-lint checks with strict rules
- **Build**: Ensures code compiles successfully
- **Race detection**: Concurrent access safety checks

See `.github/workflows/parser-ci.yml` for full CI configuration.

### Local Development Workflow

1. Make changes to parser code
2. Write tests first (TDD approach)
3. Run tests: `make test`
4. Check coverage: `make test-coverage`
5. Run linter: `make lint`
6. If AST changes are intentional, update golden files: `make update-golden`
7. Review all changes before committing
