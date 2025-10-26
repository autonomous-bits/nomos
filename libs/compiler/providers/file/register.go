package file

import (
	"fmt"

	"github.com/autonomous-bits/nomos/libs/compiler"
)

// RegisterFileProvider registers a file provider with the given registry.
//
// Parameters:
//   - registry: The provider registry to register with
//   - alias: The alias to register the provider under (e.g., "myfile")
//   - filePath: The file path to read from
//
// Example:
//
//	registry := compiler.NewProviderRegistry()
//	if err := RegisterFileProvider(registry, "config", "./configs/base.json"); err != nil {
//	    log.Fatal(err)
//	}
func RegisterFileProvider(registry compiler.ProviderRegistry, alias string, filePath string) error {
	if registry == nil {
		return fmt.Errorf("registry cannot be nil")
	}

	if alias == "" {
		return fmt.Errorf("alias cannot be empty")
	}

	if filePath == "" {
		return fmt.Errorf("file path cannot be empty")
	}

	// Create a constructor function that captures the file path
	constructor := func(opts compiler.ProviderInitOptions) (compiler.Provider, error) {
		provider := &FileProvider{
			configFilePath: filePath, // Store file path to be used during Init
		}

		// Return the provider without initializing - the registry will call Init
		return provider, nil
	}

	registry.Register(alias, constructor)
	return nil
}
