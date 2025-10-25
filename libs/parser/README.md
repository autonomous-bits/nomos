# Nomos Parser Library

The parser converts Nomos source text into a well-defined abstract syntax tree (AST). It is intentionally focused on syntax; semantics (imports, providers, references) are handled by the compiler.

## What it should achieve

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

func parseWithPool(filename string) (*parser.AST, error) {
    p := parserPool.Get().(*parser.Parser)
    defer parserPool.Put(p)

    return p.ParseFile(filename)
}
```

### Working with AST Nodes

All AST nodes include source span information for precise error reporting:

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
