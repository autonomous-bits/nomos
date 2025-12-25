package downloader

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
)

// githubRelease represents a GitHub release response from the API.
type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
}

// githubAsset represents a release asset from the GitHub API.
type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
	ContentType        string `json:"content_type"`
}

// resolveAssetFromGitHub resolves an asset by querying the GitHub Releases API.
// It handles version normalization, asset inference, and returns a typed error
// if no matching asset is found.
func (c *Client) resolveAssetFromGitHub(ctx context.Context, spec *ProviderSpec) (*AssetInfo, error) {
	// Auto-detect OS and Arch if not specified
	targetOS := spec.OS
	if targetOS == "" {
		targetOS = runtime.GOOS
	}

	targetArch := spec.Arch
	if targetArch == "" {
		targetArch = runtime.GOARCH
	}

	// Normalize version (handle "v" prefix)
	version := normalizeVersion(spec.Version)

	// Fetch release from GitHub. If the release/tag can't be found the
	// underlying error may be an AssetNotFoundError that doesn't include
	// the resolved target OS/Arch (those are determined here). Wrap that
	// error so callers get consistent diagnostics showing the OS/Arch
	// that were used to attempt resolution.
	release, err := c.fetchRelease(ctx, spec.Owner, spec.Repo, version)
	if err != nil {
		// If the error indicates the release/tag wasn't found, return an
		// AssetNotFoundError that includes the target OS/Arch for better
		// user-facing messages.
		if _, ok := err.(*AssetNotFoundError); ok {
			return nil, &AssetNotFoundError{
				Owner:   spec.Owner,
				Repo:    spec.Repo,
				Version: version,
				OS:      targetOS,
				Arch:    targetArch,
			}
		}
		return nil, err
	}

	// Try to find matching asset using ordered matchers
	c.debugf("Searching for asset matching: repo=%s, version=%s, os=%s, arch=%s", spec.Repo, version, targetOS, targetArch)
	assetName := c.findMatchingAsset(release.Assets, spec.Repo, version, targetOS, targetArch)
	if assetName == "" {
		c.debugf("No matching asset found")
		return nil, &AssetNotFoundError{
			Owner:   spec.Owner,
			Repo:    spec.Repo,
			Version: version,
			OS:      targetOS,
			Arch:    targetArch,
		}
	}
	c.debugf("Matched asset: %s", assetName)

	// Find the asset details
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			return &AssetInfo{
				URL:         asset.BrowserDownloadURL,
				Name:        asset.Name,
				Size:        asset.Size,
				ContentType: asset.ContentType,
			}, nil
		}
	}

	// This should never happen since findMatchingAsset returned a name
	return nil, fmt.Errorf("asset %q found but details missing", assetName)
}

// fetchRelease fetches a release from the GitHub API.
// If version is empty or "latest", it fetches the latest release.
// Otherwise, it fetches the release by tag.
func (c *Client) fetchRelease(ctx context.Context, owner, repo, version string) (*githubRelease, error) {
	var url string
	if version == "" || version == "latest" {
		url = fmt.Sprintf("%s/repos/%s/%s/releases/latest", c.baseURL, owner, repo)
	} else {
		// Try both with and without "v" prefix
		url = fmt.Sprintf("%s/repos/%s/%s/releases/tags/%s", c.baseURL, owner, repo, version)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add GitHub token if provided
	if c.githubToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.githubToken)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	// Log the request URL
	c.debugf("GitHub API request: %s", url)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GitHub API request failed: %w", err)
	}
	defer resp.Body.Close()

	// Log the response status
	c.debugf("GitHub API response: HTTP %d", resp.StatusCode)

	if resp.StatusCode == http.StatusNotFound {
		// Try alternate version format (add/remove "v" prefix)
		altVersion := version
		if strings.HasPrefix(version, "v") {
			altVersion = strings.TrimPrefix(version, "v")
		} else {
			altVersion = "v" + version
		}

		altURL := fmt.Sprintf("%s/repos/%s/%s/releases/tags/%s", c.baseURL, owner, repo, altVersion)
		altReq, err := http.NewRequestWithContext(ctx, "GET", altURL, nil)
		if err != nil {
			return nil, &AssetNotFoundError{
				Owner:   owner,
				Repo:    repo,
				Version: version,
			}
		}

		if c.githubToken != "" {
			altReq.Header.Set("Authorization", "Bearer "+c.githubToken)
		}
		altReq.Header.Set("Accept", "application/vnd.github+json")

		altResp, err := c.httpClient.Do(altReq)
		if err != nil || altResp.StatusCode != http.StatusOK {
			if altResp != nil {
				altResp.Body.Close()
			}
			return nil, &AssetNotFoundError{
				Owner:   owner,
				Repo:    repo,
				Version: version,
			}
		}
		defer altResp.Body.Close()
		resp = altResp
	}

	if resp.StatusCode == http.StatusForbidden {
		// Check for rate limit
		if resp.Header.Get("X-RateLimit-Remaining") == "0" {
			return nil, ErrRateLimitExceeded
		}
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode GitHub API response: %w", err)
	}

	// Log the release tag and available assets
	c.debugf("Release tag: %s", release.TagName)
	c.debugf("Available assets (%d):", len(release.Assets))
	for i, asset := range release.Assets {
		c.debugf("  [%d] %s (size: %d, type: %s)", i+1, asset.Name, asset.Size, asset.ContentType)
		c.debugf("      URL: %s", asset.BrowserDownloadURL)
	}

	return &release, nil
}

// findMatchingAsset applies ordered matching rules to find the best asset.
// Returns the asset name if found, or empty string if no match.
func (c *Client) findMatchingAsset(assets []githubAsset, repo, version, targetOS, targetArch string) string {
	assetNames := make([]string, len(assets))
	for i, asset := range assets {
		assetNames[i] = asset.Name
	}

	// Strip "v" prefix from version for pattern matching (e.g., "v0.1.0" -> "0.1.0")
	versionNumber := strings.TrimPrefix(version, "v")

	// 1. Try exact patterns in priority order
	// Pattern format: {repo}-{version}-{os}-{arch}[.extension]
	patterns := []string{
		// With version: repo-version-os-arch (most specific, matches actual releases)
		fmt.Sprintf("%s-%s-%s-%s", repo, versionNumber, targetOS, targetArch),
		// Legacy patterns (for backwards compatibility)
		fmt.Sprintf("%s-%s-%s", repo, targetOS, targetArch),
		fmt.Sprintf("nomos-provider-%s-%s", targetOS, targetArch),
		fmt.Sprintf("%s-%s", repo, targetOS),
	}

	c.debugf("Trying exact patterns:")
	for i, pattern := range patterns {
		c.debugf("  [%d] %s", i+1, pattern)
	}

	for _, pattern := range patterns {
		for _, name := range assetNames {
			// Check exact match
			if name == pattern {
				c.debugf("Found exact match: %s", name)
				return name
			}
			// Also check with common extensions (.tar.gz, .zip, .tgz, etc.)
			if strings.HasPrefix(name, pattern+".") {
				c.debugf("Found pattern match with extension: %s (pattern: %s)", name, pattern)
				return name
			}
		}
	}
	c.debugf("No exact pattern matches found")

	// 2. Fallback: substring matching (case-insensitive)
	// Normalize arch names for matching (amd64 == x86_64)
	archVariants := []string{targetArch}
	switch targetArch {
	case "amd64":
		archVariants = append(archVariants, "x86_64", "x86-64")
	case "arm64":
		archVariants = append(archVariants, "aarch64")
	}

	c.debugf("Trying substring matching (case-insensitive) - looking for: os=%s, arch=%v, version=%s", targetOS, archVariants, versionNumber)

	// First pass: prefer matches that include version in filename
	c.debugf("First pass: looking for assets with version in name")
	for _, name := range assetNames {
		nameLower := strings.ToLower(name)
		osMatch := strings.Contains(nameLower, strings.ToLower(targetOS))
		versionMatch := strings.Contains(nameLower, strings.ToLower(versionNumber))

		archMatch := false
		for _, arch := range archVariants {
			if strings.Contains(nameLower, strings.ToLower(arch)) {
				archMatch = true
				break
			}
		}

		if osMatch && archMatch && versionMatch {
			c.debugf("Found substring match (with version): %s", name)
			return name
		}
	}
	c.debugf("No matches found with version in filename")

	// Second pass: match without version requirement (legacy support)
	c.debugf("Second pass: looking for assets without version requirement")
	for _, name := range assetNames {
		nameLower := strings.ToLower(name)
		osMatch := strings.Contains(nameLower, strings.ToLower(targetOS))

		archMatch := false
		for _, arch := range archVariants {
			if strings.Contains(nameLower, strings.ToLower(arch)) {
				archMatch = true
				break
			}
		}

		if osMatch && archMatch {
			c.debugf("Found substring match (legacy): %s", name)
			return name
		}
	}

	c.debugf("No substring matches found")
	return ""
}

// normalizeVersion normalizes version strings by ensuring they have a "v" prefix.
// If the version is empty, it returns "latest".
func normalizeVersion(version string) string {
	if version == "" {
		return "latest"
	}
	if !strings.HasPrefix(version, "v") {
		return "v" + version
	}
	return version
}
