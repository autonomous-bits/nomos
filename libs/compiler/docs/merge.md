# Merge Semantics

This document describes the composition and merge semantics used by the Nomos compiler when combining multiple configuration files.

## Overview

When the Nomos compiler processes multiple `.csl` files (either through explicit imports or directory compilation), it merges their content using deterministic composition semantics:

1. **Maps are deep-merged** — Nested map structures are merged recursively
2. **Arrays are replaced** — No deep-array merge; last file wins
3. **Scalars follow last-wins** — The last file to define a key determines its value
4. **Deterministic order** — Files are processed in lexicographic order

## Merge Rules

### Maps (Deep Merge)

Maps are merged recursively. When two files define the same top-level section, their nested keys are combined:

**base.csl:**
```yaml
database:
  host: 'localhost'
  port: '5432'
  name: 'myapp'
```

**override.csl:**
```yaml
database:
  port: '5433'
  ssl: 'true'
```

**Result:**
```json
{
  "database": {
    "host": "localhost",
    "port": "5433",
    "name": "myapp",
    "ssl": "true"
  }
}
```

In this example:
- `database.host` and `database.name` come from `base.csl` (not overridden)
- `database.port` comes from `override.csl` (last-wins)
- `database.ssl` is added from `override.csl` (new key)

### Arrays (Replacement)

Arrays are **not** deep-merged. The last file to define an array completely replaces any previous array value:

**base.csl:**
```yaml
servers:
  hosts: 'server1,server2,server3'
```

**override.csl:**
```yaml
servers:
  hosts: 'serverA,serverB'
```

**Result:**
```json
{
  "servers": {
    "hosts": "serverA,serverB"
  }
}
```

The array `hosts` from `override.csl` completely replaces the one from `base.csl`.

### Scalars (Last-Wins)

Scalar values (strings, numbers, booleans) use last-wins semantics:

**base.csl:**
```yaml
config:
  timeout: '30'
```

**override.csl:**
```yaml
config:
  timeout: '60'
```

**Result:**
```json
{
  "config": {
    "timeout": "60"
  }
}
```

### Type Conflicts

When different types are assigned to the same key, the last file's value wins regardless of type:

**Example 1: Scalar replaces map**
```yaml
# base.csl
config:
  nested:
    key: 'value'

# override.csl
config:
  nested: 'simple string'

# Result: config.nested = "simple string"
```

**Example 2: Map replaces scalar**
```yaml
# base.csl
config:
  value: 'string'

# override.csl
config:
  value:
    nested: 'data'

# Result: config.value = { "nested": "data" }
```

## Deterministic File Order

When compiling a directory, the compiler processes `.csl` files in **lexicographic (alphabetical) order** by filename. This ensures deterministic, reproducible builds:

```
directory/
  ├── 01-base.csl      # Processed first
  ├── 02-override.csl  # Processed second
  └── 99-final.csl     # Processed last
```

Files processed later override values from earlier files (last-wins).

## Provenance Tracking

The compiler tracks the origin of each top-level configuration key in `Snapshot.Metadata.PerKeyProvenance`. This helps debug last-wins behavior:

```go
snapshot.Metadata.PerKeyProvenance["database"]
// Returns: Provenance{Source: "/path/to/override.csl"}
```

The provenance records the **last file** that modified each top-level key. This applies even for deep-merged maps — if any nested key is modified, the top-level key's provenance is updated.

## Implementation Details

The merge logic is implemented in `merge.go`:

- `DeepMerge(dst, src map[string]any) map[string]any` — Pure merge function
- `DeepMergeWithProvenance(...)` — Merge with provenance tracking
- Both functions are non-mutating; they return new maps

### Deep-Copy Guarantees

All merge operations create new data structures without mutating inputs. This ensures:
- Input ASTs are never modified
- Intermediate merge results can be safely cached
- Tests can rely on predictable input state

## Examples

### Example 1: Multi-Environment Configuration

**common.csl:**
```yaml
app:
  name: 'myapp'
  version: '1.0'

database:
  host: 'localhost'
  port: '5432'
```

**production.csl:**
```yaml
database:
  host: 'db.prod.example.com'
  ssl: 'true'

app:
  debug: 'false'
```

**Merged Result:**
```json
{
  "app": {
    "name": "myapp",
    "version": "1.0",
    "debug": "false"
  },
  "database": {
    "host": "db.prod.example.com",
    "port": "5432",
    "ssl": "true"
  }
}
```

### Example 2: Nested Map Merging

**base.csl:**
```yaml
network:
  vpc:
    cidr: '10.0.0.0/16'
    region: 'us-east-1'
  subnets:
    public: '10.0.1.0/24'
```

**override.csl:**
```yaml
network:
  vpc:
    cidr: '10.1.0.0/16'
  subnets:
    private: '10.1.2.0/24'
```

**Merged Result:**
```json
{
  "network": {
    "vpc": {
      "cidr": "10.1.0.0/16",
      "region": "us-east-1"
    },
    "subnets": {
      "public": "10.0.1.0/24",
      "private": "10.1.2.0/24"
    }
  }
}
```

## Testing

The merge semantics are validated by:

1. **Unit tests** (`merge_test.go`): Test pure merge functions with table-driven tests
2. **Integration tests** (`test/merge_integration_test.go`): Test end-to-end compilation with real `.csl` files
3. **Golden tests**: Compare compiled output against expected JSON snapshots

To run tests:

```bash
# Unit tests
go test -v -run TestDeepMerge

# Integration tests
go test -v -run TestMergeSemantics ./test

# Update golden files
GOLDEN_UPDATE=1 go test -v -run TestMergeSemantics_GoldenOutput ./test
```

## See Also

- [Compiler API Documentation](../README.md)
- [Parser AST Types](../../parser/pkg/ast/types.go)
- [Test Fixtures](../testdata/merge_semantics/)
