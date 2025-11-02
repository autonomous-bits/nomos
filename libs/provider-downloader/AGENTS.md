# Agent Instructions: libs/provider/downloader

## Module Purpose

This library provides reusable functionality for resolving, downloading, and installing provider binaries from GitHub Releases. It is consumed by `nomos init` and other tools managing provider lifecycles.

## Layout

```
downloader/
├── go.mod              # Module definition
├── README.md           # User documentation
├── CHANGELOG.md        # Version history
├── AGENTS.md           # This file
├── doc.go              # Package-level documentation
├── types.go            # Public types (ProviderSpec, AssetInfo, etc.)
├── client.go           # Client and public API methods
├── errors.go           # Typed errors
├── resolver.go         # GitHub Release asset resolver (internal)
├── download.go         # Streaming downloader with retry (internal)
├── checksum.go         # SHA256 verification helpers (internal)
├── install.go          # Atomic install logic (internal)
├── client_test.go      # Unit tests for client API
├── resolver_test.go    # Unit tests for resolver
├── download_test.go    # Unit tests for downloader
└── integration_test.go # Hermetic httptest-based integration tests
```

## Development Standards

- **Go version**: 1.25.3
- **TDD**: Write tests before implementation (Red → Green → Refactor)
- **No network calls in unit tests**: Use httptest for all GitHub API interactions
- **Testable design**: Accept interfaces, return concrete types
- **Documentation**: Every exported symbol must have GoDoc comments
- **Error handling**: Use typed errors and wrap with context

## Build and Test

```bash
# Run all tests
go test ./...

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific tests
go test -run TestClient_NewClient ./...

# Run with race detector
go test -race ./...

# Benchmarks
go test -bench=. -benchmem
```

## API Design Principles

1. **Context-aware**: All public methods accept context.Context as first parameter
2. **Options pattern**: Use functional options or struct options for configuration
3. **Typed errors**: Define sentinel errors for common failure modes
4. **Atomic operations**: Install operations must be atomic (download → verify → rename)
5. **Auto-detection**: Auto-detect OS/Arch when not specified in ProviderSpec
6. **Retry logic**: Handle transient failures with exponential backoff

## Implementation Guidelines

### Public API Surface (keep minimal)

- `NewClient(ctx, opts) *Client`
- `(*Client) ResolveAsset(ctx, spec) (AssetInfo, error)`
- `(*Client) DownloadAndInstall(ctx, asset, dest) (InstallResult, error)`

### Internal Helpers (not exported)

- Asset name inference based on common patterns
- GitHub API client wrapper
- Checksum verification
- File I/O with atomic rename

### Error Types

Define typed errors for:
- Asset not found
- Checksum mismatch
- Invalid spec
- Rate limit exceeded
- Network failure

### Testing Strategy

1. **Unit tests**: Test each function in isolation with mocks/fakes
2. **Integration tests**: Use httptest to emulate GitHub API responses
3. **Edge cases**: Test missing fields, invalid checksums, network errors
4. **Benchmarks**: Measure performance of resolution and download logic

## Dependencies

- Standard library only for v1
- No external dependencies unless absolutely necessary
- If adding dependencies, justify in PR description

## Related Modules

- `libs/compiler`: Consumer of this library
- `apps/command-line`: Uses this library via `nomos init`
- `libs/provider-proto`: Provider gRPC protocol definition

## PRD Reference

Feature: #68 — Init: download providers from GitHub Releases (remove --from)  
User Story: #69 — provider/downloader: public API and scaffolding

## Questions?

- Check feature docs: `docs/architecture/nomos-external-providers-feature-breakdown.md`
- Check provider authoring guide: `docs/guides/provider-authoring-guide.md`
- Check Go monorepo structure: `docs/architecture/go-monorepo-structure.md`
