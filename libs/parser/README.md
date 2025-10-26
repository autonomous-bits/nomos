# Nomos Parser Library

This package parses Nomos configuration files (.csl) into a stable, strongly-typed
Abstract Syntax Tree (AST) used by the compiler.

The README below documents the parser's current public surface and behaviour as
implemented in `parser.go`, `errors.go` and the AST types under `pkg/ast`.

See `docs/architecture/` for higher-level design docs and diagrams.

## Highlights / Implementation notes

- Public parsing entry points: `ParseFile(path string) (*ast.AST, error)` and
  `Parse(r io.Reader, filename string) (*ast.AST, error)`.
- You can create and reuse parser instances via `NewParser()` and call
  the instance methods `(*Parser).ParseFile` / `(*Parser).Parse`.
- Inline references are first-class expressions (`ast.ReferenceExpr`) and have the
  form `reference:alias:dotted.path` when used as a value. Top-level
  `reference:` statements are rejected (deprecated) and the parser returns a
  `SyntaxError` with a migration hint.
- Parse errors are represented by `*parser.ParseError` with kinds
  `LexError`, `SyntaxError`, or `IOError`. Errors include filename, line/column
  and an optional context snippet. Use `FormatParseError` to render a
  human-friendly message with a caret marker.

## Public API (summary)

- NewParser(opts ...ParserOption) *Parser
  - Create parser instances that can be reused (good for pooling).

- ParseFile(path string) (*ast.AST, error)
  - Convenience top-level function that reads from disk and parses.

- Parse(r io.Reader, filename string) (*ast.AST, error)
  - Parse from an arbitrary reader; `filename` is used in error messages
    and node source spans.

Returned AST:
- `*ast.AST` with `Statements []ast.Stmt` and a top-level `SourceSpan`.

Quick example (file):

```go
ast, err := parser.ParseFile("config.csl")
if err != nil {
    return err
}
fmt.Printf("parsed %d statements\n", len(ast.Statements))
```

Quick example (reader):

```go
r := strings.NewReader(sourceText)
ast, err := parser.Parse(r, "inline.csl")
```

Pooling reuse example:

```go
pool := sync.Pool{New: func() interface{} { return parser.NewParser() }}
p := pool.Get().(*parser.Parser)
defer pool.Put(p)
_, _ = p.Parse(strings.NewReader(source), "p.csl")
```

## AST and expressions

- Section declarations, `source` declarations, and `import` statements are
  represented as concrete `ast.Stmt` implementations (e.g. `ast.SourceDecl`,
  `ast.ImportStmt`, `ast.SectionDecl`).
- Values in key/value pairs are `ast.Expr` values. Currently supported value
  expressions include:
  - `*ast.StringLiteral` — plain string values (the parser strips quotes)
  - `*ast.ReferenceExpr` — inline reference values parsed from
    `reference:alias:path.to.value` (the parser splits the dotted path into
    components)

All AST nodes carry `ast.SourceSpan` (filename, start/end line/column) which
is used by the compiler and error reporting.

## Inline references and migration note

The parser implements inline references as value expressions:

  key: reference:someAlias:parent.child

This produces an `ast.ReferenceExpr{Alias: "someAlias", Path: []string{"parent","child"}, ...}`.

Top-level `reference:` statements (e.g. `reference:alias:path`) are rejected by
the parser with a `SyntaxError` and a clear migration message. The compiler
should expect references to appear in value positions.

## Error handling

Parse errors are returned as `*ParseError` with the following properties:

- Kind: one of `LexError`, `SyntaxError`, `IOError`
- Filename, Line (1-indexed), Column (1-indexed)
- Message: human-friendly description
- Snippet: optional context lines with a caret marker (set by the parser or
  generated via `FormatParseError`)

Use `FormatParseError(err, sourceText)` to get a multi-line, user-friendly
message that includes context lines and a caret pointing to the error. If you
need programmatic handling, assert `err.(*parser.ParseError)` and inspect
`Kind()` and `Span()`.

Example programmatic handling:

```go
ast, err := parser.ParseFile("config.csl")
if err != nil {
    if pe, ok := err.(*parser.ParseError); ok {
        switch pe.Kind() {
        case parser.SyntaxError:
            // handle
        case parser.LexError:
            // handle
        case parser.IOError:
            // handle
        }
    }
}
```

## Validation rules enforced by the parser

The parser performs syntax-level validation and returns `SyntaxError` for
violations. Key rules implemented in the code:

- Keywords `source` and `import` must be followed by `:` (otherwise SyntaxError).
- `source` declarations require a non-empty string `alias` field; the alias
  must be a string literal (not a reference).
- `import` requires an alias; an optional `:path` may follow (parsed as
  identifier-like token after a second `:`).
- Top-level `reference:` statements are rejected (deprecated) — use inline
  `reference:alias:dot.path` values.
- String values must be properly terminated; unterminated strings produce
  `SyntaxError` describing the missing closing quote.
- Key identifiers must be valid (empty key or invalid start character results
  in `SyntaxError`).

Notably the parser currently does NOT enforce some higher-level checks:

- Duplicate key detection is not performed (comment in code: requires scope-aware
  detection).
- Some structural/semantic checks (unknown provider types, import resolution,
  reference resolution) are intentionally left to the compiler.

## Error formatting details

Use `parser.FormatParseError(err, sourceText)` to produce a formatted message
with a machine-parsable prefix (`file:line:col: message`), the surrounding
context lines, and a caret marker. The caret logic is rune-aware so UTF-8
characters are handled correctly.

## Workspace / development

To include this module in a local Go workspace, run the repo helper script
from the project root:

```bash
./tools/scripts/work-sync.sh
```

## Tests and quality

This package contains unit, integration and concurrency tests in `test/` and
`internal/` subpackages. The parser code is intentionally minimal in
dependencies to make testing deterministic. Run `go test ./...` from the
repository root (or from the `libs/parser` module) to run the suite.

## Notes / future work

- Consider adding duplicate-key detection and scope-aware validation in the
  parser or a separate semantic validation pass.
- Expand supported expression types (arrays, maps) if language evolves.

---

If you want a change in this README (more examples, additional AST docs, or
code snippets), tell me what you'd like and I will update it.

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

- **`ReferenceExpr`**: Inline reference value (`reference:{alias}:{path}`) used anywhere a value is allowed. Section entry values are expressions (`Expr`) and can be either a `StringLiteral` or a `ReferenceExpr`. The legacy top-level `ReferenceStmt` will be removed in a future major version.

All types include JSON tags for deterministic serialization, which is useful for testing and tooling.

## Inline Reference Syntax

Nomos supports **inline references** as first-class values. References allow you to reference values from imported sources or other sections directly in value positions, eliminating the need for separate reference declarations.

### Syntax

The inline reference syntax follows the pattern: `reference:{alias}:{path}`

- **`reference`**: Keyword indicating a reference expression
- **`{alias}`**: The source or import alias to reference
- **`{path}`**: A dotted path to the target property (e.g., `vpc.cidr` or `database.connection.host`)

### Examples

#### Scalar Value

```csl
infrastructure:
  vpc_cidr: reference:network:vpc.cidr
  region: 'us-west-2'
```

The `vpc_cidr` value is an inline reference to `vpc.cidr` from the `network` source.

#### Map/Collection Context

```csl
servers:
  web:
    ip: reference:network:web.ip
    port: '8080'
  api:
    ip: reference:network:api.ip
    port: '3000'
```

Multiple inline references can be used within the same section, allowing dynamic configuration across different keys.

#### List Context (Nested Maps)

```csl
databases:
  primary:
    host: reference:infra:db.primary.host
    port: '5432'
  replica:
    host: reference:infra:db.replica.host
    port: '5433'
```

Inline references work seamlessly in deeply nested structures.

#### Mixed Literals and References

```csl
application:
  name: 'my-app'
  database_url: reference:config:database.url
  debug: 'false'
  api_key: reference:secrets:api.key
```

You can freely mix string literals and inline references within the same section.

### AST Representation

When the parser encounters an inline reference, it creates a `ReferenceExpr` node in the AST:

```json
{
  "type": "ReferenceExpr",
  "alias": "network",
  "path": ["vpc", "cidr"],
  "source_span": {
    "filename": "config.csl",
    "start_line": 3,
    "start_col": 13,
    "end_line": 3,
    "end_col": 40
  }
}
```

**Fields:**
- **`alias`** (string): The source alias to resolve (e.g., `"network"`)
- **`path`** ([]string): Array of path components (e.g., `["vpc", "cidr"]`)
- **`source_span`** (SourceSpan): Precise source location covering the entire `reference:alias:path` token

### Working with ReferenceExpr in Code

```go
package main

import (
    "fmt"
    "log"

    "github.com/autonomous-bits/nomos/libs/parser"
    "github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
)

func main() {
    source := `
infrastructure:
  vpc_cidr: reference:network:vpc.cidr
  region: 'us-west-2'
`
    
    result, err := parser.Parse(strings.NewReader(source), "example.csl")
    if err != nil {
        log.Fatal(err)
    }
    
    // Find the infrastructure section
    for _, stmt := range result.Statements {
        if section, ok := stmt.(*ast.SectionDecl); ok && section.Name == "infrastructure" {
            // Iterate over entries
            for key, expr := range section.Entries {
                switch e := expr.(type) {
                case *ast.ReferenceExpr:
                    fmt.Printf("%s is a reference to %s:%s\n", 
                        key, e.Alias, strings.Join(e.Path, "."))
                case *ast.StringLiteral:
                    fmt.Printf("%s is a literal: %s\n", key, e.Value)
                }
            }
        }
    }
}
```

**Output:**
```
vpc_cidr is a reference to network:vpc.cidr
region is a literal: us-west-2
```

### Benefits of Inline References

1. **Co-location**: References appear exactly where values are used, reducing cognitive overhead
2. **Clarity**: No need to track separate reference declarations
3. **Flexibility**: Mix references and literals freely within the same section
4. **Type Safety**: The parser enforces correct syntax and provides precise error messages

### Migration from Legacy Top-Level References

**IMPORTANT:** Top-level `reference:` statements have been **removed** as of this version. This is a **breaking change**.

Legacy syntax (no longer supported):
```csl
reference:network:vpc.cidr

infrastructure:
  vpc_cidr: ???  # How do we reference it?
```

New inline syntax (required):
```csl
infrastructure:
  vpc_cidr: reference:network:vpc.cidr
```

For migration assistance and detailed guidance, see:
- **PRD Issue**: [#10 - Inline References as First-Class Values](https://github.com/autonomous-bits/nomos/issues/10)
- **Codemod Script**: `tools/scripts/convert-top-level-references` (automated conversion tool)
- **Migration Guide**: See the "Migration Notes" section below

The codemod script can automatically convert legacy top-level references to inline references for you.

### SourceSpan Behavior

All AST nodes include precise source location information through the `SourceSpan` type:

```go
type SourceSpan struct {
    Filename  string
    StartLine int  // 1-indexed line number
    StartCol  int  // 1-indexed column (byte position, not rune)
    EndLine   int  // 1-indexed line number
    EndCol    int  // 1-indexed column (byte position, inclusive)
}
```

**Important characteristics:**

- **1-indexed**: All line and column numbers start at 1 (not 0)
- **Byte-based columns**: Column positions are byte offsets, not character (rune) counts. For ASCII text, bytes and characters are the same, but for Unicode text (e.g., Japanese 日本), multi-byte characters will consume multiple column positions.
- **Inclusive EndCol**: The `EndCol` points to the **last byte** of the token, not one-past-the-end. To extract text: `line[StartCol-1:EndCol]`

**Example:**

For the input line `"  key: reference:net:日本"` (where 日本 is 6 bytes):
- StartCol = 8 (first byte of "reference")
- EndCol = 27 (last byte of "本")
- Token length = `EndCol - StartCol + 1 = 20 bytes`

**ReferenceExpr span accuracy:**

`ReferenceExpr` nodes capture the **entire inline reference token** including the `reference:` keyword, alias, and dotted path. For example:

```csl
section:
  cidr: reference:network:vpc.cidr
```

The `ReferenceExpr` span for this value covers columns 9-36 (assuming 2-space indent), encompassing the complete text `"reference:network:vpc.cidr"` (28 bytes).

This precision enables:
- Accurate error reporting with caret positioning
- IDE features like go-to-definition and hover tooltips
- Source code transformations and refactoring tools
- Precise AST-to-source-text mappings for round-tripping

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

Current high-level grammar sketch (non-normative but reflects parser behavior):

```
File            = { Stmt } .
Stmt            = SourceDecl | ImportStmt | SectionDecl .
SourceDecl      = "source" ":" NewLine SourceConfig .
ImportStmt      = "import" ":" Alias [ ":" Path ] .
Alias           = Ident .
Path            = Ident { "." Ident } .

# Expression-valued entries (references are values, not top-level statements)
SectionDecl     = Ident ":" NewLine IndentedEntries .
IndentedEntries = { Indent Key ":" Value NewLine } .
Key             = Ident .
Value           = String | Expr .              # A key's value can be a string literal or a reference
Expr            = ReferenceExpr | IdentExpr | PathExpr .
ReferenceExpr   = "reference" ":" Alias ":" Path .
```

Legacy acceptance (to be removed in a future major version):

```
ReferenceStmt = "reference" ":" Alias ":" Path .   // deprecated
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

Inline reference example (recommended):

```
infrastructure:
  vpc:
        cidr: reference:network:vpc.cidr   # reference value
        name: 'prod-vpc'                   # string value
```

Inline references are parsed as `ReferenceExpr` nodes and section entry values are expression-backed; the parser produces `ReferenceExpr` for inline references and will reject legacy top-level `reference:` statements.

## Error behavior

- Deterministic messages with a short description, file, line/column, and a small source snippet when feasible.
- No I/O beyond reading the provided input; no network access.

## Testing strategy

- Unit tests for each grammar rule and edge case.
- `testdata/` with positive and negative examples.
- Golden tests for AST rendering or token streams when useful.

## Versioning

- Tagged as `libs/parser/vX.Y.Z`. Follow semantic versioning for public AST/type changes.

Deprecation and breaking-change notice:

- Top-level `ReferenceStmt` is deprecated and will be removed in a future major version.
- Inline reference support as first-class expressions will change section entry values from `map[string]string` to an expression-backed representation. This will ship in a major version with migration notes and updated golden tests.

Migration guidance:

- Replace any top-level `reference:{alias}:{path}` with an inline value under the appropriate section key.
- Update consumers that rely on `SectionDecl.Entries` to handle expression values (when the new major version lands).

## Migration Notes

### Breaking Change: Removal of Top-Level `reference:` Statements

As of this version, the parser **no longer supports** top-level `reference:` statements. This is a **breaking change** introduced to standardize on inline references as the canonical syntax.

#### What Changed

**Legacy Syntax (REMOVED):**
```csl
source:
  alias: 'network'
  type: 'folder'
  path: './network'

reference:network:vpc.cidr
reference:network:subnet.mask

infrastructure:
  # These references were declared above, but how to use them?
  vpc: ???
```

**New Syntax (REQUIRED):**
```csl
source:
  alias: 'network'
  type: 'folder'
  path: './network'

infrastructure:
  vpc_cidr: reference:network:vpc.cidr
  subnet_mask: reference:network:subnet.mask
```

#### Why This Change?

1. **Co-location**: References now appear exactly where values are used, reducing cognitive overhead and improving readability
2. **Clarity**: No ambiguity about where references are consumed
3. **Consistency**: All value positions support the same expression syntax (strings or references)
4. **Reduced Boilerplate**: Eliminates separate reference declaration blocks

#### Migration Path

**Option 1: Automated Migration with Codemod**

Use the provided codemod script to automatically convert legacy references:

```bash
# From repository root
./tools/scripts/convert-top-level-references path/to/config.csl
```

The script will:
- Identify all top-level `reference:` statements
- Convert them to inline `reference:` expressions in appropriate value positions
- Preserve comments and formatting where possible
- Generate a backup file (`.csl.bak`)

**Option 2: Manual Migration**

For each top-level `reference:alias:path` statement:

1. Remove the top-level `reference:` line
2. Find the section where the value should be used
3. Add a key-value pair with an inline reference: `key: reference:alias:path`

**Before:**
```csl
source:
  alias: 'config'
  type: 'folder'

reference:config:database.host
reference:config:database.port

application:
  # How to use the references?
```

**After:**
```csl
source:
  alias: 'config'
  type: 'folder'

application:
  db_host: reference:config:database.host
  db_port: reference:config:database.port
```

#### Error Messages

If the parser encounters a legacy top-level `reference:` statement, it will produce a helpful error:

```
config.csl:5:1: top-level 'reference:' statements are no longer supported.
Use inline references instead. Example: Instead of a top-level 'reference:alias:path',
use 'key: reference:alias:path' in a value position.

   4 | 
   5 | reference:network:vpc.cidr
   6 | 
     | ^
```

#### Additional Resources

- **Product Requirements**: [Issue #10 - Inline References as First-Class Values](https://github.com/autonomous-bits/nomos/issues/10)
- **Architecture Discussion**: See PRD Architecture Review section in Issue #10
- **Codemod Source**: `tools/scripts/convert-top-level-references`
- **Example Fixtures**: `libs/parser/testdata/fixtures/inline_ref_*.csl`

#### AST Changes for Library Consumers

If your code consumes the parser's AST, note the following changes:

**1. Section Entries are Now Expressions**

```go
// OLD (no longer valid):
// section.Entries is map[string]string

// NEW (current):
// section.Entries is map[string]Expr
for key, expr := range section.Entries {
    switch e := expr.(type) {
    case *ast.StringLiteral:
        fmt.Printf("%s: %s\n", key, e.Value)
    case *ast.ReferenceExpr:
        fmt.Printf("%s: ref to %s:%s\n", key, e.Alias, strings.Join(e.Path, "."))
    }
}
```

**2. ReferenceStmt is Deprecated**

The `ReferenceStmt` AST node type is no longer produced by the parser. Use `ReferenceExpr` in value positions instead:

```go
// OLD (deprecated):
// type ReferenceStmt struct { Alias, Path string, ... }

// NEW (use this):
// type ReferenceExpr struct { Alias string, Path []string, SourceSpan }
```

**3. Migration Timeline**

- **Current Version**: Top-level references are rejected with error
- **Recommended Action**: Update all `.csl` files to use inline references
- **Tooling Support**: Codemod script available for automated conversion

#### Frequently Asked Questions

**Q: Can I still use the old syntax?**  
A: No. Top-level `reference:` statements are no longer supported. The parser will return an error if it encounters them.

**Q: What about backward compatibility?**  
A: This is a **breaking change**. Since Nomos has not been officially released, we are making this change now to standardize on the better inline syntax before the 1.0 release.

**Q: Will my existing config files still work?**  
A: No. You must migrate to inline references using the codemod script or manual migration.

**Q: Where can I get help?**  
A: See Issue #10 for detailed examples, or open a new issue on GitHub if you encounter migration challenges.

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

## Documentation

### For Users

- **[README](./README.md)** (this file) - Quick start, API reference, and usage examples
- **[Examples](./docs/examples/)** - Example Nomos configuration files demonstrating language features
- **[CHANGELOG](./CHANGELOG.md)** - Version history and release notes

### For Developers

- **[Architecture Documentation](./docs/architecture/)** - Comprehensive technical documentation
  - [Parser Architecture](./docs/architecture/parser-architecture.md) - Design, implementation, and key decisions
  - [Architecture Diagrams](./docs/architecture/diagrams.md) - Visual reference for data flow and structure
  - [Architecture Index](./docs/architecture/README.md) - Documentation overview and quick reference
- **[Development Guide](./AGENTS.md)** - Module layout and development guidelines
- **[Monorepo Structure](../../docs/architecture/go-monorepo-structure.md)** - Repository-wide organization patterns

### Quick Links

- **Understanding the Parser**: Start with [Parser Architecture](./docs/architecture/parser-architecture.md)
- **Visual Reference**: See [Architecture Diagrams](./docs/architecture/diagrams.md)
- **Contributing**: Read [Architecture Index](./docs/architecture/README.md) and [AGENTS.md](./AGENTS.md)
- **AST Structure**: Detailed in [Parser Architecture § AST Structure](./docs/architecture/parser-architecture.md#ast-structure)
- **Error Handling**: See [Parser Architecture § Error Handling](./docs/architecture/parser-architecture.md#error-handling)
7. Review all changes before committing
