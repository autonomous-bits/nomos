package fakes_test

import (
	"context"
	"testing"

	"github.com/autonomous-bits/nomos/libs/compiler/test/fakes"
	providerv1 "github.com/autonomous-bits/nomos/libs/provider-proto/gen/go/nomos/provider/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

// TestFakeProviderServer_Init_Success tests the Init RPC method with valid inputs.
func TestFakeProviderServer_Init_Success(t *testing.T) {
	// Arrange
	fake := fakes.NewFakeProviderServer("test-provider", "1.0.0", "fake")
	server, addr, err := fakes.StartFakeProviderServer(fake)
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer server.Stop()

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	client := providerv1.NewProviderServiceClient(conn)

	// Act
	config, _ := structpb.NewStruct(map[string]any{
		"directory": "/test/path",
		"pattern":   "*.csl",
	})
	req := &providerv1.InitRequest{
		Alias:          "test-provider",
		Config:         config,
		SourceFilePath: "/source/file.csl",
	}

	resp, err := client.Init(context.Background(), req)

	// Assert
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if !fake.InitCalled() {
		t.Error("expected Init to be called on server")
	}
	if fake.InitCount() != 1 {
		t.Errorf("expected Init called once, got %d", fake.InitCount())
	}

	// Verify config was stored
	initConfig := fake.InitConfig()
	if initConfig["directory"] != "/test/path" {
		t.Errorf("expected directory='/test/path', got %v", initConfig["directory"])
	}
	if initConfig["pattern"] != "*.csl" {
		t.Errorf("expected pattern='*.csl', got %v", initConfig["pattern"])
	}
}

// TestFakeProviderServer_Init_Error tests Init RPC with configured error.
func TestFakeProviderServer_Init_Error(t *testing.T) {
	// Arrange
	fake := fakes.NewFakeProviderServer("test-provider", "1.0.0", "fake")
	fake.SetInitError(status.Error(codes.InvalidArgument, "invalid configuration"))

	server, addr, err := fakes.StartFakeProviderServer(fake)
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer server.Stop()

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	client := providerv1.NewProviderServiceClient(conn)

	// Act
	req := &providerv1.InitRequest{
		Alias:  "test-provider",
		Config: &structpb.Struct{},
	}

	_, err = client.Init(context.Background(), req)

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %T", err)
	}
	if st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", st.Code())
	}
	if st.Message() != "invalid configuration" {
		t.Errorf("expected 'invalid configuration', got %q", st.Message())
	}
}

// TestFakeProviderServer_Fetch_Success tests Fetch RPC with successful response.
func TestFakeProviderServer_Fetch_Success(t *testing.T) {
	// Arrange
	fake := fakes.NewFakeProviderServer("test-provider", "1.0.0", "fake")

	// Configure a response
	fake.SetFetchResponse([]string{"config", "database"}, map[string]any{
		"host":     "localhost",
		"port":     5432,
		"database": "testdb",
		"nested": map[string]any{
			"level1": map[string]any{
				"level2": "deep-value",
			},
		},
	})

	server, addr, err := fakes.StartFakeProviderServer(fake)
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer server.Stop()

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	client := providerv1.NewProviderServiceClient(conn)

	// Act
	req := &providerv1.FetchRequest{
		Path: []string{"config", "database"},
	}

	resp, err := client.Fetch(context.Background(), req)

	// Assert
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	if resp == nil || resp.Value == nil {
		t.Fatal("expected non-nil response with value")
	}

	// Verify Struct <-> Go map conversion
	data := resp.Value.AsMap()
	if data["host"] != "localhost" {
		t.Errorf("expected host='localhost', got %v", data["host"])
	}
	if data["port"] != float64(5432) { // JSON numbers become float64
		t.Errorf("expected port=5432, got %v", data["port"])
	}
	if data["database"] != "testdb" {
		t.Errorf("expected database='testdb', got %v", data["database"])
	}

	// Verify nested structure
	nested, ok := data["nested"].(map[string]any)
	if !ok {
		t.Fatalf("expected nested to be map[string]any, got %T", data["nested"])
	}
	level1, ok := nested["level1"].(map[string]any)
	if !ok {
		t.Fatalf("expected level1 to be map[string]any, got %T", nested["level1"])
	}
	if level1["level2"] != "deep-value" {
		t.Errorf("expected level2='deep-value', got %v", level1["level2"])
	}

	// Verify call tracking
	if fake.FetchCount() != 1 {
		t.Errorf("expected FetchCount=1, got %d", fake.FetchCount())
	}

	calls := fake.FetchCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 fetch call, got %d", len(calls))
	}
	if len(calls[0]) != 2 || calls[0][0] != "config" || calls[0][1] != "database" {
		t.Errorf("expected path ['config', 'database'], got %v", calls[0])
	}
}

// TestFakeProviderServer_Fetch_NotFound tests Fetch RPC with NotFound error.
func TestFakeProviderServer_Fetch_NotFound(t *testing.T) {
	// Arrange
	fake := fakes.NewFakeProviderServer("test-provider", "1.0.0", "fake")

	server, addr, err := fakes.StartFakeProviderServer(fake)
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer server.Stop()

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	client := providerv1.NewProviderServiceClient(conn)

	// Act - Fetch a path with no configured response
	req := &providerv1.FetchRequest{
		Path: []string{"nonexistent", "path"},
	}

	_, err = client.Fetch(context.Background(), req)

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %T", err)
	}
	if st.Code() != codes.NotFound {
		t.Errorf("expected NotFound, got %v", st.Code())
	}
}

// TestFakeProviderServer_Fetch_CustomError tests Fetch RPC with custom configured error.
func TestFakeProviderServer_Fetch_CustomError(t *testing.T) {
	// Arrange
	fake := fakes.NewFakeProviderServer("test-provider", "1.0.0", "fake")
	fake.SetFetchError([]string{"error", "path"},
		status.Error(codes.PermissionDenied, "access denied"))

	server, addr, err := fakes.StartFakeProviderServer(fake)
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer server.Stop()

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	client := providerv1.NewProviderServiceClient(conn)

	// Act
	req := &providerv1.FetchRequest{
		Path: []string{"error", "path"},
	}

	_, err = client.Fetch(context.Background(), req)

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %T", err)
	}
	if st.Code() != codes.PermissionDenied {
		t.Errorf("expected PermissionDenied, got %v", st.Code())
	}
	if st.Message() != "access denied" {
		t.Errorf("expected 'access denied', got %q", st.Message())
	}
}

// TestFakeProviderServer_Info_Success tests Info RPC method.
func TestFakeProviderServer_Info_Success(t *testing.T) {
	// Arrange
	fake := fakes.NewFakeProviderServer("my-provider", "2.3.4", "custom")

	server, addr, err := fakes.StartFakeProviderServer(fake)
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer server.Stop()

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	client := providerv1.NewProviderServiceClient(conn)

	// Act
	resp, err := client.Info(context.Background(), &providerv1.InfoRequest{})

	// Assert
	if err != nil {
		t.Fatalf("Info failed: %v", err)
	}
	if resp.Alias != "my-provider" {
		t.Errorf("expected alias='my-provider', got %q", resp.Alias)
	}
	if resp.Version != "2.3.4" {
		t.Errorf("expected version='2.3.4', got %q", resp.Version)
	}
	if resp.Type != "custom" {
		t.Errorf("expected type='custom', got %q", resp.Type)
	}

	if fake.InfoCount() != 1 {
		t.Errorf("expected InfoCount=1, got %d", fake.InfoCount())
	}
}

// TestFakeProviderServer_Health_OK tests Health RPC with OK status.
func TestFakeProviderServer_Health_OK(t *testing.T) {
	// Arrange
	fake := fakes.NewFakeProviderServer("test-provider", "1.0.0", "fake")

	server, addr, err := fakes.StartFakeProviderServer(fake)
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer server.Stop()

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	client := providerv1.NewProviderServiceClient(conn)

	// Act
	resp, err := client.Health(context.Background(), &providerv1.HealthRequest{})

	// Assert
	if err != nil {
		t.Fatalf("Health failed: %v", err)
	}
	if resp.Status != providerv1.HealthResponse_STATUS_OK {
		t.Errorf("expected STATUS_OK, got %v", resp.Status)
	}
	if resp.Message != "healthy" {
		t.Errorf("expected message='healthy', got %q", resp.Message)
	}

	if fake.HealthCount() != 1 {
		t.Errorf("expected HealthCount=1, got %d", fake.HealthCount())
	}
}

// TestFakeProviderServer_Health_Degraded tests Health RPC with degraded status.
func TestFakeProviderServer_Health_Degraded(t *testing.T) {
	// Arrange
	fake := fakes.NewFakeProviderServer("test-provider", "1.0.0", "fake")
	fake.SetHealthStatus(providerv1.HealthResponse_STATUS_DEGRADED, "partial outage")

	server, addr, err := fakes.StartFakeProviderServer(fake)
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer server.Stop()

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	client := providerv1.NewProviderServiceClient(conn)

	// Act
	resp, err := client.Health(context.Background(), &providerv1.HealthRequest{})

	// Assert
	if err != nil {
		t.Fatalf("Health failed: %v", err)
	}
	if resp.Status != providerv1.HealthResponse_STATUS_DEGRADED {
		t.Errorf("expected STATUS_DEGRADED, got %v", resp.Status)
	}
	if resp.Message != "partial outage" {
		t.Errorf("expected message='partial outage', got %q", resp.Message)
	}
}

// TestFakeProviderServer_Shutdown_Success tests Shutdown RPC method.
func TestFakeProviderServer_Shutdown_Success(t *testing.T) {
	// Arrange
	fake := fakes.NewFakeProviderServer("test-provider", "1.0.0", "fake")

	server, addr, err := fakes.StartFakeProviderServer(fake)
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer server.Stop()

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	client := providerv1.NewProviderServiceClient(conn)

	// Act
	resp, err := client.Shutdown(context.Background(), &providerv1.ShutdownRequest{})

	// Assert
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}

	if !fake.ShutdownCalled() {
		t.Error("expected Shutdown to be called on server")
	}
	if fake.ShutdownCount() != 1 {
		t.Errorf("expected ShutdownCount=1, got %d", fake.ShutdownCount())
	}
}

// TestFakeProviderServer_StructMapping_ComplexTypes tests Struct <-> Go map conversion
// with various data types including nested structures, arrays, null values, etc.
func TestFakeProviderServer_StructMapping_ComplexTypes(t *testing.T) {
	// Arrange
	fake := fakes.NewFakeProviderServer("test-provider", "1.0.0", "fake")

	complexData := map[string]any{
		"string":  "text",
		"number":  42.5,
		"integer": int64(100),
		"boolean": true,
		"null":    nil,
		"array":   []any{"a", "b", "c"},
		"object": map[string]any{
			"nested": "value",
		},
		"mixed_array": []any{
			"string",
			123.0,
			true,
			map[string]any{"key": "val"},
		},
	}

	fake.SetFetchResponse([]string{"complex"}, complexData)

	server, addr, err := fakes.StartFakeProviderServer(fake)
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer server.Stop()

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	client := providerv1.NewProviderServiceClient(conn)

	// Act
	req := &providerv1.FetchRequest{Path: []string{"complex"}}
	resp, err := client.Fetch(context.Background(), req)

	// Assert
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	data := resp.Value.AsMap()

	// Verify string
	if data["string"] != "text" {
		t.Errorf("expected string='text', got %v", data["string"])
	}

	// Verify number (becomes float64)
	if data["number"] != 42.5 {
		t.Errorf("expected number=42.5, got %v", data["number"])
	}

	// Verify boolean
	if data["boolean"] != true {
		t.Errorf("expected boolean=true, got %v", data["boolean"])
	}

	// Verify null
	if data["null"] != nil {
		t.Errorf("expected null=nil, got %v", data["null"])
	}

	// Verify array
	arr, ok := data["array"].([]any)
	if !ok {
		t.Fatalf("expected array to be []any, got %T", data["array"])
	}
	if len(arr) != 3 || arr[0] != "a" || arr[1] != "b" || arr[2] != "c" {
		t.Errorf("expected array=['a','b','c'], got %v", arr)
	}

	// Verify nested object
	obj, ok := data["object"].(map[string]any)
	if !ok {
		t.Fatalf("expected object to be map[string]any, got %T", data["object"])
	}
	if obj["nested"] != "value" {
		t.Errorf("expected object.nested='value', got %v", obj["nested"])
	}

	// Verify mixed array
	mixedArr, ok := data["mixed_array"].([]any)
	if !ok {
		t.Fatalf("expected mixed_array to be []any, got %T", data["mixed_array"])
	}
	if len(mixedArr) != 4 {
		t.Fatalf("expected mixed_array length 4, got %d", len(mixedArr))
	}
}

// TestFakeProviderServer_Reset tests that Reset clears all state and tracking.
func TestFakeProviderServer_Reset(t *testing.T) {
	// Arrange
	fake := fakes.NewFakeProviderServer("test-provider", "1.0.0", "fake")
	fake.SetFetchResponse([]string{"test"}, map[string]any{"key": "value"})

	server, addr, err := fakes.StartFakeProviderServer(fake)
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer server.Stop()

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	client := providerv1.NewProviderServiceClient(conn)

	// Perform some operations
	_, _ = client.Init(context.Background(), &providerv1.InitRequest{Alias: "test"})
	_, _ = client.Fetch(context.Background(), &providerv1.FetchRequest{Path: []string{"test"}})
	_, _ = client.Info(context.Background(), &providerv1.InfoRequest{})
	_, _ = client.Health(context.Background(), &providerv1.HealthRequest{})
	_, _ = client.Shutdown(context.Background(), &providerv1.ShutdownRequest{})

	// Verify counts
	if fake.InitCount() != 1 || fake.FetchCount() != 1 || fake.InfoCount() != 1 ||
		fake.HealthCount() != 1 || fake.ShutdownCount() != 1 {
		t.Fatal("expected all methods to be called once before reset")
	}

	// Act
	fake.Reset()

	// Assert - all counts should be zero
	if fake.InitCount() != 0 {
		t.Errorf("expected InitCount=0 after reset, got %d", fake.InitCount())
	}
	if fake.FetchCount() != 0 {
		t.Errorf("expected FetchCount=0 after reset, got %d", fake.FetchCount())
	}
	if fake.InfoCount() != 0 {
		t.Errorf("expected InfoCount=0 after reset, got %d", fake.InfoCount())
	}
	if fake.HealthCount() != 0 {
		t.Errorf("expected HealthCount=0 after reset, got %d", fake.HealthCount())
	}
	if fake.ShutdownCount() != 0 {
		t.Errorf("expected ShutdownCount=0 after reset, got %d", fake.ShutdownCount())
	}
	if fake.InitCalled() {
		t.Error("expected InitCalled=false after reset")
	}
	if fake.ShutdownCalled() {
		t.Error("expected ShutdownCalled=false after reset")
	}

	// Verify configured responses are cleared
	_, err = client.Fetch(context.Background(), &providerv1.FetchRequest{Path: []string{"test"}})
	if err == nil {
		t.Error("expected NotFound after reset, got nil error")
	}
}
