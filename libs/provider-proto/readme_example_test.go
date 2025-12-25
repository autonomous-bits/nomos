package providerproto_test

import (
	"context"
	"testing"

	providerv1 "github.com/autonomous-bits/nomos/libs/provider-proto/gen/go/nomos/provider/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

// This file verifies that the README code example compiles correctly

type myProvider struct {
	providerv1.UnimplementedProviderServiceServer
	// your state here
}

func (p *myProvider) Init(ctx context.Context, req *providerv1.InitRequest) (*providerv1.InitResponse, error) {
	// Initialize provider with config from req.Config
	return &providerv1.InitResponse{}, nil
}

func (p *myProvider) Fetch(ctx context.Context, req *providerv1.FetchRequest) (*providerv1.FetchResponse, error) {
	// Fetch data at req.Path
	value, err := structpb.NewStruct(map[string]interface{}{
		"key": "value",
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &providerv1.FetchResponse{Value: value}, nil
}

func (p *myProvider) Info(ctx context.Context, req *providerv1.InfoRequest) (*providerv1.InfoResponse, error) {
	return &providerv1.InfoResponse{
		Alias:   "my-provider",
		Version: "1.0.0",
		Type:    "custom",
	}, nil
}

func (p *myProvider) Health(ctx context.Context, req *providerv1.HealthRequest) (*providerv1.HealthResponse, error) {
	return &providerv1.HealthResponse{
		Status:  providerv1.HealthResponse_STATUS_OK,
		Message: "provider is healthy",
	}, nil
}

func (p *myProvider) Shutdown(ctx context.Context, req *providerv1.ShutdownRequest) (*providerv1.ShutdownResponse, error) {
	// Cleanup resources
	return &providerv1.ShutdownResponse{}, nil
}

// verifyREADMEExampleCompiles shows how to start the server (from README)
func verifyREADMEExampleCompiles() {
	server := grpc.NewServer()
	providerv1.RegisterProviderServiceServer(server, &myProvider{})
	// ... start server
	_ = server
}

// TestREADMEExampleCompiles ensures the README code example compiles
func TestREADMEExampleCompiles(t *testing.T) {
	// This test verifies that the README example code compiles
	// by calling the verification function
	verifyREADMEExampleCompiles()
}
