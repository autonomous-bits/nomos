# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
