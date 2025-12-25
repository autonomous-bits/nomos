---
name: go-expert
description: Provides comprehensive Go language expertise covering general practices, testing, security, performance, and idiomatic patterns for professional Go development.
---

## Standards Source
All content from: https://github.com/autonomous-bits/development-standards/tree/main/standards/go
Last synced: 2025-12-25

## Coverage Areas
- General Go Practices
- Testing Standards
- Security Best Practices
- Performance Optimization
- Naming Conventions
- Error Handling
- Concurrency Patterns
- Documentation Standards
- Project Structure

## Content

### General Go Practices

#### Code Style & Conventions

**Naming Conventions:**
- Use **MixedCaps** or **mixedCaps** (camel case) for all multi-word names
- **Never use** snake_case (except in test file names), kebab-case, or SCREAMING_SNAKE_CASE
- **PascalCase** for exported identifiers (public)
- **camelCase** for unexported identifiers (private)
- Constants use MixedCaps (not SCREAMING_SNAKE_CASE like other languages)

**Visibility:**
- Capitalization determines visibility:
  - `PublicName` - Exported (starts with capital)
  - `privateName` - Unexported (starts with lowercase)
- Applies to: functions, methods, variables, constants, types, struct fields

**Package Names:**
- Short, concise, lowercase (single word)
- Meaningful and indicative of purpose
- Usually singular
- Good: `http`, `oauth2`, `user`, `payment`
- Avoid: `util`, `common`, `helpers`, `api` (too generic)
- Don't stutter: use `user.Create()` not `user.UserCreate()`

**Receiver Names:**
- Use short 1-2 letter abbreviations
- Must be consistent across all methods of a type
- Examples: `func (c *Client) Connect()`, `func (u *User) Validate()`
- Don't use: `this`, `self`, `me`, or overly verbose names

**Interface Names:**
- Single-method interfaces end with `-er`: `Reader`, `Writer`, `Closer`
- Multi-method interfaces use descriptive nouns: `UserRepository`
- **Never** prefix with 'I' (that's C#/Java style, not Go)

**Acronyms:**
- Keep consistent: all caps or all lowercase
- Exported: `APIKey`, `HTTPServer`, `URLPath`
- Unexported: `apiKey`, `httpClient`, `urlPath`
- Wrong: `ApiKey`, `HttpServer`, `UrlPath`

**File Names:**
- Use lowercase with underscores for separation
- Examples: `user.go`, `user_test.go`, `http_client.go`, `user_repository.go`
- Not Go style: `Main.go`, `userRepository.go`, `user-repository.go`

#### File Organization
- One package per directory
- Group related functionality in same package
- Use `internal/` directory for private packages (compiler-enforced)
- `cmd/` directory for executable commands
- `pkg/` directory for public library code (optional)

#### Formatting
- **gofmt is mandatory** - all Go code MUST be formatted with gofmt
- Use `goimports` for automatic import management
- Configure editor to format on save
- No debates about spacing, indentation, or alignment

#### Code Quality
- Always run `go vet` to catch common mistakes
- Use `golangci-lint` for comprehensive linting
- Fix all lint warnings before submitting PR
- Keep functions small and focused

---

### Testing Standards

#### Testing Principles

**Test-Driven Development (TDD):**
- Write tests before implementing functionality when feasible
- Tests document expected behavior
- Red-Green-Refactor cycle
- Keep tests simple and focused

**Test Pyramid:**
- **Unit Tests (70%)**: Fast, isolated tests for individual functions/packages
- **Integration Tests (20%)**: Test interactions between components
- **End-to-End Tests (10%)**: Test complete workflows

#### Unit Testing

**Test File Organization:**
- Test files live alongside code: `user.go` → `user_test.go`
- Test function naming: `Test[FunctionName]_[Scenario]`
- Benchmark function: `Benchmark[FunctionName]`
- Example function: `Example[FunctionName]`

**Basic Test Structure:**
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
}
```

#### Table-Driven Tests

**The canonical Go pattern for unit testing:**
```go
func TestAdd(t *testing.T) {
    tests := []struct {
        name     string
        a, b     int
        expected int
    }{
        {"positive numbers", 2, 3, 5},
        {"negative numbers", -2, -3, -5},
        {"mixed numbers", -2, 3, 1},
        {"zero", 0, 0, 0},
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

**Benefits:**
- Easy to add new test cases
- Clear organization
- Better test output with `t.Run()`
- Can run specific subtests: `go test -run TestAdd/positive`

**Failure Messages:**
- Use standard format: "got X; want Y"
- Provide enough context to debug without running the test
- Example: `t.Errorf("Function(%v) = %v; want %v", input, got, want)`

#### Test Helpers

**Mark helpers with `t.Helper()`:**
```go
func newTestUser(t *testing.T, name, email string) *User {
    t.Helper() // Error reports caller's line, not this line
    
    user := &User{Name: name, Email: email}
    if err := user.Validate(); err != nil {
        t.Fatalf("failed to create test user: %v", err)
    }
    return user
}
```

#### Parallel Tests
```go
func TestParallel(t *testing.T) {
    tests := []struct {
        name  string
        input int
    }{
        {"case 1", 1},
        {"case 2", 2},
    }
    
    for _, tt := range tests {
        tt := tt  // Capture range variable
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()  // Run in parallel
            // Test logic
        })
    }
}
```

#### Integration Tests

**Use build tags:**
```go
//go:build integration
// +build integration

package user_test

func TestUserRepository_Integration(t *testing.T) {
    // Integration test requiring database
}
```

Run: `go test -tags=integration ./...`

#### Benchmark Testing

**Basic benchmarks:**
```go
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

Run: `go test -bench=. -benchmem`

**With setup:**
```go
func BenchmarkDatabaseQuery(b *testing.B) {
    db := setupTestDB(b)
    defer db.Close()
    
    b.ResetTimer() // Don't include setup
    
    for i := 0; i < b.N; i++ {
        _, _ = db.Query("SELECT * FROM users WHERE id = ?", 123)
    }
}
```

**Compare benchmarks:**
```bash
go test -bench=. -benchmem > old.txt
# Make changes
go test -bench=. -benchmem > new.txt
benchstat old.txt new.txt
```

#### Test Coverage

**Requirements:**
- **Minimum 80% code coverage** for all packages
- **100% coverage** for critical business logic
- New code must include tests
- Bug fixes must include regression tests

**Generate coverage:**
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

#### Testing Best Practices

**Do's ✅**
- Write table-driven tests for multiple scenarios
- Use `t.Helper()` for test helper functions
- Test both success and error cases
- Test edge cases and boundary conditions
- Use subtests for organization
- Clean up resources with defer
- Make tests deterministic
- Run tests in parallel when possible
- Keep tests simple and readable

**Don'ts ❌**
- Don't test standard library or third-party code
- Don't write tests that depend on external services
- Don't use `time.Sleep()` - use proper synchronization
- Don't ignore errors in tests
- Don't write flaky tests
- Don't commit failing tests
- Don't mock everything
- Don't share state between tests

---

### Security Best Practices

#### Security Principles

**Defense in Depth:**
- Multiple layers of security controls
- Authentication and authorization at every layer
- Input validation and sanitization
- Secure communication
- Secrets management
- Comprehensive logging and monitoring

**Least Privilege:**
- Grant minimum necessary permissions
- Use fine-grained access controls
- Regular permission audits

**Fail Securely:**
- Default to deny access
- Handle errors without exposing sensitive information
- Log security events
- Provide generic error messages to users

#### Input Validation

**Always validate all inputs:**
```go
func GetUser(id string) (*User, error) {
    if id == "" {
        return nil, errors.New("user ID is required")
    }
    
    if !isValidID(id) {
        return nil, errors.New("invalid user ID format")
    }
    
    return repository.GetByID(id)
}

func isValidID(id string) bool {
    matched, _ := regexp.MatchString(`^[a-zA-Z0-9-]+$`, id)
    return matched && len(id) <= 100
}
```

**SQL Injection Prevention:**
```go
// ✅ GOOD - Use parameterized queries
func GetUserByEmail(email string) (*User, error) {
    var user User
    err := db.QueryRow(
        "SELECT id, name, email FROM users WHERE email = $1",
        email,
    ).Scan(&user.ID, &user.Name, &user.Email)
    return &user, err
}

// ❌ BAD - String concatenation
query := "SELECT * FROM users WHERE email = '" + email + "'"
// Vulnerable to SQL injection!
```

**Command Injection Prevention:**
```go
// ✅ GOOD - Validate and use exec.Command properly
func PingHost(host string) error {
    if !isValidHostname(host) {
        return errors.New("invalid hostname")
    }
    
    cmd := exec.Command("ping", "-c", "4", host)
    return cmd.Run()
}
```

**Path Traversal Prevention:**
```go
// ✅ GOOD - Validate file paths
func ServeFile(filename string) ([]byte, error) {
    cleanPath := filepath.Clean(filename)
    
    baseDir := "/var/www/public"
    fullPath := filepath.Join(baseDir, cleanPath)
    
    if !strings.HasPrefix(fullPath, baseDir) {
        return nil, errors.New("invalid file path")
    }
    
    return os.ReadFile(fullPath)
}
```

#### Authentication & Authorization

**Password Hashing:**
```go
import "golang.org/x/crypto/bcrypt"

func HashPassword(password string) (string, error) {
    hash, err := bcrypt.GenerateFromPassword(
        []byte(password), 
        bcrypt.DefaultCost,
    )
    if err != nil {
        return "", err
    }
    return string(hash), nil
}

func VerifyPassword(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword(
        []byte(hash), 
        []byte(password),
    )
    return err == nil
}
```

**JWT Authentication:**
```go
import "github.com/golang-jwt/jwt/v5"

type Claims struct {
    UserID string `json:"user_id"`
    Email  string `json:"email"`
    jwt.RegisteredClaims
}

func GenerateToken(userID, email string, secret []byte) (string, error) {
    claims := Claims{
        UserID: userID,
        Email:  email,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            Issuer:    "my-app",
        },
    }
    
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(secret)
}
```

#### Cryptography

**Use crypto/rand for secure random:**
```go
import "crypto/rand"

func GenerateSecureToken(length int) (string, error) {
    bytes := make([]byte, length)
    if _, err := rand.Read(bytes); err != nil {
        return "", err
    }
    return base64.URLEncoding.EncodeToString(bytes), nil
}

// ❌ NEVER use math/rand for security
```

**AES Encryption:**
```go
import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
)

func Encrypt(plaintext []byte, key []byte) ([]byte, error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }
    
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }
    
    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, err
    }
    
    return gcm.Seal(nonce, nonce, plaintext, nil), nil
}
```

#### Secrets Management

**Never hardcode secrets:**
```go
// ❌ BAD
const APIKey = "sk_live_1234567890abcdef"

// ✅ GOOD - Use environment variables
func GetConfig() (*Config, error) {
    apiKey := os.Getenv("API_KEY")
    if apiKey == "" {
        return nil, errors.New("API_KEY not set")
    }
    
    return &Config{APIKey: apiKey}, nil
}
```

**Use godotenv for development:**
```go
import "github.com/joho/godotenv"

func init() {
    if os.Getenv("ENV") != "production" {
        godotenv.Load()
    }
}
```

#### Secure HTTP

**TLS Configuration:**
```go
func NewSecureServer(handler http.Handler) *http.Server {
    tlsConfig := &tls.Config{
        MinVersion:               tls.VersionTLS13,
        PreferServerCipherSuites: true,
    }
    
    return &http.Server{
        Addr:         ":443",
        Handler:      handler,
        TLSConfig:    tlsConfig,
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 15 * time.Second,
        IdleTimeout:  60 * time.Second,
    }
}
```

**Security Headers:**
```go
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        w.Header().Set("Strict-Transport-Security", 
            "max-age=31536000; includeSubDomains")
        w.Header().Set("Content-Security-Policy", "default-src 'self'")
        
        next.ServeHTTP(w, r)
    })
}
```

#### Rate Limiting

**Token bucket rate limiter:**
```go
import "golang.org/x/time/rate"

type RateLimiter struct {
    limiters map[string]*rate.Limiter
    mu       sync.RWMutex
    rate     rate.Limit
    burst    int
}

func RateLimitMiddleware(rl *RateLimiter) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            ip := r.RemoteAddr
            limiter := rl.GetLimiter(ip)
            
            if !limiter.Allow() {
                http.Error(w, "rate limit exceeded", 
                    http.StatusTooManyRequests)
                return
            }
            
            next.ServeHTTP(w, r)
        })
    }
}
```

#### Security Best Practices

**Do's ✅**
- Use HTTPS/TLS for all communications
- Validate and sanitize all inputs
- Use parameterized queries
- Hash passwords with bcrypt or Argon2
- Store secrets in environment variables
- Implement rate limiting on APIs
- Use security headers
- Log security events
- Keep dependencies up to date
- Use crypto/rand for cryptographic operations
- Set timeouts on HTTP servers and clients

**Don'ts ❌**
- Never store passwords in plain text
- Never hardcode secrets
- Never trust user input
- Never use weak encryption (MD5, SHA1)
- Never expose stack traces to users
- Never disable TLS verification in production
- Never use math/rand for security
- Never log sensitive information
- Never ignore security warnings from govulncheck

#### Security Checklist
- [ ] All secrets removed from code
- [ ] HTTPS/TLS enforced
- [ ] Authentication and authorization implemented
- [ ] Input validation on all user inputs
- [ ] SQL injection prevention in place
- [ ] Security headers configured
- [ ] Rate limiting implemented
- [ ] Passwords properly hashed
- [ ] Security logging enabled
- [ ] Dependencies scanned for vulnerabilities

---

### Performance Optimization

#### Performance Principles

**Measure First:**
- Always profile before optimizing
- Use benchmarks to validate improvements
- Set performance baselines
- Don't optimize based on assumptions

**The Golden Rule:**
> "Performance optimization without measurement is guesswork."

#### Profiling

**CPU Profiling:**
```go
import "runtime/pprof"

func profileCPU() {
    f, _ := os.Create("cpu.prof")
    defer f.Close()
    
    pprof.StartCPUProfile(f)
    defer pprof.StopCPUProfile()
    
    doWork()
}
```

Analyze: `go tool pprof cpu.prof`

**Memory Profiling:**
```go
func profileMemory() {
    f, _ := os.Create("mem.prof")
    defer f.Close()
    
    runtime.GC()
    pprof.WriteHeapProfile(f)
}
```

**HTTP Profiling (pprof):**
```go
import _ "net/http/pprof"

func main() {
    go http.ListenAndServe("localhost:6060", nil)
    // Your application code
}
```

Access at:
- CPU: `http://localhost:6060/debug/pprof/profile?seconds=30`
- Heap: `http://localhost:6060/debug/pprof/heap`
- Goroutines: `http://localhost:6060/debug/pprof/goroutine`

#### Benchmarking

**Writing benchmarks:**
```go
func BenchmarkFunction(b *testing.B) {
    for i := 0; i < b.N; i++ {
        Function()
    }
}
```

**Run with memory stats:**
```bash
go test -bench=. -benchmem
```

**Metrics:**
- `ns/op`: Nanoseconds per operation
- `B/op`: Bytes allocated per operation
- `allocs/op`: Number of allocations per operation

**Statistical validation with benchstat:**
```bash
go test -bench=. -count=10 > old.txt
# Make changes
go test -bench=. -count=10 > new.txt
benchstat old.txt new.txt
```

#### Memory Optimization

**Pre-allocate slices:**
```go
// ✅ GOOD
func BuildSlice(n int) []int {
    result := make([]int, 0, n)  // Pre-allocate capacity
    for i := 0; i < n; i++ {
        result = append(result, i)
    }
    return result
}

// ❌ BAD - Multiple reallocations
func BuildSlice(n int) []int {
    var result []int
    for i := 0; i < n; i++ {
        result = append(result, i)
    }
    return result
}
```

**Reuse slice memory:**
```go
fields := make([]string, 0, 10)  // Allocate once

for _, line := range lines {
    fields = fields[:0]  // Reset length, keep capacity
    fields = parseFields(line)
    process(fields)
}
```

**Use strings.Builder:**
```go
// ✅ GOOD
func BuildString(items []string) string {
    var builder strings.Builder
    builder.Grow(len(items) * 10)  // Pre-allocate if size known
    
    for _, item := range items {
        builder.WriteString(item)
    }
    return builder.String()
}

// ❌ BAD - Creates new string each iteration
func BuildString(items []string) string {
    result := ""
    for _, item := range items {
        result += item
    }
    return result
}
```

**Use sync.Pool for frequently allocated objects:**
```go
var bufferPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}

func ProcessData(data []byte) {
    buf := bufferPool.Get().(*bytes.Buffer)
    buf.Reset()
    defer bufferPool.Put(buf)
    
    buf.Write(data)
    // Process...
}
```

**Pre-size maps:**
```go
// ✅ GOOD
m := make(map[string]Item, len(items))

// ❌ BAD
m := make(map[string]Item)  // Will grow dynamically
```

#### Concurrency Optimization

**Worker pools:**
```go
func WorkerPool(jobs <-chan Job, results chan<- Result, workers int) {
    var wg sync.WaitGroup
    
    for i := 0; i < workers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for job := range jobs {
                results <- processJob(job)
            }
        }()
    }
    
    wg.Wait()
    close(results)
}
```

**Bounded concurrency:**
```go
func ProcessItemsConcurrently(items []Item, maxConcurrency int) {
    sem := make(chan struct{}, maxConcurrency)
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
}
```

#### I/O Optimization

**Buffered I/O:**
```go
func ProcessFile(filename string) error {
    file, _ := os.Open(filename)
    defer file.Close()
    
    reader := bufio.NewReader(file)  // Use buffered reader
    for {
        line, err := reader.ReadString('\n')
        if err == io.EOF {
            break
        }
        processLine(line)
    }
    return nil
}
```

**Batch database operations:**
```go
func BatchInsertUsers(users []User) error {
    tx, _ := db.Begin()
    defer tx.Rollback()
    
    stmt, _ := tx.Prepare("INSERT INTO users (name, email) VALUES (?, ?)")
    defer stmt.Close()
    
    for _, user := range users {
        stmt.Exec(user.Name, user.Email)
    }
    
    return tx.Commit()
}
```

#### HTTP Performance

**Reuse HTTP clients:**
```go
// ✅ GOOD
var httpClient = &http.Client{
    Timeout: 10 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
    },
}

// ❌ BAD - Create new client each time
func MakeRequest(url string) (*http.Response, error) {
    client := &http.Client{Timeout: 10 * time.Second}
    return client.Get(url)
}
```

**Always close response bodies:**
```go
// ✅ GOOD
resp, err := http.Get(url)
if err != nil {
    return err
}
defer resp.Body.Close()

body, _ := io.ReadAll(resp.Body)
```

#### Performance Best Practices

**Do's ✅**
- Profile before optimizing
- Use benchmarks to validate improvements
- Pre-allocate slices and maps when size is known
- Use strings.Builder for string concatenation
- Reuse HTTP clients and database connections
- Use buffered I/O for file operations
- Use worker pools for concurrent processing
- Close resources promptly with defer
- Batch database operations
- Cache frequently accessed data
- Monitor goroutine count

**Don'ts ❌**
- Don't optimize without profiling
- Don't create new HTTP clients for each request
- Don't allocate in hot paths unnecessarily
- Don't use string concatenation in loops
- Don't leak goroutines
- Don't block indefinitely
- Don't use unbounded goroutines
- Don't forget to close response bodies
- Don't perform I/O in tight loops
- Don't copy large structs by value

---

### Naming Conventions

#### General Principles

**MixedCaps Rule:**
- All multi-word names use MixedCaps or mixedCaps
- **Never** use snake_case, kebab-case, or SCREAMING_SNAKE_CASE
- Applies to: functions, methods, variables, constants, types, struct fields

**Visibility:**
- Capitalization determines visibility:
  - `ExportedName` - Public (starts with capital)
  - `unexportedName` - Private (starts with lowercase)

#### Constants

**Go's unique approach - use MixedCaps:**
```go
// ✅ Idiomatic
const MaxLength = 100
const defaultTimeout = 30 * time.Second
const APIVersion = "v1"

// ❌ Unidiomatic
const MAX_LENGTH = 100        // Wrong convention
const DEFAULT_TIMEOUT = 30    // Not Go style
```

#### Packages

**Guidelines:**
- Short, concise, lowercase (single word)
- Meaningful and indicative of purpose
- Usually singular

**Good:** `http`, `oauth2`, `user`, `payment`, `database`
**Avoid:** `util`, `common`, `helpers`, `api` (too generic)

**Don't stutter:**
```go
// ❌ Stutters
user.UserCreate()
user.UserDelete()

// ✅ Clear from context
user.Create()
user.Delete()
```

#### Receiver Names

**Use short 1-2 letter abbreviations:**
```go
// ✅ Idiomatic
func (c *Client) Connect() error { }
func (s *Scanner) Scan() error { }
func (u *User) Validate() error { }

// ❌ Unidiomatic
func (this *Client) Connect() error { }    // Don't use 'this'
func (self *Client) Connect() error { }    // Don't use 'self'
func (client *Client) Connect() error { }  // Too verbose
```

**Consistency is crucial** - use the same receiver name across all methods of a type.

#### Variables

**Short names in short scopes:**
```go
for i := 0; i < n; i++ {  // 'i' is fine
    // ...
}

if err := f(); err != nil {  // 'err' is standard
    return err
}
```

**Descriptive names in larger scopes:**
```go
var connectionPool *Pool
var httpClient *http.Client
```

**Common abbreviations:**
- `i`, `j`, `k` - Loop indices
- `err` - Errors (always)
- `ctx` - Context (always)
- `db` - Database
- `tx` - Transaction
- `b` - Bytes or buffer
- `r` - Reader
- `w` - Writer

#### Interfaces

**Single-method interfaces add "-er":**
```go
type Reader interface {
    Read(p []byte) (n int, err error)
}

type Writer interface {
    Write(p []byte) (n int, err error)
}
```

**Multi-method interfaces use descriptive nouns:**
```go
type UserRepository interface {
    FindByID(id string) (*User, error)
    Save(user *User) error
    Delete(id string) error
}
```

**Don't prefix with 'I':**
```go
// ❌ Unidiomatic
type IUserRepository interface { }

// ✅ Idiomatic
type UserRepository interface { }
```

#### Acronyms

**Keep consistent - all caps or all lowercase:**
```go
// ✅ Correct
APIKey, apiKey
HTTPServer, httpClient
URLPath, urlPath

// ❌ Incorrect
ApiKey          // Mixed capitalization
HttpServer      // Should be HTTP
UrlPath         // Should be URL
```

#### File Names

**Use lowercase with underscores:**
```
user.go
user_test.go
http_client.go
user_repository.go
```

**Not Go style:**
```
Main.go            // Capitalized
userRepository.go  // CamelCase
user-repository.go // Kebab-case
```

---

### Error Handling

#### Core Principles

**Errors are values:**
- Go uses explicit error return values, not exceptions
- No try/catch, no stack unwinding
- Every error must be explicitly handled

**Library code must never panic:**
- Panic brings down entire application
- Libraries must return errors for callers to handle
- Reserve panic only for unrecoverable programming errors

#### Error Creation

**Simple errors:**
```go
err := errors.New("connection failed")
```

**Errors with context:**
```go
err := fmt.Errorf("failed to process user %s: %w", userID, originalErr)
```

**Sentinel errors:**
```go
var (
    ErrNotFound     = errors.New("not found")
    ErrUnauthorized = errors.New("unauthorized")
    ErrInvalidInput = errors.New("invalid input")
)
```

**Custom error types:**
```go
type ValidationError struct {
    Field string
    Issue string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation error: %s - %s", e.Field, e.Issue)
}
```

#### Error Checking

**Always check errors:**
```go
// ✅ Good
result, err := DoSomething()
if err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}

// ❌ Bad - Ignore error
result, _ := DoSomething()

// ❌ Bad - Implicit ignore
DoSomething()
```

**Early return pattern:**
```go
func Process() error {
    if err := step1(); err != nil {
        return fmt.Errorf("step1: %w", err)
    }
    
    if err := step2(); err != nil {
        return fmt.Errorf("step2: %w", err)
    }
    
    return nil
}
```

**Check error types:**
```go
// For sentinel errors
if errors.Is(err, ErrNotFound) {
    // Handle not found
}

// For custom error types
var validationErr *ValidationError
if errors.As(err, &validationErr) {
    // Handle validation error
}
```

#### Error Messages

**Guidelines:**

1. **Lowercase, no punctuation:**
```go
// ✅ Good
errors.New("connection failed")

// ❌ Bad
errors.New("Connection failed.")
```

2. **Add context when wrapping:**
```go
// ✅ Good
return fmt.Errorf("save user %s: %w", user.ID, err)

// ❌ Bad
return fmt.Errorf("error: %w", err)
```

3. **Be specific and actionable:**
```go
// ✅ Good
return fmt.Errorf("invalid email format: %s", email)

// ❌ Bad
return errors.New("bad input")
```

4. **Don't expose internal details:**
```go
// ✅ Good - Generic for security
return errors.New("authentication failed")

// ❌ Bad - Exposes internal info
return fmt.Errorf("user %s not found in database", email)
```

#### When to Use Panic

**Panic is only for:**
- Unrecoverable programming errors
- Initialization failures (in main or init)
- Truly impossible situations

**Examples:**
```go
// ✅ Acceptable in main
func main() {
    config, err := loadConfig()
    if err != nil {
        panic(err)  // Can't continue without config
    }
}

// ❌ Never in library code
func GetUser(id string) *User {
    user, err := db.Query(id)
    if err != nil {
        panic(err)  // WRONG - caller can't handle this
    }
    return user
}
```

#### Error Handling Patterns

**Pattern 1: Early return**
```go
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}
```

**Pattern 2: Error wrapping with context**
```go
if err := saveUser(user); err != nil {
    return fmt.Errorf("save user %s: %w", user.ID, err)
}
```

**Pattern 3: Defer for cleanup**
```go
f, err := os.Open(filename)
if err != nil {
    return err
}
defer f.Close()

// Work with file
```

**Pattern 4: Sentinel errors**
```go
var ErrNotFound = errors.New("not found")

func GetUser(id string) (*User, error) {
    user := findUser(id)
    if user == nil {
        return nil, ErrNotFound
    }
    return user, nil
}

// Caller
user, err := GetUser(id)
if errors.Is(err, ErrNotFound) {
    // Handle not found
}
```

#### Best Practices Checklist

**Error Creation:**
- ☐ Library code never panics
- ☐ Errors returned instead of panics
- ☐ Error messages are lowercase, no punctuation
- ☐ Context added when wrapping errors

**Error Handling:**
- ☐ Every error is explicitly checked
- ☐ No blank identifier for errors (`_`)
- ☐ Early return pattern used
- ☐ Error wrapping with %w for context

**Error Checking:**
- ☐ errors.Is() used for sentinel errors
- ☐ errors.As() used for custom error types

**Panic Usage:**
- ☐ Panic only for programming errors
- ☐ No panic in library code
- ☐ Recover only in top-level handlers

---

### Concurrency Patterns

#### Core Philosophy

**"Do not communicate by sharing memory; share memory by communicating."**

- Use channels to pass data between goroutines
- Avoid shared state when possible
- Use mutexes only when channels don't fit

#### context.Context

**Mandatory for cancellation:**
```go
func DoSomething(ctx context.Context, param string) error {
    // Check for cancellation
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }
    
    // Pass context to downstream calls
    return downstream.Call(ctx, param)
}
```

**Always accept context as first parameter:**
```go
func ProcessRequest(ctx context.Context, req *Request) (*Response, error) {
    // Implementation
}
```

#### Goroutines

**Basic usage:**
```go
go processItem(item)
```

**Always ensure goroutines can exit:**
```go
func Start(ctx context.Context) {
    go func() {
        ticker := time.NewTicker(time.Second)
        defer ticker.Stop()
        
        for {
            select {
            case <-ctx.Done():
                return  // Exit when context cancelled
            case <-ticker.C:
                doWork()
            }
        }
    }()
}
```

#### Channels

**Basic patterns:**
```go
// Unbuffered channel
ch := make(chan int)

// Buffered channel
ch := make(chan int, 10)

// Send
ch <- value

// Receive
value := <-ch

// Close (sender only)
close(ch)
```

**Range over channel:**
```go
for item := range ch {
    process(item)
}
// Loop exits when channel is closed
```

**Select for multiple channels:**
```go
select {
case msg := <-ch1:
    handleMessage(msg)
case <-ch2:
    handleSignal()
case <-ctx.Done():
    return ctx.Err()
default:
    // Non-blocking option
}
```

#### Common Patterns

**Worker pool:**
```go
func WorkerPool(jobs <-chan Job, results chan<- Result, numWorkers int) {
    var wg sync.WaitGroup
    
    for i := 0; i < numWorkers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for job := range jobs {
                results <- processJob(job)
            }
        }()
    }
    
    wg.Wait()
    close(results)
}
```

**Pipeline:**
```go
func Generator(ctx context.Context, nums ...int) <-chan int {
    out := make(chan int)
    go func() {
        defer close(out)
        for _, n := range nums {
            select {
            case out <- n:
            case <-ctx.Done():
                return
            }
        }
    }()
    return out
}

func Square(ctx context.Context, in <-chan int) <-chan int {
    out := make(chan int)
    go func() {
        defer close(out)
        for n := range in {
            select {
            case out <- n * n:
            case <-ctx.Done():
                return
            }
        }
    }()
    return out
}
```

**Fan-out, fan-in:**
```go
func FanOut(ctx context.Context, in <-chan int, numWorkers int) []<-chan int {
    channels := make([]<-chan int, numWorkers)
    for i := 0; i < numWorkers; i++ {
        channels[i] = process(ctx, in)
    }
    return channels
}

func FanIn(ctx context.Context, channels ...<-chan int) <-chan int {
    out := make(chan int)
    var wg sync.WaitGroup
    
    for _, ch := range channels {
        wg.Add(1)
        go func(c <-chan int) {
            defer wg.Done()
            for n := range c {
                select {
                case out <- n:
                case <-ctx.Done():
                    return
                }
            }
        }(ch)
    }
    
    go func() {
        wg.Wait()
        close(out)
    }()
    
    return out
}
```

#### Synchronization Primitives

**sync.Mutex:**
```go
type Counter struct {
    mu    sync.Mutex
    value int
}

func (c *Counter) Increment() {
    c.mu.Lock()
    c.value++
    c.mu.Unlock()
}
```

**sync.RWMutex:**
```go
type Cache struct {
    mu    sync.RWMutex
    items map[string]interface{}
}

func (c *Cache) Get(key string) interface{} {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.items[key]
}

func (c *Cache) Set(key string, value interface{}) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.items[key] = value
}
```

**sync.WaitGroup:**
```go
var wg sync.WaitGroup

for _, item := range items {
    wg.Add(1)
    go func(item Item) {
        defer wg.Done()
        process(item)
    }(item)
}

wg.Wait()
```

**sync.Once:**
```go
var (
    instance *Singleton
    once     sync.Once
)

func GetInstance() *Singleton {
    once.Do(func() {
        instance = &Singleton{}
    })
    return instance
}
```

#### Preventing Goroutine Leaks

**Always provide exit mechanism:**
```go
// ✅ Good - Can be cancelled
func Start(ctx context.Context) {
    go func() {
        for {
            select {
            case <-ctx.Done():
                return  // Exit path
            default:
                doWork()
            }
        }
    }()
}

// ❌ Bad - No way to stop
func Start() {
    go func() {
        for {
            doWork()  // Runs forever
        }
    }()
}
```

#### Race Detection

**Always run tests with race detector:**
```bash
go test -race ./...
go build -race
go run -race main.go
```

#### Best Practices Checklist

**Context Usage:**
- ☐ Context passed as first parameter
- ☐ Context checked for cancellation
- ☐ Context propagated to downstream calls

**Channel Safety:**
- ☐ Only sender closes channels
- ☐ Check for closed channels when needed
- ☐ Buffered channels sized appropriately

**Goroutine Management:**
- ☐ All goroutines have exit mechanism
- ☐ WaitGroups used for coordination
- ☐ Context used for cancellation

**Synchronization:**
- ☐ Prefer channels over mutexes
- ☐ Use RWMutex when appropriate
- ☐ Defer unlock calls

**Testing:**
- ☐ All tests run with -race
- ☐ Race detector passes
- ☐ Timeouts used in tests

---

### Documentation Standards

#### Core Philosophy

**Documentation is not optional - it's part of the API:**
- All exported identifiers MUST have documentation comments
- Comments are for godoc (Go's documentation tool)
- Focus on "why" not "what" (code shows what)

#### Documentation Comments

**Format requirements:**

1. **Complete sentences:**
```go
// ✅ Good
// Process validates and saves the user data.

// ❌ Bad
// process user data
```

2. **Begin with the name being described:**
```go
// ✅ Good
// CalculateTotal computes the sum of all items in the cart.

// ❌ Bad
// This function computes the sum of all items.
```

3. **End with period:**
```go
// ✅ Good
// Validate checks if the email format is correct.

// ❌ Bad
// Validate checks if the email format is correct
```

#### Package Comments

**Package comment directly before package clause:**
```go
// Package user provides types and functions for managing user accounts.
// It supports creating, updating, and deleting users, as well as
// authentication and authorization workflows.
package user
```

**For complex packages, include usage example:**
```go
// Package config provides configuration management for the application.
//
// Example usage:
//
//     cfg, err := config.Load("config.yaml")
//     if err != nil {
//         log.Fatal(err)
//     }
//     server := NewServer(cfg.Port)
//
package config
```

#### Function Documentation

```go
// Add returns the sum of two integers.
// It handles overflow by returning the maximum integer value.
func Add(a, b int) int {
    // Implementation
}

// NewClient creates a configured HTTP client with default timeout.
// The client uses a connection pool of 100 connections and includes
// retry logic for transient failures.
func NewClient() *Client {
    // Implementation
}
```

#### Type Documentation

```go
// User represents an authenticated user in the system.
// Each user has a unique ID and may belong to multiple organizations.
type User struct {
    ID           string
    Name         string
    Organizations []string
}

// Repository defines the interface for user data access.
// Implementations must be safe for concurrent use.
type Repository interface {
    FindByID(id string) (*User, error)
    Save(user *User) error
}
```

#### When to Write Comments

**Do comment:**
- All exported identifiers (package, type, function, constant, variable)
- Complex algorithms or non-obvious logic
- Why something is done a certain way
- Important constraints or assumptions
- Performance considerations

**Don't comment:**
- Obvious code (the code itself is the comment)
- What the code does (focus on why)
- Redundant information

#### Implementation Comments

**For complex logic:**
```go
func ProcessData(data []byte) error {
    // Parse the header to determine format version.
    // Version 1: legacy format with fixed-width fields
    // Version 2: JSON format with schema validation
    version := data[0]
    
    switch version {
    case 1:
        return processLegacy(data[1:])
    case 2:
        return processJSON(data[1:])
    default:
        return fmt.Errorf("unsupported version: %d", version)
    }
}
```

#### Godoc Integration

**View documentation:**
```bash
go doc package
go doc package.Type
go doc package.Function
```

**Generate HTML documentation:**
```bash
godoc -http=:6060
```

#### Documentation Checklist

**Package Level:**
- ☐ Package comment exists
- ☐ Comment directly before package clause
- ☐ Starts with "Package <name>"
- ☐ Describes package purpose clearly

**Exported Symbols:**
- ☐ All exported types documented
- ☐ All exported functions documented
- ☐ All exported methods documented
- ☐ All exported constants documented

**Comment Quality:**
- ☐ Comments are complete sentences
- ☐ Comments begin with name being described
- ☐ Comments end with period
- ☐ Comments explain "why" not "what"
- ☐ No outdated or incorrect comments

---

### Project Structure

#### Core Principle

**Start simple, grow as needed:**
- Small projects: single `main.go` at root
- Add structure only when complexity demands it
- Don't over-engineer for imagined future needs

#### Primary Directories

**`/cmd`**
- **Purpose:** Main applications for the project
- **When to use:** Multiple binaries
- **Structure:**
```
/cmd
  /server
    main.go
  /cli
    main.go
```

**`/internal`**
- **Purpose:** Private application and library code
- **Key feature:** Compiler-enforced privacy
  - Code in `/internal` cannot be imported by external projects
  - Go compiler prevents `import "github.com/user/project/internal/..."`
- **When to use:** Most of your application code
- **Structure:**
```
/internal
  /server
    server.go
  /user
    service.go
    repository.go
  /auth
    middleware.go
```

**`/pkg`**
- **Purpose:** Public library code intended for external import
- **When to use:** Building reusable libraries
- **When NOT to use:** Private application code
- **Structure:**
```
/pkg
  /client
    client.go
  /models
    user.go
```

#### Additional Common Directories

**`/api`**
- API specifications (OpenAPI/Swagger, Protocol Buffers, GraphQL schemas)

**`/configs`**
- Configuration file templates or defaults

**`/scripts`**
- Build, install, analysis scripts

**`/test`**
- Additional test data, test utilities
- Note: Test files should live alongside code (`*_test.go`)

#### What NOT to Include

**Avoid:**
- `/src` - Unnecessary in Go
- `/models` at root - Data structures should live with domain logic
- `/controllers` - Organize by domain, not technical layer
- `/utils` or `/helpers` - Too generic, be specific

#### Package Organization Philosophy

**Package by domain, not by type:**

✅ **Good (by domain):**
```
/internal
  /user
    service.go
    repository.go
    types.go
  /payment
    service.go
    repository.go
```

❌ **Bad (by type):**
```
/internal
  /services
    user_service.go
    payment_service.go
  /repositories
    user_repository.go
    payment_repository.go
```

#### Evolution Path

**Start simple:**
```
main.go
user.go
user_test.go
```

**Add structure as needed:**
```
cmd/
  server/
    main.go
internal/
  user/
    service.go
    service_test.go
```

**Full structure for complex projects:**
```
cmd/
  server/
    main.go
  cli/
    main.go
internal/
  user/
  payment/
  auth/
pkg/
  client/
api/
configs/
test/
```

#### Decision Guide

**Question 1: How many binaries?**
- One → Maybe don't need `/cmd` yet
- Multiple → Use `/cmd` with subdirectories

**Question 2: Will others import this?**
- Yes → Expose stable API in `/pkg`
- No → Keep everything in `/internal`

**Question 3: Need architectural boundaries?**
- Yes → Use `/internal` (compiler-enforced)
- No → Simple package structure is fine

**Question 4: Team size and complexity?**
- Small/Simple → Flat structure
- Large/Complex → Full standard layout

---

## Usage

Consult this agent for all Go language-specific questions including:
- Code structure and organization
- Idiomatic Go patterns and conventions
- Testing approaches and strategies
- Security considerations and best practices
- Performance optimization techniques
- Error handling patterns
- Concurrency patterns and goroutine management
- Documentation standards and godoc integration
- Project layout and package design
- Naming conventions across the Go ecosystem

This agent provides comprehensive, professional-level guidance based on official Go standards and industry best practices from the Go community.
