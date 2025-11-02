# Migration Guide: Removing Replace Directives

This guide helps you migrate from using `replace` directives to consuming published Nomos library modules.

## Background

Previously, external projects depending on Nomos libraries had to use `replace` directives pointing to local checkouts:

```go
require (
    github.com/autonomous-bits/nomos/libs/compiler v0.0.0-00010101000000-000000000000
)

replace github.com/autonomous-bits/nomos/libs/compiler => ../nomos/libs/compiler
replace github.com/autonomous-bits/nomos/libs/parser => ../nomos/libs/parser
```

This approach had several limitations:
- ❌ Doesn't work in CI/CD without special setup
- ❌ Requires specific directory layout
- ❌ Difficult for external contributors
- ❌ No proper versioning

Now that Nomos libraries are published as proper Go modules with semantic versions, you can use standard `require` directives:

```go
require (
    github.com/autonomous-bits/nomos/libs/compiler v0.1.0
    github.com/autonomous-bits/nomos/libs/parser v0.1.0
    github.com/autonomous-bits/nomos/libs/provider-proto v0.1.0
)
```

Benefits:
- ✅ Works in CI/CD out of the box
- ✅ No directory layout requirements
- ✅ Easy for external contributors
- ✅ Proper semantic versioning
- ✅ Standard Go tooling support

## Migration Steps

### Step 1: Update go.mod

Edit your `go.mod` file to use versioned `require` directives:

**Before:**
```go
module github.com/example/my-project

go 1.22

require (
    github.com/autonomous-bits/nomos/libs/compiler v0.0.0-00010101000000-000000000000
    github.com/autonomous-bits/nomos/libs/parser v0.0.0-00010101000000-000000000000
)

replace github.com/autonomous-bits/nomos/libs/compiler => ../nomos/libs/compiler
replace github.com/autonomous-bits/nomos/libs/parser => ../nomos/libs/parser
```

**After:**
```go
module github.com/example/my-project

go 1.22

require (
    github.com/autonomous-bits/nomos/libs/compiler v0.1.0
    github.com/autonomous-bits/nomos/libs/parser v0.1.0
    github.com/autonomous-bits/nomos/libs/provider-proto v0.1.0
)
```

### Step 2: Download Dependencies

Run `go mod tidy` to download the published modules:

```bash
go mod tidy
```

This will:
- Download the specified versions from GitHub
- Update `go.sum` with checksums
- Remove any unused dependencies

### Step 3: Verify Builds

Verify your project builds successfully with the published modules:

```bash
go build ./...
go test ./...
```

### Step 4: Update CI/CD

If your CI/CD pipeline had special handling for `replace` directives, you can now remove it. Standard Go commands will work:

```yaml
# GitHub Actions example - simplified
- name: Build
  run: |
    go mod download
    go build ./...
    
- name: Test
  run: go test -v ./...
```

No need for:
- Checking out the Nomos repository
- Setting up directory structure
- Environment variable hacks

## Automated Migration Script

For projects with many `replace` directives, use this script:

```bash
#!/bin/bash
# remove-replace-directives.sh

# Backup go.mod
cp go.mod go.mod.backup

# Remove replace directives for Nomos libraries
sed -i.tmp '/replace github.com\/autonomous-bits\/nomos\/libs/d' go.mod
rm -f go.mod.tmp

# Update pseudo-versions to v0.1.0 (adjust version as needed)
sed -i.tmp 's/v0\.0\.0-00010101000000-000000000000/v0.1.0/g' go.mod
rm -f go.mod.tmp

# Tidy dependencies
go mod tidy

echo "Migration complete! Review changes with: diff go.mod.backup go.mod"
```

Usage:

```bash
chmod +x remove-replace-directives.sh
./remove-replace-directives.sh
```

## For Local Development (Nomos Monorepo Contributors)

If you're working within the Nomos monorepo itself, **do not use `replace` directives**. Instead, use Go workspaces:

### The go.work file handles module resolution

The repository root has a `go.work` file that configures all modules:

```go
go 1.25.3

use (
    ./apps/command-line
    ./examples/consumer
    ./libs/compiler
    ./libs/parser
    ./libs/provider-proto
)
```

### Workflow for monorepo development:

```bash
# Clone and enter repository
git clone https://github.com/autonomous-bits/nomos.git
cd nomos

# Sync workspace (done automatically by go commands)
go work sync

# Build everything
make build

# Test everything
make test

# Work on a specific module
cd libs/compiler
go test ./...
```

### Adding a new module to the workspace:

```bash
# From repository root
go work use ./path/to/new/module
go work sync
```

## Troubleshooting

### Problem: "go: no matching versions for query"

**Cause**: The specified version hasn't been published yet.

**Solution**: Check available versions:

```bash
go list -m -versions github.com/autonomous-bits/nomos/libs/compiler
```

Use an available version or wait for the desired version to be published.

### Problem: "replace directives still being used"

**Cause**: Local `go.mod` still has `replace` directives.

**Solution**: Remove all `replace` directives for Nomos libraries:

```bash
# Remove replace directives
sed -i '' '/replace github.com\/autonomous-bits\/nomos/d' go.mod
go mod tidy
```

### Problem: "module not found" in CI

**Cause**: CI environment can't access the published modules.

**Solution**: Ensure:
1. Modules are published with proper tags
2. Repository is public or CI has access
3. Using `go mod download` before build

### Problem: "checksum mismatch"

**Cause**: Module checksum changed (rare, usually indicates tampering).

**Solution**: Clear cache and re-download:

```bash
go clean -modcache
go mod download
```

If problem persists, verify the module tags haven't been force-pushed.

## Version Selection

### Using Specific Versions

```go
require (
    github.com/autonomous-bits/nomos/libs/compiler v0.1.0  // Exact version
)
```

### Using Version Ranges (Go 1.17+)

Go doesn't support ranges directly, but you can:

```bash
# Get latest minor/patch for v0.x
go get github.com/autonomous-bits/nomos/libs/compiler@v0

# Get latest patch for v0.1.x
go get github.com/autonomous-bits/nomos/libs/compiler@v0.1

# Get specific version
go get github.com/autonomous-bits/nomos/libs/compiler@v0.1.2
```

### Using Latest

```bash
# Get latest version (including pre-releases)
go get github.com/autonomous-bits/nomos/libs/compiler@latest

# Get latest stable (no pre-releases)
go get github.com/autonomous-bits/nomos/libs/compiler@upgrade
```

## Example Projects

### Minimal Example

See [examples/consumer/](../../examples/consumer/README.md) for a minimal working example.

### Provider Example

See [docs/examples/local-provider/](../docs/examples/local-provider/) for a provider implementation example.

## Getting Help

- **Documentation**: [Go Modules Reference](https://go.dev/ref/mod)
- **Issues**: [GitHub Issues](https://github.com/autonomous-bits/nomos/issues)
- **Discussions**: [GitHub Discussions](https://github.com/autonomous-bits/nomos/discussions)

## Changelog

- **2025-11-02**: Initial migration guide created
- Libraries published with v0.1.0 tags
- Removed replace directives from examples

---

**Next Steps**: After migrating, consider contributing back to Nomos or creating your own providers!
