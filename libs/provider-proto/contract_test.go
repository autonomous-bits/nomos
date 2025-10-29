package providerproto_test

import (
	"context"
	"testing"

	providerv1 "github.com/autonomous-bits/nomos/libs/provider-proto/gen/go/nomos/provider/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// TestInitRequest_MessageStructure verifies InitRequest has all required fields.
func TestInitRequest_MessageStructure(t *testing.T) {
	req := &providerv1.InitRequest{
		Alias:          "test-provider",
		Config:         &structpb.Struct{},
		SourceFilePath: "/path/to/source.csl",
	}

	if req.Alias != "test-provider" {
		t.Errorf("expected Alias to be 'test-provider', got %q", req.Alias)
	}

	if req.Config == nil {
		t.Error("expected Config to be non-nil")
	}

	if req.SourceFilePath != "/path/to/source.csl" {
		t.Errorf("expected SourceFilePath to be '/path/to/source.csl', got %q", req.SourceFilePath)
	}
}

// TestHealthResponse_StatusEnum verifies the health status enum values.
func TestHealthResponse_StatusEnum(t *testing.T) {
	tests := []struct {
		name     string
		status   providerv1.HealthResponse_Status
		expected providerv1.HealthResponse_Status
	}{
		{"unspecified status", providerv1.HealthResponse_STATUS_UNSPECIFIED, providerv1.HealthResponse_STATUS_UNSPECIFIED},
		{"ok status", providerv1.HealthResponse_STATUS_OK, providerv1.HealthResponse_STATUS_OK},
		{"degraded status", providerv1.HealthResponse_STATUS_DEGRADED, providerv1.HealthResponse_STATUS_DEGRADED},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &providerv1.HealthResponse{
				Status:  tt.status,
				Message: "test message",
			}

			if resp.Status != tt.expected {
				t.Errorf("expected status %v, got %v", tt.expected, resp.Status)
			}

			if resp.Message != "test message" {
				t.Errorf("expected message 'test message', got %q", resp.Message)
			}
		})
	}
}

// TestMockProvider_ImplementsInterface verifies a mock implementation can satisfy the interface.
type mockProvider struct {
	providerv1.UnimplementedProviderServiceServer
}

func (m *mockProvider) Init(ctx context.Context, req *providerv1.InitRequest) (*providerv1.InitResponse, error) {
	return &providerv1.InitResponse{}, nil
}

func (m *mockProvider) Fetch(ctx context.Context, req *providerv1.FetchRequest) (*providerv1.FetchResponse, error) {
	value, _ := structpb.NewStruct(map[string]interface{}{"test": "data"})
	return &providerv1.FetchResponse{Value: value}, nil
}

func (m *mockProvider) Info(ctx context.Context, req *providerv1.InfoRequest) (*providerv1.InfoResponse, error) {
	return &providerv1.InfoResponse{
		Alias:   "mock",
		Version: "0.1.0",
		Type:    "mock",
	}, nil
}

func (m *mockProvider) Health(ctx context.Context, req *providerv1.HealthRequest) (*providerv1.HealthResponse, error) {
	return &providerv1.HealthResponse{
		Status:  providerv1.HealthResponse_STATUS_OK,
		Message: "healthy",
	}, nil
}

func (m *mockProvider) Shutdown(ctx context.Context, req *providerv1.ShutdownRequest) (*providerv1.ShutdownResponse, error) {
	return &providerv1.ShutdownResponse{}, nil
}

func TestMockProvider_ImplementsInterface(t *testing.T) {
	var _ providerv1.ProviderServiceServer = (*mockProvider)(nil)
}

func TestMockProvider_InitCall(t *testing.T) {
	mock := &mockProvider{}
	ctx := context.Background()

	req := &providerv1.InitRequest{
		Alias:          "test",
		Config:         &structpb.Struct{},
		SourceFilePath: "/test.csl",
	}

	resp, err := mock.Init(ctx, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestMockProvider_FetchCall(t *testing.T) {
	mock := &mockProvider{}
	ctx := context.Background()

	req := &providerv1.FetchRequest{
		Path: []string{"config", "database"},
	}

	resp, err := mock.Fetch(ctx, req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp == nil {
		t.Fatal("expected non-nil response")
	}

	if resp.Value == nil {
		t.Fatal("expected non-nil value")
	}
}

func TestMockProvider_HealthCall(t *testing.T) {
	mock := &mockProvider{}
	ctx := context.Background()

	resp, err := mock.Health(ctx, &providerv1.HealthRequest{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.Status != providerv1.HealthResponse_STATUS_OK {
		t.Errorf("expected status OK, got %v", resp.Status)
	}

	if resp.Message == "" {
		t.Error("expected non-empty message")
	}
}
