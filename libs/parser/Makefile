.PHONY: help test test-verbose test-race test-coverage update-golden bench lint clean

# Default target
help:
	@echo "Available targets:"
	@echo "  test           - Run all parser tests"
	@echo "  test-verbose   - Run tests with verbose output"
	@echo "  test-race      - Run tests with race detector"
	@echo "  test-coverage  - Run tests and generate coverage report"
	@echo "  update-golden  - Update golden test files (use carefully!)"
	@echo "  bench          - Run benchmark tests"
	@echo "  lint           - Run golangci-lint"
	@echo "  clean          - Clean generated files"

# Run all tests
test:
	@echo "Running parser tests..."
	go test -v ./...

# Run tests with verbose output
test-verbose:
	@echo "Running parser tests (verbose)..."
	go test -v -count=1 ./...

# Run tests with race detector
test-race:
	@echo "Running parser tests with race detector..."
	go test -race ./...

# Run tests and generate coverage report
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out -covermode=atomic ./...
	@echo "\nCoverage by package:"
	@go test -coverprofile=coverage.out -covermode=atomic ./... 2>&1 | grep -E "coverage:" | grep -v "no statements"
	@echo "\nOverall coverage summary:"
	go tool cover -func=coverage.out | tail -1
	@echo "\nNote: Tests in test/ directory use black-box testing (external package)"
	@echo "This is best practice but results in lower coverage percentage for main package."
	@echo "Scanner package (with co-located tests) shows actual coverage: ~86%"
	@echo "\nGenerating HTML coverage report..."
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report saved to coverage.html"

# Update golden test files (destructive - use with caution)
update-golden:
	@echo "WARNING: This will update all golden test files!"
	@echo "Press Ctrl+C to cancel, or wait 3 seconds to continue..."
	@sleep 3
	@echo "\nUpdating golden files..."
	@rm -f testdata/golden/*.json testdata/golden/errors/*.json 2>/dev/null || true
	go test -v ./test -run Golden
	@echo "\nGolden files updated. Please review changes before committing!"

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

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
