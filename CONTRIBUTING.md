# Contributing to Nomos

Thanks for your interest in contributing! This monorepo hosts the Nomos CLI and supporting Go libraries. Please follow these guidelines to keep contributions smooth, consistent, and high-quality.

> Standards first: This project follows the Development Standards Constitution from the general-standards space (quality gates, code review, testing). Conventional Commits with gitmoji are required.

## Prerequisites
- Go 1.22+
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
- `make test-module MODULE=libs/parser` – Test a single module
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
  - `make test` – verbose tests for apps and libs
  - `make test-race` – race detector across modules
  - `make test-module MODULE=libs/compiler` – scope to a single module
- Integration tests live under each module's `test/` or within `apps/command-line/test/`.

## Linting & Formatting
- Go formatting: standard `gofmt` via your editor/tools.
- Lint: `golangci-lint` (optional but recommended). Run `make lint`.

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