# Changelog

All notable changes to the Nomos CLI will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- [CLI] YAML output format support via `--format yaml` flag
- [CLI] Terraform .tfvars output format support via `--format tfvars` flag
- [CLI] Automatic file extension handling for all output formats (`.json`, `.yaml`, `.tfvars`)
- [CLI] Format-specific key validation (HCL identifiers for tfvars, null byte check for YAML)
- [CLI] Format-specific type handling documentation in README explaining type preservation differences across JSON, YAML, and tfvars formats
- [CLI] Validation for negative `--max-concurrent-providers` flag values (rejects with clear error message)
- [CLI] `--include-metadata` flag to restore metadata in build output (opt-in for debugging and auditing) (#005)

### Changed
- [CLI] **BREAKING**: Default build output now excludes metadata for cleaner, production-ready configs. Metadata is now opt-in via `--include-metadata` flag. Previous behavior (metadata included by default) can be restored with this flag (#005)
- [CLI] Exit code for I/O errors (non-writable output paths) is now 1 (runtime error) instead of 2

### Fixed
- [CLI] Non-writable output path test now uses portable read-only directory approach with correct exit code expectation
- [CLI] Parser now uses `Value` field for inline scalar values instead of empty-string keys, enabling clean HCL/tfvars serialization
- [CLI] Compiler output structure is now clean and flat for scalar values, fully supporting tfvars format

## [2.0.0] - 2026-01-11

### Added
- [CLI] Automatic provider download during build
  - Providers are discovered from `.csl` source declarations and downloaded on first build
  - Downloaded providers are cached in `.nomos/providers/` for reuse
  - Lockfile `.nomos/providers.lock.json` is created/updated automatically
  - Eliminates separate provider-installation step
- [CLI] `--force-providers` flag for `nomos build` command
  - Forces re-download of all providers even if cached versions exist
  - Useful for debugging provider issues or forcing provider updates
  - Updates checksums in lockfile after successful downloads
- [CLI] `--dry-run` flag for `nomos build` command
  - Previews provider downloads without actually downloading or building
  - Shows which providers would be downloaded and from where
  - Useful for verifying provider configuration before execution

## [1.1.0] - 2025-12-28

### Removed
- [CLI] YAML and HCL serialization stubs removed (#phase6)
  - `ToYAML()` and `ToHCL()` functions removed from `internal/serialize`
  - `--format` flag now only accepts `json` (was: json, yaml, hcl)
  - Documentation updated to reflect JSON-only support
  - Rationale: No user demand identified, reduces maintenance burden, avoids misleading users
  - Note: YAML/HCL support may be added in future releases if requested

## [1.0.0] - 2025-12-26

First production release of the Nomos CLI with complete Cobra framework integration.

### BREAKING CHANGES
- **CLI migrated to Cobra framework** (#phase2): Major UX overhaul with new command structure
  - All commands now use Cobra framework for consistent behavior and help text
  - Exit codes changed: Cobra returns exit code `1` for all errors (previously used `2` for usage errors)
  - Help text format updated to Cobra standard ("Available Commands:", "Flags:" instead of "Commands:", "Options:")
  - Error messages now use Cobra's standard format ("Error: required flag(s) not set" instead of custom messages)
  - Command invocation unchanged, but internal structure refactored
  - Migration: CI/CD scripts checking for exit code `2` should check for exit code `1` instead
- **In-process providers removed** (#51): External providers now required
  - `NewProviderRegistries()` no longer registers in-process `file` provider as fallback
  - Missing or malformed lockfile (`.nomos/providers.lock.json`) returns empty registry
  - Build fails with clear error message directing users to run `nomos build`
  - Example error: "provider type 'file' not found: external providers are required (in-process providers removed in v0.3.0). Run 'nomos build' to install provider binaries."
  - Removed import of `github.com/autonomous-bits/nomos/libs/compiler/providers/file`
  - Migration guide: `docs/guides/external-providers-migration.md`

### Added
- [CLI] Shell completion support for Bash, Zsh, Fish, and PowerShell via `nomos completion` command
- [CLI] `--color` global flag with `auto`, `always`, `never` modes for colored output control
- [CLI] `--quiet` global flag to suppress non-error output
- [CLI] `nomos version` command displaying version, commit, build date, and Go version
- [CLI] `nomos validate` command for syntax-only validation without building
  - Performs parsing and type checking without provider invocation
  - Useful for pre-commit hooks, CI/CD pipelines, and editor integrations
  - Supports `--path` flag to specify files or directories to validate
- [CLI] `nomos providers list` command to display installed providers
  - Shows table with alias, type, version, OS, arch, and path
  - Supports `--json` flag for machine-readable output
- [CLI] Colored diagnostics output with `fatih/color` integration
  - Errors displayed in red, warnings in yellow (when color enabled)
  - Headers added: "Errors:" and "Warnings:" sections for clarity
  - Respects `--color` flag and terminal detection
- [CLI] Validation summary in build output
  - Shows "Compilation failed: N error(s), M warning(s)" after diagnostics
  - Provides clear feedback on build status
- [CLI] Success message when output file written: "Output written to <path>"
  - Lockfile includes RFC3339 timestamp recording when providers were installed
  - Atomic lockfile writes using temp file + rename pattern for crash safety
  - Skip-download optimization: providers matching lockfile entries by version/checksum are not re-downloaded unless `--force` is used

### Changed
- [CLI] All command handlers now return errors instead of calling `os.Exit()` directly
  - Improves testability and allows for proper error handling in main()
  - All exits now occur in `main()` function only
- [CLI] Help text improved with Cobra's structured format
  - Long descriptions with examples and usage notes
  - Consistent flag documentation across all commands
  - Better organization with sections for flags, exit codes, and examples

### Dependencies
- Added `github.com/spf13/cobra` v1.10.2 for CLI framework
- Added `github.com/spf13/pflag` v1.0.9 for POSIX/GNU flag parsing
- Added `github.com/olekukonko/tablewriter` v1.1.2 for table output
- Added `github.com/briandowns/spinner` v1.23.2 for progress indicators
- Added `github.com/fatih/color` v1.18.0 for colored output
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

[Unreleased]: https://github.com/autonomous-bits/nomos/compare/apps/command-line/v2.0.0...HEAD
[2.0.0]: https://github.com/autonomous-bits/nomos/compare/apps/command-line/v1.1.0...apps/command-line/v2.0.0
[1.1.0]: https://github.com/autonomous-bits/nomos/compare/apps/command-line/v1.0.0...apps/command-line/v1.1.0
[1.0.0]: https://github.com/autonomous-bits/nomos/releases/tag/apps/command-line/v1.0.0
