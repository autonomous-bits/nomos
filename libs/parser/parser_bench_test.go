// Package parser_test provides benchmarks for parser performance testing.
package parser_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/autonomous-bits/nomos/libs/parser"
)

// BenchmarkParse_Small benchmarks parsing of a small configuration file.
func BenchmarkParse_Small(b *testing.B) {
	source := `source:
  alias: myConfig
  type: yaml

import:baseConfig:./base.csl

database:
  host: localhost
  port: 5432
  connection: reference:base:config.database
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader([]byte(source))
		_, err := parser.Parse(reader, "test.csl")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParse_Medium benchmarks parsing of a medium-sized configuration file.
func BenchmarkParse_Medium(b *testing.B) {
	var builder strings.Builder
	builder.WriteString("source:\n")
	builder.WriteString("  alias: mediumConfig\n")
	builder.WriteString("  type: yaml\n\n")

	// Add 100 sections (~6KB)
	for i := 0; i < 100; i++ {
		builder.WriteString(fmt.Sprintf("section%d:\n", i))
		builder.WriteString(fmt.Sprintf("  key%d: value%d\n", i, i))
		builder.WriteString(fmt.Sprintf("  data%d: test-data-%d\n", i, i))
	}

	source := builder.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader([]byte(source))
		_, err := parser.Parse(reader, "test.csl")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParse_Large benchmarks parsing of a large configuration file (~1MB).
func BenchmarkParse_Large(b *testing.B) {
	var builder strings.Builder
	builder.WriteString("source:\n")
	builder.WriteString("  alias: largeConfig\n")
	builder.WriteString("  type: yaml\n\n")

	// Add ~17,500 sections to reach ~1MB
	for i := 0; i < 17500; i++ {
		builder.WriteString(fmt.Sprintf("section%d:\n", i))
		builder.WriteString(fmt.Sprintf("  key%d: value%d\n", i, i))
		builder.WriteString(fmt.Sprintf("  data%d: test-data-%d\n", i, i))
	}

	source := builder.String()
	b.Logf("Benchmark file size: %.2f MB", float64(len(source))/(1024*1024))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader([]byte(source))
		_, err := parser.Parse(reader, "test.csl")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseFile benchmarks parsing from the filesystem.
func BenchmarkParseFile(b *testing.B) {
	// Use an existing test fixture
	path := "testdata/fixtures/all_grammar.csl"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.ParseFile(path)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParser_Reuse benchmarks parser instance reuse.
func BenchmarkParser_Reuse(b *testing.B) {
	source := `source:
  alias: myConfig
  type: yaml

database:
  host: localhost
  port: 5432
`

	p := parser.NewParser()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader([]byte(source))
		_, err := p.Parse(reader, "test.csl")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParser_NewEachTime benchmarks creating a new parser for each parse.
func BenchmarkParser_NewEachTime(b *testing.B) {
	source := `source:
  alias: myConfig
  type: yaml

database:
  host: localhost
  port: 5432
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader([]byte(source))
		_, err := parser.Parse(reader, "test.csl")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParse_Parallel benchmarks concurrent parsing.
func BenchmarkParse_Parallel(b *testing.B) {
	source := `source:
  alias: myConfig
  type: yaml

database:
  host: localhost
  port: 5432
`

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			reader := bytes.NewReader([]byte(source))
			_, err := parser.Parse(reader, "test.csl")
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
