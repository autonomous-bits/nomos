package imports

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/autonomous-bits/nomos/libs/compiler/internal/core"
	"github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
)

// mockProvider for testing
type mockProvider struct {
	initCalled  bool
	initErr     error
	fetchResult any
	fetchErr    error
	fetchCalls  [][]string
}

func (m *mockProvider) Init(_ context.Context, _ core.ProviderInitOptions) error {
	m.initCalled = true
	return m.initErr
}

func (m *mockProvider) Fetch(_ context.Context, path []string) (any, error) {
	m.fetchCalls = append(m.fetchCalls, path)
	if m.fetchErr != nil {
		return nil, m.fetchErr
	}
	return m.fetchResult, nil
}

// mockRegistry for testing
type mockRegistry struct {
	providers    map[string]core.Provider
	constructors map[string]core.ProviderConstructor
	getErr       error
}

func newMockRegistry() *mockRegistry {
	return &mockRegistry{
		providers:    make(map[string]core.Provider),
		constructors: make(map[string]core.ProviderConstructor),
	}
}

func (r *mockRegistry) Register(alias string, constructor core.ProviderConstructor) {
	r.constructors[alias] = constructor
}

func (r *mockRegistry) GetProvider(_ context.Context, alias string) (core.Provider, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	if p, ok := r.providers[alias]; ok {
		return p, nil
	}
	if c, ok := r.constructors[alias]; ok {
		provider, err := c(core.ProviderInitOptions{})
		if err != nil {
			return nil, err
		}
		r.providers[alias] = provider
		return provider, nil
	}
	return nil, errors.New("provider not found: " + alias)
}

func (r *mockRegistry) RegisteredAliases() []string {
	aliases := make([]string, 0, len(r.constructors)+len(r.providers))
	for alias := range r.constructors {
		aliases = append(aliases, alias)
	}
	for alias := range r.providers {
		if _, ok := r.constructors[alias]; !ok {
			aliases = append(aliases, alias)
		}
	}
	return aliases
}

// mockTypeRegistry for testing
type mockTypeRegistry struct {
	providers map[string]core.Provider // Store providers directly in mock
	types     map[string]func(map[string]any) (core.Provider, error)
}

func newMockTypeRegistry() *mockTypeRegistry {
	return &mockTypeRegistry{
		providers: make(map[string]core.Provider),
		types:     make(map[string]func(map[string]any) (core.Provider, error)),
	}
}

func (r *mockTypeRegistry) RegisterType(_ string, _ core.ProviderTypeConstructor) {
	// Not used in tests
}

func (r *mockTypeRegistry) CreateProvider(_ context.Context, typeName string, _ string, config map[string]any) (core.Provider, error) {
	// Check if a provider instance is registered first (test shortcut)
	if provider, ok := r.providers[typeName]; ok {
		return provider, nil
	}
	// Otherwise use constructor function
	if fn, ok := r.types[typeName]; ok {
		return fn(config)
	}
	return nil, errors.New("type not found: " + typeName)
}

func (r *mockTypeRegistry) IsTypeRegistered(typeName string) bool {
	if _, ok := r.providers[typeName]; ok {
		return true
	}
	_, ok := r.types[typeName]
	return ok
}

// RegisterProvider is a test helper to register a provider instance directly
func (r *mockTypeRegistry) RegisterProvider(typeName string, provider core.Provider) {
	r.providers[typeName] = provider
}

func (r *mockTypeRegistry) RegisteredTypes() []string {
	types := make([]string, 0, len(r.providers)+len(r.types))
	for t := range r.providers {
		types = append(types, t)
	}
	for t := range r.types {
		if _, exists := r.providers[t]; !exists {
			types = append(types, t)
		}
	}
	return types
}

func TestResolveImports_SimpleFile(t *testing.T) {
	// Create a test file
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "config.csl")
	content := `app:
  name: "myapp"`
	if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Setup
	registry := newMockRegistry()
	typeRegistry := newMockTypeRegistry()

	// Act
	result, err := ResolveImports(context.Background(), filePath, registry, typeRegistry)

	// Assert
	if err != nil {
		t.Fatalf("ResolveImports failed: %v", err)
	}

	if len(result) == 0 {
		t.Fatal("expected non-empty result")
	}

	app, ok := result["app"].(map[string]any)
	if !ok {
		t.Fatalf("expected app to be map, got %T", result["app"])
	}
	if app["name"] != "myapp" {
		t.Errorf("expected name 'myapp', got %v", app["name"])
	}
}

func TestResolveImports_WithSource(t *testing.T) {
	// Create a test file with source declaration
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "config.csl")
	content := `source:
  alias: "configs"
  type: "file"
  directory: "/tmp"

app:
  name: "myapp"`
	if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Setup
	registry := newMockRegistry()
	typeRegistry := newMockTypeRegistry()

	mockProvider := &mockProvider{}
	typeRegistry.RegisterProvider("file", mockProvider)

	// Act
	result, err := ResolveImports(context.Background(), filePath, registry, typeRegistry)

	// Assert
	if err != nil {
		t.Fatalf("ResolveImports failed: %v", err)
	}

	if !mockProvider.initCalled {
		t.Error("expected provider Init to be called")
	}

	app, ok := result["app"].(map[string]any)
	if !ok {
		t.Fatalf("expected app to be map, got %T", result["app"])
	}
	if app["name"] != "myapp" {
		t.Errorf("expected name 'myapp', got %v", app["name"])
	}
}

func TestResolveImports_WithImport(t *testing.T) {
	// Create a test file with import statement
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "config.csl")
	content := `source:
  alias: "configs"
  type: "file"
  directory: "/tmp"

import:configs:base.csl

app:
  name: "myapp"`
	if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Setup
	registry := newMockRegistry()
	typeRegistry := newMockTypeRegistry()

	mockProvider := &mockProvider{
		fetchResult: map[string]any{
			"database": map[string]any{
				"host": "localhost",
			},
		},
	}
	typeRegistry.RegisterProvider("file", mockProvider)

	// Act
	result, err := ResolveImports(context.Background(), filePath, registry, typeRegistry)

	// Assert
	if err != nil {
		t.Fatalf("ResolveImports failed: %v", err)
	}

	// Check imported data
	database, ok := result["database"].(map[string]any)
	if !ok {
		t.Fatalf("expected database to be map, got %T", result["database"])
	}
	if database["host"] != "localhost" {
		t.Errorf("expected host 'localhost', got %v", database["host"])
	}

	// Check main file data
	app, ok := result["app"].(map[string]any)
	if !ok {
		t.Fatalf("expected app to be map, got %T", result["app"])
	}
	if app["name"] != "myapp" {
		t.Errorf("expected name 'myapp', got %v", app["name"])
	}
}

func TestResolveImports_ParseError(t *testing.T) {
	// Create a file with syntax error
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "bad.csl")
	content := `app
  name: "incomplete"`
	if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Setup
	registry := newMockRegistry()
	typeRegistry := newMockTypeRegistry()

	// Act
	_, err := ResolveImports(context.Background(), filePath, registry, typeRegistry)

	// Assert
	if err == nil {
		t.Fatal("expected error for unparsable file")
	}
}

func TestInitializeProvider_Success(t *testing.T) {
	// Setup
	src := SourceDecl{
		Alias:  "test",
		Type:   "file",
		Config: map[string]any{"key": "value"},
	}

	registry := newMockRegistry()
	typeRegistry := newMockTypeRegistry()

	mockProvider := &mockProvider{}
	typeRegistry.RegisterProvider("file", mockProvider)

	// Act
	err := initializeProvider(context.Background(), src, "/test/source.csl", registry, typeRegistry)

	// Assert
	if err != nil {
		t.Fatalf("initializeProvider failed: %v", err)
	}

	if !mockProvider.initCalled {
		t.Error("expected Init to be called")
	}

	// Verify provider was registered
	if len(registry.constructors) == 0 {
		t.Error("expected provider to be registered")
	}
}

func TestInitializeProvider_AlreadyExists(t *testing.T) {
	// Setup: pre-existing provider
	registry := newMockRegistry()
	existingProvider := &mockProvider{}
	registry.providers["existing"] = existingProvider

	typeRegistry := newMockTypeRegistry()
	createCallCount := 0
	var createErr error // Could be set to test error path
	// Use a closure that tracks calls but still returns provider
	typeRegistry.types["file"] = func(_ map[string]any) (core.Provider, error) {
		createCallCount++
		if createErr != nil {
			return nil, createErr
		}
		return &mockProvider{}, nil
	}

	src := SourceDecl{
		Alias:  "existing",
		Type:   "file",
		Config: map[string]any{},
	}

	// Act
	err := initializeProvider(context.Background(), src, "/test/source.csl", registry, typeRegistry)

	// Assert
	if err != nil {
		t.Fatalf("initializeProvider failed: %v", err)
	}

	if createCallCount != 0 {
		t.Errorf("expected CreateProvider not to be called, but it was called %d times", createCallCount)
	}
}

func TestInitializeProvider_NoTypeRegistry(t *testing.T) {
	// Setup
	src := SourceDecl{
		Alias:  "test",
		Type:   "file",
		Config: map[string]any{},
	}

	registry := newMockRegistry()

	// Act
	err := initializeProvider(context.Background(), src, "/test/source.csl", registry, nil)

	// Assert
	if err == nil {
		t.Fatal("expected error for nil type registry")
	}
	if err.Error()[:len("cannot create provider")] != "cannot create provider" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestInitializeProvider_CreateError(t *testing.T) {
	// Setup
	src := SourceDecl{
		Alias:  "test",
		Type:   "badtype",
		Config: map[string]any{},
	}

	registry := newMockRegistry()
	typeRegistry := newMockTypeRegistry()
	// Don't register the type, causing CreateProvider to fail

	// Act
	err := initializeProvider(context.Background(), src, "/test/source.csl", registry, typeRegistry)

	// Assert
	if err == nil {
		t.Fatal("expected error for unknown type")
	}
}

func TestInitializeProvider_InitError(t *testing.T) {
	// Setup
	src := SourceDecl{
		Alias:  "test",
		Type:   "file",
		Config: map[string]any{},
	}

	registry := newMockRegistry()
	typeRegistry := newMockTypeRegistry()

	initErr := errors.New("init failed")
	typeRegistry.RegisterProvider("file", &mockProvider{initErr: initErr})

	// Act
	err := initializeProvider(context.Background(), src, "/test/source.csl", registry, typeRegistry)

	// Assert
	if err == nil {
		t.Fatal("expected error from Init failure")
	}
	if !errors.Is(err, initErr) && err.Error()[:len("failed to initialize provider")] != "failed to initialize provider" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestResolveImport_Success(t *testing.T) {
	// Setup
	imp := ImportDecl{
		Alias: "configs",
		Path:  []string{"base.csl"},
	}

	provider := &mockProvider{
		fetchResult: map[string]any{
			"server": map[string]any{
				"port": 8080,
			},
		},
	}

	registry := newMockRegistry()
	registry.providers["configs"] = provider

	// Act
	result, err := resolveImport(context.Background(), imp, registry)

	// Assert
	if err != nil {
		t.Fatalf("resolveImport failed: %v", err)
	}

	server, ok := result["server"].(map[string]any)
	if !ok {
		t.Fatalf("expected server to be map, got %T", result["server"])
	}
	if server["port"] != 8080 {
		t.Errorf("expected port 8080, got %v", server["port"])
	}

	if len(provider.fetchCalls) != 1 {
		t.Errorf("expected 1 fetch call, got %d", len(provider.fetchCalls))
	}
}

func TestResolveImport_ProviderNotFound(t *testing.T) {
	// Setup
	imp := ImportDecl{
		Alias: "nonexistent",
		Path:  []string{"file.csl"},
	}

	registry := newMockRegistry()

	// Act
	_, err := resolveImport(context.Background(), imp, registry)

	// Assert
	if err == nil {
		t.Fatal("expected error for nonexistent provider")
	}
	if err.Error()[:len("provider")] != "provider" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestResolveImport_EmptyPath(t *testing.T) {
	// Setup
	imp := ImportDecl{
		Alias: "configs",
		Path:  []string{},
	}

	provider := &mockProvider{}
	registry := newMockRegistry()
	registry.providers["configs"] = provider

	// Act
	_, err := resolveImport(context.Background(), imp, registry)

	// Assert
	if err == nil {
		t.Fatal("expected error for empty path")
	}
	if err.Error()[:len("import")] != "import" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestResolveImport_FetchError(t *testing.T) {
	// Setup
	imp := ImportDecl{
		Alias: "configs",
		Path:  []string{"missing.csl"},
	}

	fetchErr := errors.New("file not found")
	provider := &mockProvider{
		fetchErr: fetchErr,
	}

	registry := newMockRegistry()
	registry.providers["configs"] = provider

	// Act
	_, err := resolveImport(context.Background(), imp, registry)

	// Assert
	if err == nil {
		t.Fatal("expected error from fetch failure")
	}
}

func TestResolveImport_NonMapResult(t *testing.T) {
	// Setup
	imp := ImportDecl{
		Alias: "configs",
		Path:  []string{"data.csl"},
	}

	// Return a non-map result
	provider := &mockProvider{
		fetchResult: "not a map",
	}

	registry := newMockRegistry()
	registry.providers["configs"] = provider

	// Act
	_, err := resolveImport(context.Background(), imp, registry)

	// Assert
	if err == nil {
		t.Fatal("expected error for non-map result")
	}
	if err.Error()[:len("import")] != "import" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestDeepMerge_SimpleOverride(t *testing.T) {
	// Setup
	base := map[string]any{
		"key1": "value1",
		"key2": "value2",
	}

	override := map[string]any{
		"key2": "newvalue2",
		"key3": "value3",
	}

	// Act
	result := deepMerge(base, override)

	// Assert
	if result["key1"] != "value1" {
		t.Errorf("expected key1='value1', got %v", result["key1"])
	}
	if result["key2"] != "newvalue2" {
		t.Errorf("expected key2='newvalue2', got %v", result["key2"])
	}
	if result["key3"] != "value3" {
		t.Errorf("expected key3='value3', got %v", result["key3"])
	}
}

func TestDeepMerge_NestedMaps(t *testing.T) {
	// Setup
	base := map[string]any{
		"server": map[string]any{
			"host": "localhost",
			"port": 8080,
		},
	}

	override := map[string]any{
		"server": map[string]any{
			"port": 9000,
			"ssl":  true,
		},
	}

	// Act
	result := deepMerge(base, override)

	// Assert
	server, ok := result["server"].(map[string]any)
	if !ok {
		t.Fatalf("expected server to be map, got %T", result["server"])
	}

	if server["host"] != "localhost" {
		t.Errorf("expected host='localhost', got %v", server["host"])
	}
	if server["port"] != 9000 {
		t.Errorf("expected port=9000, got %v", server["port"])
	}
	if server["ssl"] != true {
		t.Errorf("expected ssl=true, got %v", server["ssl"])
	}
}

func TestDeepMerge_ArrayOverride(t *testing.T) {
	// Setup: arrays use last-wins, not element-wise merge
	base := map[string]any{
		"tags": []string{"a", "b"},
	}

	override := map[string]any{
		"tags": []string{"c", "d"},
	}

	// Act
	result := deepMerge(base, override)

	// Assert
	tags, ok := result["tags"].([]string)
	if !ok {
		t.Fatalf("expected tags to be []string, got %T", result["tags"])
	}

	if len(tags) != 2 || tags[0] != "c" || tags[1] != "d" {
		t.Errorf("expected tags to be replaced with ['c', 'd'], got %v", tags)
	}
}

func TestDeepMerge_TypeMismatch(t *testing.T) {
	// Setup: when types don't match, use last-wins
	base := map[string]any{
		"value": map[string]any{"nested": true},
	}

	override := map[string]any{
		"value": "scalar",
	}

	// Act
	result := deepMerge(base, override)

	// Assert
	if result["value"] != "scalar" {
		t.Errorf("expected value='scalar', got %v", result["value"])
	}
}

func TestDeepMerge_EmptyBase(t *testing.T) {
	// Setup
	base := map[string]any{}
	override := map[string]any{
		"key": "value",
	}

	// Act
	result := deepMerge(base, override)

	// Assert
	if result["key"] != "value" {
		t.Errorf("expected key='value', got %v", result["key"])
	}
}

func TestDeepMerge_EmptyOverride(t *testing.T) {
	// Setup
	base := map[string]any{
		"key": "value",
	}
	override := map[string]any{}

	// Act
	result := deepMerge(base, override)

	// Assert
	if result["key"] != "value" {
		t.Errorf("expected key='value', got %v", result["key"])
	}
}

func TestExprToValue_StringLiteral(t *testing.T) {
	expr := &ast.StringLiteral{Value: "test"}
	result := exprToValue(expr)

	if result != "test" {
		t.Errorf("expected 'test', got %v", result)
	}
}

func TestExprToValue_ReferenceExpr(t *testing.T) {
	expr := &ast.ReferenceExpr{
		Alias: "config",
		Path:  []string{"db", "host"},
	}

	result := exprToValue(expr)

	ref, ok := result.(*ast.ReferenceExpr)
	if !ok {
		t.Fatalf("expected *ast.ReferenceExpr, got %T", result)
	}

	if ref.Alias != "config" {
		t.Errorf("expected alias 'config', got %s", ref.Alias)
	}
}

func TestExprToValue_UnsupportedType(t *testing.T) {
	result := exprToValue(nil)

	if result != nil {
		t.Errorf("expected nil for unsupported type, got %v", result)
	}
}

func TestAlreadyInitializedProvider_Init(t *testing.T) {
	// Setup
	innerProvider := &mockProvider{}
	wrapper := &alreadyInitializedProvider{
		provider: innerProvider,
	}

	// Act
	err := wrapper.Init(context.Background(), core.ProviderInitOptions{})

	// Assert
	if err != nil {
		t.Errorf("Init should be no-op, got error: %v", err)
	}

	if innerProvider.initCalled {
		t.Error("expected inner provider Init not to be called")
	}
}

func TestAlreadyInitializedProvider_Fetch(t *testing.T) {
	// Setup
	expectedResult := map[string]any{"key": "value"}
	innerProvider := &mockProvider{
		fetchResult: expectedResult,
	}
	wrapper := &alreadyInitializedProvider{
		provider: innerProvider,
	}

	// Act
	result, err := wrapper.Fetch(context.Background(), []string{"test"})

	// Assert
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok || resultMap["key"] != "value" {
		t.Errorf("expected result to be passed through")
	}

	if len(innerProvider.fetchCalls) != 1 {
		t.Errorf("expected Fetch to be called on inner provider")
	}
}
