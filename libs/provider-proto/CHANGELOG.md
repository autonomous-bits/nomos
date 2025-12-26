# Changelog

All notable changes to the Provider Proto module will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Comprehensive error code documentation in proto comments for all RPC methods
  - Init: InvalidArgument, FailedPrecondition, Unavailable, PermissionDenied
  - Fetch: NotFound, InvalidArgument, FailedPrecondition, PermissionDenied, DeadlineExceeded, Unavailable
  - Info, Health, Shutdown: Internal error handling guidance
  - Error documentation appears in generated Go code for provider developers
- Reserved field ranges (4-10 or 1-10) to all message definitions for future-proofing
  - Prevents accidental field number reuse in protocol evolution
  - Follows protobuf best practices for backward compatibility
- `STATUS_STARTING` enum value to HealthResponse.Status
  - Indicates provider is still initializing (useful for lengthy startup periods)
  - Allows compiler to distinguish between "not ready yet" and "unhealthy"
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
