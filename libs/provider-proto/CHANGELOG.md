# Changelog

All notable changes to the Provider Proto module will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- [Build] Makefile target `generate-protoc` as fallback for protobuf code generation when buf is unavailable (#56)
- [Build] gRPC dependency added to go.mod to support generated service code (#56)
- [Docs] Protobuf code generation instructions in README covering both buf and protoc workflows (#56)
- Initial protobuf schema for Provider gRPC service contract
- Service definition with Init, Fetch, Info, Health, and Shutdown RPCs
- Request/response message types using `google.protobuf.Struct` for flexible data exchange
- HealthResponse.Status enum (UNKNOWN, OK, DEGRADED)
- Generated Go stubs for service and messages
- Buf configuration for code generation and linting
- Contract validation tests with mock provider implementation
- Comprehensive README with usage examples
- AGENTS.md with development workflow documentation
- Makefile for common development tasks

[Unreleased]: https://github.com/autonomous-bits/nomos/compare/HEAD...HEAD
