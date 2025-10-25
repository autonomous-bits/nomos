package parser_test

import (
	"fmt"
	"testing"

	"github.com/autonomous-bits/nomos/libs/parser/internal/scanner"
)

// TestScannerReadValue tests what ReadValue returns
func TestScannerReadValue(t *testing.T) {
	input := `key: reference:network:vpc.cidr
	`
	s := scanner.New(input, "test.csl")

	// Read "key"
	key := s.ReadIdentifier()
	fmt.Printf("Key: %q\n", key)

	// Expect ':'
	s.Expect(':')

	// Skip whitespace
	s.SkipWhitespace()

	// Read value
	value := s.ReadValue()
	fmt.Printf("Value: %q\n", value)

	if value != "reference:network:vpc.cidr" {
		t.Errorf("expected 'reference:network:vpc.cidr', got %q", value)
	}
}
