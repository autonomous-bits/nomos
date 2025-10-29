package compiler

import "context"

// ProviderResolver resolves provider types to binary paths using a lockfile or manifest.
// This interface is used by ProviderTypeRegistry to locate external provider binaries.
type ProviderResolver interface {
	// ResolveBinaryPath returns the absolute path to the provider binary for the given type.
	// Returns ErrProviderNotRegistered if the provider type is not found in the lockfile.
	ResolveBinaryPath(ctx context.Context, providerType string) (string, error)
}

// ProviderManager manages the lifecycle of external provider subprocesses.
// This interface abstracts the providerproc.Manager to avoid import cycles.
type ProviderManager interface {
	// GetProvider starts (if needed) and returns a Provider instance for the given alias.
	// binaryPath is the absolute path to the provider executable.
	// opts contains the provider initialization options.
	GetProvider(ctx context.Context, alias string, binaryPath string, opts ProviderInitOptions) (Provider, error)

	// Shutdown gracefully shuts down all running provider processes.
	Shutdown(ctx context.Context) error
}
