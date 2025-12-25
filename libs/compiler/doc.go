// Package compiler provides functionality to parse and compile Nomos scripts into
// configuration snapshots. It integrates with the parser library to process .csl files,
// resolves external references via provider plugins, and produces deterministic output
// with comprehensive metadata and provenance tracking.
//
// # Overview
//
// The compiler orchestrates the entire compilation process:
//
//   - Parsing .csl files via the parser library
//   - Managing external provider subprocesses via gRPC
//   - Resolving references and imports with cycle detection
//
// # Architecture
//
// The Manager follows a per-alias process model: one subprocess per provider alias
// per compilation run. Processes are started lazily on first use and cached for
// subsequent calls. All processes are gracefully terminated when Shutdown is called.
//
// # Usage Example
//
//	manager := providerproc.NewManager()
//	defer manager.Shutdown(context.Background())
//
//	opts := compiler.ProviderInitOptions{
//	    Alias: "configs",
//	    Config: map[string]any{"directory": "./configs"},
//	}
//
//	provider, err := manager.GetProvider(ctx, "configs", "/path/to/provider", opts)
//	if err != nil {
//	    return err
//	}
//
//	// Use provider as normal compiler.Provider
//	data, err := provider.Fetch(ctx, []string{"database", "prod"})
//
// # Process Lifecycle
//
//  1. Binary validation: Verify executable exists
//  2. Subprocess start: Launch provider with context cancellation support
//  3. Port discovery: Read provider's listening port from stdout
//  4. Connection: Establish gRPC client connection
//  5. Health check: Verify provider is ready via Health RPC
//  6. Caching: Store client for reuse by alias
//  7. Shutdown: Gracefully terminate all processes on Manager.Shutdown
//
// # Security
//
// Providers should only be executed from trusted locations (typically .nomos/providers/).
// The Manager captures stderr for debugging but does not execute world-writable binaries.
//
// # Error Handling
//
// Errors are surfaced at multiple levels:
//   - Binary not found: Immediate error with remediation
//   - Connection failures: Retry with exponential backoff (future enhancement)
//   - RPC failures: Delegate to gRPC error codes with context
//   - Subprocess crashes: Captured stderr included in error messages
//
// # Thread Safety
//
// The Manager is safe for concurrent use. Multiple goroutines can call GetProvider
// simultaneously; the Manager ensures only one subprocess is started per alias.
package compiler
