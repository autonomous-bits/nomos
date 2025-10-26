# File Provider

The file provider resolves inline references from local files, supporting JSON and YAML formats. It enforces path traversal protection to prevent access outside the configured base directory.

## Features

- ✅ **Local file resolution**: Fetch data from JSON and YAML files
- ✅ **Path security**: Prevents path traversal attacks outside base directory
- ✅ **Nested paths**: Supports multi-component paths (e.g., `["configs", "network", "vpc.json"]`)
- ✅ **Context-aware**: Respects context cancellation and timeouts
- ✅ **Format validation**: Clear errors for unsupported file formats

## Installation

The file provider is part of the `libs/compiler` module:

```bash
go get github.com/autonomous-bits/nomos/libs/compiler
```

## Quick Start

### Using RegisterFileProvider

The simplest way to use the file provider:

```go
package main

import (
	"context"
	"log"

	"github.com/autonomous-bits/nomos/libs/compiler"
	"github.com/autonomous-bits/nomos/libs/compiler/providers/file"
)

func main() {
	// Create provider registry
	registry := compiler.NewProviderRegistry()

	// Register file provider with alias "file" and base directory "./data"
	if err := file.RegisterFileProvider(registry, "file", "./data"); err != nil {
		log.Fatal(err)
	}

	// Use in compilation
	ctx := context.Background()
	opts := compiler.Options{
		Path:             "./configs",
		ProviderRegistry: registry,
	}

	snapshot, err := compiler.Compile(ctx, opts)
	if err != nil {
		log.Fatal(err)
	}

	// Use snapshot...
	_ = snapshot
}
```

### Manual Provider Creation

For more control, create the provider manually:

```go
package main

import (
	"context"
	"log"

	"github.com/autonomous-bits/nomos/libs/compiler"
	"github.com/autonomous-bits/nomos/libs/compiler/providers/file"
)

func main() {
	// Create provider instance
	provider := &file.FileProvider{}

	// Initialize with configuration
	opts := compiler.ProviderInitOptions{
		Alias: "file",
		Config: map[string]any{
			"baseDir": "./data",
		},
	}

	if err := provider.Init(context.Background(), opts); err != nil {
		log.Fatal(err)
	}

	// Fetch a file
	result, err := provider.Fetch(context.Background(), []string{"config.json"})
	if err != nil {
		log.Fatal(err)
	}

	// result is parsed JSON/YAML as map[string]any
	data, ok := result.(map[string]any)
	if !ok {
		log.Fatal("unexpected result type")
	}

	log.Printf("Loaded config: %+v\n", data)
}
```

## Supported File Formats

### JSON (.json)

```json
{
  "database": {
    "host": "localhost",
    "port": 5432,
    "name": "myapp"
  }
}
```

### YAML (.yaml, .yml)

```yaml
database:
  host: localhost
  port: 5432
  name: myapp
```

## Path Resolution

Paths are resolved relative to the configured `baseDir`:

```go
// Given baseDir = "./data"
// Fetch(["config.json"]) → reads ./data/config.json
// Fetch(["configs", "network.yaml"]) → reads ./data/configs/network.yaml
```

### Security

The provider enforces strict path security:

```go
// ✅ Allowed: relative paths within baseDir
Fetch([]string{"config.json"})
Fetch([]string{"configs", "db.yaml"})

// ❌ Blocked: path traversal attempts
Fetch([]string{"..", "etc", "passwd"})      // Error: path resolves outside base directory
Fetch([]string{"/etc/passwd"})              // Error: path traversal attempt
```

**Security guarantees:**
- All paths are canonicalized using `filepath.Clean`
- Absolute path components are rejected
- Traversal outside `baseDir` is prevented
- Base directory must exist and be a valid directory

## Context Support

The provider respects Go context for cancellation and timeouts:

```go
// With timeout
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

result, err := provider.Fetch(ctx, []string{"large-file.json"})
if err == context.DeadlineExceeded {
	log.Println("Fetch timed out")
}

// With cancellation
ctx, cancel := context.WithCancel(context.Background())
go func() {
	time.Sleep(1 * time.Second)
	cancel() // Cancel after 1 second
}()

result, err := provider.Fetch(ctx, []string{"config.json"})
if err == context.Canceled {
	log.Println("Fetch was cancelled")
}
```

## Error Handling

The provider returns descriptive errors:

```go
result, err := provider.Fetch(ctx, []string{"nonexistent.json"})
if err != nil {
	// Error examples:
	// - "file not found: nonexistent.json"
	// - "unsupported file format \".txt\" for file \"data.txt\" (supported: .json, .yaml, .yml)"
	// - "failed to parse JSON file \"bad.json\": invalid character '}' ..."
	// - "path traversal attempt: component \"/etc/passwd\" is absolute"
	log.Printf("Fetch error: %v\n", err)
}
```

## Configuration

### ProviderInitOptions

When initializing manually or via constructor:

```go
type ProviderInitOptions struct {
	Alias  string         // Provider alias (e.g., "file")
	Config map[string]any // Configuration map
}
```

Required config keys:
- `baseDir` (string): Base directory for file resolution (must exist)

### RegisterFileProvider Parameters

```go
func RegisterFileProvider(registry ProviderRegistry, alias string, baseDir string) error
```

- `registry`: The provider registry to register with
- `alias`: The alias to register under (e.g., "file", "data", "configs")
- `baseDir`: The base directory path (relative or absolute)

## Testing

### Unit Tests

Run unit tests:

```bash
cd libs/compiler/providers/file
go test -v
```

### Integration Tests

Run integration tests (require `integration` build tag):

```bash
cd libs/compiler/providers/file
go test -v -tags=integration
```

### Test Coverage

Generate coverage report:

```bash
go test -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Examples

### Example 1: Simple Configuration Loading

```go
// data/app.json
{
  "name": "myapp",
  "version": "1.0.0"
}
```

```go
registry := compiler.NewProviderRegistry()
file.RegisterFileProvider(registry, "file", "./data")

provider, _ := registry.GetProvider("file")
result, _ := provider.Fetch(context.Background(), []string{"app.json"})

config := result.(map[string]any)
fmt.Println(config["name"])    // "myapp"
fmt.Println(config["version"]) // "1.0.0"
```

### Example 2: Nested Directory Structure

```
data/
├── configs/
│   ├── database.yaml
│   └── cache.json
└── secrets/
    └── api-keys.yaml
```

```go
registry := compiler.NewProviderRegistry()
file.RegisterFileProvider(registry, "config", "./data/configs")
file.RegisterFileProvider(registry, "secrets", "./data/secrets")

configProvider, _ := registry.GetProvider("config")
secretProvider, _ := registry.GetProvider("secrets")

// Fetch from configs
dbConfig, _ := configProvider.Fetch(ctx, []string{"database.yaml"})

// Fetch from secrets
apiKeys, _ := secretProvider.Fetch(ctx, []string{"api-keys.yaml"})
```

### Example 3: Multiple Environments

```go
env := os.Getenv("ENV") // "dev", "staging", "prod"
baseDir := filepath.Join("./configs", env)

registry := compiler.NewProviderRegistry()
file.RegisterFileProvider(registry, "file", baseDir)

// Now Fetch reads from the environment-specific directory
```

## Best Practices

1. **Use absolute or well-known relative paths for baseDir**
   ```go
   // ✅ Good
   file.RegisterFileProvider(registry, "file", "./data")
   file.RegisterFileProvider(registry, "file", "/etc/myapp/configs")
   
   // ❌ Avoid
   file.RegisterFileProvider(registry, "file", "../../data")
   ```

2. **Validate baseDir exists before registration**
   ```go
   baseDir := "./data"
   if _, err := os.Stat(baseDir); os.IsNotExist(err) {
       log.Fatalf("baseDir does not exist: %s", baseDir)
   }
   file.RegisterFileProvider(registry, "file", baseDir)
   ```

3. **Use descriptive aliases for multiple providers**
   ```go
   file.RegisterFileProvider(registry, "app-config", "./configs")
   file.RegisterFileProvider(registry, "templates", "./templates")
   file.RegisterFileProvider(registry, "schemas", "./schemas")
   ```

4. **Handle errors appropriately**
   ```go
   result, err := provider.Fetch(ctx, path)
   if err != nil {
       if os.IsNotExist(err) {
           // Handle missing file
       } else if errors.Is(err, context.DeadlineExceeded) {
           // Handle timeout
       } else {
           // Handle other errors
       }
   }
   ```

## Security Notes

⚠️ **Important Security Considerations:**

1. **baseDir Validation**: Always use trusted baseDir paths. Do not allow user input to determine baseDir.

2. **Path Components**: The provider validates path components but does not perform additional access control. Ensure baseDir permissions are correctly configured at the OS level.

3. **File Permissions**: The provider reads files with the process's permissions. Ensure sensitive files have appropriate OS-level protections.

4. **No Network Access**: This provider only accesses local files. For remote files, use a different provider (e.g., HTTP provider).

5. **Symbolic Links**: The current implementation does not specially handle symbolic links. Links are followed if they point within baseDir after canonicalization.

## Version

Current version: **v0.1.0**

## License

See the root repository LICENSE file.

## Contributing

Contributions are welcome! Please:
1. Write tests for new functionality
2. Follow Go coding standards
3. Run `go fmt` and `golangci-lint`
4. Update this README for API changes

## Support

For issues and questions:
- Open an issue on GitHub
- See the main `libs/compiler` README for general compiler documentation
