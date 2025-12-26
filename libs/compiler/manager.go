package compiler

import (
	"context"
	"time"

	"github.com/autonomous-bits/nomos/libs/compiler/internal/core"
	"github.com/autonomous-bits/nomos/libs/compiler/internal/providers"
)

// ManagerOptions configures the Manager behavior.
type ManagerOptions struct {
	// ShutdownTimeout is the maximum time to wait for graceful provider shutdown.
	// After this timeout, providers are forcefully terminated.
	// Default: 5 seconds.
	ShutdownTimeout time.Duration
}

// Manager manages the lifecycle of external provider subprocesses.
// It starts providers on-demand, caches them per alias, and handles
// graceful shutdown with configurable timeouts.
//
// This is a public wrapper around the internal implementation.
type Manager struct {
	impl *providers.Manager
}

// NewManager creates a new Manager instance with default options.
func NewManager() *Manager {
	return &Manager{
		impl: providers.NewManager(nil),
	}
}

// NewManagerWithOptions creates a new Manager instance with the given options.
func NewManagerWithOptions(opts ManagerOptions) *Manager {
	providerOpts := &providers.ManagerOptions{
		ShutdownTimeout: opts.ShutdownTimeout,
	}
	return &Manager{
		impl: providers.NewManager(providerOpts),
	}
}

// GetProvider returns a Provider instance for the given alias.
// If the provider subprocess is not already running, it starts it
// and establishes a gRPC connection.
//
// On error, any partially initialized resources (subprocess, connection) are cleaned up.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - alias: Provider alias (e.g., "configs")
//   - binaryPath: Absolute path to the provider executable
//   - opts: Provider initialization options
//
// Returns:
//   - Provider instance that delegates to the gRPC service
//   - Error if the subprocess cannot be started or connection fails
func (m *Manager) GetProvider(ctx context.Context, alias string, binaryPath string, opts core.ProviderInitOptions) (core.Provider, error) {
	return m.impl.GetProvider(ctx, alias, binaryPath, opts)
}

// Shutdown gracefully shuts down all running provider processes.
// It first attempts graceful shutdown by calling the Shutdown RPC on each provider
// and waiting up to ShutdownTimeout for the process to exit.
// If a process doesn't exit within the timeout, it is forcefully terminated.
func (m *Manager) Shutdown(ctx context.Context) error {
	return m.impl.Shutdown(ctx)
}
