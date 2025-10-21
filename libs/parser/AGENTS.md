# AGENTS: libs/parser

This document explains the `libs/parser` module layout and development guidance. It is scoped to the parser library which provides parsing capabilities for Nomos language sources. It complements repository-level guidance in `docs/architecture/go-monorepo-structure.md` but focuses on the local module.

Purpose
-------

The `libs/parser` module is responsible for turning Nomos source text into an AST or token stream suitable for further processing by the `libs/compiler` module. The parser should offer a clear programmatic API and stable data structures for downstream consumers.

Repository layout (local)
-------------------------

Common files and directories:

- `go.mod` / `go.sum` — module declaration and dependencies.
- `pkg/` or `parser.go` — public API surface (e.g., `Parse(reader io.Reader) (*AST, error)`).
- `internal/` — implementation details such as scanner/lexer, grammar-driven code, tree walkers, and helpers.
- `grammar/` — generated parser code or grammar definitions if using a generator (optional).
- `test/` and `testdata/` — unit and integration tests for parser behavior and edge cases.
- `examples/` — example programs demonstrating how to use the parser API.

Example layout:

```
libs/parser/
├── go.mod
├── go.sum
├── parser.go          # public entry points
├── pkg/               # optional public packages with AST definitions
├── grammar/           # grammar files or generated parser code
├── internal/
│   ├── scanner/
│   └── tree/
├── test/
└── testdata/
```

Public API contract (suggested)
-------------------------------

Provide a simple, dependency-light API for the parser:

- `Parse(r io.Reader) (*AST, error)` — parse input and return an AST
- `ParseFile(path string) (*AST, error)` — convenience for filesystem-based inputs
- AST data structures — well-documented, stable shapes used by the compiler

Design notes
------------

- Keep parsing concerns separate: lexical scanning should be in its own package (e.g., `internal/scanner`).
- If you use a parser generator, keep generated code under `grammar/` and check-in only when appropriate (or provide a clear generate step).
- Export only the types and functions necessary for consumers; place helpers under `internal/`.

Developer tasks
---------------

Run unit tests

   cd libs/parser
   go test ./... -v

Run parser integration tests

   cd libs/parser
   go test ./test -v

Working with the compiler
-------------------------

- The `libs/compiler` module should import `github.com/autonomous-bits/nomos/libs/parser` to obtain an AST or token stream. Keep the AST public types small and documented so the compiler can analyze them.
- Avoid two-way imports. If `libs/parser` needs semantic helpers, factor those into `libs/common` or move into `internal/` under the appropriate module.

Best practices
--------------

- Keep parser error messages deterministic and rich (include line/column, source snippet).
- Maintain a `testdata/` directory with representative Nomos source examples and negative test cases.
- If you adopt a generator (e.g., ANTLR, pigeon), include a `make generate` or `go:generate` instructions and document the workflow.

If you need help
---------------

Open a PR describing parser changes, include sample inputs in `testdata/`, and add or update tests demonstrating the expected AST shape or parse errors.
