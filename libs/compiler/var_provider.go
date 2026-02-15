package compiler

import (
	"context"
	"fmt"
	"strings"

	"github.com/autonomous-bits/nomos/libs/compiler/internal/core"
)

// varProvider implements core.Provider for variable access.
// It serves values provided in Compile options via "var" alias.
type varProvider struct {
	vars map[string]any
}

// Init implements core.Provider.
func (p *varProvider) Init(_ context.Context, _ core.ProviderInitOptions) error {
	// No initialization needed
	return nil
}

// Fetch implements core.Provider.
func (p *varProvider) Fetch(_ context.Context, path []string) (any, error) {
	if len(p.vars) == 0 {
		return nil, fmt.Errorf("no variables defined")
	}

	// Navigate the path in the vars map
	return navigateVars(p.vars, path)
}

// navigateVars traverses the vars map using the provided path.
func navigateVars(data map[string]any, path []string) (any, error) {
	if len(path) == 0 {
		return data, nil
	}

	current := any(data)

	for i, segment := range path {
		currentMap, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("path segment %d is not a map (got %T): %w", i, current, fmt.Errorf("property path invalid"))
		}

		value, exists := currentMap[segment]
		if !exists {
			availableKeys := make([]string, 0, len(currentMap))
			for k := range currentMap {
				availableKeys = append(availableKeys, k)
			}
			return nil, fmt.Errorf("variable %q not found (available: %s)", strings.Join(path[:i+1], "."), strings.Join(availableKeys, ", "))
		}

		current = value
	}

	return current, nil
}
