# Provider Downloader Module Agent

## Purpose
Specialized agent for `libs/provider-downloader` - handles provider binary resolution, downloading, version management, and caching for the Nomos ecosystem.

## Module Context
- **Path**: `libs/provider-downloader`
- **Module Name**: `github.com/autonomous-bits/nomos/libs/provider/downloader`
- **Go Version**: 1.25.3
- **Responsibilities**:
  - Asset resolution from GitHub Releases based on OS/architecture
  - Streaming downloads with SHA256 verification
  - Atomic installation with proper permissions
  - Retry logic with exponential backoff for transient failures
  - Version normalization and auto-detection
  - Error handling for network, checksum, and rate limit issues

- **Key Files**:
  - `client.go` - Client and public API methods
  - `resolver.go` - GitHub Release asset resolver (internal)
  - `download.go` - Streaming downloader with retry (internal)
  - `errors.go` - Typed errors and error handling
  - `types.go` - Public types (ProviderSpec, AssetInfo, InstallResult, etc.)
  - `client_test.go` - Unit tests for client API
  - `resolver_test.go` - Unit tests for asset resolution
  - `download_test.go` - Unit tests for downloader
  - `resolver_version_test.go` - Version resolution tests

## Delegation Instructions
For general Go questions, **consult go-expert.md**  
For testing questions, **consult testing-expert.md**  
For compiler integration, **consult compiler-module.md**

## Provider Downloader-Specific Patterns

### Version Resolution
The downloader implements semantic version resolution with automatic normalization:

- **Version normalization**: Automatically tries both `v1.0.0` and `1.0.0` formats
- **Latest version**: Empty or "latest" version fetches the latest GitHub Release
- **Exact matching**: Uses GitHub API to fetch specific release by tag
- **Fallback logic**: Tries with and without "v" prefix if initial resolution fails

```go
// Example: Version resolution patterns
spec := &downloader.ProviderSpec{
    Owner:   "autonomous-bits",
    Repo:    "nomos-provider-file",
    Version: "1.0.0",  // Will try "1.0.0" and "v1.0.0"
}

// Latest version
spec := &downloader.ProviderSpec{
    Owner: "autonomous-bits",
    Repo:  "nomos-provider-file",
    // Empty version = latest release
}
```

### Asset Resolution Strategy
Multi-stage matching strategy to find the correct binary:

**1. Exact Pattern Matching (Priority Order):**
1. `{repo}-{os}-{arch}` (e.g., `nomos-provider-file-linux-amd64`)
2. `nomos-provider-{os}-{arch}` (e.g., `nomos-provider-darwin-arm64`)
3. `{repo}-{os}` (e.g., `test-provider-linux`)

**2. Substring Matching (Fallback):**
- Case-insensitive substring matching
- Must contain both OS and architecture
- Handles common variations:
  - `amd64`, `x86_64`, `x86-64` (all match for amd64)
  - `arm64`, `aarch64` (all match for arm64)

**3. Auto-Detection:**
- If `OS` is empty: uses `runtime.GOOS`
- If `Arch` is empty: uses `runtime.GOARCH`

### Download Strategies
Multi-step robust download process:

**1. Streaming Download:**
- Downloads to temporary file (under `.nomos-tmp` or system temp)
- Computes SHA256 incrementally during download (no full buffering)
- Supports context cancellation at any stage

**2. Checksum Verification:**
- Verifies SHA256 against expected hash from `AssetInfo.Checksum`
- Returns `ChecksumMismatchError` with expected and actual values
- Prevents installation of corrupted or tampered binaries

**3. Retry Logic:**
- **Retryable errors**: 5xx server errors, timeouts, connection issues
- **Non-retryable errors**: 4xx client errors, invalid specs, checksum mismatches
- **Exponential backoff**: Delay doubles with each retry (1s, 2s, 4s, ...)
- **Jitter**: 10% randomization to prevent thundering herd
- **Configurable**: `RetryAttempts` (default: 3) and `RetryDelay` (default: 1s)
- **File reset**: Temp file truncated between retries for clean attempts

**4. Atomic Installation:**
- Sets executable permissions (0755) on downloaded binary
- Atomically renames temp file to final destination
- Handles cross-filesystem renames (falls back to copy+remove)
- Final path: `{destDir}/provider`

```go
// Example: Download with retry configuration
client := downloader.NewClient(ctx, &downloader.ClientOptions{
    RetryAttempts: 5,                  // Retry up to 5 times
    RetryDelay:    2 * time.Second,    // Start with 2s delay
    GitHubToken:   os.Getenv("GITHUB_TOKEN"), // Optional
})

result, err := client.DownloadAndInstall(ctx, asset, destDir)
if err != nil {
    // Handle after all retries exhausted
}
```

### Error Handling
Typed errors for common failure modes:

**Sentinel Errors:**
- `ErrAssetNotFound`: No matching asset found for the given spec
- `ErrChecksumMismatch`: Downloaded file checksum doesn't match expected
- `ErrInvalidSpec`: Provider spec missing required fields
- `ErrRateLimitExceeded`: GitHub API rate limit exceeded
- `ErrNetworkFailure`: Network error after all retry attempts
- `ErrNotImplemented`: Operations not yet implemented

**Typed Error Structs:**
- `AssetNotFoundError`: Provides owner, repo, version, OS, arch details
- `ChecksumMismatchError`: Provides expected and actual checksums

**Error Handling Pattern:**
```go
asset, err := client.ResolveAsset(ctx, spec)
if err != nil {
    var assetErr *downloader.AssetNotFoundError
    if errors.As(err, &assetErr) {
        log.Printf("Asset not found: %s/%s@%s (os=%s, arch=%s)",
            assetErr.Owner, assetErr.Repo, assetErr.Version,
            assetErr.OS, assetErr.Arch)
    }
    if errors.Is(err, downloader.ErrRateLimitExceeded) {
        // Handle rate limiting
    }
    return err
}
```

### Build Tags
Currently no build tags used in this module. All functionality is cross-platform using standard library.

### Module Dependencies
- **Used by**: 
  - `libs/compiler` - To fetch provider binaries during compilation
  - `apps/command-line` - Via `nomos init` command to download providers
- **Integrates with**: 
  - GitHub Releases API for asset discovery and download
  - External provider registries (GitHub-hosted providers)
- **Dependencies**: 
  - Standard library only (no external dependencies)
  - Uses `net/http` for HTTP client
  - Uses `crypto/sha256` for checksum verification

## API Design Principles

1. **Context-aware**: All public methods accept `context.Context` as first parameter
2. **Options pattern**: Use struct options for configuration (`ClientOptions`)
3. **Typed errors**: Define sentinel errors and typed error structs
4. **Atomic operations**: Download → verify → rename for safe installation
5. **Auto-detection**: Auto-detect OS/Arch from runtime when not specified
6. **Retry logic**: Handle transient failures with exponential backoff

### Public API Surface (keep minimal)
- `NewClient(ctx, opts) *Client` - Create downloader client
- `(*Client) ResolveAsset(ctx, spec) (AssetInfo, error)` - Resolve GitHub Release asset
- `(*Client) DownloadAndInstall(ctx, asset, dest) (InstallResult, error)` - Download and install

### Internal Helpers (not exported)
- Asset name inference based on common patterns
- GitHub API client wrapper
- Checksum verification
- File I/O with atomic rename

## Common Tasks

### 1. Improving Version Resolution Logic
When enhancing version resolution:
- Maintain backward compatibility with existing patterns
- Test with both `v` prefixed and non-prefixed versions
- Update `resolver_version_test.go` with new test cases
- Consider semantic versioning edge cases (pre-release, build metadata)

### 2. Download Error Handling and Retries
When working on download reliability:
- Distinguish between retryable and non-retryable errors
- Test retry logic with httptest mock servers
- Ensure file cleanup on failures
- Verify context cancellation works at all stages
- Test `download_test.go` with various failure scenarios

### 3. Cache Management
When implementing caching features:
- Consider cache invalidation strategies
- Use atomic operations for cache writes
- Store checksums alongside cached binaries
- Implement cache cleanup/expiration logic
- Test cache hit/miss scenarios

### 4. Checksum Verification
When enhancing checksum verification:
- Support multiple hash algorithms (SHA256, SHA512)
- Parse checksums from GitHub Release notes
- Handle various checksum file formats
- Test with invalid/corrupted checksums
- Update error messages with actionable details

### 5. Registry Integration
When adding new registry support:
- Abstract registry interface for multiple backends
- Implement GitHub-specific logic in separate adapter
- Support private registries with authentication
- Test with mock registry servers
- Document authentication requirements

## Testing Strategy

### Unit Tests
- **Principle**: No network calls - use httptest for all GitHub API interactions
- **Coverage**: Test each function in isolation with mocks/fakes
- **Files**: `client_test.go`, `resolver_test.go`, `download_test.go`, `errors_test.go`

### Integration Tests
- Use httptest to emulate GitHub API responses
- Test end-to-end workflows (resolve → download → install)
- Cover edge cases: missing fields, invalid checksums, network errors

### Test Execution
```bash
# Run all tests
go test ./...

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific tests
go test -run TestClient_NewClient ./...

# Run with race detector
go test -race ./...

# Benchmarks
go test -bench=. -benchmem
```

## Nomos-Specific Details

### Provider Binary Naming Convention
Nomos providers follow specific naming patterns:
- Standard format: `nomos-provider-{name}-{os}-{arch}`
- Example: `nomos-provider-file-linux-amd64`
- Alternative: `{repo}-{os}-{arch}` for non-standard repos

### Installation Path Structure
Providers are installed in a predictable directory structure:
```
.nomos/providers/{owner}/{repo}/{version}/provider
```

Example:
```
.nomos/providers/autonomous-bits/nomos-provider-file/1.0.0/provider
```

### Integration with Nomos Init
The `nomos init` command uses this library to:
1. Parse provider declarations from config files
2. Resolve provider versions and download URLs
3. Download and verify provider binaries
4. Install providers in the correct directory structure
5. Update lockfiles with resolved versions and checksums

### Provider Lifecycle Management
- **Resolution**: Determine correct binary for current platform
- **Download**: Fetch binary with integrity verification
- **Installation**: Atomically place binary in provider directory
- **Caching**: Future enhancement to cache downloaded binaries
- **Updates**: Check for newer versions and upgrade providers

### Design Considerations
- **Hermetic builds**: All tests must be hermetic (no real network calls)
- **Cross-platform**: Support Linux, macOS, Windows
- **Architecture variants**: Support amd64, arm64, 386
- **Security**: Always verify checksums before installation
- **Reliability**: Retry transient failures, fail fast on permanent errors

## Development Workflow

### TDD Approach
1. **Red**: Write failing test for new functionality
2. **Green**: Implement minimal code to make test pass
3. **Refactor**: Clean up implementation while keeping tests green

### Code Review Checklist
- [ ] All exported symbols have GoDoc comments
- [ ] Tests use httptest (no real network calls)
- [ ] Error types are properly defined and wrapped
- [ ] Context cancellation is respected
- [ ] File operations are atomic (temp file → rename)
- [ ] Retry logic distinguishes retryable vs non-retryable errors
- [ ] Tests cover edge cases and error paths

### Documentation Updates
When making changes:
1. Update GoDoc comments in source files
2. Update `README.md` with new examples
3. Update `CHANGELOG.md` with version and changes
4. Update `AGENTS.md` if development process changes
