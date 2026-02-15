# Inline Reference Examples

This directory contains example Nomos configuration files demonstrating various inline reference patterns.

## Examples

### Basic Usage

**[inline_reference_basic.csl](./inline_reference_basic.csl)**

The simplest use case: scalar inline references in a flat section.

```csl
infrastructure:
	vpc_cidr: @network:config.vpc.cidr
	region: 'us-west-2'
```

### Map/Collection Context

**[inline_reference_map.csl](./inline_reference_map.csl)**

Inline references within nested map structures, showing how different keys can reference different properties:

```csl
servers:
	web:
		ip: @network:config.web.ip
		port: '8080'
	api:
		ip: @network:config.api.ip
		port: '3000'
```

### Mixed Literals and References

**[inline_reference_mixed.csl](./inline_reference_mixed.csl)**

The most common real-world pattern: mixing string literals and inline references in the same section:

```csl
application:
	name: 'my-application'
	database_url: @config:config.database.connection_string
	debug_mode: 'false'
	api_key: @secrets:config.third_party.api_key
```

### Deeply Nested Structures

**[inline_reference_nested.csl](./inline_reference_nested.csl)**

Inline references at various nesting levels:

```csl
databases:
	primary:
		host: @infra:config.db.primary.host
		connection:
			max_pool_size: '20'
	replica:
		host: @infra:config.db.replica.host
```

## Using These Examples

You can parse these examples using the parser library:

```go
package main

import (
    "fmt"
    "log"

    "github.com/autonomous-bits/nomos/libs/parser"
)

func main() {
    ast, err := parser.ParseFile("docs/examples/inline_reference_basic.csl")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Parsed %d statements\n", len(ast.Statements))
}
```

## Related Documentation

- **Parser README**: `../README.md` - Full parser documentation
- **Migration Guide**: See "Migration Notes" section in `../README.md`
- **AST Types**: `../../pkg/ast/types.go` - ReferenceExpr definition
