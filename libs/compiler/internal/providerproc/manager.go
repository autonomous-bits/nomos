// Package providerproc manages external provider subprocesses and gRPC communication.
//
// This package provides a Manager that can start provider executables,
// establish gRPC connections, and expose them via the compiler.Provider interface.
package providerproc

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/autonomous-bits/nomos/libs/compiler"
	providerv1 "github.com/autonomous-bits/nomos/libs/provider-proto/gen/go/nomos/provider/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// providerProcess represents a running provider subprocess.
type providerProcess struct {
	cmd    *exec.Cmd
	client *Client
	alias  string
}

// Manager manages the lifecycle of external provider subprocesses.
// It starts providers on-demand, caches them per alias, and handles
// graceful shutdown.
type Manager struct {
	mu        sync.RWMutex
	processes map[string]*providerProcess // keyed by alias
}

// NewManager creates a new Manager instance.
func NewManager() *Manager {
	return &Manager{
		processes: make(map[string]*providerProcess),
	}
}

// GetProvider returns a Provider instance for the given alias.
// If the provider subprocess is not already running, it starts it
// and establishes a gRPC connection.
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
func (m *Manager) GetProvider(ctx context.Context, alias string, binaryPath string, opts compiler.ProviderInitOptions) (compiler.Provider, error) {
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
				cmd.Process.Kill()
				return nil, fmt.Errorf("invalid port format: %s", portStr)
			}
			break
		}
	}

	if port == 0 {
		cmd.Process.Kill()
		return nil, fmt.Errorf("provider did not report port")
	}

	// Connect to the provider via gRPC
	target := fmt.Sprintf("127.0.0.1:%d", port)
	conn, err := grpc.NewClient(
		target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		cmd.Process.Kill()
		return nil, fmt.Errorf("failed to connect to provider: %w", err)
	}

	// Verify connection by calling Health
	healthClient := providerv1.NewProviderServiceClient(conn)
	_, err = healthClient.Health(ctx, &providerv1.HealthRequest{})
	if err != nil {
		conn.Close()
		cmd.Process.Kill()
		return nil, fmt.Errorf("provider health check failed: %w", err)
	}

	// Create client
	client := NewClient(conn, alias)

	// Store process
	proc := &providerProcess{
		cmd:    cmd,
		client: client,
		alias:  alias,
	}
	m.processes[alias] = proc

	return client, nil
}

// Shutdown gracefully shuts down all running provider processes.
// It calls the Shutdown RPC on each provider and waits for processes to exit.
// If a process doesn't exit within the context deadline, it is forcefully terminated.
func (m *Manager) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error

	for alias, proc := range m.processes {
		// Call Shutdown RPC (best effort)
		_, err := proc.client.client.Shutdown(ctx, &providerv1.ShutdownRequest{})
		if err != nil {
			// Log but don't fail on RPC error
			errs = append(errs, fmt.Errorf("shutdown RPC failed for %s (continuing): %w", alias, err))
		}

		// Close gRPC connection
		if err := proc.client.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close connection for %s: %w", alias, err))
		}

		// Kill the process (since we closed the connection, it should exit)
		// Use Kill instead of waiting to avoid hangs
		if proc.cmd.Process != nil {
			if err := proc.cmd.Process.Kill(); err != nil {
				errs = append(errs, fmt.Errorf("failed to kill process %s: %w", alias, err))
			}
		}
	}

	// Clear processes map
	m.processes = make(map[string]*providerProcess)

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}

	return nil
}
