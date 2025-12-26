package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// LockfileProviderResolver implements compiler.ProviderResolver using a lockfile.
// It translates provider type lookups to binary paths from the lockfile.
type LockfileProviderResolver struct {
	resolver    *Resolver
	baseDirFunc func() string // Function to get the base directory for provider binaries
}

// NewLockfileProviderResolver creates a ProviderResolver that uses the lockfile
// to resolve provider types to binary paths.
//
// Parameters:
//   - lockfilePath: Path to the lockfile (.nomos/providers.lock.json)
//   - manifestPath: Path to the manifest (.nomos/providers.yaml, optional)
//   - baseDirFunc: Function that returns the base directory for provider binaries
//     (typically returns ".nomos/providers" resolved to an absolute path)
func NewLockfileProviderResolver(lockfilePath, manifestPath string, baseDirFunc func() string) (*LockfileProviderResolver, error) {
	resolver, err := NewResolver(lockfilePath, manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create resolver: %w", err)
	}

	if baseDirFunc == nil {
		return nil, fmt.Errorf("baseDirFunc must not be nil")
	}

	return &LockfileProviderResolver{
		resolver:    resolver,
		baseDirFunc: baseDirFunc,
	}, nil
}

// ResolveBinaryPath resolves a provider type to its binary path using the lockfile.
// It returns the absolute path to the provider executable after validating its checksum.
//
// Note: This implementation assumes a 1:1 mapping between provider type and alias
// for simplicity. In practice, multiple aliases could share the same type.
// For the initial implementation, we use type as a lookup key.
func (r *LockfileProviderResolver) ResolveBinaryPath(_ context.Context, providerType string) (string, error) {
	// Get all providers and find one matching the type
	// In the current design, we use type as the lookup key
	// The compiler will call this when creating a provider from a type
	allProviders := r.resolver.GetAllProviders()

	for _, p := range allProviders {
		if p.Type == providerType {
			// Determine absolute path
			var binaryPath string
			if filepath.IsAbs(p.Path) {
				binaryPath = p.Path
			} else {
				// Resolve relative to base directory
				baseDir := r.baseDirFunc()
				binaryPath = filepath.Join(baseDir, p.Path)
			}

			// Verify the binary exists
			if _, err := os.Stat(binaryPath); err != nil {
				return "", fmt.Errorf("provider binary not found at %s: %w (run 'nomos init' to install providers)", binaryPath, err)
			}

			// Validate checksum (CRITICAL for security - MANDATORY)
			if p.Checksum == "" {
				return "", fmt.Errorf("provider binary for %s has no checksum in lockfile - refusing to execute (security risk); run 'nomos init' to regenerate lockfile with checksums", providerType)
			}
			if err := ValidateChecksum(binaryPath, p.Checksum); err != nil {
				return "", fmt.Errorf("provider binary checksum validation failed for %s: %w", providerType, err)
			}

			return binaryPath, nil
		}
	}

	return "", fmt.Errorf("provider type %q not found in lockfile; run 'nomos init' to install providers", providerType)
}
