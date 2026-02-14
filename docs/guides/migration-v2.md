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

---

## Optional Metadata Output (v2.0.0+)

**Feature**: Metadata Now Opt-In via `--include-metadata` Flag  
**Version**: v2.0.0+  
**Date**: 2026-02-14

### Overview

Starting in Nomos CLI v2.0.0+, **metadata is excluded from build output by default**. This produces cleaner, production-ready configuration files without compilation metadata. Metadata is now opt-in via the `--include-metadata` flag.

This change reduces output file sizes by 30-50% and aligns with standard configuration file formats expected by tools like Kubernetes, Terraform, and Docker Compose.

---

### What Changed

#### Before (v1.x Default)

Metadata was always included in output:

```bash
nomos build -p config.csl
```

```json
{
  "data": {
    "app": "example"
  },
  "metadata": {
    "start_time": "2026-02-14T10:00:00Z",
    "end_time": "2026-02-14T10:00:01Z",
    "input_files": ["/path/to/config.csl"],
    "provider_aliases": [],
    "per_key_provenance": {},
    "errors": [],
    "warnings": []
  }
}
```

#### After (v2.0.0+ Default)

Metadata excluded by default (cleaner output):

```bash
nomos build -p config.csl
```

```json
{
  "app": "example"
}
```

#### After (v2.0.0+ with --include-metadata)

Restore v1.x behavior with explicit flag:

```bash
nomos build -p config.csl --include-metadata
```

```json
{
  "data": {
    "app": "example"
  },
  "metadata": {
    "start_time": "2026-02-14T10:00:00Z",
    "end_time": "2026-02-14T10:00:01Z",
    "input_files": ["/path/to/config.csl"],
    "provider_aliases": [],
    "per_key_provenance": {
      "app": {
        "source": "/path/to/config.csl",
        "provider_alias": ""
      }
    },
    "errors": [],
    "warnings": []
  }
}
```

---

### Why This Change

**Benefits of Metadata-Free Output:**

1. **Cleaner Production Configs**: Output matches standard format expected by infrastructure tools
2. **Smaller Files**: 30-50% size reduction without metadata
3. **Faster Parsing**: Less data to process in CI/CD pipelines and target systems
4. **Better Diffs**: Version control diffs don't include timestamp changes in metadata
5. **Standard Compliance**: Works directly with Kubernetes, Terraform, Ansible without post-processing

**When Metadata is Useful:**

- **Debugging**: Trace which source files contributed specific values
- **Auditing**: Track configuration provenance for compliance requirements
- **Tooling**: Custom scripts that parse metadata for analysis
- **Development**: Understanding compilation behavior during troubleshooting

---

### Migration Steps

#### Quick Fix: Add --include-metadata Flag

If your workflow depends on metadata (debugging, auditing, custom tooling):

```bash
# Before (v1.x)
nomos build -p config.csl -o output.json

# After (v2.0.0+ with metadata)
nomos build -p config.csl -o output.json --include-metadata
```

#### Finding Scripts to Update

Search for build commands that might need metadata:

```bash
# Find all nomos build commands in shell scripts
grep -r "nomos build" . --include="*.sh"

# Find build commands in CI/CD workflows
grep -r "nomos build" . --include="*.yml" --include="*.yaml"

# Find Makefiles with build targets
grep -r "nomos build" . --include="Makefile" --include="*.mk"
```

#### Bulk Update Scripts

For batch updates across multiple files:

```bash
# Backup your files first
find . -type f \( -name "*.sh" -o -name "*.yml" -o -name "*.yaml" \) -exec cp {} {}.bak \;

# Add --include-metadata to all build commands
find . -type f \( -name "*.sh" -o -name "*.yml" -o -name "*.yaml" \) -exec sed -i.tmp 's/nomos build /nomos build --include-metadata /g' {} \;

# Review changes
git diff

# If satisfied, commit
git add -A && git commit -m "chore: add --include-metadata for v2.0.0 compatibility"

# If not satisfied, restore backups
find . -name "*.bak" -exec bash -c 'mv "$0" "${0%.bak}"' {} \;
```

**Important:** Test the sed command on a single file first to verify it works correctly in your environment.

---

### When to Use --include-metadata

#### Use the Flag When You Need:

✅ **Debugging**
- Investigating configuration merge behavior
- Tracing which source files contributed specific values
- Understanding provider data flow

✅ **Auditing**
- Compliance requirements tracking config provenance
- Security audits requiring source attribution
- Change tracking for regulated environments

✅ **Custom Tooling**
- Scripts that parse metadata for analysis
- Automated validation against provenance rules
- Build systems that extract metadata fields

✅ **Migration/Compatibility**
- Temporary compatibility during v1.x → v2.0.0 transition
- Systems expecting v1.x output format
- Gradual rollout with fallback to old behavior

#### Don't Use the Flag For:

❌ **Production Deployments**
- Kubernetes ConfigMaps and Secrets
- Terraform variable files
- Application configuration files
- Docker Compose configs

❌ **CI/CD Artifacts**
- Build outputs for deployment
- Generated configs for infrastructure
- Release artifacts

❌ **Version Control**
- Cleaner diffs without timestamp changes
- Reduced repository size
- Better code review experience

❌ **Standard Tool Consumption**
- Tools expecting standard JSON/YAML structure
- APIs consuming configuration data
- Parsers that don't understand metadata wrapper

---

### Format-Specific Behavior

Metadata exclusion applies to all output formats:

#### JSON Format

**Without metadata (default):**
```json
{
  "app": "example",
  "region": "us-west-2"
}
```

**With metadata:**
```json
{
  "data": {
    "app": "example",
    "region": "us-west-2"
  },
  "metadata": { ... }
}
```

#### YAML Format

**Without metadata (default):**
```yaml
app: example
region: us-west-2
```

**With metadata:**
```yaml
data:
  app: example
  region: us-west-2
metadata:
  start_time: "2026-02-14T10:00:00Z"
  # ... additional metadata fields
```

#### Terraform .tfvars Format

**Without metadata (default):**
```hcl
app = "example"
region = "us-west-2"
```

**With metadata:**

⚠️ **Note**: Metadata is not compatible with tfvars format. The `--include-metadata` flag is ignored when using `--format tfvars`, as HCL variable files expect flat key-value structures. Use JSON or YAML formats if you need metadata output.

```bash
# This will ignore --include-metadata
nomos build -p config.csl --format tfvars --include-metadata
```

---

### Migration Examples

#### Example 1: CI/CD Pipeline (No Changes Needed)

**Scenario**: Production deployment pipeline

**Before (v1.x):**
```yaml
# .github/workflows/deploy.yml
- name: Build config
  run: nomos build -p k8s/config.csl -o configmap.yaml --format yaml
```

**After (v2.0.0+):**
```yaml
# .github/workflows/deploy.yml
- name: Build config
  run: nomos build -p k8s/config.csl -o configmap.yaml --format yaml
# No changes needed - cleaner output is desired
```

**Impact**: ✅ Output is now cleaner and directly usable by Kubernetes.

---

#### Example 2: Debugging Script (Add Flag)

**Scenario**: Development script that analyzes provenance

**Before (v1.x):**
```bash
#!/bin/bash
# debug-config.sh
nomos build -p config.csl -o debug-output.json
python analyze-provenance.py debug-output.json
```

**After (v2.0.0+):**
```bash
#!/bin/bash
# debug-config.sh
nomos build -p config.csl -o debug-output.json --include-metadata
python analyze-provenance.py debug-output.json
```

**Impact**: ✅ Script continues to work with explicit metadata flag.

---

#### Example 3: Makefile with Multiple Targets

**Scenario**: Makefile with production and debug builds

**Before (v1.x):**
```makefile
# Makefile
.PHONY: build debug

build:
	nomos build -p config.csl -o dist/config.json

debug:
	nomos build -p config.csl -o debug.json
```

**After (v2.0.0+):**
```makefile
# Makefile
.PHONY: build debug

build:
	nomos build -p config.csl -o dist/config.json
	# No changes - production wants clean output

debug:
	nomos build -p config.csl -o debug.json --include-metadata
	# Add flag for debugging target only
```

**Impact**: ✅ Production builds cleaner, debug builds retain metadata.

---

#### Example 4: Docker Build with Multi-Stage

**Scenario**: Multi-stage Docker build

**Before (v1.x):**
```dockerfile
FROM nomos-cli:v1.x AS builder
COPY configs/ /configs/
RUN nomos build -p /configs/*.csl -o /output/config.json
```

**After (v2.0.0+):**
```dockerfile
FROM nomos-cli:v2.0.0 AS builder
COPY configs/ /configs/
RUN nomos build -p /configs/*.csl -o /output/config.json
# No changes - cleaner output preferred for production
```

**Impact**: ✅ Smaller final image due to reduced config file size.

---

### Testing Your Migration

After migrating, verify behavior:

```bash
# Test 1: Verify default output excludes metadata
nomos build -p config.csl | jq 'has("metadata")'
# Expected: false

# Test 2: Verify flag includes metadata
nomos build -p config.csl --include-metadata | jq 'has("metadata")'
# Expected: true

# Test 3: Verify data structure
nomos build -p config.csl --include-metadata | jq 'has("data")'
# Expected: true

# Test 4: Compare file sizes
nomos build -p config.csl -o without-metadata.json
nomos build -p config.csl -o with-metadata.json --include-metadata
ls -lh *.json
# Expected: without-metadata.json is 30-50% smaller
```

---

### Troubleshooting

#### Issue: Missing Metadata Fields

**Symptom**: Script fails with "metadata field not found" error

**Cause**: Script expects v1.x output format with metadata

**Solution 1** (Recommended): Update script to use clean data:
```bash
# Old script
jq '.data.app' output.json

# New script (v2.0.0+)
jq '.app' output.json
```

**Solution 2**: Add `--include-metadata` flag:
```bash
nomos build -p config.csl --include-metadata -o output.json
```

---

#### Issue: Provenance Tracking Broken

**Symptom**: Auditing/compliance system can't find provenance info

**Cause**: System relies on `metadata.per_key_provenance`

**Solution**: Add `--include-metadata` to build command:
```bash
nomos build -p config.csl --include-metadata -o audit-output.json
```

---

#### Issue: Different Output Structure

**Symptom**: JSON structure changed from `{"data": {...}, "metadata": {...}}` to `{...}`

**Cause**: Default behavior changed in v2.0.0+

**Solution A** (Recommended): Update consuming systems to expect flat structure:
```python
# Old code (v1.x)
config = json.load(f)['data']

# New code (v2.0.0+)
config = json.load(f)
# Already flat, no ['data'] accessor needed
```

**Solution B**: Use `--include-metadata` flag temporarily:
```bash
nomos build -p config.csl --include-metadata
```

---

#### Issue: Terraform tfvars Format

**Symptom**: `--include-metadata` doesn't add metadata to `.tfvars` output

**Cause**: HCL/tfvars format doesn't support metadata structure

**Solution**: Use JSON or YAML format when metadata is needed:
```bash
# For Terraform with metadata
nomos build -p config.csl --format json --include-metadata -o vars.json
```

---

### Rollback Plan

If you need to rollback the default behavior:

**Option 1**: Downgrade to v1.x (not recommended)

See "Rollback Plan" section earlier in this migration guide.

**Option 2**: Add `--include-metadata` globally (temporary fix)

```bash
# Add alias in shell profile (~/.bashrc, ~/.zshrc)
alias nomos='nomos --include-metadata'

# Or use wrapper script
echo '#!/bin/bash
/usr/local/bin/nomos --include-metadata "$@"' > ~/bin/nomos
chmod +x ~/bin/nomos
```

**Option 3**: Update all scripts (recommended)

Add `--include-metadata` flag to scripts that need it:
```bash
find . -name "*.sh" -exec sed -i 's/nomos build /nomos build --include-metadata /g' {} \;
```

---

### FAQ

#### Q: Do I need to change my configs?

**A**: No. Your `.csl` configuration files don't need any changes. Only the build command may need the `--include-metadata` flag if metadata is required.

---

#### Q: Will my existing scripts break?

**A**: Most production scripts won't break because they typically consume the `data` section. Scripts that access `metadata` fields will need the `--include-metadata` flag.

---

#### Q: Can I get the old behavior back permanently?

**A**: Yes. Add `--include-metadata` to all build commands. Consider using a shell alias or wrapper script if you prefer metadata by default.

---

#### Q: What happens to my lockfile?

**A**: The lockfile (`.nomos/providers.lock.json`) is unaffected by this change. Metadata output is independent of provider management.

---

#### Q: Does this affect provider behavior?

**A**: No. Providers continue to function identically. This change only affects the final serialized output format.

---

#### Q: Can I use --include-metadata with YAML format?

**A**: Yes. All formats support the `--include-metadata` flag except `tfvars`, which ignores it due to HCL format constraints.

```bash
# YAML with metadata
nomos build -p config.csl --format yaml --include-metadata
```

---

#### Q: How much smaller are files without metadata?

**A**: Typically 30-50% smaller, depending on configuration complexity. Files with many keys and providers see larger savings.

**Example:**
```bash
# With metadata: 12.5 KB
nomos build -p large-config.csl --include-metadata | wc -c

# Without metadata: 6.8 KB (45% smaller)
nomos build -p large-config.csl | wc -c
```

---

#### Q: Is metadata excluded in all output formats?

**A**: Yes, metadata is excluded by default in JSON, YAML, and tfvars formats. Use `--include-metadata` to restore it (JSON and YAML only; tfvars ignores the flag).

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
