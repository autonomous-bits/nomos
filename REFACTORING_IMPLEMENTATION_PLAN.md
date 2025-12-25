# Nomos Codebase Refactoring & Optimization Implementation Plan

**Generated:** 2025-12-25  
**Status:** Planning Phase  
**Estimated Total Effort:** ~10-12 weeks (1 developer)

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Phase 1: Critical Fixes & Foundation (Week 1-2)](#phase-1-critical-fixes--foundation-week-1-2)
3. [Phase 2: CLI Modernization (Week 3-4)](#phase-2-cli-modernization-week-3-4)
4. [Phase 3: Testing & Quality (Week 5-6)](#phase-3-testing--quality-week-5-6)
5. [Phase 4: Compiler Refactoring (Week 7-8)](#phase-4-compiler-refactoring-week-7-8)
6. [Phase 5: Infrastructure & Polish (Week 9-10)](#phase-5-infrastructure--polish-week-9-10)
7. [Phase 6: Documentation & Finalization (Week 11-12)](#phase-6-documentation--finalization-week-11-12)
8. [Appendix: Detailed Findings](#appendix-detailed-findings)

---

## Executive Summary

This plan consolidates findings from comprehensive analysis of all Nomos modules:

### Key Findings by Module

| Module | Overall Grade | Critical Issues | Coverage | Priority Focus |
|--------|---------------|-----------------|----------|----------------|
| **CLI** (apps/command-line) | B+ | No Cobra framework, scattered `os.Exit()` calls, no color support | Integration tests good | Cobra migration, UX improvements |
| **Parser** (libs/parser) | B+ | 0% error handling coverage, debug tests | 44.2% overall | Error handling tests, remove debug code |
| **Compiler** (libs/compiler) | B | Adapter proliferation, 21.3% imports coverage, context misuse | 50.1% overall | Consolidate adapters, fix context propagation |
| **Provider Downloader** (libs/provider-downloader) | A- | Debug logging pollution, no caching implementation | 81.6% | Remove debug logs, implement caching |
| **Provider Proto** (libs/provider-proto) | A | Trivial contract tests, no real gRPC validation | N/A | Rewrite contract tests |
| **Monorepo Governance** | B+ | Incomplete Makefile, no linting config, no CI for provider-downloader | N/A | Add missing targets, standardize tooling |

### Strategic Priorities

1. **Immediate** (Weeks 1-2): Fix critical bugs, security issues, and test gaps
2. **Near-term** (Weeks 3-4): Modernize CLI with Cobra, improve UX
3. **Medium-term** (Weeks 5-8): Comprehensive testing, refactor compiler architecture
4. **Long-term** (Weeks 9-12): Infrastructure improvements, documentation, polish

---

## Phase 1: Critical Fixes & Foundation (Week 1-2)

**Goal:** Resolve critical bugs, security issues, and test coverage gaps

### 1.1 Parser Module Critical Fixes

- [ ] **[CRITICAL]** Remove debug test files
  - [ ] Delete `test/debug_test.go` (37 lines of debug code)
  - [ ] Delete `test/scanner_debug_test.go` (36 lines of debug code)
  - **Effort:** 15 minutes | **Impact:** HIGH

- [ ] **[CRITICAL]** Fix benchmark suite
  - [ ] Update `parser_bench_test.go` to use inline reference syntax
  - [ ] Remove deprecated `reference:base:config.database` line
  - **Effort:** 15 minutes | **Impact:** HIGH

- [ ] **[CRITICAL]** Add error handling test suite (0% → 80% coverage)
  - [ ] Create `test/error_formatting_test.go`
  - [ ] Test `FormatParseError()` with various error types
  - [ ] Test UTF-8 handling in `generateSnippet()`
  - [ ] Test edge cases (empty source, out-of-bounds lines)
  - [ ] Test error unwrapping logic
  - **Effort:** 1-2 days | **Impact:** HIGH

- [ ] **[CRITICAL]** Resolve test fixture inconsistencies
  - [ ] Review 4 "knownValid" files in golden error tests
  - [ ] Fix parser to reject them OR update test expectations
  - [ ] Document intentional validation gaps
  - **Effort:** 1 day | **Impact:** MEDIUM

### 1.2 Compiler Module Critical Fixes

- [ ] **[CRITICAL]** Fix import resolution test coverage (21.3% → 75%)
  - [ ] Add integration tests for import chains
  - [ ] Test import cycles
  - [ ] Test error cases in import resolution
  - **Effort:** 1-2 days | **Impact:** HIGH

- [ ] **[CRITICAL]** Fix context propagation bugs
  - [ ] Thread context from `Compile()` through to provider initialization
  - [ ] Replace `context.Background()` in `provider.go:106`
  - [ ] Replace `context.Background()` in `provider_type_registry.go:105`
  - **Effort:** 2-4 hours | **Impact:** HIGH

- [ ] **[CRITICAL]** Add provider binary validation
  - [ ] Implement checksum verification in `lockfile_resolver.go`
  - [ ] Validate checksums before executing provider binaries
  - [ ] Add tests for checksum validation
  - **Effort:** 4-6 hours | **Impact:** HIGH (security)

### 1.3 Provider Downloader Critical Fixes

- [ ] **[CRITICAL]** Remove debug logging pollution
  - [ ] Remove all 20+ `fmt.Printf()` debug statements from `download.go`
  - [ ] Add optional logger interface to `ClientOptions` (if needed)
  - [ ] Update tests to not rely on debug output
  - **Effort:** 1 hour | **Impact:** HIGH (production readiness)

- [ ] **[HIGH]** Fix test helper usage
  - [ ] Replace custom `asError()` with `errors.As()` from stdlib
  - [ ] Update test at line 126 in `download_test.go`
  - **Effort:** 30 minutes | **Impact:** MEDIUM

- [ ] **[HIGH]** Add archive extraction tests (0% → 90% coverage)
  - [ ] Test successful tar.gz extraction
  - [ ] Test extraction with nested directories
  - [ ] Test extraction failure (corrupted archive)
  - [ ] Test binary not found in archive
  - [ ] Test multiple executables in archive
  - **Effort:** 2-3 hours | **Impact:** HIGH

### 1.4 Provider Proto Critical Fixes

- [ ] **[CRITICAL]** Rewrite contract tests with real gRPC integration
  - [ ] Add helper to start real gRPC test server
  - [ ] Test `Init`, `Fetch`, `Info`, `Health`, `Shutdown` methods
  - [ ] Test error handling with gRPC status codes
  - [ ] Validate data serialization round-trips (Struct ↔ map[string]any)
  - [ ] Test lifecycle ordering (Init before Fetch)
  - **Effort:** 4-8 hours | **Impact:** HIGH

- [ ] **[HIGH]** Fix README code example
  - [ ] Correct enum value: `HealthResponse_OK` → `HealthResponse_STATUS_OK`
  - [ ] Verify all code examples compile
  - **Effort:** 5 minutes | **Impact:** MEDIUM

### 1.5 Monorepo Governance Critical Fixes

- [ ] **[CRITICAL]** Add provider-downloader to Makefile test targets
  - [ ] Update test loop in lines 52-56
  - [ ] Update test-race loop in lines 63-66
  - [ ] Update lint loop
  - **Effort:** 5 minutes | **Impact:** HIGH

- [ ] **[CRITICAL]** Standardize Go version across all CI workflows
  - [ ] Update `compiler-ci.yml` to Go 1.25.3
  - [ ] Update `parser-ci.yml` to Go 1.25.3
  - [ ] Verify `cli-ci.yml` is already 1.25.3
  - **Effort:** 5 minutes | **Impact:** HIGH

- [ ] **[CRITICAL]** Add `.golangci.yml` linting config
  - [ ] Create `.golangci.yml` at repo root with standard linters
  - [ ] Update Makefile lint target to use config
  - [ ] Update all 3 CI workflows to pin golangci-lint version
  - **Effort:** 30 minutes | **Impact:** HIGH

**Phase 1 Checkpoint:** ✅ Critical bugs fixed, security validated, test gaps closed

---

## Phase 2: CLI Modernization (Week 3-4)

**Goal:** Migrate CLI to Cobra framework and improve user experience

### 2.1 Cobra Migration

- [ ] **Add Cobra framework**
  - [ ] Add `github.com/spf13/cobra` dependency
  - [ ] Create root command in `cmd/nomos/root.go`
  - [ ] Define global flags (--verbose, --color)
  - **Effort:** 2 hours | **Impact:** VERY HIGH

- [ ] **Migrate core commands**
  - [ ] Migrate `help` command (easiest, test pattern)
  - [ ] Migrate `build` command to `cmd/nomos/build.go`
  - [ ] Migrate `init` command to `cmd/nomos/init.go`
  - [ ] Update flag parsing to use Cobra's flag sets
  - **Effort:** 6-8 hours | **Impact:** VERY HIGH

- [ ] **Add shell completion support**
  - [ ] Implement completion generation command
  - [ ] Add Fish completion
  - [ ] Add Bash completion
  - [ ] Add Zsh completion
  - [ ] Document completion installation in README
  - **Effort:** 2 hours | **Impact:** HIGH

- [ ] **Add version command**
  - [ ] Create `cmd/nomos/version.go`
  - [ ] Display version, commit hash, build date
  - [ ] Add `--version` flag to root command
  - **Effort:** 30 minutes | **Impact:** MEDIUM

- [ ] **Update tests for Cobra**
  - [ ] Refactor integration tests to use Cobra command testing patterns
  - [ ] Keep same test fixtures
  - [ ] Verify all integration tests pass
  - **Effort:** 4 hours | **Impact:** HIGH

### 2.2 CLI Code Quality Improvements

- [ ] **Remove `os.Exit()` from command handlers**
  - [ ] Refactor `cmd/nomos/build.go` (lines 23, 32)
  - [ ] Refactor `cmd/nomos/main.go` (line 55)
  - [ ] Return errors instead of calling `os.Exit()`
  - [ ] Handle all exits in `main()` function
  - **Effort:** 2 hours | **Impact:** HIGH (testability)

- [ ] **Implement structured result from `internal/initcmd`**
  - [ ] Create `InitResult` struct with Installed/Skipped providers
  - [ ] Refactor `internal/initcmd/init.go` to return result instead of printing
  - [ ] Move output formatting to command handler
  - [ ] Add tests for result formatting
  - **Effort:** 3 hours | **Impact:** HIGH

- [ ] **Consolidate test binary building**
  - [ ] Create shared test fixture builder in `test/fixtures.go`
  - [ ] Use `sync.Once` for single binary build per package
  - [ ] Update all integration test files to use shared builder
  - **Effort:** 1 hour | **Impact:** MEDIUM

### 2.3 CLI UX Enhancements

- [ ] **Implement `--color` flag support**
  - [ ] Add `--color={auto,always,never}` flag
  - [ ] Update `internal/diagnostics/` to use color setting
  - [ ] Test color output with different settings
  - **Effort:** 2 hours | **Impact:** HIGH

- [ ] **Add table output for `init` results**
  - [ ] Implement table formatter using `github.com/olekukonko/tablewriter`
  - [ ] Display provider alias, type, version, status
  - [ ] Add `--json` flag for machine-readable output
  - **Effort:** 4 hours | **Impact:** HIGH

- [ ] **Implement progress indicators for downloads**
  - [ ] Add spinner/progress bar for provider downloads
  - [ ] Use `github.com/briandowns/spinner` or similar
  - [ ] Show progress during `nomos init`
  - **Effort:** 4 hours | **Impact:** HIGH

- [ ] **Better error messages with suggestions**
  - [ ] Add "Did you mean...?" for typos
  - [ ] Suggest `nomos init` for missing provider errors
  - [ ] Show error context with code snippets
  - [ ] Add validation summary (e.g., "3 errors, 2 warnings")
  - **Effort:** 5 hours | **Impact:** HIGH

- [ ] **Add `--quiet` flag**
  - [ ] Suppress non-error output
  - [ ] Document in help text
  - **Effort:** 1 hour | **Impact:** MEDIUM

### 2.4 New Commands (Optional Enhancements)

- [ ] **Add `nomos validate` command** (syntax check only)
  - [ ] Parse files without compilation
  - [ ] Report syntax errors only
  - **Effort:** 3 hours | **Impact:** MEDIUM

- [ ] **Add `nomos format` command** (format .csl files)
  - [ ] Implement basic formatter for Nomos syntax
  - [ ] Add `--check` flag for CI usage
  - **Effort:** 8 hours | **Impact:** LOW

- [ ] **Add `nomos providers list` command**
  - [ ] List installed providers from lockfile
  - [ ] Show version, location, status
  - **Effort:** 2 hours | **Impact:** MEDIUM

**Phase 2 Checkpoint:** ✅ Modern CLI with Cobra, shell completions, improved UX

---

## Phase 3: Testing & Quality (Week 5-6)

**Goal:** Comprehensive test coverage and quality improvements

### 3.1 Parser Testing Improvements

- [ ] **Improve parser test coverage (44% → 75%)**
  - [ ] Test `parseSourceDecl` validation paths
  - [ ] Test `parseImportStmt` error handling
  - [ ] Test `parseReferenceStmt` rejection
  - [ ] Add scanner edge case tests (`GetIndentLevel`, `SkipToNextLine`)
  - [ ] Test error recovery paths
  - **Effort:** 3-4 days | **Impact:** HIGH

- [ ] **Refactor parameter passing in parser**
  - [ ] Store `sourceText` in Parser struct
  - [ ] Add `startPos` field to track statement start
  - [ ] Reduce parameter count from 4 to 2 in parsing functions
  - **Effort:** 1 day | **Impact:** MEDIUM (maintainability)

- [ ] **Improve test organization**
  - [ ] Move `nested_maps_test.go` to `test/` directory
  - [ ] Standardize on `parser_test` package
  - [ ] Group related tests in same file
  - **Effort:** 1 hour | **Impact:** LOW

### 3.2 Compiler Testing Improvements

- [ ] **Consolidate test infrastructure**
  - [ ] Create `test/fakes/provider_registry.go` with configurable behavior
  - [ ] Migrate 6 duplicate provider registry implementations
  - [ ] Update all test files to use shared registry
  - **Effort:** 1 day | **Impact:** HIGH

- [ ] **Add E2E smoke test**
  - [ ] Create `test/e2e/smoke_test.go` at repo root
  - [ ] Test full CLI → Compiler → Parser → Provider pipeline
  - [ ] Verify lockfile creation, snapshot output, determinism
  - **Effort:** 2 hours | **Impact:** HIGH

- [ ] **Complete skipped tests**
  - [ ] Implement import cycle detection test (`import_test.go:66`)
  - [ ] Implement cycle detection graph builder test
  - [ ] Implement network timeout integration test
  - [ ] Implement provider caching integration test
  - **Effort:** 1-2 weeks (depends on features) | **Impact:** MEDIUM

### 3.3 Provider Downloader Testing Improvements

- [ ] **Implement basic caching**
  - [ ] Add `cacheDir` field to `Client`
  - [ ] Implement cache lookup before download
  - [ ] Cache successful downloads by checksum
  - [ ] Add tests for cache hit/miss
  - [ ] Update documentation
  - **Effort:** 4-6 hours | **Impact:** HIGH (performance)

- [ ] **Refactor archive extraction**
  - [ ] Create `internal/archive/` package
  - [ ] Define `Extractor` interface
  - [ ] Implement `TarGzExtractor` and `ZipExtractor`
  - [ ] Add factory function `GetExtractor(filename)`
  - [ ] Update tests to test extractors independently
  - **Effort:** 3-4 hours | **Impact:** MEDIUM (maintainability)

- [ ] **Implement zip extraction**
  - [ ] Create `internal/archive/zip.go`
  - [ ] Test zip extraction
  - **Effort:** 2 hours | **Impact:** LOW

- [ ] **Add integration test suite**
  - [ ] Create `integration_test.go`
  - [ ] Test full flow: Resolve → Download → Install
  - [ ] Test multiple providers in sequence
  - [ ] Test concurrent downloads with race detection
  - **Effort:** 2 hours | **Impact:** MEDIUM

### 3.4 Monorepo Testing Improvements

- [ ] **Create provider-downloader CI workflow**
  - [ ] Create `.github/workflows/provider-downloader-ci.yml`
  - [ ] Add test, lint, race detection jobs
  - [ ] Use Go 1.25.3
  - **Effort:** 20 minutes | **Impact:** HIGH

- [ ] **Standardize integration test layout**
  - [ ] Add `//go:build integration` tag to ~20 files
  - [ ] Update Makefile with `test-unit`, `test-integration`, `test-network` targets
  - [ ] Document testing conventions in `CONTRIBUTING.md`
  - **Effort:** 1-2 hours | **Impact:** MEDIUM

- [ ] **Enforce commit message format in CI**
  - [ ] Create `.github/workflows/pr-validation.yml`
  - [ ] Validate Conventional Commits + gitmoji format
  - [ ] Add git hook option (lefthook)
  - **Effort:** 1 hour | **Impact:** MEDIUM

**Phase 3 Checkpoint:** ✅ Comprehensive test coverage, clean test infrastructure

---

## Phase 4: Compiler Refactoring (Week 7-8)

**Goal:** Simplify architecture and eliminate technical debt

### 4.1 Eliminate Adapter Pattern Duplication

- [ ] **Extract shared interfaces to `internal/core`**
  - [ ] Create `internal/core/provider.go` with shared Provider interface
  - [ ] Move `ProviderInitOptions` to core
  - [ ] Update imports across compiler
  - **Effort:** 4 hours | **Impact:** HIGH

- [ ] **Remove adapter layer**
  - [ ] Delete `imports_adapters.go` (~100 LOC of boilerplate)
  - [ ] Use embedding instead of wrapping
  - [ ] Update compiler to use shared interfaces directly
  - **Effort:** 1 day | **Impact:** HIGH (maintainability)

- [ ] **Consolidate provider interfaces**
  - [ ] Define clear interface hierarchy:
    - `Provider` (basic Fetch)
    - `InitializableProvider` (adds Init)
    - `ManagedProvider` (adds lifecycle methods)
  - [ ] Update 27 provider struct types to use hierarchy
  - **Effort:** 1 day | **Impact:** MEDIUM

### 4.2 Simplify Compilation Flow

- [ ] **Refactor `compiler.go` into phases**
  - [ ] Extract `discoverAndParseFiles()` function
  - [ ] Extract `initializeProvidersFromSources()` function
  - [ ] Extract `resolveImportsAndMerge()` function
  - [ ] Extract `validateSemantics()` function
  - [ ] Extract `resolveReferences()` function
  - [ ] Extract `buildSnapshot()` function
  - **Effort:** 2 days | **Impact:** HIGH (readability, testability)

- [ ] **Fix implicit control flow**
  - [ ] Replace `nil` sentinel value in `resolveFileImports()`
  - [ ] Return explicit `ImportResolutionResult` struct
  - [ ] Update call sites
  - **Effort:** 2 hours | **Impact:** MEDIUM

### 4.3 Improve Provider Lifecycle Management

- [ ] **Implement graceful provider shutdown**
  - [ ] Add timeout-based graceful shutdown in `manager.go`
  - [ ] Send shutdown RPC with timeout
  - [ ] Fallback to `Kill()` after timeout
  - [ ] Test graceful and forced shutdown
  - **Effort:** 4-6 hours | **Impact:** MEDIUM

- [ ] **Move Manager to internal package**
  - [ ] Create `internal/providers/manager.go`
  - [ ] Move `Manager` type from root package
  - [ ] Update imports
  - **Effort:** 1 hour | **Impact:** LOW (organization)

### 4.4 Error Collection Enhancement

- [ ] **Implement multi-error collection**
  - [ ] Create `CompilationResult` struct with Errors/Warnings
  - [ ] Create `ErrorCollector` to accumulate errors
  - [ ] Update compilation phases to collect errors instead of stopping
  - [ ] Return aggregated errors
  - **Effort:** 2-3 days | **Impact:** HIGH (UX)

### 4.5 Improve Internal Package Structure

- [ ] **Reorganize internal packages**
  - [ ] Create `internal/core/` for shared types
  - [ ] Create `internal/pipeline/` for compilation stages
  - [ ] Create `internal/providers/` for provider management
  - [ ] Migrate code to new structure
  - [ ] Update imports and tests
  - **Effort:** 1 week | **Impact:** MEDIUM (long-term maintainability)

**Phase 4 Checkpoint:** ✅ Clean architecture, maintainable codebase, better error handling

---

## Phase 5: Infrastructure & Polish (Week 9-10)

**Goal:** Improve development experience and infrastructure

### 5.1 Makefile Enhancements

- [ ] **Dynamic module discovery**
  - [ ] Add module discovery using `find` commands
  - [ ] Replace hardcoded module lists
  - [ ] Prevent future oversights
  - **Effort:** 30 minutes | **Impact:** HIGH

- [ ] **Add missing Makefile targets**
  - [ ] Add `test-integration` target
  - [ ] Add `test-coverage` target with threshold
  - [ ] Add `fmt` target for formatting
  - [ ] Add `mod-tidy` target
  - [ ] Add `install` target for local CLI installation
  - **Effort:** 1 hour | **Impact:** MEDIUM

- [ ] **Fix lint target**
  - [ ] Remove `|| true` that swallows errors
  - [ ] Reference `.golangci.yml` config
  - **Effort:** 5 minutes | **Impact:** MEDIUM

### 5.2 Development Tooling

- [ ] **Add `.editorconfig`**
  - [ ] Create `.editorconfig` at repo root
  - [ ] Define Go, YAML, Markdown formatting rules
  - **Effort:** 5 minutes | **Impact:** MEDIUM

- [ ] **Add pre-commit hooks (optional)**
  - [ ] Create `.lefthook.yml` with fmt, lint, mod-tidy hooks
  - [ ] Document setup in `CONTRIBUTING.md`
  - **Effort:** 1 hour | **Impact:** LOW (optional)

- [ ] **Add watch mode helper (optional)**
  - [ ] Create `.air.toml` for auto-rebuild
  - [ ] Add `watch` target to Makefile
  - **Effort:** 30 minutes | **Impact:** LOW (optional)

### 5.3 Parser Optimizations

- [ ] **Optimize scanner performance**
  - [ ] Replace save/restore with proper lookahead buffer
  - [ ] Add token buffering for `PeekToken()`
  - [ ] Profile allocation hot spots
  - [ ] Consider reusing AST nodes via `sync.Pool`
  - **Effort:** 2 days | **Impact:** MEDIUM (10-20% perf improvement)

- [ ] **Extract validation helpers**
  - [ ] Create `expectColonAfterKeyword()` helper
  - [ ] Reduce duplication in parser validation
  - **Effort:** 2 hours | **Impact:** LOW

### 5.4 Provider Downloader Enhancements

- [ ] **Replace custom string utilities**
  - [ ] Replace `contains()` with `strings.Contains()`
  - [ ] Remove `findSubstring()` function
  - **Effort:** 15 minutes | **Impact:** LOW

- [ ] **Add progress reporting callback**
  - [ ] Add `ProgressCallback` to `ClientOptions`
  - [ ] Implement progress tracking during download
  - [ ] Use in CLI for better UX
  - **Effort:** 2 hours | **Impact:** LOW

- [ ] **Make HTTP timeout configurable**
  - [ ] Add `HTTPTimeout` to `ClientOptions`
  - [ ] Default to 30s, allow override
  - **Effort:** 30 minutes | **Impact:** LOW

### 5.5 Provider Proto Enhancements

- [ ] **Add error documentation to proto comments**
  - [ ] Document gRPC status codes in each RPC method
  - [ ] Ensure docs appear in generated Go code
  - **Effort:** 30 minutes | **Impact:** MEDIUM (developer experience)

- [ ] **Add reserved fields to proto messages**
  - [ ] Add `reserved 4 to 10;` to all messages
  - [ ] Prevent accidental field number reuse
  - **Effort:** 10 minutes | **Impact:** LOW (future-proofing)

- [ ] **Add `STATUS_STARTING` enum value (optional)**
  - [ ] Add to `HealthResponse.Status`
  - [ ] Document use case for long initialization
  - **Effort:** 15 minutes | **Impact:** LOW

**Phase 5 Checkpoint:** ✅ Improved development experience, optimized performance

---

## Phase 6: Documentation & Finalization (Week 11-12)

**Goal:** Comprehensive documentation and final polish

### 6.1 Documentation Updates

- [ ] **Update README files**
  - [ ] Update CLI README with Cobra commands
  - [ ] Update compiler README with external provider usage
  - [ ] Update parser README with error handling info
  - [ ] Update provider-downloader README (clarify caching status)
  - [ ] Update provider-proto README (fix code examples)
  - **Effort:** 3 hours | **Impact:** HIGH

- [ ] **Create `docs/CODING_STANDARDS.md`**
  - [ ] Document error handling patterns
  - [ ] Document naming conventions
  - [ ] Document testing guidelines
  - [ ] Link from `CONTRIBUTING.md`
  - **Effort:** 1 hour | **Impact:** MEDIUM

- [ ] **Update `CONTRIBUTING.md`**
  - [ ] Add troubleshooting section
  - [ ] Document testing conventions
  - [ ] Document local provider setup
  - [ ] Update with new Makefile targets
  - **Effort:** 30 minutes | **Impact:** MEDIUM

- [ ] **Create testing guide**
  - [ ] Create `docs/TESTING_GUIDE.md`
  - [ ] Explain test organization
  - [ ] Document how to use fakes and test utilities
  - [ ] Show examples of integration vs. unit tests
  - **Effort:** 1 hour | **Impact:** MEDIUM

- [ ] **Update architecture docs**
  - [ ] Update `nomos-external-providers-feature-breakdown.md` status
  - [ ] Document new compiler architecture
  - [ ] Add provider lifecycle diagrams
  - **Effort:** 2 hours | **Impact:** MEDIUM

### 6.2 CHANGELOG Updates

- [ ] **Update all module CHANGELOGs**
  - [ ] Document breaking changes (Cobra migration, API changes)
  - [ ] Document new features (caching, shell completions)
  - [ ] Document bug fixes (context propagation, checksum validation)
  - [ ] Update comparison links
  - **Effort:** 2 hours | **Impact:** HIGH

- [ ] **Tag provider-downloader v0.1.0**
  - [ ] Review CHANGELOG
  - [ ] Create annotated tag
  - [ ] Push tag
  - **Effort:** 15 minutes | **Impact:** MEDIUM

### 6.3 CI/CD Enhancements (Optional)

- [ ] **Add release automation**
  - [ ] Create `.github/workflows/release.yml`
  - [ ] Extract CHANGELOG sections automatically
  - [ ] Create GitHub Releases on tag push
  - **Effort:** 1 hour | **Impact:** MEDIUM

- [ ] **Add PR size check**
  - [ ] Create `.github/workflows/pr-size.yml`
  - [ ] Warn on PRs >500 lines, fail on >1000
  - **Effort:** 15 minutes | **Impact:** LOW

- [ ] **Add dependency vulnerability scanning**
  - [ ] Add `govulncheck` to CI workflows
  - [ ] Run on schedule and PR
  - **Effort:** 30 minutes | **Impact:** MEDIUM

### 6.4 Final Cleanup

- [ ] **Remove obsolete TODOs**
  - [ ] Review all TODO comments in codebase
  - [ ] Address or create GitHub issues
  - [ ] Remove outdated TODOs
  - **Effort:** 1 hour | **Impact:** LOW

- [ ] **Fix YAML/HCL serialization stubs**
  - [ ] Either implement or remove from CLI help
  - [ ] Update documentation
  - **Effort:** 1 hour (removal) or 8 hours (implementation) | **Impact:** MEDIUM

- [ ] **API cleanup**
  - [ ] Remove unused `ParserOption` pattern or implement options
  - [ ] Mark `ReferenceStmt` as deprecated in parser
  - [ ] Plan removal in next major version
  - **Effort:** 1 hour | **Impact:** LOW

**Phase 6 Checkpoint:** ✅ Comprehensive documentation, production-ready codebase

---

## Success Metrics

### Code Quality Metrics

- [ ] **Test Coverage**
  - Parser: 44% → 75% ✅
  - Parser error handling: 0% → 80% ✅
  - Compiler: 50% → 70% ✅
  - Compiler imports: 21% → 75% ✅
  - Provider downloader: 81% → 90% ✅

- [ ] **Linting**
  - Zero golangci-lint errors ✅
  - All code passes `gofmt` ✅
  - Consistent with `.golangci.yml` ✅

- [ ] **CI/CD**
  - All workflows pass ✅
  - All modules have CI coverage ✅
  - Go version consistent (1.25.3) ✅

### User Experience Metrics

- [ ] **CLI UX**
  - Shell completions available (Fish, Bash, Zsh) ✅
  - Color output supported ✅
  - Table output for `init` results ✅
  - Progress indicators for downloads ✅
  - Better error messages with suggestions ✅

- [ ] **Developer Experience**
  - Clear test organization ✅
  - Comprehensive documentation ✅
  - Onboarding time <5 minutes ✅
  - All Makefile targets work ✅

### Architecture Metrics

- [ ] **Compiler Simplification**
  - Adapter count: 6 → 0 ✅
  - Compilation flow: complex → linear phases ✅
  - Error collection: single → multi-error ✅

- [ ] **Code Organization**
  - Clear package boundaries ✅
  - Proper use of internal packages ✅
  - Minimal duplication ✅

---

## Appendix: Detailed Findings

### A.1 Parser Module Detailed Analysis

**File-by-File Issues:**
- `errors.go` (210 lines): 0% coverage on error formatting
- `parser.go` (633 lines): 71.3% coverage, parameter passing issues
- `test/debug_test.go`: DELETE - throwaway debug code
- `test/scanner_debug_test.go`: DELETE - debug code
- `parser_bench_test.go`: BROKEN - uses deprecated syntax
- `nested_maps_test.go`: MOVE to `test/` directory

**Critical Functions Needing Tests:**
- `FormatParseError()` (errors.go:117) - 0% coverage
- `generateSnippet()` (errors.go:163) - 0% coverage
- `parseSourceDecl()` (parser.go) - validation paths untested
- `parseImportStmt()` (parser.go) - 0% coverage
- `parseReferenceStmt()` (parser.go) - rejection logic untested

### A.2 Compiler Module Detailed Analysis

**Adapter Files to Refactor:**
- `imports_adapters.go` (6 adapter types, ~100 LOC)
- Duplicated interfaces in `compiler.Provider` and `imports.Provider`
- Duplicated `ProviderInitOptions` in 2 locations

**Context Propagation Bugs:**
- `provider.go:106` - hardcoded `context.Background()`
- `provider_type_registry.go:105` - hardcoded `context.Background()`

**Test Infrastructure to Consolidate:**
- `compiler_test.go` - `mockProviderRegistry`
- `resolver_integration_test.go` - `fakeProviderRegistry`
- `test/integration_network_test.go` - `integrationProviderRegistry`
- `test/concurrency_test.go` - `concurrencyTestRegistry`
- `test/bench/compiler_bench_test.go` - `benchProviderRegistry`
- `internal/resolver/resolver_test.go` - `fakeProviderRegistry`

### A.3 CLI Module Detailed Analysis

**Files with `os.Exit()` calls:**
- `cmd/nomos/build.go` (lines 23, 32)
- `cmd/nomos/main.go` (line 55)

**Files needing refactoring:**
- `internal/initcmd/init.go` - direct `fmt.Println()` calls
- `internal/flags/flags.go` - duplicate logic with `options.go`
- `internal/initcmd/provider_utils.go` - manual string parsing

**Missing features:**
- No Cobra framework
- No shell completions
- No `--color` support (partially implemented)
- No table output for results
- No progress indicators

### A.4 Provider Downloader Detailed Analysis

**Debug logging locations (download.go):**
- Lines: 68, 71, 80, 122, 131, 190-194, 222, 224, 231, 236, 241, 252, 255, 270, 274, 277, 291

**Missing implementations:**
- Caching (mentioned in docs, not implemented)
- Zip extraction (marked as TODO)
- Checksum file parsing
- Progress reporting

**Test gaps:**
- Archive extraction (0% coverage)
- Rate limiting error path
- GitHub token authentication verification

### A.5 Provider Proto Detailed Analysis

**Trivial tests to replace (contract_test.go):**
- `TestInitRequest_MessageStructure` - tests struct assignment
- `TestMockProvider_FetchCall` - tests mock without gRPC

**Missing contract tests:**
- Real gRPC integration tests
- Error status code validation
- Data serialization round-trips
- Lifecycle ordering validation
- Concurrent request handling

### A.6 Monorepo Governance Detailed Analysis

**Makefile issues:**
- Lines 52-56: Missing provider-downloader in test loop
- Lines 63-66: Missing provider-downloader in test-race loop
- Line 81: `|| true` swallows lint errors
- Missing targets: `test-integration`, `test-coverage`, `fmt`, `mod-tidy`, `install`

**CI workflow issues:**
- No workflow for provider-downloader
- Go version inconsistency (1.25.3 vs 1.22)
- Missing: vulnerability scanning, commit message validation, release automation

**Tooling gaps:**
- No `.golangci.yml` config
- No `.editorconfig`
- No pre-commit hooks
- No automated formatting checks

---

## Notes on Execution

### Parallelization Opportunities

These tasks can be worked on in parallel by different developers:
- Parser improvements (Phase 1.1, 3.1)
- Compiler refactoring (Phase 1.2, 4.x)
- CLI modernization (Phase 2.x)
- Provider modules (Phase 1.3, 1.4, 3.3)
- Infrastructure (Phase 5.x)

### Risk Mitigation

**Breaking Changes:**
- CLI Cobra migration: Keep backward compatibility where possible
- Compiler API changes: Document migration path
- Provider interface changes: Version appropriately

**Testing Strategy:**
- Run full integration tests after each phase
- Keep existing tests passing during refactoring
- Add new tests before changing implementation

**Rollback Plan:**
- Work on feature branches
- Merge to main after each phase completion
- Tag after successful phase completion

### Optional Enhancements

Low-priority tasks marked as "(optional)" can be deferred:
- Watch mode
- Pre-commit hooks
- PR size checks
- `nomos format` command
- Additional CLI commands

---

## Revision History

| Date | Version | Changes |
|------|---------|---------|
| 2025-12-25 | 1.0 | Initial plan created from agent analysis |

---

**Next Steps:**
1. Review this plan with team
2. Prioritize phases based on business needs
3. Assign tasks to developers
4. Create GitHub issues from checkboxes
5. Begin Phase 1 execution
