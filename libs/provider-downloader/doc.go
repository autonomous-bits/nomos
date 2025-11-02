// Package downloader provides functionality for resolving, downloading,
// and installing provider binaries from GitHub Releases.
//
// The downloader library enables consistent, testable downloads of provider
// binaries from GitHub Releases. It handles asset resolution based on OS/architecture,
// streaming downloads with retry logic, SHA256 verification, and atomic installation.
//
// # Basic Usage
//
// Create a client and download a provider binary:
//
//	ctx := context.Background()
//	client := downloader.NewClient(ctx, &downloader.ClientOptions{
//		GitHubToken: os.Getenv("GITHUB_TOKEN"), // Optional
//	})
//
//	spec := &downloader.ProviderSpec{
//		Owner:   "autonomous-bits",
//		Repo:    "nomos-provider-file",
//		Version: "1.0.0",
//	}
//
//	asset, err := client.ResolveAsset(ctx, spec)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	result, err := client.DownloadAndInstall(ctx, asset, "./providers")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	fmt.Printf("Installed at: %s\n", result.Path)
//
// # Error Handling
//
// The library provides typed errors for common failure modes:
//
//	asset, err := client.ResolveAsset(ctx, spec)
//	if errors.Is(err, downloader.ErrAssetNotFound) {
//		// Handle missing asset
//	}
//	if errors.Is(err, downloader.ErrRateLimitExceeded) {
//		// Handle rate limit
//	}
//
// # Asset Resolution
//
// The resolver uses an intelligent matching strategy to find the correct binary:
//
//  1. Exact pattern matching (repo-os-arch, nomos-provider-os-arch)
//  2. Substring fallback matching (case-insensitive, handles arch variants)
//  3. Auto-detection of OS/Arch from runtime when not specified
//  4. Version normalization (handles v-prefix variations)
//
// Example resolution:
//
//	spec := &downloader.ProviderSpec{
//		Owner:   "autonomous-bits",
//		Repo:    "nomos-provider-file",
//		Version: "1.0.0",
//		// OS and Arch auto-detected from runtime
//	}
//	// Matches: nomos-provider-file-linux-amd64 (on Linux x86-64)
//
// # Testing
//
// All network operations are abstracted to support hermetic testing with httptest.
// See the test files for examples.
package downloader
