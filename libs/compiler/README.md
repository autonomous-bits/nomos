# Nomos Compiler Library

The compiler turns Nomos source files into a deterministic configuration snapshot. It is a small, stable Go library consumed by the CLI and other tools.

This README documents the compiler expectations and, importantly, the parser contract the compiler relies on. The compiler depends on `libs/parser` for a stable AST and precise parse errors — read the `Parser contract` section below for details you must rely on when implementing compilation, error reporting and diagnostics.

## Goals

- Parse Nomos scripts and build an internal representation (via the parser).
- Resolve `source`, `import` and inline `reference` constructs via pluggable provider APIs.
- Compose configuration deterministically with last-wins override semantics and deep-merge for maps.
- Detect and report cycles, missing references and provider errors with context-rich diagnostics.
- Produce a serializable snapshot (data + metadata) suitable for JSON/YAML/HCL rendering.

## Files accepted

- The compiler consumes files with the `.csl` extension and directories containing `.csl` files. Callers pass a path (file or directory) via `Options.Path`. The compiler must traverse directories in a deterministic order.

## Relationship to other projects

- Consumed by `apps/command-line` (the CLI).
- Depends on `libs/parser` for tokenizing/parsing and AST construction.

```
CLI -> Compiler -> Parser
```

## Public API (contract)

Keep the surface minimal and stable. Example public types used by consumers (CLI):

```go
package compiler

type Options struct {
        // Root path or file to compile.
        Path string
        // Provider registry, variables, timeouts etc.
        Providers ProviderRegistry
        Vars      map[string]any
}

type Snapshot struct {
        Data     map[string]any // fully-composed configuration
        Metadata Metadata       // sources, versions, timestamps
}

func Compile(opts Options) (Snapshot, error)
```

Notes:
- `Snapshot` is format-agnostic; the caller chooses JSON/YAML/HCL rendering.
- Providers are pluggable and are addressed by alias in scripts.

## Parser contract (what compiler relies on)

The compiler must consume the parser output and error types in specific ways. The parser (in `libs/parser`) exposes a stable set of entry points and AST shapes — the compiler must treat parsing errors as lower-level, recoverable diagnostics and use AST node spans to produce rich error messages.

Key parser behaviours the compiler relies on:

- Public parse entry points:
    - `parser.ParseFile(path string) (*ast.AST, error)` — convenience top-level function.
    - `parser.Parse(r io.Reader, filename string) (*ast.AST, error)` — parse from an arbitrary reader.
    - `parser.NewParser()` and `(*Parser).Parse*` allow reusing pooled parser instances.

- AST shape:
    - Root `*ast.AST` contains `Statements []ast.Stmt` and a `SourceSpan`.
    - Statement types include `ast.SourceDecl`, `ast.ImportStmt`, `ast.SectionDecl` (sections with map entries).
    - Values in section entries are `ast.Expr`. Supported value expressions currently include `*ast.StringLiteral` and `*ast.ReferenceExpr` for inline references.
    - `ReferenceExpr` has `Alias string`, `Path []string` and carries a `SourceSpan`.

- Inline references:
    - The parser treats inline references as first-class values: `key: reference:alias:dot.path` produces an `ast.ReferenceExpr` assigned as the entry value.
    - Top-level `reference:` statements (legacy) are rejected by the parser with a `SyntaxError` and a migration hint — the compiler should not expect top-level reference statements.

- Source spans and position information:
    - Every AST node includes `SourceSpan` with filename, start/end line and column.
    - Line and column numbers are 1-indexed. Columns are byte offsets (not rune counts); `EndCol` is inclusive (points to last byte of the token).

- Parse errors:
    - The parser returns `*parser.ParseError` with kinds `LexError`, `SyntaxError`, or `IOError` and includes filename/line/column and an optional snippet.
    - Use `parser.FormatParseError(err, sourceText)` to render caret-marked user-friendly messages.

- Validation scope:
    - The parser only enforces syntax-level rules (e.g., required `:` after keywords, properly-terminated strings, alias is a string literal).
    - Structural/semantic checks such as duplicate-key detection, provider existence, and reference resolution are intentionally left to the compiler.

Implication for the compiler:

- Treat `ReferenceExpr` nodes as value placeholders to be resolved by providers/import resolution. Do not attempt to interpret `reference:` syntax as a top-level statement — it's only valid as a value.
- When reporting errors about user code, prefer to decorate parser errors by using the `SourceSpan` and `FormatParseError` output so messages are consistent with parser diagnostics.
- Expect some syntax validation to already be performed by the parser; the compiler must perform semantic validation (duplicate keys, cycles, missing imports, provider errors).

## Composition and resolution semantics

- Imports are applied in evaluation order; on conflict keys are overwritten (last-wins).
- Maps are deep-merged; arrays replace by default.
- References (inline `ReferenceExpr`) are resolved after imports/values from providers are materialized, allowing cross-file linking and importing.
- Cycles across imports/references must be detected and reported by the compiler.

## Providers and sources

- Providers resolve data from backing systems (filesystem, Git, HTTP, cloud state).
- `source` declarations in the AST map to provider instances by alias and type.
- The compiler should use a provider registry to instantiate providers and cache provider results for the duration of a single compilation.

## Errors and diagnostics

- Use parser-provided `ParseError` for syntax/lexing faults. For semantic errors, return structured errors that include `SourceSpan` when possible.
- Wrap lower-level errors (`fmt.Errorf("...: %w", err)`) to preserve root causes.

## Example usage (compiler consumer)

```go
package main

import (
        "fmt"
        "github.com/autonomous-bits/nomos/libs/compiler"
)

func main() {
        snap, err := compiler.Compile(compiler.Options{Path: "./examples/dev"})
        if err != nil { panic(err) }
        fmt.Println(len(snap.Data))
}
```

## Testing strategy

- Unit tests for merge semantics, reference resolution and provider behavior (use fakes/mocks for providers).
- Integration tests compile fixtures in `testdata/` and assert snapshots. Keep tests deterministic and avoid external network calls.

## Versioning

- Tag releases as `libs/compiler/vX.Y.Z` and follow semantic versioning. Breaking API changes require a major version bump.

## Notes / Follow-ups

- Consider adding a small semantic validation pass (duplicate key detection, unresolved references) that runs after parsing but before provider resolution.
- The parser's `SourceSpan` semantics (1-indexed, byte-based columns, inclusive end column) are important when slicing source text for display — prefer using `parser.FormatParseError` for consistent caret rendering.
