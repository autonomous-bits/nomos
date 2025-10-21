---
description: 'Instructions for writing Go code following idiomatic Go practices and community standards'
applyTo: '**/*.go,**/go.mod,**/go.sum'
---

# Go Development Instructions

Follow idiomatic Go practices and community standards when writing Go code. These instructions are based on [Effective Go](https://go.dev/doc/effective_go), [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments), and [Google's Go Style Guide](https://google.github.io/styleguide/go/).

## General Instructions

- Write simple, clear, and idiomatic Go code
- Favor clarity and simplicity over cleverness
- Follow the principle of least surprise
- Keep the happy path left-aligned (minimize indentation)
- Return early to reduce nesting
- Make the zero value useful
- Document exported types, functions, methods, and packages
- Use Go modules for dependency management

## Naming Conventions

### Packages

- Use lowercase, single-word package names
- Avoid underscores, hyphens, or mixedCaps
- Choose names that describe what the package provides, not what it contains
- Avoid generic names like `util`, `common`, or `base`
- Package names should be singular, not plural

### Variables and Functions

- Use mixedCaps or MixedCaps (camelCase) rather than underscores
- Keep names short but descriptive
- Use single-letter variables only for very short scopes (like loop indices)
- Exported names start with a capital letter
- Unexported names start with a lowercase letter
- Avoid stuttering (e.g., avoid `http.HTTPServer`, prefer `http.Server`)

### Interfaces

- Name interfaces with -er suffix when possible (e.g., `Reader`, `Writer`, `Formatter`)
- Single-method interfaces should be named after the method (e.g., `Read` → `Reader`)
- Keep interfaces small and focused

### Constants

- Use MixedCaps for exported constants
- Use mixedCaps for unexported constants
- Group related constants using `const` blocks
- Consider using typed constants for better type safety

## Code Style and Formatting

### Formatting

- Always use `gofmt` to format code
- Use `goimports` to manage imports automatically
- Keep line length reasonable (no hard limit, but consider readability)
- Add blank lines to separate logical groups of code

### Comments

- Write comments in complete sentences
- Start sentences with the name of the thing being described
- Package comments should start with "Package [name]"
- Use line comments (`//`) for most comments
- Use block comments (`/* */`) sparingly, mainly for package documentation
- Document why, not what, unless the what is complex

### Error Handling

- Check errors immediately after the function call
- Don't ignore errors using `_` unless you have a good reason (document why)
- Wrap errors with context using `fmt.Errorf` with `%w` verb
- Create custom error types when you need to check for specific errors
- Place error returns as the last return value
- Name error variables `err`
- Keep error messages lowercase and don't end with punctuation

## Architecture and Project Structure

### Package Organization

- Follow standard Go project layout conventions
- Keep `main` packages in `cmd/` directory
- Put reusable packages in `pkg/` or `internal/`
- Use `internal/` for packages that shouldn't be imported by external projects
- Group related functionality into packages
- Avoid circular dependencies

### Dependency Management

- Use Go modules (`go.mod` and `go.sum`)
- Keep dependencies minimal
- Regularly update dependencies for security patches
- Use `go mod tidy` to clean up unused dependencies
- Vendor dependencies only when necessary

## Type Safety and Language Features

### Type Definitions

- Define types to add meaning and type safety
- Use struct tags for JSON, XML, database mappings
- Prefer explicit type conversions
- Use type assertions carefully and check the second return value

### Pointers vs Values

- Use pointers for large structs or when you need to modify the receiver
- Use values for small structs and when immutability is desired
- Be consistent within a type's method set
- Consider the zero value when choosing pointer vs value receivers

### Interfaces and Composition

- Accept interfaces, return concrete types
- Keep interfaces small (1-3 methods is ideal)
- Use embedding for composition
- Define interfaces close to where they're used, not where they're implemented
- Don't export interfaces unless necessary

## Concurrency

### Goroutines

- Don't create goroutines in libraries; let the caller control concurrency
- Always know how a goroutine will exit
- Use `sync.WaitGroup` or channels to wait for goroutines
- Avoid goroutine leaks by ensuring cleanup

### Channels

- Use channels to communicate between goroutines
- Don't communicate by sharing memory; share memory by communicating
- Close channels from the sender side, not the receiver
- Use buffered channels when you know the capacity
- Use `select` for non-blocking operations

### Synchronization

- Use `sync.Mutex` for protecting shared state
- Keep critical sections small
- Use `sync.RWMutex` when you have many readers
- Prefer channels over mutexes when possible
- Use `sync.Once` for one-time initialization

## Error Handling Patterns

### Creating Errors

- Use `errors.New` for simple static errors
- Use `fmt.Errorf` for dynamic errors
- Create custom error types for domain-specific errors
- Export error variables for sentinel errors
- Use `errors.Is` and `errors.As` for error checking

### Error Propagation

- Add context when propagating errors up the stack
- Don't log and return errors (choose one)
- Handle errors at the appropriate level
- Consider using structured errors for better debugging

## API Design

### HTTP Handlers

- Use `http.HandlerFunc` for simple handlers
- Implement `http.Handler` for handlers that need state
- Use middleware for cross-cutting concerns
- Set appropriate status codes and headers
- Handle errors gracefully and return appropriate error responses

### JSON APIs

- Use struct tags to control JSON marshaling
- Validate input data
- Use pointers for optional fields
- Consider using `json.RawMessage` for delayed parsing
- Handle JSON errors appropriately

## Performance Optimization

### Memory Management

- Minimize allocations in hot paths
- Reuse objects when possible (consider `sync.Pool`)
- Use value receivers for small structs
- Preallocate slices when size is known
- Avoid unnecessary string conversions

### Profiling

- Use built-in profiling tools (`pprof`)
- Benchmark critical code paths
- Profile before optimizing
- Focus on algorithmic improvements first
- Consider using `testing.B` for benchmarks

## Testing

### Test Organization

- Keep tests in the same package (white-box testing)
- Use `_test` package suffix for black-box testing
- Name test files with `_test.go` suffix
- Place test files next to the code they test

### Writing Tests

- Use table-driven tests for multiple test cases
- Name tests descriptively using `Test_functionName_scenario`
- Use subtests with `t.Run` for better organization
- Test both success and error cases
- Use `testify` or similar libraries sparingly

### Test Helpers

- Mark helper functions with `t.Helper()`
- Create test fixtures for complex setup
- Use `testing.TB` interface for functions used in tests and benchmarks
- Clean up resources using `t.Cleanup()`

## Security Best Practices

### Input Validation

- Validate all external input
- Use strong typing to prevent invalid states
- Sanitize data before using in SQL queries
- Be careful with file paths from user input
- Validate and escape data for different contexts (HTML, SQL, shell)

### Cryptography

- Use standard library crypto packages
- Don't implement your own cryptography
- Use crypto/rand for random number generation
- Store passwords using bcrypt or similar
- Use TLS for network communication

## Documentation

### Code Documentation

- Document all exported symbols
- Start documentation with the symbol name
- Use examples in documentation when helpful
- Keep documentation close to code
- Update documentation when code changes

### README and Documentation Files

- Include clear setup instructions
- Document dependencies and requirements
- Provide usage examples
- Document configuration options
- Include troubleshooting section

## Tools and Development Workflow

### Essential Tools

- `go fmt`: Format code
- `go vet`: Find suspicious constructs
- `golint` or `golangci-lint`: Additional linting
- `go test`: Run tests
- `go mod`: Manage dependencies
- `go generate`: Code generation

### Development Practices

- Run tests before committing
- Use pre-commit hooks for formatting and linting
- Keep commits focused and atomic
- Write meaningful commit messages
- Review diffs before committing

## Common Pitfalls to Avoid

- Not checking errors
- Ignoring race conditions
- Creating goroutine leaks
- Not using defer for cleanup
- Modifying maps concurrently
- Not understanding nil interfaces vs nil pointers
- Forgetting to close resources (files, connections)
- Using global variables unnecessarily
- Over-using empty interfaces (`interface{}`)
- Not considering the zero value of types

---

## Architectural Standards for Go Applications

This section defines architectural standards and best practices that must be followed when building Go applications in this repository. These standards apply to architectural reviews and implementation validation.

### Application Security Standard

When working with application security in Go applications, adhere to the following standards:

**Official Go Security Guidelines:**
- [Go Security Best Practices](https://go.dev/doc/security/best-practices)
- [OWASP Go Secure Coding Practices](https://owasp.org/www-project-go-secure-coding-practices-guide/)

**Key Security Requirements:**
- Always validate and sanitize user inputs
- Use parameterized queries with database/sql to prevent SQL injection
- Implement proper authentication and authorization
- Use crypto/rand for cryptographic operations (never math/rand)
- Use bcrypt, scrypt, or argon2 for password hashing
- Protect against timing attacks using subtle.ConstantTimeCompare
- Never hardcode secrets, credentials, or API keys
- Use environment variables or secret management services
- Implement proper error handling without leaking sensitive information
- Keep dependencies up-to-date and scan for vulnerabilities (govulncheck)
- Use HTTPS/TLS for all network communication
- Implement rate limiting and request throttling
- Validate all file uploads and restrict file types

**PRD Requirements:**
When reviewing a PRD, ensure that:
- Security requirements are explicitly defined
- Authentication and authorization mechanisms are specified
- Data protection requirements are clear
- Compliance requirements are identified
- Security acceptance criteria are included

### Web API Standards

When building or reviewing Web APIs in Go, follow these standards:

**Key API Design Requirements:**
- Use standard HTTP methods appropriately (GET, POST, PUT, DELETE, PATCH)
- Implement RESTful resource naming conventions
- Use proper HTTP status codes
- Version APIs using URL paths or headers
- Implement consistent error response formats
- Use middleware for cross-cutting concerns (logging, auth, CORS)
- Implement pagination for collections
- Use context.Context for request-scoped values and cancellation
- Implement graceful shutdown with signal handling
- Use structured logging (slog in Go 1.21+)
- Document APIs using OpenAPI/Swagger specifications

**Recommended Libraries:**
- `net/http` for HTTP servers (standard library)
- `gorilla/mux` or `chi` for advanced routing
- `go-swagger` or `swaggo/swag` for API documentation

### Logging Standards

When implementing logging in Go applications, follow these standards:

**Key Logging Requirements:**
- Use structured logging with the `log/slog` package (Go 1.21+)
- For older versions, use `go.uber.org/zap` or `github.com/sirupsen/logrus`
- Implement appropriate log levels (Debug, Info, Warn, Error)
- Never log sensitive information (passwords, tokens, PII)
- Use contextual logging with request IDs and correlation IDs
- Log to stdout/stderr for container-friendly logging
- Use JSON format for production environments
- Implement log sampling for high-volume services
- Add structured fields for better observability

**Example using slog:**
```go
import "log/slog"

func ProcessOrder(orderID string) error {
    logger := slog.Default()
    logger.Info("Processing order",
        slog.String("order_id", orderID),
        slog.String("status", "started"))
    
    // ... processing logic
    
    return nil
}
```

### Container Delivery Standards (Docker)

When containerizing Go applications, follow these standards:

**Key Container Requirements:**
- Use multi-stage builds to minimize final image size
- Use official Go base images from Docker Hub
- Build stage: `golang:1.23` or later
- Runtime stage: `alpine`, `distroless`, or `scratch` for minimal images
- Build static binaries with `CGO_ENABLED=0` for maximum portability
- Run containers as non-root users
- Use .dockerignore to exclude unnecessary files
- Implement health check endpoints
- Use semantic versioning for image tags
- Include git commit hashes in image labels

**Example Multi-Stage Dockerfile:**
```dockerfile
# Build stage
FROM golang:1.23 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Runtime stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
USER nobody
EXPOSE 8080
CMD ["./main"]
```

**Container Best Practices:**
- Keep images small (aim for < 50MB for static binaries)
- Use layer caching effectively
- Scan images for vulnerabilities
- Implement proper signal handling for graceful shutdowns
- Use health checks and readiness probes
- Set resource limits in orchestration platforms
- Use secrets management for sensitive data

---

## Architectural Review Responsibilities

When performing architectural reviews of Go code:

1. **Verify adherence to Go idioms**: Ensure code follows Go best practices and idioms
2. **Check error handling**: Validate that all errors are properly handled
3. **Review concurrency patterns**: Ensure goroutines and channels are used correctly
4. **Assess security**: Check for common security vulnerabilities
5. **Validate API design**: Ensure Web APIs follow RESTful principles
6. **Review logging**: Ensure structured logging is implemented correctly
7. **Check container configuration**: Review Dockerfiles for best practices
8. **Identify anti-patterns**: Look for common Go anti-patterns and misuse of interfaces
