# Nomos Provider Proto Agent-Specific Patterns

> **Note**: For comprehensive guidance, see `.github/agents/nomos-provider-specialist.agent.md`  
> For task coordination, start with `.github/agents/nomos-orchestrator.agent.md`

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

---

## Task Completion Verification

**MANDATORY**: Before completing ANY task, the agent MUST verify all of the following:

### 1. Build Verification ✅
```bash
go build ./...
```
- All code must compile without errors
- Generated protobuf code is up to date
- No unresolved imports or type errors

### 2. Protobuf Verification ✅
```bash
buf lint
buf breaking --against .git#branch=main
make generate  # Ensure generated code is current
```
- Proto files pass buf linting
- No breaking changes (unless intentional and documented)
- Generated Go code matches proto definitions

### 3. Test Verification ✅
```bash
go test ./...
go test ./... -race  # Check for race conditions
```
- All existing tests must pass
- New tests must be added for new RPC methods
- Race detector must report no data races
- gRPC integration tests must pass
- Contract tests validate all message types

### 4. gRPC Integration Verification ✅
- Real client-server tests pass
- All RPC methods tested (Init, Fetch, Info, Health, Shutdown)
- Error status codes are correct
- Data serialization round-trips work
- Lifecycle ordering is enforced

### 5. Linting Verification ✅
```bash
go vet ./...
golangci-lint run
```
- No `go vet` warnings
- No golangci-lint errors
- Code follows Go best practices
- No unused functions in test files

### 6. Documentation Updates ✅
- Update CHANGELOG.md if contract changed
- Update README.md if API changed
- Update proto comments for new fields/methods
- Verify README examples compile

### Verification Checklist Template

When completing a task, report:
```
✅ Build: Successful
✅ Protobuf: Lint clean, no breaking changes
✅ Generated Code: Up to date
✅ Tests: XX/XX passed
✅ Race Detector: Clean
✅ gRPC Integration: All methods tested
✅ Linting: Clean
✅ Documentation: Updated [list files]
```

**DO NOT** mark a task as complete without running ALL verification steps and reporting results.
