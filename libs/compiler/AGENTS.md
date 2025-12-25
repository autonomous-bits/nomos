# Nomos Compiler Agent-Specific Patterns

> **Note**: For comprehensive compiler development guidance, see `.github/agents/compiler-module.agent.md`  
> For task coordination, start with `.github/agents/nomos.agent.md`

## Nomos-Specific Patterns

### Provider Registration

**External Providers Architecture (Current)**

Nomos uses a per-alias subprocess model for provider execution:

```go
// Provider lifecycle managed via internal/providerproc
manager := providerproc.NewManager()
defer manager.Shutdown(context.Background())

opts := compiler.ProviderInitOptions{
    Alias:  "configs",
    Config: map[string]any{"directory": "./configs"},
}

provider, err := manager.GetProvider(ctx, "configs", binaryPath, opts)
```

**Process Lifecycle:**
1. Binary validation from trusted `.nomos/providers/` directory
2. Lazy start on first `GetProvider` call
3. Port discovery via `PROVIDER_PORT=<port>` stdout parsing
4. gRPC connection with health check
5. Compile-duration lifetime
6. Graceful shutdown with context cancellation

**Key Files:**
- `manager.go` - Provider lifecycle and registry
- `provider_resolver.go` - Provider resolution and initialization
- `provider_type_registry.go` - Type registry with remote query support
- `internal/providerproc/` - Subprocess management

### Lockfile Format

**Location:** `.nomos/providers.lock.json`

**Schema:**
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

**Resolution Precedence:**
1. `.nomos/providers.lock.json` (authoritative paths and checksums)
2. Inline `.csl` source declaration (version authority)
3. `.nomos/providers.yaml` (source hints only)

**Installation Layout:**
```
.nomos/
  providers/
    {name}/
      {version}/
        {os}-{arch}/
          provider         # executable
          CHECKSUM
```

**Key Files:**
- `lockfile_resolver.go` - Lockfile parsing and version resolution

### Import Resolution

**Nomos Import Semantics:**
- Declaration-order processing
- Lexicographic file traversal (deterministic)
- Deep-merge for maps, last-wins for scalars/arrays
- Circular import detection

**Import Path Adapters:**
- File system: relative or absolute paths
- Git: `git://repo/path` (future)
- HTTP: `https://domain/path` (future)

**Error Handling:**
- Missing imports → fatal with source location
- Circular imports → fatal with cycle path
- Parse errors → wrapped with import context

**Key Files:**
- `import_resolution.go` - Import statement processing
- `internal/imports/` - Resolution implementation
- `internal/validator/` - Cycle detection

### Type Registry

**Purpose:** Maps Nomos provider types to implementations

**Local Registration:**
```go
registry := NewProviderTypeRegistry()
registry.Register("file", ProviderTypeInfo{
    Type:        "file",
    Description: "Filesystem provider",
    Constructor: NewFileProvider,
})
```

**Remote Type Registry:**
- Queries provider binaries via gRPC Info RPC
- Caches type metadata per compilation run
- Validates compatibility with compiler version

**Key Files:**
- `provider_type_registry.go` - Registry implementation
- `provider_type_registry_remote_test.go` - Remote query tests

### Reference Resolution

**Nomos Reference Syntax:** `reference:{alias}:{path}`

**Path Navigation:**
```
reference:configs:storage.config.storage.type
→ Fetch from 'configs' provider: ["storage"]
→ Navigate: ["config", "storage", "type"]
```

**Caching Behavior:**
- Per-compilation run cache
- Same provider+path → single fetch
- Thread-safe with mutex protection
- Context-aware timeouts

**Error Modes:**
- Fatal (default): provider failures stop compilation
- Non-fatal: `Options.AllowMissingProvider = true` → warnings

**Key Files:**
- `client.go` - Reference resolution with caching
- `internal/resolver/` - Resolution engine

### Configuration Merge Semantics

**Nomos Deep-Merge Rules:**
- **Maps:** Recursive merge, combining keys
- **Scalars:** Last-wins (newer replaces older)
- **Arrays:** Last-wins (entire array replaced)
- **Null:** Explicit null overrides previous value

**Provenance Tracking:**
```go
type Provenance struct {
    Source        string `json:"source"`         // File path
    ProviderAlias string `json:"provider_alias"` // Provider source
}

result, provenance := compiler.DeepMerge(base, override, baseSource, overrideSource)
```

**Determinism:**
- Lexicographic import ordering
- Stable map key output
- Per-run fetch caching prevents non-determinism

**Key Files:**
- `merge.go` - Deep-merge implementation
- `merge_test.go` - Merge semantics validation

### Snapshot Metadata

**Nomos Snapshot Structure:**
```go
type Snapshot struct {
    Data     map[string]any  `json:"data"`
    Metadata Metadata        `json:"metadata"`
}

type Metadata struct {
    InputFiles       []string              `json:"input_files"`       // Sorted absolute paths
    ProviderAliases  []string              `json:"provider_aliases"`  // Sorted aliases
    StartTime        time.Time             `json:"start_time"`        // Audit trail
    EndTime          time.Time             `json:"end_time"`          // Duration
    Errors           []string              `json:"errors"`            // Fatal errors
    Warnings         []string              `json:"warnings"`          // Non-fatal issues
    PerKeyProvenance map[string]Provenance `json:"per_key_provenance"` // Origin tracking
}
```

**Usage:**
- Auditing: track which files/providers contributed to output
- Debugging: identify source of configuration values
- Reproducibility: snapshot includes full compilation context

**Key Files:**
- `compiler.go` - Snapshot and Metadata types
- `metadata_test.go` - Metadata validation

### Build Tags

**Integration Tests:**
```go
//go:build integration

// Network-dependent tests requiring external services
```

**Usage:**
```bash
# Run standard tests (no network)
go test ./...

# Run integration tests
go test -tags=integration ./test
```

**Key Files:**
- `test/integration_network_test.go` - Network-dependent tests
- `test/integration_test.go` - Standard integration tests
- `test/concurrency_test.go` - Race condition tests

### Deterministic Compilation

**Nomos Determinism Guarantees:**
- Identical inputs → identical outputs (cross-platform)
- Lexicographic directory traversal
- Sorted provider aliases in metadata
- Stable merge ordering
- Deterministic JSON output (sorted keys)
- Per-run caching prevents fetch variability

**Testing:**
```bash
# Golden file tests validate determinism
go test ./test -v

# Regenerate golden files
GOLDEN_UPDATE=1 go test ./test
```

**Key Files:**
- `testdata/` - Golden files for snapshot validation
- `test/integration_test.go` - Determinism tests

### Diagnostic Formatting

**Nomos Error Format:**
```
app.csl:3:9: error: unresolved reference to provider 'config'
   3 |   port: reference:config:port
     |         ^
```

**Features:**
- Machine-parseable: `file:line:col: severity: message`
- Source snippets with caret pointing to error
- UTF-8 aware column positioning
- Delegates to parser for syntax errors

**Key Files:**
- `internal/diagnostic/` - Error formatting
- `errors.go` - Error types and wrapping

### Parser Contract

**What Compiler Expects:**
- `parser.ParseFile(path string) (*ast.AST, error)`
- `parser.Parse(r io.Reader, filename string) (*ast.AST, error)`
- `ast.ReferenceExpr` for inline references: `key: reference:alias:path`
- **1-indexed** line and column numbers
- **Byte-based columns** (not rune counts)
- `SourceSpan` on every AST node for error reporting

**Error Handling:**
- Parser: syntax-level validation only
- Compiler: semantic validation (cycles, references, providers)

**Key Files:**
- `compiler.go` - Parser integration
- `internal/converter/` - AST conversion
- `interface_check.go` - Parser interface validation

### Provider Security Model

**Trusted Execution:**
- Only execute binaries from `.nomos/providers/`
- Checksum verification (lockfile)
- Subprocess isolation (no shared state)
- Localhost-only gRPC (no network exposure)

**Lifecycle Management:**
- Lazy start reduces attack surface
- Graceful shutdown prevents resource leaks
- Health checks before first use
- Context cancellation for timeouts

**Key Files:**
- `provider_resolver.go` - Binary validation
- `internal/providerproc/` - Subprocess security

### Timeout Configuration

**Nomos Timeout Settings:**
```go
opts := compiler.Options{
    Path: "./config",
    Timeouts: compiler.OptionsTimeouts{
        PerProviderFetch: 5 * time.Second, // Per-fetch timeout
    },
}

// Context cancellation for overall compilation
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

snapshot, err := compiler.Compile(ctx, opts)
```

**Key Files:**
- `compiler.go` - Options and OptionsTimeouts types

---

## References

- **Architecture:** `docs/architecture/nomos-external-providers-feature-breakdown.md`
- **Provider Authoring:** `docs/guides/provider-authoring-guide.md`
- **Merge Semantics:** `docs/merge.md`
- **Provider Contract:** `docs/provider.md`
- **gRPC Proto:** `libs/provider-proto/README.md`
- **Parser Integration:** `libs/parser/README.md`

