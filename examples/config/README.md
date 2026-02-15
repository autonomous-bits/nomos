# Nomos Configuration Examples

This directory contains example Nomos configuration files demonstrating various features and patterns.

## Basic Examples

### config.csl
Basic configuration demonstrating:
- Source provider configuration
- Property references using `@alias:path` syntax
- Environment-specific values

### config2.csl
Multi-source configuration showing:
- Multiple provider instances with different aliases
- Mixing references from different sources
- Lists and nested objects

## Test Files

### test-simple.csl
Minimal configuration for testing basic parsing and compilation.

### test-provider.csl
Demonstrates all three reference modes:
- **Root references**: `@alias:*` - includes all properties at the provider root
- **Map references**: `@alias:path` - includes a specific map
- **Property references**: `@alias:path.to.value` - single value

### test-scalars.csl
Tests property references with various scalar types (strings, numbers, etc.).

### Additional Test Files
- `test-deeply-nested.csl` - Nested configuration structures
- `test-final.csl` - Final composition testing
- `test-no-provider.csl` - Config without provider dependencies
- `test-source.csl` - Source declaration patterns

## Pattern Examples

### example-root-reference.csl
Demonstrates **map references** (`@alias:path`) to include all properties from a path with selective overrides using deep merge semantics.

**Pattern**: Base configuration + targeted overrides
```csl
database:
  @base:prod
  host: 'localhost'  # Override base host, preserve other properties
```

### example-map-reference.csl
Demonstrates **map references** to include specific nested maps without unrelated properties.

**Pattern**: Granular imports of configuration sections
```csl
database:
  @shared:config.database
  # Only database.* properties included, not network, logging, etc.
```

### example-property-reference.csl
Demonstrates **property references** to reference single scalar values.

**Pattern**: Precise value reuse
```csl
app:
  api_url: @common:config.api.url
  api_key: @common:config.api.key
```

### example-layered-config/
Complete example of the **layered configuration pattern** with:
- `base.csl` - Common defaults
- `dev.csl` - Development overrides
- `staging.csl` - Staging overrides  
- `prod.csl` - Production overrides

**Pattern**: Environment-specific configurations inheriting from base
```csl
# prod.csl
config:
  @base:base
  database:
    host: 'prod-db.internal'  # Only override what's different
```

## Reference Syntax

Nomos uses a single reference syntax for all references:

```
@alias:path
```

- **`alias`**: Provider instance alias (configured in `source:` block)
- **`path`**: Dot/bracket path only (no additional `:`)
- **`*`**: Only allowed as the final path segment (e.g., `@alias:*` or `@alias:path.*`)

### Three Reference Modes

1. **Root Reference** - Include all properties at the provider root:
   ```csl
  @alias:*
   ```

2. **Map Reference** - Include a specific nested map:
   ```csl
  @alias:path.to.map
   ```

3. **Property Reference** - Resolve a single value:
   ```csl
  @alias:path.to.property
   ```

## Deep Merge Semantics

When properties are defined after a reference, Nomos uses **deep merge** with override semantics:

- **Maps are deep-merged**: Nested properties override individually
- **Arrays are replaced**: No array merging, last-wins
- **Scalars follow last-wins**: Later values override earlier ones

Example:
```csl
database:
  @base:config
  # Base has: host='base-host', port=5432, pool={min=5, max=20}
  
  host: 'override-host'  # Overrides base host
  pool:
    max: 50              # Overrides pool.max, preserves pool.min=5
  # port=5432 is preserved from base
```

Result:
```yaml
database:
  host: 'override-host'  # From override
  port: 5432             # From base (preserved)
  pool:
    min: 5               # From base (preserved)
    max: 50              # From override
```

## File Provider Conventions

When using the file provider:
- The first path segment is treated as the filename without the `.csl` extension
- The provider automatically appends `.csl` when resolving files
- Example: `@configs:database.property` resolves to `database.csl` in the configured directory

```csl
source:
  alias: 'configs'
  type: 'autonomous-bits/nomos-provider-file'
  directory: './data'

# References 'database.csl' in './data/' directory
app:
  db_host: @configs:database.host
```

## Running Examples

Compile any example configuration:

```bash
# Basic example
nomos compile examples/config/config.csl

# Layered configuration (environment-specific)
nomos compile examples/config/example-layered-config/prod.csl

# Test specific patterns
nomos compile examples/config/example-root-reference.csl
```

## Getting Started

1. Start with `example-property-reference.csl` for simple value references
2. Try `example-map-reference.csl` to import configuration sections
3. Study `example-root-reference.csl` for base + override pattern
4. Explore `example-layered-config/` for environment-specific configurations

## Migration from Old Syntax

If you have configurations using the old `import` statement syntax, see the [migration guide](../../docs/guides/expand-at-references-migration.md) for step-by-step conversion instructions.

## Additional Resources

- **Migration Guide**: `docs/guides/expand-at-references-migration.md`
- **Parser README**: `libs/parser/README.md`
- **Compiler README**: `libs/compiler/README.md`
- **Feature Specification**: `specs/006-expand-at-references/spec.md`
