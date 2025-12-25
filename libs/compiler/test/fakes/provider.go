package fakes

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/autonomous-bits/nomos/libs/compiler"
)

// FakeProvider is a test double for the compiler.Provider interface.
// It records Init and Fetch call counts and allows configuring responses.
type FakeProvider struct {
	mu sync.Mutex

	// InitCount tracks how many times Init was called.
	InitCount int

	// FetchCount tracks how many times Fetch was called.
	FetchCount int

	// InitError is returned by Init if non-nil.
	InitError error

	// FetchError is returned by Fetch if non-nil.
	FetchError error

	// FetchResponses maps path strings to responses.
	// Keys are path components joined with "/".
	FetchResponses map[string]any

	// FetchCalls records all Fetch call paths in order.
	FetchCalls [][]string

	// Alias for the provider (used by Info).
	Alias string

	// Version for the provider (used by Info).
	Version string
}

// NewFakeProvider creates a new FakeProvider with sensible defaults.
func NewFakeProvider(alias string) *FakeProvider {
	return &FakeProvider{
		FetchResponses: make(map[string]any),
		Alias:          alias,
		Version:        "test-v1.0.0",
	}
}

// Init implements compiler.Provider.Init.
func (f *FakeProvider) Init(_ context.Context, opts compiler.ProviderInitOptions) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.InitCount++

	// Update alias from opts if provided
	if opts.Alias != "" && f.Alias == "" {
		f.Alias = opts.Alias
	}

	return f.InitError
}

// Fetch implements compiler.Provider.Fetch.
func (f *FakeProvider) Fetch(_ context.Context, path []string) (any, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.FetchCount++
	f.FetchCalls = append(f.FetchCalls, append([]string{}, path...))

	if f.FetchError != nil {
		return nil, f.FetchError
	}

	key := strings.Join(path, "/")
	if val, ok := f.FetchResponses[key]; ok {
		return val, nil
	}

	return nil, fmt.Errorf("no response configured for path %v", path)
}

// Info returns the provider's alias and version.
func (f *FakeProvider) Info() (alias string, version string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.Alias, f.Version
}

// Reset clears all call counts and recorded calls.
func (f *FakeProvider) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.InitCount = 0
	f.FetchCount = 0
	f.FetchCalls = nil
}
