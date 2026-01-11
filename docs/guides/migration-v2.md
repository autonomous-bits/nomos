# Migration Guide: Auto-Download Providers in Build

**Feature**: Remove Init Command and Auto-Download Providers  
**Version**: v2.0.0  
**Date**: 2026-01-10

## Overview

Starting in Nomos CLI v2.0.0, the `nomos init` command has been **removed**. Provider installation now happens automatically during `nomos build`, simplifying the workflow from two commands to one.

This guide helps users migrate from the old two-step workflow to the new streamlined process.

---

## What Changed

### Before (v1.x)

Two-step workflow required running `init` before `build`:

```bash
# Step 1: Install providers
nomos init config.csl

# Step 2: Build configuration
nomos build config.csl -o output.json
```

### After (v2.0.0)

Single-step workflow automatically downloads providers:

```bash
# Providers downloaded automatically before build
nomos build config.csl -o output.json
```

---

## Breaking Changes

### Removed Command

**Impact**: `nomos init` command no longer exists.

**Error Message** (if attempted):
```
Error: unknown command "init" for "nomos"

The 'init' command has been removed in v2.0.0.
Providers are now automatically downloaded during 'nomos build'.

Migration:
  Old: nomos init config.csl && nomos build config.csl
  New: nomos build config.csl

For more information, see: https://github.com/autonomous-bits/nomos/docs/migration-guide.md
```

**Migration**: Remove `nomos init` from scripts and workflows. Use `nomos build` directly.

---

### Automatic Provider Downloads

**Impact**: First `build` after upgrade will download providers if not already cached.

**User Experience**:
```bash
$ nomos build config.csl
Checking providers...
↓ Downloading file (autonomous-bits/nomos-provider-file@1.0.0) - 2.1 MB
↓ Downloading aws (autonomous-bits/nomos-provider-aws@2.1.0) - 5.2 MB
(2 downloaded, 0 cached)

Compiling config.csl...
[compilation output]
```

**Subsequent Builds** (providers cached):
```bash
$ nomos build config.csl
Checking providers...
(all cached)

Compiling config.csl...
[compilation output]
```

**No Action Required**: Existing `.nomos/providers/` directory and `providers.lock.json` continue to work without changes.

---

## Migration Scenarios

### 1. Local Development

**Old Workflow** (v1.x):
```bash
# One-time setup
nomos init config.csl

# Daily development
nomos build config.csl
nomos build config.csl  # Fast, uses cache
```

**New Workflow** (v2.0.0):
```bash
# First build (downloads providers)
nomos build config.csl

# Subsequent builds (uses cache)
nomos build config.csl  # Fast, same as before
```

**Impact**: ✅ Minimal. First build after upgrade downloads providers. Subsequent builds are identical to v1.x.

---

### 2. CI/CD Pipelines

**Old Workflow** (v1.x):
```yaml
# .github/workflows/build.yml
steps:
  - name: Install providers
    run: nomos init config.csl
    
  - name: Build configuration
    run: nomos build config.csl -o output.json
```

**New Workflow** (v2.0.0):
```yaml
# .github/workflows/build.yml
steps:
  - name: Build configuration
    run: nomos build config.csl -o output.json
    # Providers downloaded automatically
```

**Impact**: ✅ Simpler. Remove separate init step. Build time unchanged (downloads still occur, just integrated).

---

### 3. Docker Builds

**Old Workflow** (v1.x):
```dockerfile
# Dockerfile
RUN nomos init /configs/*.csl
RUN nomos build /configs/*.csl -o /output/config.json
```

**New Workflow** (v2.0.0):
```dockerfile
# Dockerfile
RUN nomos build /configs/*.csl -o /output/config.json
# Providers cached in .nomos/ layer
```

**Optimization** (cache providers across builds):
```dockerfile
# Dockerfile with provider caching
COPY configs/ /configs/

# Cache provider downloads in Docker layer
RUN nomos build --dry-run /configs/*.csl || true
# Downloads providers to .nomos/, build fails (expected)

# Actual build reuses cached providers
RUN nomos build /configs/*.csl -o /output/config.json
```

**Impact**: ⚠️ Slightly longer build time if providers not cached in Docker layer. Use `--dry-run` trick to pre-cache providers in a separate layer.

---

### 4. Pre-Production Testing

**Old Workflow** (v1.x):
```bash
# Preview provider installations
nomos init --dry-run config.csl

# Install providers
nomos init config.csl

# Validate configuration
nomos build config.csl
```

**New Workflow** (v2.0.0):
```bash
# Preview provider downloads
nomos build --dry-run config.csl

# Build with automatic provider installation
nomos build config.csl
```

**Impact**: ✅ Equivalent functionality. Use `--dry-run` on build instead of init.

---

### 5. Cross-Platform Builds

**Old Workflow** (v1.x):
```bash
# Install Linux providers on macOS
nomos init --os linux --arch amd64 config.csl

# Build for Linux
nomos build config.csl -o linux-config.json
```

**New Workflow** (v2.0.0):
```bash
# Not directly supported via build flags
# Workaround: Manually edit lockfile or use separate .nomos/ directories
```

**Impact**: ⚠️ **Limitation**: v2.0.0 does not support `--os` and `--arch` flags on build command. Providers are downloaded for the current platform only.

**Future Enhancement**: These flags may be added in a future release if user demand justifies the complexity.

**Workaround**:
```bash
# Build in Docker container matching target platform
docker run --platform linux/amd64 nomos-cli:v2.0.0 nomos build config.csl
```

---

## Feature Parity

### Preserved Features

All `init` command functionality is preserved in the new automatic download mechanism:

| Feature | v1.x (init) | v2.0.0 (build) | Status |
|---------|-------------|----------------|--------|
| **GitHub Releases Download** | ✅ | ✅ | Identical |
| **Provider Caching** | ✅ | ✅ | Identical |
| **Lockfile Management** | ✅ | ✅ | Identical |
| **Checksum Verification** | ✅ | ✅ | Enhanced (auto-retry) |
| **Dry-Run Preview** | ✅ `--dry-run` | ✅ `--dry-run` | Identical |
| **Force Re-Download** | ✅ `--force` | ✅ `--force-providers` | Renamed flag |
| **GitHub Token Support** | ✅ `GITHUB_TOKEN` | ✅ `GITHUB_TOKEN` | Identical |
| **Concurrent Downloads** | ✅ | ✅ | Identical (via build flags) |
| **Timeout Configuration** | ✅ | ✅ | Identical (via build flags) |

### New Features

Enhancements in v2.0.0:

| Feature | Description |
|---------|-------------|
| **Automatic Retry** | Provider checksum failures automatically retry once before failing |
| **Progress Feedback** | Always shows "Checking providers..." with cached/downloaded status |
| **Lockfile Preservation** | Lockfile entries are never automatically pruned (safer) |

### Removed Features

Features not carried forward to v2.0.0:

| Feature | v1.x | v2.0.0 | Workaround |
|---------|------|--------|------------|
| **Cross-Platform Install** | `--os`, `--arch` flags | ❌ Not supported | Use Docker with target platform |
| **Upgrade Flag** | `--upgrade` | ❌ Not supported | Use `--force-providers` to re-download |
| **Standalone Init** | `nomos init` command | ❌ Removed | Automatic during build |

---

## Troubleshooting

### Error: "unknown command 'init'"

**Cause**: You're running Nomos CLI v2.0.0 or later, which removed the `init` command.

**Solution**: Remove `nomos init` from your workflow. Use `nomos build` directly.

```bash
# Old (v1.x)
nomos init config.csl && nomos build config.csl

# New (v2.0.0+)
nomos build config.csl
```

---

### Slow First Build After Upgrade

**Cause**: First build downloads providers. This is expected behavior.

**Observation**:
```bash
$ nomos build config.csl
Checking providers...
↓ Downloading file (autonomous-bits/nomos-provider-file@1.0.0) - 2.1 MB
↓ Downloading aws (autonomous-bits/nomos-provider-aws@2.1.0) - 5.2 MB
(2 downloaded, 0 cached)
```

**Solution**: Subsequent builds will be fast (providers cached). No action needed.

---

### Provider Download Failures in CI

**Cause**: Network issues, GitHub API rate limits, or missing releases.

**Solution 1**: Use GitHub token for higher rate limits:
```bash
export GITHUB_TOKEN=ghp_xxxxxxxxxxxx
nomos build config.csl
```

**Solution 2**: Pre-cache providers in CI setup:
```yaml
# Cache .nomos/ directory across CI runs
- uses: actions/cache@v3
  with:
    path: .nomos/providers
    key: nomos-providers-${{ hashFiles('config.csl') }}
```

**Solution 3**: Use `--allow-missing-provider` to tolerate failures:
```bash
nomos build --allow-missing-provider config.csl
```

---

### Docker Builds Take Longer

**Cause**: Providers not cached in Docker layer.

**Solution**: Pre-cache providers in separate Docker layer:
```dockerfile
# Dockerfile with optimized provider caching
FROM nomos-cli:v2.0.0

# Copy configs first (changes rarely)
COPY configs/ /configs/

# Pre-download providers (separate layer)
RUN nomos build --dry-run /configs/*.csl || true

# Actual build (reuses cached providers)
COPY . /app
RUN nomos build /app/configs/*.csl -o /output/config.json
```

---

### Need to Force Re-Download Providers

**Cause**: Corrupted provider binaries or testing new versions.

**Solution**: Use `--force-providers` flag:
```bash
nomos build --force-providers config.csl
```

This deletes cached providers and re-downloads all from GitHub Releases.

---

## Rollback Plan

If you need to rollback to v1.x:

1. **Uninstall v2.0.0**:
   ```bash
   # Remove v2.0.0 binary
   rm /usr/local/bin/nomos
   ```

2. **Reinstall v1.x**:
   ```bash
   # Download latest v1.x release
   # (check GitHub releases for latest v1.x tag)
   wget https://github.com/autonomous-bits/nomos/releases/download/apps/command-line/v1.X.X/nomos-darwin-arm64
   mv nomos-darwin-arm64 /usr/local/bin/nomos
   chmod +x /usr/local/bin/nomos
   ```

3. **Restore old workflow**:
   ```bash
   nomos init config.csl
   nomos build config.csl
   ```

**Note**: Your `.nomos/` directory and `providers.lock.json` remain compatible across versions. No data loss occurs during rollback.

---

## FAQ

### Q: Do I need to delete my `.nomos/` directory?

**A**: No. Existing `.nomos/providers/` and `providers.lock.json` continue to work without changes. The build command will reuse cached providers.

---

### Q: What happens to my existing lockfile?

**A**: The lockfile format is unchanged. v2.0.0 reads v1.x lockfiles without issues. The build command preserves all existing entries and only adds new providers.

---

### Q: Can I still use `nomos init`?

**A**: No. The `init` command has been completely removed in v2.0.0. Use `nomos build` instead.

---

### Q: How do I preview provider downloads without executing?

**A**: Use `nomos build --dry-run config.csl`. This shows which providers would be downloaded without actually downloading them.

---

### Q: How do I force re-download providers?

**A**: Use `nomos build --force-providers config.csl`. This ignores the cache and re-downloads all providers from GitHub Releases.

---

### Q: Will builds take longer now?

**A**: First build after upgrade will download providers (same time as old `nomos init`). Subsequent builds are identical in speed to v1.x (uses cache).

---

### Q: Can I install providers for a different platform?

**A**: Not directly in v2.0.0. Providers are downloaded for the current platform (runtime.GOOS/GOARCH). Use Docker with the target platform to work around this limitation.

---

### Q: What if I have multiple config files with different provider versions?

**A**: The build will fail with a clear error message indicating conflicting versions. You must update your configs to use a single version per provider.

---

## Support

For additional help:
- **Documentation**: https://github.com/autonomous-bits/nomos/docs
- **Issues**: https://github.com/autonomous-bits/nomos/issues
- **Discussions**: https://github.com/autonomous-bits/nomos/discussions

---

## Changelog Reference

See `apps/command-line/CHANGELOG.md` for complete v2.0.0 release notes.

**Summary**:
- **BREAKING**: Removed `nomos init` command
- **ADDED**: Automatic provider download during `nomos build`
- **ADDED**: `--force-providers` flag for build command
- **ADDED**: `--dry-run` flag for build command
- **ENHANCED**: Provider validation with automatic retry on checksum failure
- **ENHANCED**: Progress feedback with "Checking providers..." message
