.PHONY: help test test-verbose test-race test-coverage test-integration update-golden bench lint clean

# Default target
help:
	@echo "Available targets:"
	@echo "  test              - Run all compiler tests (excluding integration)"
	@echo "  test-verbose      - Run tests with verbose output"
	@echo "  test-race         - Run tests with race detector"
	@echo "  test-coverage     - Run tests and generate coverage report"
	@echo "  test-integration  - Run integration tests (including network-backed)"
	@echo "  update-golden     - Update golden test files (use carefully!)"
	@echo "  bench             - Run benchmark tests"
	@echo "  lint              - Run golangci-lint"
	@echo "  clean             - Clean generated files"

# Run all tests (excluding integration)
test:
	@echo "Running compiler tests..."
	go test -v ./...

# Run tests with verbose output
test-verbose:
	@echo "Running compiler tests (verbose)..."
	go test -v -count=1 ./...

# Run tests with race detector
test-race:
	@echo "Running compiler tests with race detector..."
	go test -v -race ./...

# Run tests and generate coverage report
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out -covermode=atomic ./...
	@echo "\nCoverage summary:"
	go tool cover -func=coverage.out | tail -1
	@echo "\nGenerating HTML coverage report..."
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report saved to coverage.html"

# Run integration tests (including network-backed)
test-integration:
	@echo "Running integration tests (including network-backed)..."
	@echo "WARNING: These tests may make real network calls"
	go test -v -tags=integration ./test

# Update golden test files
# CAUTION: This will regenerate all golden files from current test outputs
update-golden:
	@echo "WARNING: This will update all golden test files!"
	@echo "Press Ctrl+C to cancel, or wait 3 seconds to continue..."
	@sleep 3
	@echo "\nUpdating golden files..."
	GOLDEN_UPDATE=1 go test -v ./test -run=Golden
	@echo "\nGolden files updated. Please review changes before committing!"

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./test/bench

# Run benchmarks with detailed output
bench-verbose:
	@echo "Running benchmarks (verbose)..."
	go test -bench=. -benchmem -benchtime=10s ./test/bench

# Run linter
lint:
	@echo "Running golangci-lint..."
	golangci-lint run ./...

# Clean generated files
clean:
	@echo "Cleaning generated files..."
	rm -f coverage.out coverage.html
	go clean -cache -testcache
	@echo "Clean complete"

# Run all quality checks (tests, race, coverage, lint)
check-all: test-race test-coverage lint
	@echo "All quality checks passed!"
