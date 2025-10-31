# External Providers Migration Guide

**Version:** v0.3.0  
**Date:** October 31, 2025  
**Status:** Breaking Change

## Overview

Starting with Nomos v0.3.0, **in-process providers have been removed**. All providers must now be external executables managed via the `nomos init` command. This change improves security, isolation, and enables a broader ecosystem of provider implementations.

## What Changed

### Before (v0.2.x and earlier)

```go
// In-process provider registration (NO LONGER SUPPORTED)
import "github.com/autonomous-bits/nomos/libs/compiler/providers/file"

providerTypeRegistry.RegisterType("file", file.NewFileProviderFromConfig)
```

Projects could use providers without any setup - the `file` provider was built directly into the compiler.

### After (v0.3.0+)

```bash
# External providers must be installed via nomos init
nomos init --from configs=/path/to/nomos-provider-file config.csl
```

Providers are now separate executables started as subprocesses and managed by the Nomos compiler.

## Migration Steps

### 1. Update Your .csl Files

Ensure all source declarations include a `version` field:

```yaml
# Before (missing version)
source:
  alias: 'configs'
  type: 'file'
  directory: './shared-configs'

# After (with version - REQUIRED)
source:
  alias: 'configs'
  type: 'file'
  version: '0.2.0'  # ← Required in v0.3.0+
  directory: './shared-configs'
```

### 2. Run `nomos init`

Install provider binaries for your project:

```bash
# Install from local binary
nomos init --from configs=/path/to/nomos-provider-file config.csl

# Preview what would be installed (dry-run)
nomos init --dry-run config.csl
```

This creates:
- `.nomos/providers/{type}/{version}/{os-arch}/provider` — Installed binaries
- `.nomos/providers.lock.json` — Lock file with resolved versions and paths

### 3. Commit the Lock File

Add `.nomos/providers.lock.json` to your version control:

```bash
git add .nomos/providers.lock.json
git commit -m "feat: add provider lockfile for external providers"
```

**Do NOT commit the `.nomos/providers/` directory** - it contains binaries that are platform-specific.

Add to `.gitignore`:

```gitignore
# Nomos provider binaries (platform-specific)
.nomos/providers/
```

### 4. Build Your Project

```bash
nomos build -p config.csl
```

If you see errors like:

```
provider type "file" not found: external providers are required (in-process providers removed in v0.3.0). 
Run 'nomos init' to install provider binaries.
```

This means you need to run `nomos init` (step 2) first.

## For Provider Authors

If you maintained an in-process provider, you must now distribute it as a standalone executable.

### Migration Checklist

1. **Extract provider logic** into a separate repository
2. **Implement the gRPC protocol** defined in `libs/provider-proto`
3. **Publish releases** on GitHub with platform-specific binaries
4. **Document installation** for users

### Example: File Provider

The Nomos file provider has been extracted to:

- Repository: [`autonomous-bits/nomos-provider-file`](https://github.com/autonomous-bits/nomos-provider-file)
- Releases: Binary assets for darwin/arm64, darwin/amd64, linux/amd64

Users install it with:

```bash
nomos init --from configs=/path/to/downloaded/nomos-provider-file config.csl
```

## Removed Packages

The following Go packages have been **removed** and are no longer available:

- `github.com/autonomous-bits/nomos/libs/compiler/providers/file`
  - `RegisterFileProvider()` — Removed
  - `NewFileProviderFromConfig()` — Removed
  - `FileProvider` type — Removed

### Code Migration

If your code imported the file provider directly:

```go
// ❌ NO LONGER WORKS (v0.3.0+)
import "github.com/autonomous-bits/nomos/libs/compiler/providers/file"

registry := compiler.NewProviderTypeRegistry()
registry.RegisterType("file", file.NewFileProviderFromConfig)
```

**Solution:** Remove the import and registration. Use `nomos init` to install the external provider binary instead.

## CI/CD Integration

### GitHub Actions Example

```yaml
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      
      - name: Install Nomos CLI
        run: |
          go install github.com/autonomous-bits/nomos/apps/command-line/cmd/nomos@latest
      
      - name: Download file provider
        run: |
          curl -L -o nomos-provider-file \
            https://github.com/autonomous-bits/nomos-provider-file/releases/download/v0.2.0/nomos-provider-file-0.2.0-linux-amd64
          chmod +x nomos-provider-file
      
      - name: Install providers
        run: nomos init --from configs=./nomos-provider-file config.csl
      
      - name: Build configuration
        run: nomos build -p config.csl -o snapshot.json
```

### Docker Example

```dockerfile
FROM golang:1.22-alpine AS builder

# Install Nomos CLI
RUN go install github.com/autonomous-bits/nomos/apps/command-line/cmd/nomos@latest

# Download provider binary
RUN wget -O /usr/local/bin/nomos-provider-file \
    https://github.com/autonomous-bits/nomos-provider-file/releases/download/v0.2.0/nomos-provider-file-0.2.0-linux-amd64 && \
    chmod +x /usr/local/bin/nomos-provider-file

WORKDIR /workspace
COPY . .

# Install providers and build
RUN nomos init --from configs=/usr/local/bin/nomos-provider-file config.csl && \
    nomos build -p config.csl -o snapshot.json
```

## Troubleshooting

### Error: "provider type 'file' not found"

**Cause:** No lockfile exists or the provider binary is missing.

**Solution:**

```bash
nomos init --from configs=/path/to/provider-binary config.csl
```

### Error: Lockfile is malformed

**Cause:** `.nomos/providers.lock.json` contains invalid JSON.

**Solution:**

```bash
rm .nomos/providers.lock.json
nomos init --from configs=/path/to/provider-binary config.csl
```

### Error: Binary not found after init

**Cause:** The provider binary path in the lockfile is incorrect or the binary was deleted.

**Solution:**

```bash
nomos init --force --from configs=/path/to/provider-binary config.csl
```

### Provider crashes during build

**Cause:** The provider binary is incompatible or corrupt.

**Solution:**

1. Re-download the provider binary
2. Verify checksum (if provided)
3. Run init again:

```bash
nomos init --force --from configs=/path/to/new-binary config.csl
```

## Benefits of External Providers

1. **Security:** Providers run in isolated subprocesses with limited permissions
2. **Flexibility:** Use any language to implement providers (not just Go)
3. **Versioning:** Lock provider versions for reproducible builds
4. **Ecosystem:** Independent release cycles for providers and compiler
5. **Distribution:** Decentralized provider distribution via GitHub Releases

## Support

- **Documentation:** [External Providers Architecture](../architecture/nomos-external-providers-feature-breakdown.md)
- **File Provider:** [nomos-provider-file GitHub](https://github.com/autonomous-bits/nomos-provider-file)
- **Issues:** [nomos/issues](https://github.com/autonomous-bits/nomos/issues)

## Version Compatibility

| Nomos Version | In-Process Providers | External Providers | Notes |
|---------------|---------------------|-------------------|-------|
| v0.1.x - v0.2.x | ✅ Supported | ❌ Not Available | In-process only |
| v0.3.0+ | ❌ Removed | ✅ Required | External only (breaking change) |

---

**Last Updated:** October 31, 2025  
**Applies To:** Nomos v0.3.0 and later
