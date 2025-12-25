---
name: parser-module
description: Specialized agent for libs/parser - handles parsing Nomos configuration scripts into AST, maintaining parser logic, error handling, and ensuring high performance.
---

## Module Context
- **Path**: `libs/parser`
- **Responsibilities**: 
  - Lexing and parsing Nomos configuration scripts into AST
  - Providing stable AST data structures for compiler consumption
  - Rich error reporting with source location and context
  - Parser performance optimization
- **Key Files**:
  - `parser.go` - Main parser implementation with public API
  - `errors.go` - Parse error types and formatting utilities
  - `pkg/ast/types.go` - AST node definitions (exported)
  - `internal/scanner/` - Lexical analysis (tokenization)
  - `internal/testutil/` - Testing utilities for AST comparison
  - `testdata/fixtures/` - Test input files (.csl)
  - `testdata/golden/` - Expected AST outputs (JSON)
  - `testdata/errors/` - Error case test data
  - `parser_bench_test.go` - Performance benchmarks
  - `test/` - Integration and API tests
- **Test Pattern**: 
  - Golden file testing (parse → serialize to JSON → compare with golden)
  - Integration tests in `test/` package (`parser_test`)
  - Unit tests alongside implementation files
  - Error validation with expected error messages
  - Benchmark tests for performance validation

## Delegation Instructions
For general Go questions, **consult go-expert.agent.md**  
For testing questions, **consult testing-expert.agent.md**  
For compiler integration questions, **consult compiler-module.agent.md**

## Parser-Specific Patterns

### Public API Surface

The parser exports a minimal, dependency-light API:

```go
// Top-level convenience functions
ParseFile(path string) (*ast.AST, error)
Parse(r io.Reader, filename string) (*ast.AST, error)

// Instance-based API (for pooling/reuse)
NewParser(opts ...ParserOption) *Parser
(*Parser).ParseFile(path string) (*ast.AST, error)
(*Parser).Parse(r io.Reader, filename string) (*ast.AST, error)
```

**Best Practice**: Parser instances can be reused via `sync.Pool` for better performance in high-throughput scenarios.

### AST Node Types

All AST nodes implement the `Node` interface with `Span() SourceSpan` method.

**Statement Types** (`ast.Stmt`):
- `SourceDecl` - Source provider declaration with alias, type, and configuration
- `ImportStmt` - Import statement with alias and optional path
- `SectionDecl` - Configuration section with name and key-value entries
- `ReferenceStmt` - **DEPRECATED**: Top-level references (parser rejects these)

**Expression Types** (`ast.Expr`):
- `StringLiteral` - Plain string values (quotes stripped by parser)
- `ReferenceExpr` - Inline reference values: `reference:alias:dotted.path`

All nodes carry `SourceSpan` with filename, line, and column information.

### Error Handling

Parse errors are represented by `*ParseError` with three kinds:

1. **LexError** - Tokenization failures (invalid characters, unterminated strings)
2. **SyntaxError** - Grammar violations, malformed structures
3. **IOError** - File system or I/O failures

Error structure includes:
- `Filename`, `Line` (1-indexed), `Column` (1-indexed)
- `Message` - Human-friendly description
- `Snippet` - Optional context lines with caret marker

**Error Formatting**: Use `FormatParseError(err, sourceText)` to generate multi-line output with context and caret marker (rune-aware for UTF-8).

### Inline References

**Current Syntax**: References are first-class expressions used in value positions:
```
key: reference:someAlias:parent.child.value
```

Produces: `ast.ReferenceExpr{Alias: "someAlias", Path: []string{"parent", "child", "value"}}`

**Deprecated**: Top-level `reference:` statements are **rejected** by the parser with a migration hint.

### Parser Validation Rules

The parser enforces **syntax-level validation**:

✅ **Enforced by Parser**:
- Keywords `source` and `import` must be followed by `:`
- `source` declarations require non-empty string `alias` field
- `import` requires an alias; optional `:path` may follow
- String values must be properly terminated
- Key identifiers must be valid (non-empty, valid start character)
- Top-level `reference:` statements are rejected

❌ **NOT Enforced by Parser** (deferred to compiler):
- Duplicate key detection (requires scope-aware analysis)
- Unknown provider types
- Import resolution
- Reference resolution
- Semantic validation

### Test Data Organization

```
testdata/
├── fixtures/       # Input .csl files for testing
│   ├── simple.csl
│   ├── references.csl
│   └── nested.csl
├── golden/         # Expected AST outputs (JSON)
│   ├── simple.csl.json
│   └── ...
└── errors/         # Error test cases
    ├── invalid_syntax.csl
    └── ...
```

**Golden Test Pattern**:
1. Parse input file from `fixtures/`
2. Serialize AST to canonical JSON using `testutil.CanonicalJSON()`
3. Compare with golden file in `golden/`
4. Auto-generate golden file if missing (requires manual verification)

**Error Test Pattern**:
- Test files in `errors/` directory
- Assertions on `ParseError.Kind()`, message content, and location
- Verify error messages are actionable and user-friendly

### Performance Considerations

**Benchmarking Suite** (`parser_bench_test.go`):
- `BenchmarkParse_Small` - ~100 bytes input
- `BenchmarkParse_Medium` - ~6KB input (100 sections)
- `BenchmarkParse_Large` - ~1MB input (simulated large config)
- `BenchmarkParseFile` - Filesystem I/O overhead
- `BenchmarkParser_Reuse` - Instance reuse with `sync.Pool`

**Performance Goals**:
- Small files (<1KB): Sub-millisecond parsing
- Medium files (~10KB): <5ms parsing
- Large files (~1MB): <100ms parsing
- Parser instance reuse reduces allocation overhead

**Run Benchmarks**:
```bash
cd libs/parser
go test -bench=. -benchmem -run=^$ ./...
```

### Module Dependencies

**No dependencies on other Nomos modules** - foundational library

**Used By**:
- `libs/compiler` - Consumes AST for compilation
- `apps/command-line` - CLI uses parser via compiler

**External Dependencies**:
- Minimal: Standard library only
- No parser generators currently used (hand-written parser)

### Development Workflow

**Run Tests**:
```bash
cd libs/parser
go test ./...              # All tests
go test ./test -v          # Integration tests
go test -run=TestGolden    # Golden tests only
```

**Run Benchmarks**:
```bash
go test -bench=. -benchmem ./...
```

**Coverage**:
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

**Update Golden Files**:
Golden files auto-generate on first run when missing. To regenerate:
1. Delete existing golden file
2. Run test (generates new golden)
3. Manually verify AST structure is correct
4. Commit updated golden file

## Common Tasks

### 1. Adding New Syntax to Nomos Language

**Steps**:
1. Update scanner in `internal/scanner/` to recognize new tokens
2. Extend AST types in `pkg/ast/types.go` with new node types
3. Implement parsing logic in `parser.go`
4. Add validation rules if syntax-level checks are needed
5. Create test fixtures in `testdata/fixtures/`
6. Add golden tests for new syntax
7. Update error messages if new failure modes exist
8. Run benchmarks to ensure no performance regression

**Example Checklist**:
- [ ] Token recognition added
- [ ] AST node defined with `Span()` method
- [ ] Parse function implemented
- [ ] Test fixture created
- [ ] Golden test added
- [ ] Error case tested
- [ ] README.md AST section updated
- [ ] Benchmarks pass without regression

### 2. Improving Parse Error Messages

**Guidelines**:
- Include source location (filename, line, column)
- Provide actionable guidance ("expected X, found Y")
- Add migration hints for deprecated syntax
- Use `FormatParseError()` for consistent formatting
- Test error messages in error test suite

**Error Message Template**:
```
<filename>:<line>:<col>: <kind>: <concise description>

<context line with error>
    ^
<detailed explanation or hint>
```

### 3. Performance Optimization

**Profiling Workflow**:
```bash
go test -bench=BenchmarkParse_Large -cpuprofile=cpu.prof
go tool pprof cpu.prof
```

**Common Optimizations**:
- Reduce allocations in hot paths (use `sync.Pool` for buffers)
- Avoid repeated string conversions
- Optimize scanner token buffering
- Profile memory allocations (`-memprofile`)
- Benchmark before and after changes

**Performance Regression Check**:
Always run full benchmark suite before committing:
```bash
go test -bench=. -benchmem -run=^$ | tee bench_before.txt
# Make changes
go test -bench=. -benchmem -run=^$ | tee bench_after.txt
benchstat bench_before.txt bench_after.txt
```

### 4. Maintaining Test Coverage

**Coverage Targets**:
- Parser core logic: >90% coverage
- Error paths: All error kinds tested
- Edge cases: Documented in test comments

**Coverage Review**:
```bash
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | grep -E "parser.go|scanner.go"
```

**Missing Coverage**:
Focus on:
- Scanner edge cases (unusual token sequences)
- Error recovery paths
- Boundary conditions (empty files, malformed input)

## Nomos-Specific Details

### Nomos Language Features

The parser implements the following Nomos language constructs:

1. **Source Declarations** - Provider configuration:
   ```
   source:
     alias: myProvider
     type: terraform
   ```

2. **Import Statements** - Module imports:
   ```
   import:baseConfig:./base.csl
   ```

3. **Section Declarations** - Configuration blocks:
   ```
   database:
     host: localhost
     port: 5432
   ```

4. **Inline References** - Value-level references:
   ```
   app:
     db_host: reference:base:database.host
   ```

### Migration from Legacy Syntax

**Deprecated**: Top-level `reference:` statements are no longer supported.

**Parser Behavior**: Returns `SyntaxError` with migration hint when encountering:
```
reference:alias:path.to.value
```

**Migration Path**: Convert to inline references in value positions (see "Inline References" section above).

### Parser Design Philosophy

- **Separation of Concerns**: Lexical analysis in `internal/scanner`, syntax in `parser.go`
- **Minimal API Surface**: Export only what compiler needs, hide implementation
- **Stable AST**: AST types are considered public API contract
- **Deterministic Errors**: Same input always produces same error
- **Performance by Design**: Hand-written parser for predictable performance

### Future Enhancements

Potential areas for expansion (not currently implemented):

- [ ] Complex expression types (arrays, maps, interpolation)
- [ ] Duplicate key detection (requires scope analysis)
- [ ] Syntax-directed error recovery for better multi-error reporting
- [ ] Incremental parsing for editor/LSP use cases
- [ ] Parser generator adoption (ANTLR, pigeon) if grammar becomes complex

---

## Quick Reference

**Parse a file**:
```go
ast, err := parser.ParseFile("config.csl")
```

**Parse from reader**:
```go
ast, err := parser.Parse(reader, "config.csl")
```

**Handle errors**:
```go
if pe, ok := err.(*parser.ParseError); ok {
    fmt.Println(parser.FormatParseError(pe, sourceText))
}
```

**Reuse parser instances**:
```go
p := parser.NewParser()
ast1, _ := p.ParseFile("file1.csl")
ast2, _ := p.ParseFile("file2.csl")
```

## See Also

- `AGENTS.md` - Module-level development guidance
- `README.md` - Complete parser documentation
- `docs/architecture/` - High-level design documents
- `pkg/ast/types.go` - AST node reference
