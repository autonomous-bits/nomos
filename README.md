# Nomos

Nomos is a configuration scripting language aimed to reduce configuration toil by promoting re-use and cascading overrides. Nomos aims to:

- Allow users to compose configuration by importing configuration layers, where each layer would replace any conflicting configuration values.
- Allow users to group configuration into cohesive environments.
- Allow users to reference configuration across configuration groups (environments)

These configuration scripts will be compiled producing a versioned snapshot that will be used as inputs for infrastructure as code.

## Scripting Language

The scripting language supports the following keywords:

| Keyword | Description |
| :-| :- |
| `source` | A configurable source provider, at a minimum you should be able to provide an alias and the type of provider. |
| `import` | Using a source, configuration could be imported i.e. when compiled those values should be part of a snapshot. Syntax should be `import:{alias}` or `import:{alias}:{path_to_map}`. If two or more files have conflicting properties the last import will override the previous properties. |
| `reference` | Using a source, load a specific value from the configuration. Syntax should be `reference:{alias}:{path.to.property}` where the path uses dot notation to navigate into nested structures. For file providers, the format is `reference:{alias}:{filename}.{nested.path}` |

### Reference Syntax Details

References allow you to access specific values from imported sources using dot-separated paths:

**For file providers:**
```
reference:{alias}:{filename}.{path.to.value}
```

**Example:**

Given a file `storage.csl` in a `configs` provider:
```
storage:
  type: 's3'
buckets:
  primary: 'my-app-data'
encryption:
  algorithm: 'AES256'
```

You can reference specific values:
```
source:
  alias: 'configs'
  type: 'file'
  directory: './shared-configs'

app:
  storage_type: reference:configs:storage.storage.type        # Resolves to 's3'
  bucket: reference:configs:storage.buckets.primary           # Resolves to 'my-app-data'
  encryption: reference:configs:storage.encryption.algorithm  # Resolves to 'AES256'
```

### Source Provider Types

- **File Source Provider**: The built-in source provider that allows a user to import and reference files from a directory containing `.csl` files. Supports path navigation to access nested values within files.
- **OpenTofu State Provider**: A provider that allows to reference output values from OpenTofu IaC. 

### Example Config

```
source:
  alias: 'configs'
  type: 'file'
  directory: './shared-configs'

import:configs:base

app:
  name: 'my-app'
  # Reference specific values from files using dot notation
  db_host: reference:configs:database.connection.host
  storage_type: reference:configs:storage.type
  
config-section-name:
  key1: value1
  key2: value2
```

## File Extension

The file extension for the file type are ".csl" which is short for configuration scripting langauge. 

## Tooling

A command line interface (CLI) will be provided where a script or set of scripts could be provided as inputs and then compiled. The compilation will produce a snapshot of the configuration as the output.

The CLI has one command called `build` which accepts a `--path, -p` argument to a file or folder and a `--format, -f` argument that specifies the output format `JSON, YAML, HCL`.

## Development

This repository is organized as a Go monorepo with multiple independent modules:

- `apps/command-line` - Nomos CLI application
- `libs/compiler` - Nomos compiler library
- `libs/parser` - Nomos parser library
- `libs/provider-proto` - Provider protocol definitions (protobuf)

### Local Development Setup

This repository uses Go workspaces for local development:

```bash
# Clone the repository
git clone https://github.com/autonomous-bits/nomos.git
cd nomos

# The go.work file is already configured with all modules
# Sync workspace dependencies
go work sync

# Build all applications
make build

# Run tests across all modules
make test

# Run tests with race detector
make test-race

# Build the CLI application
make build-cli

# Build or test a specific module
make build-module MODULE=libs/compiler
make test-module MODULE=libs/parser
```

### Available Make Targets

Run `make help` to see all available targets:

- `build` - Build all applications
- `build-cli` - Build the CLI application
- `build-module MODULE=<path>` - Build a specific module
- `test` - Run all tests across all modules
- `test-race` - Run tests with race detector
- `test-module MODULE=<path>` - Test a specific module
- `lint` - Run linters across all modules
- `work-sync` - Sync Go workspace dependencies
- `clean` - Clean build artifacts

### CI Verification

A verification script is available to validate workspace integrity and run tests:

```bash
./scripts/verify-ci-modules.sh
```

This script:
1. Verifies `go.work` exists and is properly configured
2. Syncs workspace dependencies
3. Verifies all required modules are present
4. Runs tests for each module
5. Reports success or failure with clear output

### Module Structure

Each module maintains its own:
- `go.mod` and `go.sum` for dependency management
- `CHANGELOG.md` following [Keep a Changelog](https://keepachangelog.com/)
- `README.md` with module-specific documentation
- `Makefile` with module-level build targets

For detailed architecture guidelines, see [Go Monorepo Structure](docs/architecture/go-monorepo-structure.md)

### Using Nomos Libraries in External Projects

Nomos libraries are published as independent Go modules and can be imported into external projects:

```go
// In your project's go.mod
module github.com/example/my-project

go 1.22

require (
    github.com/autonomous-bits/nomos/libs/compiler v0.1.0
    github.com/autonomous-bits/nomos/libs/parser v0.1.0
    github.com/autonomous-bits/nomos/libs/provider-proto v0.1.0
)
```

Example usage:

```go
package main

import (
    "context"
    "github.com/autonomous-bits/nomos/libs/compiler"
)

func main() {
    opts := compiler.Options{
        Path: "config.csl",
        // Configure other options...
    }
    
    snapshot, err := compiler.Compile(context.Background(), opts)
    if err != nil {
        // Handle error
    }
    
    // Use compiled snapshot
}
```

For a complete working example, see [examples/consumer/](examples/consumer/README.md).

**Note for local development**: When working within this monorepo, the `go.work` file handles module resolution automatically—no `replace` directives are needed in individual `go.mod` files 