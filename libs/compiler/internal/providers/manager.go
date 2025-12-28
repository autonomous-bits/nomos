package providers

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/autonomous-bits/nomos/libs/compiler/internal/core"
	providerv1 "github.com/autonomous-bits/nomos/libs/provider-proto/gen/go/nomos/provider/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// DefaultShutdownTimeout is the default time to wait for graceful shutdown.
const DefaultShutdownTimeout = 5 * time.Second

// ManagerOptions configures the Manager behavior.
type ManagerOptions struct {
	// ShutdownTimeout is the maximum time to wait for graceful provider shutdown.
	// After this timeout, providers are forcefully terminated.
	// Default: 5 seconds.
	ShutdownTimeout time.Duration
}

// providerProcess represents a running provider subprocess.
type providerProcess struct {
	cmd    *exec.Cmd
	client ProviderClient
	alias  string
	conn   *grpc.ClientConn
}

// ProviderClient is an interface for provider gRPC client operations.
// This allows for better testing and decoupling.
type ProviderClient interface {
	core.Provider
	Close() error
	Shutdown(ctx context.Context) error
}

// Manager manages the lifecycle of external provider subprocesses.
// It starts providers on-demand, caches them per alias, and handles
// graceful shutdown with configurable timeouts.
type Manager struct {
	mu              sync.RWMutex
	processes       map[string]*providerProcess // keyed by alias
	shutdownTimeout time.Duration
}

// NewManager creates a new Manager instance with the given options.
// If opts is nil, default options are used.
func NewManager(opts *ManagerOptions) *Manager {
	timeout := DefaultShutdownTimeout
	if opts != nil && opts.ShutdownTimeout > 0 {
		timeout = opts.ShutdownTimeout
	}

	return &Manager{
		processes:       make(map[string]*providerProcess),
		shutdownTimeout: timeout,
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
//   - opts: Provider initialization options (currently unused, reserved for future)
//
// Returns:
//   - Provider instance that delegates to the gRPC service
//   - Error if the subprocess cannot be started or connection fails
func (m *Manager) GetProvider(ctx context.Context, alias string, binaryPath string, _ core.ProviderInitOptions) (core.Provider, error) {
	// Check if process already exists (fast path)
	m.mu.RLock()
	if proc, ok := m.processes[alias]; ok {
		m.mu.RUnlock()
		return proc.client, nil
	}
	m.mu.RUnlock()

	// Start process (hold write lock)
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if proc, ok := m.processes[alias]; ok {
		return proc.client, nil
	}

	// Verify binary exists
	if _, err := os.Stat(binaryPath); err != nil {
		return nil, fmt.Errorf("provider binary not found at %s: %w", binaryPath, err)
	}

	// Start the subprocess
	cmd := exec.CommandContext(ctx, binaryPath)
	cmd.Stderr = os.Stderr // Capture provider stderr

	// Create a pipe to read stdout (provider will print port)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start provider process: %w", err)
	}

	// Ensure cleanup on error paths
	var conn *grpc.ClientConn
	cleanup := func() {
		if conn != nil {
			_ = conn.Close()
		}
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait() // Reap zombie process
		}
	}

	// Read the port from stdout
	// Provider should print: PROVIDER_PORT=<port>
	scanner := bufio.NewScanner(stdout)
	var port int
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "PROVIDER_PORT=") {
			portStr := strings.TrimPrefix(line, "PROVIDER_PORT=")
			port, err = strconv.Atoi(portStr)
			if err != nil {
				cleanup()
				return nil, fmt.Errorf("invalid port format: %s", portStr)
			}
			break
		}
	}

	if port == 0 {
		cleanup()
		return nil, fmt.Errorf("provider did not report port")
	}

	// Connect to the provider via gRPC
	target := fmt.Sprintf("127.0.0.1:%d", port)
	conn, err = grpc.NewClient(
		target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("failed to connect to provider: %w", err)
	}

	// Verify connection by calling Health
	healthClient := providerv1.NewProviderServiceClient(conn)
	_, err = healthClient.Health(ctx, &providerv1.HealthRequest{})
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("provider health check failed: %w", err)
	}

	// Create client
	client := NewClient(conn, alias)

	// Store process
	proc := &providerProcess{
		cmd:    cmd,
		client: client,
		alias:  alias,
		conn:   conn,
	}
	m.processes[alias] = proc

	return client, nil
}

// Shutdown gracefully shuts down all running provider processes.
// It first attempts graceful shutdown by calling the Shutdown RPC on each provider
// and waiting up to ShutdownTimeout for the process to exit.
// If a process doesn't exit within the timeout, it is forcefully terminated.
//
// The context parameter allows for overall operation cancellation, but each
// provider gets its own timeout context derived from the Manager's ShutdownTimeout.
func (m *Manager) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error

	for alias, proc := range m.processes {
		if err := m.shutdownProvider(ctx, alias, proc); err != nil {
			errs = append(errs, err)
		}
	}

	// Clear processes map
	m.processes = make(map[string]*providerProcess)

	if len(errs) > 0 {
		return fmt.Errorf("shutdown encountered %d error(s): %v", len(errs), errs)
	}

	return nil
}

// shutdownProvider performs graceful shutdown of a single provider.
func (m *Manager) shutdownProvider(ctx context.Context, alias string, proc *providerProcess) error {
	// Create timeout context for this provider's shutdown
	shutdownCtx, cancel := context.WithTimeout(ctx, m.shutdownTimeout)
	defer cancel()

	// Step 1: Call Shutdown RPC (graceful)
	if err := proc.client.Shutdown(shutdownCtx); err != nil {
		// Log but continue - we'll still try to clean up
		return fmt.Errorf("shutdown RPC failed for %s: %w", alias, err)
	}

	// Step 2: Wait for process to exit gracefully
	// Use WaitDelay to ensure we don't hold references too long
	done := make(chan error, 1)
	go func() {
		err := proc.cmd.Wait()
		done <- err
	}()

	select {
	case err := <-done:
		// Process exited (gracefully or with error)
		_ = proc.client.Close()
		if err != nil && !isExpectedExitError(err) {
			return fmt.Errorf("provider %s exited with error: %w", alias, err)
		}
		return nil

	case <-shutdownCtx.Done():
		// Timeout - force kill
		_ = proc.client.Close()
		if proc.cmd.Process != nil {
			if err := proc.cmd.Process.Kill(); err != nil {
				// Still wait for the goroutine to complete
				<-done
				return fmt.Errorf("failed to kill provider %s after timeout: %w", alias, err)
			}
			// Wait for the goroutine's Wait() to complete (it should return quickly now that we killed it)
			<-done
		}
		return fmt.Errorf("provider %s did not exit within timeout, was forcefully terminated", alias)
	}
}

// isExpectedExitError checks if an error from Wait() is expected (e.g., signal: killed).
func isExpectedExitError(err error) bool {
	// When we call Kill(), Wait() returns exit status -1 or "signal: killed"
	return strings.Contains(err.Error(), "signal: killed") ||
		strings.Contains(err.Error(), "exit status -1")
}
