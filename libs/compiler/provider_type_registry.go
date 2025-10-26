package compiler

import (
	"fmt"
	"sync"
)

// ProviderTypeConstructor creates a Provider from configuration.
// Unlike ProviderConstructor which is called during GetProvider, this is called
// when processing source declarations in .csl files.
type ProviderTypeConstructor func(config map[string]any) (Provider, error)

// ProviderTypeRegistry manages provider type constructors.
// This allows creating providers dynamically from source declarations in .csl files.
type ProviderTypeRegistry interface {
	// RegisterType registers a provider type constructor.
	// Example: RegisterType("file", NewFileProvider)
	RegisterType(typeName string, constructor ProviderTypeConstructor)

	// CreateProvider creates a provider instance of the given type with the provided config.
	// Returns an error if the type is not registered.
	CreateProvider(typeName string, config map[string]any) (Provider, error)

	// IsTypeRegistered checks if a provider type is registered.
	IsTypeRegistered(typeName string) bool

	// RegisteredTypes returns all registered provider type names.
	RegisteredTypes() []string
}

// providerTypeRegistry is the default implementation of ProviderTypeRegistry.
type providerTypeRegistry struct {
	mu           sync.RWMutex
	constructors map[string]ProviderTypeConstructor
}

// NewProviderTypeRegistry creates a new ProviderTypeRegistry.
func NewProviderTypeRegistry() ProviderTypeRegistry {
	return &providerTypeRegistry{
		constructors: make(map[string]ProviderTypeConstructor),
	}
}

// RegisterType implements ProviderTypeRegistry.RegisterType.
func (r *providerTypeRegistry) RegisterType(typeName string, constructor ProviderTypeConstructor) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.constructors[typeName] = constructor
}

// CreateProvider implements ProviderTypeRegistry.CreateProvider.
func (r *providerTypeRegistry) CreateProvider(typeName string, config map[string]any) (Provider, error) {
	r.mu.RLock()
	constructor, ok := r.constructors[typeName]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("provider type %q not registered", typeName)
	}

	// Create provider instance
	provider, err := constructor(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider of type %q: %w", typeName, err)
	}

	return provider, nil
}

// IsTypeRegistered implements ProviderTypeRegistry.IsTypeRegistered.
func (r *providerTypeRegistry) IsTypeRegistered(typeName string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.constructors[typeName]
	return ok
}

// RegisteredTypes implements ProviderTypeRegistry.RegisteredTypes.
func (r *providerTypeRegistry) RegisteredTypes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]string, 0, len(r.constructors))
	for typeName := range r.constructors {
		types = append(types, typeName)
	}

	return types
}
