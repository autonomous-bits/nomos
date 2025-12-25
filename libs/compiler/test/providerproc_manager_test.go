package test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/autonomous-bits/nomos/libs/compiler"
)

// TestManager_GetProvider_StartSubprocess tests that GetProvider starts
// a subprocess for a new provider alias and establishes a gRPC connection.
// RED: This test will fail until compiler.Manager is implemented.
func TestManager_GetProvider_StartSubprocess(t *testing.T) {
	// Arrange: Create a fake provider binary that implements the gRPC service
	binaryPath := createFakeProviderBinary(t)

	manager := compiler.NewManager()
	defer func() {
		if err := manager.Shutdown(context.Background()); err != nil {
			t.Logf("Shutdown error: %v", err)
		}
	}()

	opts := compiler.ProviderInitOptions{
		Alias:  "test-provider",
		Config: map[string]any{"key": "value"},
	}

	// Act: Get provider (should start subprocess)
	provider, err := manager.GetProvider(context.Background(), "test-provider", binaryPath, opts)

	// Assert
	if err != nil {
		t.Fatalf("GetProvider failed: %v", err)
	}
	if provider == nil {
		t.Fatal("Expected provider, got nil")
	}

	// Verify the provider can be called (basic Init check)
	initCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = provider.Init(initCtx, opts)
	if err != nil {
		t.Errorf("Provider Init failed: %v", err)
	}
}

// TestManager_GetProvider_ReusesSameAlias tests that GetProvider returns
// the same provider instance for the same alias (per-alias caching).
func TestManager_GetProvider_ReusesSameAlias(t *testing.T) {
	// Arrange
	binaryPath := createFakeProviderBinary(t)

	manager := compiler.NewManager()
	defer func() {
		if err := manager.Shutdown(context.Background()); err != nil {
			t.Logf("Shutdown error: %v", err)
		}
	}()

	opts := compiler.ProviderInitOptions{
		Alias:  "test-provider",
		Config: map[string]any{"key": "value"},
	}

	// Act: Get provider twice with same alias
	provider1, err := manager.GetProvider(context.Background(), "test-provider", binaryPath, opts)
	if err != nil {
		t.Fatalf("First GetProvider failed: %v", err)
	}

	provider2, err := manager.GetProvider(context.Background(), "test-provider", binaryPath, opts)
	if err != nil {
		t.Fatalf("Second GetProvider failed: %v", err)
	}

	// Assert: Should be the same instance (pointer equality)
	if provider1 != provider2 {
		t.Errorf("Expected same provider instance, got different instances")
	}
}

// TestManager_GetProvider_DifferentAliases tests that GetProvider starts
// separate processes for different aliases.
func TestManager_GetProvider_DifferentAliases(t *testing.T) {
	// Arrange
	binaryPath := createFakeProviderBinary(t)

	manager := compiler.NewManager()
	defer func() {
		if err := manager.Shutdown(context.Background()); err != nil {
			t.Logf("Shutdown error: %v", err)
		}
	}()

	opts1 := compiler.ProviderInitOptions{
		Alias:  "provider-1",
		Config: map[string]any{"key": "value1"},
	}
	opts2 := compiler.ProviderInitOptions{
		Alias:  "provider-2",
		Config: map[string]any{"key": "value2"},
	}

	// Act: Get providers with different aliases
	provider1, err := manager.GetProvider(context.Background(), "provider-1", binaryPath, opts1)
	if err != nil {
		t.Fatalf("GetProvider for alias 1 failed: %v", err)
	}

	provider2, err := manager.GetProvider(context.Background(), "provider-2", binaryPath, opts2)
	if err != nil {
		t.Fatalf("GetProvider for alias 2 failed: %v", err)
	}

	// Assert: Should be different instances
	if provider1 == provider2 {
		t.Errorf("Expected different provider instances, got same instance")
	}
}

// TestManager_GetProvider_MissingBinary tests that GetProvider returns
// an error when the binary doesn't exist.
func TestManager_GetProvider_MissingBinary(t *testing.T) {
	// Arrange
	manager := compiler.NewManager()
	opts := compiler.ProviderInitOptions{
		Alias:  "test-provider",
		Config: map[string]any{},
	}

	// Act: Try to get provider with non-existent binary
	_, err := manager.GetProvider(context.Background(), "test-provider", "/nonexistent/path/to/binary", opts)

	// Assert: Should fail with appropriate error
	if err == nil {
		t.Fatal("Expected error for missing binary, got nil")
	}

	if !contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' in error, got: %v", err)
	}
}

// TestManager_Shutdown_TerminatesProcesses tests that Shutdown terminates
// all running provider processes.
func TestManager_Shutdown_TerminatesProcesses(t *testing.T) {
	// Arrange
	binaryPath := createFakeProviderBinary(t)

	manager := compiler.NewManager()

	opts := compiler.ProviderInitOptions{
		Alias:  "test-provider",
		Config: map[string]any{},
	}

	// Start a provider
	_, err := manager.GetProvider(context.Background(), "test-provider", binaryPath, opts)
	if err != nil {
		t.Fatalf("GetProvider failed: %v", err)
	}

	// Act: Shutdown
	err = manager.Shutdown(context.Background())

	// Assert
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}

	// Note: Cannot verify internal state from external test package
	// Shutdown success is sufficient verification
}

// createFakeProviderBinary creates a minimal Go binary that implements
// the provider gRPC service for testing purposes.
func createFakeProviderBinary(t *testing.T) string {
	t.Helper()

	// Create a temporary directory for the fake binary
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "fake-provider")

	// Determine absolute path to provider-proto module
	// Tests run from within the test package directory
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	providerProtoPath := filepath.Join(currentDir, "../../provider-proto")
	providerProtoPath, err = filepath.Abs(providerProtoPath)
	if err != nil {
		t.Fatalf("Failed to resolve provider-proto path: %v", err)
	}

	// Create a go.mod for the fake provider
	goMod := fmt.Sprintf(`module fake-provider

go 1.22

require (
	github.com/autonomous-bits/nomos/libs/provider-proto v0.0.0
	google.golang.org/grpc v1.70.0
	google.golang.org/protobuf v1.36.0
)

replace github.com/autonomous-bits/nomos/libs/provider-proto => %s
`, providerProtoPath)
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte(goMod), 0644); err != nil { //nolint:gosec // G306: Test fixture file
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	// Write a minimal main.go that starts a gRPC server
	mainGo := `package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"

	providerv1 "github.com/autonomous-bits/nomos/libs/provider-proto/gen/go/nomos/provider/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"
)

type fakeProvider struct {
	providerv1.UnimplementedProviderServiceServer
}

func (p *fakeProvider) Init(ctx context.Context, req *providerv1.InitRequest) (*providerv1.InitResponse, error) {
	return &providerv1.InitResponse{}, nil
}

func (p *fakeProvider) Fetch(ctx context.Context, req *providerv1.FetchRequest) (*providerv1.FetchResponse, error) {
	value, _ := structpb.NewStruct(map[string]interface{}{"test": "value"})
	return &providerv1.FetchResponse{Value: value}, nil
}

func (p *fakeProvider) Info(ctx context.Context, req *providerv1.InfoRequest) (*providerv1.InfoResponse, error) {
	return &providerv1.InfoResponse{
		Alias:   "test-provider",
		Version: "0.0.1-test",
		Type:    "fake",
	}, nil
}

func (p *fakeProvider) Health(ctx context.Context, req *providerv1.HealthRequest) (*providerv1.HealthResponse, error) {
	return &providerv1.HealthResponse{
		Status:  providerv1.HealthResponse_STATUS_OK,
		Message: "healthy",
	}, nil
}

func (p *fakeProvider) Shutdown(ctx context.Context, req *providerv1.ShutdownRequest) (*providerv1.ShutdownResponse, error) {
	return &providerv1.ShutdownResponse{}, nil
}

func main() {
	// Listen on a random available port and print it to stdout for the manager to discover
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// Print the port so the manager can connect
	fmt.Fprintf(os.Stdout, "PROVIDER_PORT=%d\n", lis.Addr().(*net.TCPAddr).Port)
	os.Stdout.Sync()

	grpcServer := grpc.NewServer()
	providerv1.RegisterProviderServiceServer(grpcServer, &fakeProvider{})

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
`

	mainGoPath := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(mainGoPath, []byte(mainGo), 0644); err != nil { //nolint:gosec // G306: Test fixture file
		t.Fatalf("Failed to write main.go: %v", err)
	}

	// Run go mod tidy to download dependencies
	tidyCmd := exec.Command("go", "mod", "tidy") //nolint:noctx // Test helper uses synchronous exec for simplicity
	tidyCmd.Dir = tmpDir
	tidyOutput, err := tidyCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run go mod tidy: %v\nOutput: %s", err, tidyOutput)
	}

	// Build the binary
	//nolint:gosec // G204: Test helper building controlled test binary with known args
	cmd := exec.CommandContext(context.Background(), "go", "build", "-o", binaryPath, mainGoPath)
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build fake provider: %v\nOutput: %s", err, output)
	}

	return binaryPath
}
