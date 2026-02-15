# Migration Guide: Expand At-Reference Syntax

**Feature**: `006-expand-at-references`  
**Breaking Change**: Yes - MAJOR version bump required  
**Migration Effort**: Low to Medium (mostly syntax changes, no semantic changes)

## What Changed

The Nomos configuration language has been simplified by removing the `import` statement and clarifying the at-reference (`@`) syntax to treat everything after the first `:` as a dot-only path:

### Removed

- **`import` statement**: Top-level import declarations have been completely removed from the language

### Added

- **Path-based reference syntax**: `@alias:path` where:
  - `alias` = configured provider instance (from `source:` block)
  - `path` = dot/bracket path only (no additional `:`)
- **Root references**: `@alias:*` includes all properties at the provider root
- **Wildcard placement**: `*` is only allowed as the final path segment (e.g., `@alias:path.*`)
- **Map references**: `@alias:path.to.map` includes a specific nested map
- **Property references**: `@alias:path.to.property` resolves a single value
- **Deep merge with overrides**: Properties defined after a reference override using deep merge semantics

### Changed

- **Reference resolution**: Everything after the first `:` is passed to providers as dot-separated path segments
- **File provider behavior**: The first path segment is treated as the filename (without `.csl`), appended automatically
- **Import/merge behavior**: Instead of `import:alias` followed by section redefinitions, use map references followed by property overrides

## Syntax Changes

### Before (Old Syntax with `import`)

```csl
source:
  alias: 'configs'
  type: 'autonomous-bits/nomos-provider-file'
  version: '0.2.1'
  directory: './data'

# Import entire config file
import:configs:database

# Reference single value
database:
  url: @configs:prod.database.url
```

### After (Current @alias:path Reference Syntax)

```csl
source:
  alias: 'configs'
  type: 'autonomous-bits/nomos-provider-file'
  version: '0.2.1'
  directory: './data'

# Include all properties from database path using a map reference
database:
  @configs:database

# Reference single value with a provider path
database:
  url: @configs:prod.database.url
```

## Step-by-Step Migration

### Step 1: Identify All `import` Statements

Search your codebase for `import:` statements:

```bash
# Find all files with import statements
grep -r "^import:" examples/config/
```

### Step 2: Convert `import` Statements to Map References

For each `import:alias:path` statement, replace with `@alias:path` at the appropriate indentation level.

**Pattern**:
```
# Old
import:alias:path

section:
  key: value

# New
section:
  @alias:path
  key: value  # This overrides property from the referenced path
```

**Example**:
```csl
# Before
import:configs:database

database:
  host: 'localhost'

# After
database:
  @configs:database
  host: 'localhost'  # Overrides host from imported config
```

### Step 3: Update Reference Paths for Provider Segments

If the provider expects a leading path segment (for example, a filename for the file provider), add it as the first dot path segment.

**Pattern**:
```
# Old
@alias:path.to.property

# New (example with a leading provider path segment)
@alias:segment.path.to.property
```

**Example**:
```csl
# Before
database:
  url: @configs:database.url

# After
database:
  url: @configs:prod.database.url  # 'prod' is the first path segment (prod.csl)
```

### Step 4: Remove File Extensions from Filename Segments

If using the file provider, remove `.csl` extensions from filename segments (they're added automatically).

**Pattern**:
```
# Old (if you had extensions)
@alias:database.csl.property

# New
@alias:database.property
```

### Step 5: Apply Deep Merge Overrides

Where you previously had separate import and override sections, combine them using map references followed by property overrides.

**Example**:
```csl
# Before (separate import and section)
import:base:common

app:
  name: 'myapp'

database:
  host: 'prod.example.com'

# After (map reference with overrides)
config:
  @base:common
  app:
    name: 'myapp'
  database:
    host: 'prod.example.com'
    # Other database properties from common are preserved
```

### Step 6: Verify Compilation

After making changes, verify your configuration compiles:

```bash
nomos compile config.csl
```

## Common Patterns

### Pattern 1: Import and Override

**Before**:
```csl
import:base:config

database:
  host: 'override-host'
```

**After**:
```csl
database:
  @base:config
  host: 'override-host'
```

**Deep Merge Behavior**: The `host` property is overridden, but `port`, `timeout`, and other properties from the base config are preserved.

---

### Pattern 2: Multiple Imports

**Before**:
```csl
import:configs:database
import:configs:network
```

**After**:
```csl
config:
  @configs:database
  @configs:network
```

**Note**: References are processed in order. If both paths define overlapping properties, later references override earlier ones.

---

### Pattern 3: Selective Property Import

**Before**:
```csl
import:configs:database

app:
  db_host: @configs:database.host
  db_port: @configs:database.port
```

**After**:
```csl
app:
  db_host: @configs:database.host
  db_port: @configs:database.port
```

**Note**: No longer need to import the entire path first - just reference the specific properties directly.

---

### Pattern 4: Nested Map Import

**Before**: Not directly supported - had to import entire file and use references

**After**:
```csl
database:
  # Include only the connection pool configuration
  pool: @configs:database.connection.pool
```

**Benefit**: More granular control - only include the specific map you need.

---

### Pattern 5: Environment-Specific Overrides

**Before**:
```csl
import:base:common

database:
  host: 'prod.example.com'
  port: 5432
```

**After**:
```csl
config:
  @base:common
  database:
    host: 'prod.example.com'
    # Only override what's different for prod
    # port, pool settings, etc. from common are preserved
```

**Benefit**: Deep merge means you only need to specify what's different, not repeat the entire configuration.

---

### Pattern 6: Layered Configuration

**Before**: Required complex import chains and manual property copying

**After**:
```csl
# base.csl - common defaults
application:
  name: 'myapp'
  timeout: '30s'

# prod.csl - production overrides
source:
  alias: 'base'
  type: 'autonomous-bits/nomos-provider-file'
  directory: '.'

config:
  @base:base
  application:
    timeout: '60s'  # Override timeout for prod
    # name is preserved from base
```

**Benefit**: Clear inheritance and override pattern without boilerplate.

## Troubleshooting

### Error: "alias not found"

**Problem**: Reference uses an alias that isn't configured in a `source:` block.

```
Error: alias 'unknown' not found in source configuration
```

**Solution**: Ensure the alias is defined before use:
```csl
source:
  alias: 'configs'  # Define the alias
  type: 'autonomous-bits/nomos-provider-file'
  directory: './data'

database:
  @configs:database  # Now this works
```

---

### Error: "path not found"

**Problem**: Provider cannot resolve the path (e.g., file doesn't exist).

```
Error: path 'missing' not found by provider 'configs'
```

**Solution**: 
- For file provider: Ensure the file exists in the configured directory
  - Looking for `database` → ensure `./data/database.csl` exists
- Check path segment spelling
- Verify provider configuration (correct directory path)

---

### Error: "property path invalid"

**Problem**: Property path doesn't exist in the resolved data.

```
Error: property path 'database.invalid.path' not found (available keys: [host, port, timeout])
```

**Solution**:
- Check the source data to confirm the property path exists
- Fix typos in the property path
- Use the "available keys" hint from the error message to find the correct path

---

### Error: "circular reference detected"

**Problem**: Configuration A references B, which references A (cycle).

```
Error: circular reference detected: base:app → base:common → base:app
```

**Solution**:
- Review the reference chain shown in the error
- Break the cycle by removing one of the references or restructuring the configuration
- Consider creating a third shared path that both can reference

---

### Error: "expected map at path, got string"

**Problem**: Using map reference syntax with a path that resolves to a scalar.

```
Error: expected map at configs:database.host, got string
```

**Solution**: This is actually treated as forgiving behavior (FR-016) - the system will insert the scalar value. If you get this error, it means you're using an outdated compiler version.

Update to the latest version or change your reference to explicitly use property mode if you intend to reference a scalar:
```csl
# Instead of trying to use map mode on a scalar
database:
  @configs:prod.database.host  # Property reference

# Use this syntax
database:
  host: @configs:prod.database.host
```

---

### Compilation is Slow

**Problem**: Deep property path traversal or complex deep merges taking too long.

**Solution**:
- Reduce nesting depth where possible
- Break large configurations into smaller, focused paths or files
- Use specific map or property references instead of root references when you only need a subset
- Ensure you're not creating circular reference chains (causes repeated resolution attempts)

---

### Properties Not Overriding as Expected

**Problem**: After migration, property overrides don't work as expected.

**Example**:
```csl
database:
  @base:database
  host: 'override'  # This isn't overriding?
```

**Solution**: Ensure overrides are at the correct nesting level and come after the reference:
```csl
# Correct
database:
  @base:database
  host: 'override'  # Same level as other database properties
```

Deep merge preserves structure, so if the base has `database.connection.host`, override at that same path:
```csl
database:
  @base:database
  connection:
    host: 'override'  # Correct nesting to override connection.host
```

---

### Lost Properties After Migration

**Problem**: Properties that existed before migration are missing after.

**Solution**: This usually happens when converting imports to references at the wrong indentation level. 

**Before**:
```csl
import:base:config

application:
  name: 'myapp'
```

**Wrong**:
```csl
@base:*  # Root level - imports everything at root
application:
  name: 'myapp'
```

**Correct**:
```csl
# If base:config has application.name, override it specifically
application:
  @base:config.application
  name: 'myapp'
```

Or use root reference and let override work:
```csl
@base:*
application:
  name: 'myapp'  # Overrides base application properties
```

## Migration Checklist

- [ ] Identify all files using `import:` statements
- [ ] For each `import:`, determine if it should be:
  - Root reference (`@alias:*`) - most common
  - Map reference (`@alias:path.to.map`) - for specific sections
  - Multiple property references - for individual values
- [ ] Convert multi-segment references to dot-only paths
- [ ] Remove `.csl` extensions from filename path segments
- [ ] Ensure overrides are placed at correct indentation levels
- [ ] Test compilation with `nomos compile`
- [ ] Verify output matches expected configuration
- [ ] Update any documentation mentioning `import` statements
- [ ] Update CI/CD pipelines if they reference the old syntax
- [ ] Notify team members of the syntax change

## Additional Resources

- **Examples**: See `examples/config/example-*.csl` for working examples
- **Specification**: `specs/006-expand-at-references/spec.md`
- **Parser Changes**: `libs/parser/CHANGELOG.md`
- **Compiler Changes**: `libs/compiler/CHANGELOG.md`

## Need Help?

If you encounter issues not covered by this guide, please:
1. Check the error message carefully - they include helpful context
2. Review the examples in `examples/config/`
3. Open an issue on GitHub with:
   - Your configuration file (before and after)
   - The error message
   - Expected vs. actual behavior
