# Nomos CLI Agent-Specific Patterns

> **Note**: For comprehensive CLI development guidance, see `.github/agents/cli-module.agent.md`  
> For task coordination, start with `.github/agents/nomos.agent.md`

## Purpose

This document contains **Nomos-specific** patterns for the CLI module. General CLI development practices (command routing, flag parsing, testing) are covered in `.github/agents/cli-module.agent.md`.

## Nomos-Specific Patterns

### Command Structure

The Nomos CLI uses a **simple switch-based routing** (not Cobra) with three core commands:
- `build` — compile `.csl` files to configuration snapshots
- `init` — discover and install provider dependencies
- `help` — show usage information

Command structure reflects **offline-first philosophy**: network operations (provider fetches) are explicit via `init`, not automatic during `build`.

### Configuration File Handling

#### Input Files
- `.csl` files (Nomos script language)
- UTF-8 lexicographic ordering via `internal/traverse` (ensures deterministic builds)
- Depth-first directory traversal
- **Reproducibility guarantee**: identical file sets produce bit-identical output

#### Provider Configuration
```
.nomos/
  providers.lock.json          # version lockfile
  providers/
    {type}/
      {version}/
        {os-arch}/
          provider             # installed binary
```

**Lockfile format:**
```json
{
  "providers": [
    {
      "type": "autonomous-bits/nomos-provider-aws",
      "version": "1.2.3",
      "path": ".nomos/providers/...",
      "os": "darwin",
      "arch": "arm64",
      "checksum": "sha256:..."
    }
  ]
}
```

#### Output Formats
- JSON (via `internal/serialize`)
- **Deterministic serialization**: sorted keys, canonical formatting
- Note: YAML/HCL may be added in future if user demand justifies it
- Used for Terraform/IaC tool integration

### Provider Commands

#### `nomos init`
Extract provider requirements from `.csl` files and install:

```bash
nomos init config.csl              # install providers
nomos init --dry-run config.csl    # preview without installing
nomos init --force config.csl      # overwrite existing
nomos init --upgrade config.csl    # upgrade to latest versions
```

**Cross-platform installs:**
```bash
nomos init --os linux --arch amd64 config.csl  # install for different platform
```

**Flow:**
1. Parse `.csl` files for `source:` declarations
2. Collect unique provider requirements (type + version)
3. Fetch from GitHub Releases via `libs/provider-downloader`
4. Write `.nomos/providers.lock.json`
5. Install binaries to `.nomos/providers/{type}/{version}/{os-arch}/`

#### Provider Discovery Pattern
Parser extracts from `.csl` syntax:
```nomos
source:
  alias: 'aws'
  type: 'autonomous-bits/nomos-provider-aws'
  version: '1.2.3'
```

### Build Command Specifics

#### Offline-First Philosophy
- Default: fail if providers missing (deterministic, CI-friendly)
- `--allow-missing-provider`: tolerate fetch failures
- `--timeout-per-provider`: network timeout control (`5s`, `1m`)
- `--max-concurrent-providers`: limit concurrent fetches (default: 4)

#### Variable Substitution
```bash
nomos build -p config.csl --var env=prod --var region=us-west
```
Passed to compiler as `vars` map for parameterized configurations.

#### Strict Mode
```bash
nomos build --strict config.csl
```
Treats warnings as errors (exit code 1). Useful for CI pipelines.

### Reference Syntax
Nomos supports cross-provider references:
```nomos
app:
  config: reference:alias:filename.path.to.value
```
Compiled by `libs/compiler` but affects CLI examples and documentation.

### Build Tags

**None currently used** — CLI is platform-agnostic. Provider binaries are platform-specific (handled by `libs/provider-downloader`).

### Test Organization

#### Hermetic Testing
- Test fixtures in `testdata/`
- Mock provider registries (avoid network calls)
- Example: `internal/initcmd/init_hermetic_test.go`

#### Integration Tests
- Located in `test/` directory
- Build binary and invoke with test inputs
- Verify output matches expected snapshots
- Test provider installation flows

#### Deterministic Test Fixtures
- Use consistent file ordering (UTF-8 lexicographic)
- Snapshot-based output verification
- Ensures reproducible test results across platforms

### Internal Package Structure

Nomos-specific internal packages:
- `internal/traverse/` — deterministic `.csl` file discovery
- `internal/serialize/` — deterministic JSON output formatting
- `internal/diagnostics/` — Nomos error/warning formatting
- `internal/initcmd/` — provider discovery and installation logic
- `internal/flags/` — CLI flag parsing (simple, not Cobra)
- `internal/options/` — compiler options builder from CLI flags

### Exit Codes

Nomos CLI exit codes:
- `0` — success
- `1` — compilation/provider errors
- `2` — usage/flag parsing errors

**Strict mode**: warnings cause exit code `1`.

### Output Conventions

- Diagnostics (errors/warnings) → **stderr**
- Compilation results → **stdout** (or file via `-o`)
- Status messages (verbose mode) → **stderr**
- Deterministic serialization for all output formats

### Module Versioning

Tagged as `apps/command-line/v1.x.x` for releases.  
See `docs/RELEASE.md` for version tagging strategy.

---

## Task Completion Verification

**MANDATORY**: Before completing ANY task, the agent MUST verify all of the following:

### 1. Build Verification ✅
```bash
go build ./cmd/nomos
./nomos --help  # Verify binary works
```
- All code must compile without errors
- Binary executes and shows help
- No unresolved imports or type errors
- All commands are accessible

### 2. Test Verification ✅
```bash
go test ./...
go test ./... -race  # Check for race conditions
go test ./test/... -v  # Integration tests
```
- All existing tests must pass
- New tests must be added for new commands/flags
- Race detector must report no data races
- Integration tests with real binary must pass
- Command output matches expectations

### 3. Linting Verification ✅
```bash
go vet ./...
golangci-lint run
```
- No `go vet` warnings
- No golangci-lint errors (warnings are acceptable if documented)
- Code follows Go best practices
- No os.Exit() in command handlers (return errors instead)

### 4. CLI UX Verification ✅
- Help text is clear and accurate
- Error messages are actionable
- Flags work as documented
- Output format is consistent
- Exit codes are correct (0 for success, non-zero for errors)

### 5. Command Integration Verification ✅
```bash
# Test actual command execution
./nomos build testdata/simple.csl
./nomos init testdata/simple.csl
./nomos --version
```
- Commands execute successfully
- File I/O works correctly
- Diagnostics display properly
- Provider initialization works

### 6. Documentation Updates ✅
- Update CHANGELOG.md if behavior changed
- Update README.md for new commands/flags
- Update help text for modified commands
- Add/update usage examples

### Verification Checklist Template

When completing a task, report:
```
✅ Build: Successful (binary executes)
✅ Tests: XX/XX passed (YY.Y% coverage)
✅ Race Detector: Clean
✅ Integration Tests: All passed
✅ Linting: Clean (or list acceptable warnings)
✅ CLI UX: Help text updated, commands work
✅ Documentation: Updated [list files]
```

**DO NOT** mark a task as complete without running ALL verification steps and reporting results.
