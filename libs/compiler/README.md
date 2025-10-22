# Nomos Compiler Library

The compiler turns Nomos source files into a deterministic configuration snapshot. It is a small, stable Go library intended to be consumed by the CLI and other tools.

## What it should achieve

- Parse Nomos scripts and build an internal representation (via the parser).
- Resolve `source`, `import`, and `reference` constructs with clear, extensible provider APIs.
- Compose configuration deterministically with last-wins override semantics.
- Detect and report cycles, missing references, and invalid provider usage with context-rich errors.
- Produce a serializable snapshot (data + metadata) suitable for JSON/YAML/HCL rendering.

### File types

- The compiler accepts Nomos source files with the `.csl` extension and/or directories containing `.csl` files.
- Callers provide a path (file or directory) via `Options.Path`; directory traversal order must be deterministic.

### Built-in provider types

The compiler should ship with a small set of providers by default:
- Folder Source Provider — imports configuration files from a local folder
- OpenTofu State Provider — resolves `reference:` values from OpenTofu IaC outputs

## Relationship to other projects

- Consumed by `apps/command-line` (the CLI) as its primary API surface.
- Depends on `libs/parser` for tokenizing/parsing and AST construction.
- May optionally use `libs/common` (future) for shared utilities.

```
CLI -> Compiler -> Parser
```

## Public API (proposed contract)

Keep the surface minimal and stable:

```go
// Package compiler provides functions to compile Nomos sources into snapshots.
package compiler

type Options struct {
    // Root path or file to compile.
    Path string
    // Optional: provider registry/config, variables, network timeouts.
    Providers ProviderRegistry
    Vars      map[string]any
}

type Snapshot struct {
    Data     map[string]any // fully-composed configuration
    Metadata Metadata       // sources, versions, timestamps
}

func Compile(opts Options) (Snapshot, error)
```

Notes:
- Output `Snapshot` is format-agnostic; the caller (CLI) chooses JSON/YAML/HCL rendering.
- Providers are pluggable and addressed by alias in scripts.

## Providers and sources

- A provider resolves data from a backing system (e.g., filesystem, Git, HTTP, key-value store).
- Scripts declare `source <alias> <type> { config... }`.
- The compiler uses a provider registry to instantiate providers for each alias/type.
- Providers should be side-effect free and cache sensibly within a single compile.

Keywords and syntax (from the root README):
- `import:{alias}` or `import:{alias}:{path_to_map}` — include configuration from a source (last-wins on conflict)
- `reference:{alias}:{path_to_property}` — splice in a specific value from a source

## Composition semantics

- Imports are applied in evaluation order with last-wins key overwrite.
- Deep-merge on objects/maps; arrays replace by default (future: strategy flags).
- References are resolved after imports are materialized to allow cross-file linking.
- Cycles across imports/references are detected and reported.

## Example configuration

```
source:
    alias: 'folder'
    type:  'folder'
    path:  '../config'

import:folder:filename

config-section-name:
    key1: value1
    key2: value2
```

## Errors and diagnostics

- Include file/line/column and source alias/type where possible.
- Differentiate parse errors (from parser), semantic errors (references, cycles), and provider errors (I/O, auth, not found).
- Prefer wrapping errors (`fmt.Errorf("...: %w", err)`) to preserve root cause.

## Usage example

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

- Unit tests for merge semantics, reference resolution, and provider behaviors (with fakes).
- Integration tests under `test/` that compile fixtures in `testdata/` and assert snapshots.
- No external network in tests; use local fixtures or recorded responses.

## Versioning

- Tagged as `libs/compiler/vX.Y.Z`. Follow semantic versioning.
- Any breaking change to the public API requires a major version bump and migration notes.
