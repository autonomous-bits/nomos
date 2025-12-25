# Testing Expert Agent

## Purpose
Provides comprehensive testing expertise for Go projects, covering unit tests, integration tests, table-driven patterns, test organization, benchmarking, test coverage, mocking, and testing best practices. This agent serves as the authoritative reference for all testing-related decisions in Go development.

## Standards Source
- https://github.com/autonomous-bits/development-standards/blob/main/go/tests.md
- https://github.com/autonomous-bits/development-standards/blob/main/go_practices/table_driven_testing.md
- https://github.com/autonomous-bits/development-standards/blob/main/quality_culture/test_pyramid_strategy.md
- https://github.com/autonomous-bits/development-standards/blob/main/quality_culture/testability_through_interface_design.md

Last synced: 2025-12-25

## Coverage Areas
1. **Table-Driven Test Patterns** - Core Go testing pattern with subtests
2. **Test Organization & Naming** - File structure and naming conventions
3. **Test Coverage** - Coverage targets, tools, and analysis
4. **Integration Testing** - Build tags, TestContainers, HTTP tests
5. **Benchmark Testing** - Performance testing and profiling
6. **Test Fixtures** - Setup, teardown, and TestMain usage
7. **Mocking & Stubbing** - Interface-based mocking and testify/mock
8. **Test Helpers** - t.Helper() and utility functions
9. **Build Tags** - Separating unit and integration tests

## Content

### Testing Principles

#### Test-Driven Development (TDD)
- Write tests before implementing functionality when feasible
- Tests document expected behavior
- Red-Green-Refactor cycle
- Keep tests simple and focused

#### Test Pyramid
Follow the test pyramid for balanced test coverage:
- **Unit Tests (70%)**: Fast, isolated tests for individual functions/packages
- **Integration Tests (20%)**: Test interactions between components
- **End-to-End Tests (10%)**: Test complete workflows

**Benefits:**
- Provides immediate feedback for unit tests
- Pinpoint exact location of failures
- Can run on every commit
- Encourages good design through testability

**Avoid Inverted Pyramid:**
- Slow feedback (minutes to hours)
- High flakiness
- Difficult to maintain
- Poor defect localization
- Expensive CI/CD infrastructure

### Table-Driven Test Patterns

#### Core Pattern
Table-driven testing is the canonical Go pattern for unit testing. Define test cases as a table and iterate through each:

```go
func TestFunction(t *testing.T) {
    tests := []struct {
        name     string  // Test case name
        input    Input   // Input values
        expected Output  // Expected result
    }{
        // Test cases here
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test logic
        })
    }
}
```

#### Basic Example

```go
func TestAdd(t *testing.T) {
    tests := []struct {
        name     string
        a        int
        b        int
        expected int
    }{
        {
            name:     "positive numbers",
            a:        2,
            b:        3,
            expected: 5,
        },
        {
            name:     "negative numbers",
            a:        -2,
            b:        -3,
            expected: -5,
        },
        {
            name:     "mixed signs",
            a:        5,
            b:        -3,
            expected: 2,
        },
        {
            name:     "zero values",
            a:        0,
            b:        0,
            expected: 0,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := Add(tt.a, tt.b)
            if result != tt.expected {
                t.Errorf("Add(%d, %d) = %d; want %d", 
                    tt.a, tt.b, result, tt.expected)
            }
        })
    }
}
```

#### Using t.Run() for Subtests

**Benefits:**
- **Clear output:** Shows which specific case failed
- **Focused execution:** Run single test case with `-run` flag
- **Better organization:** Groups related assertions
- **Parallel execution:** Can run subtests in parallel

**Running Specific Subtest:**
```bash
# Run all tests
go test -v

# Run only "divide by zero" case
go test -v -run TestDivide/divide_by_zero

# Run all TestDivide cases with "zero" in name
go test -v -run TestDivide/zero
```

#### Testing with Multiple Return Values

```go
func TestDivide(t *testing.T) {
    tests := []struct {
        name      string
        a, b      float64
        expected  float64
        expectErr bool
    }{
        {
            name:     "normal division",
            a:        10,
            b:        2,
            expected: 5,
        },
        {
            name:      "divide by zero",
            a:         10,
            b:         0,
            expectErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := Divide(tt.a, tt.b)
            
            if tt.expectErr {
                if err == nil {
                    t.Error("Expected error but got nil")
                }
                return
            }
            
            if err != nil {
                t.Errorf("Unexpected error: %v", err)
            }
            
            if result != tt.expected {
                t.Errorf("Divide(%f, %f) = %f; want %f",
                    tt.a, tt.b, result, tt.expected)
            }
        })
    }
}
```

#### Helpful Failure Messages

**Standard Format: "got X, want Y"**

```go
func TestFunction(t *testing.T) {
    got := Function(input)
    want := expected
    
    if got != want {
        t.Errorf("Function(%v) = %v; want %v", input, got, want)
        // Or simply:
        t.Errorf("got %v; want %v", got, want)
    }
}
```

**Why This Format?**
- **Clear:** Immediately shows actual vs. expected
- **Standard:** Recognized by all Go developers
- **Debuggable:** Provides all info needed to investigate
- **Assumption:** Person debugging is not you

### Test Organization & Naming

#### File Organization
```
project/
├── user.go
├── user_test.go          # Tests for user.go
├── service/
│   ├── user_service.go
│   └── user_service_test.go
```

#### Naming Conventions
- **Test file**: `[filename]_test.go`
- **Test function**: `Test[FunctionName]_[Scenario]`
- **Benchmark function**: `Benchmark[FunctionName]`
- **Example function**: `Example[FunctionName]`

#### Test Structure (AAA Pattern)

```go
func TestGetUser_ValidID_ReturnsUser(t *testing.T) {
    // Arrange
    userID := "123"
    expected := &User{ID: userID, Name: "John Doe"}
    
    // Act
    result, err := GetUser(userID)
    
    // Assert
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }
    if result.ID != expected.ID {
        t.Errorf("expected ID %s, got %s", expected.ID, result.ID)
    }
    if result.Name != expected.Name {
        t.Errorf("expected name %s, got %s", expected.Name, result.Name)
    }
}
```

#### Test Case Naming

```go
tests := []struct {
    name string
    // ...
}{
    {name: "empty string"},              // ✅ Clear
    {name: "positive number"},           // ✅ Descriptive
    {name: "returns error on timeout"},  // ✅ States expectation
    
    {name: "test1"},                     // ❌ Not descriptive
    {name: "TestEmptyString"},           // ❌ Redundant "Test" prefix
}
```

#### Organize Test Data

```go
// ✅ Good: Vertical alignment, easy to read
tests := []struct {
    name     string
    input    string
    expected int
}{
    {name: "empty",    input: "",      expected: 0},
    {name: "single",   input: "a",     expected: 1},
    {name: "multiple", input: "hello", expected: 5},
}
```

### Unit Testing

#### Test Helpers

```go
// Helper function for creating test users
func newTestUser(t *testing.T, name, email string) *User {
    t.Helper() // Mark as helper to improve error reporting
    
    user := &User{
        ID:    generateID(),
        Name:  name,
        Email: email,
    }
    
    if err := user.Validate(); err != nil {
        t.Fatalf("failed to create test user: %v", err)
    }
    
    return user
}

// Usage
func TestUserService_CreateUser(t *testing.T) {
    user := newTestUser(t, "John", "john@example.com")
    // Test with user
}
```

#### Testing with Helper Functions

```go
func TestValidateEmail(t *testing.T) {
    tests := []struct {
        name    string
        email   string
        wantErr bool
    }{
        {"valid email", "user@example.com", false},
        {"missing @", "userexample.com", true},
        {"missing domain", "user@", true},
        {"empty string", "", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateEmail(tt.email)
            assertError(t, err, tt.wantErr)
        })
    }
}

// Helper function
func assertError(t *testing.T, err error, wantErr bool) {
    t.Helper()  // Marks this as helper for better error reporting
    
    if wantErr && err == nil {
        t.Error("Expected error but got nil")
    }
    if !wantErr && err != nil {
        t.Errorf("Unexpected error: %v", err)
    }
}
```

### Test Fixtures

#### Setup and Teardown

```go
// Setup and teardown
func TestMain(m *testing.M) {
    // Setup
    setupTestDatabase()
    
    // Run tests
    code := m.Run()
    
    // Teardown
    teardownTestDatabase()
    
    os.Exit(code)
}

// Per-test setup
func setup(t *testing.T) (*Database, func()) {
    t.Helper()
    
    db := setupTestDB(t)
    
    cleanup := func() {
        db.Close()
    }
    
    return db, cleanup
}

// Usage
func TestUserRepository_Save(t *testing.T) {
    db, cleanup := setup(t)
    defer cleanup()
    
    repo := NewUserRepository(db)
    // Test repository
}
```

#### Testing with Setup/Teardown in Table-Driven Tests

```go
func TestUserService(t *testing.T) {
    tests := []struct {
        name      string
        setup     func(*testing.T) *User
        userID    string
        wantFound bool
    }{
        {
            name: "existing user",
            setup: func(t *testing.T) *User {
                user := &User{ID: "123", Name: "Alice"}
                // Setup: Create user in test DB
                return user
            },
            userID:    "123",
            wantFound: true,
        },
        {
            name: "non-existent user",
            setup: func(t *testing.T) *User {
                return nil  // No setup needed
            },
            userID:    "999",
            wantFound: false,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup
            expectedUser := tt.setup(t)
            
            // Teardown
            defer func() {
                // Cleanup test data
            }()
            
            // Test
            user, err := service.GetUser(tt.userID)
            
            if tt.wantFound {
                if err != nil {
                    t.Errorf("Expected user, got error: %v", err)
                }
                if !reflect.DeepEqual(user, expectedUser) {
                    t.Errorf("Got %+v; want %+v", user, expectedUser)
                }
            } else {
                if err == nil {
                    t.Error("Expected error for non-existent user")
                }
            }
        })
    }
}
```

### Integration Testing

#### Build Tags

```go
//go:build integration
// +build integration

package user_test

import "testing"

func TestUserRepository_Integration(t *testing.T) {
    // Integration test that requires database
}
```

Run integration tests:
```bash
go test -tags=integration ./...
```

#### TestContainers

```go
import (
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/wait"
)

func setupPostgres(t *testing.T) (string, func()) {
    t.Helper()
    
    ctx := context.Background()
    
    req := testcontainers.ContainerRequest{
        Image:        "postgres:15",
        ExposedPorts: []string{"5432/tcp"},
        Env: map[string]string{
            "POSTGRES_PASSWORD": "password",
            "POSTGRES_DB":       "testdb",
        },
        WaitingFor: wait.ForLog("database system is ready to accept connections"),
    }
    
    postgres, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: req,
        Started:          true,
    })
    if err != nil {
        t.Fatalf("failed to start container: %v", err)
    }
    
    host, _ := postgres.Host(ctx)
    port, _ := postgres.MappedPort(ctx, "5432")
    connString := fmt.Sprintf("postgres://postgres:password@%s:%s/testdb?sslmode=disable",
        host, port.Port())
    
    cleanup := func() {
        postgres.Terminate(ctx)
    }
    
    return connString, cleanup
}

func TestUserRepository_Save_Integration(t *testing.T) {
    connString, cleanup := setupPostgres(t)
    defer cleanup()
    
    db, err := sql.Open("postgres", connString)
    if err != nil {
        t.Fatalf("failed to connect: %v", err)
    }
    defer db.Close()
    
    repo := NewUserRepository(db)
    // Test repository operations
}
```

#### HTTP Integration Tests

```go
func TestUserAPI_Integration(t *testing.T) {
    // Setup test server
    handler := NewUserHandler(repo)
    server := httptest.NewServer(handler)
    defer server.Close()
    
    // Test request
    resp, err := http.Get(server.URL + "/users/123")
    if err != nil {
        t.Fatalf("failed to make request: %v", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        t.Errorf("expected status 200, got %d", resp.StatusCode)
    }
}
```

### Benchmark Testing

#### Basic Benchmarks

```go
func BenchmarkStringConcatenation(b *testing.B) {
    for i := 0; i < b.N; i++ {
        result := ""
        for j := 0; j < 100; j++ {
            result += "a"
        }
    }
}

func BenchmarkStringBuilder(b *testing.B) {
    for i := 0; i < b.N; i++ {
        var builder strings.Builder
        for j := 0; j < 100; j++ {
            builder.WriteString("a")
        }
        _ = builder.String()
    }
}
```

Run benchmarks:
```bash
go test -bench=. -benchmem
```

#### Benchmarks with Setup

```go
func BenchmarkDatabaseQuery(b *testing.B) {
    db := setupTestDB(b)
    defer db.Close()
    
    b.ResetTimer() // Don't include setup in benchmark
    
    for i := 0; i < b.N; i++ {
        _, err := db.Query("SELECT * FROM users WHERE id = ?", 123)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

#### Parallel Benchmarks

```go
func BenchmarkConcurrentWrites(b *testing.B) {
    cache := NewCache()
    
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            cache.Set("key", "value")
        }
    })
}
```

#### Memory Profiling

```bash
# Generate profile
go test -memprofile=mem.prof -bench=.

# Analyze allocations
go tool pprof mem.prof

(pprof) top -alloc_space     # Most allocations
(pprof) list Func            # Where allocations occur
```

#### Benchmark Comparison

```bash
# Save baseline
go test -bench=. -benchmem > old.txt

# Make changes...

# Run new benchmarks
go test -bench=. -benchmem > new.txt

# Compare
benchstat old.txt new.txt
```

### Test Coverage

#### Generate Coverage Report

```bash
# Run tests with coverage
go test -coverprofile=coverage.out ./...

# View coverage in terminal
go tool cover -func=coverage.out

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html
```

#### Coverage Requirements
- **Minimum 80% code coverage** for all packages
- **100% coverage** for critical business logic
- **New code** must include tests
- **Bug fixes** must include regression tests

#### Coverage in CI/CD

```yaml
# GitHub Actions example
- name: Run tests with coverage
  run: go test -v -coverprofile=coverage.out -covermode=atomic ./...

- name: Upload coverage to Codecov
  uses: codecov/codecov-action@v3
  with:
    files: ./coverage.out
```

### Mocking & Test Doubles

#### Interface-Based Mocking

```go
// Define interface for dependency
type UserRepository interface {
    GetByID(ctx context.Context, id string) (*User, error)
    Save(ctx context.Context, user *User) error
}

// Create stub implementation for testing
type StubUserRepo struct {
    users map[string]*User
}

func (r *StubUserRepo) GetByID(ctx context.Context, id string) (*User, error) {
    if user, ok := r.users[id]; ok {
        return user, nil
    }
    return nil, ErrNotFound
}

func (r *StubUserRepo) Save(ctx context.Context, user *User) error {
    r.users[user.ID] = user
    return nil
}

// Use in tests
func TestUserService_GetUser(t *testing.T) {
    repo := &StubUserRepo{
        users: map[string]*User{
            "123": {ID: "123", Name: "Alice"},
        },
    }
    
    service := NewUserService(repo)
    user, err := service.GetUser(context.Background(), "123")
    
    if err != nil || user.Name != "Alice" {
        t.Errorf("Expected Alice, got %v, %v", user, err)
    }
}
```

#### Table-Driven Tests with Stubs

```go
func TestUserService_GetUser(t *testing.T) {
    tests := []struct {
        name      string
        userID    string
        mockUsers map[string]*User
        wantUser  *User
        wantErr   bool
    }{
        {
            name:   "existing user",
            userID: "123",
            mockUsers: map[string]*User{
                "123": {ID: "123", Name: "Alice"},
            },
            wantUser: &User{ID: "123", Name: "Alice"},
            wantErr:  false,
        },
        {
            name:      "non-existent user",
            userID:    "999",
            mockUsers: map[string]*User{},
            wantUser:  nil,
            wantErr:   true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Create stub with test data
            repo := &StubUserRepo{users: tt.mockUsers}
            service := NewUserService(repo)
            
            // Test
            user, err := service.GetUser(tt.userID)
            
            if tt.wantErr {
                if err == nil {
                    t.Error("Expected error but got nil")
                }
                return
            }
            
            if err != nil {
                t.Errorf("Unexpected error: %v", err)
            }
            
            if !reflect.DeepEqual(user, tt.wantUser) {
                t.Errorf("got %+v; want %+v", user, tt.wantUser)
            }
        })
    }
}
```

#### Using testify/mock

```go
import (
    "github.com/stretchr/testify/mock"
    "github.com/stretchr/testify/assert"
)

type MockUserRepository struct {
    mock.Mock
}

func (m *MockUserRepository) GetByID(ctx context.Context, id string) (*User, error) {
    args := m.Called(ctx, id)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*User), args.Error(1)
}

// Usage
func TestUserService_GetUser(t *testing.T) {
    mockRepo := new(MockUserRepository)
    
    // Setup expectations
    expectedUser := &User{ID: "123", Name: "John"}
    mockRepo.On("GetByID", mock.Anything, "123").Return(expectedUser, nil)
    
    service := NewUserService(mockRepo)
    user, err := service.GetUser(context.Background(), "123")
    
    assert.NoError(t, err)
    assert.Equal(t, "John", user.Name)
    mockRepo.AssertExpectations(t)
}
```

### Parallel Test Execution

#### Running Tests in Parallel

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
        tt := tt  // Capture range variable
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()  // Mark for parallel execution
            
            // Test logic here
            result := SlowFunction(tt.input)
            // Assertions...
        })
    }
}
```

**Important:** Must capture range variable (`tt := tt`) when using parallel tests

### Assertion Libraries

#### testify/assert

```go
import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestUser_Validation(t *testing.T) {
    user := User{Name: "John", Email: "john@example.com"}
    
    // Basic assertions
    assert.Equal(t, "John", user.Name)
    assert.NotEmpty(t, user.Email)
    assert.True(t, user.IsValid())
    
    // Require (stops test on failure)
    require.NotNil(t, user)
    require.NoError(t, user.Validate())
    
    // Collections
    users := []User{user}
    assert.Len(t, users, 1)
    assert.Contains(t, users, user)
}
```

### Testing Best Practices

#### Do's ✅
- Write table-driven tests for multiple scenarios
- Use `t.Helper()` for test helper functions
- Test one thing per test function
- Use meaningful test names that describe the scenario
- Test both success and error cases
- Test edge cases and boundary conditions
- Use subtests for grouping related tests
- Clean up resources with defer
- Make tests deterministic (same result every time)
- Use build tags for integration tests
- Run tests in parallel when possible
- Use mocks/stubs for external dependencies
- Keep tests simple and readable
- Test public APIs, not implementation details

#### Don'ts ❌
- Don't test standard library or third-party code
- Don't write tests that depend on external services
- Don't write tests that depend on execution order
- Don't use time.Sleep() - use proper synchronization
- Don't ignore errors in tests
- Don't test multiple scenarios in one test
- Don't write flaky tests
- Don't commit failing tests
- Don't mock everything (test real behavior when possible)
- Don't make tests overly complex
- Don't share state between tests
- Don't test private functions directly

#### Common Patterns Checklist

- ☐ Test cases defined as slice of structs
- ☐ Each case has descriptive `name` field
- ☐ Tests use `t.Run()` for subtests
- ☐ Failure messages follow "got X; want Y" format
- ☐ Helper functions marked with `t.Helper()`
- ☐ Interfaces used for test doubles
- ☐ Test data clearly organized
- ☐ Each test focuses on single concept
- ☐ Range variable captured for parallel tests
- ☐ Setup/teardown properly handled

### Test Organization Structure

```
project/
├── internal/
│   ├── user/
│   │   ├── user.go
│   │   ├── user_test.go
│   │   ├── service.go
│   │   └── service_test.go
│   └── auth/
│       ├── auth.go
│       └── auth.go
├── test/
│   ├── integration/
│   │   └── user_integration_test.go
│   ├── e2e/
│   │   └── api_e2e_test.go
│   └── fixtures/
│       └── testdata.go
```

### Continuous Integration

```yaml
# GitHub Actions example
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Run unit tests
        run: go test -v -race -coverprofile=coverage.out ./...
      
      - name: Run integration tests
        run: go test -v -tags=integration ./...
      
      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./coverage.out
```

## Usage
Consult this agent for testing questions including:
- Implementing table-driven test patterns
- Setting up test fixtures and helpers
- Writing integration tests with TestContainers
- Creating mocks and stubs for dependencies
- Benchmark testing and performance profiling
- Achieving and maintaining test coverage targets
- Organizing test code and test data
- Running tests in CI/CD pipelines
- Using build tags to separate test types
- Parallel test execution strategies

## References
- [Go Testing Package](https://pkg.go.dev/testing)
- [Table Driven Tests](https://github.com/golang/go/wiki/TableDrivenTests)
- [Testify](https://github.com/stretchr/testify)
- [TestContainers for Go](https://golang.testcontainers.org/)
- [Go Fuzzing](https://go.dev/doc/fuzz/)
