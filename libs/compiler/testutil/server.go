package testutil

import (
	"context"
	"fmt"
	"net"
	"sync"

	providerv1 "github.com/autonomous-bits/nomos/libs/provider-proto/gen/go/nomos/provider/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

// FakeProviderServer is a minimal gRPC server implementation of nomos.provider.v1
// for testing purposes. It can be started in-process and provides configurable
// responses for all RPC methods.
//
// This server is meant for contract testing and unit tests, not production use.
type FakeProviderServer struct {
	providerv1.UnimplementedProviderServiceServer

	mu sync.RWMutex

	// Configuration
	alias   string
	version string
	pType   string

	// State tracking
	initCalled     bool
	initConfig     map[string]any
	healthStatus   providerv1.HealthResponse_Status
	healthMessage  string
	shutdownCalled bool

	// Configurable responses
	fetchResponses map[string]map[string]any // path (joined by "/") -> response map
	fetchErrors    map[string]error          // path (joined by "/") -> error to return

	// Error injection
	initError   error
	infoError   error
	healthError error

	// Call tracking for assertions
	initCalls     int
	fetchCalls    [][]string
	infoCalls     int
	healthCalls   int
	shutdownCalls int
}

// NewFakeProviderServer creates a new fake provider server with sensible defaults.
func NewFakeProviderServer(alias, version, providerType string) *FakeProviderServer {
	return &FakeProviderServer{
		alias:          alias,
		version:        version,
		pType:          providerType,
		healthStatus:   providerv1.HealthResponse_STATUS_OK,
		healthMessage:  "healthy",
		fetchResponses: make(map[string]map[string]any),
		fetchErrors:    make(map[string]error),
	}
}

// Init implements the Init RPC method.
func (s *FakeProviderServer) Init(_ context.Context, req *providerv1.InitRequest) (*providerv1.InitResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.initCalls++
	s.initCalled = true

	// Store config if provided
	if req.Config != nil {
		s.initConfig = req.Config.AsMap()
	}

	// Use alias from request if our alias is empty
	if s.alias == "" {
		s.alias = req.Alias
	}

	if s.initError != nil {
		return nil, s.initError
	}

	return &providerv1.InitResponse{}, nil
}

// Fetch implements the Fetch RPC method.
func (s *FakeProviderServer) Fetch(_ context.Context, req *providerv1.FetchRequest) (*providerv1.FetchResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.fetchCalls = append(s.fetchCalls, append([]string{}, req.Path...))

	pathKey := pathToKey(req.Path)

	// Check for configured error
	if err, ok := s.fetchErrors[pathKey]; ok {
		return nil, err
	}

	// Check for configured response
	if resp, ok := s.fetchResponses[pathKey]; ok {
		value, err := structpb.NewStruct(resp)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to create response struct: %v", err)
		}
		return &providerv1.FetchResponse{Value: value}, nil
	}

	// Default: return NotFound
	return nil, status.Errorf(codes.NotFound, "path not found: %v", req.Path)
}

// Info implements the Info RPC method.
func (s *FakeProviderServer) Info(_ context.Context, _ *providerv1.InfoRequest) (*providerv1.InfoResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.infoCalls++

	if s.infoError != nil {
		return nil, s.infoError
	}

	return &providerv1.InfoResponse{
		Alias:   s.alias,
		Version: s.version,
		Type:    s.pType,
	}, nil
}

// Health implements the Health RPC method.
func (s *FakeProviderServer) Health(_ context.Context, _ *providerv1.HealthRequest) (*providerv1.HealthResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.healthCalls++

	if s.healthError != nil {
		return nil, s.healthError
	}

	return &providerv1.HealthResponse{
		Status:  s.healthStatus,
		Message: s.healthMessage,
	}, nil
}

// Shutdown implements the Shutdown RPC method.
func (s *FakeProviderServer) Shutdown(_ context.Context, _ *providerv1.ShutdownRequest) (*providerv1.ShutdownResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.shutdownCalls++
	s.shutdownCalled = true

	return &providerv1.ShutdownResponse{}, nil
}

// Configuration methods

// SetFetchResponse configures a response for a specific path.
func (s *FakeProviderServer) SetFetchResponse(path []string, response map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.fetchResponses[pathToKey(path)] = response
}

// SetFetchError configures an error for a specific path.
func (s *FakeProviderServer) SetFetchError(path []string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.fetchErrors[pathToKey(path)] = err
}

// SetInitError configures Init to return an error.
func (s *FakeProviderServer) SetInitError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.initError = err
}

// SetInfoError configures Info to return an error.
func (s *FakeProviderServer) SetInfoError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.infoError = err
}

// SetHealthError configures Health to return an error.
func (s *FakeProviderServer) SetHealthError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.healthError = err
}

// SetHealthStatus sets the health status and message.
func (s *FakeProviderServer) SetHealthStatus(status providerv1.HealthResponse_Status, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.healthStatus = status
	s.healthMessage = message
}

// Query methods for assertions

// InitCalled returns whether Init has been called.
func (s *FakeProviderServer) InitCalled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.initCalled
}

// InitCount returns how many times Init was called.
func (s *FakeProviderServer) InitCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.initCalls
}

// InitConfig returns the config passed to the most recent Init call.
func (s *FakeProviderServer) InitConfig() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.initConfig
}

// FetchCount returns how many times Fetch was called.
func (s *FakeProviderServer) FetchCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.fetchCalls)
}

// FetchCalls returns all paths that were fetched.
func (s *FakeProviderServer) FetchCalls() [][]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to avoid race conditions
	result := make([][]string, len(s.fetchCalls))
	for i, call := range s.fetchCalls {
		result[i] = append([]string{}, call...)
	}
	return result
}

// InfoCount returns how many times Info was called.
func (s *FakeProviderServer) InfoCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.infoCalls
}

// HealthCount returns how many times Health was called.
func (s *FakeProviderServer) HealthCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.healthCalls
}

// ShutdownCalled returns whether Shutdown has been called.
func (s *FakeProviderServer) ShutdownCalled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.shutdownCalled
}

// ShutdownCount returns how many times Shutdown was called.
func (s *FakeProviderServer) ShutdownCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.shutdownCalls
}

// Reset clears all state and call tracking.
func (s *FakeProviderServer) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.initCalled = false
	s.initConfig = nil
	s.initCalls = 0
	s.fetchCalls = nil
	s.infoCalls = 0
	s.healthCalls = 0
	s.shutdownCalls = 0
	s.shutdownCalled = false
	s.fetchResponses = make(map[string]map[string]any)
	s.fetchErrors = make(map[string]error)
	s.initError = nil
	s.infoError = nil
	s.healthError = nil
	s.healthStatus = providerv1.HealthResponse_STATUS_OK
	s.healthMessage = "healthy"
}

// Helper function to convert path slice to map key
func pathToKey(path []string) string {
	if len(path) == 0 {
		return ""
	}
	result := ""
	for i, p := range path {
		if i > 0 {
			result += "/"
		}
		result += p
	}
	return result
}

// StartFakeProviderServer starts a fake provider gRPC server on a local listener.
// Returns the server instance, the address it's listening on, and an error.
// The caller should call Stop() on the server when done.
func StartFakeProviderServer(fakeImpl *FakeProviderServer) (*grpc.Server, string, error) {
	//nolint:noctx // Test helper, context not needed for ephemeral test listener
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, "", fmt.Errorf("failed to create listener: %w", err)
	}

	server := grpc.NewServer()
	providerv1.RegisterProviderServiceServer(server, fakeImpl)

	go func() {
		_ = server.Serve(lis)
	}()

	return server, lis.Addr().String(), nil
}
