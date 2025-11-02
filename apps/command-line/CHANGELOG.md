# Changelog

All notable changes to the Nomos CLI will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### BREAKING CHANGES
- **`--from` flag removed from `nomos init`** (#72): Providers now installed from GitHub Releases
  - The `--from alias=path` flag has been removed in favor of automatic downloads from GitHub Releases
  - Provider `type` field in `.csl` source declarations must now use `owner/repo` format (e.g., `autonomous-bits/nomos-provider-file`)
  - Providers are automatically downloaded from GitHub Releases based on `owner/repo` and `version`
  - Installation path changed to `.nomos/providers/{owner}/{repo}/{version}/{os-arch}/provider`
  - Lockfile entries now include GitHub metadata (owner, repo, release_tag, asset) and SHA256 checksum
  - For local development, use GitHub Releases or manually install providers to the expected path
  - Migration: Update `.csl` files to use `type: 'owner/repo'` format instead of simple names like `type: 'file'`
  - Set `GITHUB_TOKEN` environment variable for higher rate limits and private repository access
- **In-process providers removed** (#51): External providers now required
  - `NewProviderRegistries()` no longer registers in-process `file` provider as fallback
  - Missing or malformed lockfile (`.nomos/providers.lock.json`) returns empty registry
  - Build fails with clear error message directing users to run `nomos init`
  - Example error: "provider type 'file' not found: external providers are required (in-process providers removed in v0.3.0). Run 'nomos init' to install provider binaries."
  - Removed import of `github.com/autonomous-bits/nomos/libs/compiler/providers/file`
  - Migration guide: `docs/guides/external-providers-migration.md`
  - CI/CD pipelines must add `nomos init` step before `nomos build`

### Added
- [CLI] `nomos init` command for discovering and installing provider dependencies (#46, #72)
  - Scans `.csl` files to discover provider requirements (alias, type, version)
  - Validates that all providers have required `version` field in source declarations
  - Downloads and installs provider binaries from GitHub Releases based on `owner/repo` type format
  - Creates `.nomos/providers/{owner}/{repo}/{version}/{os-arch}/provider` directory structure
  - Writes `.nomos/providers.lock.json` with resolved versions, GitHub metadata, SHA256 checksums, and paths
  - Supports `--dry-run` flag to preview actions without installing
  - Supports `--force` flag to overwrite existing providers/lockfile
  - Supports `--os` and `--arch` flags to override target platform (auto-detected from runtime by default)
  - Supports `--upgrade` flag for future version upgrade functionality
  - Sets executable permissions (0755) on installed provider binaries
  - Automatic asset resolution based on OS/architecture from GitHub Releases
  - Streaming downloads with retry logic and SHA256 verification
  - Clear error messages for missing version, network failures, asset not found, and checksum mismatches
  - Comprehensive unit and integration test coverage (including hermetic httptest mocks)
  - Hermetic httptest-based tests for init flow that mock GitHub API and asset servers (init_hermetic_test.go)
  - Enhanced gated network integration test with complete GitHub metadata validation (release_tag, asset fields)
  - Lockfile entries now include complete GitHub metadata (owner, repo, release_tag, asset) and binary size (#73)
  - Lockfile includes RFC3339 timestamp recording when providers were installed
  - Atomic lockfile writes using temp file + rename pattern for crash safety
  - Skip-download optimization: providers matching lockfile entries by version/checksum are not re-downloaded unless `--force` is used
  - Comprehensive unit tests verify lockfile schema, atomic writes, and skip logic
- [CLI] Initial implementation of `nomos build` command for compiling .csl files to configuration snapshots
- [CLI] Flag parsing support for `--path/-p`, `--format/-f`, `--out/-o`, `--var`, `--strict`, `--allow-missing-provider`, `--timeout-per-provider`, `--max-concurrent-providers`, and `--verbose`
- [CLI] JSON output format support (default) with canonical serialization for deterministic output (#39)
- [CLI] Canonical serialization guarantees byte-for-byte stable output for CI reproducibility:
  - Alphabetically sorted object keys at all nesting levels
  - UTF-8 string normalization
  - Deterministic structure ordering (metadata timestamps naturally vary per execution)
- [CLI] YAML and HCL format stubs (--format yaml|hcl returns clear "not yet implemented" error)
- [CLI] Enhanced file output handling with automatic directory creation for --out paths
- [CLI] Comprehensive help text with `--help` flag showing usage examples
- [CLI] Exit code mapping: 0 for success, 1 for compilation errors, 2 for usage errors
- [CLI] Deterministic file discovery and ordering:
  - Recursive discovery of `.csl` files in directories using UTF-8 lexicographic sort
  - Handles nested directories, symlinks, and symlink loop detection
  - Consistent ordering across platforms for reproducible builds
  - Clear error messages for empty directories and unreadable files
  - Documented ordering semantics in README and `--help` text
- [CLI] Variable substitution via repeatable `--var key=value` flags
- [CLI] Diagnostic output with warnings and errors from compiler metadata
- [CLI] Strict mode (`--strict`) treating warnings as errors
- [CLI] Integration with libs/compiler for compilation semantics
- [CLI] Test coverage: unit tests for flag parsing and integration tests for CLI behavior
- [CLI] Enhanced diagnostics formatting with file:line:col information (#40)
  - Created `internal/diagnostics` package for formatting compiler errors and warnings
  - Diagnostics preserve compiler's detailed formatting including source snippets and caret markers
  - Support for optional ANSI color output (infrastructure in place for future enhancement)
  - Clear separation of errors (stderr) and warnings (stderr) with proper formatting
  - Integration tests validating exit code behavior for various compilation scenarios
  - Documentation updates in README and `--help` text describing diagnostic formats
  - Comprehensive unit test coverage for diagnostics formatter
- [CLI] Comprehensive test suite and CI automation (#41)
  - Unit tests for all internal packages: flags (93.3%), options (100%), diagnostics (94.6%), serialize (75%), traverse (82.9%)
  - Integration tests using built CLI binary to validate end-to-end behavior
  - Determinism test validating byte-for-byte JSON reproducibility across 10 runs
  - CI workflow (`.github/workflows/cli-ci.yml`) with 4 jobs: unit tests, integration tests, determinism test, linting
  - Coverage threshold enforcement (80% minimum) in CI
  - Race detector enabled for all test runs
  - Offline-by-default test execution (no network dependencies)
  - Comprehensive testing documentation in README with examples and troubleshooting
- [Internal] `traverse` package for deterministic file discovery with comprehensive test coverage
- [Documentation] Enhanced README.md with comprehensive documentation (#42):
  - Added explicit PRD reference linking to issue #35
  - Added links to `libs/compiler` documentation for compiler-level details
  - Enhanced installation section with prerequisites and verification steps
  - Added prominent "Network and Safety Defaults" section explaining offline-first behavior
  - Updated all examples to reference `testdata/` fixtures for reproducibility
  - Added "Running Examples" subsection in Testing section
  - Improved macOS-focused installation guidance
- [Documentation] Enhanced help text with networking and safety notes (#42):
  - Added "Network and Safety" section to `nomos build --help` explaining offline-first behavior
  - Fixed duplicate "Examples:" and "Exit Codes:" sections in build help
  - Reorganized help text for better clarity and consistency with README
  - Added explicit `-h, --help` flag in Options section
- [Tests] Automated help text content validation (#42):
  - Created `test/help_test.go` with comprehensive help text assertions
  - Tests verify presence of all required flags, sections, and keywords
  - Tests check for duplicate sections and consistency issues
  - Validates networking and determinism keywords in help output
- [Tests] README example smoke tests (test infrastructure) (#42):
  - Created `test/readme_examples_test.go` for validating README examples
  - Tests verify examples can be executed and produce expected outputs
  - Tests validate file output behavior and deterministic compilation
  - Currently skipped due to pre-existing compiler bug (imports.ExtractImports nil pointer)
  - Will be enabled once compiler issue is resolved
- [Tests] Test data fixtures for documentation examples (#42):
  - Created `testdata/simple.csl` for basic single-file examples
  - Created `testdata/configs/` directory with multi-file examples (1-base.csl, 2-network.csl)
  - Created `testdata/with-vars.csl` for variable substitution examples
  - All fixtures work offline and demonstrate deterministic behavior
- [Internal] `options` package for building compiler.Options with provider wiring and dependency injection (#38)
- [Internal] Integration tests verifying options builder with compiler test doubles (#38)

### Changed
- [Internal] Refactored `build.go` to use `options.BuildOptions()` for improved testability and provider wiring (#38)
- [Internal] Provider registries now use factory pattern (`options.NewProviderRegistries()`) supporting custom injection for tests (#38)

[Unreleased]: https://github.com/autonomous-bits/nomos/compare/main...HEAD
