# Nomos Parser Project Patterns

## Purpose

This document contains **Nomos-specific patterns** for the Parser module, providing context for development tasks including:
- Parser implementation patterns (AST design, scanner architecture)
- Testing conventions (golden file tests, fuzzing, benchmarks)
- Design constraints (lexer/parser separation)
- Error message formatting and handling

## Nomos-Specific Patterns

### Test Data Organization

The Nomos parser uses a structured testdata directory for different test types:

```
testdata/
├── fixtures/       # Input .csl files for testing
│   ├── simple.csl
│   ├── references.csl
│   ├── nested.csl
│   └── lists/      # List-specific test inputs
│       ├── simple_list.csl
│       ├── nested_lists.csl
│       ├── empty_list.csl
│       ├── list_with_objects.csl
│       └── list_in_reference.csl
├── golden/         # Expected AST outputs (JSON)
│   ├── simple.csl.json
│   ├── lists/      # Expected AST for list tests
│   │   ├── simple_list.csl.json
│   │   ├── nested_lists.csl.json
│   │   └── ...
│   └── ...
└── errors/         # Error test cases
    ├── invalid_syntax.csl
    ├── lists/      # List-specific error tests
    │   ├── empty_item.csl
    │   ├── inconsistent_indent.csl
    │   ├── tab_character.csl
    │   ├── whitespace_only.csl
    │   └── depth_exceeded.csl
    └── ...
```

**Directory Structure for List Tests**:
- `testdata/fixtures/lists/` - Valid and edge-case list syntax inputs
- `testdata/golden/lists/` - Canonical AST JSON for list tests
- `testdata/errors/lists/` - Invalid list syntax that should produce errors

**Golden Test Pattern**:
1. Parse input file from `fixtures/`
2. Serialize AST to canonical JSON using `testutil.CanonicalJSON()`
3. Compare with golden file in `golden/`
4. Auto-generate golden file if missing (requires manual verification)

**Error Test Pattern**:
- Test files in `errors/` directory
- Assertions on `ParseError.Kind()`, message content, and location
- Verify error messages are actionable and user-friendly

#### List Parsing Test Patterns

**List Golden Tests** - Test AST generation for valid list syntax:

```go
// Test simple list parsing
func TestParse_Lists_Simple(t *testing.T) {
    input := readFixture("testdata/fixtures/lists/simple_list.csl")
    ast, err := parser.Parse(input)
    require.NoError(t, err)
    
    actual := testutil.CanonicalJSON(ast)
    golden := readGolden("testdata/golden/lists/simple_list.csl.json")
    assert.JSONEq(t, golden, actual)
}
```

**List Error Tests** - Validate error handling for invalid list syntax:

```go
// Test empty list item rejection
func TestParse_Lists_EmptyItem_Error(t *testing.T) {
    input := readFixture("testdata/errors/lists/empty_item.csl")
    _, err := parser.Parse(input)
    require.Error(t, err)
    
    parseErr := err.(parser.ParseError)
    assert.Equal(t, parser.SyntaxError, parseErr.Kind())
    assert.Contains(t, parseErr.Message(), "empty list item")
    assert.Contains(t, parseErr.Message(), "use [] for empty lists")
}
```

**List Integration Tests** - End-to-end parsing workflows:

```go
// Test parser + scanner cooperation for list syntax
func TestParse_Lists_Integration(t *testing.T) {
    tests := []struct {
        name    string
        fixture string
        verify  func(*testing.T, *ast.Node)
    }{
        {
            name:    "nested lists",
            fixture: "testdata/fixtures/lists/nested_lists.csl",
            verify:  verifyNestedListStructure,
        },
        {
            name:    "lists with objects",
            fixture: "testdata/fixtures/lists/list_with_objects.csl",
            verify:  verifyMixedListStructure,
        },
    }
    // Run integration tests...
}
```

**List Test File Naming Conventions**:
- `simple_list.csl` - Basic list with scalar values
- `nested_lists.csl` - Lists containing other lists (2-3 levels)
- `empty_list.csl` - Empty list using `[]` notation
- `list_with_objects.csl` - Lists containing object/section items
- `list_in_reference.csl` - Lists used in reference expressions
- `deep_nesting.csl` - Maximum depth testing (19-21 levels)
- `empty_item.csl` - Error case: `- ` with no value
- `inconsistent_indent.csl` - Error case: non-2-space indentation
- `tab_character.csl` - Error case: tabs instead of spaces
- `whitespace_only.csl` - Error case: list with only whitespace
- `depth_exceeded.csl` - Error case: >20 levels of nesting

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

#### List Validation Error Patterns

**List Indentation Errors** - 2-space indentation strictly enforced:
```
list_items.csl:5:3: syntax error: inconsistent list item indentation

    -   value
    ^
Expected 2-space indentation for list items, got 4 spaces.
```

**Empty List Item Errors** - Empty items must use `[]` notation:
```
list_items.csl:8:5: syntax error: empty list item not allowed

  - 
    ^
List items cannot be empty. Use [] for an empty list.
```

**Tab Character Errors** - Tabs are not allowed in list indentation:
```
list_items.csl:3:1: syntax error: tab character in indentation

	- item
^
Indentation must use spaces only. Replace tabs with 2 spaces per level.
```

**Whitespace-Only List Errors** - Lists must contain actual values:
```
list_items.csl:10:1: syntax error: list contains only whitespace

  - 
  -   
^
List must contain at least one non-whitespace value or use [].
```

**Depth Limit Errors** - Maximum nesting depth of 20 levels:
```
deep_list.csl:45:41: syntax error: list nesting depth exceeded

                                        - item
                                        ^
Maximum list nesting depth is 20 levels. Current depth: 21.
Simplify your data structure or split into multiple sections.
```

**List Validation Requirements**:
- 2-space indentation per level (no tabs, no 4-space, no mixed)
- Maximum nesting depth: 20 levels
- Empty list items rejected (use `[]` for empty lists)
- Error messages include: line number, column number, context line, actionable guidance

### Nomos Language Constructs

The parser implements these Nomos-specific language features:

**Source Declarations** - Provider configuration:
```
source:
  alias: myProvider
  type: terraform
```

**Import Statements** - Removed from the language (parser rejects `import:` with migration guidance)

**Section Declarations** - Configuration blocks:
```
database:
  host: localhost
  port: 5432
```

**Inline References** - Value-level references:
```
app:
  db_host: @base:database:host
```

**List/Array Syntax** - YAML-style list support:
```
# Simple list with scalars
servers:
  - web01
  - web02
  - web03

# Nested lists
matrix:
  - - 1
    - 2
  - - 3
    - 4

# Empty list
tags: []

# List with objects
users:
  - name: alice
    role: admin
  - name: bob
    role: user

# Lists in references
backup_servers: @aws:database:replicas
```

**List Parsing Rules**:
- List items begin with `- ` (hyphen + space)
- 2-space indentation strictly enforced (no tabs)
- Maximum nesting depth: 20 levels
- Empty lists use `[]` notation
- Empty list items (`- ` with no value) are not allowed
- List items can be: scalars (strings, numbers, booleans), nested lists, or objects/sections

### List Test Coverage Requirements

**Required Test Scenarios** - All list implementations must include tests for:

1. **Simple Lists** (`testdata/fixtures/lists/simple_list.csl`):
   - List of strings
   - List of numbers
   - List of booleans
   - Mixed scalar types

2. **Nested Lists** (`testdata/fixtures/lists/nested_lists.csl`):
   - 2-level nesting (list of lists)
   - 3-level nesting (list of lists of lists)
   - Mixed nesting with scalars at different depths

3. **Empty Lists** (`testdata/fixtures/lists/empty_list.csl`):
   - Using `[]` notation
   - Empty list as value
   - Empty list in nested structure

4. **Lists with Objects** (`testdata/fixtures/lists/list_with_objects.csl`):
   - List items that are sections/objects
   - Mixed list (objects and scalars)
   - Objects with nested lists

5. **Lists in References** (`testdata/fixtures/lists/list_in_reference.csl`):
   - Reference to entire list
   - Reference to list item by index
   - List containing references

6. **Maximum Depth** (`testdata/fixtures/lists/deep_nesting.csl`):
   - 19 levels (should pass)
   - 20 levels (should pass - at limit)
   - 21 levels (should fail - exceeds limit)

**Required Error Test Scenarios** - Must validate rejection of:

1. **Empty Items** (`testdata/errors/lists/empty_item.csl`):
   ```
   items:
     - value1
     - 
     - value2
   ```
   Error: "empty list item not allowed"

2. **Inconsistent Indentation** (`testdata/errors/lists/inconsistent_indent.csl`):
   ```
   items:
     - item1
       - nested  # Wrong: should be 2 spaces from parent
   ```
   Error: "inconsistent list item indentation"

3. **Tab Characters** (`testdata/errors/lists/tab_character.csl`):
   ```
   items:
   	- item  # Tab character used
   ```
   Error: "tab character in indentation"

4. **Whitespace-Only Lists** (`testdata/errors/lists/whitespace_only.csl`):
   ```
   items:
     - 
     -   
   ```
   Error: "list contains only whitespace"

5. **Depth Exceeded** (`testdata/errors/lists/depth_exceeded.csl`):
   - 21+ levels of nesting
   Error: "list nesting depth exceeded"

**Golden File Format** - AST representation in JSON:
```json
{
  "type": "Section",
  "key": "servers",
  "value": {
    "type": "List",
    "items": [
      {"type": "String", "value": "web01"},
      {"type": "String", "value": "web02"}
    ]
  }
}
```

**Integration Test Requirements**:
- Parser and scanner must cooperate correctly on list token boundaries
- List parsing must work in combination with other language features (imports, references, sources)
- List AST nodes must serialize/deserialize correctly
- List error recovery must not corrupt subsequent parsing

### Deprecated Syntax Handling

**Deprecated**: Top-level `reference:` statements are no longer supported (User Story 1 - Breaking Change).

**AST Changes**:
- `ReferenceStmt` type removed from AST in Phase 2
- Top-level references were never used in practice and added unnecessary complexity
- Parser now rejects this syntax at parse time

**Parser Behavior**: Returns `SyntaxError` with migration hint when encountering:
```
reference:alias:path.to.value
```

**Error Message Format**:
```
invalid syntax: top-level reference statements are no longer supported

Detected: "reference:alias:path"

Migration Guide:
Top-level reference: statements must be converted to inline reference expressions in value positions.

Before (deprecated):
  reference:alias:path

After (correct):
  key: @alias:path
```

**Migration Path**: Convert to inline references in value positions (see "Inline References" above).

**Test Coverage**:
- `test/deprecated_reference_test.go` - Comprehensive rejection tests
- `test/parser_grammar_test.go` - Basic rejection tests
- `testdata/errors/deprecated_reference.csl` - Test fixture with multiple scenarios
- All tests verify error kind, message content, and migration guidance

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
