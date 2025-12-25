# Monorepo Governance Agent

## Purpose
Handles cross-cutting concerns across the Nomos monorepo including workspace management, versioning, changelog coordination, and commit message enforcement. This agent ensures consistency, maintainability, and operational excellence across all projects in the monorepo.

## Standards Source
- https://github.com/autonomous-bits/development-standards/blob/main/project-structure.md
- https://github.com/autonomous-bits/development-standards/blob/main/changelog.md
- https://github.com/autonomous-bits/development-standards/blob/main/commit-messages.md
- https://github.com/autonomous-bits/development-standards/blob/main/go_practices/modules_and_vendoring.md
- https://github.com/autonomous-bits/development-standards/blob/main/go_practices/standard_project_layout.md

Last synced: 2025-12-25

## Coverage Areas

1. **Go Workspace Management** - Managing the `go.work` file and module coordination
2. **Independent Module Versioning** - Per-module semantic versioning strategy
3. **CHANGELOG Coordination** - Keep a Changelog format with monorepo-specific guidance
4. **Conventional Commits with Gitmoji** - Enforcing commit message standards
5. **Module Interdependencies** - Managing dependencies between workspace modules
6. **Monorepo Structure** - Maintaining consistent directory organization

---

## Go Workspace Management

### Workspace File Structure
The Nomos monorepo uses Go workspaces (`go.work`) to coordinate multiple modules:

```go
go 1.25.3

use (
    ./apps/command-line
    ./examples/consumer
    ./libs/compiler
    ./libs/parser
    ./libs/provider-downloader
    ./libs/provider-proto
)
```

### Adding New Modules

When adding a new module to the workspace:

1. **Create module directory** with appropriate structure:
   ```
   libs/new-module/
   ├── go.mod              # Module definition
   ├── README.md           # Module documentation
   ├── CHANGELOG.md        # Module changelog
   ├── *.go                # Go source files
   └── *_test.go           # Test files
   ```

2. **Initialize module**:
   ```bash
   cd libs/new-module
   go mod init github.com/nomos-project/nomos/libs/new-module
   ```

3. **Add to workspace**:
   ```bash
   # From repository root
   go work use ./libs/new-module
   ```

4. **Verify workspace**:
   ```bash
   go work sync
   go mod tidy
   ```

### Workspace Best Practices

- **Always commit** `go.work` and `go.work.sum` files
- **Run `go work sync`** after modifying module dependencies
- **Keep modules independent** - minimize cross-module dependencies where possible
- **Use replace directives** in `go.mod` for local development when needed, but document removal strategy
- **Vendor dependencies** for production builds: `go mod vendor` per module

### Module Path Conventions

All modules follow the pattern:
- Apps: `github.com/nomos-project/nomos/apps/<name>`
- Libraries: `github.com/nomos-project/nomos/libs/<name>`
- Examples: `github.com/nomos-project/nomos/examples/<name>`

---

## Independent Module Versioning

### Semantic Versioning Strategy

Each module in the Nomos monorepo follows **independent semantic versioning**:

```
MAJOR.MINOR.PATCH

MAJOR: Breaking changes (v1.0.0 → v2.0.0)
MINOR: New features, backwards compatible (v1.0.0 → v1.1.0)
PATCH: Bug fixes, backwards compatible (v1.0.0 → v1.0.1)
```

### Version Bump Rules

**MAJOR version** when:
- Breaking API changes in exported functions/types
- Removal of public APIs
- Incompatible changes to configuration formats
- Changes requiring consumer code updates

**MINOR version** when:
- New features added (backwards compatible)
- New public APIs introduced
- New CLI commands or flags added
- Deprecations announced (not yet removed)

**PATCH version** when:
- Bug fixes
- Performance improvements (no API changes)
- Documentation updates
- Internal refactoring (no external impact)
- Security patches

### Tagging Strategy

Each module is tagged independently:

```bash
# For apps/command-line
git tag apps/command-line/v1.2.3

# For libs/compiler
git tag libs/compiler/v2.0.0

# For libs/parser
git tag libs/parser/v1.5.1
```

### Version Coordination

When multiple modules need coordinated releases:

1. **Update all affected modules** atomically in a single PR
2. **Document cross-module impacts** in commit message
3. **Tag modules in dependency order** (dependencies first, consumers second)
4. **Update inter-module dependencies** to reference new versions

Example coordinated change:
```bash
# If compiler depends on parser, tag parser first
git tag libs/parser/v1.6.0
git push origin libs/parser/v1.6.0

# Then update compiler's go.mod and tag
git tag libs/compiler/v2.1.0
git push origin libs/compiler/v2.1.0
```

---

## CHANGELOG Coordination

### Per-Module CHANGELOGs

Each module maintains its own `CHANGELOG.md` following Keep a Changelog format:

```markdown
# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added
- New provider resolution cache (#234)

### Changed
- Improved error messages for type mismatches

### Fixed
- Race condition in concurrent import resolution (#245)

## [2.1.0] - 2025-12-20

### Added
- Support for external Terraform providers (#223)
- Provider type registry with remote resolution

### Changed
- **BREAKING**: Renamed `ResolveImport` to `ResolveReference` (#220)

[Unreleased]: https://github.com/nomos-project/nomos/compare/libs/compiler/v2.1.0...HEAD
[2.1.0]: https://github.com/nomos-project/nomos/compare/libs/compiler/v2.0.0...libs/compiler/v2.1.0
```

### Category Guidelines

Use these categories in order (omit empty categories):

1. **Added** - New features, APIs, capabilities
2. **Changed** - Changes in existing functionality (mark BREAKING changes)
3. **Deprecated** - Features marked for removal (with timeline)
4. **Removed** - Deleted features (always BREAKING)
5. **Fixed** - Bug fixes
6. **Security** - Security vulnerability patches (include CVE numbers)
7. **Performance** - Performance improvements

### Entry Style

- **Use imperative mood**: "Add feature" not "Added feature"
- **Be specific**: Include what changed and why when relevant
- **Reference issues/PRs**: `(#123)` or `closes #456`
- **Scope changes**: Use brackets for clarity: `[Parser] Add support for...`
- **Mark breaking changes**: Prefix with `**BREAKING**:` in Changed or Removed sections

### Maintaining Unreleased Section

For every merged PR with user-visible changes:

1. **Update Unreleased section** in affected module's CHANGELOG
2. **Add bullet** under appropriate category
3. **Keep newest changes first** within each category
4. **Don't include** internal-only changes (CI, tooling, refactors with no external impact)

### Release Process

When preparing a release:

1. **Determine version bump** based on changes in Unreleased section
2. **Create new version section** with date: `## [X.Y.Z] - YYYY-MM-DD`
3. **Move Unreleased items** to new section
4. **Update compare links** at bottom of file
5. **Keep Unreleased section** empty but present
6. **Commit and tag** together

---

## Commit Message Format

### Conventional Commits with Gitmoji

All commits follow Conventional Commits 1.0.0 with gitmoji prefixes:

```
<type>[optional scope][!]: <gitmoji> <description>

[optional body]

[optional footer(s)]
```

### Type and Gitmoji Mapping

| Type | Gitmoji | Description |
|------|---------|-------------|
| **feat** | ✨ | New feature for users |
| **fix** | 🐛 | Bug fix for users |
| **docs** | 📝 | Documentation only |
| **style** | 🎨 | Code style (formatting, etc.) |
| **refactor** | ♻️ | Code change (no bug fix or feature) |
| **perf** | ⚡️ | Performance improvement |
| **test** | 🧪 | Adding or updating tests |
| **build** | 🛠️ | Build system or dependencies |
| **chore** | 🔧 | Auxiliary tools, maintenance |
| **ci** | 👷 | CI/CD configuration |
| **security** | 🔒 | Security vulnerability fix |
| **wip** | 🚧 | Work in progress |

### Scope Guidelines

Use scopes to identify affected module or area:

- **Module scopes**: `parser`, `compiler`, `cli`, `provider-downloader`, `provider-proto`
- **Cross-cutting scopes**: `workspace`, `deps`, `docs`, `ci`
- **Feature scopes**: `auth`, `import`, `validation`, `providers`

Examples:
```bash
# Single module change
feat(parser): ✨ add support for nested map literals

# Cross-module change
fix(compiler,parser): 🐛 resolve type inference edge case

# Breaking change
feat(compiler)!: ✨ redesign provider resolution API

BREAKING CHANGE: ProviderResolver interface now requires context parameter
```

### Subject Line Rules

- **Use imperative mood**: "add feature" not "added feature"
- **No capitalization**: lowercase first letter (after gitmoji)
- **No period at end**
- **Maximum 72 characters** (including type, scope, gitmoji)
- **Be specific**: Clear description of what changed

### Body Guidelines

- **Separate from subject** with blank line
- **Wrap at 72 characters**
- **Explain what and why**, not how
- **Use bullet points** for multiple items
- **Reference issues**: `Fixes #123`, `Closes #456`, `Refs #789`

### Footer Guidelines

- **Breaking changes**: `BREAKING CHANGE: description`
- **Issue references**: `Fixes #123`, `Closes #456`
- **Co-authors**: `Co-authored-by: Name <email>`

### Examples

**Simple feature:**
```
feat(parser): ✨ add lexer support for multiline strings

Implements tokenization for strings that span multiple lines using
triple quotes. This enables more readable configuration files.

Closes #234
```

**Bug fix:**
```
fix(compiler): 🐛 prevent panic on nil provider reference

Add nil check before dereferencing provider in merge logic. This
fixes a crash when compiling configurations with missing imports.

Fixes #567
```

**Breaking change:**
```
feat(compiler)!: ✨ redesign import resolution API

BREAKING CHANGE: ImportResolver.Resolve now requires context.Context
as first parameter. This enables cancellation and timeout support.

Migration guide: Add ctx parameter to all Resolve calls:
  resolver.Resolve(ctx, path)

Closes #890
```

**Multi-module change:**
```
refactor(compiler,parser): ♻️ unify error handling

Extract common error types to new internal/errors package. Both
compiler and parser now use consistent error wrapping and context.

- Removes duplicate error definitions
- Adds structured error metadata
- Improves error messages for end users
```

---

## Module Interdependencies

### Dependency Architecture

```
apps/command-line
  └── depends on libs/compiler

libs/compiler
  ├── depends on libs/parser
  ├── depends on libs/provider-downloader
  └── depends on libs/provider-proto

libs/provider-downloader
  └── depends on libs/provider-proto

libs/parser
  └── (no dependencies - foundation layer)

libs/provider-proto
  └── (no dependencies - protocol definitions)

examples/consumer
  └── depends on libs/compiler (for testing)
```

### Dependency Rules

1. **No circular dependencies** between modules
2. **Apps depend on libs**, never the reverse
3. **Examples depend on everything**, but nothing depends on examples
4. **Foundation libraries** (parser, provider-proto) have no dependencies
5. **Use semantic versioning** for inter-module dependencies

### Managing Internal Dependencies

In `go.mod`, reference other workspace modules by import path:

```go
module github.com/nomos-project/nomos/libs/compiler

require (
    github.com/nomos-project/nomos/libs/parser v1.5.0
    github.com/nomos-project/nomos/libs/provider-downloader v1.2.1
    github.com/nomos-project/nomos/libs/provider-proto v1.0.0
)
```

For local development, workspace automatically resolves these. For external consumers:

```bash
# External projects can import like:
go get github.com/nomos-project/nomos/libs/compiler@v2.1.0
```

### Updating Dependencies

When updating an internal dependency:

1. **Update the dependency** (e.g., libs/parser)
2. **Tag new version** of dependency
3. **Update go.mod** in dependent modules (e.g., libs/compiler)
4. **Run `go mod tidy`** in dependent modules
5. **Test dependent modules** thoroughly
6. **Tag new versions** of dependent modules
7. **Update CHANGELOG** in all affected modules

### Breaking Changes Across Modules

When making breaking changes to a library that other modules depend on:

1. **Document the breaking change** in library's CHANGELOG
2. **Update all dependent modules** in same PR
3. **Coordinate version bumps**: 
   - Library: MAJOR version bump
   - Dependents: MAJOR bump if API affected, MINOR otherwise
4. **Provide migration guide** in commit message or docs
5. **Test entire workspace**: `go test ./...` from root

---

## Nomos Workspace Structure

### Directory Organization

```
nomos/
├── .github/                      # GitHub configuration
│   ├── agents/                   # AI agent instructions
│   ├── instructions/             # Copilot instructions
│   └── workflows/                # CI/CD workflows
├── apps/                         # End-user applications
│   └── command-line/             # Nomos CLI application
│       ├── cmd/                  # CLI commands
│       ├── internal/             # Private CLI logic
│       ├── test/                 # CLI tests
│       ├── testdata/             # Test fixtures
│       ├── go.mod
│       ├── CHANGELOG.md
│       └── README.md
├── libs/                         # Shared libraries
│   ├── compiler/                 # Nomos compiler library
│   │   ├── internal/             # Private compiler logic
│   │   ├── test/                 # Compiler tests
│   │   ├── testdata/             # Test fixtures
│   │   ├── go.mod
│   │   ├── CHANGELOG.md
│   │   └── README.md
│   ├── parser/                   # Nomos parser library
│   │   ├── internal/             # Private parser logic
│   │   ├── pkg/                  # Public parser APIs
│   │   ├── test/                 # Parser tests
│   │   ├── testdata/             # Test fixtures
│   │   ├── go.mod
│   │   ├── CHANGELOG.md
│   │   └── README.md
│   ├── provider-downloader/      # Provider download logic
│   │   ├── internal/             # Private download logic
│   │   ├── testutil/             # Test utilities
│   │   ├── go.mod
│   │   ├── CHANGELOG.md
│   │   └── README.md
│   └── provider-proto/           # Provider protocol definitions
│       ├── gen/                  # Generated protobuf code
│       ├── proto/                # Protobuf definitions
│       ├── go.mod
│       ├── CHANGELOG.md
│       └── README.md
├── examples/                     # Example projects
│   ├── config/                   # Example configurations
│   └── consumer/                 # Example consumer
│       ├── cmd/                  # Consumer commands
│       ├── testdata/             # Test data
│       ├── go.mod
│       └── README.md
├── bin/                          # Compiled binaries
├── docs/                         # Documentation
│   ├── architecture/             # Architecture docs
│   ├── examples/                 # Usage examples
│   └── guides/                   # How-to guides
├── tools/                        # Development tools
│   └── scripts/                  # Build/utility scripts
├── go.work                       # Go workspace definition
├── go.work.sum                   # Workspace checksums
├── Makefile                      # Build automation
└── README.md                     # Repository overview
```

### Directory Guidelines

**`/apps`**: End-user applications
- Each app has own `go.mod` with independent versioning
- Must include `CHANGELOG.md` and `README.md`
- Use `/cmd` for multiple binaries, `/internal` for private code
- Include comprehensive tests in `/test` directories

**`/libs`**: Shared libraries
- Each library has own `go.mod` with semantic versioning
- Must include `CHANGELOG.md`, `README.md`, and comprehensive tests
- Public APIs in root or `/pkg`, private logic in `/internal`
- Use `/testutil` for test helpers shared with consumers

**`/examples`**: Usage examples and consumer tests
- Demonstrates how to use libs and apps
- Not for production use
- Can depend on all other modules

**`/docs`**: Repository documentation
- Architecture decisions, guides, examples
- Separate from per-module READMEs
- Includes migration guides and design docs

**`/tools`**: Development and build tools
- Scripts for automation, code generation, etc.
- Not part of main workspace (unless needed)

### File Naming Conventions

- Go files: `lowercase_with_underscores.go`
- Test files: `*_test.go` (suffix required)
- Generated files: include `// Code generated` comment
- Documentation: `README.md`, `CHANGELOG.md` (uppercase)

---

## Common Tasks

### 1. Adding a New Module to Workspace

**Steps:**

```bash
# 1. Create module directory
mkdir -p libs/new-module

# 2. Initialize module
cd libs/new-module
go mod init github.com/nomos-project/nomos/libs/new-module

# 3. Add to workspace
cd ../..
go work use ./libs/new-module

# 4. Create initial files
touch libs/new-module/README.md
touch libs/new-module/CHANGELOG.md
cat > libs/new-module/CHANGELOG.md << 'EOF'
# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

- Initial release

[Unreleased]: https://github.com/nomos-project/nomos/commits/HEAD/libs/new-module
EOF

# 5. Sync workspace
go work sync

# 6. Commit changes
git add .
git commit -m "chore(workspace): 🔧 add new-module to workspace

Initialize new-module library with basic structure. Adds to go.work
for local development.

Refs #XXX"
```

### 2. Coordinating Multi-Module Changes

**When changes affect multiple modules:**

```bash
# 1. Make changes to all affected modules in feature branch
git checkout -b feature/cross-module-improvement

# 2. Update go.mod in dependent modules if needed
cd libs/dependent-module
go get github.com/nomos-project/nomos/libs/updated-module@main
go mod tidy

# 3. Update CHANGELOG.md in ALL affected modules
# (Add entries to [Unreleased] sections)

# 4. Test entire workspace
cd ../..
go test ./...

# 5. Commit with descriptive message
git add .
git commit -m "feat(compiler,parser): ✨ add metadata tracking

Add metadata fields to AST nodes and compilation results. This
enables better error messages and debugging capabilities.

Changes:
- Parser exports metadata fields in Node interface
- Compiler propagates metadata through compilation pipeline
- CLI displays enhanced error messages with source locations

Closes #456"

# 6. After merge, tag modules in dependency order
git tag libs/parser/v1.6.0
git push origin libs/parser/v1.6.0
git tag libs/compiler/v2.2.0
git push origin libs/compiler/v2.2.0
git tag apps/command-line/v1.3.0
git push origin apps/command-line/v1.3.0
```

### 3. CHANGELOG Updates Across Modules

**For each affected module:**

```markdown
## [Unreleased]

### Added
- Metadata tracking for AST nodes (#456)
```

**After release:**

```markdown
## [Unreleased]

## [X.Y.Z] - 2025-12-25

### Added
- Metadata tracking for AST nodes (#456)

[Unreleased]: https://github.com/nomos-project/nomos/compare/libs/MODULE/vX.Y.Z...HEAD
[X.Y.Z]: https://github.com/nomos-project/nomos/compare/libs/MODULE/vA.B.C...libs/MODULE/vX.Y.Z
```

### 4. Version Bumping and Tagging

**Determine version bump:**

```bash
# Check unreleased changes in CHANGELOG.md
# - Breaking changes → MAJOR
# - New features → MINOR
# - Bug fixes → PATCH

# Example: libs/compiler has breaking change
CURRENT_VERSION="2.1.3"
NEW_VERSION="3.0.0"  # MAJOR bump

# Update CHANGELOG.md (move Unreleased to new version section)

# Commit CHANGELOG update
git add libs/compiler/CHANGELOG.md
git commit -m "docs(compiler): 📝 prepare v3.0.0 release"

# Tag the release
git tag libs/compiler/v3.0.0 -a -m "Release compiler v3.0.0

Breaking changes:
- Redesigned provider resolution API
- Removed deprecated ImportResolver methods

See CHANGELOG.md for full details"

# Push tag
git push origin libs/compiler/v3.0.0
```

### 5. Managing Module Dependencies

**Updating a dependency to latest version:**

```bash
# Update specific dependency
cd libs/compiler
go get github.com/nomos-project/nomos/libs/parser@v1.6.0
go mod tidy

# Or update to latest in workspace
go get github.com/nomos-project/nomos/libs/parser@latest
go mod tidy

# Verify changes
go list -m github.com/nomos-project/nomos/libs/parser

# Test
go test ./...

# Commit
git add go.mod go.sum
git commit -m "build(compiler): 🛠️ update parser to v1.6.0"
```

**Checking dependency graph:**

```bash
# From repository root
go mod graph | grep nomos-project

# Or visualize module dependencies
go mod why github.com/nomos-project/nomos/libs/parser
```

---

## Nomos-Specific Patterns

### Configuration Files Organization

Configuration example files are in `examples/config/`:

```
examples/config/
├── config.csl                    # Main config example
├── test-deeply-nested.csl        # Complex nesting
├── test-provider.csl             # Provider usage
├── test-scalars.csl              # Scalar types
└── data/                         # Test data
```

### Build and Binary Management

Compiled binaries go to `/bin`:

```bash
# Build CLI
cd apps/command-line
go build -o ../../bin/nomos ./cmd/nomos

# Build consumer example
cd examples/consumer
go build -o ../../bin/consumer-example ./cmd/consumer-example
```

### Test Data Conventions

Each module includes `testdata/` directories:

```
libs/compiler/testdata/
├── valid/                        # Valid test inputs
├── invalid/                      # Invalid inputs (error cases)
└── fixtures/                     # Reusable test fixtures
```

### Documentation Structure

```
docs/
├── architecture/                 # ADRs and design docs
│   ├── go-monorepo-structure.md
│   └── nomos-external-providers-feature-breakdown.md
├── examples/                     # Usage examples
│   └── README.md
└── guides/                       # How-to guides
    ├── provider-authoring-guide.md
    ├── external-providers-migration.md
    └── removing-replace-directives.md
```

### Agent Instructions

Agent-specific guidance in `.github/agents/`:

```
.github/
├── agents/                       # Agent instructions
│   └── monorepo-governance.md    # This document
└── instructions/                 # Copilot instructions
    ├── changelog.instructions.md
    └── commit-messages.instructions.md
```

### Makefile Targets

Common `Makefile` targets (if present):

```makefile
.PHONY: build test lint clean

build:              # Build all binaries
test:               # Run all tests
lint:               # Run linters
clean:              # Remove build artifacts
mod-tidy:           # Tidy all modules
mod-update:         # Update dependencies
```

### CI/CD Conventions

Workflows typically include:

- **Test on PR**: Run tests for changed modules
- **Lint on PR**: Check code style and formatting
- **Build on merge**: Build binaries for all platforms
- **Release on tag**: Create GitHub release with binaries

---

## Quick Reference Checklist

### Before Committing

- [ ] All tests pass: `go test ./...`
- [ ] Code formatted: `gofmt -w .` or `goimports -w .`
- [ ] `go.mod` updated: `go mod tidy` in each changed module
- [ ] `go.work` synced: `go work sync` if workspace structure changed
- [ ] CHANGELOG updated: Added entry to [Unreleased] in affected modules
- [ ] Commit message follows conventions: `<type>[scope]: <gitmoji> <description>`
- [ ] References issues: Include `Fixes #123` or `Closes #456` when applicable

### Before Releasing a Module

- [ ] All tests pass for module and dependents
- [ ] CHANGELOG complete: All unreleased changes documented
- [ ] Version determined: MAJOR/MINOR/PATCH based on changes
- [ ] CHANGELOG updated: Moved [Unreleased] to [X.Y.Z] with date
- [ ] Compare links updated: Fixed GitHub compare URLs
- [ ] Dependent modules updated: If breaking changes, update go.mod
- [ ] Tag created: `git tag libs/MODULE/vX.Y.Z`
- [ ] Tag pushed: `git push origin libs/MODULE/vX.Y.Z`

### Before Major Cross-Module Change

- [ ] Impact assessed: Identified all affected modules
- [ ] Migration plan: Created guide for breaking changes
- [ ] Tests written: Coverage for all affected modules
- [ ] Documentation updated: READMEs, guides, examples
- [ ] Dependencies coordinated: Ordered release plan for dependent modules
- [ ] Team notified: Communication about breaking changes

---

## Emergency Procedures

### Reverting a Bad Release

```bash
# 1. Delete the bad tag locally and remotely
git tag -d libs/MODULE/vX.Y.Z
git push origin :refs/tags/libs/MODULE/vX.Y.Z

# 2. Revert problematic commits
git revert <commit-hash>

# 3. Fix issues and test
go test ./...

# 4. Re-release with patch version
git tag libs/MODULE/vX.Y.Z+1
git push origin libs/MODULE/vX.Y.Z+1
```

### Fixing Workspace Corruption

```bash
# 1. Remove workspace files
rm go.work go.work.sum

# 2. Recreate workspace
go work init
go work use ./apps/command-line
go work use ./libs/compiler
go work use ./libs/parser
go work use ./libs/provider-downloader
go work use ./libs/provider-proto
go work use ./examples/consumer

# 3. Sync and verify
go work sync
go mod verify
```

### Resolving Circular Dependencies

If circular dependency detected:

1. **Identify the cycle**: Use `go mod graph` to trace
2. **Extract common code**: Move shared types to new module
3. **Refactor imports**: Point both modules to new common module
4. **Update workspace**: Add new module to `go.work`
5. **Test thoroughly**: Verify no circular imports remain

---

## Governance Notes

- This document is the **source of truth** for monorepo operations
- When standards conflict, this agent takes precedence for Nomos-specific decisions
- Review and update quarterly or when major workspace changes occur
- Propose changes via PR with rationale and team consensus

---

**Version**: 1.0.0  
**Last Updated**: 2025-12-25  
**Owner**: Nomos Maintainers
