# Changelog

All notable changes to the Nomos parser library will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- [Parser] Add list and nested list syntax support in CSL files
  - Use dash notation for block lists:
    - `servers:`
    - `  - web-01`
    - `  - web-02`
  - Use `[]` for empty lists: `tags: []`
  - Reference list items with index notation: `@alias:config.servers[0]`
- **YAML-style comment support**: Single-line `#` comments for inline documentation
  - Comments extend from `#` to end-of-line, supporting inline and full-line comments
  - Context-aware parsing: `#` preserved within quoted strings (`"api#key"` remains literal)
  - Zero breaking changes: Existing configurations without comments work unchanged
  - Performance validated: <5% overhead (typically <1%) across all benchmark suites
  - See [Comment Support documentation](README.md#comment-support) for usage examples

### Fixed
- [Parser] `SectionDecl` now uses `Value` field for inline scalar values instead of creating map entries with empty-string keys, enabling clean HCL/tfvars serialization

### Removed
- **BREAKING CHANGE: ReferenceStmt removed from AST** (User Story 1)
  - Top-level `reference:alias:path` syntax no longer supported
  - `ReferenceStmt` AST type removed from `pkg/ast` package
  - Parser now rejects top-level reference statements with `SyntaxError`
  - Error message includes migration guidance to inline reference syntax
  - Migration: Convert `reference:alias:path` to `key: @alias:path` (inline @ syntax)
  - Rationale: Top-level references were never used in practice and added unnecessary complexity
  - Test coverage: `test/deprecated_reference_test.go` validates rejection and error messages

### Added
- **FEATURE: @ reference syntax** (User Story 1)
  - New @ symbol prefix for inline references: `key: @alias:path.to.value`
  - Replaces legacy `reference:alias:path` keyword syntax
  - Benefits: Shorter, more intuitive syntax similar to modern languages
  - Parser automatically recognizes @ prefix and validates reference format
  - Full backward compatibility: All existing reference features work with @ syntax
  - Reference list items: `@alias:config.servers[0]`
  - Reference nested values: `@network:vpc.cidr`
  - Validation: Whitespace not allowed, double @@ rejected, empty paths rejected
  - See [Inline Reference Syntax documentation](README.md#inline-reference-syntax) for examples

### Changed
- **API Documentation Improvements**: Enhanced public API documentation for production readiness
  - Added comprehensive godoc for `Parser` struct explaining instance reuse and concurrency safety
  - Documented `Option` functional options pattern and its future extensibility design
  - Added "API Design Philosophy" section to README covering:
    - Functional options pattern rationale and future examples
    - Parser instance reuse patterns (single-use, reusable, pooling)
    - Public vs internal package separation
  - Clarified `ReferenceStmt` deprecation status with migration guidance
  - Fixed terminology inconsistency (ParserOption → Option)
- All exported symbols now have complete documentation

## [0.8.0] - 2025-12-26

### Added
- Comprehensive error handling test suite with 14 test functions and 40+ test cases
  - Test `FormatParseError()` with various error types (lexer, syntax, IO)
  - Test UTF-8 handling in `generateSnippet()` with multibyte characters
  - Test edge cases (empty source, out-of-bounds lines, missing newlines)
  - Test error unwrapping logic for nested errors
  - Coverage improved from 0% to 80%+ for error formatting code
- Source span accuracy tests for all expression types

### Changed
- Refactored parameter passing in Parser struct
  - Store `sourceText` in Parser struct instead of passing as parameter
  - Add `startPos` field to track statement start position
  - Reduce parameter count from 4 to 2 in parsing functions
  - Improves code readability and maintainability

### Fixed
- Fixed benchmark suite to use inline reference syntax
  - Removed deprecated `reference:base:config.database` syntax
  - All benchmarks now passing and accurate

### Performance
- **Scanner optimization**: Replaced save/restore mechanism in `GetIndentLevel()` and `PeekToken()` with direct string scanning
- Achieved 10-20% performance improvement across all benchmark suites (6-13% measured)
- Reduced memory allocations in token peeking operations

### Refactoring
- Extracted `expectColonAfterKeyword()` helper function to reduce code duplication
- Applied to `parseSourceDecl` and `parseImportStmt` for improved maintainability
- Removed debug test files (`test/debug_test.go`, `test/scanner_debug_test.go`)
- Moved `nested_maps_test.go` to `test/` directory for consistency

### Testing
- Test coverage improved from 44.2% to 86.9% (exceeded 75% goal by 11.9%)
- Added comprehensive validation path tests for `parseSourceDecl`
- Added error handling tests for `parseImportStmt`
- Added scanner edge case tests (`GetIndentLevel`, `SkipToNextLine`)
- Added error recovery path tests

## [0.1.0] - 2025-11-02

Initial release of the Nomos parser library.

### Added
- Inline reference syntax support: parse `reference:alias:path` in value positions
- `ReferenceExpr` AST node type with `Alias`, `Path`, and precise `SourceSpan` fields
- Comprehensive documentation with inline reference syntax guide and migration notes
- Documentation examples directory (`docs/examples/`) with reference usage patterns
- Automated documentation validation tests
- Source span accuracy tests for all expression types
- Golden tests for inline references (valid and malformed syntax)
- Complete Nomos grammar support: `source`, `import`, sections, mappings
- Structured error types (`LexError`, `SyntaxError`, `IOError`) with source locations
- Parser API: `ParseFile()` and `Parse()` with reusable parser instances
- Concurrency-safe implementation with no package-level mutable state

### Changed
- **BREAKING**: `SectionDecl.Entries` and `SourceDecl.Config` now use `map[string]Expr` instead of `map[string]string`
- **BREAKING**: Top-level `reference:` statements removed; use inline references in value positions
- Updated golden test files for new AST structure

### Fixed
- Section declaration parsing no longer drops first entry

[Unreleased]: https://github.com/autonomous-bits/nomos/compare/libs/parser/v0.8.0...HEAD
[0.8.0]: https://github.com/autonomous-bits/nomos/compare/libs/parser/v0.1.0...libs/parser/v0.8.0
[0.1.0]: https://github.com/autonomous-bits/nomos/releases/tag/libs/parser/v0.1.0
