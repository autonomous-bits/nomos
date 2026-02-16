# Changelog

All notable changes to the Provider Proto module will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.1] - 2026-02-16

### Changed
- upgraded Go version to 1.26.0 to fix TLS vulnerabilities
- Docs: update AGENTS.md with new agent roles

## [0.2.0] - 2025-12-26

### Added
- Comprehensive error code documentation in proto comments for all RPC methods
  - Init: InvalidArgument, FailedPrecondition, Unavailable, PermissionDenied
  - Fetch: NotFound, InvalidArgument, FailedPrecondition, PermissionDenied, DeadlineExceeded, Unavailable
  - Info, Health, Shutdown: Internal error handling guidance
  - Error documentation appears in generated Go code for provider developers
  - Improves developer experience when implementing providers
- Reserved field ranges to all 9 message definitions for future-proofing
  - Added `reserved 4 to 10;` (or `1 to 10;`) to prevent accidental field number reuse
  - Follows protobuf best practices for protocol evolution
  - Ensures backward compatibility in future versions
- `STATUS_STARTING` enum value to HealthResponse.Status
  - Indicates provider is still initializing (useful for lengthy startup periods)
  - Allows compiler to distinguish between "not ready yet" and "unhealthy"
  - Useful for providers with long initialization times
- Comprehensive gRPC integration tests with real client-server communication
  - Rewrote trivial contract tests with actual gRPC server and client
  - Test coverage for all RPC methods (Init, Fetch, Info, Health, Shutdown)
  - Error handling tests with gRPC status codes (InvalidArgument, FailedPrecondition, NotFound)
  - Data serialization round-trip tests (Struct ↔ map[string]interface{})
  - Lifecycle ordering tests (Init before Fetch requirement)
  - Context cancellation tests
  - Real end-to-end validation of protocol contract
- README code example compilation verification test
  - Ensures documentation examples compile and work correctly

### Fixed
- README code example: corrected enum value `HealthResponse_OK` → `HealthResponse_STATUS_OK`
  - Documentation now matches actual generated code
  - Verified all code examples compile

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

[Unreleased]: https://github.com/autonomous-bits/nomos/compare/libs/provider-proto/v0.2.0...HEAD
[0.2.0]: https://github.com/autonomous-bits/nomos/compare/libs/provider-proto/v0.1.1...libs/provider-proto/v0.2.0
[0.1.1]: https://github.com/autonomous-bits/nomos/compare/libs/provider-proto/v0.1.0...libs/provider-proto/v0.1.1
[0.1.0]: https://github.com/autonomous-bits/nomos/releases/tag/libs/provider-proto/v0.1.0
