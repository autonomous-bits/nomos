# AGENTS: libs/compiler

This document describes the `libs/compiler` module layout and development guidance. It focuses on the local structure for the compiler library (the public surface exposed to apps such as the `command-line` CLI) and the internal packages used during compilation.

Purpose
-------

The `libs/compiler` module provides compilation functionality for Nomos scripts. It exposes a small, stable public API that higher-level applications (for example `apps/command-line`) call to compile inputs into configuration snapshots.

Repository layout (local)
-------------------------

Typical files and directories:

- `go.mod` / `go.sum` — module declaration and dependency pins for the compiler library.
- `compiler.go` (or a `pkg/compiler/` package) — the public API entry points (e.g., `Compile(path string) (Result, error)`).
- `internal/` — compiler internals such as lexer, parser adapters, IR, analyzer, and codegen. These packages are private to the module.
- `pkg/` — optional packages that provide public sub-packages used by other modules.
- `test/` — integration tests validating the compiler end-to-end (may use fixtures under `testdata/`).
- `examples/` — small examples showing how to invoke the compiler library programmatically.

Example layout:

```
libs/compiler/
├── go.mod
├── go.sum
├── compiler.go        # public API (or pkg/compiler/)
├── pkg/               # optional public packages
├── internal/
│   ├── lexer/
│   ├── parser/        # may call into libs/parser
│   ├── ir/
│   ├── analyzer/
│   └── codegen/
├── test/
│   └── integration_test.go
└── examples/
```

Public API contract (suggested)
-------------------------------

Keep the public API small and stable. A minimal contract might look like:

- `Compile(inputPath string, opts ...CompileOption) (Snapshot, error)`
- `Snapshot` — a serializable artifact containing compiled configuration and metadata
- Errors — use wrapped errors with context for easier debugging

Design notes
------------

- Put implementation details into `internal/` packages to avoid accidental external dependencies.
- If the parser logic is shared, the compiler should depend on `github.com/autonomous-bits/nomos/libs/parser` as a module import.
- Avoid circular dependencies: the compiler should import the parser, not vice-versa.

Developer tasks
---------------

Run unit tests

	cd libs/compiler
	go test ./... -v

Run integration tests

Integration tests may exercise the library against sample source files in `testdata/`:

	cd libs/compiler
	go test ./test -v

Build and run examples

Examples can be built or run with `go run`:

	go run ./examples/simple

Consuming this library from the CLI
----------------------------------

- Import via module path: `github.com/autonomous-bits/nomos/libs/compiler`.
- During local development, make sure the root `go.work` includes both `apps/command-line` and `libs/compiler` so the CLI resolves to the local module.

Best practices
--------------

- Keep public packages under `pkg/` or at the module root and internal helpers under `internal/`.
- Write deterministic tests that use fixtures under `testdata/` and avoid relying on external network calls.
- Provide small example programs showing the typical usage pattern for callers (the CLI or other tools).
- Version the module independently and follow semantic versioning for any breaking changes.

If you need help
---------------

Open a PR describing the change and reference the module-level design if you think a public API change is needed.

