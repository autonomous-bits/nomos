# Nomos

Nomos is a configuration scripting language that reduces configuration toil through composition, reuse, and cascading overrides. It lets you:

- Compose configuration by importing layered sources with last-write-wins semantics
- Group configuration into cohesive environments
- Cross-reference values across configuration groups (environments)

Nomos scripts compile into a versioned snapshot that becomes input for your infrastructure-as-code tools.

• CLI: `apps/command-line`  • Compiler library: `libs/compiler`  • Parser: `libs/parser`  • Provider contracts: `libs/provider-proto`

Jump to: [Language](#scripting-language) · [Providers](#source-provider-types) · [Examples](#example-config) · [Development](#development) · [Contributing](#contributing)

---

## 🚀 What's New in v2.0.0

**Breaking Change**: The `nomos init` command has been removed. Provider installation now happens automatically during `nomos build`, simplifying the workflow from two commands to one.

**Before (v1.x)**:
```bash
nomos init config.csl        # Step 1: Install providers
nomos build config.csl       # Step 2: Build configuration
```

**After (v2.0.0)**:
```bash
nomos build config.csl       # Providers installed automatically
```

**📖 See the [Migration Guide](docs/guides/migration-v2.md)** for detailed migration instructions, CI/CD updates, and troubleshooting.

---

## Scripting Language

The scripting language supports the following keywords:

| Keyword | Description |
| :-| :- |
| `source` | A configurable source provider, at a minimum you should be able to provide an alias and the type of provider. |
| `import` | Using a source, configuration could be imported i.e. when compiled those values should be part of a snapshot. Syntax should be `import:{alias}` or `import:{alias}:{path_to_map}`. If two or more files have conflicting properties the last import will override the previous properties. |
| `@` | Using a source, load a specific value from the configuration. Syntax is `@{alias}:{path.to.property}` where the path uses dot notation to navigate into nested structures. For file providers, the format is `@{alias}:{filename}.{nested.path}` |

**Comment Support**: Document your configurations with YAML-style `#` comments:
```
app:
  name: 'my-app'
  database:
    host: localhost  # Production database server
    port: 5432       # PostgreSQL default port
```
Comments are single-line, context-aware, and preserved within quoted strings. See the [parser documentation](libs/parser/README.md#comment-support) for complete details.

### Reference Syntax Details

References allow you to access specific values from imported sources using dot-separated paths:

**For file providers:**
```
@{alias}:{filename}.{path.to.value}
```

**Example:**

Given a file `storage.csl` in a `configs` provider:
```
config:
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
  type: 'autonomous-bits/nomos-provider-file'
  version: '0.1.1'
  directory: './shared-configs'

app:
  storage_type: @configs:storage.config.storage.type        # Resolves to 's3'
  bucket: @configs:storage.config.buckets.primary           # Resolves to 'my-app-data'
  encryption: @configs:storage.config.encryption.algorithm  # Resolves to 'AES256'
```

### Source Provider Types

- **File Source Provider**: The built-in source provider that allows a user to import and reference files from a directory containing `.csl` files. Supports path navigation to access nested values within files.
- **OpenTofu State Provider**: A provider that allows to reference output values from OpenTofu IaC. 

### Example Config

```
source:
  alias: 'configs'
  type: 'autonomous-bits/nomos-provider-file'
  version: '0.1.1'
  directory: './shared-configs'

import:configs:base

app:
  name: 'my-app'
  database:
    host: @configs:database.connection.host
  storage:
    type: @configs:storage.config.type
  
config-section-name:
  key1: value1
  key2: value2
```

## File Extension

Use the `.csl` extension (Configuration Scripting Language).

## Tooling

The Nomos CLI compiles scripts into a snapshot artifact.

- Command: `build`
- Flags:
  - `--path, -p` Path to a `.csl` file or folder
  - `--format, -f` Output format: `json`, `yaml`, or `hcl`

Quick start (local):

```bash
make build-cli
./bin/nomos --help
```

## Development

This repository is a Go workspace (monorepo) containing multiple independent modules. Run `make help` for available tasks.

Common tasks:

```bash
# Sync workspace and build the CLI
make work-sync
make build-cli

# Build everything and run tests
make build
make test
```

- Module docs and architecture guidance: see [docs/architecture/go-monorepo-structure.md](docs/architecture/go-monorepo-structure.md)
- CI helper: `./tools/scripts/verify-ci-modules.sh` validates workspace and runs tests
  
Each module maintains its own `go.mod`, `README.md`, and `CHANGELOG.md`.

### Using Nomos Libraries in External Projects

Nomos libraries are published as independent Go modules and can be imported into external projects:

```go
// In your project's go.mod
module github.com/example/my-project

go 1.26.0

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

**Note for local development**: When working within this monorepo, the `go.work` file handles module resolution automatically—no `replace` directives are needed in individual `go.mod` files.

## Contributing

Contributions are welcome! Please read the [CONTRIBUTING.md](CONTRIBUTING.md) for branching, commit message, testing, and PR guidelines.

## License

See [LICENSE](LICENSE).