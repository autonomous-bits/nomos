# AGENTS: apps/command-line

This document explains the local structure and development workflow for the `apps/command-line` module. It is a companion to the repository-level guidance in `docs/architecture/go-monorepo-structure.md` but scoped to this specific application.

Purpose
-------
The `command-line` module contains the Nomos CLI application. It's an executable-focused module that depends on shared libraries in `libs/` (for example `libs/compiler` and `libs/parser`). This file documents the on-disk layout, common developer tasks (build, test, run), and guidance for local changes.

Repository layout (local)
-------------------------

Typical files and directories you will find in this module:
- `go.mod` / `go.sum` — module declaration and dependency pins for this CLI.
- `cmd/nomos/main.go` — the main package(s) that build the CLI executable.
- `internal/` — application-specific packages that are not intended for external import (e.g., CLI wiring, configuration, and helpers).
- `pkg/` (optional) — exported packages intended for reuse by other modules or tests.
- `test/` — integration or end-to-end tests that exercise the CLI binary.
- `README.md`, `CHANGELOG.md` — module-level documentation and history.

Example layout:

```
apps/command-line/
├── go.mod
├── go.sum
├── cmd/
│   └── nomos/
│       └── main.go
├── internal/
│   ├── cli/
│   ├── config/
│   └── runner/
├── pkg/        # optional
├── test/
└── README.md
```

How this module fits in the monorepo
-----------------------------------

- This module is an application that imports libraries from `libs/` (for example `github.com/autonomous-bits/nomos/libs/compiler`).
- During local development use the repository `go.work` to link local modules so you don't need replace directives.

Common developer tasks
----------------------

Build the CLI locally
1. From repository root (recommended): ensure `go.work` includes this module and the libs:

	go work use ./apps/command-line
2. Build the CLI executable:

	cd apps/command-line && go build -o ../../bin/nomos ./cmd/nomos
Run the CLI

	./bin/nomos <args>
Run unit tests for the module

	cd apps/command-line
	go test ./... -v

Run integration/e2e tests
Integration tests often build the binary and invoke it. From repository root:

	cd apps/command-line
	go test ./test -v

Working with shared libraries
- Prefer `go.work` at repository root for local development so imports resolve to local `libs/*` modules.
- If you need to iterate quickly on both CLI and library code, edit the library in `libs/` and run the CLI build/test; the `go.work` setup should pick up local changes.

Best practices and notes
------------------------

- Keep `internal/` packages focused on application concerns (parsing flags, command wiring, I/O). Shared logic should live in `libs/`.
- CLI commands should be thin: parse args, validate inputs, delegate to library APIs.
- For reproducible builds, avoid embedding ephemeral build-time replace directives in `go.mod`; use `go.work` for local workflows.
- Maintain small, focused tests for `internal/` packages and put full end-to-end tests under `test/`.

Versioning and releases
-----------------------

- This app is versioned by tagging at the repository level using the module path (see `docs/architecture/go-monorepo-structure.md`). Typical tag: `apps/command-line/v1.0.0`.
- Release artifacts (binaries) can be produced using a root-level `Makefile` or `goreleaser` configuration.

If you need help
---------------

If you're unsure where to place new code, open a short PR describing the change and label it `needs-architecture-review` so we can keep module boundaries clean.
