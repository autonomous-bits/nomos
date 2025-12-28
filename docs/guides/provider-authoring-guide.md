# Provider Authoring Guide

Last updated: 2025-10-31

This guide explains how to create and distribute external providers for Nomos. Providers are standalone executables that communicate with the Nomos compiler via gRPC to supply configuration data from various sources (files, APIs, databases, cloud services, etc.).

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Understanding the Protocol](#understanding-the-protocol)
- [Implementation Guide](#implementation-guide)
- [Asset Naming and Packaging](#asset-naming-and-packaging)
- [Checksum Publishing](#checksum-publishing)
- [Release Instructions](#release-instructions)
- [Testing Your Provider](#testing-your-provider)
- [Common Patterns](#common-patterns)
- [Troubleshooting](#troubleshooting)
- [References](#references)

## Overview

### What is a Provider?

A provider is a standalone executable that:
- Implements the Nomos Provider gRPC service contract
- Runs as a subprocess managed by the Nomos compiler
- Fetches configuration data from external sources
- Returns structured data compatible with Nomos value types

### Architecture

```
┌─────────────────────────────────────┐
│   Nomos Compiler (nomos build)      │
│                                     │
│  ┌────────────────────────────┐     │
│  │  Provider Process Manager  │     │
│  └────────────────────────────┘     │
│              │                      │
│              │ gRPC                 │
│              ▼                      │
│  ┌────────────────────────────┐     │
│  │    Provider Subprocess     │     │
│  │  (your executable)         │     │
│  │                            │     │
│  │  Init, Fetch, Info, Health │     │
│  └────────────────────────────┘     │
└─────────────────────────────────────┘
```

### When to Write a Provider

Create a provider when you need to:
- Load configuration from a custom source (API, database, vault)
- Transform data from external formats to Nomos configurations
- Integrate proprietary systems with Nomos
- Reuse provider logic across multiple projects

## Quick Start

### Prerequisites

- Go 1.25+ (or another language with gRPC support)
- Protocol Buffer Compiler or Buf CLI
- Understanding of gRPC basics

### Minimal Provider (Go)

Here's a complete minimal provider in ~100 lines:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"

	providerv1 "github.com/autonomous-bits/nomos/libs/provider-proto/gen/go/nomos/provider/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

type myProvider struct {
	providerv1.UnimplementedProviderServiceServer
	alias string
}

func (p *myProvider) Init(ctx context.Context, req *providerv1.InitRequest) (*providerv1.InitResponse, error) {
	p.alias = req.Alias
	log.Printf("Provider initialized with alias: %s", p.alias)
	return &providerv1.InitResponse{}, nil
}

func (p *myProvider) Fetch(ctx context.Context, req *providerv1.FetchRequest) (*providerv1.FetchResponse, error) {
	// Fetch your data based on req.Path
	data := map[string]interface{}{
		"example": "data",
		"path":    req.Path,
	}

	value, err := structpb.NewStruct(data)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &providerv1.FetchResponse{Value: value}, nil
}

func (p *myProvider) Info(ctx context.Context, req *providerv1.InfoRequest) (*providerv1.InfoResponse, error) {
	return &providerv1.InfoResponse{
		Alias:   p.alias,
		Version: "1.0.0",
		Type:    "my-provider",
	}, nil
}

func (p *myProvider) Health(ctx context.Context, req *providerv1.HealthRequest) (*providerv1.HealthResponse, error) {
	return &providerv1.HealthResponse{
		Status:  providerv1.HealthResponse_STATUS_OK,
		Message: "healthy",
	}, nil
}

func (p *myProvider) Shutdown(ctx context.Context, req *providerv1.ShutdownRequest) (*providerv1.ShutdownResponse, error) {
	log.Println("Provider shutting down")
	return &providerv1.ShutdownResponse{}, nil
}

func main() {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// Print port to stdout for discovery
	addr := lis.Addr().(*net.TCPAddr)
	fmt.Printf("PORT=%d\n", addr.Port)
	os.Stdout.Sync()

	server := grpc.NewServer()
	providerv1.RegisterProviderServiceServer(server, &myProvider{})

	log.Printf("Provider listening on %s", lis.Addr())
	if err := server.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
```

### Building and Running

```bash
# Build the provider
go build -o nomos-provider-my-provider main.go

# Test it manually
./nomos-provider-my-provider
# Output: PORT=xxxxx
```

## Understanding the Protocol

### Protocol Buffer Contract

The Nomos Provider contract is defined in `libs/provider-proto/proto/nomos/provider/v1/provider.proto`:

```protobuf
service ProviderService {
  rpc Init(InitRequest) returns (InitResponse);
  rpc Fetch(FetchRequest) returns (FetchResponse);
  rpc Info(InfoRequest) returns (InfoResponse);
  rpc Health(HealthRequest) returns (HealthResponse);
  rpc Shutdown(ShutdownRequest) returns (ShutdownResponse);
}
```

### RPC Methods

#### Init

**Purpose**: Initialize the provider with configuration.

**When called**: Once per provider alias when first accessed.

**Request**:
```protobuf
message InitRequest {
  string alias = 1;                    // Provider instance name
  google.protobuf.Struct config = 2;   // Provider-specific config
  string source_file_path = 3;         // Path to .csl file
}
```

**Response**: Empty on success.

**Common errors**:
- `InvalidArgument`: Invalid or missing required configuration
- `FailedPrecondition`: Dependencies not available
- `Unavailable`: External resource unreachable

**Implementation tips**:
- Validate all required configuration keys
- Initialize connections (databases, APIs)
- Store alias for logging
- Return detailed error messages

#### Fetch

**Purpose**: Retrieve data at the specified path.

**When called**: Multiple times during compilation for each import reference.

**Request**:
```protobuf
message FetchRequest {
  repeated string path = 1;  // Path segments (e.g., ["database", "prod"])
}
```

**Response**:
```protobuf
message FetchResponse {
  google.protobuf.Struct value = 1;  // Structured data
}
```

**Common errors**:
- `NotFound`: Path does not exist
- `InvalidArgument`: Invalid path format
- `PermissionDenied`: Access denied
- `DeadlineExceeded`: Fetch timed out

**Implementation tips**:
- Support concurrent calls (use thread-safe data structures)
- Return maps for nested configuration
- Handle missing paths gracefully
- Implement caching for repeated fetches

#### Info

**Purpose**: Return provider metadata.

**When called**: Once during provider startup for logging and diagnostics.

**Request**: Empty.

**Response**:
```protobuf
message InfoResponse {
  string alias = 1;    // Provider instance name
  string version = 2;  // Implementation version (semver)
  string type = 3;     // Provider type identifier
}
```

**Implementation tips**:
- Use semantic versioning (e.g., "1.2.3")
- Type should match the provider name (e.g., "file", "http")
- Include build metadata if useful

#### Health

**Purpose**: Check provider operational status.

**When called**: After connecting and before first use.

**Request**: Empty.

**Response**:
```protobuf
message HealthResponse {
  enum Status {
    STATUS_UNKNOWN = 0;
    STATUS_OK = 1;
    STATUS_DEGRADED = 2;
  }
  Status status = 1;
  string message = 2;  // Optional diagnostic message
}
```

**Implementation tips**:
- Verify external dependencies are reachable
- Return `STATUS_OK` for normal operation
- Use `STATUS_DEGRADED` for partial functionality
- Include helpful diagnostic messages

#### Shutdown

**Purpose**: Gracefully shut down the provider.

**When called**: After compilation completes or on compiler exit.

**Request**: Empty.

**Response**: Empty.

**Implementation tips**:
- Close connections and release resources
- Best-effort; compiler may force terminate
- Keep shutdown fast (< 5 seconds)

## Implementation Guide

### Project Structure

Recommended repository layout:

```
nomos-provider-{type}/
├── go.mod
├── go.sum
├── README.md
├── CHANGELOG.md
├── LICENSE
├── .goreleaser.yml         # Release automation
├── cmd/
│   └── provider/
│       └── main.go         # Entry point
├── internal/
│   ├── provider.go         # Provider implementation
│   ├── config.go           # Configuration handling
│   └── fetcher.go          # Fetch logic
├── pkg/                    # Public packages (optional)
└── test/
    ├── integration_test.go
    └── fixtures/
```

### Step-by-Step Implementation

#### 1. Set Up Go Module

```bash
mkdir nomos-provider-my-provider
cd nomos-provider-my-provider
go mod init github.com/your-org/nomos-provider-my-provider
```

#### 2. Add Dependencies

```bash
go get github.com/autonomous-bits/nomos/libs/provider-proto@v0.1.0
go get google.golang.org/grpc@v1.70.0
go get google.golang.org/protobuf@v1.36.0
```

#### 3. Implement Provider Interface

Create `internal/provider.go`:

```go
package internal

import (
	"context"
	"fmt"
	"log"

	providerv1 "github.com/autonomous-bits/nomos/libs/provider-proto/gen/go/nomos/provider/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

type Provider struct {
	providerv1.UnimplementedProviderServiceServer
	alias   string
	config  *Config
	fetcher *Fetcher
}

func NewProvider() *Provider {
	return &Provider{}
}

func (p *Provider) Init(ctx context.Context, req *providerv1.InitRequest) (*providerv1.InitResponse, error) {
	p.alias = req.Alias

	// Parse configuration
	config, err := ParseConfig(req.Config.AsMap())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid config: %v", err)
	}
	p.config = config

	// Initialize fetcher
	p.fetcher, err = NewFetcher(config)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "failed to initialize: %v", err)
	}

	log.Printf("[%s] Initialized successfully", p.alias)
	return &providerv1.InitResponse{}, nil
}

func (p *Provider) Fetch(ctx context.Context, req *providerv1.FetchRequest) (*providerv1.FetchResponse, error) {
	if p.fetcher == nil {
		return nil, status.Error(codes.FailedPrecondition, "provider not initialized")
	}

	// Fetch data
	data, err := p.fetcher.Fetch(ctx, req.Path)
	if err != nil {
		if IsNotFoundError(err) {
			return nil, status.Errorf(codes.NotFound, "path not found: %v", req.Path)
		}
		return nil, status.Errorf(codes.Internal, "fetch failed: %v", err)
	}

	// Convert to protobuf Struct
	value, err := structpb.NewValue(data)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert value: %v", err)
	}

	return &providerv1.FetchResponse{
		Value: value.GetStructValue(),
	}, nil
}

func (p *Provider) Info(ctx context.Context, req *providerv1.InfoRequest) (*providerv1.InfoResponse, error) {
	return &providerv1.InfoResponse{
		Alias:   p.alias,
		Version: "1.0.0", // Replace with actual version
		Type:    "my-provider",
	}, nil
}

func (p *Provider) Health(ctx context.Context, req *providerv1.HealthRequest) (*providerv1.HealthResponse, error) {
	// Check external dependencies
	if err := p.fetcher.HealthCheck(ctx); err != nil {
		return &providerv1.HealthResponse{
			Status:  providerv1.HealthResponse_STATUS_DEGRADED,
			Message: fmt.Sprintf("health check failed: %v", err),
		}, nil
	}

	return &providerv1.HealthResponse{
		Status:  providerv1.HealthResponse_STATUS_OK,
		Message: "provider is healthy",
	}, nil
}

func (p *Provider) Shutdown(ctx context.Context, req *providerv1.ShutdownRequest) (*providerv1.ShutdownResponse, error) {
	log.Printf("[%s] Shutting down", p.alias)
	
	if p.fetcher != nil {
		p.fetcher.Close()
	}

	return &providerv1.ShutdownResponse{}, nil
}
```

#### 4. Create Configuration Handler

Create `internal/config.go`:

```go
package internal

import (
	"fmt"
)

type Config struct {
	BaseURL string
	APIKey  string
	// Add your configuration fields
}

func ParseConfig(configMap map[string]interface{}) (*Config, error) {
	config := &Config{}

	// Extract required fields
	baseURL, ok := configMap["base_url"].(string)
	if !ok || baseURL == "" {
		return nil, fmt.Errorf("missing required config: base_url")
	}
	config.BaseURL = baseURL

	// Extract optional fields
	if apiKey, ok := configMap["api_key"].(string); ok {
		config.APIKey = apiKey
	}

	return config, nil
}
```

#### 5. Implement Fetcher Logic

Create `internal/fetcher.go`:

```go
package internal

import (
	"context"
	"fmt"
)

type Fetcher struct {
	config *Config
	// Add your fetcher state
}

func NewFetcher(config *Config) (*Fetcher, error) {
	// Initialize connections, clients, etc.
	return &Fetcher{
		config: config,
	}, nil
}

func (f *Fetcher) Fetch(ctx context.Context, path []string) (interface{}, error) {
	// Implement your fetch logic
	// Return map[string]interface{} for nested structures
	// Return []interface{} for arrays
	// Return string, number, bool for primitives
	
	// Example:
	if len(path) == 0 {
		return nil, fmt.Errorf("empty path")
	}

	data := map[string]interface{}{
		"key":   path[0],
		"value": "example data",
	}

	return data, nil
}

func (f *Fetcher) HealthCheck(ctx context.Context) error {
	// Verify external dependencies are reachable
	return nil
}

func (f *Fetcher) Close() {
	// Clean up resources
}

type NotFoundError struct {
	Path []string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("path not found: %v", e.Path)
}

func IsNotFoundError(err error) bool {
	_, ok := err.(*NotFoundError)
	return ok
}
```

#### 6. Create Main Entry Point

Create `cmd/provider/main.go`:

```go
package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	providerv1 "github.com/autonomous-bits/nomos/libs/provider-proto/gen/go/nomos/provider/v1"
	"github.com/your-org/nomos-provider-my-provider/internal"
	"google.golang.org/grpc"
)

func main() {
	// Listen on random available port
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// Print port to stdout for discovery by compiler
	addr := lis.Addr().(*net.TCPAddr)
	fmt.Printf("PORT=%d\n", addr.Port)
	os.Stdout.Sync()

	// Create gRPC server
	server := grpc.NewServer()
	provider := internal.NewProvider()
	providerv1.RegisterProviderServiceServer(server, provider)

	// Handle graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-stop
		log.Println("Received shutdown signal")
		server.GracefulStop()
	}()

	// Start serving
	log.Printf("Provider listening on %s", lis.Addr())
	if err := server.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
```

### Error Handling Best Practices

Use appropriate gRPC status codes:

```go
import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Invalid configuration
return nil, status.Errorf(codes.InvalidArgument, "invalid config: %v", err)

// Resource not found
return nil, status.Errorf(codes.NotFound, "path not found: %v", path)

// Permission issues
return nil, status.Errorf(codes.PermissionDenied, "access denied: %v", err)

// External dependency unavailable
return nil, status.Errorf(codes.Unavailable, "service unavailable: %v", err)

// Timeout
return nil, status.Errorf(codes.DeadlineExceeded, "operation timed out")

// Internal errors
return nil, status.Errorf(codes.Internal, "internal error: %v", err)
```

## Asset Naming and Packaging

### Naming Convention

Follow the standard naming pattern:

```
nomos-provider-{type}-{version}-{os}-{arch}[.extension]
```

Examples:
- `nomos-provider-file-1.0.0-darwin-arm64`
- `nomos-provider-file-1.0.0-darwin-amd64`
- `nomos-provider-file-1.0.0-linux-amd64`
- `nomos-provider-file-1.0.0-linux-arm64`
- `nomos-provider-file-1.0.0-windows-amd64.exe`

### Supported Platforms

Target these platforms at minimum:
- **darwin/arm64** (Apple Silicon Macs)
- **darwin/amd64** (Intel Macs)
- **linux/amd64** (Most Linux servers)
- **linux/arm64** (ARM Linux servers)

Optional:
- **windows/amd64** (Windows support)

### Build Commands

```bash
# Build for multiple platforms
GOOS=darwin GOARCH=arm64 go build -o nomos-provider-my-provider-1.0.0-darwin-arm64 ./cmd/provider
GOOS=darwin GOARCH=amd64 go build -o nomos-provider-my-provider-1.0.0-darwin-amd64 ./cmd/provider
GOOS=linux GOARCH=amd64 go build -o nomos-provider-my-provider-1.0.0-linux-amd64 ./cmd/provider
GOOS=linux GOARCH=arm64 go build -o nomos-provider-my-provider-1.0.0-linux-arm64 ./cmd/provider
GOOS=windows GOARCH=amd64 go build -o nomos-provider-my-provider-1.0.0-windows-amd64.exe ./cmd/provider
```

### Automated Build with GoReleaser

Create `.goreleaser.yml`:

```yaml
version: 2

project_name: nomos-provider-my-provider

before:
  hooks:
    - go mod download
    - go mod verify

builds:
  - id: provider
    main: ./cmd/provider
    binary: nomos-provider-my-provider-{{ .Version }}-{{ .Os }}-{{ .Arch }}
    env:
      - CGO_ENABLED=0
    goos:
      - darwin
      - linux
      - windows
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X main.version={{ .Version }}
      - -X main.commit={{ .Commit }}
      - -X main.date={{ .Date }}

archives:
  - format: binary

checksum:
  name_template: 'checksums.txt'
  algorithm: sha256

release:
  github:
    owner: your-org
    name: nomos-provider-my-provider
  draft: false
  prerelease: auto
  name_template: "{{ .Tag }}"
```

Build release:

```bash
# Install goreleaser
brew install goreleaser

# Test the build
goreleaser build --snapshot --clean

# Create a release
git tag v1.0.0
goreleaser release --clean
```

## Checksum Publishing

### Generating Checksums

```bash
# Generate SHA256 checksums for all binaries
sha256sum nomos-provider-my-provider-* > SHA256SUMS

# Or use shasum on macOS
shasum -a 256 nomos-provider-my-provider-* > SHA256SUMS
```

Example `SHA256SUMS` file:

```
a1b2c3d4e5f6... nomos-provider-my-provider-1.0.0-darwin-arm64
b2c3d4e5f6a7... nomos-provider-my-provider-1.0.0-darwin-amd64
c3d4e5f6a7b8... nomos-provider-my-provider-1.0.0-linux-amd64
d4e5f6a7b8c9... nomos-provider-my-provider-1.0.0-linux-arm64
```

### Verification

Users can verify checksums:

```bash
# Verify checksum
sha256sum -c SHA256SUMS --ignore-missing

# Or on macOS
shasum -a 256 -c SHA256SUMS --ignore-missing
```

### Security Best Practices

1. **Always publish checksums** with your releases
2. **Sign releases** with GPG for additional verification (optional but recommended)
3. **Use HTTPS** for all downloads
4. **Document verification steps** in your README

## Release Instructions

### GitHub Releases

#### Manual Release

1. **Create and push tag**:
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. **Build binaries**:
   ```bash
   # Build for all platforms
   make build-all
   
   # Generate checksums
   sha256sum nomos-provider-* > SHA256SUMS
   ```

3. **Create GitHub Release**:
   - Go to `https://github.com/your-org/nomos-provider-my-provider/releases/new`
   - Select tag: `v1.0.0`
   - Release title: `v1.0.0`
   - Description: Include changelog
   - Upload assets:
     - All binary files
     - `SHA256SUMS`

4. **Publish release**

#### Automated with GoReleaser

1. **Configure GitHub token**:
   ```bash
   export GITHUB_TOKEN="your-token"
   ```

2. **Create and push tag**:
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

3. **Run GoReleaser**:
   ```bash
   goreleaser release --clean
   ```

GoReleaser will:
- Build binaries for all platforms
- Generate checksums
- Create GitHub release
- Upload all assets
- Generate release notes from commits

### Release Checklist

- [ ] Update CHANGELOG.md with new version
- [ ] Update version in code
- [ ] Run all tests
- [ ] Build and test locally
- [ ] Create git tag with semantic version
- [ ] Push tag to GitHub
- [ ] Build release binaries (manual or GoReleaser)
- [ ] Generate checksums
- [ ] Create GitHub Release
- [ ] Upload binaries and checksums
- [ ] Write release notes
- [ ] Verify downloads work
- [ ] Update documentation if needed
- [ ] Announce release (if applicable)

### Semantic Versioning

Follow [Semantic Versioning](https://semver.org/):

- **MAJOR** version for incompatible API changes (e.g., 1.0.0 → 2.0.0)
- **MINOR** version for backwards-compatible functionality (e.g., 1.0.0 → 1.1.0)
- **PATCH** version for backwards-compatible bug fixes (e.g., 1.0.0 → 1.0.1)

Examples:
- `v1.0.0` - Initial stable release
- `v1.1.0` - New fetch feature added
- `v1.1.1` - Bug fix in fetch logic
- `v2.0.0` - Breaking change to config format

### Pre-release Versions

Use pre-release tags for testing:

- `v1.0.0-alpha.1` - Alpha release
- `v1.0.0-beta.1` - Beta release
- `v1.0.0-rc.1` - Release candidate

## Testing Your Provider

### Manual Testing

Create a test script:

```bash
#!/bin/bash
# test-provider.sh

# Start provider in background
./nomos-provider-my-provider &
PROVIDER_PID=$!

# Wait for startup
sleep 1

# Use grpcurl to test (requires grpcurl installed)
grpcurl -plaintext -d '{"alias": "test"}' \
  localhost:50051 \
  nomos.provider.v1.ProviderService/Info

# Cleanup
kill $PROVIDER_PID
```

### Integration Testing with Nomos

1. **Install provider locally**:
   ```bash
   mkdir -p .nomos/providers/my-provider/1.0.0/darwin-arm64
   cp nomos-provider-my-provider .nomos/providers/my-provider/1.0.0/darwin-arm64/provider
   chmod +x .nomos/providers/my-provider/1.0.0/darwin-arm64/provider
   ```

2. **Create test config** `test.csl`:
   ```csl
   source my_provider as my_data {
     version = "1.0.0"
     config = {
       base_url = "https://example.com"
     }
   }
   
   config = import my_data["test"]
   ```

3. **Create lock file** `.nomos/providers.lock.json`:
   ```json
   {
     "providers": [
       {
         "alias": "my_data",
         "type": "my-provider",
         "version": "1.0.0",
         "os": "darwin",
         "arch": "arm64",
         "source": {
           "local": {
             "path": ".nomos/providers/my-provider/1.0.0/darwin-arm64/provider"
           }
         },
         "path": ".nomos/providers/my-provider/1.0.0/darwin-arm64/provider"
       }
     ]
   }
   ```

4. **Run Nomos build**:
   ```bash
   nomos build test.csl
   ```

### Automated Tests

Create integration tests:

```go
package test

import (
	"context"
	"testing"
	
	"github.com/your-org/nomos-provider-my-provider/internal"
	providerv1 "github.com/autonomous-bits/nomos/libs/provider-proto/gen/go/nomos/provider/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestProvider_Init(t *testing.T) {
	provider := internal.NewProvider()
	
	config, _ := structpb.NewStruct(map[string]interface{}{
		"base_url": "https://example.com",
	})
	
	req := &providerv1.InitRequest{
		Alias:  "test",
		Config: config,
	}
	
	_, err := provider.Init(context.Background(), req)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
}

func TestProvider_Fetch(t *testing.T) {
	provider := setupProvider(t)
	
	req := &providerv1.FetchRequest{
		Path: []string{"test", "data"},
	}
	
	resp, err := provider.Fetch(context.Background(), req)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	
	if resp.Value == nil {
		t.Fatal("Expected non-nil value")
	}
}

func setupProvider(t *testing.T) *internal.Provider {
	t.Helper()
	
	provider := internal.NewProvider()
	config, _ := structpb.NewStruct(map[string]interface{}{
		"base_url": "https://example.com",
	})
	
	req := &providerv1.InitRequest{
		Alias:  "test",
		Config: config,
	}
	
	if _, err := provider.Init(context.Background(), req); err != nil {
		t.Fatalf("Failed to initialize provider: %v", err)
	}
	
	return provider
}
```

## Common Patterns

### Configuration with Environment Variables

```go
func ParseConfig(configMap map[string]interface{}) (*Config, error) {
	config := &Config{}
	
	// Allow config to be overridden by environment variables
	if baseURL, ok := configMap["base_url"].(string); ok {
		config.BaseURL = baseURL
	} else if envURL := os.Getenv("MY_PROVIDER_BASE_URL"); envURL != "" {
		config.BaseURL = envURL
	} else {
		return nil, fmt.Errorf("missing required config: base_url")
	}
	
	return config, nil
}
```

### Caching Fetch Results

```go
type Fetcher struct {
	config *Config
	cache  map[string]interface{}
	mu     sync.RWMutex
}

func (f *Fetcher) Fetch(ctx context.Context, path []string) (interface{}, error) {
	key := strings.Join(path, "/")
	
	// Check cache
	f.mu.RLock()
	if cached, ok := f.cache[key]; ok {
		f.mu.RUnlock()
		return cached, nil
	}
	f.mu.RUnlock()
	
	// Fetch data
	data, err := f.fetchFromSource(ctx, path)
	if err != nil {
		return nil, err
	}
	
	// Store in cache
	f.mu.Lock()
	f.cache[key] = data
	f.mu.Unlock()
	
	return data, nil
}
```

### Concurrent Fetch Safety

```go
type Fetcher struct {
	config *Config
	client *http.Client // HTTP client is thread-safe
}

func (f *Fetcher) Fetch(ctx context.Context, path []string) (interface{}, error) {
	// Multiple goroutines can call Fetch concurrently
	// Use thread-safe operations
	
	url := f.config.BaseURL + "/" + strings.Join(path, "/")
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	// Parse and return data
	// ...
}
```

### Structured Logging

```go
import (
	"log"
	"os"
)

type Provider struct {
	providerv1.UnimplementedProviderServiceServer
	alias  string
	logger *log.Logger
}

func NewProvider() *Provider {
	return &Provider{
		logger: log.New(os.Stderr, "[my-provider] ", log.LstdFlags),
	}
}

func (p *Provider) Init(ctx context.Context, req *providerv1.InitRequest) (*providerv1.InitResponse, error) {
	p.alias = req.Alias
	p.logger.Printf("[%s] Initializing with config: %v", p.alias, req.Config.AsMap())
	
	// ...
	
	p.logger.Printf("[%s] Initialized successfully", p.alias)
	return &providerv1.InitResponse{}, nil
}
```

## Troubleshooting

### Common Issues

#### Provider Fails to Start

**Symptoms**: Compiler reports "failed to start provider"

**Solutions**:
- Verify binary has execute permissions: `chmod +x provider`
- Check binary architecture matches system: `file provider`
- Test running provider manually: `./provider`
- Check for missing dependencies: `ldd provider` (Linux)

#### Port Discovery Issues

**Symptoms**: Compiler reports "failed to discover port"

**Solutions**:
- Ensure provider prints `PORT=xxxxx` to stdout as first output
- Flush stdout after printing: `os.Stdout.Sync()`
- Don't print anything else to stdout before port
- Use stderr for logs: `log.SetOutput(os.Stderr)`

#### Init Fails with Invalid Configuration

**Symptoms**: Provider returns `InvalidArgument` error

**Solutions**:
- Validate all required config keys in Init
- Provide clear error messages for missing/invalid config
- Document required configuration in README
- Check config types match expectations

#### Fetch Returns Wrong Data Type

**Symptoms**: Compiler reports type conversion errors

**Solutions**:
- Return `map[string]interface{}` for objects
- Return `[]interface{}` for arrays
- Use `structpb.NewStruct()` correctly
- Test with various data types

#### Permission Denied on macOS

**Symptoms**: macOS blocks provider from running

**Solutions**:
- Remove quarantine attribute: `xattr -d com.apple.quarantine provider`
- Or allow in System Settings → Privacy & Security
- Users should download from trusted sources only

#### Checksum Verification Fails

**Symptoms**: `nomos init` reports checksum mismatch

**Solutions**:
- Regenerate checksums: `sha256sum provider > SHA256SUMS`
- Ensure checksums file is uploaded correctly
- Check for corruption during download
- Verify binary wasn't modified after checksum generation

### Debugging Tips

1. **Enable verbose logging**:
   ```bash
   nomos build --verbose test.csl
   ```

2. **Check provider stderr**:
   Provider logs are captured and shown with `--verbose`

3. **Test provider standalone**:
   ```bash
   ./provider
   # Should print PORT=xxxxx
   ```

4. **Use grpcurl for manual testing**:
   ```bash
   # Install grpcurl
   brew install grpcurl
   
   # Start provider and note the port
   ./provider
   # PORT=50051
   
   # Test Init
   grpcurl -plaintext -d '{"alias": "test", "config": {"key": "value"}}' \
     localhost:50051 \
     nomos.provider.v1.ProviderService/Init
   
   # Test Fetch
   grpcurl -plaintext -d '{"path": ["test"]}' \
     localhost:50051 \
     nomos.provider.v1.ProviderService/Fetch
   ```

5. **Check lock file**:
   ```bash
   cat .nomos/providers.lock.json | jq
   ```

6. **Verify binary architecture**:
   ```bash
   file .nomos/providers/my-provider/1.0.0/darwin-arm64/provider
   ```

## References

### Official Documentation

- [Nomos Provider Proto](../../libs/provider-proto/README.md) - gRPC service contract
- [External Providers Architecture](../architecture/nomos-external-providers-feature-breakdown.md) - Feature specification
- [External Providers Migration Guide](./external-providers-migration.md) - Migration from in-process providers
- [Terraform Providers Overview](./terraform-providers-overview.md) - Comparison with Terraform's model

### Protocol Buffers & gRPC

- [Protocol Buffers](https://protobuf.dev/) - Google's data serialization format
- [gRPC Go](https://grpc.io/docs/languages/go/) - gRPC documentation for Go
- [Buf](https://buf.build/) - Modern Protobuf tooling

### Go Development

- [Go Modules Reference](https://go.dev/ref/mod) - Go dependency management
- [Effective Go](https://go.dev/doc/effective_go) - Go best practices
- [GoReleaser](https://goreleaser.com/) - Release automation

### Example Providers

- [nomos-provider-file](https://github.com/autonomous-bits/nomos-provider-file) - Reference implementation

## Getting Help

- **GitHub Issues**: [https://github.com/autonomous-bits/nomos/issues](https://github.com/autonomous-bits/nomos/issues)
- **Discussions**: [https://github.com/autonomous-bits/nomos/discussions](https://github.com/autonomous-bits/nomos/discussions)
- **Documentation**: [https://github.com/autonomous-bits/nomos/tree/main/docs](https://github.com/autonomous-bits/nomos/tree/main/docs)

---

**Last updated**: 2025-10-31  
**Version**: 1.0.0
