# Nomos Coding Standards

This document outlines the coding standards and patterns used in the Nomos project. These standards have evolved through practical experience during the codebase refactoring (Phases 1-5) and reflect Go best practices tailored to our domain.

> **Note:** These standards build upon general Go conventions. For comprehensive Go guidance, see `.github/agents/go-expert.agent.md`.

## Table of Contents
1. [Error Handling Patterns](#error-handling-patterns)
2. [Testing Patterns](#testing-patterns)
3. [Code Organization](#code-organization)
4. [Go Idioms](#go-idioms)
5. [Documentation](#documentation)

---

## Error Handling Patterns

### Sentinel Errors

**Define sentinel errors as package-level variables** for conditions that callers need to identify:

```go
package compiler

import "errors"

// ErrImportResolutionNotAvailable is returned when import resolution cannot proceed
// due to missing dependencies (e.g., no ProviderTypeRegistry).
var ErrImportResolutionNotAvailable = errors.New("import resolution not available: ProviderTypeRegistry required")

// ErrUnresolvedReference indicates a reference could not be resolved.
var ErrUnresolvedReference = errors.New("unresolved reference")

// ErrCycleDetected indicates a cycle was detected in imports or references.
var ErrCycleDetected = errors.New("cycle detected")

// ErrProviderNotRegistered indicates a provider alias is not registered.
var ErrProviderNotRegistered = errors.New("provider not registered")
```

**Group related sentinel errors** together with clear documentation comments:

```go
// Sentinel errors for provider and compilation failures.
var (
    ErrUnresolvedReference   = errors.New("unresolved reference")
    ErrCycleDetected         = errors.New("cycle detected")
    ErrProviderNotRegistered = errors.New("provider not registered")
)
```

**Check for sentinel errors** using `errors.Is()`:

```go
func processFile(path string, opts Options) error {
    data, err := resolveFileImports(ctx, path, opts)
    if err != nil {
        // Allow graceful degradation when import resolution is not available
        if errors.Is(err, ErrImportResolutionNotAvailable) {
            // Continue without import resolution
            return compileWithoutImports(path)
        }
        return fmt.Errorf("failed to process file %s: %w", path, err)
    }
    return nil
}
```

### Error Wrapping vs Direct Return

**Wrap errors with context** when adding information about the operation:

```go
// ✅ Good - Adds context about the operation
func saveUser(user *User) error {
    if err := db.Save(user); err != nil {
        return fmt.Errorf("failed to save user %s: %w", user.ID, err)
    }
    return nil
}

func createProvider(alias string, constructor ProviderConstructor) (Provider, error) {
    provider, err := constructor()
    if err != nil {
        return nil, fmt.Errorf("failed to construct provider %q: %w", alias, err)
    }
    
    if err := provider.Init(ctx, config); err != nil {
        return nil, fmt.Errorf("failed to initialize provider %q: %w", alias, err)
    }
    
    return provider, nil
}
```

**Return sentinel errors directly** when no additional context is needed:

```go
// ✅ Good - Sentinel error is self-explanatory
func resolveFileImports(ctx context.Context, filePath string, opts Options) (map[string]any, error) {
    if opts.ProviderTypeRegistry == nil {
        return nil, ErrImportResolutionNotAvailable
    }
    // ...
}
```

**Wrap sentinel errors** when you need to add specific context:

```go
// ✅ Good - Adds context to sentinel error
func GetProviderInstance(alias string) (Provider, error) {
    instance, exists := r.instances[alias]
    if !exists {
        return nil, fmt.Errorf("%w: %s", ErrProviderNotRegistered, alias)
    }
    return instance, nil
}
```

### Error Message Formatting

**Follow these conventions** for error messages:

1. **Lowercase, no punctuation** (Go convention):
   ```go
   ✅ errors.New("connection failed")
   ✅ errors.New("invalid email format")
   ❌ errors.New("Connection failed.")
   ❌ errors.New("Invalid email format!")
   ```

2. **Add specific context** when wrapping:
   ```go
   ✅ fmt.Errorf("failed to parse file %s at line %d: %w", filename, line, err)
   ✅ fmt.Errorf("provider fetch timeout for %q after %s: %w", providerName, timeout, err)
   ❌ fmt.Errorf("error: %w", err)
   ❌ fmt.Errorf("something went wrong: %w", err)
   ```

3. **Be specific and actionable**:
   ```go
   ✅ errors.New("config file not found at /etc/nomos/config.csl")
   ✅ errors.New("invalid syntax: expected closing brace at line 42")
   ❌ errors.New("bad input")
   ❌ errors.New("processing failed")
   ```

4. **Don't expose sensitive details** in user-facing errors:
   ```go
   ✅ errors.New("authentication failed")
   ❌ fmt.Errorf("user %s not found in database", email)
   ```

### CompilationResult for Multi-Error Collection

**Use `CompilationResult`** to collect multiple errors without failing fast:

```go
// CompilationResult wraps a Snapshot and provides convenience methods
// for checking compilation status.
type CompilationResult struct {
    Snapshot Snapshot
}

func (r CompilationResult) HasErrors() bool {
    return len(r.Snapshot.Metadata.Errors) > 0
}

func (r CompilationResult) HasWarnings() bool {
    return len(r.Snapshot.Metadata.Warnings) > 0
}

func (r CompilationResult) Error() error {
    if !r.HasErrors() {
        return nil
    }
    if len(r.Snapshot.Metadata.Errors) == 1 {
        return errors.New(r.Snapshot.Metadata.Errors[0])
    }
    return fmt.Errorf("compilation failed with %d errors: %v",
        len(r.Snapshot.Metadata.Errors),
        r.Snapshot.Metadata.Errors)
}
```

**Accumulate errors during compilation**:

```go
func Compile(ctx context.Context, opts Options) CompilationResult {
    result := CompilationResult{
        Snapshot: Snapshot{
            Metadata: Metadata{
                Errors:   []string{},
                Warnings: []string{},
            },
        },
    }
    
    // Validate options - fail fast for critical errors
    if ctx == nil {
        result.Snapshot.Metadata.Errors = append(result.Snapshot.Metadata.Errors, 
            "context must not be nil")
        return result
    }
    
    // Continue processing, accumulating errors
    for _, file := range files {
        if err := processFile(file); err != nil {
            result.Snapshot.Metadata.Errors = append(result.Snapshot.Metadata.Errors,
                fmt.Sprintf("file %s: %v", file, err))
            continue // Don't fail fast - collect all errors
        }
    }
    
    return result
}
```

**Check errors in calling code**:

```go
result := compiler.Compile(ctx, opts)
if result.HasErrors() {
    for _, err := range result.Errors() {
        log.Printf("compilation error: %s", err)
    }
    return result.Error()
}

if result.HasWarnings() {
    for _, warn := range result.Warnings() {
        log.Printf("compilation warning: %s", warn)
    }
}
```

### Never Panic in Library Code

**Library code must never panic** - always return errors:

```go
// ✅ Good - Returns error
func GetProvider(alias string) (Provider, error) {
    if alias == "" {
        return nil, errors.New("alias must not be empty")
    }
    // ...
}

// ❌ Bad - Panics
func GetProvider(alias string) Provider {
    if alias == "" {
        panic("alias must not be empty")
    }
    // ...
}
```

**Panic only in `main` or `init`** for unrecoverable startup errors:

```go
// ✅ Acceptable in main
func main() {
    config, err := loadConfig()
    if err != nil {
        panic(fmt.Errorf("failed to load config: %w", err))
    }
    // ...
}
```

---

## Testing Patterns

### Table-Driven Tests

**Prefer table-driven tests** for testing multiple scenarios:

```go
func TestCompile_OptionsValidation(t *testing.T) {
    tests := []struct {
        name        string
        ctx         context.Context
        opts        compiler.Options
        expectError string
    }{
        {
            name:        "nil context",
            ctx:         nil,
            opts:        compiler.Options{Path: "/some/path", ProviderRegistry: testutil.NewFakeProviderRegistry()},
            expectError: "context must not be nil",
        },
        {
            name:        "empty Path",
            ctx:         context.Background(),
            opts:        compiler.Options{Path: "", ProviderRegistry: testutil.NewFakeProviderRegistry()},
            expectError: "options.Path must not be empty",
        },
        {
            name:        "nil ProviderRegistry",
            ctx:         context.Background(),
            opts:        compiler.Options{Path: "/some/path", ProviderRegistry: nil},
            expectError: "options.ProviderRegistry must not be nil",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := compiler.Compile(tt.ctx, tt.opts)
            if !result.HasErrors() {
                t.Fatalf("expected error, got nil")
            }
            if result.Error().Error() != tt.expectError {
                t.Errorf("expected error %q, got %q", tt.expectError, result.Error().Error())
            }
        })
    }
}
```

**Structure test cases** with clear field names:

```go
tests := []struct {
    name           string  // Test case description
    input          string  // Input parameters
    expectedOutput string  // Expected result
    expectError    bool    // Whether an error is expected
    errorContains  string  // Substring to check in error message
}{
    {
        name:           "valid input",
        input:          "test.csl",
        expectedOutput: "compiled successfully",
        expectError:    false,
    },
    {
        name:          "invalid syntax",
        input:         "invalid.csl",
        expectError:   true,
        errorContains: "syntax error",
    },
}
```

### Integration Test Tags

**Use build tags** to separate integration tests:

```go
//go:build integration
// +build integration

package compiler_test

import "testing"

func TestIntegration_RemoteProviders(t *testing.T) {
    // Test that requires:
    // - Network access
    // - External services
    // - File system operations
    // - End-to-end workflows
}
```

**When to use integration tags**:

- **Network calls**: Tests making HTTP/gRPC requests
- **External services**: Tests requiring databases, APIs, etc.
- **File system**: Tests that modify the filesystem extensively
- **End-to-end**: Full workflow tests involving multiple components
- **Slow tests**: Tests that take >1 second

**Run integration tests**:

```bash
# Run all tests including integration
make test

# Run only integration tests
make test-integration

# Run only unit tests (excludes integration)
make test-unit

# Run integration tests for specific module
make test-integration-module MODULE=libs/compiler
```

### Test Organization

**Organize tests by scope**:

```
libs/compiler/
├── compiler.go              # Implementation
├── compiler_test.go         # Unit tests (same package)
├── test/                    # Integration tests
│   └── integration_network_test.go  # //go:build integration
└── testdata/                # Test fixtures
    ├── imports/
    ├── merge_semantics/
    └── validation/
```

**Unit tests** live alongside the code in the same package:
```go
package compiler_test  // or package compiler for white-box testing

func TestCompile_BasicScenario(t *testing.T) {
    // Fast, focused test
}
```

**Integration tests** live in `test/` subdirectory:
```go
//go:build integration

package test

func TestIntegration_FullWorkflow(t *testing.T) {
    // Slower, broader test
}
```

### Testdata Organization

**Structure testdata** by feature or scenario:

```
testdata/
├── imports/                    # Import resolution test files
│   ├── simple/
│   │   ├── source.csl
│   │   └── target.csl
│   └── nested/
│       ├── a.csl
│       └── b.csl
├── merge_semantics/            # Merge behavior test files
│   ├── basic_override.csl
│   └── deep_merge.csl
└── validation/                 # Validation test cases
    ├── invalid_syntax.csl
    └── missing_field.csl
```

**Reference testdata** with clear paths:

```go
func TestImportResolution_SimpleImport(t *testing.T) {
    testDir := filepath.Join("testdata", "imports", "simple")
    sourceFile := filepath.Join(testDir, "source.csl")
    
    result := compiler.Compile(ctx, compiler.Options{
        Path: sourceFile,
        // ...
    })
    // ...
}
```

**Use `t.TempDir()`** for tests that need to write files:

```go
func TestCompile_DeterministicDirectoryTraversal(t *testing.T) {
    tmpDir := t.TempDir() // Automatically cleaned up
    
    // Create test files
    files := []string{"z.csl", "a.csl", "m.csl"}
    for _, f := range files {
        path := filepath.Join(tmpDir, f)
        if err := writeFile(path, "test: true"); err != nil {
            t.Fatalf("failed to create test file: %v", err)
        }
    }
    
    // Test with temp directory
    result := compiler.Compile(ctx, compiler.Options{Path: tmpDir, ...})
    // ...
}
```

### Mock vs Fake Implementations

**Use fakes for most test scenarios** (simplified real implementations):

```go
// Fake implementation in testutil package
package testutil

type FakeProviderRegistry struct {
    providers map[string]compiler.Provider
}

func NewFakeProviderRegistry() *FakeProviderRegistry {
    return &FakeProviderRegistry{
        providers: make(map[string]compiler.Provider),
    }
}

func (r *FakeProviderRegistry) RegisterProvider(alias string, provider compiler.Provider) {
    r.providers[alias] = provider
}

func (r *FakeProviderRegistry) GetProviderInstance(ctx context.Context, alias string) (compiler.Provider, error) {
    provider, exists := r.providers[alias]
    if !exists {
        return nil, fmt.Errorf("provider not found: %s", alias)
    }
    return provider, nil
}
```

**Use mocks for behavior verification** (when you need to assert calls):

```go
// Mock with call tracking
type MockProvider struct {
    InitCalled  bool
    FetchCalled bool
    FetchData   map[string]any
}

func (m *MockProvider) Init(ctx context.Context, opts compiler.ProviderInitOptions) error {
    m.InitCalled = true
    return nil
}

func (m *MockProvider) Fetch(ctx context.Context, config map[string]any) (map[string]any, error) {
    m.FetchCalled = true
    return m.FetchData, nil
}

func TestProvider_InitializesCalled(t *testing.T) {
    mock := &MockProvider{FetchData: map[string]any{"key": "value"}}
    
    // Use the mock
    result := processWithProvider(mock)
    
    // Verify behavior
    if !mock.InitCalled {
        t.Error("expected Init to be called")
    }
    if !mock.FetchCalled {
        t.Error("expected Fetch to be called")
    }
}
```

**Guidelines**:
- **Fakes**: Use for most tests - simpler, easier to maintain
- **Mocks**: Use when you need to verify specific calls or call order
- **Place test utilities** in `testutil/` package for reuse

---

## Code Organization

### Package Structure

**Follow the standard Go project layout**:

```
nomos/
├── cmd/                      # Command-line applications
│   └── nomos/
│       └── main.go
├── internal/                 # Private application code (compiler-enforced)
│   ├── server/
│   ├── auth/
│   └── ...
├── pkg/                      # Public library code (for external use)
│   └── client/
├── libs/                     # Monorepo: Independent library modules
│   ├── compiler/
│   ├── parser/
│   └── provider-downloader/
└── apps/                     # Monorepo: Application modules
    └── command-line/
```

**Use `internal/` for private code**:
- Code in `internal/` cannot be imported by external projects (compiler-enforced)
- Most application code should live in `internal/`
- Prevents accidental API exposure

```go
// ✅ Can import within same module
import "github.com/autonomous-bits/nomos/libs/compiler/internal/parse"

// ❌ Cannot import from external module
import "github.com/other-project/nomos/libs/compiler/internal/parse" // Compile error
```

**Use `pkg/` for public APIs**:
- Only code intended for external consumption
- Stable, documented interfaces
- Subject to semantic versioning

### When to Extract Helpers

**Extract helpers when**:

1. **Code is duplicated** across multiple functions/packages:
   ```go
   // ✅ Good - Extracted helper
   func writeFile(path, content string) error {
       file, err := os.Create(path)
       if err != nil {
           return err
       }
       defer file.Close()
       _, err = file.WriteString(content)
       return err
   }
   ```

2. **Logic is complex** and deserves its own function:
   ```go
   // ✅ Good - Complex logic extracted
   func validateProviderConfig(config map[string]any) error {
       // 20+ lines of validation logic
   }
   ```

3. **Testing would benefit** from isolation:
   ```go
   // ✅ Good - Can test independently
   func parseTimeout(s string) (time.Duration, error) {
       // Timeout parsing logic
   }
   ```

**Don't extract when**:
- Used only once
- Logic is trivial (1-2 lines)
- Would require many parameters (keep logic inline)

### Interface Design Principles

**Keep interfaces small** - prefer single-method interfaces when possible:

```go
// ✅ Good - Small, focused interfaces
type Provider interface {
    Fetch(ctx context.Context, config map[string]any) (map[string]any, error)
}

type ProviderWithInit interface {
    Provider
    Init(ctx context.Context, opts ProviderInitOptions) error
}
```

**Define interfaces at point of use** (not at point of implementation):

```go
// ✅ Good - Interface defined where it's used
package compiler

type ProviderRegistry interface {
    GetProviderInstance(ctx context.Context, alias string) (Provider, error)
}

func Compile(ctx context.Context, opts Options) CompilationResult {
    // Uses ProviderRegistry interface
}
```

```go
// Implementation in same or different package
package mypkg

type MyRegistry struct {
    // ...
}

func (r *MyRegistry) GetProviderInstance(ctx context.Context, alias string) (compiler.Provider, error) {
    // Implementation - no explicit "implements" needed
}
```

**Accept interfaces, return concrete types**:

```go
// ✅ Good
func NewClient(registry ProviderRegistry) *Client {
    return &Client{registry: registry}
}

// ❌ Bad - Forces callers to work with interface
func NewClient(registry ProviderRegistry) ProviderRegistry {
    return &Client{registry: registry}
}
```

### Context Propagation

**Always pass `context.Context` as the first parameter**:

```go
func ProcessFile(ctx context.Context, path string, opts Options) error {
    // ...
}

func (p *Provider) Fetch(ctx context.Context, config map[string]any) (map[string]any, error) {
    // ...
}
```

**Check for context cancellation** in long-running operations:

```go
func ProcessFiles(ctx context.Context, files []string) error {
    for _, file := range files {
        // Check for cancellation
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }
        
        if err := processFile(ctx, file); err != nil {
            return err
        }
    }
    return nil
}
```

**Propagate context to downstream calls**:

```go
func Compile(ctx context.Context, opts Options) CompilationResult {
    // Pass context through
    data, err := fetchProviderData(ctx, opts.ProviderRegistry)
    if err != nil {
        // ...
    }
    
    result, err := processData(ctx, data)
    // ...
}
```

**Use context for timeouts and cancellation**:

```go
func FetchWithTimeout(ctx context.Context, provider Provider, timeout time.Duration) (map[string]any, error) {
    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()
    
    return provider.Fetch(ctx, config)
}
```

---

## Go Idioms

### Prefer Stdlib Over Custom Utilities

**Use standard library when available**:

```go
// ✅ Good - Use strings package
import "strings"
result := strings.TrimSpace(input)

// ❌ Bad - Custom utility
result := util.Trim(input)
```

**Examples of stdlib over custom**:

- `strings`, `strconv`, `bytes` - String/byte manipulation
- `path/filepath` - Path operations (not custom path utils)
- `encoding/json` - JSON handling
- `errors` - Error handling (`errors.Is`, `errors.As`)
- `context` - Cancellation and timeouts
- `sync` - Concurrency primitives

**When to write custom utilities**:
- Domain-specific logic
- Repeated complex operations
- Wrapping stdlib with domain context

### Error Handling - No Silent Failures

**Always check and handle errors explicitly**:

```go
// ✅ Good
result, err := doSomething()
if err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}

// ❌ Bad - Silent failure
result, _ := doSomething()

// ❌ Bad - Ignored error
doSomething()
```

**Handle errors at the appropriate level**:

```go
// ✅ Good - Handle at appropriate level
func ProcessFile(path string) error {
    data, err := readFile(path)
    if err != nil {
        // Log and return - let caller decide
        log.Printf("failed to read file %s: %v", path, err)
        return fmt.Errorf("process file: %w", err)
    }
    // ...
}

// ❌ Bad - Swallow error
func ProcessFile(path string) {
    data, err := readFile(path)
    if err != nil {
        log.Printf("error: %v", err)
        return // Error lost
    }
    // ...
}
```

### Resource Cleanup

**Use `defer` for cleanup** - runs even if function panics:

```go
func processFile(path string) error {
    file, err := os.Open(path)
    if err != nil {
        return err
    }
    defer file.Close() // Always closes, even on error
    
    // Process file
    return nil
}
```

**Defer runs in LIFO order** (last defer runs first):

```go
func transaction() error {
    tx, err := db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback() // Runs second (if not committed)
    
    if err := doWork(tx); err != nil {
        return err
    }
    
    defer log.Println("committed") // Runs first
    return tx.Commit()
}
```

**Handle defer errors** when cleanup can fail:

```go
func writeFile(path string, data []byte) (err error) {
    file, err := os.Create(path)
    if err != nil {
        return err
    }
    defer func() {
        if cerr := file.Close(); cerr != nil && err == nil {
            err = cerr // Propagate close error if no other error
        }
    }()
    
    _, err = file.Write(data)
    return err
}
```

### Concurrency Patterns

**Use goroutines with synchronization**:

```go
func processItemsConcurrently(items []Item) error {
    var wg sync.WaitGroup
    errChan := make(chan error, len(items))
    
    for _, item := range items {
        wg.Add(1)
        go func(item Item) {
            defer wg.Done()
            if err := processItem(item); err != nil {
                errChan <- err
            }
        }(item)
    }
    
    wg.Wait()
    close(errChan)
    
    // Collect errors
    for err := range errChan {
        if err != nil {
            return err
        }
    }
    return nil
}
```

**Use context for cancellation**:

```go
func worker(ctx context.Context, jobs <-chan Job) {
    for {
        select {
        case <-ctx.Done():
            return // Exit on cancellation
        case job := <-jobs:
            processJob(job)
        }
    }
}

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    jobs := make(chan Job)
    go worker(ctx, jobs)
    
    // Work...
    
    cancel() // Stop workers
}
```

**Bounded concurrency** with semaphore pattern:

```go
func processWithLimit(items []Item, maxConcurrent int) error {
    sem := make(chan struct{}, maxConcurrent)
    var wg sync.WaitGroup
    
    for _, item := range items {
        wg.Add(1)
        go func(item Item) {
            defer wg.Done()
            
            sem <- struct{}{}        // Acquire
            defer func() { <-sem }() // Release
            
            processItem(item)
        }(item)
    }
    
    wg.Wait()
    return nil
}
```

---

## Documentation

### Doc Comments

**All exported identifiers must have doc comments**:

```go
// Package compiler provides compilation functionality for Nomos configuration scripts.
//
// The compiler transforms Nomos .csl source files into deterministic, serializable
// configuration snapshots. It integrates with the parser library for syntax analysis,
// supports pluggable provider adapters for external data sources, and enforces
// deterministic composition semantics with deep-merge and last-wins behavior.
package compiler

// Options configures a compilation run.
type Options struct {
    // Path specifies the input file or directory to compile.
    Path string
    
    // ProviderRegistry provides access to external data sources.
    ProviderRegistry ProviderRegistry
}

// Compile transforms Nomos configuration source into a snapshot.
// It returns a CompilationResult which may contain errors in the metadata
// even if compilation completes.
func Compile(ctx context.Context, opts Options) CompilationResult {
    // ...
}
```

**Doc comment conventions**:

1. **Complete sentences** starting with the identifier name:
   ```go
   // ✅ Good
   // Process validates and saves the user data.
   func Process(user *User) error

   // ❌ Bad
   // validates and saves user data
   func Process(user *User) error
   ```

2. **Explain *what* and *why*, not *how***:
   ```go
   // ✅ Good
   // ErrImportResolutionNotAvailable is returned when import resolution cannot proceed
   // due to missing dependencies (e.g., no ProviderTypeRegistry).
   var ErrImportResolutionNotAvailable = errors.New("...")
   
   // ❌ Bad
   // ErrImportResolutionNotAvailable creates a new error with this text
   var ErrImportResolutionNotAvailable = errors.New("...")
   ```

3. **Document behavior, constraints, and side effects**:
   ```go
   // resolveFileImports processes a single file's imports and returns merged data.
   // Returns ErrImportResolutionNotAvailable if the file has no type registry
   // for dynamic provider creation.
   func resolveFileImports(ctx context.Context, filePath string, opts Options) (map[string]any, error)
   ```

### CHANGELOG.md Requirements

**Follow [Keep a Changelog](https://keepachangelog.com/) format**:

```markdown
# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- New `ProviderTypeRegistry` interface for dynamic provider creation

### Changed
- Refactored `Compile()` to return `CompilationResult` instead of raw snapshot
- Import resolution now uses `resolveFileImports()` helper

### Fixed
- Fixed race condition in provider initialization

### Deprecated
- `CompileWithProviders()` - use `Compile()` with Options.ProviderRegistry

### Removed
- Legacy `CompileFile()` function - use `Compile()` with file path

### Security
- Added timeout support for provider fetch operations

## [0.2.0] - 2025-01-15

### Added
- Provider timeout configuration
- Concurrent provider fetch support
```

**Update CHANGELOG.md** for every significant change:
- Before merging PR
- Group changes by type (Added, Changed, Fixed, etc.)
- Keep Unreleased section at top
- Move to versioned section on release

### Commit Message Format

**Use Conventional Commits with gitmoji**:

```
<type>(<scope>): <gitmoji> <description>

[optional body]

[optional footer]
```

**Common types and gitmojis**:

- `feat(compiler): ✨ add import resolution support`
- `fix(parser): 🐛 handle nested references correctly`
- `docs(readme): 📝 add provider examples`
- `refactor(compiler): ♻️ extract validation logic`
- `test(compiler): 🧪 add integration tests for providers`
- `perf(parser): ⚡️ optimize AST traversal`
- `chore(deps): 🔧 update dependencies`

**Breaking changes**:

```
feat(compiler)!: ✨ change Compile signature to return CompilationResult

BREAKING CHANGE: Compile() now returns CompilationResult instead of (Snapshot, error).
Update callers to use result.Snapshot and result.Error().
```

**Examples**:

```
feat(compiler): ✨ add ProviderTypeRegistry support

Enables dynamic provider creation from source declarations.
Introduces ErrImportResolutionNotAvailable for graceful degradation.

Closes #123
```

```
fix(compiler): 🐛 prevent race in provider initialization

Added mutex around provider instance creation map.
```

```
refactor(compiler)!: ♻️ reorganize internal packages

BREAKING CHANGE: Moved internal packages to internal/ directory.
External projects cannot import these anymore (compiler-enforced).
```

**Full mapping** in `.github/instructions/commit-messages.instructions.md`.

---

## Summary

These coding standards represent proven patterns from the Nomos codebase:

1. **Error Handling**: Use sentinel errors, wrap with context, leverage CompilationResult
2. **Testing**: Table-driven tests, integration tags, testdata organization
3. **Code Organization**: Standard layout, internal/ privacy, small interfaces
4. **Go Idioms**: Stdlib preference, explicit errors, defer cleanup, context propagation
5. **Documentation**: Doc comments on exports, CHANGELOG updates, conventional commits

**When in doubt**:
- Look at existing code in the same module
- Prefer simplicity over cleverness
- Write tests first (or alongside) implementation
- Document the "why", not the "what"

For additional Go guidance, see `.github/agents/go-expert.agent.md`.
