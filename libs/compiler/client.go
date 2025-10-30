package compiler

import (
	"context"
	"fmt"

	providerv1 "github.com/autonomous-bits/nomos/libs/provider-proto/gen/go/nomos/provider/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

// Client implements the Provider interface by delegating to a gRPC provider service.
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
func (c *Client) Init(ctx context.Context, opts ProviderInitOptions) error {
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
		// Check for NotFound error
		if st, ok := status.FromError(err); ok {
			if st.Code() == codes.NotFound {
				return nil, fmt.Errorf("path not found: %v", path)
			}
		}
		return nil, fmt.Errorf("provider fetch failed: %w", err)
	}

	// Convert protobuf Struct to map[string]any
	return resp.Value.AsMap(), nil
}

// Info implements ProviderWithInfo by calling the gRPC Info RPC.
func (c *Client) Info() (alias string, version string) {
	ctx := context.Background()
	resp, err := c.client.Info(ctx, &providerv1.InfoRequest{})
	if err != nil {
		// If Info fails, return basic information
		return c.alias, "unknown"
	}

	return resp.Alias, resp.Version
}

// Close closes the underlying gRPC connection.
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
