# Setup Go Environment Action

A composite action that sets up the Go environment with the correct version and workspace configuration for the Nomos monorepo.

## Usage

```yaml
- name: Setup Go Environment
  uses: ./.github/actions/setup-go
  with:
    go-version: '1.25.3'  # Optional, defaults to 1.25.3
```

## Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `go-version` | Go version to use | No | `1.25.3` |
| `cache-dependency-path` | Path to go.sum files for caching | No | All module go.sum files |
| `working-directory` | Working directory for workspace verification | No | `.` |

## What This Action Does

1. **Sets up Go** with the specified version using `actions/setup-go@v5`
2. **Enables caching** for faster builds using Go module cache
3. **Verifies workspace** exists and syncs it
4. **Auto-creates workspace** if missing (with warning)
5. **Displays environment info** for debugging

## Why This Action Exists

The Nomos monorepo uses a Go workspace (`go.work`) to manage multiple modules. This action ensures:

- Consistent Go version across all CI jobs
- Proper workspace initialization and synchronization
- Dependency caching for faster builds
- Clear visibility into the environment setup

## Example Output

```
=== Go Environment ===
go version go1.25.3 linux/amd64

go.work found, syncing workspace...

=== Workspace Modules ===
./libs/parser
./libs/compiler
./libs/provider-proto
./apps/command-line
```

## Maintenance

When updating the minimum Go version for the monorepo:

1. Update `go.work` with the new version
2. Update all `go.mod` files with the new version
3. Update the default `go-version` in this action
4. Update CI workflows if they specify a different version
