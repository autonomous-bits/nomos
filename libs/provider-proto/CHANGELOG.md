# Changelog

All notable changes to the Provider Proto module will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.1] - 2025-11-02

### Fixed
- Include generated protobuf Go code (`gen/` directory) in repository - files were incorrectly excluded by `.gitignore`, causing module to be unusable

## [0.1.0] - 2025-11-02

Initial release of the Provider Proto module.

### Added
- Provider gRPC service contract with Init, Fetch, Info, Health, and Shutdown RPCs
- Request/response message types using `google.protobuf.Struct` for flexible data exchange
- HealthResponse.Status enum (UNKNOWN, OK, DEGRADED)
- Generated Go stubs for service and messages
- Buf configuration for code generation and linting
- Makefile with `generate-protoc` fallback target for protobuf generation
- Contract validation tests with mock provider implementation
- Comprehensive README and AGENTS.md documentation

[Unreleased]: https://github.com/autonomous-bits/nomos/compare/libs/provider-proto/v0.1.1...HEAD
[0.1.1]: https://github.com/autonomous-bits/nomos/compare/libs/provider-proto/v0.1.0...libs/provider-proto/v0.1.1
[0.1.0]: https://github.com/autonomous-bits/nomos/releases/tag/libs/provider-proto/v0.1.0
