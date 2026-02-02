# Contributing to Nomos

Thanks for your interest in contributing! This monorepo hosts the Nomos CLI and supporting Go libraries. Please follow these guidelines to keep contributions smooth, consistent, and high-quality.

> Standards first: This project follows the Development Standards Constitution from the general-standards space (quality gates, code review, testing). Conventional Commits with gitmoji are required.

## Prerequisites
- Go 1.25.6+
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

### Makefile Targets Overview

Run `make help` for the full list of available targets. Common commands:

**Building:**
- `make build` – Build all applications
- `make build-cli` – Build the CLI app to `bin/nomos`
- `make build-test` – Build test binaries (required for some integration tests)

**Testing:**
- `make test` / `make test-race` – Run tests across all modules
- `make test-unit` – Run only unit tests (faster, excludes integration tests)
- `make test-integration` – Run only integration tests
- `make test-coverage` – Generate coverage reports (HTML)
- `make test-module MODULE=libs/parser` – Test a single module

**Code Quality:**
- `make fmt` – Format all Go code
- `make mod-tidy` – Tidy all module dependencies
- `make lint` – Run linters (requires golangci-lint)

**Development:**
- `make watch` – Auto-rebuild on file changes (requires air)
- `make work-sync` – Sync Go workspace dependencies
- `make install` – Install nomos binary to GOPATH/bin

## Troubleshooting

### Common Build Issues

**Problem:** `Test binary not found` or test execution failures  
**Solution:** Build test binaries first:
```bash
make build-test
```

**Problem:** `go.work` out of sync or module resolution errors  
**Solution:** Sync workspace and tidy dependencies:
```bash
make work-sync
make mod-tidy
```

**Problem:** Build fails with dependency errors  
**Solution:** Clean and rebuild:
```bash
go clean -cache
make mod-tidy
make build
```

### Test Failures

**Problem:** Integration tests failing with "connection refused" or timeout  
**Solution:** Ensure you're running integration tests explicitly (they're skipped by default):
```bash
make test-integration
```

**Problem:** Race detector reports data races  
**Solution:** Run race detector and fix reported issues:
```bash
make test-race
```

**Problem:** Tests pass locally but fail in CI  
**Solution:** 
- Check if integration test tags are properly set (`//go:build integration`)
- Ensure all dependencies are committed (don't rely on local caches)
- Run the full test suite including race detector: `make test-race`

### Linting Issues

**Problem:** CI linting fails but `make lint` works locally  
**Solution:** Ensure you have the same golangci-lint version as CI:
```bash
# Check CI version in .github/workflows/
golangci-lint --version

# Install specific version if needed
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

**Problem:** Lint reports issues in generated code  
**Solution:** Add `//nolint:all` comment or update `.golangci.yml` to exclude generated files.

**Problem:** Import formatting issues  
**Solution:** Run formatting tools:
```bash
make fmt
# Or use goimports directly
goimports -w .
```

### Module Dependency Issues

**Problem:** `ambiguous import` or version conflicts  
**Solution:** Tidy all module dependencies:
```bash
# For specific module
cd libs/compiler
go mod tidy

# For all modules
make mod-tidy
```

**Problem:** Workspace not recognizing local module changes  
**Solution:** Sync the workspace:
```bash
make work-sync
# Or manually
go work sync
```

**Problem:** Module requires newer Go version  
**Solution:** Update Go installation to 1.25.6+ or check `go.mod` for required version.

### Still Having Issues?

- Check existing [GitHub issues](https://github.com/autonomous-bits/nomos/issues)
- Review [docs/TESTING_GUIDE.md](docs/TESTING_GUIDE.md) for testing-specific problems
- Open a new issue with details about your environment and error messages
- Ask in project discussions for help

## Branching Strategy
- Create feature branches from `main` using a clear prefix:
  - `feature/<short-description>`
  - `fix/<short-description>`
  - `chore/<short-description>`
- Keep branches focused and small; prefer incremental PRs.

## Commit Messages (Required)

We use **Conventional Commits with gitmoji** for clear, consistent commit history.

📖 **See [.github/instructions/commit-messages.instructions.md](.github/instructions/commit-messages.instructions.md) for complete guidelines.**

### Format

```
<type>[optional scope][!]: <gitmoji> <description>

[optional body]

[optional footer(s)]
```

### Examples

```bash
feat(cli): ✨ add build subcommand
fix(parser): 🐛 handle nested references correctly
docs: 📝 update contributing guide
refactor(compiler)!: ♻️ redesign provider resolution API

BREAKING CHANGE: Provider interface now requires context parameter
```

### Common Types and Gitmojis

| Type | Gitmoji | Description |
|------|---------|-------------|
| `feat` | ✨ | New feature |
| `fix` | 🐛 | Bug fix |
| `docs` | 📝 | Documentation only |
| `refactor` | ♻️ | Code refactoring |
| `test` | 🧪 | Adding/updating tests |
| `chore` | 🔧 | Maintenance tasks |
| `perf` | ⚡️ | Performance improvements |
| `build` | 🛠️ | Build system/dependencies |

### Breaking Changes

Use `!` after type/scope for breaking changes:
```bash
feat(api)!: ✨ redesign authentication flow

BREAKING CHANGE: All authentication methods now require API keys
```

### Validation

PR validation automatically enforces conventional commit format. Commits that don't follow the format will be rejected.

## Pull Requests

Every PR must have a clear description and pass all quality gates.

📖 **See [.github/instructions/pull-request-description.instructions.md](.github/instructions/pull-request-description.instructions.md) for the complete PR template.**

### PR Requirements

**Before submitting:**
- ✅ Tests pass (`make test` or targeted `make test-module`)
- ✅ Lint is clean (`make lint`)
- ✅ Code is formatted (`make fmt`)
- ✅ Dependencies are tidy (`make mod-tidy`)
- ✅ **CHANGELOG.md updated** for affected modules (add to `[Unreleased]` section)
- ✅ New/changed behavior is documented (README, docs, or inline)

**PR Description must include:**
- **What**: Clear summary of the changes
- **Why**: Motivation and context for the change
- **Testing**: How it was tested, what tests were added/updated
- **Breaking Changes**: Any backwards-incompatible changes
- **Related Issues**: Use `Closes #123` or `Related to #456`

### PR Size Considerations

Keep PRs focused and manageable:
- **Small PRs** (< 400 lines): Ideal, easier to review, faster to merge
- **Medium PRs** (400-1000 lines): Acceptable, break into logical commits
- **Large PRs** (> 1000 lines): Split into multiple PRs when possible

For large refactoring work, consider:
- Breaking into phases with separate PRs
- Draft PRs for early feedback
- Clear commit structure showing progression

### CHANGELOG.md Updates (Required)

**Every PR with user-visible changes must update the CHANGELOG.md** in affected modules:

1. Add entry to `[Unreleased]` section
2. Use appropriate category: Added, Changed, Fixed, etc.
3. Include PR/issue reference: `(#123)`
4. Follow Keep a Changelog format

Example:
```markdown
## [Unreleased]

### Added
- Support for external Terraform providers (#234)

### Fixed
- Race condition in concurrent import resolution (#245)
```

See module CHANGELOGs and [docs/RELEASE.md](docs/RELEASE.md) for more details.

### Review Process

- Request review from relevant code owners when possible
- Address review feedback promptly
- Keep discussion focused on the changes
- All CI checks must pass before merging

## Testing

Write tests for all features and bug fixes. The project uses a combination of unit tests and integration tests to ensure code quality.

📖 **See [docs/TESTING_GUIDE.md](docs/TESTING_GUIDE.md) for comprehensive testing guidelines, patterns, and best practices.**

### Quick Testing Reference

Run tests across the workspace:
- `make test` – all tests (unit + integration) for apps and libs
- `make test-unit` – unit tests only (faster, excludes integration tests)
- `make test-integration` – integration tests only across all modules
- `make test-race` – race detector across modules
- `make test-module MODULE=libs/compiler` – all tests for a single module
- `make test-integration-module MODULE=libs/compiler` – integration tests for a single module

### Integration Test Tags (Required)

Integration tests **must** use the `//go:build integration` build tag:

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

### CI Workflow Requirements

All CI checks must pass before merging:
- ✅ Unit tests pass
- ✅ Integration tests pass
- ✅ Linting passes (golangci-lint)
- ✅ Code formatting is correct (gofmt)
- ✅ All modules build successfully

Integration tests are run separately in CI to control execution time and resource usage.

## Linting & Formatting
- Go formatting: standard `gofmt` via your editor/tools or `make fmt`.
- Lint: `golangci-lint` (optional but recommended). Run `make lint`.
- EditorConfig: The repo includes `.editorconfig` for consistent formatting across editors.

## Development Tools (Optional)

### Pre-commit Hooks with Lefthook (Recommended)
We provide `.lefthook.yml` for automated pre-commit checks to catch issues early:

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

Using lefthook is optional but highly recommended to catch issues before they reach CI.

### Watch Mode for Development
Auto-rebuild on file changes using Air (optional):

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
