// Package providers contains internal provider implementation details.
package providers

import (
	"context"
	"fmt"

	"github.com/autonomous-bits/nomos/libs/compiler/internal/core"
	providerv1 "github.com/autonomous-bits/nomos/libs/provider-proto/gen/go/nomos/provider/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

// Client implements the ProviderClient interface by delegating to a gRPC provider service.
// It wraps a gRPC client connection and translates between the local Provider interface
// and the remote gRPC calls.
type Client struct {
	conn   *grpc.ClientConn
	client providerv1.ProviderServiceClient
	alias  string
}

// NewClient creates a new Client that wraps a gRPC connection to a provider service.
func NewClient(conn *grpc.ClientConn, alias string) *Client {
	return &Client{
		conn:   conn,
		client: providerv1.NewProviderServiceClient(conn),
		alias:  alias,
	}
}

// Init initializes the provider via the gRPC Init RPC.
func (c *Client) Init(ctx context.Context, opts core.ProviderInitOptions) error {
	// Convert config map to protobuf Struct
	configStruct, err := structpb.NewStruct(opts.Config)
	if err != nil {
		return fmt.Errorf("failed to convert config to protobuf struct: %w", err)
	}

	req := &providerv1.InitRequest{
		Alias:          opts.Alias,
		Config:         configStruct,
		SourceFilePath: opts.SourceFilePath,
	}

	_, err = c.client.Init(ctx, req)
	if err != nil {
		return fmt.Errorf("provider init failed: %w", err)
	}

	return nil
}

// Fetch retrieves data from the provider via the gRPC Fetch RPC.
func (c *Client) Fetch(ctx context.Context, path []string) (any, error) {
	req := &providerv1.FetchRequest{
		Path: path,
	}

	resp, err := c.client.Fetch(ctx, req)
	if err != nil {
		// Check for NOT_FOUND status
		if st, ok := status.FromError(err); ok {
			if st.Code() == codes.NotFound {
				return nil, fmt.Errorf("path not found: %s", st.Message())
			}
		}
		return nil, fmt.Errorf("fetch failed: %w", err)
	}

	// Convert protobuf Struct to map[string]any
	if resp.Value == nil {
		return nil, nil
	}

	return resp.Value.AsMap(), nil
}

// Shutdown sends a graceful shutdown request to the provider.
func (c *Client) Shutdown(ctx context.Context) error {
	_, err := c.client.Shutdown(ctx, &providerv1.ShutdownRequest{})
	if err != nil {
		return fmt.Errorf("shutdown RPC failed: %w", err)
	}
	return nil
}

// Close closes the gRPC connection.
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
