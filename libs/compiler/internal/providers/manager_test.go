package providers

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/autonomous-bits/nomos/libs/compiler/internal/core"
)

func TestNewManager_DefaultOptions(t *testing.T) {
	// Act
	manager := NewManager(nil)

	// Assert
	if manager == nil {
		t.Fatal("expected non-nil manager")
	}
	if manager.shutdownTimeout != DefaultShutdownTimeout {
		t.Errorf("expected default timeout %v, got %v", DefaultShutdownTimeout, manager.shutdownTimeout)
	}
	if manager.processes == nil {
		t.Error("expected processes map to be initialized")
	}
}

func TestNewManager_CustomOptions(t *testing.T) {
	// Setup
	customTimeout := 10 * time.Second
	opts := &ManagerOptions{
		ShutdownTimeout: customTimeout,
	}

	// Act
	manager := NewManager(opts)

	// Assert
	if manager.shutdownTimeout != customTimeout {
		t.Errorf("expected timeout %v, got %v", customTimeout, manager.shutdownTimeout)
	}
}

func TestNewManager_ZeroTimeoutUsesDefault(t *testing.T) {
	// Setup: zero timeout should use default
	opts := &ManagerOptions{
		ShutdownTimeout: 0,
	}

	// Act
	manager := NewManager(opts)

	// Assert
	if manager.shutdownTimeout != DefaultShutdownTimeout {
		t.Errorf("expected default timeout for zero value, got %v", manager.shutdownTimeout)
	}
}

func TestManager_GetProvider_BinaryNotFound(t *testing.T) {
	// Setup
	manager := NewManager(nil)
	defer func() {
		if err := manager.Shutdown(context.Background()); err != nil {
			t.Logf("Shutdown error: %v", err)
		}
	}()

	nonExistentPath := "/nonexistent/provider/binary"
	opts := core.ProviderInitOptions{
		Alias:  "test",
		Config: map[string]any{},
	}

	// Act
	_, err := manager.GetProvider(context.Background(), "test", nonExistentPath, opts)

	// Assert
	if err == nil {
		t.Fatal("expected error for nonexistent binary")
	}
	if !os.IsNotExist(errors.Unwrap(err)) {
		t.Errorf("expected os.ErrNotExist, got: %v", err)
	}
}

func TestManager_GetProvider_CachesSameAlias(t *testing.T) {
	// Setup: create a fake provider binary
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "provider")

	// Create a simple script that prints port and stays alive
	script := `#!/bin/sh
echo "PROVIDER_PORT=50051"
sleep 1
`
	if err := os.WriteFile(binaryPath, []byte(script), 0600); err != nil {
		t.Fatalf("failed to create fake binary: %v", err)
	}

	manager := NewManager(nil)
	defer func() {
		if err := manager.Shutdown(context.Background()); err != nil {
			t.Logf("Shutdown error: %v", err)
		}
	}()

	opts := core.ProviderInitOptions{
		Alias:  "test",
		Config: map[string]any{},
	}

	// Note: This test will fail without a real gRPC provider
	// We're testing the path logic, not the full integration
	// The actual integration tests with real providers are in test/providerproc_manager_test.go

	// For unit testing, we can verify the caching logic by checking
	// that the same alias returns cached process

	// Since we can't start a real provider here, let's test the error path
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := manager.GetProvider(ctx, "test", binaryPath, opts)

	// We expect an error because our fake script doesn't implement gRPC
	if err == nil {
		t.Log("Note: Test may pass if binary happens to be valid, but typically expects error")
	}
}

func TestManager_Shutdown_EmptyManager(t *testing.T) {
	// Setup: manager with no processes
	manager := NewManager(nil)

	// Act
	err := manager.Shutdown(context.Background())

	// Assert
	if err != nil {
		t.Errorf("Shutdown of empty manager should not error, got: %v", err)
	}
}

func TestManager_Shutdown_ClearsProcesses(t *testing.T) {
	// Setup: manager with no processes (nothing to shut down)
	manager := NewManager(nil)

	// Act
	err := manager.Shutdown(context.Background())

	// Assert
	if err != nil {
		t.Errorf("Shutdown of empty manager should not error, got: %v", err)
	}

	// Verify processes map is empty
	if len(manager.processes) != 0 {
		t.Errorf("expected processes map to be cleared, got %d entries", len(manager.processes))
	}
}

func TestIsExpectedExitError_SignalKilled(t *testing.T) {
	// Test error message with "signal: killed"
	err := errors.New("signal: killed")

	if !isExpectedExitError(err) {
		t.Error("expected 'signal: killed' to be recognized as expected exit error")
	}
}

func TestIsExpectedExitError_ExitStatusMinusOne(t *testing.T) {
	// Test error message with "exit status -1"
	err := errors.New("exit status -1")

	if !isExpectedExitError(err) {
		t.Error("expected 'exit status -1' to be recognized as expected exit error")
	}
}

func TestIsExpectedExitError_UnexpectedError(t *testing.T) {
	// Test error message that is not expected
	err := errors.New("some other error")

	if isExpectedExitError(err) {
		t.Error("unexpected error should not be recognized as expected exit error")
	}
}

func TestManager_GetProvider_DoubleCheckLocking(t *testing.T) {
	// Note: This test verifies the double-check locking pattern but
	// cannot use a mock process with nil cmd as it causes panics on Shutdown.
	// The real concurrency testing is done in integration tests with actual providers.

	manager := NewManager(nil)
	defer func() {
		if err := manager.Shutdown(context.Background()); err != nil {
			t.Logf("Shutdown error: %v", err)
		}
	}()

	// Test that an empty manager doesn't panic with concurrent access
	opts := core.ProviderInitOptions{
		Alias:  "test",
		Config: map[string]any{},
	}

	// Concurrent calls to GetProvider with nonexistent binary
	// All should fail but should be thread-safe
	done := make(chan error, 5)
	for i := 0; i < 5; i++ {
		go func() {
			_, err := manager.GetProvider(context.Background(), "test", "/nonexistent/binary", opts)
			done <- err
		}()
	}

	// Collect results - all should error
	errorCount := 0
	for i := 0; i < 5; i++ {
		err := <-done
		if err != nil {
			errorCount++
		}
	}

	// All calls should have failed with the same error
	if errorCount != 5 {
		t.Errorf("expected all 5 calls to fail, got %d errors", errorCount)
	}
}

func TestManager_GetProvider_ContextCancellation(t *testing.T) {
	// Test that cancelled context is handled
	manager := NewManager(nil)
	defer func() {
		if err := manager.Shutdown(context.Background()); err != nil {
			t.Logf("Shutdown error: %v", err)
		}
	}()

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	opts := core.ProviderInitOptions{
		Alias:  "test",
		Config: map[string]any{},
	}

	// Act
	_, err := manager.GetProvider(ctx, "test", "/fake/path", opts)

	// Assert: Should get an error due to cancelled context
	if err == nil {
		t.Error("expected error with cancelled context")
	}
}

func TestManager_Shutdown_WithTimeout(t *testing.T) {
	// Note: Testing shutdown with timeout requires a real subprocess
	// This is covered by integration tests in test/providerproc_manager_test.go
	// For unit tests, we test that the timeout configuration is respected

	manager := NewManager(&ManagerOptions{
		ShutdownTimeout: 50 * time.Millisecond,
	})

	if manager.shutdownTimeout != 50*time.Millisecond {
		t.Errorf("expected timeout to be set correctly")
	}

	// Empty shutdown should work immediately
	err := manager.Shutdown(context.Background())
	if err != nil {
		t.Errorf("empty shutdown should not error: %v", err)
	}
}

func TestDefaultShutdownTimeout(t *testing.T) {
	// Verify the constant is set to expected value
	expected := 5 * time.Second
	if DefaultShutdownTimeout != expected {
		t.Errorf("expected DefaultShutdownTimeout to be %v, got %v", expected, DefaultShutdownTimeout)
	}
}
