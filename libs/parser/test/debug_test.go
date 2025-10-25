package parser_test

import (
	"fmt"
	"testing"

	"github.com/autonomous-bits/nomos/libs/parser"
	"github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
)

// TestDebugParsing helps debug parsing issues
func TestDebugParsing(t *testing.T) {
	// Use EXACTLY the same format as the source test
	input5 := `infrastructure:
	key: 'value'
`

	result5, err5 := parser.Parse(newReader(input5), "debug5.csl")
	if err5 != nil {
		t.Fatalf("parse error: %v", err5)
	}

	fmt.Printf("\n=== Test with same format as source test ===\n")
	fmt.Printf("Total statements: %d\n", len(result5.Statements))
	for i, stmt := range result5.Statements {
		fmt.Printf("Statement %d: %T\n", i, stmt)
		if section, ok := stmt.(*ast.SectionDecl); ok {
			fmt.Printf("  Section name: %s\n", section.Name)
			fmt.Printf("  Entries count: %d\n", len(section.Entries))
			for key, expr := range section.Entries {
				fmt.Printf("    %s: %T\n", key, expr)
			}
		}
	}
}
