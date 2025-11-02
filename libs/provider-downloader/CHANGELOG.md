# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
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
- Comprehensive unit test suite with httptest-based hermetic testing (81.6% coverage)
- Additional unit tests for timeout handling (context deadline exceeded, slow but successful downloads)
- Public `testutil` package with `BinaryFixture` utilities for generating deterministic test binaries with SHA256 checksums
  - `CreateBinaryFixture(t, size, prefix)` generates reproducible test binaries
  - `CreateCorruptedFixture(t, size, prefix)` creates fixtures for testing checksum validation failures

[Unreleased]: https://github.com/autonomous-bits/nomos/compare/libs/provider/downloader/v0.0.0...HEAD
