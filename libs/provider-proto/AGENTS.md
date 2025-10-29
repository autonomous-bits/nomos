# Provider Proto Library

## Purpose

This module contains the protobuf schema and generated Go stubs for the Nomos Provider gRPC contract. External provider implementations must use this contract to communicate with the Nomos compiler.

## Structure

```
libs/provider-proto/
├── go.mod                          # Module definition
├── AGENTS.md                       # This file
├── README.md                       # Public documentation
├── CHANGELOG.md                    # Module changelog
├── buf.yaml                        # Buf configuration
├── buf.gen.yaml                    # Buf code generation config
├── proto/
│   └── nomos/
│       └── provider/
│           └── v1/
│               └── provider.proto  # Service and message definitions
└── gen/                            # Generated Go code
    └── go/
        └── nomos/
            └── provider/
                └── v1/
                    └── provider.pb.go
                    └── provider_grpc.pb.go
```

## Development Workflow

### Prerequisites

- Go 1.22+
- Protocol Buffer Compiler (`protoc`) or Buf CLI (`buf`)
- protoc-gen-go and protoc-gen-go-grpc plugins

### Generating Code

Using Buf (recommended):
```bash
buf generate
```

Using protoc directly:
```bash
protoc \
  --go_out=gen/go \
  --go_opt=paths=source_relative \
  --go-grpc_out=gen/go \
  --go-grpc_opt=paths=source_relative \
  proto/nomos/provider/v1/provider.proto
```

### Running Tests

```bash
go test -v ./...
```

### Linting

```bash
buf lint
```

## Usage for Provider Authors

External provider repositories should:

1. Add this module as a dependency:
   ```bash
   go get github.com/autonomous-bits/nomos/libs/provider-proto@v0.1.0
   ```

2. Import the generated types:
   ```go
   import (
       pb "github.com/autonomous-bits/nomos/libs/provider-proto/gen/go/nomos/provider/v1"
       "google.golang.org/grpc"
   )
   ```

3. Implement the `ProviderServer` interface:
   ```go
   type myProvider struct {
       pb.UnimplementedProviderServer
   }
   
   func (p *myProvider) Init(ctx context.Context, req *pb.InitRequest) (*pb.InitResponse, error) {
       // Implementation
   }
   
   // Implement Fetch, Info, Health, Shutdown...
   ```

4. Register with gRPC server:
   ```go
   server := grpc.NewServer()
   pb.RegisterProviderServer(server, &myProvider{})
   ```

## Versioning

This module follows semantic versioning. Breaking changes to the protobuf schema will result in a new major version.

## Related Documentation

- Feature specification: `docs/architecture/nomos-external-providers-feature-breakdown.md`
- Provider authoring guide: TBD
