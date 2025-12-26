# Changelog

All notable changes to the Nomos parser library will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Performance
- **Scanner optimization**: Replaced save/restore mechanism in `GetIndentLevel()` and `PeekToken()` with direct string scanning
- Achieved 10-20% performance improvement across all benchmark suites
- Reduced memory allocations in token peeking operations

### Refactoring
- Extracted `expectColonAfterKeyword()` helper function to reduce code duplication in parser validation
- Improved code maintainability in keyword parsing logic

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

[Unreleased]: https://github.com/autonomous-bits/nomos/compare/libs/parser/v0.1.0...HEAD
[0.1.0]: https://github.com/autonomous-bits/nomos/releases/tag/libs/parser/v0.1.0
