# Contributing to Nomos

Thanks for your interest in contributing! This monorepo hosts the Nomos CLI and supporting Go libraries. Please follow these guidelines to keep contributions smooth, consistent, and high-quality.

> Standards first: This project follows the Development Standards Constitution from the general-standards space (quality gates, code review, testing). Conventional Commits with gitmoji are required.

## Prerequisites
- Go 1.25+
- Git
- macOS, Linux, or Windows
- Optional: golangci-lint (for `make lint`)

## Quick Start (Local Dev)
```bash
# 1) Clone
git clone https://github.com/autonomous-bits/nomos.git
cd nomos

# 2) Sync workspace modules
make work-sync

# 3) Build the CLI
make build-cli

# 4) Run tests (all modules)
make test

# 5) Lint (requires golangci-lint installed)
make lint
```

Useful targets (run `make help` for the full list):
- `make build` – Build all applications
- `make build-cli` – Build the CLI app to `bin/nomos`
- `make test` / `make test-race` – Run tests across all modules
- `make test-unit` – Run only unit tests (faster)
- `make test-integration` – Run only integration tests
- `make test-coverage` – Generate coverage reports (HTML)
- `make test-module MODULE=libs/parser` – Test a single module
- `make fmt` – Format all Go code
- `make mod-tidy` – Tidy all module dependencies
- `make install` – Install nomos binary to GOPATH/bin
- `make lint` – Run linters (requires golangci-lint)
- `make watch` – Auto-rebuild on file changes (requires air)
- `make work-sync` – Sync Go workspace dependencies

## Branching Strategy
- Create feature branches from `main` using a clear prefix:
  - `feature/<short-description>`
  - `fix/<short-description>`
  - `chore/<short-description>`
- Keep branches focused and small; prefer incremental PRs.

## Commit Messages (Required)
We use Conventional Commits with gitmoji:
- Format: `<type>(optional scope)!: :gitmoji: short description`
- Examples:
  - `feat(cli): ✨ add build subcommand`
  - `fix(parser): 🐛 handle nested references correctly`
  - `docs: 📝 add contributing guide`
- Breaking changes: add `!` after type/scope and a `BREAKING CHANGE:` footer when needed.

See `.github/instructions/commit-messages.instructions.md` for the full rules and emoji mapping.

## Pull Requests
Every PR must have a clear description and pass quality gates.
- Use the template in `.github/instructions/pull-request-description.instructions.md`.
- Include what changed, why, testing details, and any breaking changes.
- Ensure:
  - Tests pass (`make test` or targeted `make test-module`)
  - Lint is clean (`make lint`)
  - New/changed behavior is documented (README, docs, or inline)
- Request review from relevant code owners when possible.

## Testing
- Write tests for all features and bug fixes.
- Run tests across the workspace:
  - `make test` – all tests (unit + integration) for apps and libs
  - `make test-unit` – unit tests only (faster, excludes integration tests)
  - `make test-integration` – integration tests only across all modules
  - `make test-race` – race detector across modules
  - `make test-module MODULE=libs/compiler` – all tests for a single module
  - `make test-integration-module MODULE=libs/compiler` – integration tests for a single module

### Integration Test Conventions
Integration tests require the `//go:build integration` build tag:

```go
//go:build integration
// +build integration

package mypackage

import "testing"

func TestIntegration_SomeFeature(t *testing.T) {
    // Test code that requires:
    // - External services (network calls)
    // - File system operations
    // - End-to-end workflows
    // - Longer execution time
}
```

**When to use integration tests:**
- End-to-end compilation workflows
- Real file system operations (not using temp dirs)
- Network/HTTP requests to external services
- Provider binary execution
- Multi-component interactions

**When to use unit tests:**
- Pure functions and isolated logic
- Mocked dependencies
- Fast, deterministic tests
- Core algorithms and data structures

Integration tests are:
- Located in module root (`*_integration_test.go`) or `test/` directories
- Excluded from default `go test ./...` runs (use `-tags=integration` to include)
- Run separately in CI to control execution time
- Allowed longer timeouts and may have external dependencies

## Linting & Formatting
- Go formatting: standard `gofmt` via your editor/tools or `make fmt`.
- Lint: `golangci-lint` (optional but recommended). Run `make lint`.
- EditorConfig: The repo includes `.editorconfig` for consistent formatting across editors.

## Development Tools (Optional)

### Pre-commit Hooks with Lefthook
We provide `.lefthook.yml` for automated pre-commit checks:

```bash
# Install lefthook
go install github.com/evilmartians/lefthook@latest

# Setup hooks
lefthook install
```

This will automatically run on every commit:
- `make fmt` – Format code
- `make mod-tidy` – Tidy dependencies
- `make lint` – Run linters
- Conventional commit message validation

### Watch Mode for Development
Auto-rebuild on file changes using Air:

```bash
# Install air
go install github.com/cosmtrek/air@latest

# Start watch mode
make watch
```

The CLI will automatically rebuild when you modify source files in `apps/` or `libs/`.

## Project Structure
- `apps/command-line` – Nomos CLI
- `libs/compiler` – compiler library
- `libs/parser` – parser library
- `libs/provider-proto` – provider protobuf contracts
- `docs/` – architecture, guides, and examples

The repo uses a Go workspace (`go.work`) to wire modules together for local development.

## Releases
- Libraries are tagged per-module (see `make release-check` and `make release-lib`).
- Keep module CHANGELOGs up to date following Keep a Changelog + SemVer.
- Coordinate cross-module changes atomically within the monorepo when needed.

## Security
- Do not commit secrets.
- Keep dependencies up to date.
- Report vulnerabilities privately via repository security channels.

## Questions
Open a discussion or file an issue if you need help getting started or want feedback on an approach before you implement it.

---

By submitting a pull request, you confirm your contribution complies with the project standards and that you’re okay with your changes being licensed under the repository’s license.
