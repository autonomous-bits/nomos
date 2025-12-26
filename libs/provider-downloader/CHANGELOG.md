# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Progress reporting**: `ProgressCallback` option for download progress updates
  - Add `ProgressCallback` field to `ClientOptions` for receiving download progress
  - Callback signature: `func(downloaded, total int64)`
  - Called periodically during download with bytes downloaded and total size
  - Useful for CLI progress indicators and UI integration
  - Comprehensive tests for progress callback functionality
- **Configurable HTTP timeout**: `HTTPTimeout` field in `ClientOptions`
  - Make HTTP client timeout configurable (previously hardcoded to 30s)
  - Default: 30 seconds
  - Sensible zero-value handling (falls back to default)
  - Tests for custom, default, and zero timeout values
- **Caching support**: Optional binary caching to avoid redundant downloads
  - Cache keyed by SHA256 checksum
  - Only caches when `AssetInfo.Checksum` is provided
  - Configure via `ClientOptions.CacheDir`
  - Cache hit avoids network calls entirely
  - Comprehensive cache tests (hit/miss/disabled scenarios)
- **Archive extraction**: Automatic extraction of provider binaries from archives
  - Support for `.tar.gz`, `.tgz`, and `.zip` formats
  - Automatic format detection based on file extension
  - Searches for `provider` or `nomos-provider-*` binaries in archives
  - Refactored into `internal/archive` package with `Extractor` interface
  - Separate `TarGzExtractor` and `ZipExtractor` implementations
- **Integration test suite**: Comprehensive end-to-end testing
  - Full download → extract → install flow testing
  - Multiple provider sequence downloads
  - Concurrent download tests with race detection
  - Archive extraction integration tests
  - Cache efficiency tests
  - Context cancellation tests
- Initial public API with `NewClient`, `ResolveAsset`, and `DownloadAndInstall`
- Core types: `ClientOptions`, `ProviderSpec`, `AssetInfo`, `InstallResult`
- Typed errors for common failure modes (`ErrAssetNotFound`, `ErrChecksumMismatch`, etc.)
- Package documentation and comprehensive README
- GitHub Releases asset resolver with intelligent matching:
  - Exact pattern matching (repo-os-arch, nomos-provider-os-arch, repo-os)
  - Substring fallback matching with case-insensitive search
  - Architecture variant handling (amd64/x86_64, arm64/aarch64)
  - Auto-detection of OS and Arch from runtime when not specified
  - Version normalization (handles v-prefix variations)
- Streaming download implementation with atomic install:
  - Downloads to temporary file while computing SHA256 checksum incrementally
  - Verifies checksum against expected value if provided
  - Atomically renames to final destination with 0755 permissions
  - Handles cross-filesystem renames gracefully
- Robust retry logic for transient network failures:
  - Exponential backoff with jitter (configurable retry count and delay)
  - Automatic retry on 5xx errors, timeouts, and connection issues
  - File reset between retries to ensure clean download attempts
  - Context cancellation support throughout download lifecycle
- Comprehensive unit test suite with httptest-based hermetic testing (85.9% coverage, up from 81.8%)
- Additional unit tests for timeout handling (context deadline exceeded, slow but successful downloads)
- Public `testutil` package with `BinaryFixture` utilities for generating deterministic test binaries with SHA256 checksums
  - `CreateBinaryFixture(t, size, prefix)` generates reproducible test binaries
  - `CreateCorruptedFixture(t, size, prefix)` creates fixtures for testing checksum validation failures

### Changed
- **Code modernization**: Replaced custom `contains()` function with `strings.Contains()` from stdlib
  - Reduces code duplication and uses idiomatic Go
  - Applied across `download.go` and `internal/archive` package
  - Maintains backward compatibility
- Refactored archive extraction logic into separate `internal/archive` package
- Improved code organization with cleaner separation of concerns
- Enhanced README with caching and archive extraction documentation

[Unreleased]: https://github.com/autonomous-bits/nomos/compare/libs/provider/downloader/v0.0.0...HEAD
