package file

import (
	"fmt"

	"github.com/autonomous-bits/nomos/libs/compiler"
)

// NewFileProviderFromConfig creates a FileProvider from configuration map.
// This is used by the ProviderTypeRegistry when processing source declarations.
//
// Required config keys (one of):
//   - directory (string): The directory path containing .csl files (v0.2.0+)
//   - file (string): The file path to read from (legacy, v0.1.0)
//
// Example config (v0.2.0+):
//
//	{
//	    "directory": "./shared-configs"
//	}
//
// Example config (legacy):
//
//	{
//	    "file": "./configs/base.json"
//	}
func NewFileProviderFromConfig(config map[string]any) (compiler.Provider, error) {
	provider := &FileProvider{}

	// Try directory first (v0.2.0+)
	if dirVal, ok := config["directory"]; ok {
		dirPath, ok := dirVal.(string)
		if !ok {
			return nil, fmt.Errorf("directory must be a string, got %T", dirVal)
		}
		if dirPath == "" {
			return nil, fmt.Errorf("directory cannot be empty")
		}
		provider.configFilePath = dirPath
		return provider, nil
	}

	// Fall back to file (legacy)
	if fileVal, ok := config["file"]; ok {
		filePath, ok := fileVal.(string)
		if !ok {
			return nil, fmt.Errorf("file must be a string, got %T", fileVal)
		}
		if filePath == "" {
			return nil, fmt.Errorf("file cannot be empty")
		}
		provider.configFilePath = filePath
		return provider, nil
	}

	return nil, fmt.Errorf("file provider requires 'directory' or 'file' in config")
}
