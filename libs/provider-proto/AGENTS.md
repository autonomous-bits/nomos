# Nomos Provider Proto Agent-Specific Patterns

> **Note**: For comprehensive guidance, see `.github/agents/provider-proto-module.agent.md`  
> For task coordination, start with `.github/agents/nomos.agent.md`

## Nomos-Specific Patterns

### Provider Protocol

Nomos defines a **subprocess + gRPC** architecture for external providers:

1. Compiler spawns provider executable as subprocess
2. Provider starts gRPC server on allocated port
3. Compiler establishes client connection
4. **Lifecycle**: Init → Fetch (multiple) → Shutdown
5. Provider process terminates

**Service Contract** - Five RPC methods:
- **Init**: Initialize with config (called once, receives `alias`, `config` struct, `source_file_path`)
- **Fetch**: Retrieve data at path (primary method, receives path segments array)
- **Info**: Return metadata (alias, version, type identifier)
- **Health**: Check operational status (returns status and message)
- **Shutdown**: Graceful cleanup (best-effort, compiler may force terminate)

### Contract Versioning

Nomos provider protocol follows **semantic versioning**:

**Breaking changes** (require major version bump):
- Removing/renaming fields
- Changing field types
- Removing RPC methods
- Changing method signatures

**Non-breaking changes** (minor/patch):
- Adding optional fields
- Adding new RPC methods
- Adding new enum values (with care)

**Validation**: Use `buf breaking --against '.git#branch=main'` before releasing

### Code Generation

**Versioned API structure**:
```
proto/nomos/provider/v1/provider.proto  →  gen/go/nomos/provider/v1/*.pb.go
```

**Generation workflow**:
```bash
buf lint                                    # Check style
buf breaking --against '.git#branch=main'  # Check compatibility
buf generate                                # Generate code
go test -v ./...                            # Verify
```

### Data Structure Compatibility

Fetch responses use `google.protobuf.Struct` and must map to Nomos value types:
- **Scalars**: string, number, boolean, null
- **Collections**: maps and lists
- **Nested structures** supported

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

### Error Handling Patterns

Use standard gRPC status codes:
- `InvalidArgument` - Bad configuration or path
- `NotFound` - Path doesn't exist in provider
- `FailedPrecondition` - Provider not ready (Init not called)
- `Unavailable` - External resource unreachable
- `PermissionDenied` - Access denied
- `DeadlineExceeded` - Operation timeout

### Build Tags

No special build tags required. Generated code works across all platforms.
