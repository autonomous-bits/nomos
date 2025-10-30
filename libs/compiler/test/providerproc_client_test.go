package test

import "github.com/autonomous-bits/nomos/libs/compiler"

import (
	"context"
	"testing"

	"github.com/autonomous-bits/nomos/libs/compiler/test/fakes"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TestClient_Fetch tests that the compiler.Client correctly delegates Fetch calls to the gRPC service.
func TestClient_Fetch(t *testing.T) {
	// Arrange: Start a fake provider server
	fake := fakes.NewFakeProviderServer("test-provider", "0.0.1-test", "fake")
	fake.SetFetchResponse([]string{"test", "path"}, map[string]any{"test": "value"})

	server, addr, err := fakes.StartFakeProviderServer(fake)
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	// Connect to the server
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := compiler.NewClient(conn, "test-alias")

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

// TestClient_Fetch_NotFound tests that compiler.Client returns appropriate error for NotFound.
func TestClient_Fetch_NotFound(t *testing.T) {
	// Arrange: Start a fake provider server (no responses configured)
	fake := fakes.NewFakeProviderServer("test-provider", "0.0.1-test", "fake")

	server, addr, err := fakes.StartFakeProviderServer(fake)
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := compiler.NewClient(conn, "test-alias")

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

// TestClient_Init tests that compiler.Client delegates Init correctly.
func TestClient_Init(t *testing.T) {
	// Arrange
	fake := fakes.NewFakeProviderServer("test-provider", "0.0.1-test", "fake")

	server, addr, err := fakes.StartFakeProviderServer(fake)
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := compiler.NewClient(conn, "test-alias")

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

	// Verify Init was called on the fake server
	if !fake.InitCalled() {
		t.Error("Expected Init to be called on server")
	}
}

// TestClient_Info tests that compiler.Client implements ProviderWithInfo.
func TestClient_Info(t *testing.T) {
	// Arrange
	fake := fakes.NewFakeProviderServer("test-provider", "0.0.1-test", "fake")

	server, addr, err := fakes.StartFakeProviderServer(fake)
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := compiler.NewClient(conn, "test-alias")

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
