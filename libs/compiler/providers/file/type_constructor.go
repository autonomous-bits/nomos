package file

import (
	"fmt"

	"github.com/autonomous-bits/nomos/libs/compiler"
)

// NewFileProviderFromConfig creates a FileProvider from configuration map.
// This is used by the ProviderTypeRegistry when processing source declarations.
//
// Required config keys:
//   - file (string): The file path to read from
//
// Example config:
//
//	{
//	    "file": "./configs/base.json"
//	}
func NewFileProviderFromConfig(config map[string]any) (compiler.Provider, error) {
	// Extract file path
	fileVal, ok := config["file"]
	if !ok {
		return nil, fmt.Errorf("file provider requires 'file' in config")
	}

	filePath, ok := fileVal.(string)
	if !ok {
		return nil, fmt.Errorf("file must be a string, got %T", fileVal)
	}

	if filePath == "" {
		return nil, fmt.Errorf("file cannot be empty")
	}

	// Create provider with config
	provider := &FileProvider{
		configFilePath: filePath,
	}

	return provider, nil
}
