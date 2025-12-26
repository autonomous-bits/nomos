# Nomos Compiler Library

The compiler turns Nomos source files into a deterministic configuration snapshot. It is a small, stable Go library consumed by the CLI and other tools.

## Overview

The Nomos compiler library provides compilation functionality for Nomos configuration scripts (.csl files). It integrates with the parser library for syntax analysis, supports pluggable provider adapters for external data sources, and enforces deterministic composition semantics with deep-merge and last-wins behavior.

## Installation

```bash
go get github.com/autonomous-bits/nomos/libs/compiler
```

## Basic Usage

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/autonomous-bits/nomos/libs/compiler"
)

// Example provider registry implementation
type SimpleRegistry struct {
	providers map[string]compiler.Provider
}

func (r *SimpleRegistry) GetProvider(alias string) (compiler.Provider, error) {
	if p, ok := r.providers[alias]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("provider %q not found", alias)
}

func main() {
	ctx := context.Background()
	
	// Configure compilation options
	opts := compiler.Options{
		Path:             "./configs",          // Directory or file path
		ProviderRegistry: &SimpleRegistry{},    // Provider registry
		Vars:             map[string]any{},     // Optional variables
	}

	// Compile source files into snapshot
	snapshot, err := compiler.Compile(ctx, opts)
	if err != nil {
		log.Fatal(err)
	}

	// Access compiled data
	fmt.Printf("Compiled %d files\n", len(snapshot.Metadata.InputFiles))
	fmt.Printf("Configuration: %+v\n", snapshot.Data)
}
```

## API Reference

### Core Types

#### Options

Configures a compilation run:

```go
type Options struct {
	Path                 string            // Input file or directory path (required)
	ProviderRegistry     ProviderRegistry  // Provider registry (required)
	Vars                 map[string]any    // Variable substitutions (optional)
	Timeouts             OptionsTimeouts   // Timeout configuration
	AllowMissingProvider bool              // Allow provider fetch failures (default: false)
}
```

#### Snapshot

The compiled output containing data and metadata:

```go
type Snapshot struct {
	Data     map[string]any  // Compiled configuration
	Metadata Metadata        // Provenance and diagnostics
}
```

#### Metadata

Provenance and diagnostic information:

```go
type Metadata struct {
	InputFiles       []string              // Source files processed
	ProviderAliases  []string              // Providers used
	StartTime        time.Time             // Compilation start
	EndTime          time.Time             // Compilation end
	Warnings         []string              // Non-fatal issues
	PerKeyProvenance map[string]Provenance // Value origins
}
```

### Provider Interface

Providers fetch data from external sources:

```go
type Provider interface {
	Init(ctx context.Context, opts ProviderInitOptions) error
	Fetch(ctx context.Context, path []string) (any, error)
}

type ProviderRegistry interface {
	GetProvider(alias string) (Provider, error)
}
```

### Functions

#### Compile

```go
func Compile(ctx context.Context, opts Options) (Snapshot, error)
```

Compiles Nomos source files into a deterministic configuration snapshot. The context controls cancellation and timeout. Returns a Snapshot on success or an error with location information on failure.

## Determinism

Compilation is deterministic: given identical inputs and provider responses, the compiler produces identical snapshots. Directory traversal is performed in lexicographic order to ensure consistency across platforms.

## Error Handling

The compiler returns structured errors with source location information when available:

```go
snapshot, err := compiler.Compile(ctx, opts)
if err != nil {
	if errors.Is(err, compiler.ErrUnresolvedReference) {
		// Handle unresolved reference
	}
	if errors.Is(err, compiler.ErrCycleDetected) {
		// Handle cycle detection
	}
	// Handle other errors
	log.Fatal(err)
}
```

## Goals

- Parse Nomos scripts and build an internal representation (via the parser).
- Resolve `source`, `import` and inline `reference` constructs via pluggable provider APIs.
- Compose configuration deterministically with last-wins override semantics and deep-merge for maps.
- Detect and report cycles, missing references and provider errors with context-rich diagnostics.
- Produce a serializable snapshot (data + metadata) suitable for JSON/YAML/HCL rendering.

## Relationship to other projects

- Consumed by `apps/command-line` (the CLI).
- Depends on `libs/parser` for tokenizing/parsing and AST construction.

```
CLI -> Compiler -> Parser
```

Notes:
- `Snapshot` is format-agnostic; the caller chooses JSON/YAML/HCL rendering.
- Providers are pluggable and are addressed by alias in scripts.

## Parser contract (what compiler relies on)

The compiler must consume the parser output and error types in specific ways. The parser (in `libs/parser`) exposes a stable set of entry points and AST shapes — the compiler must treat parsing errors as lower-level, recoverable diagnostics and use AST node spans to produce rich error messages.

Key parser behaviours the compiler relies on:

- Public parse entry points:
    - `parser.ParseFile(path string) (*ast.AST, error)` — convenience top-level function.
    - `parser.Parse(r io.Reader, filename string) (*ast.AST, error)` — parse from an arbitrary reader.
    - `parser.NewParser()` and `(*Parser).Parse*` allow reusing pooled parser instances.

- AST shape:
    - Root `*ast.AST` contains `Statements []ast.Stmt` and a `SourceSpan`.
    - Statement types include `ast.SourceDecl`, `ast.ImportStmt`, `ast.SectionDecl` (sections with map entries).
    - Values in section entries are `ast.Expr`. Supported value expressions currently include `*ast.StringLiteral` and `*ast.ReferenceExpr` for inline references.
    - `ReferenceExpr` has `Alias string`, `Path []string` and carries a `SourceSpan`.

- Inline references:
    - The parser treats inline references as first-class values: `key: reference:alias:dot.path` produces an `ast.ReferenceExpr` assigned as the entry value.
    - Top-level `reference:` statements (legacy) are rejected by the parser with a `SyntaxError` and a migration hint — the compiler should not expect top-level reference statements.

- Source spans and position information:
    - Every AST node includes `SourceSpan` with filename, start/end line and column.
    - Line and column numbers are 1-indexed. Columns are byte offsets (not rune counts); `EndCol` is inclusive (points to last byte of the token).

- Parse errors:
    - The parser returns `*parser.ParseError` with kinds `LexError`, `SyntaxError`, or `IOError` and includes filename/line/column and an optional snippet.
    - Use `parser.FormatParseError(err, sourceText)` to render caret-marked user-friendly messages.

- Validation scope:
    - The parser only enforces syntax-level rules (e.g., required `:` after keywords, properly-terminated strings, alias is a string literal).
    - Structural/semantic checks such as duplicate-key detection, provider existence, and reference resolution are intentionally left to the compiler.

Implication for the compiler:

- Treat `ReferenceExpr` nodes as value placeholders to be resolved by providers/import resolution. Do not attempt to interpret `reference:` syntax as a top-level statement — it's only valid as a value.
- When reporting errors about user code, prefer to decorate parser errors by using the `SourceSpan` and `FormatParseError` output so messages are consistent with parser diagnostics.
- Expect some syntax validation to already be performed by the parser; the compiler must perform semantic validation (duplicate keys, cycles, missing imports, provider errors).

## Composition and resolution semantics

- Imports are applied in evaluation order; on conflict keys are overwritten (last-wins).
- Maps are deep-merged; arrays replace by default.
- References (inline `ReferenceExpr`) are resolved after imports/values from providers are materialized, allowing cross-file linking and importing.
- Cycles across imports/references must be detected and reported by the compiler.

### Reference Resolution

The compiler resolves inline `reference:` expressions via the provider system. References are first-class values that can appear anywhere a value is expected.

**Syntax:** `reference:{alias}:{path}`

Where:
- `{alias}` is the provider alias (from a `source:` declaration)
- `{path}` is a dot-separated path to navigate into the provider's data

For file providers, the path format is: `{filename}.{nested.path.to.value}`

**Per-run caching:** Provider fetch results are cached for the duration of a single compilation run. Identical provider+path combinations result in a single provider call, with subsequent resolutions using the cached value.

**Context-aware:** All provider fetch operations respect the provided context for cancellation and timeouts. Use `Options.Timeouts.PerProviderFetch` to set a default timeout.

**Error handling:** By default, provider fetch failures are fatal and cause compilation to fail. Set `Options.AllowMissingProvider = true` to treat failures as non-fatal warnings recorded in `Snapshot.Metadata.Warnings`.

Example with path navigation:

```go
// File provider pointing to ./configs directory containing storage.csl:
// config:
//   storage:
//     type: 's3'
//     region: 'us-west-2'
//   buckets:
//     primary: 'my-app-data'
//   encryption:
//     algorithm: 'AES256'

// Configuration with references using path navigation
config := `
source:
  alias: 'configs'
  type: 'autonomous-bits/nomos-provider-file'
  version: '0.1.1'
  directory: './configs'

app:
  storage_type: reference:configs:storage.config.storage.type        # Resolves to 's3'
  region: reference:configs:storage.config.storage.region            # Resolves to 'us-west-2'
  bucket: reference:configs:storage.config.buckets.primary           # Resolves to 'my-app-data'
  encryption: reference:configs:storage.config.encryption.algorithm  # Resolves to 'AES256'
`

// Compile with provider
opts := compiler.Options{
	Path:             "config.csl",
	ProviderRegistry: registry,
	AllowMissingProvider: false, // Fatal on missing provider (default)
}

snapshot, err := compiler.Compile(ctx, opts)
// References resolved to actual values from providers
// snapshot.Data["app"]["storage_type"] == "s3"
// snapshot.Data["app"]["bucket"] == "my-app-data"
```

**Thread safety:** The internal cache uses read-write locks for safe concurrent access.

## Providers and sources

- Providers resolve data from backing systems (filesystem, Git, HTTP, cloud state).
- `source` declarations in the AST map to provider instances by alias and type.
- The compiler should use a provider registry to instantiate providers and cache provider results for the duration of a single compilation.

## Errors and diagnostics

- Use parser-provided `ParseError` for syntax/lexing faults. For semantic errors, return structured errors that include `SourceSpan` when possible.
- Wrap lower-level errors (`fmt.Errorf("...: %w", err)`) to preserve root causes.

### Diagnostic Formatting

The `diagnostic` package provides `FormatDiagnostic` for formatting compiler diagnostics with source snippets and caret markers:

```go
import "github.com/autonomous-bits/nomos/libs/compiler/internal/diagnostic"

// Format a diagnostic with source snippet
formatted := diagnostic.FormatDiagnostic(diag, sourceText, parseErr)
```

FormatDiagnostic produces output like:

```
app.csl:3:9: error: unresolved reference to provider 'config'
   3 |   port: reference:config:port
     |         ^
```

The function:
- Uses `parser.FormatParseError` for parse errors when a ParseError is provided
- Generates caret-based snippets for semantic/provider errors using SourceSpan information
- Returns a machine-parseable prefix (`file:line:col: severity: message`) followed by context lines
- Handles multi-byte UTF-8 characters correctly in column positioning

### Snapshot Metadata

Every compilation produces a `Snapshot` containing both the compiled `Data` and rich `Metadata`:

```go
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

Fields:
- **InputFiles**: Sorted list of all `.csl` source files processed (absolute paths)
- **ProviderAliases**: All provider aliases registered in the ProviderRegistry
- **StartTime/EndTime**: Compilation timestamps for profiling and audit trails
- **Errors**: Fatal compilation errors (typically empty for successful compilations)
- **Warnings**: Non-fatal issues (e.g., provider fetch failures when `AllowMissingProvider` is true)
- **PerKeyProvenance**: Maps each top-level configuration key to its origin

#### Provenance Tracking

Each top-level key in the compiled data includes provenance information:

```go
type Provenance struct {
	Source        string `json:"source"`
	ProviderAlias string `json:"provider_alias"`
}
```

Example:

```go
snapshot, _ := compiler.Compile(ctx, opts)

// Check where a value came from
if prov, ok := snapshot.Metadata.PerKeyProvenance["database"]; ok {
	fmt.Printf("'database' came from: %s\n", prov.Source)
	if prov.ProviderAlias != "" {
		fmt.Printf("  via provider: %s\n", prov.ProviderAlias)
	}
}
```

The provenance map helps users:
- Debug last-wins merge behavior by seeing which file contributed each key
- Understand which providers were involved in resolving configuration values
- Trace configuration back to source for auditing and compliance

**JSON Serialization**: Metadata uses snake_case JSON field names following Go conventions:

```json
{
  "data": { "app": "myapp" },
  "metadata": {
    "input_files": ["/path/to/config.csl"],
    "provider_aliases": ["file", "env"],
    "start_time": "2025-10-26T10:00:00Z",
    "end_time": "2025-10-26T10:00:05Z",
    "errors": [],
    "warnings": [],
    "per_key_provenance": {
      "app": {
        "source": "/path/to/config.csl",
        "provider_alias": ""
      }
    }
  }
}
```

## Testing

The compiler library includes comprehensive test coverage across unit tests, integration tests, concurrency tests, and performance benchmarks.

### Running Tests

```bash
# Run all tests (excludes integration tests by default)
make test

# Run tests with race detector
make test-race

# Generate coverage report (HTML)
make test-coverage
# Opens coverage.html in your browser

# Run integration tests (network-backed providers)
make test-integration

# Run all tests including integration
go test -tags=integration -v ./...
```

### Running Benchmarks

Benchmarks measure performance of merge semantics and reference resolution:

```bash
# Run all benchmarks
make bench

# Run specific benchmark
go test -bench=BenchmarkMergeSmall -benchmem ./test/bench

# Run with higher iterations for stability
go test -bench=. -benchtime=10s ./test/bench
```

**Baseline Performance (Apple M2, Go 1.22):**

| Benchmark | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| MergeSmall | 743 | 1680 | 10 |
| MergeLarge (100 keys × 3 levels) | 107,752,271 | 193,104,036 | 1,060,804 |
| MergeWithProvenance | 805 | 1680 | 10 |
| ReferenceResolution (single) | 144 | 108 | 5 |
| ReferenceResolution (10 unique) | 1,386 | 1,584 | 45 |
| ReferenceResolution (10 cached) | 976 | 1,544 | 35 |
| ReferenceResolution (100 cached) | 7,851 | 15,528 | 308 |
| CompileEmpty | 13,342 | 1,984 | 18 |

### Golden Data Regeneration

Integration tests use golden files in `testdata/` for deterministic snapshot validation:

```bash
# Regenerate golden files after intentional changes
make update-golden

# Or use environment variable
GOLDEN_UPDATE=1 go test ./...
```

**Important:** Only regenerate golden files when test behavior intentionally changes. Review diffs carefully before committing.

### Test Organization

- **Unit tests** (`*_test.go`): Test individual functions and types; use `test/fakes` for provider mocking
- **Integration tests** (`test/integration_test.go`): Test end-to-end compilation with fixtures
- **Network integration tests** (`test/integration_network_test.go`): Require `//go:build integration` tag; excluded from default CI
- **Concurrency tests** (`test/concurrency_test.go`): Validate thread-safety with `-race` detector
- **Benchmarks** (`test/bench/compiler_bench_test.go`): Performance measurements for merge and reference resolution

### CI Coverage

The CI workflow (`.github/workflows/compiler-ci.yml`) enforces:
- ✅ 80% minimum test coverage
- ✅ Race detector passes on all tests
- ✅ Integration tests excluded from default runs
- ✅ Benchmark performance tracked as artifacts

### Test Helpers

The `test/fakes` package provides test doubles:

```go
import "github.com/autonomous-bits/nomos/libs/compiler/test/fakes"

// Create a fake provider for testing
fake := fakes.NewFakeProvider("test-provider")
fake.FetchResponses["config/vpc"] = map[string]any{
    "cidr": "10.0.0.0/16",
}

// Use in tests
registry := &mockRegistry{providers: map[string]compiler.Provider{
    "test": fake,
}}
```

## Example usage (compiler consumer)

```go
package main

import (
        "fmt"
        "github.com/autonomous-bits/nomos/libs/compiler"
)

func main() {
        snap, err := compiler.Compile(compiler.Options{Path: "./examples/dev"})
        if err != nil { panic(err) }
        fmt.Println(len(snap.Data))
}
```

## Testing strategy

- Unit tests for merge semantics, reference resolution and provider behavior (use fakes/mocks for providers).
- Integration tests compile fixtures in `testdata/` and assert snapshots. Keep tests deterministic and avoid external network calls.

## External Providers

The compiler supports external providers as separate executables that communicate via gRPC. This is the recommended approach for production use.

### Overview

External providers:
- Run as subprocesses managed by the compiler
- Implement the gRPC service contract defined in `libs/provider-proto`
- Are distributed via GitHub Releases or local paths
- Provide isolation, independent versioning, and language flexibility

### Using External Providers

The compiler's `internal/providerproc` package manages external provider processes:

```go
import (
	"github.com/autonomous-bits/nomos/libs/compiler"
	"github.com/autonomous-bits/nomos/libs/compiler/internal/providerproc"
)

// Create provider process manager
manager := providerproc.NewManager()
defer manager.Shutdown(context.Background())

// Get provider (starts subprocess if not already running)
provider, err := manager.GetProvider(
	ctx,
	"configs",                                    // alias
	".nomos/providers/file/1.0.0/darwin-arm64/provider", // binary path
	compiler.ProviderInitOptions{
		Alias:  "configs",
		Config: map[string]any{"directory": "./data"},
	},
)
if err != nil {
	return err
}

// Use provider through standard interface
data, err := provider.Fetch(ctx, []string{"database", "prod"})
```

### Provider Discovery and Installation

Users install providers with the `nomos init` CLI command:

```bash
# From GitHub Releases (default)
nomos init config.csl

# For local/testing scenarios: copy the provider binary into the `.nomos/providers/{owner}/{repo}/{version}/{os-arch}/provider`
# layout and then run `nomos init` to record it in the lockfile (see docs/examples/local-provider for details).
```

This creates:
- `.nomos/providers/` - Installed provider binaries
- `.nomos/providers.lock.json` - Version and checksum lock file

### Security: Binary Checksum Validation

**CRITICAL**: Provider binaries are validated using SHA256 checksums before execution.

The compiler enforces mandatory checksum verification for all provider binaries:

1. **Lockfile Requirement**: Every provider entry in `.nomos/providers.lock.json` must include a `checksum` field in the format `sha256:<hexdigest>`

2. **Validation Timing**: Checksums are verified when resolving provider binaries, before any process execution

3. **Failure Modes**:
   - **Missing Checksum**: Compilation fails with error directing user to run `nomos init`
   - **Checksum Mismatch**: Compilation fails with error indicating potential tampering
   - **Binary Missing**: Compilation fails with clear error message

4. **Security Properties**:
   - Uses cryptographically secure SHA256 hash function
   - Fails closed - any validation failure prevents execution
   - Detects binary tampering, corruption, or substitution attacks
   - Computed at provider installation time by `nomos init`

Example lockfile entry with checksum:
```json
{
  "providers": [
    {
      "alias": "configs",
      "type": "autonomous-bits/nomos-provider-file",
      "version": "0.2.0",
      "os": "darwin",
      "arch": "arm64",
      "path": ".nomos/providers/file/0.2.0/darwin-arm64/provider",
      "checksum": "sha256:a3c4f5e7d9b1c2a4f6e8d0b2c4a6f8e0d2b4c6a8e0f2d4b6c8a0e2f4d6b8a0e2"
    }
  ]
}
```

**Error Example**:
```
Error: provider binary for file has no checksum in lockfile - refusing to execute (security risk); run 'nomos init' to regenerate lockfile with checksums
```

### Documentation

- **For users**: See [docs/examples](../../docs/examples/) for usage examples
- **For provider authors**: See [docs/guides/provider-authoring-guide.md](../../docs/guides/provider-authoring-guide.md)
- **Architecture**: See [docs/architecture/nomos-external-providers-feature-breakdown.md](../../docs/architecture/nomos-external-providers-feature-breakdown.md)
- **gRPC contract**: See [libs/provider-proto](../provider-proto/README.md)

## Versioning

- Tag releases as `libs/compiler/vX.Y.Z` and follow semantic versioning. Breaking API changes require a major version bump.

## Notes / Follow-ups

- Consider adding a small semantic validation pass (duplicate key detection, unresolved references) that runs after parsing but before provider resolution.
- The parser's `SourceSpan` semantics (1-indexed, byte-based columns, inclusive end column) are important when slicing source text for display — prefer using `parser.FormatParseError` for consistent caret rendering.
