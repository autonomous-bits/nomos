# Nomos CLI

The Nomos CLI is the user-facing tool that compiles Nomos `.csl` scripts into deterministic, serializable configuration snapshots.

## Phase 2 Modernization Complete ✓

The CLI has been modernized with:
- **Cobra framework** — Professional command structure with subcommands and consistent flag handling
- **Shell completions** — Built-in support for Bash, Zsh, Fish, and PowerShell
- **Enhanced UX** — Table output, progress indicators, and colored diagnostics
- **New commands** — `validate`, `providers list`, `completion`, and `version`
- **Improved flags** — `--color` (auto/always/never) and `--quiet` for output control

The CLI is a thin wrapper around the compiler library (`libs/compiler`) and is responsible for argument parsing, wiring provider adapters, marshaling output, and mapping compiler results to exit codes.

> **Product Requirements:** See [PRD issue #35](https://github.com/autonomous-bits/nomos/issues/35) for complete feature specification and design decisions.

> **Compiler Documentation:** For compiler-level details, semantics, and provider adapters, see [`libs/compiler`](../../libs/compiler/README.md).

## Installation

### Prerequisites

- Go 1.26.0 or later
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

### Shell Completions

The CLI provides shell completion support for Bash, Zsh, Fish, and PowerShell.

**Bash:**
```bash
# Load completions for current session
source <(nomos completion bash)

# Install completions permanently (Linux)
nomos completion bash > /etc/bash_completion.d/nomos

# Install completions permanently (macOS with Homebrew)
nomos completion bash > $(brew --prefix)/etc/bash_completion.d/nomos
```

**Zsh:**
```bash
# Enable completions (if not already enabled)
echo "autoload -U compinit; compinit" >> ~/.zshrc

# Install completions permanently
nomos completion zsh > "${fpath[1]}/_nomos"

# Restart shell or reload config
source ~/.zshrc
```

**Fish:**
```bash
# Load completions for current session
nomos completion fish | source

# Install completions permanently
nomos completion fish > ~/.config/fish/completions/nomos.fish
```

**PowerShell:**
```powershell
# Load completions for current session
nomos completion powershell | Out-String | Invoke-Expression

# Install completions permanently
nomos completion powershell > nomos.ps1
# Add to PowerShell profile
```

## What's New in Phase 2

The CLI has been completely modernized with professional tooling and enhanced user experience:

### Framework Migration
- **Cobra integration** — Industry-standard CLI framework used by kubectl, hugo, and other professional tools
- **Consistent command structure** — All commands follow `nomos <command> [flags]` pattern
- **Built-in help** — `nomos help <command>` for detailed command documentation

### New Commands
| Command | Description | Example |
|---------|-------------|---------|
| `validate` | Fast syntax/semantic checks | `nomos validate -p config.csl` |
| `providers list` | View installed providers | `nomos providers list` |
| `completion` | Shell completions | `nomos completion bash` |
| `version` | Version information | `nomos version` |

### UX Improvements
- ✓ **Progress indicators** — Animated spinner during provider downloads
- ✓ **Table output** — Formatted tables for providers list
- ✓ **Color support** — Automatic color detection with `--color` flag
- ✓ **Quiet mode** — `--quiet` flag for CI/CD pipelines
- ✓ **Better errors** — Enhanced diagnostics with context and suggestions
- ✓ **Smart defaults** — Color auto-detection, sensible timeouts

### Shell Completions
Out-of-the-box tab completion for Bash, Zsh, Fish, and PowerShell. See [Shell Completions](#shell-completions) section for installation.

## Quick Start

Compile a single .csl file:

```bash
nomos build -p testdata/simple.csl
```

Compile a directory to file:

```bash
nomos build -p testdata/configs -o snapshot.json
```

Compile with variable substitution:

```bash
nomos build -p testdata/with-vars.csl --var region=us-west --var env=dev
```

Enable color output:

```bash
nomos build -p config.csl --color always
```

Quiet mode (perfect for scripts):

```bash
if nomos validate -p config.csl --quiet; then
  nomos build -p config.csl -o output.json --quiet
fi
```

## CLI Commands

The CLI provides the following commands (all implemented using the Cobra framework):

- **`build`** — Compile Nomos scripts into configuration snapshots (JSON/YAML/HCL)
- **`validate`** — Validate .csl files without building (syntax and semantic checks only)
- **`providers list`** — List installed providers from lockfile with details
- **`version`** — Display version information with build metadata
- **`completion`** — Generate shell completion scripts (bash/zsh/fish/powershell)
- **`help`** — Help about any command

### Migration from v1.x

**Breaking Change:** The `nomos init` command has been removed in v2.0.0.

Providers now download automatically during `nomos build` when needed. You no longer need a separate initialization step.

**Before (v1.x):**
```bash
nomos init config.csl
nomos build config.csl
```

**After (v2.0.0):**
```bash
nomos build config.csl  # Providers download automatically
```

The lockfile (`.nomos/providers.lock.json`) is still created and used for version pinning and reproducibility.

### Global Flags

Available for all commands:

- `--color <mode>` — Colorize output: `auto` (default), `always`, or `never`
- `--quiet, -q` — Suppress non-error output
- `--help, -h` — Show help for any command

## Network and Safety Defaults

**The CLI does NOT make network calls by default** (offline-first behavior).

- Provider fetches only occur when provider types are explicitly configured and required by your `.csl` scripts
- This ensures safe, reproducible builds in CI environments without network dependencies
- Use `--allow-missing-provider` to tolerate provider fetch failures if needed
- Control network behavior with `--timeout-per-provider` and `--max-concurrent-providers` flags

This design ensures deterministic, hermetic builds by default.

## Command Reference

### `nomos build`

Compile Nomos scripts into configuration snapshots.

Relevant flags:

- `--path, -p` (required): Path to a `.csl` file or folder containing `.csl` files
- `--format, -f`: Output format (only `json` currently supported)
- `--out, -o`: Write output to file (default: stdout)
- `--var`: Set variable: key=value (repeatable)
- `--strict`: Treat warnings as errors
- `--allow-missing-provider`: Allow compilation with missing providers
- `--timeout-per-provider`: Timeout for provider operations (e.g., `5s`, `1m`) (default: `30s`)
- `--max-concurrent-providers`: Max concurrent provider operations (default: `4`)
- `--verbose, -v`: Enable verbose output

**Exit Codes:**
- `0` — Success
- `1` — Compilation errors (or warnings in strict mode)

### `nomos validate`

Validate `.csl` files for syntax and semantic errors without performing a full build.

**Phase 2 Addition:** This command was added as part of the Cobra migration to provide fast feedback during development.

This command is useful for:
- Pre-commit hooks
- CI/CD pipelines (fast fail-fast checks)
- Quick syntax verification
- Editor integrations (language server protocol)

Usage:

```bash
nomos validate --path <path> [flags]
```

Flags:
- `--path, -p`: Path to .csl file or directory (required)
- `--verbose, -v`: Enable verbose output
- `--color`: Colorize output (auto/always/never)
- `--quiet, -q`: Suppress non-error output

The validate command performs parsing and type checking but does not:
- Invoke providers
- Generate output snapshots
- Perform provider resolution

**Example:**

```bash
# Validate a single file
nomos validate -p config.csl

# Validate an entire directory
nomos validate -p configs/

# Quiet mode (CI-friendly)
nomos validate -p configs/ --quiet
```

**Exit Codes:**
- `0` — Validation passed
- `1` — Validation failed with errors

### `nomos providers list`

List all providers installed in the `.nomos/providers` directory.

**Phase 2 Addition:** This command provides visibility into installed providers with formatted table output.

Usage:

```bash
nomos providers list [flags]
```

Flags:
- `--json`: Output as JSON (for scripting)
- `--quiet, -q`: Suppress summary line
- `--color`: Colorize output (auto/always/never)

**Example output:**

```
┌─────────┬─────────────────────────────────────┬─────────┬────────┬───────┬────────────────────────────────────────┐
│  ALIAS  │                TYPE                 │ VERSION │   OS   │ ARCH  │                  PATH                  │
├─────────┼─────────────────────────────────────┼─────────┼────────┼───────┼────────────────────────────────────────┤
│ file    │ autonomous-bits/nomos-provider-file │ 1.0.0   │ darwin │ arm64 │ .nomos/providers/autonomous-bits/...   │
│ github  │ autonomous-bits/nomos-provider-gh   │ 0.2.1   │ darwin │ arm64 │ .nomos/providers/autonomous-bits/...   │
└─────────┴─────────────────────────────────────┴─────────┴────────┴───────┴────────────────────────────────────────┘

Total: 2 provider(s)
```

**JSON Output:**

```bash
nomos providers list --json
```

Outputs structured JSON with all provider details including checksums for CI/CD validation.

### `nomos version`

Display version information including build metadata.

**Phase 2 Addition:** This command was added as part of the Cobra migration following CLI best practices.

Usage:

```bash
nomos version [flags]
```

Flags:
- `--quiet, -q`: Output only the version number (for scripting)

**Example output:**

```
Nomos CLI Version: 0.2.0
Commit: a1b2c3d4e5f
Build Date: 2025-12-26T10:30:00Z
Go Version: go1.26+
Module: github.com/autonomous-bits/nomos/apps/command-line
```

**Quiet mode (scripting-friendly):**

```bash
nomos version --quiet
# Output: 0.2.0

# Use in scripts
if [[ $(nomos version -q) == "0.2.0" ]]; then
  echo "Version matches"
fi
```

## Commands

### build

Compile Nomos .csl files into a configuration snapshot.

**Usage:**
```bash
nomos build [flags]
```

**Options:**

- `-p, --path <path>` — Path to .csl file or directory (required)
- `-f, --format <format>` — Output format (only json currently supported)
- `-o, --out <file>` — Write output to file (default: stdout)
- `--var <key=value>` — Variable substitution (repeatable)
- `--strict` — Treat warnings as errors
- `--allow-missing-provider` — Allow missing provider fetches
- `--timeout-per-provider <duration>` — Timeout for each provider fetch (e.g., 5s, 1m)
- `--max-concurrent-providers <int>` — Maximum concurrent provider fetches
- `--include-metadata` — Include compilation metadata in output (opt-in for debugging/auditing)
- `--verbose, -v` — Enable verbose logging
- `--color <mode>` — **[Phase 2]** Colorize output: auto, always, never (default: auto)
- `--quiet, -q` — **[Phase 2]** Suppress non-error output
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

**Syntax:** `@{alias}:{filename}.{path.to.value}`

**Example source file with references:**

```nomos
source:
  alias: 'configs'
  type: 'autonomous-bits/nomos-provider-file'
  version: '0.1.1'
  directory: './shared-configs'

app:
  name: 'my-app'
  storage:
    type: @configs:storage.config.storage.type
    bucket: @configs:storage.config.buckets.primary
    encryption: @configs:storage.config.encryption.algorithm
```

Given `shared-configs/storage.csl`:
```nomos
config:
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

The CLI is a thin layer over the compiler library, built with the Cobra framework for professional command-line UX:

```
CLI (apps/command-line)
  ├── Cobra Framework (command structure, flags, completions)
  └── uses → Compiler (libs/compiler)
               └── uses → Parser (libs/parser)
```

### Key Components

**Phase 2 Structure (Cobra-based):**

- `cmd/nomos/main.go` — Entry point with color setup
- `cmd/nomos/root.go` — Root command and global flags (--color, --quiet)
- `cmd/nomos/build.go` — Build command implementation
- `cmd/nomos/validate.go` — Validate command (new in Phase 2)
- `cmd/nomos/providers.go` — Providers list command (new in Phase 2)
- `internal/flags/` — Flag parsing and validation (legacy, transitioning to Cobra)
- `internal/options/` — Compiler options builder with provider wiring
- `internal/diagnostics/` — Error/warning formatting with color support

**Note:** `cmd/nomos/init.go` was removed in v2.0.0 as providers now download automatically during build.

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

### Metadata Output Control

**Default Behavior (v2.0.0+):** The CLI outputs only the compiled configuration data without metadata. This produces cleaner, production-ready configuration files that are smaller and faster to parse.

**When You Need Metadata:** Use the `--include-metadata` flag when you need:
- **Debugging**: See which files contributed which values
- **Auditing**: Track configuration provenance for compliance
- **Tooling**: Parse metadata for custom workflows
- **Migration**: Temporary compatibility with v1.x behavior

**Example - Default Output (No Metadata):**

```bash
nomos build -p config.csl
```

```json
{
  "app": "example",
  "database": {
    "host": "localhost",
    "port": 5432
  }
}
```

**Example - With Metadata:**

```bash
nomos build -p config.csl --include-metadata
```

```json
{
  "data": {
    "app": "example",
    "database": {
      "host": "localhost",
      "port": 5432
    }
  },
  "metadata": {
    "start_time": "2026-02-14T10:00:00Z",
    "end_time": "2026-02-14T10:00:01Z",
    "input_files": ["/path/to/config.csl"],
    "provider_aliases": [],
    "per_key_provenance": {
      "app": {
        "source": "/path/to/config.csl",
        "provider_alias": ""
      },
      "database": {
        "source": "/path/to/config.csl",
        "provider_alias": ""
      }
    },
    "errors": [],
    "warnings": []
  }
}
```

**YAML Format with Metadata:**

```bash
nomos build -p config.csl --format yaml --include-metadata
```

```yaml
data:
  app: example
  database:
    host: localhost
    port: 5432
metadata:
  start_time: "2026-02-14T10:00:00Z"
  end_time: "2026-02-14T10:00:01Z"
  input_files:
    - /path/to/config.csl
  provider_aliases: []
  per_key_provenance:
    app:
      source: /path/to/config.csl
      provider_alias: ""
    database:
      source: /path/to/config.csl
      provider_alias: ""
  errors: []
  warnings: []
```

**Migration Note:** In v1.x, metadata was included by default. Starting in v2.0.0, metadata is opt-in via `--include-metadata`. See the [migration guide](../../docs/guides/migration-v2.md#optional-metadata-output-v200) for details.

### Output Formats and Serialization

The CLI supports multiple output formats via the `--format` flag, with deterministic serialization to ensure byte-for-byte identical results for identical inputs (critical for CI reproducibility).

**Supported Formats:**

- `json` (default) — Canonical JSON with deterministic key ordering
- `yaml` — YAML format for Kubernetes, Ansible, Docker Compose
- `tfvars` — Terraform .tfvars format (HCL syntax)

#### JSON Format (Default)

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

**Usage:**

```bash
# JSON to stdout (default)
nomos build -p config.csl

# JSON to file (extension added automatically)
nomos build -p config.csl -o output
# Creates: output.json

# Explicit JSON format
nomos build -p config.csl --format json -o output.json
```

#### YAML Format

The YAML serializer produces valid YAML 1.2 output compatible with standard YAML parsers and tools like Kubernetes, Ansible, Docker Compose, and GitHub Actions.

**Features:**

- **Deterministic ordering**: Map keys sorted alphabetically (arrays preserve insertion order)
- **2-space indentation**: Following YAML conventions
- **Full snapshot**: Includes both `data` and `metadata` sections
- **Proper escaping**: Strings with special characters are quoted and escaped correctly

**Format-Specific Validation:**

- Keys cannot contain null bytes (`\x00`) as prohibited by YAML specification
- Unsupported Go types (channels, functions) cause compilation errors with clear messages

**Example YAML Output:**

```yaml
data:
  alpha:
    value: first
  middle:
    nested: value
  zebra:
    value: last
metadata:
  end_time: "2025-10-26T20:00:00Z"
  errors: []
  input_files:
    - /path/to/config.csl
  per_key_provenance:
    alpha:
      provider_alias: ""
      source: /path/to/config.csl
    middle:
      provider_alias: ""
      source: /path/to/config.csl
    zebra:
      provider_alias: ""
      source: /path/to/config.csl
  provider_aliases: []
  start_time: "2025-10-26T20:00:00Z"
  warnings: []
```

**Usage:**

```bash
# YAML to stdout
nomos build -p config.csl --format yaml

# YAML to file (extension added automatically)
nomos build -p config.csl --format yaml -o kubernetes-config
# Creates: kubernetes-config.yaml

# Use with Kubernetes
nomos build -p k8s/app.csl --format yaml -o deployment.yaml
kubectl apply -f deployment.yaml

# Use with Docker Compose
nomos build -p docker/services.csl --format yaml -o docker-compose.yaml
docker-compose -f docker-compose.yaml up
```

**Common Use Cases:**

- **Kubernetes**: Generate ConfigMaps, Deployments, Services
- **Docker Compose**: Multi-container application definitions
- **Ansible**: Playbook variable files
- **GitHub Actions**: Workflow configuration
- **Helm**: Values files for chart customization

#### Terraform .tfvars Format

The tfvars serializer produces valid HCL (HashiCorp Configuration Language) output for use with Terraform variable files.

**Features:**

- **HCL syntax**: Native Terraform variable file format
- **Deterministic ordering**: Keys sorted alphabetically
- **Type preservation**: Strings, numbers, booleans, maps, and lists
- **Data only**: Only the `data` section is serialized (metadata omitted for Terraform compatibility)

**Format-Specific Validation:**

- **Strict identifier rules**: Keys must match HCL identifier pattern `[a-zA-Z_][a-zA-Z0-9_-]*`
  - Must start with letter or underscore
  - Can contain only letters, digits, underscores, and hyphens
  - Examples: `region` ✓, `vpc_id` ✓, `enable-dns` ✓, `my-key` ✗ (space), `123key` ✗ (starts with digit)
- Invalid keys cause compilation errors with clear messages listing all problematic keys

**Example .tfvars Output:**

```hcl
alpha = "first"
middle = {
  nested = "value"
}
vpc = {
  cidr_block = "10.0.0.0/16"
  enable_dns = true
  subnets = [
    {
      cidr = "10.0.1.0/24"
      zone = "us-west-2a"
    },
    {
      cidr = "10.0.2.0/24"
      zone = "us-west-2b"
    }
  ]
}
zebra = "last"
```

**Usage:**

```bash
# Tfvars to stdout
nomos build -p config.csl --format tfvars

# Tfvars to file (extension added automatically)
nomos build -p terraform/vars.csl --format tfvars -o terraform
# Creates: terraform.tfvars

# Use with Terraform
nomos build -p terraform/prod.csl --format tfvars -o prod.tfvars
terraform plan -var-file=prod.tfvars
terraform apply -var-file=prod.tfvars

# Multi-environment setup
nomos build -p envs/dev.csl --format tfvars -o dev.auto.tfvars
nomos build -p envs/prod.csl --format tfvars -o prod.auto.tfvars
# Terraform automatically loads *.auto.tfvars files
```

**Common Use Cases:**

- **Terraform modules**: Variable definitions for infrastructure modules
- **Environment-specific configs**: Separate .tfvars files per environment (dev, staging, prod)
- **Team collaboration**: Shared variable definitions in version control
- **CI/CD pipelines**: Generated variable files from configuration sources

**Terraform Integration Pattern:**

```bash
# 1. Define variables in Terraform
# variables.tf:
variable "region" { type = string }
variable "vpc" { type = object({ cidr_block = string }) }

# 2. Generate .tfvars from Nomos
nomos build -p config.csl --format tfvars -o terraform.tfvars

# 3. Use in Terraform
terraform apply -var-file=terraform.tfvars
```

#### Automatic File Extension Handling

When using the `--out` flag without an explicit extension, the CLI automatically appends the correct extension based on the format:

```bash
# Automatic extensions
nomos build -p config.csl --format json -o output    # Creates: output.json
nomos build -p config.csl --format yaml -o config    # Creates: config.yaml
nomos build -p config.csl --format tfvars -o vars    # Creates: vars.tfvars

# Explicit extensions are preserved  
nomos build -p config.csl --format yaml -o data.yml  # Creates: data.yml (not .yaml)
nomos build -p config.csl --format json -o out.json  # Creates: out.json
```

**Extension Rules:**

- **Recognized extensions are preserved**: `.json`, `.yaml`, `.yml`, `.tfvars`, `.hcl`
- **Unrecognized suffixes get format extension**: `config.prod` → `config.prod.json` (for JSON format)
- **Multi-part tfvars extensions**: `.auto.tfvars` is recognized and preserved
- **Directories are created automatically**: `nomos build -p config.csl -o build/snapshots/output` creates `build/snapshots/output.json`

#### Format Validation and Error Handling

The CLI validates configuration compatibility with the target format before serialization:

**YAML Validation:**
- Checks for null bytes in keys (prohibited by YAML spec)
- Detects unsupported Go types (channels, functions, complex numbers)
- Error example: `YAML key cannot contain null bytes: "invalid\x00key"`

**Tfvars Validation:**
- Validates all keys match HCL identifier pattern
- Detects unsupported types
- Error example: `invalid keys for HCL identifiers (must match [a-zA-Z_][a-zA-Z0-9_-]*): ["my key", "123invalid"]`

**General Validation:**
- Directory writability checked before compilation
- Invalid format values rejected with supported format list
- Clear error messages with file locations and problematic keys

**Output Destination:**

- **Default**: Writes to stdout
- **With `--out` flag**: Writes to specified file path
- **Directories created automatically** if they don't exist
- **Non-writable paths** result in exit code 2 with clear error message

**Implementation Details:**

The serializer is located in `internal/serialize` and provides:
- `ToJSON(snapshot)` — Canonical JSON serialization
- `ToYAML(snapshot)` — YAML 1.2 serialization with sorted keys
- `ToTfvars(snapshot)` — HCL .tfvars serialization with validation

See `internal/serialize/` tests for comprehensive validation and determinism tests.

#### Format-Specific Type Handling

Different output formats handle data types according to their specifications. Understanding these differences helps you choose the right format for your use case.

**Type Preservation Behavior**

| Type | JSON | YAML | Tfvars |
|------|------|------|--------|
| Numbers | Preserves strings | Native numbers (type inference) | Native numbers |
| Empty strings | `""` | `null` | `""` |
| Booleans | Preserves strings | Native booleans (type inference) | Native booleans |

**Format Recommendations**

**Choose JSON when:**
- Exact string preservation is required
- Consuming system expects JSON format
- Working with APIs or generic data processors
- You need deterministic serialization

**Choose YAML when:**
- Deploying to Kubernetes (ConfigMaps, manifests)
- Human readability is important
- Native type inference is desired (strings → numbers/booleans)
- Working with tools that expect YAML input

**Choose Tfvars when:**
- Integrating with Terraform
- Strong typing is required (HCL semantics)
- Variable declarations needed for IaC tools
- Working with HashiCorp ecosystem

**Cross-Format Equivalence**

**Note:** Cross-format type preservation is not guaranteed for all types due to differences in format specifications. When type fidelity is critical:

1. Use JSON for exact preservation
2. Define explicit types in your source `.csl` files
3. Test with target system to verify compatibility

**Examples**

**Source (.csl):**
```nomos
region: "us-west-2"
port: 5432
enabled: true
tags: []
```

**JSON Output:**
```json
{
  "data": {
    "region": "us-west-2",
    "port": "5432",
    "enabled": "true",
    "tags": []
  }
}
```

**YAML Output:**
```yaml
data:
  region: us-west-2    # string
  port: 5432           # number (inferred)
  enabled: true        # boolean (inferred)
  tags: []             # empty array
```

**Tfvars Output:**
```hcl
region = "us-west-2"   # string
port = "5432"          # string (as quoted in source)
enabled = "true"       # string (as quoted in source)
tags = []              # empty list
```

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

- Go 1.26.0 or later
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

- **YAML and HCL output formats** — May be added if user demand justifies the implementation (currently JSON-only)
- ~~Provider credential handling~~ (basic support exists)
- ~~Remote provider support with explicit opt-in~~ (GitHub Releases supported)
- ~~Additional commands (`validate`, `fmt`)~~ ✓ **Completed in Phase 2** (validate); **Note:** `init` was added in Phase 2 and removed in v2.0.0 (auto-download)
- **Format command** — `nomos fmt` to auto-format .csl files (planned)
- **Watch mode** — `nomos build --watch` for live recompilation
- **Language server** — LSP integration for IDE support
- **Telemetry** — Usage analytics (opt-in only)
- **Performance benchmarking** — Compilation speed targets

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

The compiler populates `Metadata.Errors` and `Metadata.Warnings` with formatted messages including source locations. The CLI's diagnostics package formats these for terminal output.

**Phase 2 UX Enhancements:**
- ✓ **Color support** via `--color` flag (auto-detects TTY by default)
- ✓ **Better error context** with source snippets
- ✓ **Consistent formatting** for both errors and warnings
- ✓ **Machine-parseable** format for tooling integration

**Example error output:**

```
config.csl:10:5: error: unresolved reference to provider 'db'
   10 |   host: @db:host
      |         ^
```

**Features:**
- File, line, and column information from parser/compiler diagnostics
- Context snippets with caret markers pointing to error locations
- Consistent `file:line:col: severity: message` format for machine parsing
- Color output controlled via `--color auto|always|never`
- Summary line showing total error/warning count

**Color Modes:**
- `auto` — Enable colors when outputting to a terminal (default)
- `always` — Force colors even when piping output
- `never` — Disable colors (for logs, CI, or accessibility)

Use `--strict` to treat warnings as errors (causes exit code 1).

## External Providers

Nomos uses external providers as separate executables for fetching configuration data. This is the recommended approach for production use.

### Overview

External providers:
- Are standalone executables distributed via GitHub Releases or local paths
- Communicate with the compiler via gRPC
- Support any language with gRPC support (Go, Python, Node.js, etc.)
- Provide isolation, independent versioning, and security boundaries

### Provider Auto-Download (v2.0.0+)

**Providers are automatically downloaded during `nomos build`** — no separate installation step is needed.

When you run `nomos build`, the compiler:
1. Scans your `.csl` files for provider requirements
2. Downloads any missing providers from GitHub Releases
3. Creates/updates `.nomos/providers.lock.json` for version pinning
4. Caches providers in `.nomos/providers/` for subsequent builds

```bash
# Providers download automatically on first build
nomos build config.csl
```

**Lockfile behavior:**
- First build creates `.nomos/providers.lock.json` with resolved versions
- Subsequent builds reuse the locked versions (reproducible builds)
- Commit the lockfile to version control for team consistency

**Migration from v1.x:** If you were using `nomos init`, simply remove it from your workflow. The `build` command now handles everything.

### Building with Providers

Simply use `nomos build` — providers are automatically managed:

```bash
nomos build config.csl
```

The compiler will:
1. Check for provider requirements in your `.csl` files
2. Download missing providers from GitHub Releases (first build only)
3. Read/update the lock file (`.nomos/providers.lock.json`)
4. Start provider subprocesses as needed
5. Call provider RPCs to fetch data
6. Shut down providers after compilation

### Workflow Example

Complete workflow from scratch (v2.0.0+):

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

# 2. Build configuration (providers download automatically)
nomos build config.csl -o output.json
# → Downloads nomos-provider-file from GitHub (first time)
# → Creates .nomos/providers/ and lock file
# → Starts file provider subprocess
# → Fetches data via gRPC
# → Compiles to output.json

# 3. Subsequent builds use cached provider
nomos build config.csl
# → Reuses already-downloaded provider (fast)
# → No re-download unless version changes
```

### Upgrading Providers

To upgrade to newer provider versions:

```bash
# 1. Update version in config.csl
sed -i 's/version = "1.0.0"/version = "1.1.0"/' config.csl

# 2. Delete the lockfile to allow re-resolution
rm .nomos/providers.lock.json

# 3. Build to download new version
nomos build config.csl
# → Downloads provider v1.1.0
# → Creates new lockfile
```

**Note:** In v1.x, you used `nomos init --upgrade`. In v2.0.0+, delete the lockfile and rebuild.

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
