// Package providercmd implements provider management functionality for the nomos CLI.
package providercmd

import (
	"fmt"
	"os"

	"github.com/autonomous-bits/nomos/libs/parser"
	"github.com/autonomous-bits/nomos/libs/parser/pkg/ast"
)

// DiscoverProviders scans .csl files and extracts provider requirements.
// It parses each file and extracts SourceDecl nodes, converting them to
// DiscoveredProvider structs. Duplicate provider aliases are automatically
// deduplicated (first occurrence wins).
//
// Returns a slice of discovered providers and any parsing errors encountered.
func DiscoverProviders(paths []string) ([]DiscoveredProvider, error) {
	providers := []DiscoveredProvider{}
	seen := make(map[string]bool)

	for _, path := range paths {
		//nolint:gosec // G304: Path comes from user CLI input, intentional file inclusion
		file, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("failed to open %s: %w", path, err)
		}
		defer func() { _ = file.Close() }()

		tree, err := parser.Parse(file, path)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", path, err)
		}

		// Extract source declarations from AST
		for _, stmt := range tree.Statements {
			srcDecl, ok := stmt.(*ast.SourceDecl)
			if !ok {
				continue
			}

			// Skip duplicates
			if seen[srcDecl.Alias] {
				continue
			}
			seen[srcDecl.Alias] = true

			// Convert config expressions to values
			config := make(map[string]any)
			for k, expr := range srcDecl.Config {
				config[k] = exprToValue(expr)
			}

			// Use the Version field from SourceDecl (not from Config map)
			version := srcDecl.Version

			providers = append(providers, DiscoveredProvider{
				Alias:   srcDecl.Alias,
				Type:    srcDecl.Type,
				Version: version,
				Config:  config,
			})
		}
	}

	return providers, nil
}
