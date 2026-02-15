# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed
- **BREAKING CHANGE: Path-based reference syntax** (Feature 006-expand-at-references)
  - References now treat everything after the first `:` as provider path segments: `@alias:path`
  - Root references use `@alias:.` to include all properties at the provider root
  - Map references: `@alias:path.to.map` includes specific nested map
  - Property references: `@alias:path.to.property` resolves single value
  - Deep merge semantics: Properties after reference override using deep merge (preserve siblings)
  - Circular reference detection: Tracks resolution chain and fails with full cycle path
  - File provider: Filename path segment does not require `.csl` extension (auto-appended)
  - Migration: Update all references to use `@alias:path`; see migration guide
  - See [Migration Guide](../../docs/guides/expand-at-references-migration.md)

### Added
- **Reference resolution modes** (Feature 006-expand-at-references)
  - `ReferenceMode` enum: PropertyMode, MapMode, RootMode
  - `ResolvedReference` struct with mode-specific Value or Entries
  - `ResolutionContext` for circular reference detection with Push/Pop/formatCycle
  - `DetermineReferenceMode` function to classify references by path structure
  - `ResolveReference` function with context-aware resolution and error wrapping
- **Deep merge implementation** (Feature 006-expand-at-references)
  - `DeepMerge` function with recursive map merging
  - Array replacement (no deep array merge)
  - Scalar last-wins override
  - Sibling property preservation during nested overrides
- **Comprehensive error types** (Feature 006-expand-at-references)
  - `ErrAliasNotFound`: Source alias not configured
  - `ErrPathNotFound`: Provider cannot resolve path
  - `ErrPropertyPathInvalid`: Property path doesn't exist (includes available keys)
  - `ErrCircularReference`: Cycle detected in resolution chain
  - All errors include source span for precise error reporting

### Fixed
- [Compiler] Converter properly handles `SectionDecl.Value` field for inline scalars, producing flat output structure compatible with tfvars format

## [0.7.0] - 2025-12-26

### BREAKING CHANGES
- **`Compile()` now returns `CompilationResult` instead of `(Snapshot, error)`**
  - Use `result.Snapshot` to access snapshot data
  - Use `result.HasErrors()` to check for errors
  - Use `result.Error()` to get a combined error (returns nil if no errors)
  - Use `result.Errors()` to get individual error messages
  - Use `result.HasWarnings()` and `result.Warnings()` for warning access
  - Migration: Replace `snapshot, err := compiler.Compile(ctx, opts)` with `result := compiler.Compile(ctx, opts)` followed by `snapshot := result.Snapshot` and error checking via `result.HasErrors()`
  - Updated 20+ call sites throughout CLI and tests

### Added
- Created `internal/core` package with unified provider interfaces
  - Eliminated adapter pattern duplication between compiler and imports packages (~100 LOC removed)
  - All components now use `core.Provider`, `core.ProviderRegistry`, and `core.ProviderTypeRegistry`
  - Added `core.ProviderTypeConstructor` type alias for consistency
  - Removed `imports_adapters.go` boilerplate
- Multi-error collection with `CompilationResult`
  - Compilation now attempts to continue through recoverable errors
  - Collects all issues in a single run instead of stopping at first error
  - Provides better developer experience with comprehensive error reporting
  - Added convenience methods: `HasErrors()`, `HasWarnings()`, `Error()`, `Errors()`, `Warnings()`
- Organized compilation stages into `internal/pipeline` package
  - `pipeline.DiscoverInputFiles()` for file discovery with deterministic ordering (64 lines)
  - `pipeline.InitializeProvidersFromSources()` for provider initialization from source declarations (89 lines)
  - `pipeline.ResolveReferences()` for reference resolution with provider integration (64 lines)
  - Improved code organization and separation of concerns
  - Reduced `compiler.go` from 510 to 341 lines (-33% reduction)
- Configurable graceful shutdown for provider Manager
  - Added `ManagerOptions` with `ShutdownTimeout` configuration (default: 5 seconds)
  - `NewManagerWithOptions()` function for custom timeout configuration
  - Improved error path cleanup in `GetProvider()` to prevent resource leaks
- Import resolution test coverage improvements
  - Integration tests for import chains, cycles, and error cases
  - Coverage improved from 21.3% to 75%+
  - Comprehensive test suite with resolver integration tests

### Changed
- Import resolution now uses explicit error instead of nil sentinel
  - `resolveFileImports()` returns `ErrImportResolutionNotAvailable` when type registry is unavailable
  - Eliminates implicit control flow and makes error handling explicit
- Manager now attempts graceful provider shutdown before force termination
  - Providers are given `ShutdownTimeout` to respond to Shutdown RPC and exit gracefully
  - Processes that don't exit within timeout are forcefully terminated
  - Shutdown reports timeout as informational error (providers are still terminated successfully)
  - All provider subprocesses are properly reaped to prevent zombie processes
- Manager implementation moved to `internal/providers` package
  - Public API wrapper maintained at root for backward compatibility
  - Improved internal package organization

### Fixed
- Context propagation bugs in provider initialization
  - Thread context from `Compile()` through to provider initialization
  - Replaced `context.Background()` in `provider.go` and `provider_type_registry.go`
  - Proper cancellation and timeout support throughout compilation pipeline
- Provider subprocesses now have proper cleanup on all error paths during initialization
- Zombie processes prevented by calling `Wait()` after `Kill()`

### Testing
- Consolidated test infrastructure: 6 mock implementations → 1 shared `FakeProviderServer`
  - Migrated `mockProviderRegistry`, `fakeProviderRegistry`, `integrationProviderRegistry`, `concurrencyTestRegistry`, `benchProviderRegistry`
  - Created `testutil/provider_registry.go` with configurable behavior
  - Fixed import cycle by creating local fakes in internal/resolver
- Added E2E smoke test suite at repo root (`test/e2e/smoke_test.go`)
  - 4/4 smoke tests passing: compilation pipeline, provider references, snapshot determinism, error handling
  - Fixed parser syntax limitations and source declaration syntax
  - Implemented `simpleProvider` test double
  - Added `integration` build tag for proper test isolation
- All 500+ unit tests passing across entire compiler module
- Zero compilation errors across entire module

## [0.1.1] - 2025-12-26

### Security
- **CRITICAL: Mandatory checksum validation for provider binaries** - Provider binaries are now required to have checksums in the lockfile and are verified before execution. This prevents execution of tampered or corrupted binaries. Lockfiles without checksums will fail with a clear error message directing users to run `nomos build`.

## [0.1.0] - 2025-11-02

Initial release of the Nomos compiler library.

### BREAKING CHANGES
- **In-process providers removed** (#51): All providers must now be external executables via gRPC
  - Removed `libs/compiler/providers/file` package
  - Users must run `nomos build` to install provider binaries
  - File provider distributed separately at `github.com/autonomous-bits/nomos-provider-file`
  - See migration guide: `docs/guides/external-providers-migration.md`

### Added
- Reusable fake provider gRPC server (`test/fakes/FakeProviderServer`) for testing
- External provider support via `internal/providerproc` package with process management and gRPC delegation
- Remote provider resolution with `ProviderResolver` and `ProviderManager` interfaces
- Provider configuration management (`internal/config`) with lockfile and manifest support
- Parser integration for `.csl` files with structured diagnostics
- Provider registry with caching and concurrency support
- Comprehensive test infrastructure including benchmarks and integration tests

### Changed
- Refactored tests to use centralized `FakeProviderServer`

[Unreleased]: https://github.com/autonomous-bits/nomos/compare/libs/compiler/v0.7.0...HEAD
[0.7.0]: https://github.com/autonomous-bits/nomos/compare/libs/compiler/v0.1.1...libs/compiler/v0.7.0
[0.1.1]: https://github.com/autonomous-bits/nomos/compare/libs/compiler/v0.1.0...libs/compiler/v0.1.1
[0.1.0]: https://github.com/autonomous-bits/nomos/releases/tag/libs/compiler/v0.1.0
