# Changelog

All notable changes to the Nomos monorepo will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed
- [Compiler] Preserve list expressions during AST conversion for configuration data
## Nomos Refactoring Initiative (Phases 1-6) - 2025-12-26

A comprehensive 12-week refactoring effort across all Nomos modules, improving code quality, test coverage, architecture, and user experience.

### Overview

This release represents the completion of Phases 1-5 of the Nomos Refactoring Implementation Plan, with significant improvements across all modules:

- **apps/command-line**: v1.0.0 - Complete Cobra migration with modern CLI features
- **libs/parser**: v0.8.0 - 44% → 86.9% test coverage, performance optimizations
- **libs/compiler**: v0.7.0 - Clean architecture, multi-error collection, 75%+ import resolution coverage
- **libs/provider-downloader**: v0.1.0 - First stable release with caching, archive extraction, 81.8% coverage
- **libs/provider-proto**: v0.2.0 - Real gRPC integration tests, comprehensive error documentation

### Phase 1: Critical Fixes & Foundation (Week 1-2) ✅

**Security & Bug Fixes:**
- **[CRITICAL]** Mandatory SHA256 checksum validation for provider binaries before execution
- Fixed context propagation bugs in compiler (replaced hardcoded `context.Background()`)
- Fixed import resolution test coverage (21.3% → 75%+)
- Removed 20+ debug `fmt.Printf()` statements from provider-downloader

**Code Quality:**
- Removed debug test files from parser (`debug_test.go`, `scanner_debug_test.go`)
- Fixed parser benchmark suite (deprecated syntax → inline references)
- Added comprehensive error handling tests (0% → 80% coverage for parser error formatting)
- Rewrote provider-proto contract tests with real gRPC integration

**Infrastructure:**
- Added provider-downloader to Makefile test targets
- Standardized Go version to 1.25.3 across all CI workflows
- Created `.golangci.yml` linting config with pinned golangci-lint version

### Phase 2: CLI Modernization (Week 3-4) ✅

**Released as apps/command-line v1.0.0**

**BREAKING CHANGES:**
- Migrated to Cobra framework (exit codes, help text format changed)
- `nomos init` now returns structured results (breaking for programmatic usage)
- Removed `--from` flag (providers now installed from GitHub Releases)
- In-process providers removed (external providers required)

**New Features:**
- Shell completion support (Bash, Zsh, Fish, PowerShell) via `nomos completion`
- `nomos validate` command for syntax-only validation
- `nomos providers list` command to display installed providers
- `nomos version` command with build metadata
- `--color` flag (auto/always/never) for colored output
- `--quiet` flag to suppress non-error output
- Table output with progress indicators for `nomos init`
- Enhanced error messages with colored diagnostics

### Phase 3: Testing & Quality (Week 5-6) ✅

**Test Coverage Improvements:**
- Parser: 44.2% → 86.9% (exceeded 75% goal by 11.9%)
- Compiler: Consolidated 6 mock implementations → 1 shared `FakeProviderServer`
- Provider-downloader: Implemented caching, archive extraction refactored
- E2E smoke tests: 4/4 passing with real provider integration

**Infrastructure:**
- Created provider-downloader CI workflow
- Standardized integration test layout with `//go:build integration` tags
- Added `test-unit`, `test-integration`, `test-network` Makefile targets
- Created PR validation workflow for commit message format
- All 500+ unit tests passing across entire monorepo

### Phase 4: Compiler Refactoring (Week 7-8) ✅

**Released as libs/compiler v0.7.0**

**Architecture Improvements:**
- Created `internal/core` package with unified provider interfaces (~100 LOC removed)
- Created `internal/pipeline` package organizing compilation into 3 stages
- Reduced `compiler.go` from 510 → 341 lines (-33% reduction)
- Eliminated adapter pattern duplication

**Error Handling:**
- Implemented `CompilationResult` with multi-error collection
- Compilation now continues through recoverable errors (better developer experience)
- Explicit `ErrImportResolutionNotAvailable` replaces nil sentinel

**Provider Lifecycle:**
- Configurable graceful shutdown with 5-second default timeout
- Improved error path cleanup to prevent resource leaks
- Proper zombie process prevention with `Wait()` after `Kill()`

### Phase 5: Infrastructure & Polish (Week 9-10) ✅

**Development Tooling:**
- Dynamic module discovery in Makefile (uses `find` instead of hardcoded lists)
- Added missing Makefile targets (`test-integration`, `test-coverage`, `fmt`, `mod-tidy`, `install`)
- Created `.editorconfig` with Go, YAML, JSON, Markdown, Shell, Proto configs
- Created `.lefthook.yml` with pre-commit, pre-push, commit-msg hooks
- Created `.air.toml` for watch mode with auto-rebuild

**Performance:**
- Parser scanner optimization: 10-20% performance improvement (6-13% measured)
- Replaced save/restore mechanism with direct string scanning
- Reduced memory allocations in token peeking operations

**Provider Libraries:**
- Provider-downloader: Added progress reporting callback and configurable HTTP timeout
- Provider-proto: Added error documentation, reserved fields, `STATUS_STARTING` enum

### Module-Specific Changes

See individual module CHANGELOGs for complete details:
- [apps/command-line/CHANGELOG.md](apps/command-line/CHANGELOG.md) - CLI v1.0.0
- [libs/parser/CHANGELOG.md](libs/parser/CHANGELOG.md) - Parser v0.8.0
- [libs/compiler/CHANGELOG.md](libs/compiler/CHANGELOG.md) - Compiler v0.7.0
- [libs/provider-downloader/CHANGELOG.md](libs/provider-downloader/CHANGELOG.md) - Provider Downloader v0.1.0
- [libs/provider-proto/CHANGELOG.md](libs/provider-proto/CHANGELOG.md) - Provider Proto v0.2.0

### Migration Guides

**For CLI Users:**
- Run `nomos init` after upgrading (in-process providers removed)
- Update CI/CD scripts checking exit code `2` to check for `1` instead
- Update `.csl` files to use `type: 'owner/repo'` format for provider types
- Set `GITHUB_TOKEN` environment variable for higher GitHub API rate limits

**For Compiler Library Users:**
- Replace `snapshot, err := compiler.Compile(ctx, opts)` with `result := compiler.Compile(ctx, opts)`
- Access snapshot via `result.Snapshot`
- Check errors via `result.HasErrors()` or `result.Error()`
- Access individual errors via `result.Errors()` and warnings via `result.Warnings()`

**For Provider Developers:**
- Review error documentation in proto comments
- Implement graceful shutdown (5-second timeout before force termination)
- Consider using `STATUS_STARTING` for lengthy initialization periods

### Success Metrics

- **Test Coverage**: Parser 86.9%, Compiler 50%+, Provider Downloader 81.8%
- **Code Quality**: Zero linting errors across all modules
- **Architecture**: 100+ LOC removed through consolidation, 33% reduction in compiler.go
- **Developer Experience**: Comprehensive documentation, standardized tooling, clear testing conventions

### What's Next: Phase 6 (Week 11-12)

Phase 6 (Documentation & Finalization) is planned for future work:
- Update all module README files with Phase 1-5 changes
- Create comprehensive testing guide (`docs/TESTING_GUIDE.md`)
- Update architecture documentation
- Add release automation workflows
- API cleanup and deprecation planning

[Unreleased]: https://github.com/autonomous-bits/nomos/compare/v1.0.0...HEAD
