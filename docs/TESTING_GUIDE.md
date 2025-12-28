# Testing Guide

This guide covers testing practices, patterns, and infrastructure for the Nomos project.

## Table of Contents

- [Test Organization](#test-organization)
- [Writing Tests](#writing-tests)
- [Module-Specific Testing](#module-specific-testing)
- [Running Tests](#running-tests)
- [Test Coverage Goals](#test-coverage-goals)
- [Best Practices](#best-practices)

---

## Test Organization

Nomos follows a structured approach to organizing tests by type and purpose.

### Unit Tests

**Location:** Same package as the code being tested  
**File naming:** `<filename>_test.go`  
**Build tag:** None (run by default)

Unit tests live alongside the code they test and run in the standard `go test` execution. They are fast, isolated, and form the foundation of our test suite.

**Example structure:**
```
libs/parser/
├── parser.go
├── parser_test.go              # Unit tests
├── pkg/
│   └── ast/
│       ├── types.go
│       └── types_test.go       # Unit tests
```

### Integration Tests

**Location:** 
- Module root (e.g., `libs/compiler/resolver_integration_test.go`)
- `test/` subdirectory (e.g., `libs/compiler/test/integration_test.go`)

**File naming:** `*_integration_test.go` or `*_test.go` in `test/` directory  
**Build tag:** `//go:build integration`

Integration tests verify interactions between components and external dependencies. They require the `integration` build tag and are excluded from default test runs.

**Example structure:**
```
libs/compiler/
├── compiler.go
├── compiler_test.go                        # Unit tests
├── resolver_integration_test.go            # Integration test (root)
├── metadata_integration_test.go            # Integration test (root)
└── test/
    ├── integration_test.go                 # Integration tests
    ├── integration_network_test.go         # Network integration tests
    └── e2e/
        └── smoke_test.go                   # E2E tests
```

**Build tag format:**
```go
//go:build integration
// +build integration

package compiler_test

import "testing"

func TestIntegrationScenario(t *testing.T) {
    // Integration test logic
}
```

### E2E (End-to-End) Tests

**Location:** `test/e2e/` at module or repo root  
**File naming:** `*_test.go`  
**Build tag:** `//go:build integration` (same as integration tests)

E2E tests verify complete workflows and user scenarios, typically at the CLI or API level.

**Example structure:**
```
apps/command-line/
└── test/
    ├── integration_test.go                 # CLI integration tests
    ├── determinism_integration_test.go     # Determinism tests
    ├── migration_integration_test.go       # Migration tests
    └── exitcode_integration_test.go        # Exit code behavior tests
```

### Test Utilities and Fixtures

**testutil/** - Reusable test helpers, fakes, and utilities  
**testdata/** - Test fixtures and data files

```
libs/compiler/
├── testutil/
│   ├── fake_provider_test.go               # Tests for fake provider
│   ├── provider.go                         # Fake provider implementation
│   ├── provider_registry.go                # Fake registry
│   └── file_provider.go                    # File-based provider for tests
└── testdata/
    ├── config1.csl                         # Test fixture
    ├── config2.csl                         # Test fixture
    └── expected_output.json                # Expected results

libs/parser/
└── testdata/
    ├── fixtures/                           # Input test files
    │   ├── simple.csl
    │   └── complex.csl
    ├── golden/                             # Expected output (golden files)
    │   ├── simple.csl.json
    │   └── complex.csl.json
    └── errors/                             # Error test cases
        ├── invalid_syntax.csl
        └── missing_colon.csl
```

---

## Writing Tests

### Table-Driven Test Pattern

Nomos uses table-driven tests as the standard pattern for unit tests. This approach provides clear, maintainable test coverage.

**Basic structure:**
```go
func TestFunction(t *testing.T) {
    tests := []struct {
        name     string  // Test case description
        input    Input   // Input values
        expected Output  // Expected result
    }{
        {
            name:     "descriptive case name",
            input:    /* input data */,
            expected: /* expected output */,
        },
        // More test cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Arrange - already done in table
            
            // Act
            result := Function(tt.input)
            
            // Assert
            if result != tt.expected {
                t.Errorf("got %v; want %v", result, tt.expected)
            }
        })
    }
}
```

**Real example from parser:**
```go
func TestReferenceExpr_Constructor(t *testing.T) {
    tests := []struct {
        name       string
        alias      string
        path       []string
        sourceSpan ast.SourceSpan
    }{
        {
            name:  "simple reference",
            alias: "network",
            path:  []string{"vpc", "cidr"},
            sourceSpan: ast.SourceSpan{
                Filename:  "test.csl",
                StartLine: 1,
                StartCol:  10,
                EndLine:   1,
                EndCol:    35,
            },
        },
        {
            name:  "single path component",
            alias: "config",
            path:  []string{"key"},
            sourceSpan: ast.SourceSpan{
                Filename:  "app.csl",
                StartLine: 5,
                StartCol:  5,
                EndLine:   5,
                EndCol:    20,
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ref := &ast.ReferenceExpr{
                Alias:      tt.alias,
                Path:       tt.path,
                SourceSpan: tt.sourceSpan,
            }

            if ref.Alias != tt.alias {
                t.Errorf("Alias = %v, want %v", ref.Alias, tt.alias)
            }
            // More assertions...
        })
    }
}
```

**Key benefits:**
- Clear test case names make failures easy to understand
- Easy to add new test cases
- Each case runs as a subtest with `t.Run()`
- Can run specific cases: `go test -run TestFunction/simple_reference`

### Test Fixture Organization

#### testdata/ Directory Structure

```
testdata/
├── fixtures/           # Input files for tests
│   ├── valid/
│   └── invalid/
├── golden/             # Expected outputs (golden files)
│   ├── valid/
│   └── errors/
└── <other>/            # Module-specific test data
```

#### Golden File Testing

Golden files store expected output for comparison. They're particularly useful for parsers, compilers, and formatters.

**Example from parser:**
```go
func TestGolden_ErrorScenarios(t *testing.T) {
    errorsDir := "../testdata/errors"
    goldenDir := "../testdata/golden/errors"

    entries, err := os.ReadDir(errorsDir)
    if err != nil {
        t.Fatalf("failed to read errors directory: %v", err)
    }

    for _, entry := range entries {
        if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".csl") {
            continue
        }

        name := entry.Name()
        t.Run(name, func(t *testing.T) {
            fixturePath := filepath.Join(errorsDir, name)
            goldenPath := filepath.Join(goldenDir, name+".error.json")

            // Parse (expecting error)
            _, err := parser.ParseFile(fixturePath)
            if err == nil {
                t.Fatalf("expected parse error for %s, got nil", name)
            }

            // Serialize error to JSON
            actualJSON := serializeError(err)

            // Compare with golden file
            expectedJSON, err := os.ReadFile(goldenPath)
            if err != nil {
                // Create golden file if it doesn't exist
                t.Logf("Creating golden file: %s", goldenPath)
                os.WriteFile(goldenPath, actualJSON, 0644)
                return
            }

            if !bytes.Equal(actualJSON, expectedJSON) {
                t.Errorf("output mismatch\nExpected:\n%s\nActual:\n%s",
                    expectedJSON, actualJSON)
            }
        })
    }
}
```

**Golden file workflow:**
1. Run test without golden file → test creates it
2. Review the generated golden file
3. Commit golden file to repository
4. Future test runs compare against committed golden file

**Updating golden files:**
```bash
# Delete golden files to regenerate
rm testdata/golden/*.json

# Run tests to regenerate
go test ./...

# Review changes
git diff testdata/golden/

# Commit if correct
git add testdata/golden/
git commit -m "test: update golden files"
```

### Using testutil/ Helpers

testutil packages provide reusable test utilities, fakes, and helpers.

**Example: Fake Provider (compiler testutil):**
```go
// testutil/provider.go
type FakeProvider struct {
    InitCount  int
    FetchCount int
    responses  map[string]any
}

func NewFakeProvider(alias string) *FakeProvider {
    return &FakeProvider{
        responses: make(map[string]any),
    }
}

func (p *FakeProvider) SetResponse(path string, value any) {
    p.responses[path] = value
}

func (p *FakeProvider) Fetch(ctx context.Context, path string) (any, error) {
    p.FetchCount++
    if val, ok := p.responses[path]; ok {
        return val, nil
    }
    return nil, ErrNotFound
}

// Usage in tests:
func TestCompiler_WithProvider(t *testing.T) {
    fake := testutil.NewFakeProvider("config")
    fake.SetResponse("db/host", "localhost")
    fake.SetResponse("db/port", 5432)

    registry := NewProviderRegistry()
    registry.Register("config", func(_ ProviderInitOptions) (Provider, error) {
        return fake, nil
    })

    // Use registry in compiler...
}
```

**Test helpers follow these conventions:**
- Use `t.Helper()` to mark helper functions
- Return error to caller instead of calling `t.Fatal()` in helpers
- Keep helpers focused and reusable

```go
func createTestFile(t *testing.T, dir, name, content string) string {
    t.Helper()
    
    path := filepath.Join(dir, name)
    if err := os.WriteFile(path, []byte(content), 0644); err != nil {
        t.Fatalf("failed to create test file: %v", err)
    }
    
    return path
}
```

### Benchmark Tests

Benchmark tests measure performance and identify regressions.

**Example from parser:**
```go
func BenchmarkParse_Small(b *testing.B) {
    source := `source:
  alias: myConfig
  type: yaml

database:
  host: localhost
  port: 5432
`

    b.ResetTimer() // Don't include setup time
    for i := 0; i < b.N; i++ {
        reader := bytes.NewReader([]byte(source))
        _, err := parser.Parse(reader, "test.csl")
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkParse_Large(b *testing.B) {
    // Generate large config
    var builder strings.Builder
    for i := 0; i < 1000; i++ {
        builder.WriteString(fmt.Sprintf("section%d:\n", i))
        builder.WriteString(fmt.Sprintf("  key: value%d\n", i))
    }
    source := builder.String()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        reader := bytes.NewReader([]byte(source))
        _, err := parser.Parse(reader, "bench.csl")
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

**Running benchmarks:**
```bash
# Run all benchmarks
go test -bench=. -benchmem

# Run specific benchmark
go test -bench=BenchmarkParse_Small -benchmem

# Compare before/after
go test -bench=. -benchmem > old.txt
# Make changes...
go test -bench=. -benchmem > new.txt
benchstat old.txt new.txt
```

---

## Module-Specific Testing

### Parser Testing

The parser module tests AST generation, error handling, and syntax validation.

#### AST Tests

Test that parsing produces correct AST structures.

```go
func TestParse_ReferenceExpression(t *testing.T) {
    input := `database:
  host: reference:config:db.host
`
    
    result, err := parser.Parse(strings.NewReader(input), "test.csl")
    if err != nil {
        t.Fatalf("parse failed: %v", err)
    }
    
    // Navigate AST
    dbSection := result.Data["database"].(map[string]any)
    hostValue := dbSection["host"]
    
    // Assert it's a reference
    ref, ok := hostValue.(*ast.ReferenceExpr)
    if !ok {
        t.Fatalf("expected ReferenceExpr, got %T", hostValue)
    }
    
    if ref.Alias != "config" {
        t.Errorf("alias = %q, want %q", ref.Alias, "config")
    }
    
    expectedPath := []string{"db", "host"}
    if !reflect.DeepEqual(ref.Path, expectedPath) {
        t.Errorf("path = %v, want %v", ref.Path, expectedPath)
    }
}
```

#### Error Format Tests

Test that parse errors include proper location information.

```go
func TestParse_InvalidSyntax_ReportsLocation(t *testing.T) {
    input := `config:
  key value  # Missing colon
`
    
    _, err := parser.Parse(strings.NewReader(input), "test.csl")
    if err == nil {
        t.Fatal("expected parse error, got nil")
    }
    
    // Check error has location info
    parseErr, ok := err.(interface {
        Line() int
        Column() int
    })
    if !ok {
        t.Fatalf("error doesn't implement location interface: %v", err)
    }
    
    if parseErr.Line() != 2 {
        t.Errorf("error line = %d, want 2", parseErr.Line())
    }
}
```

#### Golden File Testing

See [Golden File Testing](#golden-file-testing) section above for parser golden file patterns.

### Compiler Testing

The compiler module tests import resolution, reference resolution, and provider integration.

#### Using Fake Provider Registry

The testutil package provides fake providers for testing without external dependencies.

```go
func TestCompiler_ResolveReferences(t *testing.T) {
    // Create fake registry
    registry := testutil.NewFakeProviderRegistry()
    
    // Create fake provider
    fakeProvider := testutil.NewFakeProvider("config")
    fakeProvider.SetResponse("db/host", "localhost")
    fakeProvider.SetResponse("db/port", 5432)
    
    // Register provider
    registry.Register("config", func(_ compiler.ProviderInitOptions) (compiler.Provider, error) {
        return fakeProvider, nil
    })
    
    // Create compiler with fake registry
    comp := compiler.New(compiler.Options{
        ProviderRegistry: registry,
    })
    
    // Test compilation with references
    input := `database:
  host: reference:config:db/host
  port: reference:config:db/port
`
    
    result, err := comp.Compile(context.Background(), input)
    if err != nil {
        t.Fatalf("compile failed: %v", err)
    }
    
    // Verify references were resolved
    db := result.Data["database"].(map[string]any)
    if db["host"] != "localhost" {
        t.Errorf("host = %v, want localhost", db["host"])
    }
    if db["port"] != 5432 {
        t.Errorf("port = %v, want 5432", db["port"])
    }
}
```

#### Import Resolution Tests

Test import resolution with file-based providers.

```go
func TestCompiler_ImportResolution(t *testing.T) {
    tmpDir := t.TempDir()
    
    // Create base config
    baseConfig := `baseValue: 42`
    baseFile := filepath.Join(tmpDir, "base.csl")
    os.WriteFile(baseFile, []byte(baseConfig), 0644)
    
    // Create config that imports base
    mainConfig := fmt.Sprintf(`import:base:%s

derived:
  value: reference:base:baseValue
`, baseFile)
    
    // Compile with file provider
    comp := compiler.New(compiler.Options{
        FileProvider: testutil.NewFileProvider(),
    })
    
    result, err := comp.Compile(context.Background(), mainConfig)
    if err != nil {
        t.Fatalf("compile failed: %v", err)
    }
    
    // Verify import was resolved
    derived := result.Data["derived"].(map[string]any)
    if derived["value"] != 42 {
        t.Errorf("value = %v, want 42", derived["value"])
    }
}
```

#### Integration Tests

Integration tests verify end-to-end compilation with real providers.

```go
//go:build integration
// +build integration

package compiler

func TestResolveReferences_Integration(t *testing.T) {
    t.Run("resolves references in data", func(t *testing.T) {
        registry := NewProviderRegistry()
        registry.Register("config", func(_ ProviderInitOptions) (Provider, error) {
            return &testProvider{
                responses: map[string]any{
                    "db/host": "localhost",
                    "db/port": 5432,
                },
            }, nil
        })

        data := map[string]any{
            "database": map[string]any{
                "host": &ast.ReferenceExpr{
                    Alias: "config",
                    Path:  []string{"db", "host"},
                },
            },
        }

        resolved, err := pipeline.ResolveReferences(
            context.Background(),
            data,
            pipeline.ResolveOptions{
                ProviderRegistry: registry,
            },
        )

        if err != nil {
            t.Fatalf("resolution failed: %v", err)
        }

        db := resolved["database"].(map[string]any)
        if db["host"] != "localhost" {
            t.Errorf("host = %v, want localhost", db["host"])
        }
    })
}
```

### CLI Testing

The command-line app uses integration tests to verify CLI commands end-to-end.

#### Cobra Command Testing

```go
//go:build integration
// +build integration

package test

func TestBuildCommand_Integration(t *testing.T) {
    binPath := buildCLI(t)
    defer os.Remove(binPath)

    tests := []struct {
        name           string
        args           []string
        setupFixture   func(t *testing.T) string
        wantExitCode   int
        wantStdoutCont string
        wantStderrCont string
    }{
        {
            name:           "valid config",
            args:           []string{"build", "-p", "test.csl", "-f", "json"},
            setupFixture:   createBasicFixture,
            wantExitCode:   0,
            wantStdoutCont: "config",
        },
        {
            name: "help flag",
            args: []string{"build", "--help"},
            setupFixture: func(_ *testing.T) string {
                return ""
            },
            wantExitCode:   0,
            wantStdoutCont: "Build compiles",
        },
        {
            name: "missing required flag",
            args: []string{"build"},
            setupFixture: func(_ *testing.T) string {
                return ""
            },
            wantExitCode:   1,
            wantStderrCont: "required flag",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            fixturePath := ""
            if tt.setupFixture != nil {
                fixturePath = tt.setupFixture(t)
                if fixturePath != "" {
                    defer os.RemoveAll(filepath.Dir(fixturePath))
                }
            }

            cmd := exec.Command(binPath, tt.args...)
            var stdout, stderr bytes.Buffer
            cmd.Stdout = &stdout
            cmd.Stderr = &stderr

            err := cmd.Run()
            
            exitCode := 0
            if err != nil {
                if exitErr, ok := err.(*exec.ExitError); ok {
                    exitCode = exitErr.ExitCode()
                }
            }

            if exitCode != tt.wantExitCode {
                t.Errorf("exit code = %d, want %d", exitCode, tt.wantExitCode)
            }

            if tt.wantStdoutCont != "" && !strings.Contains(stdout.String(), tt.wantStdoutCont) {
                t.Errorf("stdout doesn't contain %q\nGot: %s", tt.wantStdoutCont, stdout.String())
            }

            if tt.wantStderrCont != "" && !strings.Contains(stderr.String(), tt.wantStderrCont) {
                t.Errorf("stderr doesn't contain %q\nGot: %s", tt.wantStderrCont, stderr.String())
            }
        })
    }
}

// buildCLI compiles the CLI binary for testing
func buildCLI(t *testing.T) string {
    t.Helper()
    
    binPath := filepath.Join(t.TempDir(), "nomos-test")
    cmd := exec.Command("go", "build", "-o", binPath, "../cmd/nomos")
    
    if output, err := cmd.CombinedOutput(); err != nil {
        t.Fatalf("failed to build CLI: %v\n%s", err, output)
    }
    
    return binPath
}

// createBasicFixture creates a minimal test fixture
func createBasicFixture(t *testing.T) string {
    t.Helper()
    
    tmpDir := t.TempDir()
    fixturePath := filepath.Join(tmpDir, "test.csl")
    
    content := `source:
  alias: test
  type: yaml

config:
  key: value
`
    
    if err := os.WriteFile(fixturePath, []byte(content), 0644); err != nil {
        t.Fatalf("failed to create fixture: %v", err)
    }
    
    return fixturePath
}
```

### Provider Module Testing

Provider modules (provider-downloader, provider-proto) have specialized tests.

#### Archive Extraction Tests

```go
func TestExtractArchive(t *testing.T) {
    tests := []struct {
        name        string
        archiveType string
        setupFunc   func(t *testing.T) string // Returns archive path
        wantFiles   []string
    }{
        {
            name:        "zip archive",
            archiveType: "zip",
            setupFunc:   createTestZip,
            wantFiles:   []string{"plugin.exe", "README.md"},
        },
        {
            name:        "tar.gz archive",
            archiveType: "tar.gz",
            setupFunc:   createTestTarGz,
            wantFiles:   []string{"plugin", "LICENSE"},
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            archivePath := tt.setupFunc(t)
            defer os.Remove(archivePath)

            destDir := t.TempDir()

            err := ExtractArchive(archivePath, destDir)
            if err != nil {
                t.Fatalf("extraction failed: %v", err)
            }

            // Verify extracted files
            for _, file := range tt.wantFiles {
                path := filepath.Join(destDir, file)
                if _, err := os.Stat(path); err != nil {
                    t.Errorf("expected file %q not found: %v", file, err)
                }
            }
        })
    }
}
```

#### gRPC Integration Tests

```go
//go:build integration
// +build integration

package provider_test

func TestProviderGRPC_Integration(t *testing.T) {
    // Start test gRPC server
    server, addr := startTestServer(t)
    defer server.Stop()

    // Create client
    conn, err := grpc.Dial(addr, grpc.WithInsecure())
    if err != nil {
        t.Fatalf("failed to connect: %v", err)
    }
    defer conn.Close()

    client := pb.NewProviderClient(conn)

    // Test RPC call
    ctx := context.Background()
    resp, err := client.GetSchema(ctx, &pb.GetSchemaRequest{})
    if err != nil {
        t.Fatalf("GetSchema failed: %v", err)
    }

    if resp.Schema == nil {
        t.Error("expected schema, got nil")
    }
}
```

---

## Running Tests

The Makefile provides convenient targets for running different types of tests.

### Common Test Commands

```bash
# Run all unit tests (default behavior)
make test

# Run unit tests only (explicitly exclude integration)
make test-unit

# Run with race detector
make test-race

# Run integration tests only
make test-integration

# Generate coverage reports (HTML)
make test-coverage

# Test specific module
make test-module MODULE=libs/parser
make test-module MODULE=libs/compiler

# Run integration tests for specific module
make test-integration-module MODULE=libs/compiler
```

### Direct Go Commands

```bash
# Run all tests in current module
go test ./...

# Run tests with verbose output
go test -v ./...

# Run specific test
go test -run TestCompiler_Basic

# Run specific test case in table
go test -run TestCompiler/simple_import

# Run integration tests
go test -tags=integration ./...

# Run with race detector
go test -race ./...

# Generate coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run benchmarks
go test -bench=. -benchmem

# Run tests in short mode (skip slow tests)
go test -short ./...
```

### CI Workflow Testing

The CI workflows run comprehensive test suites on each push/PR.

**Parser CI workflow:**
```yaml
- name: Run unit tests
  run: cd libs/parser && go test -v -race ./...

- name: Run integration tests
  run: cd libs/parser && go test -v -tags=integration ./...
```

**Compiler CI workflow:**
```yaml
- name: Run unit tests
  run: cd libs/compiler && go test -v -race ./...

- name: Check coverage
  run: |
    cd libs/compiler
    go test -coverprofile=coverage.out ./...
    COVERAGE=$(go tool cover -func=coverage.out | tail -1 | awk '{print $NF}' | sed 's/%//')
    echo "Coverage: $COVERAGE%"
```

**CLI CI workflow:**
```yaml
- name: Run unit tests
  run: cd apps/command-line && go test -v -race ./...

- name: Run integration tests
  run: cd apps/command-line && go test -v -tags=integration ./test/...

- name: Check coverage threshold (80%)
  run: |
    cd apps/command-line
    go test -coverprofile=coverage.out ./...
    COVERAGE=$(go tool cover -func=coverage.out | tail -1 | awk '{print $NF}' | sed 's/%//')
    COVERAGE_INT=$(echo $COVERAGE | cut -d'.' -f1)
    if [ $COVERAGE_INT -lt 80 ]; then
      echo "Coverage $COVERAGE% is below 80% threshold"
      exit 1
    fi
```

---

## Test Coverage Goals

Test coverage goals ensure adequate testing across the codebase.

### Current Coverage by Module

| Module                  | Current Coverage | Goal | Status |
|-------------------------|------------------|------|--------|
| **parser**              | 86.9%            | 80%  | ✅ Met  |
| **compiler**            | ~50%             | 80%  | 🟡 WIP  |
| **command-line**        | 80%+             | 80%  | ✅ Met  |
| **provider-downloader** | High             | 80%  | ✅ Met  |
| **provider-proto**      | Adequate         | 80%  | ✅ Met  |

### Coverage Requirements

**Minimum thresholds:**
- **80% overall coverage** for all modules
- **100% coverage** for critical business logic paths
- **New code** must include tests before merging
- **Bug fixes** must include regression tests

**Critical paths requiring 100% coverage:**
- Reference resolution logic
- Import cycle detection
- Provider initialization and caching
- Error handling and reporting
- CLI exit codes and error messages

### Checking Coverage

```bash
# Generate coverage report
make test-coverage

# View coverage in browser
open coverage/parser.html
open coverage/compiler.html

# Check coverage percentage
cd libs/compiler
go test -cover ./...

# Detailed function-level coverage
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# Find uncovered code
go tool cover -func=coverage.out | grep -v 100.0%
```

### Coverage in CI

CI enforces coverage thresholds:
- Fails if coverage drops below 80% for CLI module
- Reports coverage for all modules
- Tracks coverage trends over time

### Skipped Tests Documentation

When tests are intentionally skipped (deferred features), document them in `SKIPPED_TESTS.md`:

**Example from compiler module:**
```markdown
# Skipped Tests Status

## Current Skipped Tests

### 1. Import Cycle Detection Test (`import_test.go:66`)

**Status:** Deferred - Feature not yet implemented  
**Test:** `TestCompile_ImportCycle`  
**Reason:** Import cycle detection requires integration of the validator.DependencyGraph 
with import resolution. This is tracked in a GitHub issue and will be implemented when 
import resolution is enhanced.

**Implementation Requirements:**
- Integrate `internal/validator` cycle detection with `internal/imports` resolution
- Add cycle path tracking across import chains
- Enhance error reporting with full import cycle path

**Expected Implementation:** Phase 4 (Compiler Refactoring)
```

**Usage in code:**
```go
func TestCompile_ImportCycle(t *testing.T) {
    t.Skip("Deferred: Import cycle detection not yet implemented. See SKIPPED_TESTS.md")
    
    // Test code here...
}
```

---

## Best Practices

### Test Naming Conventions

**Test files:**
- Unit tests: `<filename>_test.go`
- Integration tests: `<filename>_integration_test.go` or in `test/` directory
- Benchmark tests: `<filename>_bench_test.go` or within `<filename>_test.go`

**Test functions:**
```go
// Unit test
func TestFunctionName(t *testing.T) { }

// Table-driven with descriptive case names
func TestFunctionName(t *testing.T) {
    tests := []struct {
        name string
        // ...
    }{
        {name: "valid input returns success"},
        {name: "empty input returns error"},
        {name: "nil value panics"},
    }
}

// Integration test
//go:build integration

func TestIntegration_FeatureName(t *testing.T) { }

// Benchmark
func BenchmarkFunctionName(b *testing.B) { }
```

### Test Isolation and Cleanup

**Use `t.TempDir()` for temporary files:**
```go
func TestFileOperations(t *testing.T) {
    tmpDir := t.TempDir() // Automatically cleaned up
    
    testFile := filepath.Join(tmpDir, "test.txt")
    // Use testFile...
}
```

**Clean up resources with defer:**
```go
func TestDatabaseConnection(t *testing.T) {
    db, err := setupTestDB()
    if err != nil {
        t.Fatalf("setup failed: %v", err)
    }
    defer db.Close() // Always cleanup
    
    // Test with db...
}
```

**Isolate tests from each other:**
- Don't share state between tests
- Each test should be runnable independently
- Don't rely on test execution order
- Use subtests for logical grouping

### Avoiding Test Interdependencies

**❌ Bad - Tests depend on execution order:**
```go
var sharedState string

func TestA(t *testing.T) {
    sharedState = "value"
}

func TestB(t *testing.T) {
    // Assumes TestA ran first
    if sharedState != "value" {
        t.Fail()
    }
}
```

**✅ Good - Tests are independent:**
```go
func TestA(t *testing.T) {
    state := "value"
    // Test with state...
}

func TestB(t *testing.T) {
    state := "value" // Set up own state
    // Test with state...
}
```

### Using Subtests Effectively

**Organize related tests:**
```go
func TestUserValidation(t *testing.T) {
    t.Run("email validation", func(t *testing.T) {
        t.Run("valid email", func(t *testing.T) {
            // Test valid email
        })
        
        t.Run("invalid email", func(t *testing.T) {
            // Test invalid email
        })
    })
    
    t.Run("password validation", func(t *testing.T) {
        t.Run("strong password", func(t *testing.T) {
            // Test strong password
        })
        
        t.Run("weak password", func(t *testing.T) {
            // Test weak password
        })
    })
}
```

**Run specific subtests:**
```bash
# Run all email validation tests
go test -run TestUserValidation/email

# Run specific subtest
go test -run TestUserValidation/email/valid

# Run all validation tests
go test -run TestUserValidation
```

### Test Failure Messages

**Use clear failure messages:**
```go
// ❌ Bad
if result != expected {
    t.Fail()
}

// ✅ Good
if result != expected {
    t.Errorf("got %v; want %v", result, expected)
}

// ✅ Better - with context
if result != expected {
    t.Errorf("ProcessData(%v) = %v; want %v", input, result, expected)
}
```

**For complex assertions, show details:**
```go
if !reflect.DeepEqual(got, want) {
    t.Errorf("result mismatch\nGot:  %+v\nWant: %+v", got, want)
}
```

### Helper Functions

**Mark helpers with `t.Helper()`:**
```go
func assertNoError(t *testing.T, err error) {
    t.Helper() // Makes test failure point to caller
    
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
}

func createTestUser(t *testing.T, name string) *User {
    t.Helper()
    
    user := &User{Name: name}
    if err := user.Validate(); err != nil {
        t.Fatalf("failed to create test user: %v", err)
    }
    
    return user
}
```

### Parallel Test Execution

**Run independent tests in parallel:**
```go
func TestParallel(t *testing.T) {
    tests := []struct {
        name  string
        input int
    }{
        {"case 1", 1},
        {"case 2", 2},
        {"case 3", 3},
    }
    
    for _, tt := range tests {
        tt := tt // Capture range variable
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel() // Run in parallel
            
            result := SlowFunction(tt.input)
            // Assertions...
        })
    }
}
```

**Note:** Must capture range variable when using `t.Parallel()`

### Testing Error Cases

**Always test error paths:**
```go
func TestDivide(t *testing.T) {
    tests := []struct {
        name      string
        a, b      float64
        want      float64
        wantErr   bool
        errContains string
    }{
        {
            name: "normal division",
            a:    10,
            b:    2,
            want: 5,
        },
        {
            name:        "divide by zero",
            a:           10,
            b:           0,
            wantErr:     true,
            errContains: "division by zero",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Divide(tt.a, tt.b)
            
            if tt.wantErr {
                if err == nil {
                    t.Fatal("expected error, got nil")
                }
                if !strings.Contains(err.Error(), tt.errContains) {
                    t.Errorf("error = %q, want to contain %q", err, tt.errContains)
                }
                return
            }
            
            if err != nil {
                t.Fatalf("unexpected error: %v", err)
            }
            
            if got != tt.want {
                t.Errorf("got %v; want %v", got, tt.want)
            }
        })
    }
}
```

---

## Summary

This guide provides comprehensive coverage of testing practices in the Nomos project:

1. **Test Organization** - Clear separation of unit, integration, and E2E tests
2. **Writing Tests** - Table-driven patterns, golden files, fixtures, and benchmarks
3. **Module Testing** - Specific patterns for parser, compiler, CLI, and providers
4. **Running Tests** - Makefile targets and direct commands
5. **Coverage Goals** - 80% minimum with critical path requirements
6. **Best Practices** - Naming, isolation, helpers, and error testing

For questions or improvements to this guide, please open an issue or submit a PR.

**Related Documentation:**
- [CONTRIBUTING.md](../CONTRIBUTING.md) - Development workflow and standards
- [CODING_STANDARDS.md](CODING_STANDARDS.md) - Code style and conventions
- Module-specific READMEs in each `libs/` and `apps/` directory
