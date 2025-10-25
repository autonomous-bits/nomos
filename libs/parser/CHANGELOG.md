# Changelog

All notable changes to the Nomos parser library will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0-beta] - 2025-10-25

### Added
- Parser API with `ParseFile(path string)` and `Parse(r io.Reader, filename string)` (#3, #8)
- Parser instance pattern with `NewParser(opts...)` for pooling and reuse (#8)
- Complete Nomos grammar support: `source`, `import`, `reference`, sections, mappings (#3)
- Stable AST types with source location tracking (file, line, column) (#3)
- Structured error types: `LexError`, `SyntaxError`, `IOError` with precise location info (#3)
- Error formatting with context snippets and caret position indicators (#3)
- Concurrency tests for 100 concurrent parses and 1MB files (#8)
- Integration tests for error handling and real-world scenarios (#3)
- Golden tests for AST deterministic JSON output and error messages (#3)
- Performance benchmarks for small, medium, and large files (~1MB) (#8)
- Workspace sync script (`tools/scripts/work-sync.sh`) for go.work management (#8)
- Makefile with comprehensive test, coverage, and benchmark targets (#3)

### Performance
- Parser is concurrency-safe with no package-level mutable state (#8)
- Benchmarked performance: ~1.1μs for small files, ~11ms for 1MB files (#8)
- Parallel benchmark demonstrates scalability (#8)

## Notes

This is the initial beta release of the Nomos parser library targeting production readiness.

### Added
- Parser validation for syntax errors (#TBD)
  - Validate keywords (`source`, `import`, `reference`) must be followed by `:`
  - Validate empty aliases are rejected in `source` declarations
  - Validate aliases are required for `import` and `reference` statements
  - Validate reference statements require both alias and path
  - Validate unterminated string literals are detected and rejected
  - Validate invalid characters in key identifiers (e.g., `@`, `!`, etc.)
- Comprehensive test fixtures covering all grammar constructs in `testdata/fixtures/` (#7)
  - `source_complete.csl` - Source declaration with multiple configuration keys
  - `import_with_path.csl` - Import statements with and without paths
  - `reference_dotted_path.csl` - Reference statements with dotted paths
  - `mapping_nested.csl` - Nested mapping structures
  - `complex_config.csl` - Complete configuration with all grammar constructs
  - `empty.csl` - Empty file edge case
  - `whitespace_only.csl` - Whitespace-only edge case
  - `unicode.csl` - Unicode character support test
  - `all_grammar.csl` - Comprehensive grammar coverage
- Negative test fixtures for error scenarios in `testdata/fixtures/negative/` (#7)
  - `source_missing_colon.csl` - Missing colon after keyword
  - `unterminated_string.csl` - Unterminated string literal
  - `import_no_alias.csl` - Incomplete import statement
  - `reference_no_alias.csl` - Incomplete reference statement
  - `invalid_indentation.csl` - Invalid indentation
  - `invalid_key_character.csl` - Invalid characters in keys
  - `empty_alias.csl` - Empty alias value
  - `duplicate_key.csl` - Duplicate key in mapping
- Golden tests for error scenarios (`test/golden_errors_test.go`) (#7)
  - Validates error messages match expected structure
  - Ensures error kind, line, column, and message are correctly reported
  - Generates golden files for error outputs in `testdata/golden/errors/`
- Makefile with test automation targets (#7)
  - `make test` - Run all parser tests
  - `make test-verbose` - Run tests with verbose output
  - `make test-race` - Run tests with race detector
  - `make test-coverage` - Generate coverage report (HTML and summary)
  - `make update-golden` - Update golden test files (with safety prompt)
  - `make bench` - Run benchmark tests
  - `make lint` - Run golangci-lint
  - `make clean` - Clean generated files
- CI/CD workflow (`.github/workflows/parser-ci.yml`) (#7)
  - Automated tests with race detection on every push/PR
  - Coverage reporting with 80% minimum threshold
  - golangci-lint integration with strict checks
  - Build verification
  - Go workspace verification and setup
  - Codecov integration for coverage tracking
- Development documentation in README (#7)
  - Running Tests section with all Make targets
  - Golden Tests section with update instructions
  - Test Organization structure diagram
  - Benchmarking guide
  - Linting setup and usage
  - CI/CD description
  - Local Development Workflow checklist
- Typed error model with `ParseError` struct containing filename, line, column, message, snippet, and error kind fields
- Error kind enumeration (`LexError`, `SyntaxError`, `IOError`) for programmatic error handling
- `FormatParseError` function that generates human-friendly error messages with:
  - Machine-parseable `file:line:col:` prefix
  - Context lines showing 1-3 lines around the error
  - Caret marker (`^`) pointing to exact error position
  - Rune-aware column counting for correct multi-byte UTF-8 character handling
- Comprehensive unit tests for error model including unicode/multi-byte character tests
- Integration tests demonstrating real-world error scenarios and CLI-like error formatting
- Documentation of error handling in README with examples for basic and programmatic error handling

### Changed
- Parser now returns `*ParseError` instead of generic `fmt.Errorf` errors
- All parse errors include precise source location information with typed fields
- Error messages are now deterministic and include source snippets when source text is available
- Test organization improved with dedicated directories for fixtures, golden files, and errors
- README enhanced with comprehensive testing and development guidelines
- Negative test fixtures updated to reflect new validation (#TBD)
  - 9 fixtures now properly trigger validation errors (empty_alias, import_no_alias, reference_no_alias, source_missing_colon, unterminated_string, incomplete_reference, invalid_key_character, missing_colon, missing_colon_after_keyword)
  - 5 fixtures remain as valid syntax or unimplementable validations (duplicate_key, incomplete_import, invalid_indentation, unknown_statement, unicode_context)
  - Improved test documentation explaining why certain files are skipped

## [0.1.0] - 2025-10-25

### Added
- [Parser][Tests] Comprehensive scanner unit tests covering tokenization, position tracking, and edge cases (#5)
- [Parser][Tests] Negative test cases for malformed input with error location validation (#5)
- [Parser][Tests] End-to-end integration test validating all grammar constructs (source, import, reference, sections, path tokenization) (#5)
- [Parser][Tests] Test fixtures in `testdata/fixtures/negative/` for error handling validation (#5)

## [1.0.0-beta] - 2025-10-25

### Added
- [Parser] Public API with `ParseFile` and `Parse` functions for parsing Nomos .csl files (closes #4)
- [Parser] Core AST types (`AST`, `SourceDecl`, `ImportStmt`, `ReferenceStmt`, `SectionDecl`) with JSON tags
- [Parser] `SourceSpan` type for precise source location tracking (filename, line, column) on all AST nodes
- [Parser] Parser instance type with `NewParser()` for pooling and reuse scenarios
- [Parser] Comprehensive test suite with unit tests, integration tests, and golden tests
- [Parser] Deterministic AST serialization to JSON for testing and tooling
- [Parser] Error messages with precise file location and context
- [Parser] Support for source declarations with alias, type, and configuration blocks
- [Parser] Support for import statements with optional path
- [Parser] Support for reference statements with dotted paths
- [Parser] Support for configuration sections with key-value pairs
- [Parser] README with usage examples and API documentation

[Unreleased]: https://github.com/autonomous-bits/nomos/compare/libs/parser/v1.0.0-beta...HEAD
[1.0.0-beta]: https://github.com/autonomous-bits/nomos/releases/tag/libs/parser/v1.0.0-beta
