# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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