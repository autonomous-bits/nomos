# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Phase 4.1**: Created `internal/core` package with unified provider interfaces
  - Eliminated adapter pattern duplication between compiler and imports packages
  - All components now use `core.Provider`, `core.ProviderRegistry`, and `core.ProviderTypeRegistry`
  - Added `core.ProviderTypeConstructor` type alias for consistency
- **Phase 4.3**: Configurable graceful shutdown for provider Manager
  - Added `ManagerOptions` with `ShutdownTimeout` configuration (default: 5 seconds)
  - `NewManagerWithOptions()` function for custom timeout configuration
  - Manager implementation moved to `internal/providers` package with public API wrapper at root
  - Improved error path cleanup in `GetProvider()` to prevent resource leaks
- **Phase 4.4**: Multi-error collection with CompilationResult
  - Added `CompilationResult` struct with `Snapshot` and helper methods (`HasErrors()`, `HasWarnings()`, `Error()`, `Errors()`, `Warnings()`)
  - Compilation now attempts to continue through recoverable errors to collect all issues in a single run
  - Provides better developer experience by reporting multiple errors at once instead of requiring iterative fixes
- **Phase 4.5**: Organized compilation stages into `internal/pipeline` package
  - Created `pipeline.DiscoverInputFiles()` for file discovery with deterministic ordering
  - Created `pipeline.InitializeProvidersFromSources()` for provider initialization from source declarations
  - Created `pipeline.ResolveReferences()` for reference resolution with provider integration
  - Improved code organization and separation of concerns
  - Reduced compiler.go from 510 to 340 lines by extracting stage functions

### Changed
- **Phase 4.2**: Import resolution now uses explicit error instead of nil sentinel
  - `resolveFileImports()` returns `ErrImportResolutionNotAvailable` when type registry is unavailable
  - Eliminates implicit control flow and makes error handling explicit
- **Phase 4.3**: Manager now attempts graceful provider shutdown before force termination
  - Providers are given `ShutdownTimeout` to respond to Shutdown RPC and exit gracefully
  - Processes that don't exit within timeout are forcefully terminated
  - Shutdown reports timeout as informational error (providers are still terminated successfully)
  - All provider subprocesses are properly reaped to prevent zombie processes
- **Phase 4.4**: **BREAKING**: `Compile()` now returns `CompilationResult` instead of `(Snapshot, error)`
  - Use `result.Snapshot` to access snapshot data
  - Use `result.HasErrors()` to check for errors
  - Use `result.Error()` to get a combined error (returns nil if no errors)
  - Use `result.Errors()` to get individual error messages
  - Use `result.HasWarnings()` and `result.Warnings()` for warning access
  - Migration: Replace `snapshot, err := compiler.Compile(ctx, opts)` with `result := compiler.Compile(ctx, opts)` followed by `snapshot := result.Snapshot` and error checking via `result.HasErrors()`

### Fixed
- Provider subprocesses now have proper cleanup on all error paths during initialization
- Zombie processes are prevented by calling `Wait()` after `Kill()`

## [0.1.1] - 2025-12-26

### Security
- **CRITICAL: Mandatory checksum validation for provider binaries** - Provider binaries are now required to have checksums in the lockfile and are verified before execution. This prevents execution of tampered or corrupted binaries. Lockfiles without checksums will fail with a clear error message directing users to run `nomos init`.

## [0.1.0] - 2025-11-02

Initial release of the Nomos compiler library.

### BREAKING CHANGES
- **In-process providers removed** (#51): All providers must now be external executables via gRPC
  - Removed `libs/compiler/providers/file` package
  - Users must run `nomos init` to install provider binaries
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

[Unreleased]: https://github.com/autonomous-bits/nomos/compare/libs/compiler/v0.1.1...HEAD
[0.1.1]: https://github.com/autonomous-bits/nomos/compare/libs/compiler/v0.1.0...libs/compiler/v0.1.1
[0.1.0]: https://github.com/autonomous-bits/nomos/releases/tag/libs/compiler/v0.1.0

[Unreleased]: https://github.com/autonomous-bits/nomos/compare/libs/compiler/v0.1.0...HEAD
[0.1.0]: https://github.com/autonomous-bits/nomos/releases/tag/libs/compiler/v0.1.0