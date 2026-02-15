//go:build integration
// +build integration

// Package test contains integration tests for property reference resolution (User Story 3).
// This file implements T065-T067 (TDD phase - tests written BEFORE implementation).
package test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/autonomous-bits/nomos/libs/compiler"
)

// T065: Test end-to-end property reference compilation with file provider
func TestIntegration_PropertyReference_EndToEnd(t *testing.T) {
	t.Helper()

	tests := []struct {
		name     string
		baseFile string
		appFile  string
		wantErr  bool
		errMatch string
	}{
		{
			name: "single property reference",
			baseFile: `api:
	url: 'http://api.example.com'
	timeout: 30
`,
			appFile: `source:
	alias: 'base'
	type: 'file'
	directory: '.'

config:
	api_url: @base:base:api.url
`,
			wantErr: false,
		},
		{
			name: "two-level nested property reference",
			baseFile: `database:
	host: 'prod.example.com'
	port: 5432
	ssl: true
`,
			appFile: `source:
	alias: 'base'
	type: 'file'
	directory: '.'

config:
	db_host: @base:base:database.host
`,
			wantErr: false,
		},
		{
			name: "deeply nested property reference (4 levels)",
			baseFile: `app:
	server:
		http:
			tls:
				cert: '/path/to/cert.pem'
				key: '/path/to/key.pem'
				enabled: true
`,
			appFile: `source:
	alias: 'base'
	type: 'file'
	directory: '.'

config:
	tls_cert: @base:base:app.server.http.tls.cert
`,
			wantErr: false,
		},
		{
			name: "property reference to nonexistent path",
			baseFile: `database:
	host: 'localhost'
	port: 5432
`,
			appFile: `source:
	alias: 'base'
	type: 'file'
	directory: '.'

config:
	invalid: @base:base:database.nonexistent
`,
			wantErr:  true,
			errMatch: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Write base file
			basePath := filepath.Join(tmpDir, "base.csl")
			if err := os.WriteFile(basePath, []byte(tt.baseFile), 0644); err != nil {
				t.Fatalf("failed to write base.csl: %v", err)
			}

			// Write app file
			appPath := filepath.Join(tmpDir, "app.csl")
			if err := os.WriteFile(appPath, []byte(tt.appFile), 0644); err != nil {
				t.Fatalf("failed to write app.csl: %v", err)
			}

			// Compile
			ctx := context.Background()
			registry := compiler.NewProviderRegistry()
			result := compiler.Compile(ctx, compiler.Options{
				Path:             appPath,
				ProviderRegistry: registry,
			})

			if tt.wantErr {
				if !result.HasErrors() {
					t.Fatal("expected error, got nil")
				}
				if tt.errMatch != "" && !contains(result.Error().Error(), tt.errMatch) {
					t.Errorf("error = %q, want substring %q", result.Error().Error(), tt.errMatch)
				}
				return
			}

			if result.HasErrors() {
				t.Fatalf("unexpected compilation error: %v", result.Error())
			}

			if len(result.Snapshot.Data) == 0 {
				t.Error("expected non-empty compiled data")
			}
		})
	}
}

// T066: Test string interpolation with property references
func TestIntegration_PropertyReference_StringInterpolation(t *testing.T) {
	t.Helper()

	baseFile := `api:
	host: 'api.example.com'
	protocol: 'https'
	port: 443
`

	appFile := `source:
	alias: 'base'
	type: 'file'
	directory: '.'

config:
	api_url: '@base:base:api.protocol://@base:base:api.host:@base:base:api.port'
`

	tmpDir := t.TempDir()

	basePath := filepath.Join(tmpDir, "base.csl")
	if err := os.WriteFile(basePath, []byte(baseFile), 0644); err != nil {
		t.Fatalf("failed to write base.csl: %v", err)
	}

	appPath := filepath.Join(tmpDir, "app.csl")
	if err := os.WriteFile(appPath, []byte(appFile), 0644); err != nil {
		t.Fatalf("failed to write app.csl: %v", err)
	}

	ctx := context.Background()
	registry := compiler.NewProviderRegistry()
	result := compiler.Compile(ctx, compiler.Options{
		Path:             appPath,
		ProviderRegistry: registry,
	})

	if result.HasErrors() {
		t.Fatalf("unexpected compilation error: %v", result.Error())
	}

	if len(result.Snapshot.Data) == 0 {
		t.Error("expected non-empty compiled data")
	}
}

// T067: Test config with all three reference modes (root, map, property)
func TestIntegration_AllReferenceModels_MixedUsage(t *testing.T) {
	t.Helper()

	baseFile := `api:
	url: 'http://api.example.com'
	timeout: 30
	retries: 3

database:
	host: 'localhost'
	port: 5432
	ssl: false
	pool:
		min: 5
		max: 20

server:
	host: 'localhost'
	port: 8080
	timeout: 60
`

	appFile := `source:
	alias: 'base'
	type: 'file'
	directory: '.'

application:
	api_url: @base:base:api.url
	db_host: @base:base:database.host
	server_port: @base:base:server.port

config:
	database_pool: @base:base:database.pool

database:
	host: @base:base:database.host
	port: 5433
	ssl: true
	pool: @base:base:database.pool
`

	tmpDir := t.TempDir()

	basePath := filepath.Join(tmpDir, "base.csl")
	if err := os.WriteFile(basePath, []byte(baseFile), 0644); err != nil {
		t.Fatalf("failed to write base.csl: %v", err)
	}

	appPath := filepath.Join(tmpDir, "app.csl")
	if err := os.WriteFile(appPath, []byte(appFile), 0644); err != nil {
		t.Fatalf("failed to write app.csl: %v", err)
	}

	ctx := context.Background()
	registry := compiler.NewProviderRegistry()
	result := compiler.Compile(ctx, compiler.Options{
		Path:             appPath,
		ProviderRegistry: registry,
	})

	if result.HasErrors() {
		t.Fatalf("unexpected compilation error: %v", result.Error())
	}

	if len(result.Snapshot.Data) == 0 {
		t.Error("expected non-empty compiled data")
	}
}
