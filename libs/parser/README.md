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

## Public API (proposed contract)

```go
package parser

type AST struct{ /* ... */ }

// ParseFile parses a file path into an AST.
func ParseFile(path string) (*AST, error)

// Parse parses from an io.Reader; useful for tests and embedding.
func Parse(r io.Reader, filename string) (*AST, error)
```

Key requirements:
- All errors include filename and line/column.
- AST node types (e.g., `SourceDecl`, `ImportStmt`, `ReferenceExpr`) are documented and stable.

File types:
- Parser targets files with the `.csl` extension by convention. Filenames should be preserved in diagnostics.

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
