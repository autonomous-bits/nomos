# File Provider

The file provider resolves imports from a folder containing `.csl` (Nomos configuration) files. It enables importing configuration snippets by name using the `import:{alias}:{name}` syntax.

## Features

- ✅ **Folder-based resolution**: Points to a directory containing `.csl` files
- ✅ **Named imports**: Use `import:{alias}:{name}` to reference files by base name
- ✅ **Strict .csl support**: Only `.csl` files are recognized (other extensions ignored)
- ✅ **Context-aware**: Respects context cancellation and timeouts
- ✅ **Duplicate detection**: Fails registration if duplicate base names exist

## Breaking Changes

**Version 0.2.0+**: The file provider now requires a **directory path** instead of a single file path. Only `.csl` files are supported. This is a breaking change from previous versions.

## Installation

The file provider is part of the `libs/compiler` module:

```bash
go get github.com/autonomous-bits/nomos/libs/compiler
```

## Quick Start

### Using RegisterFileProvider

Register a file provider with a directory containing `.csl` files:

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

	// Register file provider with alias "configs" pointing to a directory
	// The directory should contain .csl files like: network.csl, database.csl, etc.
	if err := file.RegisterFileProvider(registry, "configs", "./config-files"); err != nil {
		log.Fatal(err)
	}

	// Use in compilation - source files can now import using:
	// import:configs:network  -> resolves to ./config-files/network.csl
	// import:configs:database -> resolves to ./config-files/database.csl
	ctx := context.Background()
	opts := compiler.Options{
		Path:             "./sources",
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

}
```

## Import Syntax

### Basic Import

In your `.csl` source files, reference provider files using:

```
import:{alias}:{filename-without-extension}
```

**Example:**

If you registered a provider with alias `configs` pointing to `./config-files/`, and that directory contains:
- `network.csl`
- `database.csl`
- `secrets.csl`

You can import them as:
```
import:configs:network
import:configs:database
import:configs:secrets
```

### How It Works

1. Provider registration scans the directory for all `.csl` files
2. Files are indexed by their base name (filename without `.csl` extension)
3. Import statements resolve to the corresponding file content
4. Non-`.csl` files in the directory are ignored

## Supported File Format

### CSL (.csl)

Only `.csl` (Nomos configuration script language) files are supported. Other file types (`.json`, `.yaml`, `.txt`, etc.) in the provider directory are ignored.

**Example `network.csl`:**
```
config network {
  vpc_cidr = "10.0.0.0/16"
  region = "us-west-2"
}
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

The provider returns descriptive errors for common failure scenarios:

```go
// Example errors:
// - "directory does not exist: /path/to/configs"
// - "path is not a directory: /path/to/file.csl"
// - "no .csl files found in directory: /path/to/empty"
// - "duplicate file base name 'network' found in directory"
// - "import file 'missing' not found in provider 'configs'"
```

## Configuration

### ProviderInitOptions

When initializing via registry or manually:

```go
type ProviderInitOptions struct {
	Alias  string         // Provider alias (e.g., "configs")
	Config map[string]any // Configuration map
}
```

Required config keys:
- `directory` (string): Directory containing `.csl` files (must exist and be a directory)

### RegisterFileProvider Parameters

```go
func RegisterFileProvider(registry ProviderRegistry, alias string, directory string) error
```

- `registry`: The provider registry to register with
- `alias`: The alias to register under (e.g., "configs", "base", "shared")
- `directory`: The directory path containing `.csl` files (relative or absolute)

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

### Example 1: Basic Provider Registration

```
config-files/
├── network.csl
├── database.csl
└── security.csl
```

```go
registry := compiler.NewProviderRegistry()
file.RegisterFileProvider(registry, "configs", "./config-files")

// In your source .csl files, you can now use:
// import:configs:network
// import:configs:database
// import:configs:security
```

### Example 2: Multiple Provider Directories

```go
registry := compiler.NewProviderRegistry()

// Register different directories with different aliases
file.RegisterFileProvider(registry, "base", "./base-configs")
file.RegisterFileProvider(registry, "env", "./env-configs")
file.RegisterFileProvider(registry, "secrets", "./secrets")

// Source files can import from any registered provider:
// import:base:defaults
// import:env:production
// import:secrets:api-keys
```

### Example 3: Environment-Specific Configurations

```go
env := os.Getenv("ENV") // "dev", "staging", "prod"
configDir := filepath.Join("./configs", env)

registry := compiler.NewProviderRegistry()
file.RegisterFileProvider(registry, "env-config", configDir)

// Now imports resolve from the environment-specific directory
// import:env-config:database  → resolves to ./configs/{env}/database.csl
```

## Best Practices

1. **Organize .csl files by purpose**
   ```
   configs/
   ├── network.csl        # Network configurations
   ├── database.csl       # Database settings
   ├── observability.csl  # Logging, metrics
   └── security.csl       # Security policies
   ```

2. **Use descriptive provider aliases**
   ```go
   // ✅ Good - clear and descriptive
   file.RegisterFileProvider(registry, "base-configs", "./base")
   file.RegisterFileProvider(registry, "env-overrides", "./environments/prod")
   
   // ❌ Avoid - too generic
   file.RegisterFileProvider(registry, "provider1", "./base")
   file.RegisterFileProvider(registry, "data", "./environments/prod")
   ```

3. **Validate directory existence before registration**
   ```go
   configDir := "./config-files"
   if info, err := os.Stat(configDir); os.IsNotExist(err) || !info.IsDir() {
       log.Fatalf("config directory does not exist or is not a directory: %s", configDir)
   }
   file.RegisterFileProvider(registry, "configs", configDir)
   ```

4. **Handle registration errors**
   ```go
   if err := file.RegisterFileProvider(registry, "configs", "./config-files"); err != nil {
       log.Fatalf("failed to register provider: %v", err)
   }
   ```

## Security Notes

⚠️ **Important Security Considerations:**

1. **Directory Validation**: Always use trusted directory paths. Do not allow user input to determine provider directories without validation.

2. **File Permissions**: The provider reads files with the process's permissions. Ensure sensitive `.csl` files have appropriate OS-level protections.

3. **No Network Access**: This provider only accesses local files. For remote configurations, use a different provider (e.g., HTTP provider).

4. **Symbolic Links**: The current implementation follows symbolic links within the directory. Ensure links do not point to sensitive locations outside the intended scope.

## Version

Current version: **v0.2.0** (Breaking change: now requires directory, .csl only)

## License

See the root repository LICENSE file.

## Contributing

Contributions are welcome! Please:
1. Follow Test-Driven Development (TDD): write tests before implementation
2. Follow Go coding standards (see `go-standards` in repository)
3. Run `go fmt` and `golangci-lint`
4. Update this README for API changes
5. Update CHANGELOG.md following Keep a Changelog format

## Support

For issues and questions:
- Open an issue on GitHub
- See the main `libs/compiler` README for general compiler documentation
