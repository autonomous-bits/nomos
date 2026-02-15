// Package resolver implements reference resolution for the Nomos compiler.
//
// The resolver walks AST values, identifies ReferenceExpr nodes, and resolves
// them by calling appropriate providers. Results are cached per compilation run
// to avoid redundant fetches.
package resolver

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/autonomous-bits/nomos/libs/compiler/internal/core"
	"github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
)

// Sentinel errors for reference resolution failures.
var (
	// ErrUnresolvedReference indicates a reference could not be resolved.
	ErrUnresolvedReference = errors.New("unresolved reference")

	// ErrProviderNotRegistered indicates a provider alias is not registered.
	ErrProviderNotRegistered = errors.New("provider not registered")
)

// Provider is an alias to core.Provider for backward compatibility.
// Resolvers work with the core Provider interface.
type Provider = core.Provider

// ProviderRegistry provides access to registered providers.
// This interface is satisfied by compiler.ProviderRegistry.
type ProviderRegistry interface {
	GetProvider(alias string) (core.Provider, error)
}

// ResolverOptions configures the reference resolver.
//
//nolint:revive // Resolver prefix is part of public API naming convention
type ResolverOptions struct {
	// ProviderRegistry provides access to external data sources.
	ProviderRegistry ProviderRegistry

	// AllowMissingProvider controls behavior when a provider fetch fails.
	// If true, fetch failures are treated as non-fatal warnings.
	// If false (default), fetch failures cause the compilation to fail.
	AllowMissingProvider bool

	// OnWarning is called when a non-fatal warning occurs.
	// Only used when AllowMissingProvider is true.
	OnWarning func(warning string)
}

// Resolver resolves ReferenceExpr nodes to their actual values using providers.
type Resolver struct {
	opts   ResolverOptions
	cache  *fetchCache
	resCtx *ResolutionContext
}

// ResolutionContext tracks active resolution stack to detect circular references.
// This is a minimal version that mirrors the public API in reference_resolution.go.
type ResolutionContext struct {
	mu    sync.Mutex
	Stack []PathRef
}

// PathRef identifies a unique reference path being resolved.
type PathRef struct {
	Alias string
	Path  string
}

// String returns a human-readable representation of the PathRef.
func (r PathRef) String() string {
	return fmt.Sprintf("%s:%s", r.Alias, r.Path)
}

// Push adds a path to the resolution stack.
func (ctx *ResolutionContext) Push(alias string, path []string) error {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	ref := PathRef{Alias: alias, Path: pathKey(path)}

	for _, existing := range ctx.Stack {
		if existing == ref {
			return fmt.Errorf("%w: %s", ErrCircularReference, ctx.formatCycle(ref))
		}
	}

	ctx.Stack = append(ctx.Stack, ref)
	return nil
}

// Pop removes the most recent path from the resolution stack.
func (ctx *ResolutionContext) Pop() {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	if len(ctx.Stack) > 0 {
		ctx.Stack = ctx.Stack[:len(ctx.Stack)-1]
	}
}

// formatCycle creates a human-readable cycle path for error messages.
func (ctx *ResolutionContext) formatCycle(ref PathRef) string {
	parts := make([]string, 0, len(ctx.Stack)+1)
	for _, r := range ctx.Stack {
		parts = append(parts, r.String())
	}
	parts = append(parts, ref.String())
	return strings.Join(parts, " → ")
}

func pathKey(path []string) string {
	if len(path) == 0 {
		return "."
	}
	return strings.Join(path, ":")
}

// ErrCircularReference indicates a cycle was detected in the resolution chain.
var ErrCircularReference = errors.New("circular reference detected")

// New creates a new Resolver with the given options.
func New(opts ResolverOptions) *Resolver {
	if opts.ProviderRegistry == nil {
		panic("resolver: ProviderRegistry must not be nil")
	}

	return &Resolver{
		opts:   opts,
		cache:  newFetchCache(),
		resCtx: &ResolutionContext{Stack: []PathRef{}},
	}
}

// ResolveValue resolves a single value, replacing ReferenceExpr nodes with their resolved values.
// Returns the resolved value or an error if resolution fails.
func (r *Resolver) ResolveValue(ctx context.Context, val any) (any, error) {
	switch v := val.(type) {
	case *ast.ReferenceExpr:
		// Resolve reference expression
		return r.resolveReference(ctx, v)

	case map[string]any:
		// Recursively resolve map entries
		return r.resolveMap(ctx, v)

	case []any:
		// Recursively resolve slice elements
		return r.resolveSlice(ctx, v)

	default:
		// Scalar values and other types pass through
		return val, nil
	}
}

// resolveReference resolves a single ReferenceExpr by calling the appropriate provider.
func (r *Resolver) resolveReference(ctx context.Context, ref *ast.ReferenceExpr) (any, error) {
	if err := r.resCtx.Push(ref.Alias, ref.Path); err != nil {
		return nil, fmt.Errorf("resolving @%s:%s at %s:%d: %w",
			ref.Alias, pathKey(ref.Path),
			ref.SourceSpan.Filename, ref.SourceSpan.StartLine,
			err)
	}
	defer r.resCtx.Pop()

	// Build cache key
	cacheKey := buildCacheKey(ref.Alias, ref.Path)

	// Check cache first
	if val, ok := r.cache.get(cacheKey); ok {
		return val, nil
	}

	// Get provider
	provider, err := r.opts.ProviderRegistry.GetProvider(ref.Alias)
	if err != nil {
		return nil, r.handleProviderError(ref, err)
	}

	// Fetch value from provider
	val, err := provider.Fetch(ctx, ref.Path)
	if err != nil {
		return nil, r.handleFetchError(ref, ref.Path, err)
	}

	// Unwrap single-key "value" objects that some providers return for scalars
	// This is a workaround for provider implementations that wrap scalar values
	if m, ok := val.(map[string]any); ok && len(m) == 1 {
		if unwrapped, exists := m["value"]; exists {
			val = unwrapped
		}
	}

	// Resolve any nested references returned by the provider.
	resolved, err := r.ResolveValue(ctx, val)
	if err != nil {
		return nil, err
	}

	// Cache result
	r.cache.set(cacheKey, resolved)

	return resolved, nil
}

// resolveMap resolves all values in a map.
func (r *Resolver) resolveMap(ctx context.Context, m map[string]any) (map[string]any, error) {
	result := make(map[string]any, len(m))

	for k, v := range m {
		resolved, err := r.ResolveValue(ctx, v)
		if err != nil {
			return nil, fmt.Errorf("resolving key %q: %w", k, err)
		}
		result[k] = resolved
	}

	return result, nil
}

// resolveSlice resolves all elements in a slice.
func (r *Resolver) resolveSlice(ctx context.Context, s []any) ([]any, error) {
	result := make([]any, len(s))

	for i, v := range s {
		resolved, err := r.ResolveValue(ctx, v)
		if err != nil {
			return nil, fmt.Errorf("resolving index %d: %w", i, err)
		}
		result[i] = resolved
	}

	return result, nil
}

// handleProviderError handles errors from GetProvider.
func (r *Resolver) handleProviderError(ref *ast.ReferenceExpr, _ error) error {
	if r.opts.AllowMissingProvider && r.opts.OnWarning != nil {
		warning := fmt.Sprintf("provider %q not found for reference at %s:%d:%d",
			ref.Alias,
			ref.SourceSpan.Filename,
			ref.SourceSpan.StartLine,
			ref.SourceSpan.StartCol,
		)
		r.opts.OnWarning(warning)
		return nil // Return nil value for missing provider
	}

	// Fatal error
	return fmt.Errorf("%w: provider %q at %s:%d:%d",
		ErrProviderNotRegistered,
		ref.Alias,
		ref.SourceSpan.Filename,
		ref.SourceSpan.StartLine,
		ref.SourceSpan.StartCol,
	)
}

// handleFetchError handles errors from provider.Fetch.
func (r *Resolver) handleFetchError(ref *ast.ReferenceExpr, path []string, err error) error {
	if r.opts.AllowMissingProvider && r.opts.OnWarning != nil {
		warning := fmt.Sprintf("failed to fetch reference %q:%v at %s:%d:%d: %v",
			ref.Alias,
			path,
			ref.SourceSpan.Filename,
			ref.SourceSpan.StartLine,
			ref.SourceSpan.StartCol,
			err,
		)
		r.opts.OnWarning(warning)
		return nil // Return nil value for failed fetch
	}

	// Fatal error
	return fmt.Errorf("%w: failed to fetch %q:%v at %s:%d:%d: %w",
		ErrUnresolvedReference,
		ref.Alias,
		path,
		ref.SourceSpan.Filename,
		ref.SourceSpan.StartLine,
		ref.SourceSpan.StartCol,
		err,
	)
}

// buildCacheKey creates a cache key from provider alias and path.
func buildCacheKey(alias string, path []string) string {
	return alias + ":" + strings.Join(path, "/")
}

// fetchCache stores provider fetch results for the compilation run.
//
// The cache is thread-safe using read-write locks for concurrent access.
// For high-concurrency scenarios with many duplicate references, consider
// upgrading to use golang.org/x/sync/singleflight to deduplicate in-flight
// requests and avoid thundering herd on cold cache.
type fetchCache struct {
	mu    sync.RWMutex
	store map[string]any
}

// newFetchCache creates a new fetchCache.
func newFetchCache() *fetchCache {
	return &fetchCache{
		store: make(map[string]any),
	}
}

// get retrieves a cached value.
func (c *fetchCache) get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	val, ok := c.store[key]
	return val, ok
}

// set stores a value in the cache.
func (c *fetchCache) set(key string, val any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.store[key] = val
}
