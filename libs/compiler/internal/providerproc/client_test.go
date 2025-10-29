package providerproc

import (
	"context"
	"net"
	"testing"

	"github.com/autonomous-bits/nomos/libs/compiler"
	providerv1 "github.com/autonomous-bits/nomos/libs/provider-proto/gen/go/nomos/provider/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

// TestClient_Fetch tests that the Client correctly delegates Fetch calls to the gRPC service.
// RED: This will initially fail until Client Fetch implementation is verified.
func TestClient_Fetch(t *testing.T) {
	// Arrange: Start a fake provider server
	server, addr := startFakeProviderServer(t)
	defer server.Stop()

	// Connect to the server
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := NewClient(conn, "test-alias")

	// Act: Call Fetch
	ctx := context.Background()
	result, err := client.Fetch(ctx, []string{"test", "path"})

	// Assert
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("Expected map[string]any, got %T", result)
	}

	if val, ok := resultMap["test"]; !ok || val != "value" {
		t.Errorf("Expected test=value, got %v", resultMap)
	}
}

// TestClient_Fetch_NotFound tests that Client returns appropriate error for NotFound.
// RED: This will initially fail until error handling is implemented.
func TestClient_Fetch_NotFound(t *testing.T) {
	// Arrange: Start a fake provider server that returns NotFound
	server, addr := startFakeProviderServer(t)
	defer server.Stop()

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := NewClient(conn, "test-alias")

	// Act: Call Fetch with path that doesn't exist
	ctx := context.Background()
	_, err = client.Fetch(ctx, []string{"nonexistent", "path"})

	// Assert
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Verify error message contains "not found"
	if !contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' in error, got: %v", err)
	}
}

// TestClient_Init tests that Client delegates Init correctly.
func TestClient_Init(t *testing.T) {
	// Arrange
	server, addr := startFakeProviderServer(t)
	defer server.Stop()

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := NewClient(conn, "test-alias")

	// Act
	ctx := context.Background()
	opts := compiler.ProviderInitOptions{
		Alias:  "test-alias",
		Config: map[string]any{"key": "value"},
	}
	err = client.Init(ctx, opts)

	// Assert
	if err != nil {
		t.Errorf("Init failed: %v", err)
	}
}

// TestClient_Info tests that Client implements ProviderWithInfo.
func TestClient_Info(t *testing.T) {
	// Arrange
	server, addr := startFakeProviderServer(t)
	defer server.Stop()

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := NewClient(conn, "test-alias")

	// Act
	alias, version := client.Info()

	// Assert
	if alias != "test-provider" {
		t.Errorf("Expected alias 'test-provider', got '%s'", alias)
	}
	if version != "0.0.1-test" {
		t.Errorf("Expected version '0.0.1-test', got '%s'", version)
	}
}

// startFakeProviderServer starts a fake gRPC server for testing.
// Returns the server and its address.
func startFakeProviderServer(t *testing.T) (*grpc.Server, string) {
	t.Helper()

	lis, err := newLocalListener(t)
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}

	server := grpc.NewServer()
	providerv1.RegisterProviderServiceServer(server, &fakeProviderService{})

	go func() {
		if err := server.Serve(lis); err != nil {
			t.Logf("Server error: %v", err)
		}
	}()

	return server, lis.Addr().String()
}

// fakeProviderService is a test implementation of the provider service.
type fakeProviderService struct {
	providerv1.UnimplementedProviderServiceServer
}

func (s *fakeProviderService) Init(ctx context.Context, req *providerv1.InitRequest) (*providerv1.InitResponse, error) {
	return &providerv1.InitResponse{}, nil
}

func (s *fakeProviderService) Fetch(ctx context.Context, req *providerv1.FetchRequest) (*providerv1.FetchResponse, error) {
	// Return NotFound for nonexistent paths
	if len(req.Path) > 0 && req.Path[0] == "nonexistent" {
		return nil, status.Error(codes.NotFound, "path not found")
	}

	// Return test data
	value, _ := structpb.NewStruct(map[string]interface{}{"test": "value"})
	return &providerv1.FetchResponse{Value: value}, nil
}

func (s *fakeProviderService) Info(ctx context.Context, req *providerv1.InfoRequest) (*providerv1.InfoResponse, error) {
	return &providerv1.InfoResponse{
		Alias:   "test-provider",
		Version: "0.0.1-test",
		Type:    "fake",
	}, nil
}

func (s *fakeProviderService) Health(ctx context.Context, req *providerv1.HealthRequest) (*providerv1.HealthResponse, error) {
	return &providerv1.HealthResponse{
		Status:  providerv1.HealthResponse_STATUS_OK,
		Message: "healthy",
	}, nil
}

func (s *fakeProviderService) Shutdown(ctx context.Context, req *providerv1.ShutdownRequest) (*providerv1.ShutdownResponse, error) {
	return &providerv1.ShutdownResponse{}, nil
}

// Helper functions

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || indexString(s, substr) >= 0)
}

func indexString(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func newLocalListener(t *testing.T) (net.Listener, error) {
	t.Helper()
	return net.Listen("tcp", "127.0.0.1:0")
}
