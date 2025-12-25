# Changelog

All notable changes to the Provider Proto module will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Comprehensive gRPC integration tests with real client-server communication
- Test coverage for all RPC methods (Init, Fetch, Info, Health, Shutdown)
- Error handling tests with gRPC status codes (InvalidArgument, FailedPrecondition, NotFound)
- Data serialization round-trip tests (Struct ↔ map[string]interface{})
- Lifecycle ordering tests (Init before Fetch requirement)
- Context cancellation tests
- README code example compilation verification test

### Fixed
- README code example: corrected enum value `HealthResponse_OK` → `HealthResponse_STATUS_OK`

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
