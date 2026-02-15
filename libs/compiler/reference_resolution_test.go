package compiler

import (
	"fmt"
	"strings"
	"testing"

	"github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
)

func TestReferenceMode_String(t *testing.T) {
	tests := []struct {
		name string
		mode ReferenceMode
		want string
	}{
		{
			name: "PropertyMode",
			mode: PropertyMode,
			want: "Property",
		},
		{
			name: "MapMode",
			mode: MapMode,
			want: "Map",
		},
		{
			name: "RootMode",
			mode: RootMode,
			want: "Root",
		},
		{
			name: "Unknown mode",
			mode: ReferenceMode(999),
			want: "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.mode.String()
			if got != tt.want {
				t.Errorf("ReferenceMode.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPathRef_String(t *testing.T) {
	tests := []struct {
		name string
		ref  PathRef
		want string
	}{
		{
			name: "simple path",
			ref:  PathRef{Alias: "base", Path: "database"},
			want: "base:database",
		},
		{
			name: "path with hyphens",
			ref:  PathRef{Alias: "my-config", Path: "app-settings"},
			want: "my-config:app-settings",
		},
		{
			name: "empty alias",
			ref:  PathRef{Alias: "", Path: "database"},
			want: ":database",
		},
		{
			name: "empty path",
			ref:  PathRef{Alias: "base", Path: ""},
			want: "base:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ref.String()
			if got != tt.want {
				t.Errorf("PathRef.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolutionContext_Push_Success(t *testing.T) {
	tests := []struct {
		name         string
		initialStack []PathRef
		pushAlias    string
		pushPath     []string
		wantLen      int
	}{
		{
			name:         "push to empty stack",
			initialStack: []PathRef{},
			pushAlias:    "base",
			pushPath:     []string{"database"},
			wantLen:      1,
		},
		{
			name: "push to non-empty stack",
			initialStack: []PathRef{
				{Alias: "base", Path: "common"},
			},
			pushAlias: "base",
			pushPath:  []string{"database"},
			wantLen:   2,
		},
		{
			name: "push different alias",
			initialStack: []PathRef{
				{Alias: "base", Path: "database"},
			},
			pushAlias: "other",
			pushPath:  []string{"database"},
			wantLen:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &ResolutionContext{Stack: tt.initialStack}

			err := ctx.Push(tt.pushAlias, tt.pushPath)
			if err != nil {
				t.Errorf("Push() unexpected error: %v", err)
			}

			if len(ctx.Stack) != tt.wantLen {
				t.Errorf("Stack length = %d, want %d", len(ctx.Stack), tt.wantLen)
			}

			// Verify the pushed item is at the end
			if len(ctx.Stack) > 0 {
				last := ctx.Stack[len(ctx.Stack)-1]
				if last.Alias != tt.pushAlias || last.Path != strings.Join(tt.pushPath, ":") {
					t.Errorf("Last stack item = %v, want {%s %s}",
						last, tt.pushAlias, strings.Join(tt.pushPath, ":"))
				}
			}
		})
	}
}

func TestResolutionContext_Push_CircularReference(t *testing.T) {
	tests := []struct {
		name         string
		initialStack []PathRef
		pushAlias    string
		pushPath     []string
		wantErrMsg   string
	}{
		{
			name: "direct cycle",
			initialStack: []PathRef{
				{Alias: "base", Path: "database"},
			},
			pushAlias:  "base",
			pushPath:   []string{"database"},
			wantErrMsg: "circular reference detected: base:database → base:database",
		},
		{
			name: "two-hop cycle",
			initialStack: []PathRef{
				{Alias: "base", Path: "app"},
				{Alias: "base", Path: "common"},
			},
			pushAlias:  "base",
			pushPath:   []string{"app"},
			wantErrMsg: "circular reference detected: base:app → base:common → base:app",
		},
		{
			name: "three-hop cycle",
			initialStack: []PathRef{
				{Alias: "base", Path: "a"},
				{Alias: "base", Path: "b"},
				{Alias: "base", Path: "c"},
			},
			pushAlias:  "base",
			pushPath:   []string{"a"},
			wantErrMsg: "circular reference detected: base:a → base:b → base:c → base:a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &ResolutionContext{Stack: tt.initialStack}

			err := ctx.Push(tt.pushAlias, tt.pushPath)
			if err == nil {
				t.Fatal("Push() expected error, got nil")
			}

			if err.Error() != tt.wantErrMsg {
				t.Errorf("Push() error = %q, want %q", err.Error(), tt.wantErrMsg)
			}
		})
	}
}

func TestResolutionContext_Pop(t *testing.T) {
	tests := []struct {
		name         string
		initialStack []PathRef
		wantLen      int
	}{
		{
			name:         "pop from empty stack",
			initialStack: []PathRef{},
			wantLen:      0,
		},
		{
			name: "pop from single-item stack",
			initialStack: []PathRef{
				{Alias: "base", Path: "database"},
			},
			wantLen: 0,
		},
		{
			name: "pop from multi-item stack",
			initialStack: []PathRef{
				{Alias: "base", Path: "app"},
				{Alias: "base", Path: "common"},
			},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &ResolutionContext{Stack: tt.initialStack}

			ctx.Pop()

			if len(ctx.Stack) != tt.wantLen {
				t.Errorf("Stack length after Pop() = %d, want %d", len(ctx.Stack), tt.wantLen)
			}
		})
	}
}

func TestResolutionContext_formatCycle(t *testing.T) {
	tests := []struct {
		name  string
		stack []PathRef
		ref   PathRef
		want  string
	}{
		{
			name:  "direct cycle",
			stack: []PathRef{{Alias: "base", Path: "app"}},
			ref:   PathRef{Alias: "base", Path: "app"},
			want:  "base:app → base:app",
		},
		{
			name: "two-hop cycle",
			stack: []PathRef{
				{Alias: "base", Path: "app"},
				{Alias: "base", Path: "common"},
			},
			ref:  PathRef{Alias: "base", Path: "app"},
			want: "base:app → base:common → base:app",
		},
		{
			name: "cross-alias cycle",
			stack: []PathRef{
				{Alias: "config", Path: "app"},
				{Alias: "shared", Path: "common"},
			},
			ref:  PathRef{Alias: "config", Path: "app"},
			want: "config:app → shared:common → config:app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &ResolutionContext{Stack: tt.stack}

			got := ctx.formatCycle(tt.ref)
			if got != tt.want {
				t.Errorf("formatCycle() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolvedReference_PropertyMode(t *testing.T) {
	// Test that PropertyMode uses Value field
	resolved := ResolvedReference{
		Mode: PropertyMode,
		Value: &ast.StringLiteral{
			Value: "8080",
		},
		Entries: nil,
	}

	if resolved.Mode != PropertyMode {
		t.Errorf("Mode = %v, want PropertyMode", resolved.Mode)
	}

	if resolved.Value == nil {
		t.Error("Value should be populated for PropertyMode")
	}

	if resolved.Entries != nil {
		t.Error("Entries should be nil for PropertyMode")
	}
}

func TestResolvedReference_MapMode(t *testing.T) {
	// Test that MapMode uses Entries field
	resolved := ResolvedReference{
		Mode:  MapMode,
		Value: nil,
		Entries: map[string]ast.Expr{
			"host": &ast.StringLiteral{Value: "localhost"},
			"port": &ast.StringLiteral{Value: "5432"},
		},
	}

	if resolved.Mode != MapMode {
		t.Errorf("Mode = %v, want MapMode", resolved.Mode)
	}

	if resolved.Value != nil {
		t.Error("Value should be nil for MapMode")
	}

	if resolved.Entries == nil {
		t.Error("Entries should be populated for MapMode")
	}

	if len(resolved.Entries) != 2 {
		t.Errorf("Entries length = %d, want 2", len(resolved.Entries))
	}
}

func TestResolvedReference_RootMode(t *testing.T) {
	// Test that RootMode uses Entries field
	resolved := ResolvedReference{
		Mode:  RootMode,
		Value: nil,
		Entries: map[string]ast.Expr{
			"host": &ast.StringLiteral{Value: "prod.example.com"},
			"port": &ast.StringLiteral{Value: "5432"},
		},
	}

	if resolved.Mode != RootMode {
		t.Errorf("Mode = %v, want RootMode", resolved.Mode)
	}

	if resolved.Value != nil {
		t.Error("Value should be nil for RootMode")
	}

	if resolved.Entries == nil {
		t.Error("Entries should be populated for RootMode")
	}
}

// T023: Test DetermineReferenceMode returns RootMode for empty Path
func TestDetermineReferenceMode_RootReference(t *testing.T) {
	t.Helper()

	tests := []struct {
		name         string
		ref          *ast.ReferenceExpr
		resourceData map[string]any
		wantMode     ReferenceMode
	}{
		{
			name: "empty path indicates root mode",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{}, // Empty = root
			},
			resourceData: map[string]any{
				"host": "localhost",
				"port": 5432,
			},
			wantMode: RootMode,
		},
		{
			name: "single property path",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"host"},
			},
			resourceData: map[string]any{
				"host": "localhost",
				"port": 5432,
			},
			wantMode: PropertyMode,
		},
		{
			name: "nested path to scalar",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"pool", "max_connections"},
			},
			resourceData: map[string]any{
				"pool": map[string]any{
					"max_connections": 100,
				},
			},
			wantMode: PropertyMode,
		},
		{
			name: "nested path to map",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"pool"},
			},
			resourceData: map[string]any{
				"pool": map[string]any{
					"max_connections": 100,
					"min_connections": 10,
				},
			},
			wantMode: MapMode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This will fail until DetermineReferenceMode is implemented
			mode := DetermineReferenceMode(tt.ref, tt.resourceData)

			if mode != tt.wantMode {
				t.Errorf("DetermineReferenceMode() = %v, want %v", mode, tt.wantMode)
			}
		})
	}
}

// T024: Test ResolveReference with root reference fetches complete data
func TestResolveReference_RootMode(t *testing.T) {
	t.Helper()

	tests := []struct {
		name         string
		ref          *ast.ReferenceExpr
		resourceData map[string]any
		wantEntries  map[string]any
		wantErr      bool
	}{
		{
			name: "root reference includes all properties",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{}, // Root reference
			},
			resourceData: map[string]any{
				"host": "prod.example.com",
				"port": 5432,
				"ssl":  true,
			},
			wantEntries: map[string]any{
				"host": "prod.example.com",
				"port": 5432,
				"ssl":  true,
			},
			wantErr: false,
		},
		{
			name: "root reference with nested data",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{},
			},
			resourceData: map[string]any{
				"server": map[string]any{
					"host": "localhost",
					"port": 8080,
				},
				"database": map[string]any{
					"host": "db.example.com",
					"port": 5432,
				},
			},
			wantEntries: map[string]any{
				"server": map[string]any{
					"host": "localhost",
					"port": 8080,
				},
				"database": map[string]any{
					"host": "db.example.com",
					"port": 5432,
				},
			},
			wantErr: false,
		},
		{
			name: "root reference on empty data",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{},
			},
			resourceData: map[string]any{},
			wantEntries:  map[string]any{},
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create resolution context for cycle detection
			resCtx := &ResolutionContext{Stack: []PathRef{}}
			// This will fail until ResolveReference is implemented
			resolved, err := ResolveReference(tt.ref, tt.resourceData, resCtx)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if resolved.Mode != RootMode {
				t.Errorf("Mode = %v, want RootMode", resolved.Mode)
			}

			if resolved.Entries == nil {
				t.Fatal("Entries should not be nil for RootMode")
			}

			// Compare entries (would need deep comparison helper in real implementation)
			if len(resolved.Entries) != len(tt.wantEntries) {
				t.Errorf("Entries count = %d, want %d", len(resolved.Entries), len(tt.wantEntries))
			}
		})
	}
}

// T044: Test DetermineReferenceMode returns MapMode for path resolving to map
func TestDetermineReferenceMode_MapMode(t *testing.T) {
	t.Helper()

	tests := []struct {
		name         string
		ref          *ast.ReferenceExpr
		resourceData map[string]any
		wantMode     ReferenceMode
	}{
		{
			name: "single level path to map",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"database"},
			},
			resourceData: map[string]any{
				"database": map[string]any{
					"host": "localhost",
					"port": 5432,
				},
				"api": map[string]any{
					"url": "http://api.example.com",
				},
			},
			wantMode: MapMode,
		},
		{
			name: "two-level path to map",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"app", "server"},
			},
			resourceData: map[string]any{
				"app": map[string]any{
					"server": map[string]any{
						"host": "localhost",
						"port": 8080,
					},
					"version": "1.0",
				},
			},
			wantMode: MapMode,
		},
		{
			name: "three-level path to map",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"app", "server", "config"},
			},
			resourceData: map[string]any{
				"app": map[string]any{
					"server": map[string]any{
						"config": map[string]any{
							"timeout": 30,
							"retries": 3,
						},
					},
				},
			},
			wantMode: MapMode,
		},
		{
			name: "path to scalar should be PropertyMode not MapMode",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"database", "host"},
			},
			resourceData: map[string]any{
				"database": map[string]any{
					"host": "localhost",
					"port": 5432,
				},
			},
			wantMode: PropertyMode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode := DetermineReferenceMode(tt.ref, tt.resourceData)

			if mode != tt.wantMode {
				t.Errorf("DetermineReferenceMode() = %v, want %v", mode, tt.wantMode)
			}
		})
	}
}

// T045: Test path navigation function traverses nested maps correctly
func TestNavigatePath_NestedMaps(t *testing.T) {
	t.Helper()

	tests := []struct {
		name     string
		data     map[string]any
		path     []string
		want     any
		wantErr  bool
		errMatch string
	}{
		{
			name: "1-level navigation",
			data: map[string]any{
				"database": map[string]any{
					"host": "localhost",
					"port": 5432,
				},
				"api": "http://api.example.com",
			},
			path: []string{"database"},
			want: map[string]any{
				"host": "localhost",
				"port": 5432,
			},
			wantErr: false,
		},
		{
			name: "2-level navigation",
			data: map[string]any{
				"app": map[string]any{
					"server": map[string]any{
						"host": "localhost",
						"port": 8080,
					},
					"version": "1.0",
				},
			},
			path: []string{"app", "server"},
			want: map[string]any{
				"host": "localhost",
				"port": 8080,
			},
			wantErr: false,
		},
		{
			name: "3-level navigation",
			data: map[string]any{
				"app": map[string]any{
					"server": map[string]any{
						"config": map[string]any{
							"timeout": 30,
							"retries": 3,
						},
					},
				},
			},
			path: []string{"app", "server", "config"},
			want: map[string]any{
				"timeout": 30,
				"retries": 3,
			},
			wantErr: false,
		},
		{
			name: "navigation to scalar value",
			data: map[string]any{
				"database": map[string]any{
					"host": "localhost",
				},
			},
			path:    []string{"database", "host"},
			want:    "localhost",
			wantErr: false,
		},
		{
			name: "empty path returns entire map",
			data: map[string]any{
				"host": "localhost",
				"port": 5432,
			},
			path: []string{},
			want: map[string]any{
				"host": "localhost",
				"port": 5432,
			},
			wantErr: false,
		},
		{
			name: "invalid path segment",
			data: map[string]any{
				"database": map[string]any{
					"host": "localhost",
				},
			},
			path:     []string{"nonexistent"},
			wantErr:  true,
			errMatch: "not found",
		},
		{
			name: "invalid nested path segment",
			data: map[string]any{
				"app": map[string]any{
					"server": map[string]any{
						"host": "localhost",
					},
				},
			},
			path:     []string{"app", "invalid", "config"},
			wantErr:  true,
			errMatch: "not found",
		},
		{
			name: "path through scalar (forgiving behavior)",
			data: map[string]any{
				"database": "simple-string",
			},
			path:    []string{"database"},
			want:    "simple-string",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := navigatePath(tt.data, tt.path)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errMatch != "" && !contains(err.Error(), tt.errMatch) {
					t.Errorf("error = %q, want substring %q", err.Error(), tt.errMatch)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Deep comparison for maps and values
			if !deepEqualany(got, tt.want) {
				t.Errorf("navigatePath() = %v, want %v", got, tt.want)
			}
		})
	}
}

// T046: Test ResolveReference with map reference returns only targeted map
func TestResolveReference_MapMode(t *testing.T) {
	t.Helper()

	tests := []struct {
		name         string
		ref          *ast.ReferenceExpr
		resourceData map[string]any
		wantKeys     []string
		wantErr      bool
	}{
		{
			name: "single level map reference",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"database"},
			},
			resourceData: map[string]any{
				"database": map[string]any{
					"host": "localhost",
					"port": 5432,
				},
				"other": map[string]any{
					"value": "should-not-be-included",
				},
			},
			wantKeys: []string{"host", "port"},
			wantErr:  false,
		},
		{
			name: "two-level map reference",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"app", "server"},
			},
			resourceData: map[string]any{
				"app": map[string]any{
					"server": map[string]any{
						"host":    "localhost",
						"port":    8080,
						"timeout": 30,
					},
					"version": "1.0",
				},
				"other": "excluded",
			},
			wantKeys: []string{"host", "port", "timeout"},
			wantErr:  false,
		},
		{
			name: "three-level map reference",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"infra", "network", "vpc"},
			},
			resourceData: map[string]any{
				"infra": map[string]any{
					"network": map[string]any{
						"vpc": map[string]any{
							"cidr":    "10.0.0.0/16",
							"subnets": 3,
							"region":  "us-west-2",
						},
						"load_balancer": "excluded",
					},
					"compute": "excluded",
				},
				"other": "excluded",
			},
			wantKeys: []string{"cidr", "subnets", "region"},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved, err := ResolveReference(tt.ref, tt.resourceData, &ResolutionContext{})

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify mode
			if resolved.Mode != MapMode {
				t.Errorf("Mode = %v, want MapMode", resolved.Mode)
			}

			if resolved.Entries == nil {
				t.Fatal("Entries should not be nil for MapMode")
			}

			// Verify only targeted map keys are present
			if len(resolved.Entries) != len(tt.wantKeys) {
				t.Errorf("Entries count = %d, want %d (got keys: %v, want: %v)",
					len(resolved.Entries), len(tt.wantKeys), mapKeys(resolved.Entries), tt.wantKeys)
			}

			for _, key := range tt.wantKeys {
				if _, exists := resolved.Entries[key]; !exists {
					t.Errorf("expected key %q in Entries, not found", key)
				}
			}

			// Verify Value is nil (not used for MapMode)
			if resolved.Value != nil {
				t.Error("Value should be nil for MapMode")
			}
		})
	}
}

// T047: Test map reference with invalid path segment returns error
func TestResolveReference_MapMode_InvalidPath(t *testing.T) {
	t.Helper()

	tests := []struct {
		name         string
		ref          *ast.ReferenceExpr
		resourceData map[string]any
		wantErr      bool
		errContains  string
	}{
		{
			name: "nonexistent single level path",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"nonexistent"},
			},
			resourceData: map[string]any{
				"database": map[string]any{
					"host": "localhost",
				},
				"api": "http://api.example.com",
			},
			wantErr:     true,
			errContains: "not found",
		},
		{
			name: "nonexistent nested path segment",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"app", "invalid", "config"},
			},
			resourceData: map[string]any{
				"app": map[string]any{
					"server": map[string]any{
						"host": "localhost",
					},
				},
			},
			wantErr:     true,
			errContains: "not found",
		},
		{
			name: "path segment at wrong level",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"database", "nonexistent", "deep"},
			},
			resourceData: map[string]any{
				"database": map[string]any{
					"host": "localhost",
					"port": 5432,
				},
			},
			wantErr:     true,
			errContains: "not found",
		},
		{
			name: "error message includes available keys",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"wrong"},
			},
			resourceData: map[string]any{
				"database": "value1",
				"api":      "value2",
				"cache":    "value3",
			},
			wantErr:     true,
			errContains: "available",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved, err := ResolveReference(tt.ref, tt.resourceData, &ResolutionContext{})

			if !tt.wantErr {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("expected error containing %q, got nil (resolved: %v)", tt.errContains, resolved)
			}

			if !contains(err.Error(), tt.errContains) {
				t.Errorf("error = %q, want substring %q", err.Error(), tt.errContains)
			}

			// Verify error is ErrPropertyPathInvalid
			if !contains(err.Error(), "property path invalid") &&
				!contains(err.Error(), "not found") {
				t.Errorf("expected error related to property path, got: %v", err)
			}
		})
	}
}

// T048: Test map reference followed by overrides applies deep merge
// Note: This test verifies the contract that map references produce Entries
// suitable for deep merge. The actual merge logic is tested in merge_test.go.
func TestResolveReference_MapMode_WithMerge(t *testing.T) {
	t.Helper()

	tests := []struct {
		name         string
		ref          *ast.ReferenceExpr
		resourceData map[string]any
		description  string
	}{
		{
			name: "map reference produces entries suitable for merge",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"database"},
			},
			resourceData: map[string]any{
				"database": map[string]any{
					"host": "localhost",
					"port": 5432,
					"pool": map[string]any{
						"min": 5,
						"max": 20,
					},
				},
			},
			description: "map reference returns Entries that can be merged with overrides",
		},
		{
			name: "nested map reference preserves structure",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"app", "server"},
			},
			resourceData: map[string]any{
				"app": map[string]any{
					"server": map[string]any{
						"host":    "localhost",
						"port":    8080,
						"timeout": 30,
						"tls": map[string]any{
							"enabled": true,
							"cert":    "/path/to/cert",
						},
					},
				},
			},
			description: "nested map reference maintains structure for deep merge",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved, err := ResolveReference(tt.ref, tt.resourceData, &ResolutionContext{})

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify mode is MapMode
			if resolved.Mode != MapMode {
				t.Errorf("Mode = %v, want MapMode", resolved.Mode)
			}

			// Verify Entries are populated
			if resolved.Entries == nil {
				t.Fatal("Entries should not be nil for MapMode")
			}

			// Verify Entries contain AST expressions
			for key, expr := range resolved.Entries {
				if expr == nil {
					t.Errorf("Entry[%q] is nil, expected AST expression", key)
				}

				// Verify AST expressions have correct types
				switch e := expr.(type) {
				case *ast.StringLiteral:
					// Simple values are converted to StringLiteral
				case *ast.MapExpr:
					// Nested maps are converted to MapExpr
					if e.Entries == nil {
						t.Errorf("MapExpr for key %q has nil Entries", key)
					}
				default:
					// Other types are acceptable (ListExpr, etc.)
				}
			}

			// Note: Actual merge behavior is tested in merge_test.go
			// This test only verifies that the structure is suitable for merging
		})
	}
}

// T061: Test DetermineReferenceMode returns PropertyMode for path resolving to scalar
func TestDetermineReferenceMode_PropertyMode(t *testing.T) {
	t.Helper()

	tests := []struct {
		name         string
		ref          *ast.ReferenceExpr
		resourceData map[string]any
		wantMode     ReferenceMode
	}{
		{
			name: "single level path to scalar string",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"version"},
			},
			resourceData: map[string]any{
				"version":  "1.0.0",
				"database": map[string]any{"host": "localhost"},
			},
			wantMode: PropertyMode,
		},
		{
			name: "two-level path to scalar string",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"database", "host"},
			},
			resourceData: map[string]any{
				"database": map[string]any{
					"host": "localhost",
					"port": 5432,
				},
			},
			wantMode: PropertyMode,
		},
		{
			name: "three-level path to scalar int",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"server", "http", "port"},
			},
			resourceData: map[string]any{
				"server": map[string]any{
					"http": map[string]any{
						"port":    8080,
						"timeout": 30,
					},
				},
			},
			wantMode: PropertyMode,
		},
		{
			name: "deeply nested path to scalar bool",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"app", "server", "tls", "enabled"},
			},
			resourceData: map[string]any{
				"app": map[string]any{
					"server": map[string]any{
						"tls": map[string]any{
							"enabled": true,
							"cert":    "/path/to/cert",
						},
					},
				},
			},
			wantMode: PropertyMode,
		},
		{
			name: "path to scalar float",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"database", "timeout"},
			},
			resourceData: map[string]any{
				"database": map[string]any{
					"host":    "localhost",
					"timeout": 30.5,
				},
			},
			wantMode: PropertyMode,
		},
		{
			name: "path to map should be MapMode not PropertyMode",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"database"},
			},
			resourceData: map[string]any{
				"database": map[string]any{
					"host": "localhost",
					"port": 5432,
				},
			},
			wantMode: MapMode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode := DetermineReferenceMode(tt.ref, tt.resourceData)

			if mode != tt.wantMode {
				t.Errorf("DetermineReferenceMode() = %v, want %v", mode, tt.wantMode)
			}
		})
	}
}

// T062: Test ResolveReference with property reference returns single value
func TestResolveReference_PropertyMode(t *testing.T) {
	t.Helper()

	tests := []struct {
		name         string
		ref          *ast.ReferenceExpr
		resourceData map[string]any
		wantValue    any
		wantErr      bool
	}{
		{
			name: "single level property - string",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"api_url"},
			},
			resourceData: map[string]any{
				"api_url":  "http://api.example.com",
				"database": "localhost",
			},
			wantValue: "http://api.example.com",
			wantErr:   false,
		},
		{
			name: "two-level property - string",
			ref: &ast.ReferenceExpr{
				Alias: "shared",
				Path:  []string{"database", "host"},
			},
			resourceData: map[string]any{
				"database": map[string]any{
					"host": "localhost",
					"port": 5432,
				},
			},
			wantValue: "localhost",
			wantErr:   false,
		},
		{
			name: "three-level property - int",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"server", "http", "port"},
			},
			resourceData: map[string]any{
				"server": map[string]any{
					"http": map[string]any{
						"port":    8080,
						"timeout": 30,
					},
				},
			},
			wantValue: 8080,
			wantErr:   false,
		},
		{
			name: "deeply nested property - bool",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"app", "server", "tls", "enabled"},
			},
			resourceData: map[string]any{
				"app": map[string]any{
					"server": map[string]any{
						"tls": map[string]any{
							"enabled": true,
							"cert":    "/path/to/cert",
						},
					},
				},
			},
			wantValue: true,
			wantErr:   false,
		},
		{
			name: "ultra-deep property (5 levels) - string",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"a", "b", "c", "d", "e"},
			},
			resourceData: map[string]any{
				"a": map[string]any{
					"b": map[string]any{
						"c": map[string]any{
							"d": map[string]any{
								"e": "deep-value",
							},
						},
					},
				},
			},
			wantValue: "deep-value",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved, err := ResolveReference(tt.ref, tt.resourceData, &ResolutionContext{})

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify mode is PropertyMode
			if resolved.Mode != PropertyMode {
				t.Errorf("Mode = %v, want PropertyMode", resolved.Mode)
			}

			// Verify Value field is populated (not Entries)
			if resolved.Value == nil {
				t.Fatal("Value should be populated for PropertyMode")
			}

			if resolved.Entries != nil {
				t.Error("Entries should be nil for PropertyMode")
			}

			// Verify Value is the correct AST expression
			// Convert expected value to string for comparison
			var gotValue string
			switch v := resolved.Value.(type) {
			case *ast.StringLiteral:
				gotValue = v.Value
			default:
				t.Fatalf("expected StringLiteral, got %T", resolved.Value)
			}

			wantStr := toString(tt.wantValue)
			if gotValue != wantStr {
				t.Errorf("Value = %q, want %q", gotValue, wantStr)
			}
		})
	}
}

// T063: Test property reference with non-existent path returns clear error
func TestResolveReference_PropertyMode_InvalidPath(t *testing.T) {
	t.Helper()

	tests := []struct {
		name         string
		ref          *ast.ReferenceExpr
		resourceData map[string]any
		wantErr      bool
		errContains  string
	}{
		{
			name: "nonexistent top-level property",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"nonexistent"},
			},
			resourceData: map[string]any{
				"database": "localhost",
				"api_url":  "http://api.example.com",
			},
			wantErr:     true,
			errContains: "not found",
		},
		{
			name: "nonexistent nested property",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"database", "nonexistent"},
			},
			resourceData: map[string]any{
				"database": map[string]any{
					"host": "localhost",
					"port": 5432,
				},
			},
			wantErr:     true,
			errContains: "not found",
		},
		{
			name: "deeply nested nonexistent property",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"app", "server", "tls", "invalid"},
			},
			resourceData: map[string]any{
				"app": map[string]any{
					"server": map[string]any{
						"tls": map[string]any{
							"enabled": true,
							"cert":    "/path/to/cert",
						},
					},
				},
			},
			wantErr:     true,
			errContains: "not found",
		},
		{
			name: "path segment at wrong level",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"database", "deep", "invalid"},
			},
			resourceData: map[string]any{
				"database": map[string]any{
					"host": "localhost",
				},
			},
			wantErr:     true,
			errContains: "not found",
		},
		{
			name: "error message includes path",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"api", "url"},
			},
			resourceData: map[string]any{
				"database": "localhost",
			},
			wantErr:     true,
			errContains: "api",
		},
		{
			name: "error message includes available keys",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"wrong"},
			},
			resourceData: map[string]any{
				"database": "value1",
				"api":      "value2",
				"cache":    "value3",
			},
			wantErr:     true,
			errContains: "available",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved, err := ResolveReference(tt.ref, tt.resourceData, &ResolutionContext{})

			if !tt.wantErr {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("expected error containing %q, got nil (resolved: %v)", tt.errContains, resolved)
			}

			if !contains(err.Error(), tt.errContains) {
				t.Errorf("error = %q, want substring %q", err.Error(), tt.errContains)
			}

			// Verify error is related to property path
			if !contains(err.Error(), "property path invalid") &&
				!contains(err.Error(), "not found") &&
				!contains(err.Error(), "failed to navigate") {
				t.Errorf("expected navigation/path error, got: %v", err)
			}
		})
	}
}

// T064: Test property reference with all scalar types (string, int, bool)
func TestResolveReference_PropertyMode_AllScalarTypes(t *testing.T) {
	t.Helper()

	tests := []struct {
		name         string
		ref          *ast.ReferenceExpr
		resourceData map[string]any
		wantValue    any
		wantType     string
	}{
		{
			name: "string property",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"message"},
			},
			resourceData: map[string]any{
				"message": "hello world",
			},
			wantValue: "hello world",
			wantType:  "string",
		},
		{
			name: "int property",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"count"},
			},
			resourceData: map[string]any{
				"count": 42,
			},
			wantValue: 42,
			wantType:  "int",
		},
		{
			name: "bool property - true",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"enabled"},
			},
			resourceData: map[string]any{
				"enabled": true,
			},
			wantValue: true,
			wantType:  "bool",
		},
		{
			name: "bool property - false",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"disabled"},
			},
			resourceData: map[string]any{
				"disabled": false,
			},
			wantValue: false,
			wantType:  "bool",
		},
		{
			name: "float property",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"timeout"},
			},
			resourceData: map[string]any{
				"timeout": 30.5,
			},
			wantValue: 30.5,
			wantType:  "float",
		},
		{
			name: "int64 property",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"large_number"},
			},
			resourceData: map[string]any{
				"large_number": int64(9223372036854775807),
			},
			wantValue: int64(9223372036854775807),
			wantType:  "int64",
		},
		{
			name: "nested string property",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"database", "host"},
			},
			resourceData: map[string]any{
				"database": map[string]any{
					"host": "prod.example.com",
					"port": 5432,
				},
			},
			wantValue: "prod.example.com",
			wantType:  "string",
		},
		{
			name: "nested int property",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"database", "port"},
			},
			resourceData: map[string]any{
				"database": map[string]any{
					"host": "localhost",
					"port": 5432,
				},
			},
			wantValue: 5432,
			wantType:  "int",
		},
		{
			name: "nested bool property",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"database", "ssl"},
			},
			resourceData: map[string]any{
				"database": map[string]any{
					"host": "localhost",
					"ssl":  true,
				},
			},
			wantValue: true,
			wantType:  "bool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved, err := ResolveReference(tt.ref, tt.resourceData, &ResolutionContext{})

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify mode
			if resolved.Mode != PropertyMode {
				t.Errorf("Mode = %v, want PropertyMode", resolved.Mode)
			}

			// Verify Value is populated
			if resolved.Value == nil {
				t.Fatal("Value should be populated for PropertyMode")
			}

			// Verify Entries is nil
			if resolved.Entries != nil {
				t.Error("Entries should be nil for PropertyMode")
			}

			// Verify Value is a StringLiteral (all scalars converted to string in AST)
			strLit, ok := resolved.Value.(*ast.StringLiteral)
			if !ok {
				t.Fatalf("expected StringLiteral, got %T", resolved.Value)
			}

			// Compare string representation
			wantStr := toString(tt.wantValue)
			if strLit.Value != wantStr {
				t.Errorf("Value = %q, want %q (type: %s)", strLit.Value, wantStr, tt.wantType)
			}
		})
	}
}

// --- Error Handling Tests (Phase 7: Tasks T080-T084) ---

// T080: Alias validation is tested in internal/resolver/resolver_test.go
// (removed from this file as it tests the wrong architectural layer)

// T082: Test invalid property path error (@base:config.invalid.path)
func TestResolveReference_InvalidPropertyPath(t *testing.T) {
	t.Helper()

	tests := []struct {
		name         string
		ref          *ast.ReferenceExpr
		resourceData map[string]any
		wantErr      bool
		errContains  []string
	}{
		{
			name: "nonexistent top-level property",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"nonexistent"},
				SourceSpan: ast.SourceSpan{
					Filename:  "test.csl",
					StartLine: 10,
				},
			},
			resourceData: map[string]any{
				"database": "localhost",
				"port":     5432,
			},
			wantErr:     true,
			errContains: []string{"nonexistent", "not found", "available"},
		},
		{
			name: "invalid nested path segment",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"database", "invalid", "deep"},
				SourceSpan: ast.SourceSpan{
					Filename:  "app.csl",
					StartLine: 25,
				},
			},
			resourceData: map[string]any{
				"database": map[string]any{
					"host": "localhost",
					"port": 5432,
				},
			},
			wantErr:     true,
			errContains: []string{"invalid", "not found"},
		},
		{
			name: "path through scalar value",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"version", "major"},
				SourceSpan: ast.SourceSpan{
					Filename:  "config.csl",
					StartLine: 8,
				},
			},
			resourceData: map[string]any{
				"version": "1.2.3", // scalar, not a map
			},
			wantErr:     true,
			errContains: []string{"major", "not a map"},
		},
		{
			name: "error message shows available keys",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"wrong"},
			},
			resourceData: map[string]any{
				"database": "value1",
				"api":      "value2",
				"cache":    "value3",
			},
			wantErr:     true,
			errContains: []string{"wrong", "available", "database", "api", "cache"},
		},
		{
			name: "deeply nested invalid path",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"app", "server", "tls", "invalid"},
			},
			resourceData: map[string]any{
				"app": map[string]any{
					"server": map[string]any{
						"tls": map[string]any{
							"enabled": true,
							"cert":    "/path/to/cert",
						},
					},
				},
			},
			wantErr:     true,
			errContains: []string{"invalid", "not found", "enabled", "cert"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resCtx := &ResolutionContext{Stack: []PathRef{}}

			_, err := ResolveReference(tt.ref, tt.resourceData, resCtx)

			if !tt.wantErr {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}

			if err == nil {
				t.Fatal("expected error for invalid property path, got nil")
			}

			// Verify error contains expected substrings
			errMsg := err.Error()
			for _, substr := range tt.errContains {
				if !contains(errMsg, substr) {
					t.Errorf("error message %q should contain %q", errMsg, substr)
				}
			}

			// Verify error is related to property path validation
			if !contains(errMsg, "not found") && !contains(errMsg, "not a map") {
				t.Errorf("expected path validation error, got: %v", err)
			}
		})
	}
}

// T083: Provider communication errors are tested in internal/resolver/resolver_test.go
// (removed from this file as they test the wrong architectural layer)

// T084: Test empty resource returns empty map (not error)
func TestResolveReference_EmptyResource(t *testing.T) {
	t.Helper()

	tests := []struct {
		name         string
		ref          *ast.ReferenceExpr
		resourceData map[string]any
		wantMode     ReferenceMode
		wantErr      bool
	}{
		{
			name: "root reference on empty resource",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{}, // Root reference
				SourceSpan: ast.SourceSpan{
					Filename:  "test.csl",
					StartLine: 10,
				},
			},
			resourceData: map[string]any{}, // Empty resource
			wantMode:     RootMode,
			wantErr:      false,
		},
		{
			name: "map reference on empty nested map",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{"empty_section"},
			},
			resourceData: map[string]any{
				"empty_section": map[string]any{}, // Empty nested map
				"other":         "value",
			},
			wantMode: MapMode,
			wantErr:  false,
		},
		{
			name: "empty resource with metadata still valid",
			ref: &ast.ReferenceExpr{
				Alias: "base",
				Path:  []string{},
			},
			resourceData: map[string]any{}, // Empty but valid
			wantMode:     RootMode,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resCtx := &ResolutionContext{Stack: []PathRef{}}

			resolved, err := ResolveReference(tt.ref, tt.resourceData, resCtx)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify mode
			if resolved.Mode != tt.wantMode {
				t.Errorf("Mode = %v, want %v", resolved.Mode, tt.wantMode)
			}

			// Verify empty map is returned (not error)
			switch resolved.Mode {
			case RootMode, MapMode:
				if resolved.Entries == nil {
					t.Error("Entries should not be nil for empty resource")
				}
				if len(resolved.Entries) != 0 {
					t.Errorf("expected empty Entries, got %d entries", len(resolved.Entries))
				}
			case PropertyMode:
				t.Error("empty resource should not resolve to PropertyMode")
			}
		})
	}
}

// Helper functions

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func deepEqualany(a, b any) bool {
	// Simple deep comparison for maps and scalars
	switch va := a.(type) {
	case map[string]any:
		vb, ok := b.(map[string]any)
		if !ok {
			return false
		}
		if len(va) != len(vb) {
			return false
		}
		for k, v := range va {
			if !deepEqualany(v, vb[k]) {
				return false
			}
		}
		return true
	default:
		return a == b
	}
}

func mapKeys(m map[string]ast.Expr) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func toString(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case int:
		return fmt.Sprintf("%d", val)
	case int64:
		return fmt.Sprintf("%d", val)
	case float64:
		return fmt.Sprintf("%g", val)
	case bool:
		return fmt.Sprintf("%t", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}
