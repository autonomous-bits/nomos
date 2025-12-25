# Nomos Parser Agent-Specific Patterns

> **Note**: For comprehensive parser development guidance, see `.github/agents/parser-module.agent.md`  
> For task coordination, start with `.github/agents/nomos.agent.md`

## Nomos-Specific Patterns

### Test Data Organization

The Nomos parser uses a structured testdata directory for different test types:

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

### Performance Benchmarks

The parser includes benchmarks specific to Nomos configuration file characteristics:

**Benchmark Suite** (`parser_bench_test.go`):
- `BenchmarkParse_Small` - ~100 bytes input (minimal config)
- `BenchmarkParse_Medium` - ~6KB input (100 sections)
- `BenchmarkParse_Large` - ~1MB input (simulated large configuration)
- `BenchmarkParseFile` - Filesystem I/O overhead for .csl files
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

### Build Tags

Currently, the Nomos parser module does not use custom build tags. All tests run in standard mode.

### Error Message Patterns

Nomos parser errors follow a specific format with actionable guidance:

**Error Structure**:
```
<filename>:<line>:<col>: <kind>: <concise description>

<context line with error>
    ^
<detailed explanation or hint>
```

**Error Kinds**:
1. **LexError** - Tokenization failures (invalid characters, unterminated strings)
2. **SyntaxError** - Grammar violations, malformed structures
3. **IOError** - File system or I/O failures

**Nomos-Specific Error Messages**:
- Top-level `reference:` statements include migration hint to inline references
- `source` declarations require non-empty string `alias` field validation
- Keywords `source` and `import` must be followed by `:` with helpful error

**Error Formatting**:
Use `FormatParseError(err, sourceText)` to generate multi-line output with context and caret marker (rune-aware for UTF-8).

### Nomos Language Constructs

The parser implements these Nomos-specific language features:

**Source Declarations** - Provider configuration:
```
source:
  alias: myProvider
  type: terraform
```

**Import Statements** - Module imports:
```
import:baseConfig:./base.csl
```

**Section Declarations** - Configuration blocks:
```
database:
  host: localhost
  port: 5432
```

**Inline References** - Value-level references:
```
app:
  db_host: reference:base:database.host
```

### Deprecated Syntax Handling

**Deprecated**: Top-level `reference:` statements are no longer supported.

**Parser Behavior**: Returns `SyntaxError` with migration hint when encountering:
```
reference:alias:path.to.value
```

**Migration Path**: Convert to inline references in value positions (see "Inline References" above).

### Coverage HTML Report

The module includes a checked-in `coverage.html` file for visual coverage inspection. This is regenerated during CI or manually via:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

---

## Task Completion Verification

**MANDATORY**: Before completing ANY task, the agent MUST verify all of the following:

### 1. Build Verification ✅
```bash
go build ./...
```
- All code must compile without errors
- No unresolved imports or type errors

### 2. Test Verification ✅
```bash
go test ./...
go test ./... -race  # Check for race conditions
```
- All existing tests must pass
- New tests must be added for new functionality
- Race detector must report no data races
- Minimum coverage targets must be maintained

### 3. Linting Verification ✅
```bash
go vet ./...
golangci-lint run
```
- No `go vet` warnings
- No golangci-lint errors (warnings are acceptable if documented)
- Code follows Go best practices

### 4. Integration Test Verification ✅
```bash
go test ./test/integration/... -v
```
- All integration tests must pass
- Parser correctly handles all test fixtures

### 5. Documentation Updates ✅
- Update CHANGELOG.md if behavior changed
- Update README.md if API changed
- Add/update code comments for new functions
- Update examples if syntax changed

### Verification Checklist Template

When completing a task, report:
```
✅ Build: Successful
✅ Tests: XX/XX passed (YY.Y% coverage)
✅ Race Detector: Clean
✅ Linting: Clean (or list acceptable warnings)
✅ Integration Tests: All passed
✅ Documentation: Updated [list files]
```

**DO NOT** mark a task as complete without running ALL verification steps and reporting results.
