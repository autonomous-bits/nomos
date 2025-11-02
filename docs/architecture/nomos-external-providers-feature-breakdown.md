# External Providers Architecture — Feature Breakdown

Last updated: 2025-10-29

This document breaks down the work to migrate Nomos providers from in-process libraries to out-of-process executables started as subprocesses and communicating with the compiler via gRPC, inspired by Terraform’s provider model. Distribution is decentralized: providers are obtained from GitHub Releases or a local file system path; there’s no central registry.

References:
- docs/guides/terraform-providers-overview.md (summary of Terraform’s model)
- Current compiler provider interfaces in `libs/compiler/provider.go`, `provider_type_registry.go`, and import resolution in `internal/imports/`

## Goals

- Treat providers as separate executables managed by the Nomos compiler at build time.
- Add `nomos init` to discover, install, and lock provider binaries into a well-known local location: `.nomos/providers/{provider-name}/{version}/{os-arch}/provider`.
- On `nomos build`, have the compiler start provider subprocesses on-demand and communicate via gRPC, preserving the existing `Provider` interface contract (Init, Fetch, optional Info) at the compiler boundary.
- Support offline/local installation (copy/move) and GitHub Releases download with checksums. No central registry.

## Non-Goals

- Implement remote apply/plan semantics (Nomos compiles snapshots; no runtime orchestration here).
- Build a public registry or discovery service.
- Refactor the Nomos language semantics beyond what’s needed to declare providers and versions.
- Maintain backward compatibility with in-process providers. This is a breaking change: in-process providers will be removed and existing projects must migrate to external providers.

## Terminology

- Provider: External binary implementing the Nomos Provider gRPC service.
- Provider Alias: The name referenced in `.csl` source declarations (e.g., `source file as configs { ... }`).
- Provider Type: The provider implementation type (e.g., `file`, `http`). Maps to an executable.
- Provider Source: Where to obtain the binary (GitHub or local path).

## User-facing changes

- New command: `nomos init`
  - Scans project sources to identify required providers and versions.
  - Resolves and installs providers from GitHub Releases into `.nomos/providers/...` (or records pre-installed local binaries).
  - Writes a lock file `.nomos/providers.lock.json` with resolved versions, sources, OS/arch, and checksums.
  - Flags: `--upgrade`, `--os`, `--arch`, `--dry-run`, `--force`.

- Build remains `nomos build`, with behavior changes:
  - Compiler starts provider subprocesses on first use and communicates over gRPC.
  - Uses the lock file to locate exact provider binaries.

- Optional manifest file (recommended): `.nomos/providers.yaml`
  - Declarative mapping from provider alias/type to version and source for reproducibility.
  - Example below.

## Language and configuration impacts

Source declarations already exist in `.csl` (via `SourceDecl`), providing `Alias`, `Type`, and a free-form `Config` map. We’ll standardize optional config keys used by tooling (not interpreted by providers directly):

- `version` (string, semver) — REQUIRED; authoritative provider version source
- `source` (object) — hints for installation:
  - `github: { owner: string, repo: string, asset?: string }`
  - `local: { path: string }`

Providers may still define their own config keys; those are passed unchanged to the provider via `Init`.

If a project prefers central config, use a manifest file:

```yaml
# .nomos/providers.yaml
providers:
  - alias: configs
    type: file
    # Version is authoritative in .csl, not here
    source:
      github:
        owner: autonomous-bits
        repo: nomos-provider-file
        # optional: asset name template override
    # optional default config injected to Init
    config:
      directory: "./apps/command-line/testdata/configs"
```

`nomos init` merges sources from `.csl` files and the manifest. The version MUST be declared in `.csl` source declarations and is authoritative. The manifest may provide source hints (e.g., GitHub owner/repo) but MUST NOT override versions. The lock file records the final resolved values.

## High-level architecture

- Compiler changes
  - Introduce a Provider Process Manager responsible for:
    - Locating binaries in `.nomos/providers/...` based on lock file
    - Starting subprocesses (per provider alias) and keeping them alive for the compilation run
    - Establishing and maintaining gRPC client connections
    - Enforcing timeouts, restart policy (limited), and clean shutdown
  - Introduce a gRPC client adapter that implements the existing `compiler.Provider` interface by delegating to the subprocess via gRPC.
  - Replace direct constructor-based providers with a “remote provider” constructor resolving to subprocess-backed clients.
  - Process model decision: one subprocess per provider alias (per-alias), started lazily and cached for the build run.

- CLI changes
  - `nomos init` executes discovery and installation/locking flow (see UX flows below).

- Provider packaging and distribution
  - Binaries are versioned artifacts published on GitHub Releases (or provided locally).
  - Asset naming convention defaults to: `nomos-provider-{type}-{version}-{os}-{arch}` (configurable per provider if needed).
  - Optional checksums file support (e.g., `SHA256SUMS`).

## gRPC service contract (provider side)

Proto package: `nomos.provider.v1`

Service `Provider`:
- `rpc Init(InitRequest) returns (InitResponse)`
- `rpc Fetch(FetchRequest) returns (FetchResponse)`
- `rpc Info(InfoRequest) returns (InfoResponse)`
- `rpc Health(HealthRequest) returns (HealthResponse)`
- `rpc Shutdown(ShutdownRequest) returns (ShutdownResponse)` (optional, best-effort)

Messages:
- `InitRequest { string alias; google.protobuf.Struct config; string source_file_path; }`
- `InitResponse {}`
- `FetchRequest { repeated string path; }`
- `FetchResponse { google.protobuf.Struct value; }`  
  Rationale: today Nomos expects `Fetch` to return `any` which is typically a `map[string]any` — `Struct` matches this shape.
- `InfoRequest {}`
- `InfoResponse { string alias; string version; string type; }`
- `HealthRequest {}`
- `HealthResponse { enum Status { UNKNOWN = 0; OK = 1; DEGRADED = 2; } Status status; string message; }`
- `ShutdownRequest {}` / `ShutdownResponse {}`

Error handling: use gRPC status codes (InvalidArgument, NotFound, DeadlineExceeded, Unavailable, Internal). Providers should surface precise diagnostics.

Note: We can generate both Go and cross-language stubs for provider authors.

## Process lifecycle and concurrency

- One subprocess per provider alias per compilation run (per-alias model).
- Lazy start on first `GetProvider(alias)`.
- Idle timeout not required initially (providers shut down when build finishes).
- Concurrency: allow multiple concurrent `Fetch` RPCs; provider authors must ensure internal safety.
- Timeouts: default per-RPC timeouts with user overrides via CLI flags or config.

## Installation layout and resolution

Well-known location:
```
.nomos/
  providers/
    {name}/
      {version}/
        {os}-{arch}/
          provider         # executable file name
          CHECKSUM         # optional checksum file
```

Resolution precedence during `build`:
1) `.nomos/providers.lock.json`
2) Inline `.csl` source declaration (authoritative for version)
3) `.nomos/providers.yaml` (manifest) for source hints only

If none found: fail with actionable error and suggest running `nomos init`.

## Lock file format (example)

```json
{
  "providers": [
    {
      "alias": "configs",
      "type": "file",
      "version": "0.2.0",
      "os": "darwin",
      "arch": "arm64",
      "source": { "github": { "owner": "autonomous-bits", "repo": "nomos-provider-file", "asset": "nomos-provider-file-0.2.0-darwin-arm64" } },
      "checksum": "sha256:...",
      "path": ".nomos/providers/file/0.2.0/darwin-arm64/provider"
    }
  ]
}
```

## Provider packaging guidance

- Recommended asset naming: `nomos-provider-{type}-{version}-{os}-{arch}`
- Provide checksum file (sha256) and publish in the release.
- Exit with non-zero status on unrecoverable init/fetch errors and surface via gRPC.
- Logging to stderr by default; include a `--log-level` flag as future-proofing (the compiler can set ENV or args).

## Security considerations

- Only execute binaries found under `.nomos/providers/` unless `--allow-external-path` is explicitly set.
- Validate SHA256 checksums on download; store in lock file.
- Never execute world-writable binaries; enforce `0700` perms on the executable.
- Sanitize environment; pass only required variables.

## Error handling and diagnostics

- Clear error categorization via gRPC status and messages.
- Compiler surfaces provider stderr when `--verbose` is enabled.
- Health RPC allows preflight checks before first Fetch.

## Migration plan

- Phase 1: Protocol and scaffolding
  - Define protobuf schema and generate stubs for Go.
  - Implement `providerproc` package in the compiler: process manager + gRPC client.
  - Migrate the in-repo `file` provider to an external repo and binary (`autonomous-bits/nomos-provider-file`). Remove the in-process provider from this monorepo.

- Phase 2: CLI `init`
  - Implement scanning (parse `.csl` via existing AST to collect `SourceDecl` alias/type/config).
  - Merge with optional `.nomos/providers.yaml` (versions from `.csl` only; manifest provides source hints).
  - Resolve sources, download from GitHub Releases (with `Accept: application/octet-stream`), validate checksum, install layout, write lock file.
  - Support manual pre-install of local provider binaries into the `.nomos/providers` layout (copy/symlink), which `nomos init` will record in the lockfile.

- Phase 3: Build integration
  - Wire compiler `ProviderRegistry` to construct a gRPC-backed provider client when an alias is requested.
  - Ensure compatibility with import resolution and `Fetch` shape (maps via Struct).
  - Add graceful shutdown at end of compile.

- Phase 4: Stabilization
  - Back-compat: not supported. Remove legacy in-process constructors. Provide clear migration errors instructing to run `nomos init` and use external providers.
  - Docs, examples, and a provider author guide.

## Detailed work breakdown

### A. Protocol and adapters
- Define `proto/nomos/provider/v1/provider.proto` with messages and service above.
- Generate Go code and add module to workspace (e.g., `libs/provider-proto`).
- Implement `compiler/internal/providerproc`:
  - `Manager` (resolve, start, connect, shutdown)
  - `Client` (gRPC wrapper implementing `compiler.Provider`)
  - Context/timeouts, retries (bounded), stderr capture
- Replace/extend `ProviderTypeRegistry` to support external providers:
  - New type constructor that returns a `Client` bound to a binary path derived from the alias/type via lock file.

Acceptance:
- Unit tests for Manager (start/stop, missing binary, bad checksum)
- Contract tests against a fake provider server

### B. CLI: `nomos init`
- Add new subcommand scaffold under `apps/command-line/cmd/nomos/init.go`.
- Discovery:
  - Parse all project `.csl` files and collect `SourceDecls` (reuse `imports.ExtractImports`).
  - Merge with `.nomos/providers.yaml` if present (only for source hints, not version).
- Resolution:
  - For each provider: choose version/source, compute asset name, download or copy from local path.
  - Verify checksum (if available), set execute perms, write to `.nomos/providers/...`.
  - Update `.nomos/providers.lock.json` (create if missing). Lock file reflects versions from `.csl`.
  - Flags: `--upgrade`, `--dry-run`, `--force`.

Acceptance:
- Integration test: given a manifest and a local path, `init` installs to expected location and writes lock file.

### C. Build integration
  - Modify compiler options to accept a lock-file reader/resolver.
  - In `ProviderRegistry.GetProvider`, resolve alias to type and binary path using the lock file (version from `.csl`), then use `providerproc.Manager` to start/connect and return a `Client` implementing `Provider`.
- Ensure `imports.ResolveImports` flow remains unchanged at call sites.

Acceptance:
- Integration test: compile a project that imports via `file` provider served by an out-of-process provider.

### D. Provider reference implementation
- Create external repo `autonomous-bits/nomos-provider-file`:
  - Implements the gRPC service
  - Supports `Init` with `directory` and `Fetch` as today’s semantics
  - Releases with assets for darwin/arm64, darwin/amd64, linux/amd64

Acceptance:
- E2E test: `nomos init` + `nomos build` using the released binary.

### E. Documentation
- Update `docs/guides/terraform-providers-overview.md` cross-links.
- Add provider authoring guide with protobuf contract and packaging instructions.

## UX flows

### `nomos init`
1. Parse sources, collect providers {alias, type, optional version/source}.
2. Merge with `.nomos/providers.yaml` (manifest may add source hints only; versions must come from `.csl`).
3. For each provider:
   - If lock entry exists and `--upgrade` not set: skip
   - Else: resolve source
  - GitHub: compute asset name default `nomos-provider-{type}-{version}-{os}-{arch}`; download; verify checksum
     - Local: copy or symlink `provider` into target layout
   - Write/update lock entry
4. Output summary and next steps.

### `nomos build`
1. Load lock file; error if missing required providers and suggest `nomos init`.
2. On first use of provider alias:
   - Start subprocess for the corresponding binary
   - Call `Init` with alias, config, and source file path
   - Optionally call `Health`
3. On import resolution, call `Fetch` as needed; merge results as today.
4. On completion, cleanly shut down all provider subprocesses.

## Edge cases

- Missing version in `.csl`: fail at `init`. Versions MUST be specified in `.csl` source declarations. Optionally, support “latest” GitHub release with `--allow-latest` in a future iteration.
- Provider binary not executable: fix perms and retry; else error with remediation.
- OS/arch mismatch: allow overriding `--os`/`--arch` or error clearly.
- Provider returns non-map from `Fetch`: preserve current compiler error.
- Crash loops: backoff and give up after N attempts with surfaced logs.

## Success criteria

- Builds complete using external provider executables with no regressions to import semantics.
- `nomos init` reliably installs and locks provider binaries.
- Clear, actionable errors for missing/invalid providers.
- Reference provider demonstrates authoring flow and compatibility.

## Decisions

- Process model: per-alias processes (one subprocess per provider alias per build run).
- Version source of truth: standardized in `.csl` source declarations; manifests cannot override version.
- Windows support: deferred (not in the first iteration).

## Provider project structure (external repos)

Recommended repository layout for external providers (example: `autonomous-bits/nomos-provider-file`):

```
nomos-provider-<type>/
├── go.mod
├── cmd/
│   └── provider/
│       └── main.go           # starts gRPC server implementing nomos.provider.v1
├── internal/                 # provider logic
├── pkg/                      # optional public packages
├── proto/                    # vendored or submodule for provider proto (if needed)
├── Makefile                  # build/test/release helpers
├── .goreleaser.yml           # release assets per os/arch
└── README.md
```

Build outputs and releases:
- Asset name: `nomos-provider-{type}-{version}-{os}-{arch}` (configurable if needed).
- Include SHA256 checksums. Provide darwin/arm64, darwin/amd64, linux/amd64 initially.

Implementation notes:
- Import generated stubs for `nomos.provider.v1`.
- Implement RPCs: `Init`, `Fetch`, `Info`, `Health`, `Shutdown`.
- Map Init config 1:1 with `.csl` `Config` fields (e.g., for `file` provider, `directory`).

Migration of the in-repo file provider:
- Create and publish `autonomous-bits/nomos-provider-file` implementing current semantics.
- Remove `libs/compiler/providers/file` from this monorepo.
- Update compiler to treat `file` as an external provider only.
