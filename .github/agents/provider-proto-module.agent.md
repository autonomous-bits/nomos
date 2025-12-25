# Provider Proto Module Agent

## Purpose
Specialized agent for `libs/provider-proto` - handles gRPC protocol buffer contracts for external provider communication. This module defines the service contract that all Nomos external providers must implement to communicate with the Nomos compiler.

## Module Context
- **Path**: `libs/provider-proto`
- **Module Name**: `github.com/autonomous-bits/nomos/libs/provider-proto`
- **Responsibilities**: 
  - Define protobuf schema for Nomos Provider gRPC contract
  - Generate Go stubs for provider implementations
  - Maintain versioned provider service contract
  - Support external provider executable communication
- **Key Files**:
  - `proto/nomos/provider/v1/provider.proto` - Service and message definitions
  - `gen/go/nomos/provider/v1/provider.pb.go` - Generated protobuf types
  - `gen/go/nomos/provider/v1/provider_grpc.pb.go` - Generated gRPC service stubs
  - `buf.yaml` - Buf configuration for linting and breaking change detection
  - `buf.gen.yaml` - Buf code generation configuration
- **Build System**: Buf CLI (recommended) or protoc

## Delegation Instructions
For general Go questions, **consult go-expert.md**  
For gRPC/protobuf questions, **consult api-messaging-expert.md**  
For provider implementation patterns, **consult compiler-module.md**

## Provider Proto-Specific Patterns

### Protobuf Structure
The module follows a versioned API structure:
```
proto/
└── nomos/
    └── provider/
        └── v1/              # Version 1 of the provider contract
            └── provider.proto
```

Generated code mirrors this structure:
```
gen/
└── go/
    └── nomos/
        └── provider/
            └── v1/
                ├── provider.pb.go       # Message types
                └── provider_grpc.pb.go  # Service interfaces
```

### Service Contract

The provider service defines five RPC methods:

1. **Init** - Initialize provider with configuration
   - Called once before any Fetch operations
   - Receives provider alias, config struct, and source file path
   - Errors: InvalidArgument, FailedPrecondition, Unavailable

2. **Fetch** - Retrieve data at specified path
   - Primary data retrieval method
   - Receives path segments array
   - Returns structured data compatible with Nomos value types
   - Errors: NotFound, InvalidArgument, PermissionDenied, DeadlineExceeded

3. **Info** - Return provider metadata
   - Can be called at any time
   - Returns alias, version, and type identifier

4. **Health** - Check operational status
   - Returns status (UNKNOWN, OK, DEGRADED) and message

5. **Shutdown** - Graceful cleanup
   - Best-effort cleanup; compiler may force termination
   - Provider should release resources

### Contract Versioning
- Module follows **semantic versioning**
- Breaking changes to protobuf schema require new major version
- Use buf's breaking change detection: `buf breaking`
- Provider implementations must declare compatible version ranges
- Current version: v1 (stable)

### Code Generation

**Using Buf (recommended):**
```bash
buf generate
```

**Using protoc directly:**
```bash
protoc \
  --go_out=gen/go \
  --go_opt=paths=source_relative \
  --go-grpc_out=gen/go \
  --go-grpc_opt=paths=source_relative \
  proto/nomos/provider/v1/provider.proto
```

**Prerequisites:**
- Go 1.22+
- Buf CLI or Protocol Buffer Compiler (protoc)
- protoc-gen-go and protoc-gen-go-grpc plugins

**Installation:**
```bash
# Install Buf
go install github.com/bufbuild/buf/cmd/buf@latest

# Or install protoc plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### Build Tags
No special build tags required for this module. Generated code works across all platforms.

### Module Dependencies
- **Used by:**
  - `libs/compiler` - Manages provider lifecycles, calls gRPC methods
  - External provider implementations (separate executables)
- **Defines contracts for:**
  - File-based providers
  - API-based providers  
  - Database providers
  - Custom provider types

## Common Tasks

### 1. Updating Provider Protocol Contracts
When adding or modifying RPC methods or messages:

1. Edit `proto/nomos/provider/v1/provider.proto`
2. Run `buf lint` to check for style violations
3. Run `buf breaking --against '.git#branch=main'` to detect breaking changes
4. Regenerate code: `buf generate`
5. Update CHANGELOG.md with changes
6. Update README.md documentation for new methods/fields
7. Consider versioning impact (major vs minor change)

### 2. Adding New RPC Methods
Example workflow:
```protobuf
service ProviderService {
  rpc Init(InitRequest) returns (InitResponse);
  rpc Fetch(FetchRequest) returns (FetchResponse);
  // Add new method:
  rpc Validate(ValidateRequest) returns (ValidateResponse);
}

message ValidateRequest {
  // Define request fields
}

message ValidateResponse {
  // Define response fields
}
```

Then regenerate and update tests.

### 3. Versioning Protocol Changes
**Breaking changes** (require major version bump):
- Removing or renaming fields
- Changing field types
- Removing RPC methods
- Changing method signatures

**Non-breaking changes** (minor/patch):
- Adding new optional fields
- Adding new RPC methods
- Adding new enum values (with care)

### 4. Generating Go Code from Proto Files
Standard workflow:
```bash
# Lint before generating
buf lint

# Check for breaking changes
buf breaking --against '.git#branch=main'

# Generate
buf generate

# Run tests to verify generated code
go test -v ./...
```

### 5. Testing Contract Implementations
Use `contract_test.go` to verify provider implementations:
- Tests that generated code compiles
- Validates service interface completeness
- Ensures backward compatibility

## Nomos-Specific Details

### Provider Communication Model
Nomos uses a **subprocess + gRPC** architecture:
1. Compiler spawns provider executable as subprocess
2. Provider starts gRPC server on allocated port
3. Compiler establishes client connection
4. Init → Fetch (multiple) → Shutdown lifecycle
5. Provider process terminates

### Data Structure Compatibility
- Fetch responses use `google.protobuf.Struct`
- Must be compatible with Nomos value types:
  - Scalars: string, number, boolean, null
  - Collections: maps and lists
  - Nested structures supported

### Error Handling Patterns
Use standard gRPC status codes:
- `InvalidArgument` - Bad configuration or path
- `NotFound` - Path doesn't exist in provider
- `FailedPrecondition` - Provider not ready (e.g., Init not called)
- `Unavailable` - External resource unreachable
- `PermissionDenied` - Access denied
- `DeadlineExceeded` - Operation timeout

### Provider Type Registry
The `type` field in InfoResponse identifies provider category:
- `file` - File system providers
- `http` - HTTP/REST API providers
- `terraform` - Terraform state providers
- `custom` - User-defined types

### Source File Context
Init receives `source_file_path` for:
- Resolving relative paths in provider config
- Error reporting with file context
- Debugging provider issues

## Testing

### Running Tests
```bash
# All tests
go test -v ./...

# With coverage
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Contract Validation
The `contract_test.go` ensures:
- Generated code is syntactically valid
- Service interface is complete
- Message types are properly defined

## Linting and Validation

### Buf Linting
```bash
buf lint
```

Checks for:
- Standard protobuf style conventions
- Naming consistency
- Field numbering

### Breaking Change Detection
```bash
# Against main branch
buf breaking --against '.git#branch=main'

# Against specific tag
buf breaking --against '.git#tag=v0.1.0'
```

## Integration Points

### With Compiler
The compiler (`libs/compiler`) uses this module to:
- Import generated Go types
- Create provider client connections
- Invoke provider RPC methods
- Handle provider lifecycle

### With Provider Implementations
External providers import this module:
```go
import (
    providerv1 "github.com/autonomous-bits/nomos/libs/provider-proto/gen/go/nomos/provider/v1"
    "google.golang.org/grpc"
)

type myProvider struct {
    providerv1.UnimplementedProviderServiceServer
}

// Implement methods...
```

## References
- Feature spec: `docs/architecture/nomos-external-providers-feature-breakdown.md`
- Provider authoring: `docs/guides/provider-authoring-guide.md`
- Terraform providers: `docs/guides/terraform-providers-overview.md`
- Protocol Buffers: https://protobuf.dev/
- gRPC Go: https://grpc.io/docs/languages/go/
- Buf: https://buf.build/
