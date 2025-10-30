package compiler

import (
	"context"
	"fmt"

	"github.com/autonomous-bits/nomos/libs/compiler/internal/config"
)

// LockfileProviderResolver wraps the internal config.LockfileProviderResolver
// and provides a public API for resolving provider binaries from lockfiles.
type LockfileProviderResolver struct {
	resolver *config.LockfileProviderResolver
}

// NewLockfileProviderResolver creates a ProviderResolver that uses the lockfile
// to resolve provider types to binary paths.
//
// Parameters:
//   - lockfilePath: Path to the lockfile (.nomos/providers.lock.json)
//   - manifestPath: Path to the manifest (.nomos/providers.yaml, optional)
//   - baseDirFunc: Function that returns the base directory for provider binaries
//     (typically returns ".nomos/providers" resolved to an absolute path)
//
// Returns an error if neither lockfile nor manifest exists, or if baseDirFunc is nil.
func NewLockfileProviderResolver(lockfilePath, manifestPath string, baseDirFunc func() string) (*LockfileProviderResolver, error) {
	resolver, err := config.NewLockfileProviderResolver(lockfilePath, manifestPath, baseDirFunc)
	if err != nil {
		return nil, fmt.Errorf("failed to create lockfile resolver: %w", err)
	}

	return &LockfileProviderResolver{
		resolver: resolver,
	}, nil
}

// ResolveBinaryPath resolves a provider type to its binary path using the lockfile.
// It returns the absolute path to the provider executable.
//
// This implements the ProviderResolver interface.
func (r *LockfileProviderResolver) ResolveBinaryPath(ctx context.Context, providerType string) (string, error) {
	return r.resolver.ResolveBinaryPath(ctx, providerType)
}
