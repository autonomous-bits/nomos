# Provider Proto

Protocol buffer definitions and generated Go code for the Nomos Provider gRPC contract.

## Overview

This module defines the gRPC service contract that all Nomos external providers must implement. Providers are separate executables that communicate with the Nomos compiler over gRPC to supply configuration data from various sources (files, APIs, databases, etc.).

## Usage

### For Provider Authors

Add this module as a dependency to your provider implementation:

```bash
go get github.com/autonomous-bits/nomos/libs/provider-proto@v0.1.0
```

Import and implement the `ProviderServer` interface:

```go
package main

import (
    "context"
    
    providerv1 "github.com/autonomous-bits/nomos/libs/provider-proto/gen/go/nomos/provider/v1"
    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
    "google.golang.org/protobuf/types/known/structpb"
)

type myProvider struct {
    providerv1.UnimplementedProviderServiceServer
    // your state here
}

func (p *myProvider) Init(ctx context.Context, req *providerv1.InitRequest) (*providerv1.InitResponse, error) {
    // Initialize provider with config from req.Config
    return &providerv1.InitResponse{}, nil
}

func (p *myProvider) Fetch(ctx context.Context, req *providerv1.FetchRequest) (*providerv1.FetchResponse, error) {
    // Fetch data at req.Path
    value, err := structpb.NewStruct(map[string]interface{}{
        "key": "value",
    })
    if err != nil {
        return nil, status.Error(codes.Internal, err.Error())
    }
    
    return &providerv1.FetchResponse{Value: value}, nil
}

func (p *myProvider) Info(ctx context.Context, req *providerv1.InfoRequest) (*providerv1.InfoResponse, error) {
    return &providerv1.InfoResponse{
        Alias:   "my-provider",
        Version: "1.0.0",
        Type:    "custom",
    }, nil
}

func (p *myProvider) Health(ctx context.Context, req *providerv1.HealthRequest) (*providerv1.HealthResponse, error) {
    return &providerv1.HealthResponse{
        Status:  providerv1.HealthResponse_STATUS_OK,
        Message: "provider is healthy",
    }, nil
}

func (p *myProvider) Shutdown(ctx context.Context, req *providerv1.ShutdownRequest) (*providerv1.ShutdownResponse, error) {
    // Cleanup resources
    return &providerv1.ShutdownResponse{}, nil
}

func main() {
    server := grpc.NewServer()
    providerv1.RegisterProviderServiceServer(server, &myProvider{})
    // ... start server
}
```

### For Nomos Compiler Integration

The Nomos compiler will:
1. Start provider executables as subprocesses
2. Establish gRPC connections
3. Call `Init` with provider-specific configuration
4. Call `Fetch` to retrieve configuration data
5. Call `Shutdown` when compilation completes

## Service Contract

### Init

Initializes the provider with configuration. Must be called before Fetch operations.

**Request:**
- `alias` (string): Provider instance name from source declaration
- `config` (Struct): Provider-specific configuration (free-form map)
- `source_file_path` (string): Absolute path to the .csl file declaring this provider

**Response:** Empty (success indicated by lack of error)

**Errors:**
- `InvalidArgument`: Invalid configuration
- `FailedPrecondition`: Provider cannot be initialized (e.g., missing dependencies)
- `Unavailable`: External resource unreachable

### Fetch

Retrieves data at the specified path.

**Request:**
- `path` (repeated string): Path segments identifying the data to fetch

**Response:**
- `value` (Struct): Structured data compatible with Nomos value types

**Errors:**
- `NotFound`: Path does not exist
- `InvalidArgument`: Invalid path format
- `PermissionDenied`: Access denied
- `DeadlineExceeded`: Fetch timed out

### Info

Returns provider metadata. Can be called at any time.

**Request:** Empty

**Response:**
- `alias` (string): Provider instance name
- `version` (string): Provider implementation version
- `type` (string): Provider type identifier

### Health

Checks provider operational status.

**Request:** Empty

**Response:**
- `status` (enum): UNKNOWN, OK, or DEGRADED
- `message` (string): Additional context

### Shutdown

Gracefully shuts down the provider. Best-effort; compiler may force termination.

**Request:** Empty

**Response:** Empty

## Development

### Regenerating Code

After modifying `proto/nomos/provider/v1/provider.proto`:

```bash
make generate
```

Or manually with buf:

```bash
buf generate
```

If buf is not available, use protoc directly:

```bash
make generate-protoc
```

This requires `protoc`, `protoc-gen-go`, and `protoc-gen-go-grpc` to be installed:

```bash
# Install protoc (example for Ubuntu/Debian)
sudo apt-get install protobuf-compiler

# Install Go plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### Running Tests

```bash
make test
```

### Linting Protobuf

```bash
make lint
```

## Versioning

This module follows semantic versioning. Breaking changes to the protobuf schema will result in a new major version.

Provider implementations should declare compatible versions in their releases.

## References

- Feature specification: [`docs/architecture/nomos-external-providers-feature-breakdown.md`](../../docs/architecture/nomos-external-providers-feature-breakdown.md)
- Protocol Buffers: https://protobuf.dev/
- gRPC Go: https://grpc.io/docs/languages/go/
- Buf: https://buf.build/

## License

See the root repository LICENSE file.
