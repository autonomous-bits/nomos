# CLI Module Agent

## Purpose
Specialized agent for `apps/command-line` - handles the Nomos CLI tool, command structure, flag handling, output formatting, and user interaction. This is the user-facing executable that wraps the compiler and parser libraries to provide a cohesive command-line experience.

## Module Context
- **Path**: `apps/command-line`
- **Binary**: `nomos`
- **Framework**: Standard Go `flag` package (not Cobra)
- **Key Commands**: `build`, `init`, `help`
- **Entry Point**: `cmd/nomos/main.go`
- **Dependencies**: `libs/compiler`, `libs/parser`, `libs/provider-downloader`

## Key Files
- `cmd/nomos/main.go` — CLI entry point with command routing
- `cmd/nomos/build.go` — Build command implementation
- `cmd/nomos/init.go` — Init command implementation (provider installation)
- `cmd/nomos/help.go` — Help text and usage formatting
- `internal/flags/` — CLI flag parsing and validation
- `internal/options/` — Compiler options builder from CLI flags
- `internal/serialize/` — Deterministic output serialization (JSON, YAML, HCL)
- `internal/diagnostics/` — Error/warning formatter for user output
- `internal/traverse/` — Deterministic file discovery (UTF-8 lexicographic ordering)
- `internal/initcmd/` — Init command logic (provider discovery and installation)

## Delegation Instructions
For general Go questions, **consult go-expert.md**  
For compiler semantics, **consult compiler-module.md**  
For testing questions, **consult testing-expert.md**  
For provider-related questions, **consult provider-expert.md**

## CLI Architecture Patterns

### Command Routing Structure
The CLI uses a simple switch-based command routing system in `main.go`:
- Entry point: `main()` → `run(args)`
- Command router: `switch` on first argument (`build`, `init`, `help`)
- Command handlers: `runBuild()`, `runInit()` delegate to command-specific functions
- Exit codes: `exitSuccess (0)`, `exitError (1)`, `exitUsageErr (2)`

**No Cobra framework** — uses standard `flag` package for simplicity and transparency.

### Command Hierarchy
```
nomos
├── build       — compile .csl files to snapshot
├── init        — discover and install provider dependencies
└── help        — show usage information
```

Each command has:
- Dedicated file in `cmd/nomos/` (e.g., `build.go`, `init.go`)
- Help text function (e.g., `printBuildHelp()`)
- Argument parsing via `internal/flags` or `internal/initcmd`

### Configuration File Handling
- **Input**: `.csl` (Nomos script) files
- **Provider Config**: `.nomos/providers.lock.json` — lockfile for provider versions and paths
- **Provider Storage**: `.nomos/providers/{type}/{version}/{os-arch}/provider` — installed binaries
- **Output**: JSON/YAML/HCL snapshots (to stdout or file via `-o`)

#### File Discovery
Uses `internal/traverse` package for deterministic ordering:
- UTF-8 lexicographic ordering of `.csl` files
- Depth-first directory traversal
- Ensures reproducible builds across environments

### Build Command Flow
```
build.go
  ↓
flags.Parse() → BuildFlags
  ↓
options.BuildOptions() → compiler.Options
  ↓
compiler.Compile(ctx, opts) → Snapshot
  ↓
diagnostics.Formatter → warnings/errors to stderr
  ↓
serialize.ToJSON/ToYAML/ToHCL → output
  ↓
write to stdout or file
```

### Init Command Flow
```
init.go
  ↓
initcmd.Parse() → InitOptions
  ↓
initcmd.Execute()
  ↓
parse .csl files → extract provider requirements
  ↓
provider-downloader → fetch binaries from GitHub Releases
  ↓
write .nomos/providers.lock.json
  ↓
install to .nomos/providers/{type}/{version}/{os-arch}/
```

## Output Formatting

### Serialization (internal/serialize)
Three deterministic formats:
- **JSON**: `serialize.ToJSON()` — canonical, sorted keys
- **YAML**: `serialize.ToYAML()` — deterministic ordering
- **HCL**: `serialize.ToHCL()` — Terraform-compatible

All serializers:
- Sort map keys alphabetically
- Use consistent indentation
- Produce byte-identical output for identical input

### Diagnostics (internal/diagnostics)
Compiler errors and warnings formatted for human consumption:
```
diagnostics.Formatter
  ├── PrintWarnings(w, warnings)
  └── PrintErrors(w, errors)
```

Format:
```
Warning: [warning message]
  at file.csl:line:column

Error: [error message]
  at file.csl:line:column
```

**Future enhancement**: Color support via `--color` flag.

## Error Message Patterns

### CLI-Level Errors
- Usage errors: print error + help text → exit code 2
- Flag parsing errors: print error + help text → exit code 2

### Compilation Errors
- Collect from `snapshot.Metadata.Errors`
- Print via `diagnostics.Formatter`
- Exit code 1 if errors exist

### Strict Mode
- Enabled via `--strict` flag
- Treats warnings as errors
- Exit code 1 if warnings exist

### Network/Provider Errors
- Default: fail on missing providers (safe, deterministic builds)
- `--allow-missing-provider`: tolerate provider fetch failures
- `--timeout-per-provider`: control network timeouts (e.g., `5s`, `1m`)
- `--max-concurrent-providers`: limit concurrent fetches (default: 4)

## Provider Commands

### `nomos init`
Install provider dependencies from `.csl` files:
```bash
nomos init config.csl              # install providers
nomos init --dry-run config.csl    # preview without installing
nomos init --force config.csl      # overwrite existing
nomos init --upgrade config.csl    # force upgrade to latest
```

Flags:
- `--dry-run`: preview actions without installing
- `--force`: overwrite existing providers/lockfile
- `--os`, `--arch`: override target platform (for cross-platform installs)
- `--upgrade`: force upgrade to latest versions

Creates:
- `.nomos/providers.lock.json` — lockfile with resolved versions
- `.nomos/providers/{type}/{version}/{os-arch}/provider` — binaries

### Provider Discovery
Parser extracts `source:` declarations from `.csl` files:
```nomos
source:
  alias: 'aws'
  type: 'autonomous-bits/nomos-provider-aws'
  version: '1.2.3'
```

Init command:
1. Parses all input `.csl` files
2. Collects unique provider requirements
3. Fetches from GitHub Releases via `libs/provider-downloader`
4. Writes lockfile and installs binaries

## Build Tags
- **None currently used** — CLI is platform-agnostic
- Provider binaries are platform-specific (handled by `provider-downloader`)

## Module Dependencies

### Direct Dependencies
- `libs/compiler` — compiles `.csl` to snapshots
- `libs/parser` — parses `.csl` syntax
- `libs/provider-downloader` — fetches provider binaries from GitHub

### Transitive Dependencies
- `libs/provider-proto` — provider RPC interface

### Dependency Flow
```
CLI (apps/command-line)
  ↓
compiler (libs/compiler)
  ↓
parser (libs/parser)
```

### Workspace Integration
- Uses repository-level `go.work` for local development
- No `replace` directives in `go.mod` (workspace handles local linking)
- Tagged as `apps/command-line/v1.x.x` for releases

## Common Tasks

### 1. Adding New CLI Commands
1. Create `cmd/nomos/[command].go` with command implementation
2. Add command case to `run()` switch in `main.go`
3. Implement `run[Command]()` handler function
4. Add help text function (e.g., `print[Command]Help()`)
5. Add command to `printHelp()` in `help.go`
6. Write tests in `cmd/nomos/[command]_test.go`

**Example:**
```go
// cmd/nomos/validate.go
func validateCommand(args []string) error {
    // parse flags
    // call compiler with validation-only mode
    // print diagnostics
}

// cmd/nomos/main.go
case "validate":
    return runValidate(commandArgs)
```

### 2. Adding New Flags
1. Add field to `BuildFlags` struct in `internal/flags/flags.go`
2. Register flag in `Parse()` function
3. Pass to compiler via `options.BuildOptions()` in `internal/options/options.go`
4. Update help text in `cmd/nomos/help.go`
5. Add tests in `internal/flags/flags_test.go`

**Example:**
```go
// internal/flags/flags.go
type BuildFlags struct {
    // ... existing fields
    ColorOutput bool  // new flag
}

// Parse()
fs.BoolVar(&f.ColorOutput, "color", false, "Enable colored output")
```

### 3. Improving Help Text and User Experience
- Help text lives in `cmd/nomos/help.go`
- Each command has dedicated help function (e.g., `printBuildHelp()`)
- Follow existing format: usage → flags → examples
- Keep help concise; link to docs for details

### 4. Output Format Enhancements
- Serializers in `internal/serialize/serialize.go`
- Implement new `To[Format]()` function
- Add format to `serializeSnapshot()` switch in `build.go`
- Ensure deterministic output (sorted keys, consistent indentation)
- Add tests in `internal/serialize/serialize_test.go`

### 5. Flag Handling and Validation
- Flag parsing in `internal/flags/flags.go`
- Validation in `flags.Parse()` or `options.BuildOptions()`
- Return errors with clear messages
- Print help text on validation failure
- Use `exitUsageErr` for invalid usage

### 6. Configuration File Management
- `.nomos/providers.lock.json` managed by `internal/initcmd`
- Provider binaries installed to `.nomos/providers/`
- Lockfile format: JSON with `providers` array
- Each entry: `type`, `version`, `path`, `os`, `arch`, `checksum`

## Nomos-Specific Details

### Offline-First Philosophy
**Network calls are opt-in, not default:**
- Provider fetches only occur when required by `.csl` scripts
- `--allow-missing-provider` tolerates fetch failures
- Default: fail fast if providers unavailable (deterministic builds)
- CI-friendly: no unexpected network dependencies

### Deterministic Builds
1. **File ordering**: UTF-8 lexicographic (via `internal/traverse`)
2. **Serialization**: sorted keys, canonical formatting
3. **Provider versions**: pinned in lockfile
4. **Reproducibility**: identical input → identical output (bit-for-bit)

### Variable Substitution
CLI supports `--var key=value` flags:
- Passed to compiler as `vars` map
- Used for parameterized configurations
- Example: `nomos build -p config.csl --var env=prod --var region=us-west`

### Strict Mode
`--strict` flag treats warnings as errors:
- Useful for CI pipelines
- Enforces clean compilation
- Exit code 1 if warnings exist

### Reference Syntax
Supports dot-notation references to provider sources:
```nomos
app:
  config: reference:alias:filename.path.to.value
```

Compiled by `libs/compiler` (not CLI-specific), but important for user-facing docs and examples.

## Testing Patterns

### Unit Tests
- Command logic: `cmd/nomos/*_test.go`
- Flag parsing: `internal/flags/flags_test.go`
- Serialization: `internal/serialize/serialize_test.go`
- Diagnostics: `internal/diagnostics/diagnostics_test.go`

### Integration Tests
- End-to-end CLI execution: `test/` directory
- Build binary and invoke with test inputs
- Verify output matches expected snapshots
- Test provider installation flows

### Hermetic Testing
- Use `testdata/` for test fixtures
- Mock provider registries where needed
- Avoid network calls in tests (use fixtures)
- See `internal/initcmd/init_hermetic_test.go` for examples

## Best Practices

### CLI Code
- Keep command handlers thin (delegate to libraries)
- Parse args → validate → call compiler → format output
- Application logic belongs in `libs/`, not CLI

### Error Messages
- Clear, actionable messages for users
- Include file:line:column for compiler errors
- Suggest fixes where possible
- Example: "Error: unknown flag --foo. Did you mean --format?"

### Output Conventions
- Diagnostics (errors/warnings) → stderr
- Compilation results → stdout (or file via `-o`)
- Status messages (verbose mode) → stderr
- Exit codes: 0=success, 1=error, 2=usage error

### Flag Design
- Short flags for common options (`-p`, `-f`, `-o`)
- Long flags for all options (`--path`, `--format`, `--out`)
- Boolean flags default to false
- Duration flags use Go duration format (`5s`, `1m30s`)

### Workspace Development
- Use `go.work` at repository root (already configured)
- No replace directives in `go.mod`
- Build from repository root: `cd apps/command-line && go build -o ../../bin/nomos ./cmd/nomos`
- Run: `./bin/nomos [command] [flags]`

## Future Enhancements
- **Color output**: `--color` flag for diagnostics
- **Progress indicators**: for long-running provider fetches
- **Shell completions**: bash/zsh/fish autocompletion
- **Config file**: `.nomos.yaml` for default flags
- **Watch mode**: `nomos build --watch` for live recompilation
- **Format validation**: `nomos fmt` to format `.csl` files

---

**When in doubt about CLI design decisions, prefer:**
1. Simplicity over features (thin wrapper philosophy)
2. Determinism over convenience
3. Clear error messages over terse output
4. Standard library over frameworks (no Cobra)
