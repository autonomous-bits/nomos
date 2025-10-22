# Nomos CLI

The Nomos CLI is the user-facing tool that compiles Nomos scripts into configuration snapshots. It provides a thin, reliable interface over the compiler library, handling argument parsing, validation, I/O, and exit codes.

## What it should achieve

- Simple, ergonomic command(s) to compile Nomos inputs into a structured snapshot.
- Consistent composition semantics: later inputs override earlier values on conflict (last-wins).
- Output in common formats for downstream tools: JSON, YAML, HCL.
- Clear, actionable errors (with file/line/column when possible) and non-zero exit codes on failure.
- Zero network by default unless remote references/sources are used; safe defaults and timeouts when networking is enabled.

### Built-in provider types

The CLI relies on the compiler’s provider system. Out of the box, Nomos intends to support:
- Folder Source Provider: import files from a local folder
- OpenTofu State Provider: reference outputs from OpenTofu IaC state

## Relationship to other projects

- Depends on `libs/compiler` for all compilation semantics and orchestration.
- `libs/compiler` depends on `libs/parser` to turn source files into an AST for analysis.
- The CLI does not implement language logic; it delegates to the compiler and only coordinates inputs/outputs.

```
CLI (apps/command-line)
  └── uses -> Compiler (libs/compiler)
                 └── uses -> Parser (libs/parser)
```

## Command surface

The CLI exposes a single top-level command initially:

- `build` — compile one file or a directory of Nomos scripts into a snapshot
  - `--path, -p`: path to a file or folder to compile (required)
  - `--format, -f`: output format; one of `json`, `yaml`, `hcl` (default: `json`)

Example:

```
nomos build -p ./examples/environments/dev -f yaml > snapshot.yaml
```

Assumptions for v0:
- Exit code 0 on success, non-zero on failure.
- Directory inputs are compiled in deterministic order (e.g., lexicographic by path).

## File types

- Nomos source files use the `.csl` extension (configuration scripting language).
- `--path` may point to a single `.csl` file or a directory containing `.csl` files.
- When a directory is provided, the CLI discovers `.csl` files recursively (implementation detail may evolve) and passes them to the compiler.

## Input model (from the language README)

Nomos scripts support:
- `source`: declares a remote or local source provider (alias + type + config)
- `import`: composes configuration from a source (optionally at a path within that source)
- `reference`: pulls a specific value from a source at a given path

Composition rule: when multiple imports define the same key, the later import overrides previous values (last-wins).

## Example configuration

A minimal script demonstrating sources and imports:

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

## Error modes

- Invalid arguments: print usage and return exit code 2.
- Parse/compile errors: print user-friendly message with file/line/column when available; exit code 1.
- I/O/network errors: include provider alias/type and attempted path; exit code 1.

## Development

- Use Go workspaces at the repo root to link local modules (`go work use ./apps/command-line ./libs/compiler ./libs/parser`).
- Keep CLI packages under `internal/` focused on argument parsing and app wiring; all language semantics live in `libs/*`.

## Minimal contract (CLI ↔ compiler)

- CLI calls: `compiler.Compile(path, options...) (Snapshot, error)`
- CLI marshals the `Snapshot` to the requested format and writes to stdout or a file.
- The CLI does not inspect AST internals.

## Future flags (non-breaking)

- `--strict` to treat warnings as errors.

## Versioning and releases

- Tagged as `apps/command-line/vX.Y.Z`. Follows semantic versioning.
- Public UX changes (flags, outputs) must be documented and reflected in `CHANGELOG.md`.
