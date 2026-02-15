package parser_test

import "github.com/autonomous-bits/nomos/libs/parser/pkg/ast"

func findEntry(entries []ast.MapEntry, key string) (ast.Expr, bool) {
	for _, entry := range entries {
		if entry.Spread {
			continue
		}
		if entry.Key == key {
			return entry.Value, true
		}
	}
	return nil, false
}

func hasEntry(entries []ast.MapEntry, key string) bool {
	_, ok := findEntry(entries, key)
	return ok
}

func entryMap(entries []ast.MapEntry) map[string]ast.Expr {
	result := make(map[string]ast.Expr, len(entries))
	for _, entry := range entries {
		if entry.Spread {
			continue
		}
		result[entry.Key] = entry.Value
	}
	return result
}
