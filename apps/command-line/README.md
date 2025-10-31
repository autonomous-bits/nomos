# Nomos CLI

The Nomos CLI is the user-facing tool that compiles Nomos `.csl` scripts into deterministic, serializable configuration snapshots.
It is a thin wrapper around the compiler library (`libs/compiler`) and is responsible for argument parsing, wiring provider adapters, marshaling output, and mapping compiler results to exit codes.

> **Product Requirements:** See [PRD issue #35](https://github.com/autonomous-bits/nomos/issues/35) for complete feature specification and design decisions.

> **Compiler Documentation:** For compiler-level details, semantics, and provider adapters, see [`libs/compiler`](../../libs/compiler/README.md).

## Installation

### Prerequisites

- Go 1.22 or later
- macOS, Linux, or Windows (tested primarily on macOS)

### Build from Source

From the repository root:

```bash
cd apps/command-line
go build -o ../../bin/nomos ./cmd/nomos
```

The binary will be available at `bin/nomos` in the repository root.

### Verify Installation

```bash
./bin/nomos --help
```

You should see the CLI help output.

## Quick Start

Compile a single .csl file:

```bash
nomos build -p testdata/simple.csl
```

Compile a directory to YAML:

```bash
nomos build -p testdata/configs -f yaml -o snapshot.yaml
```

Compile with variable substitution:

```bash
nomos build -p testdata/with-vars.csl --var region=us-west --var env=dev
```

## Network and Safety Defaults

**The CLI does NOT make network calls by default** (offline-first behavior).

- Provider fetches only occur when provider types are explicitly configured and required by your `.csl` scripts
- This ensures safe, reproducible builds in CI environments without network dependencies
- Use `--allow-missing-provider` to tolerate provider fetch failures if needed
- Control network behavior with `--timeout-per-provider` and `--max-concurrent-providers` flags

This design ensures deterministic, hermetic builds by default.

## CLI commands & flags

The primary commands provided by the CLI are:

- `build` — compile a file or directory of Nomos scripts into a snapshot
- `init` — discover and install provider dependencies

### `nomos build`

Compile Nomos scripts into configuration snapshots.

Relevant flags:

- `--path, -p` (required): path to a `.csl` file or a folder containing `.csl` files
- `--format, -f`: output format; one of `json`, `yaml`, `hcl` (default: `json`)
- `--allow-missing-provider`: allow missing provider fetches (compiler.Options.AllowMissingProvider)
- `--timeout-per-provider`: duration string (e.g., `5s`) used for per-provider fetch timeout
- `--max-concurrent-providers`: integer limit for concurrent provider fetches

### `nomos init`

Discover provider requirements from `.csl` files and install provider binaries.

Usage:

```bash
nomos init [flags] <file.csl> [<file2.csl> ...]
```

Relevant flags:

- `--from alias=path`: specify local provider binary path (can be repeated for multiple providers)
- `--dry-run`: preview actions without installing
- `--force`: overwrite existing providers/lockfile
- `--os`: override target OS (default: runtime OS)
- `--arch`: override target architecture (default: runtime arch)
- `--upgrade`: force upgrade to latest versions

Example:

```bash
# Install provider from local binary
nomos init --from configs=/path/to/nomos-provider-file config.csl

# Preview what would be installed
nomos init --dry-run config.csl
```

The init command creates:
- `.nomos/providers/{type}/{version}/{os-arch}/provider` — installed binaries
- `.nomos/providers.lock.json` — lock file with resolved versions and paths

## Commands

### build

Compile Nomos .csl files into a configuration snapshot.

**Usage:**
```bash
nomos build [options]
```

**Options:**

- `-p, --path <path>` — Path to .csl file or directory (required)
- `-f, --format <format>` — Output format: json, yaml, or hcl (default: json)
- `-o, --out <file>` — Write output to file (default: stdout)
- `--var <key=value>` — Variable substitution (repeatable)
- `--strict` — Treat warnings as errors
- `--allow-missing-provider` — Allow missing provider fetches
- `--timeout-per-provider <duration>` — Timeout for each provider fetch (e.g., 5s, 1m)
- `--max-concurrent-providers <int>` — Maximum concurrent provider fetches
- `--verbose` — Enable verbose logging
- `-h, --help` — Show help

**Examples:**

```bash
# Compile a single file to JSON (stdout)
nomos build -p testdata/simple.csl

# Compile directory to YAML file
nomos build -p testdata/configs -f yaml -o snapshot.yaml

# Compile with variables
nomos build -p testdata/with-vars.csl --var region=eu-west --var env=production

# Strict mode (warnings cause failure)
nomos build -p testdata/configs --strict
```

### Using References with Path Navigation

The CLI supports references to access specific values from provider sources using dot notation:

**Syntax:** `reference:{alias}:{filename}.{path.to.value}`

**Example source file with references:**

```nomos
source:
  alias: 'configs'
  type: 'file'
  directory: './shared-configs'

app:
  name: 'my-app'
  # Reference specific values using dot notation
  storage_type: reference:configs:storage.storage.type
  bucket: reference:configs:storage.buckets.primary
  encryption: reference:configs:storage.encryption.algorithm
```

Given `shared-configs/storage.csl`:
```nomos
storage:
  type: 's3'
  region: 'us-west-2'
  
buckets:
  primary: 'my-app-data'
  backup: 'my-app-backup'
  
encryption:
  enabled: true
  algorithm: 'AES256'
```

When compiled, the references resolve to:
```json
{
  "data": {
    "app": {
      "name": "my-app",
      "storage_type": "s3",
      "bucket": "my-app-data",
      "encryption": "AES256"
    }
  }
}
```

**Path Navigation Rules:**
- First component after alias = filename (without `.csl` extension)
- Remaining components = dot-separated path through nested data
- Provider fetches the file, parses it, and navigates to the requested path

See `libs/compiler/providers/file/README.md` for detailed provider documentation.

````
```

**Exit Codes:**
- `0` — Success
- `1` — Compilation errors or runtime failure
- `2` — Invalid arguments or usage error

## Architecture

The CLI is a thin layer over the compiler library:

```
CLI (apps/command-line)
  └── uses → Compiler (libs/compiler)
               └── uses → Parser (libs/parser)
```

### Key Components

- `cmd/nomos/main.go` — Entry point and command routing
- `cmd/nomos/build.go` — Build command implementation
- `cmd/nomos/help.go` — Help text and usage
- `internal/flags/` — Flag parsing and validation
- `internal/options/` — Compiler options builder with provider wiring

### Options Builder

The `internal/options` package provides utilities for building `compiler.Options` from CLI flags with proper dependency injection for testability.

**Key Functions:**

- `NewProviderRegistries()` — Creates default empty provider registries (no-network behavior)
- `BuildOptions(params BuildParams)` — Constructs compiler.Options from CLI parameters

**Provider Wiring:**

By default, the CLI creates empty provider registries ensuring no network calls are made unless explicitly configured. This aligns with the PRD requirement for safe defaults.

```go
// Default no-network behavior
providerRegistry, providerTypeRegistry := options.NewProviderRegistries()

opts, err := options.BuildOptions(options.BuildParams{
    Path:                   "/path/to/config.csl",
    Vars:                   []string{"env=prod", "region=us-west"},
    TimeoutPerProvider:     "5s",
    MaxConcurrentProviders: 10,
    AllowMissingProvider:   false,
    ProviderRegistry:       providerRegistry,
    ProviderTypeRegistry:   providerTypeRegistry,
})
```

**Custom Provider Injection (for testing):**

```go
// Create custom provider registry for testing
customPR := compiler.NewProviderRegistry()
customPR.Register("test-provider", func(opts compiler.ProviderInitOptions) (compiler.Provider, error) {
    return &TestProvider{}, nil
})

opts, err := options.BuildOptions(options.BuildParams{
    Path:             "/test/path",
    ProviderRegistry: customPR,
    // ... other params
})
```

**CLI Flag Mapping to compiler.Options:**

| CLI Flag | compiler.Options Field | Type | Notes |
|----------|------------------------|------|-------|
| `--path, -p` | `Path` | string | Input file or directory |
| `--var key=value` | `Vars["key"]` | any | Repeatable; creates map |
| `--timeout-per-provider` | `Timeouts.PerProviderFetch` | duration | Parsed from duration string |
| `--max-concurrent-providers` | `Timeouts.MaxConcurrentProviders` | int | Default 0 (unlimited) |
| `--allow-missing-provider` | `AllowMissingProvider` | bool | Default false |
| N/A (created by CLI) | `ProviderRegistry` | interface | Empty by default |
| N/A (created by CLI) | `ProviderTypeRegistry` | interface | Empty by default |

### File Discovery

When the `--path` argument is a directory, the CLI recursively discovers all `.csl` files using deterministic UTF-8 lexicographic ordering. This ordering is critical for reproducible builds because it determines the sequence in which files are compiled and affects the final configuration due to last-wins merge semantics.

**Ordering Algorithm:**

1. Recursively traverse the directory tree
2. Filter for files with `.csl` extension only
3. Sort full paths using UTF-8 lexicographic comparison (Go's `sort.Strings`)
4. Return absolute paths in sorted order

**Properties:**

- **Deterministic**: Same input directory always produces the same file order
- **Cross-platform**: Uses UTF-8 comparison, stable across operating systems
- **Symlink-aware**: Follows symlinks but detects and prevents loops
- **Recursive**: Discovers files in nested subdirectories

**Examples:**

Given this directory structure:
```
configs/
  3-database.csl
  1-base.csl
  2-network.csl
  subdir/
    4-logging.csl
```

Files will be processed in this order:
1. `configs/1-base.csl`
2. `configs/2-network.csl`
3. `configs/3-database.csl`
4. `configs/subdir/4-logging.csl`

The numeric prefixes ensure predictable ordering. Without them, lexicographic order would apply (e.g., `a.csl` < `b.csl` < `z.csl`).

**Special Cases:**

- **Empty directory**: Returns an error with exit code 2
- **No .csl files found**: Returns an error with exit code 2
- **Unreadable files**: Returns an error with file path and permissions issue
- **Symlink loops**: Detected and skipped gracefully
- **Single file**: When `--path` is a file, no discovery occurs; that single file is used

**Note:** The compiler library itself currently discovers files in a single directory level only. The CLI's traverse package supports recursive discovery for future compiler enhancements.

### Output Formats and Serialization

The CLI supports multiple output formats via the `--format` flag. All formats aim for deterministic output to ensure byte-for-byte identical results for identical inputs (critical for CI reproducibility).

**Supported Formats:**

- `json` (default) — Canonical JSON with deterministic key ordering
- `yaml` — YAML serialization (not yet implemented, returns error)
- `hcl` — HCL serialization (not yet implemented, returns error)

**JSON Format (Canonical)**

The JSON serializer implements canonical serialization that guarantees:

1. **Deterministic key ordering**: Map keys are sorted alphabetically at all nesting levels
2. **UTF-8 normalization**: Invalid UTF-8 sequences are replaced with `�`
3. **Consistent structure**: Data and metadata sections maintain stable ordering
4. **Timestamp variance**: Note that `metadata.start_time` and `metadata.end_time` will vary between runs as they capture actual compilation timestamps

**Determinism Guarantees:**

- The `data` section is byte-for-byte identical for identical logical inputs
- The `metadata` structure and key ordering are deterministic
- Only timestamp values in metadata will differ between runs
- For testing determinism, compare parsed `data` sections or use stable test inputs

**Example JSON Output:**

```json
{
  "data": {
    "alpha": {
      "value": "first"
    },
    "middle": {
      "nested": "value"
    },
    "zebra": {
      "value": "last"
    }
  },
  "metadata": {
    "end_time": "2025-10-26T20:00:00Z",
    "errors": [],
    "input_files": [
      "/path/to/config.csl"
    ],
    "per_key_provenance": {
      "alpha": {
        "provider_alias": "",
        "source": "/path/to/config.csl"
      },
      "middle": {
        "provider_alias": "",
        "source": "/path/to/config.csl"
      },
      "zebra": {
        "provider_alias": "",
        "source": "/path/to/config.csl"
      }
    },
    "provider_aliases": [],
    "start_time": "2025-10-26T20:00:00Z",
    "warnings": []
  }
}
```

Note the sorted key order: `alpha` < `middle` < `zebra` in data and all nested maps.

**Output Destination:**

- Default: Writes to stdout
- With `--out` flag: Writes to specified file path
- Directories are created automatically if they don't exist
- Non-writable paths result in exit code 2 with clear error message

**Examples:**

```bash
# JSON to stdout (default)
nomos build -p config.csl

# JSON to file
nomos build -p config.csl -o output.json

# Create nested directories
nomos build -p config.csl -o build/snapshots/output.json
```

**Implementation Details:**

The serializer is located in `internal/serialize` and provides:
- `ToJSON(snapshot)` — Canonical JSON serialization
- `ToYAML(snapshot)` — YAML serialization (stub)
- `ToHCL(snapshot)` — HCL serialization (stub)

See `internal/serialize/serialize_test.go` for comprehensive determinism tests.

### Compilation Flow

1. Parse CLI flags and validate
2. Convert flags to `compiler.Options`
3. Create provider registry (empty by default — no network)
4. Call `compiler.Compile(ctx, opts)`
5. Handle diagnostics (errors and warnings)
6. Serialize snapshot data to requested format
7. Write to stdout or file
8. Exit with appropriate code

## Development

### Prerequisites

- Go 1.22 or later
- Access to `libs/compiler` and `libs/parser` via workspace

### Building

```bash
cd apps/command-line
go build -o ../../bin/nomos ./cmd/nomos
```

### Running Tests

```bash
# Unit tests
go test ./internal/...

# Integration tests
go test ./test

# All tests with race detector
go test -race ./...
```

### Adding New Flags

1. Add field to `BuildFlags` struct in `internal/flags/flags.go`
2. Wire flag in `Parse()` function  
3. Add field to `options.BuildParams` in `internal/options/options.go`
4. Map flag value to `compiler.Options` field in `BuildOptions()` function
5. Add test cases to `internal/flags/flags_test.go`
6. Add test cases to `internal/options/options_test.go`
7. Update help text in `cmd/nomos/help.go`
8. Update flag mapping table in README

## Compiler Contract

The CLI depends on `libs/compiler` with this contract:

```go
func Compile(ctx context.Context, opts Options) (Snapshot, error)
```

**Key Option Fields:**

- `Path string` — Input file or directory
- `ProviderRegistry` — Runtime provider registry (required)
- `ProviderTypeRegistry` — Provider type constructors (optional)
- `Vars map[string]any` — Variable substitutions
- `Timeouts.PerProviderFetch time.Duration` — Per-provider timeout
- `Timeouts.MaxConcurrentProviders int` — Concurrency limit
- `AllowMissingProvider bool` — Tolerate missing providers

**Snapshot Structure:**

- `Data map[string]any` — Compiled configuration
- `Metadata.Errors []string` — Fatal errors
- `Metadata.Warnings []string` — Non-fatal warnings
- `Metadata.InputFiles []string` — Source files processed
- `Metadata.ProviderAliases []string` — Providers used
- `Metadata.PerKeyProvenance map[string]Provenance` — Value origins

## Future Enhancements

Planned for future releases:

- YAML and HCL output formats (stubs exist)
- Provider credential handling
- Remote provider support with explicit opt-in
- Additional commands (`validate`, `fmt`, `init`)
- Telemetry and usage analytics
- Performance benchmarking targets

## Testing

The CLI has comprehensive test coverage including unit tests, integration tests, and determinism tests.

### Running Tests Locally

**Unit tests (fast, ~2 seconds):**
```bash
cd apps/command-line
go test -v ./internal/... ./cmd/...
```

**Unit tests with coverage:**
```bash
cd apps/command-line
go test -v -coverprofile=coverage.out ./internal/... ./cmd/...
go tool cover -html=coverage.out  # View coverage in browser
```

**Unit tests with race detector:**
```bash
cd apps/command-line
go test -v -race ./internal/... ./cmd/...
```

**Integration tests (longer, ~5-10 seconds):**
```bash
cd apps/command-line
go test -v -timeout 10m ./test/...
```

**Determinism test only:**
```bash
cd apps/command-line
go test -v -timeout 15m -run TestDeterministicJSON ./test/...
```

**All tests:**
```bash
cd apps/command-line
go test -v ./...
```

### Test Structure

The test suite is organized as follows:

- `internal/*/` — Unit tests for each internal package (flags, options, diagnostics, serialize, traverse)
  - `flags_test.go` — Flag parsing validation (93.3% coverage)
  - `options_test.go` — Compiler options builder (100% coverage)
  - `diagnostics_test.go` — Error/warning formatting (94.6% coverage)
  - `serialize_test.go` — JSON/YAML/HCL serialization and determinism (75% coverage)
  - `traverse_test.go` — File discovery and ordering (82.9% coverage)

- `test/` — Integration tests that build and invoke the CLI binary
  - `integration_test.go` — End-to-end CLI invocation tests
  - `exitcode_integration_test.go` — Exit code validation
  - `options_integration_test.go` — Options building integration
  - `traverse_integration_test.go` — File discovery integration
  - `determinism_integration_test.go` — Byte-for-byte reproducibility test
  - `help_test.go` — Help text content and consistency validation

### Running Examples

All examples in this README reference files in `testdata/` and can be executed directly:

```bash
# From the apps/command-line directory
nomos build -p testdata/simple.csl
nomos build -p testdata/configs -f yaml
nomos build -p testdata/with-vars.csl --var region=us-east --var env=staging
```

These commands work offline and produce deterministic output.

### Coverage Goals

- **Minimum 80% coverage** for all packages (currently exceeding threshold)
- **100% coverage** for critical paths (options builder, flag parsing)
- All tests must be **offline-by-default** (no network calls)

### Continuous Integration

The CLI has dedicated CI workflows in `.github/workflows/cli-ci.yml`:

**CI Jobs:**

1. **Unit Tests** (fast, ~2 min)
   - Runs unit tests with race detector
   - Verifies 80%+ code coverage threshold
   - Uploads coverage to Codecov

2. **Integration Tests** (longer, ~5 min)
   - Builds CLI binary
   - Runs full integration test suite
   - Validates CLI invocation and exit codes

3. **Determinism Test** (separate job, ~5 min)
   - Runs deterministic JSON output test
   - Validates byte-for-byte reproducibility across multiple runs
   - Critical for CI/CD pipeline stability

4. **Lint** (fast, ~1 min)
   - Runs golangci-lint
   - Enforces Go coding standards
   - Validates code quality

**CI Triggers:**
- Push to `main` branch
- Pull requests to `main`
- Changes to CLI code, compiler, parser, or workflow files

### Test-Driven Development

All CLI development follows TDD (Test-Driven Development):

1. **Red**: Write a failing test for new behavior
2. **Green**: Implement minimal code to pass the test
3. **Refactor**: Improve design while keeping tests green

Example workflow:
```bash
# 1. Write failing test
go test -v ./internal/flags -run TestNewFlag
# FAIL: expected behavior not implemented

# 2. Implement feature
# ... edit flags.go ...

# 3. Verify test passes
go test -v ./internal/flags -run TestNewFlag
# PASS

# 4. Run full suite
go test -v ./...
# All tests PASS
```

### Troubleshooting Tests

**Issue: Integration tests fail to build CLI binary**
```bash
# Ensure Go workspace is configured
go work sync

# Build manually to debug
cd apps/command-line
go build -o ../../bin/nomos ./cmd/nomos
```

**Issue: Determinism test fails**
- Ensure no timestamps or random data in output
- Check that JSON serialization uses sorted keys
- Verify test fixture is stable

**Issue: Coverage below threshold**
```bash
# Generate detailed coverage report
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | grep -v 100.0%
# Shows lines not covered
```

**Issue: Race detector failures**
```bash
# Run with race detector for detailed output
go test -v -race ./internal/...
# Fix data races before committing
```

### Adding New Tests

When adding new functionality:

1. **Start with a unit test** in the appropriate `internal/*/` package
2. **Follow existing patterns** (table-driven tests, subtests with t.Run)
3. **Test happy path, sad path, and edge cases**
4. **Add integration test** if feature spans CLI invocation
5. **Verify coverage** doesn't drop below 80%

Example unit test template:
```go
func TestNewFeature_Scenario(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    Result
        wantErr bool
    }{
        {"valid input", "test", Expected{}, false},
        {"invalid input", "", nil, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := NewFeature(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

## Contributing

See repository-level `CONTRIBUTING.md` and `docs/architecture/go-monorepo-structure.md` for guidance on:

- Code standards
- Testing requirements  
- Changelog maintenance
- PR process

## License

See repository root for license information.

opts := compiler.Options{
    Path: path,
    ProviderRegistry: registry,
    ProviderTypeRegistry: typeRegistry,
    Vars: parsedVars,
    Timeouts: compiler.OptionsTimeouts{
        PerProviderFetch: parsedTimeout,
        MaxConcurrentProviders: parsedMaxConcurrent,
    },
    AllowMissingProvider: allowMissing,
}

snap, err := compiler.Compile(ctx, opts)
if err != nil {
    // print error and exit non-zero
}

// marshal snap.Data to format and write to stdout/file
// print diagnostics from snap.Metadata.Errors / Warnings
```

## Exit-code mapping

The CLI follows strict exit-code semantics for pipeline integration:

- **Exit code 0**: Successful compilation (or warnings only without `--strict`)
- **Exit code 1**: Compilation failed with errors, or warnings in `--strict` mode
- **Exit code 2**: Invalid usage or bad arguments

## Error and diagnostic handling

Errors and warnings are printed to stderr in human-friendly format with file:line:col information when available from compiler diagnostics.

The compiler populates `Metadata.Errors` and `Metadata.Warnings` with formatted messages including source locations. The CLI's diagnostics package formats these for terminal output:

```
config.csl:10:5: error: unresolved reference to provider 'db'
   10 |   host: reference:db:host
      |         ^
```

Features:
- File, line, and column information from parser/compiler diagnostics
- Context snippets with caret markers pointing to error locations
- Consistent `file:line:col: severity: message` format for machine parsing
- Optional color output support (future enhancement)

Use `--strict` to treat warnings as errors (causes exit code 1).

## External Providers

Nomos uses external providers as separate executables for fetching configuration data. This is the recommended approach for production use.

### Overview

External providers:
- Are standalone executables distributed via GitHub Releases or local paths
- Communicate with the compiler via gRPC
- Support any language with gRPC support (Go, Python, Node.js, etc.)
- Provide isolation, independent versioning, and security boundaries

### Installing Providers

Use `nomos init` to install providers before building:

```bash
# Install from GitHub Releases (automatic download)
nomos init config.csl

# Install from local binary
nomos init --from configs=/path/to/provider config.csl

# Install with custom OS/arch
nomos init --os linux --arch amd64 config.csl
```

This creates:
- `.nomos/providers/` — Installed provider binaries
- `.nomos/providers.lock.json` — Lock file with versions and checksums

### Building with Providers

Once providers are installed, use `nomos build` as normal:

```bash
nomos build config.csl
```

The compiler will:
1. Read the lock file to locate provider binaries
2. Start provider subprocesses as needed
3. Call provider RPCs to fetch data
4. Shut down providers after compilation

### Workflow Example

Complete workflow from scratch:

```bash
# 1. Create configuration using providers
cat > config.csl << 'EOF'
source file as configs {
  version = "1.0.0"
  config = {
    directory = "./data"
  }
}

config = import configs["database"]["prod"]
EOF

# 2. Initialize providers
nomos init config.csl
# → Downloads nomos-provider-file from GitHub
# → Creates .nomos/providers/ and lock file

# 3. Build configuration
nomos build config.csl -o output.json
# → Starts file provider subprocess
# → Fetches data via gRPC
# → Compiles to output.json

# 4. Subsequent builds use cached provider
nomos build config.csl
# → Reuses already-installed provider (fast)
```

### Upgrading Providers

To upgrade to newer provider versions:

```bash
# Update version in config.csl
sed -i 's/version = "1.0.0"/version = "1.1.0"/' config.csl

# Re-initialize with --upgrade flag
nomos init --upgrade config.csl
```

### Documentation and Examples

- **Usage examples**: See [docs/examples](../../docs/examples/) for step-by-step guides
- **Provider authoring**: See [docs/guides/provider-authoring-guide.md](../../docs/guides/provider-authoring-guide.md)
- **Migration guide**: See [docs/guides/external-providers-migration.md](../../docs/guides/external-providers-migration.md)
- **Architecture**: See [docs/architecture/nomos-external-providers-feature-breakdown.md](../../docs/architecture/nomos-external-providers-feature-breakdown.md)

## Development notes

- Use the Go workspace at the repo root for local development: `go work use ./apps/command-line ./libs/compiler ./libs/parser`.
- Keep CLI code focused on wiring, argument parsing, and I/O; all language semantics live in `libs/compiler` and `libs/parser`.

## CHANGELOG and releases

- The CLI follows semantic versioning. Tag releases as `apps/command-line/vX.Y.Z` and update `CHANGELOG.md` with UX/flag changes.

---

If you need a sample provider registry implementation or example wiring for `ProviderTypeRegistry`, see `libs/compiler` docs and the `providers` adapters under `libs/compiler/providers`.
