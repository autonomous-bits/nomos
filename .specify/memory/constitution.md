<!--
Sync Impact Report:
Version change: 0.0.0 → 1.0.0
Modified principles: N/A (initial version)
Added sections: All (initial constitution)
Removed sections: None
Templates requiring updates:
  ✅ plan-template.md - Constitution Check section aligns
  ✅ spec-template.md - Requirements structure aligns  
  ✅ tasks-template.md - Test-first approach aligns
Follow-up TODOs: None
-->

# Nomos Constitution

## Core Principles

### I. Library-First Architecture
Every feature MUST start as a standalone Go library module before being exposed to applications.

**Non-negotiable rules:**
- Libraries MUST be self-contained with explicit module boundaries (`go.mod`)
- Libraries MUST be independently testable without external dependencies on sibling modules
- Libraries MUST have clear, documented purpose—no organizational-only libraries
- Internal packages (`internal/`) MUST be used to hide implementation details from consumers
- Public APIs MUST be minimal, stable, and backward-compatible within major versions

**Rationale:** This ensures reusability, testability, and clear separation of concerns across the monorepo. It prevents tight coupling and allows independent evolution of components.

### II. CLI-Driven Interface
Every library MUST expose its functionality through a command-line interface.

**Non-negotiable rules:**
- Text-based I/O protocol: configuration files/stdin/args → stdout, errors → stderr
- Support multiple output formats: JSON, YAML, and HCL where applicable
- CLI commands MUST be composable and scriptable
- Exit codes MUST follow UNIX conventions (0 = success, non-zero = error)
- Help text and documentation MUST be comprehensive and always up-to-date

**Rationale:** CLI-first design ensures automation capability, debuggability, and seamless integration with CI/CD pipelines and other tools.

### III. Test-First Development (NON-NEGOTIABLE)
Test-Driven Development (TDD) is MANDATORY for all feature work.

**Non-negotiable rules:**
- Tests MUST be written before implementation
- Tests MUST fail initially, demonstrating they test the right behavior
- Implementation proceeds only after test approval
- Red-Green-Refactor cycle MUST be strictly enforced
- Coverage MUST be meaningful, not just numerical—focus on behavior verification
- Integration tests MUST validate end-to-end workflows with real fixtures

**Rationale:** TDD ensures code quality, prevents regressions, and serves as living documentation. It forces clear requirements definition before implementation.

### IV. Integration Testing
Integration tests MUST be provided for critical interaction boundaries.

**Focus areas requiring integration tests:**
- New library contract tests (public API behavior)
- Contract changes (backward compatibility verification)
- External provider loading and execution
- Compilation pipeline end-to-end (parsing → compilation → snapshot generation)
- Cross-module interactions within the monorepo

**Rationale:** Unit tests alone cannot catch integration issues. Real-world workflows must be validated against fixtures in `testdata/` directories.

### V. Observability & Debuggability
Code MUST be debuggable through text-based I/O and structured logging.

**Non-negotiable rules:**
- Text I/O ensures debuggability—avoid binary protocols except where necessary
- Structured logging MUST be used for all runtime diagnostics
- Error messages MUST include actionable context (file paths, line numbers, suggestions)
- Provider operations MUST log loading, execution, and failure details
- Compilation errors MUST reference source locations precisely

**Rationale:** Configuration scripting errors are common; users need clear, actionable feedback to resolve issues quickly.

### VI. Versioning & Breaking Changes
Modules MUST follow semantic versioning (MAJOR.MINOR.PATCH).

**Non-negotiable rules:**
- MAJOR version for backward-incompatible API changes
- MINOR version for backward-compatible feature additions
- PATCH version for bug fixes and non-functional improvements
- CHANGELOG.md MUST be maintained per module following Keep a Changelog format
- Breaking changes MUST be documented with migration guides
- Provider contracts MUST maintain backward compatibility within major versions

**Rationale:** Clear versioning enables safe upgrades and prevents breaking user configurations unexpectedly.

### VII. Simplicity & YAGNI
Start simple—implement only what is needed now.

**Non-negotiable rules:**
- Do not add features speculatively (You Aren't Gonna Need It)
- Prefer simple, explicit solutions over clever abstractions
- Complexity MUST be justified with documented rationale (see Complexity Tracking in plan-template.md)
- Dependencies MUST be minimal—prefer stdlib when feasible
- Provider contracts MUST remain minimal and extension-friendly

**Rationale:** Premature complexity leads to maintenance burden and cognitive overhead. Add complexity only when actual needs emerge.

## Technology Standards

### Language & Runtime
- **Primary Language**: Go 1.25+
- **Module Management**: Go modules (`go.mod`/`go.sum`) per library
- **Workspace**: Root `go.work` for local cross-module development
- **Build System**: Makefile for common tasks (`make build`, `make test`, `make lint`)

### Code Quality
- **Linting**: golangci-lint MUST pass without warnings
- **Formatting**: `gofmt` and `goimports` MUST be applied to all code
- **Testing**: `go test ./...` MUST pass for all modules before merge
- **Race Detection**: `make test-race` MUST pass for concurrent code

### External Providers
- **Provider Protocol**: gRPC-based using `provider-proto` definitions
- **Provider Discovery**: External providers loaded from configurable repositories
- **Provider Versioning**: Providers MUST declare semantic versions in manifests
- **Provider Isolation**: Provider execution MUST be sandboxed and fail-safe

## Development Workflow

### Commit Standards
- **Format**: Conventional Commits with gitmoji (see `.github/instructions/commit-messages.instructions.md`)
- **Examples**: `feat(cli): ✨ add build subcommand`, `fix(parser): 🐛 resolve nested map parsing`
- **Enforcement**: Pre-commit hooks recommended, PR reviews MUST verify compliance

### Branching & PRs
- **Branch Naming**: `feature/<short-description>`, `fix/<short-description>`, `chore/<short-description>`
- **PR Size**: Keep PRs small and focused—prefer incremental changes
- **Review Requirements**: At least one approval required; tests MUST pass
- **Merge Strategy**: Squash commits to maintain clean history

### Testing Gates
- All tests MUST pass (`make test`)
- Race detector MUST pass for concurrent code (`make test-race`)
- Linter MUST pass (`make lint`)
- Integration tests MUST pass for affected workflows
- Changelogs MUST be updated per `.github/instructions/changelog.instructions.md`

### Release Process
- Tag releases with semantic version (`v1.2.3`)
- Update CHANGELOG.md following Keep a Changelog format
- Document breaking changes with migration guides
- Coordinate module releases when cross-module changes affect public APIs

## Governance

This constitution supersedes all other development practices. All changes MUST align with these principles.

**Amendment Process:**
- Amendments require documented rationale and team consensus
- Version increment follows semantic versioning rules for constitutions:
  - MAJOR: Backward-incompatible governance changes (principle removal/redefinition)
  - MINOR: New principles or materially expanded guidance
  - PATCH: Clarifications, wording improvements, non-semantic refinements
- Migration plans MUST accompany breaking principle changes
- All dependent templates and documentation MUST be updated to reflect amendments

**Compliance:**
- All PRs and code reviews MUST verify compliance with this constitution
- Constitution Check gates in `plan-template.md` MUST be validated before implementation
- Complexity exceptions MUST be justified in Complexity Tracking section
- Refer to `CONTRIBUTING.md` for runtime development guidance and quick-start instructions

**Version**: 1.0.0 | **Ratified**: 2025-12-25 | **Last Amended**: 2025-12-25
