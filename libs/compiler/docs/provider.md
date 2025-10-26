# Provider Interface and Implementation Guide

This document describes how to implement providers for the Nomos compiler. Providers are pluggable adapters that fetch external configuration data from various sources (filesystem, cloud services, secret managers, etc.).

## Overview

The compiler uses providers to resolve **inline references** in Nomos configuration files. When the compiler encounters a reference expression like `@file["config/database.yml"]`, it delegates the resolution to the appropriate provider.

Providers are:
- **Initialized once** per compilation run when first used
- **Cached** per compilation run (providers and their fetch results are reused)
- **Pluggable** via the ProviderRegistry

## Provider Interface

Providers must implement the `compiler.Provider` interface:

```go
type Provider interface {
    // Init initializes the provider with the given options.
    // Called once per compilation run when the provider is first used.
    // Must be called before Fetch.
    Init(ctx context.Context, opts ProviderInitOptions) error

    // Fetch retrieves data from the provider at the specified path.
    // The path is a sequence of components (e.g., ["config", "network", "vpc"]).
    // Returns the resolved value or an error if the fetch fails.
    //
    // Fetch results are cached per compilation run. Subsequent calls with
    // the same path return the cached value without re-fetching.
    Fetch(ctx context.Context, path []string) (any, error)
}
```

### Optional: Metadata Interface

Providers can optionally implement `ProviderWithInfo` to expose metadata for provenance tracking:

```go
type ProviderWithInfo interface {
    Provider
    // Info returns the provider's alias and version for metadata tracking.
    Info() (alias string, version string)
}
```

## ProviderInitOptions

The `ProviderInitOptions` struct is passed to `Init` and contains:

```go
type ProviderInitOptions struct {
    // Alias is the provider's registered alias in the ProviderRegistry.
    Alias string

    // Config contains provider-specific configuration (from source declarations).
    Config map[string]any
}
```

## Implementation Pattern

### 1. Define Your Provider Struct

```go
package myprovider

import (
    "context"
    "fmt"

    "github.com/autonomous-bits/nomos/libs/compiler"
)

type MyProvider struct {
    alias   string
    baseURL string
    client  *http.Client
}
```

### 2. Implement Init

```go
func (p *MyProvider) Init(ctx context.Context, opts compiler.ProviderInitOptions) error {
    p.alias = opts.Alias

    // Extract provider-specific config
    if baseURL, ok := opts.Config["base_url"].(string); ok {
        p.baseURL = baseURL
    } else {
        return fmt.Errorf("missing required config: base_url")
    }

    // Initialize resources (connections, clients, etc.)
    p.client = &http.Client{
        Timeout: 10 * time.Second,
    }

    return nil
}
```

### 3. Implement Fetch

```go
func (p *MyProvider) Fetch(ctx context.Context, path []string) (any, error) {
    // Convert path to your provider's addressing scheme
    url := fmt.Sprintf("%s/%s", p.baseURL, strings.Join(path, "/"))

    // Fetch data from your source
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }

    resp, err := p.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch from %s: %w", url, err)
    }
    defer resp.Body.Close()

    // Parse and return data
    var data any
    if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
        return nil, fmt.Errorf("failed to decode response: %w", err)
    }

    return data, nil
}
```

### 4. (Optional) Implement Info

```go
func (p *MyProvider) Info() (alias string, version string) {
    return p.alias, "v1.0.0"
}
```

## Registration with ProviderRegistry

Providers are registered with the compiler via a `ProviderRegistry`. The registry manages provider lifecycle:

1. Register a **constructor function** for each provider alias
2. Providers are **instantiated on demand** when first accessed
3. Provider instances are **cached** for the compilation run

### Example: Registering a Provider

```go
package main

import (
    "context"
    "fmt"

    "github.com/autonomous-bits/nomos/libs/compiler"
    "mypkg/myprovider"
)

func main() {
    // Create a provider registry
    registry := compiler.NewProviderRegistry()

    // Register provider constructor
    registry.Register("myalias", func(opts compiler.ProviderInitOptions) (compiler.Provider, error) {
        provider := &myprovider.MyProvider{}
        return provider, nil
    })

    // Compile with the registry
    options := compiler.Options{
        Path:             "./configs",
        ProviderRegistry: registry,
    }

    snapshot, err := compiler.Compile(context.Background(), options)
    if err != nil {
        fmt.Printf("Compilation failed: %v\n", err)
        return
    }

    fmt.Printf("Compiled successfully: %+v\n", snapshot)
}
```

### Provider Constructor Pattern

The constructor function receives `ProviderInitOptions` and must:
1. Create the provider instance
2. Return the provider (initialization happens when `Init` is called by the registry)

```go
registry.Register("config", func(opts compiler.ProviderInitOptions) (compiler.Provider, error) {
    // Create provider
    provider := &ConfigProvider{
        baseDir: "/etc/app/config",
    }

    // Return provider (Init will be called by the registry)
    return provider, nil
})
```

## Provider Lifecycle

```
┌─────────────────────────────────────────────┐
│ 1. Register provider constructor            │
│    registry.Register("alias", constructor)  │
└──────────────────┬──────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────┐
│ 2. GetProvider called (first time)          │
│    - Constructor called                     │
│    - Init called with ProviderInitOptions   │
│    - Instance cached                        │
└──────────────────┬──────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────┐
│ 3. Fetch called (as needed)                 │
│    - Path resolved via provider logic       │
│    - Result returned (compiler may cache)   │
└──────────────────┬──────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────┐
│ 4. GetProvider called (subsequent times)    │
│    - Cached instance returned               │
│    - Init NOT called again                  │
└─────────────────────────────────────────────┘
```

### Key Points

1. **Init is called exactly once** per provider alias per compilation run
2. **GetProvider returns the same instance** when called multiple times for the same alias
3. **Fetch may be called multiple times** for different paths
4. **Providers should be concurrency-safe** if used in parallel compilation

## Testing Providers

### Using FakeProvider

The compiler provides a `FakeProvider` test double in `libs/compiler/test/fakes`:

```go
import (
    "context"
    "testing"

    "github.com/autonomous-bits/nomos/libs/compiler"
    "github.com/autonomous-bits/nomos/libs/compiler/test/fakes"
)

func TestMyCompilerIntegration(t *testing.T) {
    // Create fake provider
    fake := fakes.NewFakeProvider("config")

    // Configure responses
    fake.FetchResponses["database/host"] = "db.example.com"
    fake.FetchResponses["database/port"] = "5432"

    // Register with registry
    registry := compiler.NewProviderRegistry()
    registry.Register("config", func(opts compiler.ProviderInitOptions) (compiler.Provider, error) {
        return fake, nil
    })

    // Use in compilation
    options := compiler.Options{
        Path:             "./testdata/config.csl",
        ProviderRegistry: registry,
    }

    snapshot, err := compiler.Compile(context.Background(), options)
    if err != nil {
        t.Fatalf("compilation failed: %v", err)
    }

    // Verify provider was called
    if fake.InitCount != 1 {
        t.Errorf("expected Init called once, got %d", fake.InitCount)
    }

    if fake.FetchCount < 1 {
        t.Errorf("expected at least one Fetch call, got %d", fake.FetchCount)
    }
}
```

### FakeProvider Features

- **Configurable responses**: Set `FetchResponses` map for path-to-value mappings
- **Error injection**: Set `InitError` or `FetchError` to test error handling
- **Call tracking**: Inspect `InitCount`, `FetchCount`, and `FetchCalls`
- **Thread-safe**: Safe for concurrent use
- **Reset**: Call `Reset()` to clear counts between tests

See `libs/compiler/test/fakes/fake_provider_test.go` for comprehensive examples.

## Best Practices

### Do's ✅

1. **Accept context**: Always use `ctx` for cancellation and timeouts
2. **Wrap errors**: Use `fmt.Errorf("context: %w", err)` to add context
3. **Validate config**: Check required configuration in `Init`
4. **Handle resources**: Close connections/files properly (consider implementing cleanup)
5. **Be deterministic**: Same path should return same result (for a given state)
6. **Document behavior**: Clear docs on path format and data structure

### Don'ts ❌

1. **Don't panic**: Return errors instead
2. **Don't ignore context**: Check `ctx.Done()` for cancellation
3. **Don't log secrets**: Avoid logging sensitive data
4. **Don't store state unsafely**: Use proper synchronization if caching internally
5. **Don't make assumptions**: Validate all inputs from `opts.Config`

## Example: File Provider

See `libs/compiler/providers/file` for a complete reference implementation that:
- Reads files from the filesystem
- Resolves paths relative to a base directory
- Handles YAML/JSON parsing
- Implements proper error handling

## Error Handling

Providers should return clear, actionable errors:

```go
func (p *MyProvider) Fetch(ctx context.Context, path []string) (any, error) {
    // Check context
    if err := ctx.Err(); err != nil {
        return nil, err
    }

    // Validate input
    if len(path) == 0 {
        return nil, fmt.Errorf("path cannot be empty")
    }

    // Fetch with context
    data, err := p.fetchFromSource(ctx, path)
    if err != nil {
        // Wrap error with context
        return nil, fmt.Errorf("failed to fetch %v from %s: %w",
            path, p.alias, err)
    }

    return data, nil
}
```

### Sentinel Errors

Use the compiler's sentinel errors where applicable:

```go
import "github.com/autonomous-bits/nomos/libs/compiler"

// When a reference cannot be resolved
return nil, fmt.Errorf("%w: path %v not found", compiler.ErrUnresolvedReference, path)

// When a provider is not registered
return nil, fmt.Errorf("%w: %s", compiler.ErrProviderNotRegistered, alias)
```

## Advanced Topics

### Provider Type Registry

For dynamic provider creation from source declarations, use `ProviderTypeRegistry`:

```go
typeRegistry := compiler.NewProviderTypeRegistry()

typeRegistry.RegisterType("http", func(config map[string]any) (compiler.Provider, error) {
    baseURL := config["base_url"].(string)
    return &HTTPProvider{baseURL: baseURL}, nil
})

// Create provider from config
provider, err := typeRegistry.CreateProvider("http", map[string]any{
    "base_url": "https://api.example.com",
})
```

### Per-Provider Caching

While the compiler may cache Fetch results, providers can implement their own caching:

```go
type CachingProvider struct {
    cache map[string]any
    mu    sync.RWMutex
}

func (p *CachingProvider) Fetch(ctx context.Context, path []string) (any, error) {
    key := strings.Join(path, "/")

    // Check cache
    p.mu.RLock()
    if val, ok := p.cache[key]; ok {
        p.mu.RUnlock()
        return val, nil
    }
    p.mu.RUnlock()

    // Fetch and cache
    val, err := p.doFetch(ctx, path)
    if err != nil {
        return nil, err
    }

    p.mu.Lock()
    p.cache[key] = val
    p.mu.Unlock()

    return val, nil
}
```

## Summary

1. **Implement** the `Provider` interface (`Init` and `Fetch`)
2. **Register** your provider constructor with `ProviderRegistry`
3. **Test** using `FakeProvider` or custom test doubles
4. **Follow** best practices for error handling and context usage
5. **Document** your provider's path format and configuration

For more examples, see:
- `libs/compiler/providers/file` — File system provider
- `libs/compiler/test/fakes` — Fake provider for testing
- Integration tests in `libs/compiler/*_integration_test.go`
