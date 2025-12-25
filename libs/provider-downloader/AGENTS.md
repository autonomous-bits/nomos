# Nomos Provider Downloader Agent-Specific Patterns

> **Note**: For comprehensive guidance, see `.github/agents/provider-downloader-module.agent.md`  
> For task coordination, start with `.github/agents/nomos.agent.md`

## Nomos-Specific Patterns

### Provider Binary Naming Convention
Nomos providers follow specific naming patterns that the resolver must handle:

**Standard format:** `nomos-provider-{name}-{os}-{arch}`
- Example: `nomos-provider-file-linux-amd64`
- Example: `nomos-provider-terraform-darwin-arm64`

**Alternative format:** `{repo}-{os}-{arch}` for non-standard repos
- Example: `test-provider-linux-amd64`

**Resolution priority:**
1. `{repo}-{os}-{arch}` (exact match)
2. `nomos-provider-{os}-{arch}` (standard pattern)
3. `{repo}-{os}` (arch-agnostic fallback)
4. Substring matching with OS/arch verification

### Installation Path Structure
Providers are installed in a predictable directory structure:

```
.nomos/providers/{owner}/{repo}/{version}/provider
```

**Examples:**
```
.nomos/providers/autonomous-bits/nomos-provider-file/1.0.0/provider
.nomos/providers/autonomous-bits/nomos-provider-terraform/2.1.0/provider
```

**Key behaviors:**
- Directory structure created atomically during installation
- Binary always named `provider` (no extension)
- Permissions set to `0755` (executable)
- Atomic rename ensures partial installations don't corrupt state

### Version Resolution Logic
Nomos-specific version resolution handles both prefixed and non-prefixed versions:

**Auto-normalization:**
- User specifies `"1.0.0"` → tries both `"1.0.0"` and `"v1.0.0"`
- User specifies `"v1.0.0"` → tries both `"v1.0.0"` and `"1.0.0"`
- Empty or `"latest"` → fetches latest GitHub Release

**GitHub API integration:**
- Uses GitHub Releases API to fetch release metadata
- Parses release assets for OS/arch matching
- Falls back to substring matching if exact pattern fails
- Respects GitHub rate limits (authenticated: 5000/hr, anonymous: 60/hr)

### Integration with Nomos Init
The `nomos init` command orchestrates provider lifecycle:

1. **Parse provider declarations** from config files
2. **Resolve provider versions** using this library
3. **Download and verify** provider binaries with checksum validation
4. **Install providers** in `.nomos/providers/{owner}/{repo}/{version}/`
5. **Update lockfiles** with resolved versions and checksums

**Example integration:**
```go
// From nomos init command
client := downloader.NewClient(ctx, &downloader.ClientOptions{
    RetryAttempts: 3,
    GitHubToken:   os.Getenv("GITHUB_TOKEN"),
})

spec := &downloader.ProviderSpec{
    Owner:   "autonomous-bits",
    Repo:    "nomos-provider-file",
    Version: "1.0.0",
    // OS/Arch auto-detected from runtime.GOOS/GOARCH
}

asset, err := client.ResolveAsset(ctx, spec)
// ... handle error

destDir := filepath.Join(".nomos/providers", spec.Owner, spec.Repo, asset.Version)
result, err := client.DownloadAndInstall(ctx, asset, destDir)
// ... lockfile updated with result.Checksum
```

### Caching Strategy
Currently **not implemented** - all downloads are fresh. Future caching considerations:

**Planned cache structure:**
```
.nomos/cache/providers/{checksum}/provider
```

**Cache key:** SHA256 checksum of the binary (ensures integrity)

**Cache invalidation:**
- By checksum (content-addressed)
- Manual cleanup via `nomos clean` command
- Cache size limits (future enhancement)

**Design rationale:**
- Checksum-based cache prevents version conflicts
- Shared cache across projects on same machine
- Safe to delete entire cache directory anytime

### Build Tags
**Currently:** No build tags used in this module.

**All functionality is cross-platform using standard library:**
- `runtime.GOOS` and `runtime.GOARCH` for platform detection
- `net/http` for GitHub API calls
- `crypto/sha256` for checksum verification
- `os.Rename` with cross-filesystem fallback

**If build tags become necessary:**
- Document in this section
- Use `//go:build` directive (not `// +build`)
- Test all build tag combinations in CI

## Related Resources

- **Feature docs**: `docs/architecture/nomos-external-providers-feature-breakdown.md`
- **Provider authoring**: `docs/guides/provider-authoring-guide.md`
- **Go monorepo structure**: `docs/architecture/go-monorepo-structure.md`
- **PRD references**: Issue #68, User Story #69

---

## Task Completion Verification

**MANDATORY**: Before completing ANY task, the agent MUST verify all of the following:

### 1. Build Verification ✅
```bash
go build ./...
```
- All code must compile without errors
- No unresolved imports or type errors

### 2. Test Verification ✅
```bash
go test ./...
go test ./... -race  # Check for race conditions
```
- All existing tests must pass
- New tests must be added for new functionality
- Race detector must report no data races
- Archive extraction tests must pass
- Minimum 85%+ coverage maintained

### 3. Linting Verification ✅
```bash
go vet ./...
golangci-lint run
```
- No `go vet` warnings
- No golangci-lint errors (warnings are acceptable if documented)
- Code follows Go best practices
- No debug fmt.Printf() statements

### 4. HTTP Client Verification ✅
- Timeout configurations are reasonable
- Error handling covers network failures
- Rate limiting is respected
- GitHub API authentication works correctly

### 5. Archive Handling Verification ✅
- tar.gz extraction works correctly
- Nested directory structures are handled
- Corrupted archives are detected
- Binary permissions (0755) are set correctly

### 6. Documentation Updates ✅
- Update CHANGELOG.md if behavior changed
- Update README.md if API changed
- Add/update code comments for new functions
- Document new ClientOptions fields

### Verification Checklist Template

When completing a task, report:
```
✅ Build: Successful
✅ Tests: XX/XX passed (YY.Y% coverage)
✅ Race Detector: Clean
✅ Linting: Clean (or list acceptable warnings)
✅ Archive Extraction: Tested and working
✅ Documentation: Updated [list files]
```

**DO NOT** mark a task as complete without running ALL verification steps and reporting results.
