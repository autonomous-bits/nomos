# Nomos CLI

The Nomos CLI is the user-facing tool that compiles Nomos scripts into configuration snapshots. It provides a thin, reliable interface over the compiler library, handling argument parsing, validation, I/O, and exit codes.

## What it should achieve

- Simple, ergonomic command(s) to compile Nomos inputs into a structured snapshot.
- Consistent composition semantics: later inputs override earlier values on conflict (last-wins).
- Output in common formats for downstream tools: JSON, YAML, HCL.
- Clear, actionable errors (with file/line/column when possible) and non-zero exit codes on failure.
- Zero network by default unless remote references/sources are used; safe defaults and timeouts when networking is enabled.

# Nomos CLI

The Nomos CLI is the user-facing tool that compiles Nomos `.csl` scripts into deterministic, serializable configuration snapshots.
It is a thin wrapper around the compiler library (`libs/compiler`) and is responsible for argument parsing, wiring provider adapters, marshaling output, and mapping compiler results to exit codes.

## Goals

- Provide a small, ergonomic CLI for local development and CI.
- Output snapshots in common formats (JSON, YAML, HCL) for downstream tools.
- Surface clear diagnostics (file/line/column when available) and deterministic composition semantics.

## Relationship to other modules

- `apps/command-line` uses `libs/compiler` for compilation semantics.
- `libs/compiler` uses `libs/parser` to parse `.csl` files into an AST.

```
CLI (apps/command-line)
  └── uses -> Compiler (libs/compiler)
                 └── uses -> Parser (libs/parser)
```

## CLI commands & flags

The primary command provided by the CLI is:

- `build` — compile a file or directory of Nomos scripts into a snapshot

Relevant flags:

- `--path, -p` (required): path to a `.csl` file or a folder containing `.csl` files
- `--format, -f`: output format; one of `json`, `yaml`, `hcl` (default: `json`)
- `--allow-missing-provider`: allow missing provider fetches (compiler.Options.AllowMissingProvider)
- `--timeout-per-provider`: duration string (e.g., `5s`) used for per-provider fetch timeout
- `--max-concurrent-providers`: integer limit for concurrent provider fetches
- `--var` (repeatable): variable substitutions in `key=value` form passed to the compiler (compiler.Options.Vars)
- `--out, -o`: write output to a file instead of stdout
- `--strict`: treat warnings as errors (CLI-level flag)

Example usage:

```sh
nomos build -p ./examples/environments/dev -f yaml -o snapshot.yaml
```

Notes:
- Directory inputs are processed deterministically (lexicographic order of discovered `.csl` files). 
- Single-file inputs may trigger import-resolution paths if provider type constructors are available (see Compiler behavior below).

## File discovery and types

- Source files must use the `.csl` extension.
- `--path` may point to a single `.csl` file or a directory. When a directory is provided the CLI discovers `.csl` files and passes them (in lexicographic order) to the compiler.

## Compiler contract (what the CLI calls)

The CLI calls the compiler using the exported function:

```go
func Compile(ctx context.Context, opts Options) (Snapshot, error)
```

Key `Options` fields relevant to the CLI:

- `Path string` — input file or directory
- `ProviderRegistry` — runtime provider registry instance (required)
- `ProviderTypeRegistry` — constructors for provider types (optional but required to process `source:` declarations)
- `Vars map[string]any` — variable substitutions
- `Timeouts OptionsTimeouts` — `PerProviderFetch` and `MaxConcurrentProviders`
- `AllowMissingProvider bool` — controls behavior when a provider fetch fails

`Snapshot` returned by the compiler contains:

- `Data map[string]any` — the compiled configuration ready for serialization
- `Metadata` — provenance and diagnostics (input files, provider aliases, start/end times, `Errors`, `Warnings`, `PerKeyProvenance`)

Important runtime behavior the CLI should handle:

- The compiler may return a non-nil `error` for fatal failures. In some cases (e.g., unresolved references detected during validation) the compiler may also return a `Snapshot` containing partial `Data` and `Metadata.Errors`.
- If `Metadata.Errors` is non-empty the CLI should print diagnostics and return a non-zero exit code.
- Warnings are reported in `Metadata.Warnings` and are non-fatal unless `--strict` is set.

Example (Go pseudocode for wiring in CLI):

```go
ctx := context.Background()
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

## Exit-code mapping (recommended)

- invalid usage / bad arguments: exit code 2
- compile fatal error (err returned): exit code 1
- compile completed but `Metadata.Errors` non-empty: exit code 1
- compile completed with warnings only: exit code 0 (unless `--strict` then exit 1)

## Error and diagnostic handling

- The CLI should print human-friendly messages. Where possible include file/line/column from parser diagnostics. The compiler populates `Metadata.Errors` and `Metadata.Warnings` with formatted messages that the CLI can display.

## Development notes

- Use the Go workspace at the repo root for local development: `go work use ./apps/command-line ./libs/compiler ./libs/parser`.
- Keep CLI code focused on wiring, argument parsing, and I/O; all language semantics live in `libs/compiler` and `libs/parser`.

## CHANGELOG and releases

- The CLI follows semantic versioning. Tag releases as `apps/command-line/vX.Y.Z` and update `CHANGELOG.md` with UX/flag changes.

---

If you need a sample provider registry implementation or example wiring for `ProviderTypeRegistry`, see `libs/compiler` docs and the `providers` adapters under `libs/compiler/providers`.
