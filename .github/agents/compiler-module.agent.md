---
name: compiler-module
description: Specialized agent for libs/compiler - handles compilation of parsed AST into configuration snapshots, import resolution, provider integration, lockfile management, and deterministic configuration merging.
---

## Module Context
- **Path**: `libs/compiler`
- **Responsibilities**:
  - Parse Nomos scripts via `libs/parser` integration
  - Resolve `source`, `import`, and inline `reference` constructs via pluggable provider APIs
  - Compose configuration deterministically with last-wins override semantics and deep-merge for maps
  - Manage external provider subprocesses via gRPC (per-alias process model)
  - Detect and report cycles, missing references, and provider errors with rich diagnostics
  - Produce serializable snapshots (data + metadata) suitable for JSON/YAML/HCL rendering
  - Track provenance and metadata for auditing and debugging

- **Key Files**:
  - `compiler.go` - Public API entry point: `Compile(ctx, opts)` function
  - `manager.go` - Provider lifecycle and registry management
  - `provider.go` - Provider interface definitions and contracts
  - `provider_resolver.go` - Provider resolution and initialization logic
  - `provider_type_registry.go` - Type registry for provider mappings
  - `lockfile_resolver.go` - Lockfile parsing and provider version resolution
  - `import_resolution.go` - Import statement processing and ordering
  - `merge.go` - Deep-merge semantics and last-wins composition logic
  - `client.go` - Reference resolution and fetch caching
  - `internal/providerproc/` - External provider subprocess management (Manager + Client)
  - `internal/imports/` - Import resolution implementation
  - `internal/validator/` - Semantic validation (cycles, duplicates)
  - `internal/diagnostic/` - Error formatting with source snippets
  - `internal/resolver/` - Reference resolution engine
  - `internal/converter/` - AST-to-internal representation conversion

- **Test Pattern**:
  - Unit tests: `*_test.go` - test individual functions with fakes from `test/fakes`
  - Integration tests: `test/integration_test.go` - end-to-end compilation with fixtures
  - Network tests: `test/integration_network_test.go` (requires `//go:build integration` tag)
  - Concurrency tests: `test/concurrency_test.go` - validate thread safety with race detector
  - Benchmarks: `test/bench/compiler_bench_test.go` - performance of merge and reference resolution
  - Golden files: `testdata/` - deterministic snapshot validation (regenerate with `GOLDEN_UPDATE=1`)
  - Coverage: Enforce 80% minimum via CI

## Delegation Instructions
For general Go questions, **consult go-expert.agent.md**  
For testing questions, **consult testing-expert.agent.md**  
For provider communication/gRPC, **consult api-messaging-expert.agent.md**  
For parser AST details, **consult parser-module.agent.md**

## Compiler-Specific Patterns

### Provider Registration and Lifecycle

**In-Process Providers (Legacy, Being Phased Out)**
```go
type Provider interface {
    Init(ctx context.Context, opts ProviderInitOptions) error
    Fetch(ctx context.Context, path []string) (any, error)
}

type ProviderRegistry interface {
    GetProvider(alias string) (Provider, error)
}
```

**External Providers (Current Architecture)**
- Managed via `internal/providerproc` package
- One subprocess per provider alias per compilation run (per-alias model)
- Lazy start on first `GetProvider` call
- gRPC communication using `libs/provider-proto` contract
- Graceful shutdown via `Manager.Shutdown(ctx)`

**Process Lifecycle**:
1. Binary validation (file exists, executable)
2. Subprocess start with context cancellation
3. Port discovery: read `PROVIDER_PORT=<port>` from provider stdout
4. gRPC connection establishment
5. Health check via Health RPC
6. Client caching for reuse
7. Graceful termination on shutdown

**Provider Process Manager Usage**:
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

// Use as normal compiler.Provider interface
data, err := provider.Fetch(ctx, []string{"database", "prod"})
```

### Lockfile Management

**Lockfile Location**: `.nomos/providers.lock.json`

**Schema**:
```json
{
  "providers": [
    {
      "alias": "configs",
      "type": "file",
      "version": "0.2.0",
      "os": "darwin",
      "arch": "arm64",
      "source": {
        "github": {
          "owner": "autonomous-bits",
          "repo": "nomos-provider-file",
          "asset": "nomos-provider-file-0.2.0-darwin-arm64"
        }
      },
      "checksum": "sha256:...",
      "path": ".nomos/providers/file/0.2.0/darwin-arm64/provider"
    }
  ]
}
```

**Resolution Precedence** (during `nomos build`):
1. `.nomos/providers.lock.json` (authoritative binary paths and checksums)
2. Inline `.csl` source declaration (authoritative for version)
3. `.nomos/providers.yaml` manifest (source hints only)

**Provider Installation Layout**:
```
.nomos/
  providers/
    {name}/
      {version}/
        {os}-{arch}/
          provider         # executable
          CHECKSUM         # optional checksum file
```

### Import Resolution

**Import Statement Processing**:
- Imports are evaluated in declaration order
- Files are traversed lexicographically for deterministic ordering
- Circular import detection via cycle validator
- Deep-merge semantics: maps are merged recursively, scalars/arrays use last-wins

**Import Adapters**:
- Parse import paths (file system, Git, HTTP)
- Delegate to appropriate loader based on scheme
- Cache loaded files per compilation run

**Error Handling**:
- Missing imports → fatal error with source location
- Circular imports → fatal error with cycle path
- Import parse errors → wrapped with import statement context

### Provider Type Registry

**Purpose**: Maps provider types to implementations and metadata

**Registration Pattern**:
```go
registry := NewProviderTypeRegistry()
registry.Register("file", ProviderTypeInfo{
    Type:        "file",
    Description: "Filesystem provider",
    Constructor: NewFileProvider,
})

provider, err := registry.Get("file")
```

**Remote Type Registry** (External Providers):
- Queries provider binaries for type information via Info RPC
- Caches type metadata per compilation run
- Validates provider compatibility with compiler version

### Reference Resolution

**Syntax**: `reference:{alias}:{path}`
- `{alias}`: Provider alias from `source:` declaration
- `{path}`: Dot-separated navigation path (e.g., `storage.config.buckets.primary`)

**Resolution Behavior**:
- Per-run caching: identical provider+path → single fetch, subsequent resolutions use cache
- Context-aware: respects cancellation and timeouts (`Options.Timeouts.PerProviderFetch`)
- Thread-safe: internal cache uses read-write locks
- Error modes:
  - Fatal (default): provider fetch failures cause compilation to fail
  - Non-fatal: `Options.AllowMissingProvider = true` → failures recorded as warnings

**Path Navigation**:
For file providers: `{filename}.{nested.path.to.value}`
```
reference:configs:storage.config.storage.type  
→ Fetch from 'configs' provider: ["storage"]
→ Navigate: ["config", "storage", "type"]
```

**Client Implementation** (`client.go`):
- Maintains fetch cache with mutex protection
- Wraps provider errors with diagnostic context
- Provides helper methods for cache inspection in tests

### Merge Semantics

**Deep-Merge Rules** (from `merge.go`):
- **Maps**: Recursive merge, combining keys from both sources
- **Scalars**: Last-wins (newer value replaces older)
- **Arrays**: Last-wins (entire array replaced, no element-wise merge)
- **Null handling**: Explicit null in newer overrides older value

**Provenance Tracking**:
```go
type Provenance struct {
    Source        string `json:"source"`         // File path
    ProviderAlias string `json:"provider_alias"` // Provider if value came from reference
}
```

**Merge with Provenance**:
```go
result, provenance := compiler.DeepMerge(base, override, baseSource, overrideSource)
```

### Diagnostic Formatting

**Source-Aware Error Messages** (`internal/diagnostic/`):
```go
formatted := diagnostic.FormatDiagnostic(diag, sourceText, parseErr)
```

**Output Format**:
```
app.csl:3:9: error: unresolved reference to provider 'config'
   3 |   port: reference:config:port
     |         ^
```

**Features**:
- Uses `parser.FormatParseError` for parse errors
- Generates caret-based snippets for semantic/provider errors
- Machine-parseable prefix: `file:line:col: severity: message`
- Handles multi-byte UTF-8 correctly in column positioning
- Context lines with caret pointing to error location

### Snapshot Metadata

**Structure**:
```go
type Snapshot struct {
    Data     map[string]any  `json:"data"`
    Metadata Metadata        `json:"metadata"`
}

type Metadata struct {
    InputFiles       []string              `json:"input_files"`
    ProviderAliases  []string              `json:"provider_aliases"`
    StartTime        time.Time             `json:"start_time"`
    EndTime          time.Time             `json:"end_time"`
    Errors           []string              `json:"errors"`
    Warnings         []string              `json:"warnings"`
    PerKeyProvenance map[string]Provenance `json:"per_key_provenance"`
}
```

**Usage**:
- **InputFiles**: Sorted list of all `.csl` files (absolute paths, deterministic ordering)
- **ProviderAliases**: All registered provider aliases
- **StartTime/EndTime**: For profiling and audit trails
- **Errors**: Fatal compilation errors (empty for successful builds)
- **Warnings**: Non-fatal issues (e.g., missing provider with `AllowMissingProvider`)
- **PerKeyProvenance**: Maps top-level keys to origin file and provider

**JSON Serialization**: Uses snake_case for JSON field names following Go conventions

### Parser Contract

**What Compiler Expects from Parser**:

**Entry Points**:
- `parser.ParseFile(path string) (*ast.AST, error)` - convenience function
- `parser.Parse(r io.Reader, filename string) (*ast.AST, error)` - from reader
- `parser.NewParser()` - reusable parser instances

**AST Structure**:
- Root: `*ast.AST` with `Statements []ast.Stmt` and `SourceSpan`
- Statements: `ast.SourceDecl`, `ast.ImportStmt`, `ast.SectionDecl`
- Values: `ast.Expr` including `*ast.StringLiteral`, `*ast.ReferenceExpr`
- References: `ReferenceExpr` with `Alias`, `Path []string`, and `SourceSpan`

**Position Information**:
- Every node includes `SourceSpan` with filename, line/column
- **1-indexed**: line and column numbers start at 1
- **Byte-based columns**: not rune counts
- **Inclusive EndCol**: points to last byte of token

**Error Types**:
- `*parser.ParseError` with kinds: `LexError`, `SyntaxError`, `IOError`
- Includes filename, line, column, and optional snippet
- Use `parser.FormatParseError(err, sourceText)` for user-friendly display

**Validation Scope**:
- Parser: syntax-level only (keywords, string termination, alias format)
- Compiler: semantic validation (duplicate keys, provider existence, references, cycles)

**Inline References**:
- Top-level `reference:` statements → `SyntaxError` with migration hint
- Inline references: `key: reference:alias:path` → `ast.ReferenceExpr` as value

### Module Dependencies

**Depends On**:
- `libs/parser` - AST parsing and syntax analysis
- `libs/provider-proto` - gRPC service contract for external providers
- `libs/provider-downloader` - Provider binary download and installation (used by CLI, not compiler directly)

**Used By**:
- `apps/command-line` - CLI wraps compiler for user-facing commands

**Dependency Flow**:
```
CLI → Compiler → Parser
         ↓
  Provider Proto (gRPC)
         ↓
  External Provider Binaries
```

### Timeout Configuration

**Per-Operation Timeouts**:
```go
type OptionsTimeouts struct {
    PerProviderFetch time.Duration // Default timeout for each Fetch call
}

opts := compiler.Options{
    Path: "./config",
    Timeouts: compiler.OptionsTimeouts{
        PerProviderFetch: 5 * time.Second,
    },
}
```

**Context Cancellation**:
All provider operations respect context cancellation:
```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

snapshot, err := compiler.Compile(ctx, opts)
```

### Validation and Error Detection

**Semantic Validators** (`internal/validator/`):
- **Cycle detection**: Import cycles, reference cycles
- **Duplicate key detection**: Within single file, across imports
- **Unresolved references**: Provider not found, path doesn't exist
- **Provider errors**: Init failures, fetch failures

**Error Wrapping**:
```go
// Wrap lower-level errors with context
return fmt.Errorf("failed to resolve reference %q: %w", ref, err)

// Check error types
if errors.Is(err, compiler.ErrUnresolvedReference) { ... }
if errors.Is(err, compiler.ErrCycleDetected) { ... }
```

## Common Tasks

### 1. Implementing New Compilation Features
- Add to public API in `compiler.go` if user-facing
- Implement logic in `internal/` packages for encapsulation
- Add integration tests with golden files in `testdata/`
- Update `README.md` with usage examples
- Ensure deterministic behavior across platforms

### 2. Provider Integration and External Provider Migration
- Use `internal/providerproc` for external subprocess management
- Implement gRPC client adapter wrapping `compiler.Provider` interface
- Add lockfile resolution in `lockfile_resolver.go`
- Update provider registry to support remote type info
- Test with real provider binaries and fake providers
- Document in `docs/architecture/nomos-external-providers-feature-breakdown.md`

### 3. Import Resolution Improvements
- Modify `import_resolution.go` and `internal/imports/`
- Maintain deterministic file ordering (lexicographic)
- Update cycle detection in `internal/validator/`
- Test with complex import graphs in integration tests
- Ensure error messages include full import chain context

### 4. Lockfile Format Changes
- Update schema in `lockfile_resolver.go`
- Maintain backward compatibility or document breaking changes
- Update CLI `init` command to generate new format
- Add migration guide in `docs/guides/`
- Test with existing lockfiles for regression

### 5. Type System Enhancements
- Extend AST node types in coordination with parser team
- Update converter in `internal/converter/`
- Modify resolution logic in `internal/resolver/`
- Add validation rules in `internal/validator/`
- Update diagnostic formatting for new error cases
- Add comprehensive tests covering new types

## Nomos-Specific Details

### Deterministic Compilation
- **Goal**: Identical inputs → identical outputs across platforms
- **Mechanisms**:
  - Lexicographic directory traversal
  - Sorted provider aliases in metadata
  - Stable merge ordering (declaration order for imports)
  - Deterministic map key ordering in JSON output
  - Per-run provider fetch caching (same path → same result)

### Configuration Composition Philosophy
- **Last-Wins**: Later declarations override earlier ones
- **Deep-Merge**: Maps are merged recursively, preserving non-conflicting keys
- **Explicit Intent**: Arrays replace entirely (no element-wise merge)
- **Provenance**: Track which file contributed each top-level key for debugging

### Provider Execution Model
- **Security**: Only execute from trusted locations (`.nomos/providers/`)
- **Isolation**: Subprocess per alias, no shared state
- **Lifecycle**: Lazy start, compile-duration lifetime, graceful shutdown
- **Communication**: gRPC over localhost TCP with port discovery
- **Health**: Preflight health checks before first fetch

### Error Philosophy
- **Rich Context**: Include file, line, column, source snippet
- **Actionable**: Suggest remediation (e.g., "run `nomos init`")
- **Layered**: Wrap lower-level errors, preserve root causes
- **Consistent**: Use parser's formatting for syntax errors, match style for semantic errors
- **Machine-Parseable**: Structured error prefix for tooling integration

### Performance Considerations
- **Fetch Caching**: Cache provider responses per compilation run (avoid redundant fetches)
- **Lazy Provider Start**: Only start subprocesses when needed
- **Parallel-Ready**: Thread-safe caches and registries (safe for concurrent `Compile` calls)
- **Benchmark Baselines**: Track performance in CI, flag regressions

### Testing Philosophy
- **Determinism**: Golden files for snapshot validation
- **Isolation**: Use fakes/mocks for providers in unit tests
- **Real Integration**: Include tests with actual provider binaries
- **Concurrency**: Run with race detector enabled
- **Coverage**: Maintain 80% minimum, focus on complex logic paths
- **Performance**: Benchmark merge and reference resolution regularly

## Best Practices

1. **Keep Public API Stable**: Changes to `compiler.go` exported functions require careful consideration
2. **Use Internal Packages**: Implementation details belong in `internal/` to prevent external coupling
3. **Test with Fakes First**: Unit tests should use `test/fakes`, integration tests use real providers
4. **Document Breaking Changes**: Update `CHANGELOG.md` with clear migration paths
5. **Validate Before Publishing**: Ensure all tests pass including race detector and integration suite
6. **Format Errors Consistently**: Always use `diagnostic.FormatDiagnostic` for user-facing errors
7. **Cache Judiciously**: Provider fetches are cached per-run; document cache behavior
8. **Graceful Degradation**: Support `AllowMissingProvider` for non-critical providers
9. **Version Independently**: Tag as `libs/compiler/vX.Y.Z`, follow semantic versioning
10. **Profile Performance**: Use benchmarks to validate optimization changes

## References

- **Architecture**: `docs/architecture/nomos-external-providers-feature-breakdown.md`
- **Provider Authoring**: `docs/guides/provider-authoring-guide.md`
- **Terraform Model**: `docs/guides/terraform-providers-overview.md`
- **Import Status**: `docs/import-implementation-status.md`
- **Merge Semantics**: `docs/merge.md`
- **Provider Contract**: `docs/provider.md`
- **gRPC Proto**: `libs/provider-proto/README.md`
- **Parser Integration**: `libs/parser/README.md`

## Need Help?

- For public API changes, open a PR with design rationale
- For provider integration questions, reference external providers architecture doc
- For test failures, check golden file regeneration: `GOLDEN_UPDATE=1 go test ./...`
- For performance issues, run benchmarks and compare against baselines in CI artifacts
