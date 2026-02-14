//go:build integration
// +build integration

// Package integration provides integration tests for the parser.
package integration

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/autonomous-bits/nomos/libs/parser"
	"github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
)

// TestConcurrentParses_Smoke is a CI-friendly concurrency test with N=10.
func TestConcurrentParses_Smoke(t *testing.T) {
	testConcurrentParses(t, 10, "smoke test (CI)")
}

// TestConcurrentParses_Stress is a heavier local stress test with N=100.
// Run manually with: go test -tags=integration -run TestConcurrentParses_Stress -timeout 30s
func TestConcurrentParses_Stress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}
	testConcurrentParses(t, 100, "stress test (local)")
}

// testConcurrentParses is the shared implementation for concurrency tests.
func testConcurrentParses(t *testing.T, numGoroutines int, testName string) {
	t.Helper()

	// Sample Nomos source to parse
	// Note: Updated to use inline references instead of deprecated top-level reference statements
	source := `source:
  alias: myConfig
  type: yaml

import:baseConfig:./base.csl

database:
  host: localhost
  port: 5432
  connection: @base:config.database
`

	// Create a wait group
	var wg sync.WaitGroup
	errChan := make(chan error, numGoroutines)

	// Launch concurrent parse operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			// Each goroutine parses the same source
			filename := fmt.Sprintf("test-%d.csl", goroutineID)
			reader := bytes.NewReader([]byte(source))

			result, err := parser.Parse(reader, filename)
			if err != nil {
				errChan <- fmt.Errorf("goroutine %d: parse failed: %w", goroutineID, err)
				return
			}

			// Validate AST is not nil
			if result == nil {
				errChan <- fmt.Errorf("goroutine %d: ast is nil", goroutineID)
				return
			}

			// Validate AST has expected statements
			if len(result.Statements) == 0 {
				errChan <- fmt.Errorf("goroutine %d: expected statements, got none", goroutineID)
				return
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errChan)

	// Check for errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		t.Errorf("%s with %d goroutines failed with %d errors:", testName, numGoroutines, len(errors))
		for _, err := range errors {
			t.Errorf("  - %v", err)
		}
	} else {
		t.Logf("%s with %d goroutines passed", testName, numGoroutines)
	}
}

// TestConcurrentParses_LargeFile tests parsing of a 1MB file.
func TestConcurrentParses_LargeFile(t *testing.T) {
	// Generate a 1MB Nomos source file
	var builder strings.Builder
	builder.WriteString("source:\n")
	builder.WriteString("  alias: bigConfig\n")
	builder.WriteString("  type: yaml\n\n")

	// Add many sections to reach ~1MB
	// Each section is roughly 60 bytes, so we need ~17,500 sections
	for i := 0; i < 17500; i++ {
		builder.WriteString(fmt.Sprintf("section%d:\n", i))
		builder.WriteString(fmt.Sprintf("  key%d: value%d\n", i, i))
		builder.WriteString(fmt.Sprintf("  data%d: test-data-%d\n", i, i))
	}

	source := builder.String()
	sourceSize := len(source)

	t.Logf("Generated test file of size: %d bytes (%.2f MB)", sourceSize, float64(sourceSize)/(1024*1024))

	if sourceSize < 1024*1024 {
		t.Errorf("Expected file size >= 1MB, got %d bytes", sourceSize)
	}

	// Parse the large file concurrently
	numGoroutines := 10
	var wg sync.WaitGroup
	errChan := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			filename := fmt.Sprintf("large-test-%d.csl", goroutineID)
			reader := bytes.NewReader([]byte(source))

			result, err := parser.Parse(reader, filename)
			if err != nil {
				errChan <- fmt.Errorf("goroutine %d: parse failed: %w", goroutineID, err)
				return
			}

			// Validate AST
			if result == nil {
				errChan <- fmt.Errorf("goroutine %d: ast is nil", goroutineID)
				return
			}

			// Should have source declaration + many sections
			if len(result.Statements) < 2 {
				errChan <- fmt.Errorf("goroutine %d: expected many statements, got %d", goroutineID, len(result.Statements))
				return
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		t.Errorf("Large file test failed with %d errors:", len(errors))
		for _, err := range errors {
			t.Errorf("  - %v", err)
		}
	} else {
		t.Logf("Large file test with %d concurrent goroutines passed", numGoroutines)
	}
}

// TestParserInstance_IsolatedState tests that parser instances don't interfere.
func TestParserInstance_IsolatedState(t *testing.T) {
	source1 := `source:
  alias: config1
  type: yaml
`
	source2 := `source:
  alias: config2
  type: json
`

	// Create two parser instances
	p1 := parser.NewParser()
	p2 := parser.NewParser()

	// Use them concurrently
	var wg sync.WaitGroup
	results := make(chan string, 2)
	errors := make(chan error, 2)

	wg.Add(2)

	// Parse with first instance
	go func() {
		defer wg.Done()
		r := bytes.NewReader([]byte(source1))
		result, err := p1.Parse(r, "test1.csl")
		if err != nil {
			errors <- err
			return
		}
		// Extract alias from source declaration
		if len(result.Statements) > 0 {
			if sourceDecl, ok := result.Statements[0].(*ast.SourceDecl); ok {
				results <- sourceDecl.Alias
				return
			}
		}
		errors <- fmt.Errorf("no source declaration found in ast1")
	}()

	// Parse with second instance
	go func() {
		defer wg.Done()
		r := bytes.NewReader([]byte(source2))
		result, err := p2.Parse(r, "test2.csl")
		if err != nil {
			errors <- err
			return
		}
		// Extract alias from source declaration
		if len(result.Statements) > 0 {
			if sourceDecl, ok := result.Statements[0].(*ast.SourceDecl); ok {
				results <- sourceDecl.Alias
				return
			}
		}
		errors <- fmt.Errorf("no source declaration found in ast2")
	}()

	wg.Wait()
	close(results)
	close(errors)

	// Check for errors
	var errs []error
	for err := range errors {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		t.Errorf("Parser instance isolation test failed with errors:")
		for _, err := range errs {
			t.Errorf("  - %v", err)
		}
		return
	}

	// Validate results
	aliases := make(map[string]bool)
	for alias := range results {
		aliases[alias] = true
	}

	if !aliases["config1"] {
		t.Errorf("Expected alias 'config1' not found")
	}
	if !aliases["config2"] {
		t.Errorf("Expected alias 'config2' not found")
	}

	if len(aliases) != 2 {
		t.Errorf("Expected 2 distinct aliases, got %d: %v", len(aliases), aliases)
	}

	t.Log("Parser instance isolation test passed")
}
