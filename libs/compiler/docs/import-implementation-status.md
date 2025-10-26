# Import Statement Implementation - Status Report

## Overview

This document summarizes the current state of import statement implementation for the Nomos compiler.

## Implemented Features

### 1. Import Syntax Support
- **Syntax**: `import:alias:path` where:
  - `alias` is the provider alias defined in a source declaration
  - `path` is the resource path to import (e.g., filename for file provider)
- **Example**:
  ```
  source:
    alias: 'files'
    type: 'file'
    baseDir: '.'
  
  import: files:base.csl
  ```

### 2. Import Extraction (`internal/imports` package)
- `ExtractImports(tree *ast.AST) ExtractedData` - extracts:
  - Source declarations from `ast.SourceDecl` nodes
  - Import statements from `ast.ImportStmt` nodes  
  - Section data from `ast.SectionDecl` nodes
- Proper path handling: file paths like `base.csl` are kept as single components
- Unit tests pass: `TestExtractImports`

### 3. Test Fixtures
- `testdata/imports/base.csl` - base configuration file
- `testdata/imports/override.csl` - file with source declaration and import
- `testdata/imports/expected.golden.json` - expected merge result
- Integration test skeleton in `providers/file/import_integration_test.go`

### 4. Data Structures
```go
type SourceDecl struct {
    Alias  string
    Type   string
    Config map[string]any
}

type ImportDecl struct {
    Alias string   // Provider alias
    Path  []string // Resource path
}

type ExtractedData struct {
    Sources []SourceDecl
    Imports []ImportDecl
    Data    map[string]any
}
```

## Known Limitations

### Critical: Dynamic Provider Creation Not Implemented

**Problem**: The compiler cannot dynamically create provider instances from source declarations found in .csl files.

**Root Cause**: The `ProviderRegistry` interface only supports:
- `Register(alias, constructor)` - requires a constructor function
- `GetProvider(alias)` - returns existing provider instance

It does NOT support:
- Registering provider *types* (e.g., "file" → FileProvider constructor)
- Creating providers from runtime configuration
- `RegisterType(typeName, constructor)` method

**Current Workaround**: Providers must be pre-registered before compilation:
```go
registry := compiler.NewProviderRegistry()
file.RegisterFileProvider(registry, "file", "./config")  // Pre-register
compiler.Compile(ctx, opts)  // Can now use "file" alias in .csl
```

**Impact**: Source declarations in .csl files are extracted but not acted upon.

### Architectural Change Needed

To fully support import statements, we need:

1. **Provider Type Registry**:
   ```go
   type ProviderTypeRegistry interface {
       RegisterType(typeName string, constructor TypeConstructor)
       CreateProvider(typeName string, config map[string]any) (Provider, error)
   }
   
   type TypeConstructor func(config map[string]any) (Provider, error)
   ```

2. **Integration in Compile Flow**:
   - Parse file → Extract source declarations
   - For each source declaration:
     - Get type constructor from type registry
     - Create provider instance with config
     - Initialize provider
     - Register instance in provider registry
   - Continue with import resolution

3. **Import Resolution Flow**:
   - Extract imports from AST
   - For each import:
     - Get provider by alias
     - Fetch data via provider.Fetch(path)
     - Merge into result (imports first, main file last)
   - Return merged data

## Next Steps

### Short Term (Minimal Viable Product)
1. Document the workaround: pre-register providers in application code
2. Update integration tests to demonstrate current functionality
3. Add architectural decision record (ADR) for future provider type registry

### Medium Term (Full Implementation)
1. Design provider type registry interface
2. Implement type registry with built-in provider types (file, http, etc.)
3. Update compiler to process source declarations and create providers
4. Implement full import resolution with cycle detection
5. Add comprehensive integration tests

### Long Term (Advanced Features)
1. Nested imports (imports within imported files)
2. Import path mapping (import specific sections: `import:files:base.csl:database`)
3. Import cycle detection across file boundaries
4. Import caching and optimization
5. Conditional imports based on variables

## Testing Strategy

### Unit Tests ✅
- `internal/imports/provider_resolver_test.go`:
  - `TestExtractImports` - validates extraction logic

### Integration Tests 🚧
- `providers/file/import_integration_test.go`:
  - `TestIntegration_ImportResolution` - end-to-end test (skipped pending implementation)

### Manual Testing
To test import extraction manually:
```bash
cd libs/compiler
go test -v ./internal/imports/...
```

Expected output:
```
=== RUN   TestExtractImports
--- PASS: TestExtractImports (0.00s)
PASS
```

## Files Changed

### New Files
- `internal/imports/provider_resolver.go` - import extraction and resolution logic
- `internal/imports/provider_resolver_test.go` - unit tests
- `providers/file/import_integration_test.go` - integration test (skipped)
- `testdata/imports/base.csl` - test fixture
- `testdata/imports/override.csl` - test fixture with import
- `testdata/imports/expected.golden.json` - expected result
- `import_test.go` - compiler-level test (skipped)

### Modified Files
- `CHANGELOG.md` - documented import infrastructure addition
- `import_test.go` - added skip note and architectural context

## References

- Parser README: `/libs/parser/README.md` - import syntax documentation
- File Provider: `/libs/compiler/providers/file/` - example provider implementation
- Compiler Architecture: `/docs/architecture/go-monorepo-structure.md`

## Questions for Review

1. Should we implement provider type registry now or defer to future version?
2. Is the pre-registration workaround acceptable for MVP?
3. Should import:alias (no path) default to a specific file or remain an error?
4. How should we handle import cycles - error immediately or allow with warning?

---

**Status**: 🟡 Partially Implemented  
**Blocker**: Provider type registry architectural dependency  
**Workaround**: Pre-register providers before compilation  
**Next Action**: Decision on provider type registry design
