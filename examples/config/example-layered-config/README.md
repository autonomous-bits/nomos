# Layered Configuration Pattern

This example demonstrates the layered configuration pattern using root references with cascading overrides.

## Pattern Overview

The layered configuration pattern uses root references to build configurations from multiple layers:
1. **Base layer**: Common defaults shared across environments
2. **Environment layer**: Environment-specific overrides (dev, staging, prod)
3. **Application layer**: Application-specific customizations

Each layer uses map references (`@alias:path`) to include all properties from the previous layer, then selectively overrides specific values using deep merge semantics.

## Files

- `base.csl` - Base configuration with common defaults
- `dev.csl` - Development environment overrides
- `staging.csl` - Staging environment overrides
- `prod.csl` - Production environment overrides

## Usage

Compile any environment configuration to get the fully merged result:

```bash
# Development environment
nomos compile example-layered-config/dev.csl

# Staging environment
nomos compile example-layered-config/staging.csl

# Production environment
nomos compile example-layered-config/prod.csl
```

## Deep Merge Behavior

When using root references with overrides:
- **Maps are deep-merged**: Nested properties can be overridden individually while preserving siblings
- **Arrays are replaced**: Overriding an array replaces the entire array from the base
- **Scalars follow last-wins**: Later declarations override earlier ones

Example:
```csl
# base.csl defines:
database:
  host: 'base-host'
  port: '5432'
  pool:
    min: '5'
    max: '20'

# prod.csl includes base and overrides:
database:
  @base:database
  host: 'prod-host'  # Overrides base host
  pool:
    max: '50'  # Overrides base pool.max, preserves pool.min
```

Result:
```yaml
database:
  host: 'prod-host'     # From prod
  port: '5432'          # From base (preserved)
  pool:
    min: '5'            # From base (preserved)
    max: '50'           # From prod (overridden)
```

## Benefits

1. **DRY Principle**: Common configuration defined once in base layer
2. **Clear Overrides**: Environment differences are explicit and localized
3. **Inheritance**: Changes to base automatically propagate to all environments
4. **Flexibility**: Any property can be overridden at any layer without boilerplate
