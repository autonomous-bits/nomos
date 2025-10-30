# Provider Process Manager

Internal package for managing external Nomos provider subprocesses.

## Overview

This package provides infrastructure for starting, managing, and communicating with external provider executables via gRPC. It implements the subprocess lifecycle management required for the External Providers architecture (see `docs/architecture/nomos-external-providers-feature-breakdown.md`).

## Components

### Manager

Manages the lifecycle of provider subprocesses:
- Starts provider binaries on-demand
- Establishes gRPC connections
- Caches providers per alias (one subprocess per alias per build)
- Gracefully shuts down all processes

### Client

Implements `compiler.Provider` interface by delegating to gRPC:
- Translates `Init`, `Fetch` calls to gRPC RPCs
- Implements `ProviderWithInfo` for metadata
- Handles error translation from gRPC status codes

## Usage

```go
manager := providerproc.NewManager()
defer manager.Shutdown(context.Background())

opts := compiler.ProviderInitOptions{
    Alias:  "configs",
    Config: map[string]any{"directory": "./configs"},
}

provider, err := manager.GetProvider(ctx, "configs", binaryPath, opts)
if err != nil {
    return err
}

// Use provider normally
data, err := provider.Fetch(ctx, []string{"database", "prod"})
```

## Testing

The package includes comprehensive tests with a fake provider implementation:
- Unit tests for Client gRPC delegation
- Integration tests for Manager subprocess lifecycle
- Error handling and edge cases
- Concurrency safety (race detector clean)

Run tests:
```bash
go test ./internal/providerproc
go test -race ./internal/providerproc
```

## Architecture Notes

### Process Model
- **Per-alias**: One subprocess per provider alias per compilation run
- **Lazy start**: Processes are started only when first needed
- **Caching**: Subsequent GetProvider calls return the same client instance
- **Clean shutdown**: All processes are terminated on Manager.Shutdown

### Communication Protocol
Providers must:
1. Listen on a random available TCP port
2. Print `PROVIDER_PORT=<port>` to stdout on startup
3. Implement the `nomos.provider.v1.ProviderService` gRPC service
4. Respond to Health RPC for connection verification

### Error Handling
- Binary not found → immediate error
- Connection failures → error with context
- RPC errors → translated from gRPC status codes
- Subprocess crashes → stderr captured in error messages

### Security Considerations
- Only execute binaries from trusted locations
- Capture but don't log provider stderr by default
- Support for future permission checks (chmod validation)

## Future Enhancements

- Retry logic with exponential backoff for connection failures
- Configurable timeouts per RPC
- Enhanced stderr logging with verbosity control
- Permission validation before execution
- Support for TLS-secured gRPC connections
