# Consumer Example

This example demonstrates how external consumers can use Nomos libraries after they are published as versioned Go modules.

## Purpose

This example shows:
- How to import Nomos libraries using standard Go `require` directives (no `replace` needed)
- Basic usage of the `compiler.Compile` function
- Proper dependency management for external consumers

## Directory Structure

```
examples/consumer/
├── go.mod                          # Shows proper require usage
├── cmd/
│   └── consumer-example/
│       └── main.go                 # Example consumer application
├── testdata/
│   └── simple.csl                  # Test configuration file
├── example_test.go                 # Integration tests
└── README.md                       # This file
```

## Usage

### For External Consumers (After Publishing)

Once Nomos libraries are published with version tags, external consumers can add them to their `go.mod`:

```go
module github.com/example/my-project

go 1.26.0

require (
    github.com/autonomous-bits/nomos/libs/compiler v0.1.0
    github.com/autonomous-bits/nomos/libs/parser v0.1.0
    github.com/autonomous-bits/nomos/libs/provider-proto v0.1.0
)
```

Then use standard Go module commands:

```bash
go mod download
go build
```

### For Local Development (This Monorepo)

When developing within the Nomos monorepo, the workspace (`go.work` at repository root) automatically resolves internal dependencies:

```bash
# From repository root
go work sync

# Build the example
cd examples/consumer
go build -o consumer-example ./cmd/consumer-example

# Run the example
./consumer-example testdata/simple.csl
```

The `go.work` file handles module resolution, so no `replace` directives are needed in individual `go.mod` files.

## Example Application

The `cmd/consumer-example/main.go` demonstrates:

```go
import (
    "github.com/autonomous-bits/nomos/libs/compiler"
)

func main() {
    // Create compilation options
    opts := compiler.Options{
        Path: "config.csl",
        // ... configure options
    }
    
    // Compile configuration
    snapshot, err := compiler.Compile(context.Background(), opts)
    if err != nil {
        // handle error
    }
    
    // Use the compiled snapshot
}
```

## Migration from Replace Directives

If you're updating code that previously used `replace` directives:

**Before (local development only):**
```go
require (
    github.com/autonomous-bits/nomos/libs/compiler v0.0.0-00010101000000-000000000000
)

replace github.com/autonomous-bits/nomos/libs/compiler => ../../nomos/libs/compiler
```

**After (works for published modules):**
```go
require (
    github.com/autonomous-bits/nomos/libs/compiler v0.1.0
)
```

For local development in a monorepo, use Go workspaces (`go.work`) instead of `replace` directives.

## Testing

To verify the example builds correctly:

```bash
# From examples/consumer directory
go test ./...

# Build the binary
go build ./cmd/consumer-example

# Run with test data
./consumer-example testdata/simple.csl
```

## Further Reading

- [Go Modules Documentation](https://go.dev/doc/modules/managing-dependencies)
- [Go Workspaces Tutorial](https://go.dev/doc/tutorial/workspaces)
- [Migration Guide](../../docs/guides/removing-replace-directives.md)
- [Nomos Go Monorepo Structure](../../docs/architecture/go-monorepo-structure.md)

---

**Note**: This example uses `v0.0.0` placeholder versions in `go.mod` for local development. Once tags are published (e.g., `libs/compiler/v0.1.0`), external consumers will reference actual semantic versions.
