//go:build integration
// +build integration

package providerproto_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	providerv1 "github.com/autonomous-bits/nomos/libs/provider-proto/gen/go/nomos/provider/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

// testServer is a test implementation of the ProviderService for integration tests.
type testServer struct {
	providerv1.UnimplementedProviderServiceServer
	initialized       bool
	shutdownRequested bool
	healthStatus      providerv1.HealthResponse_Status
	storedConfig      *structpb.Struct
	storedAlias       string
	fetchData         map[string]*structpb.Struct // stores data by path key
}

func newTestServer() *testServer {
	return &testServer{
		initialized:  false,
		healthStatus: providerv1.HealthResponse_STATUS_OK,
		fetchData:    make(map[string]*structpb.Struct),
	}
}

func (s *testServer) Init(_ context.Context, req *providerv1.InitRequest) (*providerv1.InitResponse, error) {
	if s.initialized {
		return nil, status.Error(codes.FailedPrecondition, "provider already initialized")
	}

	if req.Alias == "" {
		return nil, status.Error(codes.InvalidArgument, "alias is required")
	}

	s.initialized = true
	s.storedAlias = req.Alias
	s.storedConfig = req.Config

	return &providerv1.InitResponse{}, nil
}

func (s *testServer) Fetch(_ context.Context, req *providerv1.FetchRequest) (*providerv1.FetchResponse, error) {
	if !s.initialized {
		return nil, status.Error(codes.FailedPrecondition, "provider not initialized")
	}

	if len(req.Path) == 0 {
		return nil, status.Error(codes.InvalidArgument, "path is required")
	}

	// Build path key
	pathKey := ""
	for i, segment := range req.Path {
		if i > 0 {
			pathKey += "/"
		}
		pathKey += segment
	}

	// Check if data exists
	data, exists := s.fetchData[pathKey]
	if !exists {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("path %q not found", pathKey))
	}

	return &providerv1.FetchResponse{Value: data}, nil
}

func (s *testServer) Info(_ context.Context, _ *providerv1.InfoRequest) (*providerv1.InfoResponse, error) {
	return &providerv1.InfoResponse{
		Alias:   s.storedAlias,
		Version: "1.0.0-test",
		Type:    "test",
	}, nil
}

func (s *testServer) Health(_ context.Context, _ *providerv1.HealthRequest) (*providerv1.HealthResponse, error) {
	return &providerv1.HealthResponse{
		Status:  s.healthStatus,
		Message: fmt.Sprintf("provider is %v", s.healthStatus.String()),
	}, nil
}

func (s *testServer) Shutdown(_ context.Context, _ *providerv1.ShutdownRequest) (*providerv1.ShutdownResponse, error) {
	s.shutdownRequested = true
	return &providerv1.ShutdownResponse{}, nil
}

// startTestGRPCServer starts a gRPC server for testing and returns client, server, and cleanup function.
func startTestGRPCServer(t *testing.T) (providerv1.ProviderServiceClient, *testServer, func()) {
	t.Helper()

	// Create listener on random port
	//nolint:noctx // Test helper uses simple net.Listen for test server setup
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}

	// Create gRPC server
	grpcServer := grpc.NewServer()
	testSrv := newTestServer()
	providerv1.RegisterProviderServiceServer(grpcServer, testSrv)

	// Start server in background
	go func() {
		//nolint:errcheck,revive // Error is expected when server stops gracefully during test cleanup
		_ = grpcServer.Serve(listener)
	}()

	// Create client connection
	addr := listener.Addr().String()
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		grpcServer.Stop()
		t.Fatalf("failed to create client: %v", err)
	}

	client := providerv1.NewProviderServiceClient(conn)

	cleanup := func() {
		if err := conn.Close(); err != nil {
			t.Logf("warning: failed to close connection: %v", err)
		}
		grpcServer.Stop()
	}

	return client, testSrv, cleanup
}

// TestGRPC_InitMethod tests the Init RPC method over real gRPC.
func TestGRPC_InitMethod(t *testing.T) {
	client, server, cleanup := startTestGRPCServer(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	config, err := structpb.NewStruct(map[string]interface{}{
		"directory": "/test/configs",
		"format":    "yaml",
	})
	if err != nil {
		t.Fatalf("failed to create config struct: %v", err)
	}

	req := &providerv1.InitRequest{
		Alias:          "test-provider",
		Config:         config,
		SourceFilePath: "/path/to/source.csl",
	}

	resp, err := client.Init(ctx, req)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if resp == nil {
		t.Fatal("expected non-nil response")
	}

	// Verify server state
	if !server.initialized {
		t.Error("expected server to be initialized")
	}

	if server.storedAlias != "test-provider" {
		t.Errorf("expected alias 'test-provider', got %q", server.storedAlias)
	}

	if server.storedConfig == nil {
		t.Fatal("expected stored config to be non-nil")
	}
}

// TestGRPC_InitErrors tests Init error handling.
func TestGRPC_InitErrors(t *testing.T) {
	client, server, cleanup := startTestGRPCServer(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("missing_alias", func(t *testing.T) {
		req := &providerv1.InitRequest{
			Alias:          "",
			Config:         &structpb.Struct{},
			SourceFilePath: "/test.csl",
		}

		_, err := client.Init(ctx, req)
		if err == nil {
			t.Fatal("expected error for missing alias")
		}

		st, ok := status.FromError(err)
		if !ok {
			t.Fatalf("expected gRPC status error, got %v", err)
		}

		if st.Code() != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument code, got %v", st.Code())
		}
	})

	t.Run("double_init", func(t *testing.T) {
		// First init should succeed
		req := &providerv1.InitRequest{
			Alias:          "test",
			Config:         &structpb.Struct{},
			SourceFilePath: "/test.csl",
		}

		_, err := client.Init(ctx, req)
		if err != nil {
			t.Fatalf("first Init failed: %v", err)
		}

		// Second init should fail
		_, err = client.Init(ctx, req)
		if err == nil {
			t.Fatal("expected error for double initialization")
		}

		st, ok := status.FromError(err)
		if !ok {
			t.Fatalf("expected gRPC status error, got %v", err)
		}

		if st.Code() != codes.FailedPrecondition {
			t.Errorf("expected FailedPrecondition code, got %v", st.Code())
		}

		// Reset server for next test
		server.initialized = false
	})
}

// TestGRPC_FetchMethod tests the Fetch RPC method.
func TestGRPC_FetchMethod(t *testing.T) {
	client, server, cleanup := startTestGRPCServer(t)
	defer cleanup()

	ctx := context.Background()

	// Initialize provider first
	initReq := &providerv1.InitRequest{
		Alias:          "test",
		Config:         &structpb.Struct{},
		SourceFilePath: "/test.csl",
	}
	_, err := client.Init(ctx, initReq)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Add test data
	testData, err := structpb.NewStruct(map[string]interface{}{
		"database": map[string]interface{}{
			"host": "localhost",
			"port": 5432.0,
		},
		"enabled": true,
	})
	if err != nil {
		t.Fatalf("failed to create test data: %v", err)
	}
	server.fetchData["config/database"] = testData

	// Fetch data
	fetchReq := &providerv1.FetchRequest{
		Path: []string{"config", "database"},
	}

	resp, err := client.Fetch(ctx, fetchReq)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if resp.Value == nil {
		t.Fatal("expected non-nil value")
	}

	// Verify data
	fields := resp.Value.AsMap()
	if fields["enabled"] != true {
		t.Errorf("expected enabled=true, got %v", fields["enabled"])
	}

	db, ok := fields["database"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected database to be a map, got %T", fields["database"])
	}

	if db["host"] != "localhost" {
		t.Errorf("expected host=localhost, got %v", db["host"])
	}
}

// TestGRPC_FetchErrors tests Fetch error handling.
func TestGRPC_FetchErrors(t *testing.T) {
	client, _, cleanup := startTestGRPCServer(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("not_initialized", func(t *testing.T) {
		req := &providerv1.FetchRequest{
			Path: []string{"config"},
		}

		_, err := client.Fetch(ctx, req)
		if err == nil {
			t.Fatal("expected error for uninitialized provider")
		}

		st, ok := status.FromError(err)
		if !ok {
			t.Fatalf("expected gRPC status error, got %v", err)
		}

		if st.Code() != codes.FailedPrecondition {
			t.Errorf("expected FailedPrecondition code, got %v", st.Code())
		}
	})

	t.Run("empty_path", func(t *testing.T) {
		// Initialize first
		initReq := &providerv1.InitRequest{
			Alias:          "test",
			Config:         &structpb.Struct{},
			SourceFilePath: "/test.csl",
		}
		_, err := client.Init(ctx, initReq)
		if err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		req := &providerv1.FetchRequest{
			Path: []string{},
		}

		_, err = client.Fetch(ctx, req)
		if err == nil {
			t.Fatal("expected error for empty path")
		}

		st, ok := status.FromError(err)
		if !ok {
			t.Fatalf("expected gRPC status error, got %v", err)
		}

		if st.Code() != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument code, got %v", st.Code())
		}
	})

	t.Run("not_found", func(t *testing.T) {
		req := &providerv1.FetchRequest{
			Path: []string{"nonexistent", "path"},
		}

		_, err := client.Fetch(ctx, req)
		if err == nil {
			t.Fatal("expected error for nonexistent path")
		}

		st, ok := status.FromError(err)
		if !ok {
			t.Fatalf("expected gRPC status error, got %v", err)
		}

		if st.Code() != codes.NotFound {
			t.Errorf("expected NotFound code, got %v", st.Code())
		}
	})
}

// TestGRPC_InfoMethod tests the Info RPC method.
func TestGRPC_InfoMethod(t *testing.T) {
	client, _, cleanup := startTestGRPCServer(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("before_init", func(t *testing.T) {
		req := &providerv1.InfoRequest{}

		resp, err := client.Info(ctx, req)
		if err != nil {
			t.Fatalf("Info failed: %v", err)
		}

		if resp.Version != "1.0.0-test" {
			t.Errorf("expected version '1.0.0-test', got %q", resp.Version)
		}

		if resp.Type != "test" {
			t.Errorf("expected type 'test', got %q", resp.Type)
		}
	})

	t.Run("after_init", func(t *testing.T) {
		initReq := &providerv1.InitRequest{
			Alias:          "my-provider",
			Config:         &structpb.Struct{},
			SourceFilePath: "/test.csl",
		}
		_, err := client.Init(ctx, initReq)
		if err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		infoReq := &providerv1.InfoRequest{}
		resp, err := client.Info(ctx, infoReq)
		if err != nil {
			t.Fatalf("Info failed: %v", err)
		}

		if resp.Alias != "my-provider" {
			t.Errorf("expected alias 'my-provider', got %q", resp.Alias)
		}
	})
}

// TestGRPC_HealthMethod tests the Health RPC method.
func TestGRPC_HealthMethod(t *testing.T) {
	client, server, cleanup := startTestGRPCServer(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("ok_status", func(t *testing.T) {
		req := &providerv1.HealthRequest{}

		resp, err := client.Health(ctx, req)
		if err != nil {
			t.Fatalf("Health failed: %v", err)
		}

		if resp.Status != providerv1.HealthResponse_STATUS_OK {
			t.Errorf("expected STATUS_OK, got %v", resp.Status)
		}

		if resp.Message == "" {
			t.Error("expected non-empty message")
		}
	})

	t.Run("degraded_status", func(t *testing.T) {
		server.healthStatus = providerv1.HealthResponse_STATUS_DEGRADED

		req := &providerv1.HealthRequest{}

		resp, err := client.Health(ctx, req)
		if err != nil {
			t.Fatalf("Health failed: %v", err)
		}

		if resp.Status != providerv1.HealthResponse_STATUS_DEGRADED {
			t.Errorf("expected STATUS_DEGRADED, got %v", resp.Status)
		}
	})
}

// TestGRPC_ShutdownMethod tests the Shutdown RPC method.
func TestGRPC_ShutdownMethod(t *testing.T) {
	client, server, cleanup := startTestGRPCServer(t)
	defer cleanup()

	ctx := context.Background()

	req := &providerv1.ShutdownRequest{}

	resp, err := client.Shutdown(ctx, req)
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	if resp == nil {
		t.Fatal("expected non-nil response")
	}

	if !server.shutdownRequested {
		t.Error("expected shutdown to be requested on server")
	}
}

// TestGRPC_DataSerialization tests round-trip serialization of various data types.
func TestGRPC_DataSerialization(t *testing.T) {
	client, server, cleanup := startTestGRPCServer(t)
	defer cleanup()

	ctx := context.Background()

	// Initialize provider
	initReq := &providerv1.InitRequest{
		Alias:          "test",
		Config:         &structpb.Struct{},
		SourceFilePath: "/test.csl",
	}
	_, err := client.Init(ctx, initReq)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	tests := []struct {
		name string
		data map[string]interface{}
	}{
		{
			name: "scalars",
			data: map[string]interface{}{
				"string": "hello",
				"number": 42.5,
				"bool":   true,
				"null":   nil,
			},
		},
		{
			name: "nested_maps",
			data: map[string]interface{}{
				"level1": map[string]interface{}{
					"level2": map[string]interface{}{
						"level3": "deep value",
					},
				},
			},
		},
		{
			name: "arrays",
			data: map[string]interface{}{
				"numbers": []interface{}{1.0, 2.0, 3.0},
				"strings": []interface{}{"a", "b", "c"},
				"mixed":   []interface{}{1.0, "two", true},
			},
		},
		{
			name: "complex_structure",
			data: map[string]interface{}{
				"users": []interface{}{
					map[string]interface{}{
						"name":  "Alice",
						"age":   30.0,
						"roles": []interface{}{"admin", "user"},
					},
					map[string]interface{}{
						"name":  "Bob",
						"age":   25.0,
						"roles": []interface{}{"user"},
					},
				},
				"metadata": map[string]interface{}{
					"version": "1.0",
					"enabled": true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create struct from data
			original, err := structpb.NewStruct(tt.data)
			if err != nil {
				t.Fatalf("failed to create struct: %v", err)
			}

			// Store in server
			pathKey := fmt.Sprintf("test/%s", tt.name)
			server.fetchData[pathKey] = original

			// Fetch via gRPC
			fetchReq := &providerv1.FetchRequest{
				Path: []string{"test", tt.name},
			}

			resp, err := client.Fetch(ctx, fetchReq)
			if err != nil {
				t.Fatalf("Fetch failed: %v", err)
			}

			// Convert back to map
			roundtripped := resp.Value.AsMap()

			// Deep compare
			if !deepEqual(roundtripped, tt.data) {
				t.Errorf("data mismatch:\nexpected: %#v\ngot: %#v", tt.data, roundtripped)
			}
		})
	}
}

// TestGRPC_LifecycleOrdering tests that lifecycle methods must be called in correct order.
func TestGRPC_LifecycleOrdering(t *testing.T) {
	client, server, cleanup := startTestGRPCServer(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("fetch_before_init_fails", func(t *testing.T) {
		req := &providerv1.FetchRequest{
			Path: []string{"config"},
		}

		_, err := client.Fetch(ctx, req)
		if err == nil {
			t.Fatal("expected error when calling Fetch before Init")
		}

		st, ok := status.FromError(err)
		if !ok {
			t.Fatalf("expected gRPC status error, got %v", err)
		}

		if st.Code() != codes.FailedPrecondition {
			t.Errorf("expected FailedPrecondition, got %v", st.Code())
		}
	})

	t.Run("correct_lifecycle", func(t *testing.T) {
		// 1. Info can be called before Init
		infoResp, err := client.Info(ctx, &providerv1.InfoRequest{})
		if err != nil {
			t.Fatalf("Info failed: %v", err)
		}
		if infoResp == nil {
			t.Fatal("expected non-nil Info response")
		}

		// 2. Health can be called before Init
		healthResp, err := client.Health(ctx, &providerv1.HealthRequest{})
		if err != nil {
			t.Fatalf("Health failed: %v", err)
		}
		if healthResp == nil {
			t.Fatal("expected non-nil Health response")
		}

		// 3. Init
		initReq := &providerv1.InitRequest{
			Alias:          "test",
			Config:         &structpb.Struct{},
			SourceFilePath: "/test.csl",
		}
		_, err = client.Init(ctx, initReq)
		if err != nil {
			t.Fatalf("Init failed: %v", err)
		}

		// 4. Fetch (now should work)
		testData, _ := structpb.NewStruct(map[string]interface{}{"key": "value"})
		server.fetchData["config"] = testData

		fetchResp, err := client.Fetch(ctx, &providerv1.FetchRequest{
			Path: []string{"config"},
		})
		if err != nil {
			t.Fatalf("Fetch failed: %v", err)
		}
		if fetchResp == nil {
			t.Fatal("expected non-nil Fetch response")
		}

		// 5. Shutdown
		shutdownResp, err := client.Shutdown(ctx, &providerv1.ShutdownRequest{})
		if err != nil {
			t.Fatalf("Shutdown failed: %v", err)
		}
		if shutdownResp == nil {
			t.Fatal("expected non-nil Shutdown response")
		}
	})
}

// deepEqual performs deep comparison of two values, handling maps and slices.
func deepEqual(a, b interface{}) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	switch aVal := a.(type) {
	case map[string]interface{}:
		bVal, ok := b.(map[string]interface{})
		if !ok {
			return false
		}
		if len(aVal) != len(bVal) {
			return false
		}
		for k, v := range aVal {
			if !deepEqual(v, bVal[k]) {
				return false
			}
		}
		return true

	case []interface{}:
		bVal, ok := b.([]interface{})
		if !ok {
			return false
		}
		if len(aVal) != len(bVal) {
			return false
		}
		for i := range aVal {
			if !deepEqual(aVal[i], bVal[i]) {
				return false
			}
		}
		return true

	default:
		return a == b
	}
}

// TestGRPC_ContextCancellation tests that context cancellation is properly handled.
func TestGRPC_ContextCancellation(t *testing.T) {
	client, _, cleanup := startTestGRPCServer(t)
	defer cleanup()

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	req := &providerv1.InfoRequest{}

	_, err := client.Info(ctx, req)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}

	if !errors.Is(err, context.Canceled) {
		// gRPC wraps context errors, check status code
		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.Canceled {
			t.Errorf("expected Canceled error, got %v", err)
		}
	}
}
