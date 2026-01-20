// Package parser_test provides benchmarks for parser performance testing.
package parser_test

import (
	"bytes"
	"fmt"
	"os"
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

// BenchmarkParse_TypicalConfigWithoutComments benchmarks parsing a typical configuration
// file (~2-3KB) without any comments. This serves as a baseline for measuring the performance
// impact of comment parsing.
func BenchmarkParse_TypicalConfigWithoutComments(b *testing.B) {
	var builder strings.Builder

	builder.WriteString("source:\n")
	builder.WriteString("  alias: typicalConfig\n")
	builder.WriteString("  type: yaml\n\n")

	builder.WriteString("import:baseConfig:./base.csl\n")
	builder.WriteString("import:sharedTypes:./types.csl\n\n")

	builder.WriteString("database:\n")
	builder.WriteString("  host: localhost\n")
	builder.WriteString("  port: 5432\n")
	builder.WriteString("  username: dbuser\n")
	builder.WriteString("  password: reference:secrets:db.password\n")
	builder.WriteString("  maxConnections: 100\n")
	builder.WriteString("  timeout: 30\n\n")

	builder.WriteString("api:\n")
	builder.WriteString("  endpoint: https://api.example.com\n")
	builder.WriteString("  version: v2\n")
	builder.WriteString("  timeout: 60\n")
	builder.WriteString("  retries: 3\n")
	builder.WriteString("  authentication:\n")
	builder.WriteString("    type: bearer\n")
	builder.WriteString("    token: reference:secrets:api.token\n\n")

	builder.WriteString("logging:\n")
	builder.WriteString("  level: info\n")
	builder.WriteString("  format: json\n")
	builder.WriteString("  stdout: true\n")
	builder.WriteString("  file: /var/log/app.log\n\n")

	builder.WriteString("cache:\n")
	builder.WriteString("  type: redis\n")
	builder.WriteString("  host: localhost\n")
	builder.WriteString("  port: 6379\n")
	builder.WriteString("  ttl: 3600\n\n")

	builder.WriteString("features:\n")
	builder.WriteString("  enableNewUI: true\n")
	builder.WriteString("  enableBetaFeatures: false\n")
	builder.WriteString("  experimentalMode: false\n\n")

	for i := 0; i < 20; i++ {
		builder.WriteString(fmt.Sprintf("service%d:\n", i))
		builder.WriteString(fmt.Sprintf("  name: service-%d\n", i))
		builder.WriteString(fmt.Sprintf("  port: %d\n", 8000+i))
		builder.WriteString("  enabled: true\n")
		builder.WriteString("  healthCheck: /health\n")
		builder.WriteString("  timeout: 30\n\n")
	}

	source := builder.String()
	b.Logf("File size: %d bytes, Comment count: 0", len(source))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader([]byte(source))
		_, err := parser.Parse(reader, "test.csl")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParse_TypicalConfigWith100Comments benchmarks parsing a typical configuration
// file (~2-3KB) with approximately 100 comment lines interspersed throughout. This measures
// the performance impact of parsing a realistically commented configuration file.
func BenchmarkParse_TypicalConfigWith100Comments(b *testing.B) {
	var builder strings.Builder

	builder.WriteString("# Configuration file for typical application\n")
	builder.WriteString("# Generated: 2026-01-18\n")
	builder.WriteString("# Environment: production\n\n")

	builder.WriteString("source:\n")
	builder.WriteString("  alias: typicalConfig  # Unique identifier\n")
	builder.WriteString("  type: yaml  # Configuration format\n\n")

	builder.WriteString("# Import base configurations\n")
	builder.WriteString("import:baseConfig:./base.csl\n")
	builder.WriteString("# Import shared type definitions\n")
	builder.WriteString("import:sharedTypes:./types.csl\n\n")

	builder.WriteString("# Database configuration\n")
	builder.WriteString("# Connection settings for PostgreSQL\n")
	builder.WriteString("database:\n")
	builder.WriteString("  host: localhost  # Database server hostname\n")
	builder.WriteString("  port: 5432  # Standard PostgreSQL port\n")
	builder.WriteString("  username: dbuser  # Application database user\n")
	builder.WriteString("  password: reference:secrets:db.password  # Stored securely\n")
	builder.WriteString("  maxConnections: 100  # Connection pool size\n")
	builder.WriteString("  timeout: 30  # Connection timeout in seconds\n\n")

	builder.WriteString("# API configuration\n")
	builder.WriteString("# External API integration settings\n")
	builder.WriteString("api:\n")
	builder.WriteString("  endpoint: https://api.example.com  # Base API URL\n")
	builder.WriteString("  version: v2  # API version\n")
	builder.WriteString("  timeout: 60  # Request timeout in seconds\n")
	builder.WriteString("  retries: 3  # Number of retry attempts\n")
	builder.WriteString("  # Authentication configuration\n")
	builder.WriteString("  authentication:\n")
	builder.WriteString("    type: bearer  # OAuth 2.0 bearer token\n")
	builder.WriteString("    token: reference:secrets:api.token  # Token from secrets\n\n")

	builder.WriteString("# Logging configuration\n")
	builder.WriteString("# Centralized logging settings\n")
	builder.WriteString("logging:\n")
	builder.WriteString("  level: info  # Log level: debug, info, warn, error\n")
	builder.WriteString("  format: json  # Structured logging format\n")
	builder.WriteString("  stdout: true  # Console output\n")
	builder.WriteString("  file: /var/log/app.log  # File output\n\n")

	builder.WriteString("# Cache configuration\n")
	builder.WriteString("# Redis cache settings\n")
	builder.WriteString("cache:\n")
	builder.WriteString("  type: redis  # Cache backend type\n")
	builder.WriteString("  host: localhost  # Redis server hostname\n")
	builder.WriteString("  port: 6379  # Standard Redis port\n")
	builder.WriteString("  ttl: 3600  # Time to live in seconds\n\n")

	builder.WriteString("# Feature flags\n")
	builder.WriteString("# Toggle features without deployment\n")
	builder.WriteString("features:\n")
	builder.WriteString("  enableNewUI: true  # New UI rollout\n")
	builder.WriteString("  enableBetaFeatures: false  # Beta features disabled\n")
	builder.WriteString("  experimentalMode: false  # Experimental mode off\n\n")

	builder.WriteString("# Microservices configuration\n")
	builder.WriteString("# Individual service definitions\n")
	for i := 0; i < 20; i++ {
		builder.WriteString(fmt.Sprintf("# Service %d configuration\n", i))
		builder.WriteString(fmt.Sprintf("service%d:\n", i))
		builder.WriteString(fmt.Sprintf("  name: service-%d  # Service identifier\n", i))
		builder.WriteString(fmt.Sprintf("  port: %d  # Service port\n", 8000+i))
		builder.WriteString("  enabled: true  # Service enabled\n")
		builder.WriteString("  healthCheck: /health  # Health endpoint\n")
		builder.WriteString("  timeout: 30  # Request timeout\n\n")
	}

	source := builder.String()
	commentCount := strings.Count(source, "#")
	b.Logf("File size: %d bytes, Comment count: %d", len(source), commentCount)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader([]byte(source))
		_, err := parser.Parse(reader, "test.csl")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParse_ConfigWith1000Comments benchmarks parsing a configuration file with
// 1000+ consecutive comment lines. This tests the worst-case scenario for comment parsing
// performance, such as heavily documented configuration files or auto-generated documentation.
func BenchmarkParse_ConfigWith1000Comments(b *testing.B) {
	var builder strings.Builder

	builder.WriteString("###############################################################################\n")
	builder.WriteString("# CONFIGURATION FILE DOCUMENTATION\n")
	builder.WriteString("###############################################################################\n")
	builder.WriteString("#\n")
	builder.WriteString("# This configuration file contains extensive documentation to demonstrate\n")
	builder.WriteString("# comment parsing performance with a large number of consecutive comments.\n")
	builder.WriteString("#\n")
	builder.WriteString("# TABLE OF CONTENTS:\n")
	builder.WriteString("#   1. Source Configuration\n")
	builder.WriteString("#   2. Import Declarations\n")
	builder.WriteString("#   3. Database Configuration\n")
	builder.WriteString("#   4. API Configuration\n")
	builder.WriteString("#   5. Caching Strategy\n")
	builder.WriteString("#   6. Logging Settings\n")
	builder.WriteString("#   7. Security Policies\n")
	builder.WriteString("#   8. Performance Tuning\n")
	builder.WriteString("#   9. Monitoring and Observability\n")
	builder.WriteString("#  10. Feature Flags\n")
	builder.WriteString("#\n")

	for i := 0; i < 100; i++ {
		builder.WriteString(fmt.Sprintf("# Documentation block %d: Detailed explanation of configuration options\n", i))
		builder.WriteString("# This section covers aspects of the system configuration that are\n")
		builder.WriteString("# critical for understanding deployment requirements and best practices.\n")
		builder.WriteString(fmt.Sprintf("# Reference: docs/config-guide.md#section-%d\n", i))
		builder.WriteString("#\n")
	}

	builder.WriteString("###############################################################################\n")
	builder.WriteString("# 1. SOURCE CONFIGURATION\n")
	builder.WriteString("###############################################################################\n")
	builder.WriteString("#\n")
	builder.WriteString("# The source block defines metadata about this configuration file including\n")
	builder.WriteString("# its alias (unique identifier), type (format), and version information.\n")
	builder.WriteString("#\n")
	builder.WriteString("# Properties:\n")
	builder.WriteString("#   - alias: Unique identifier for this configuration\n")
	builder.WriteString("#   - type: Configuration format (yaml, json, etc.)\n")
	builder.WriteString("#   - version: Semantic version of this configuration\n")
	builder.WriteString("#\n")

	for i := 0; i < 50; i++ {
		builder.WriteString(fmt.Sprintf("# Additional source documentation line %d\n", i))
	}

	builder.WriteString("source:\n")
	builder.WriteString("  alias: heavilyDocumentedConfig\n")
	builder.WriteString("  type: yaml\n")
	builder.WriteString("  version: 1.0.0\n\n")

	builder.WriteString("###############################################################################\n")
	builder.WriteString("# 2. IMPORT DECLARATIONS\n")
	builder.WriteString("###############################################################################\n")
	builder.WriteString("#\n")
	builder.WriteString("# Import statements allow you to reference external configuration files,\n")
	builder.WriteString("# promoting reusability and modular configuration design.\n")
	builder.WriteString("#\n")

	for i := 0; i < 50; i++ {
		builder.WriteString(fmt.Sprintf("# Import documentation line %d\n", i))
	}

	builder.WriteString("import:baseConfig:./base.csl\n")
	builder.WriteString("import:sharedTypes:./types.csl\n\n")

	builder.WriteString("###############################################################################\n")
	builder.WriteString("# 3. DATABASE CONFIGURATION\n")
	builder.WriteString("###############################################################################\n")
	builder.WriteString("#\n")
	builder.WriteString("# Database connection settings for the application's primary data store.\n")
	builder.WriteString("# Supports PostgreSQL, MySQL, and other relational databases.\n")
	builder.WriteString("#\n")

	for i := 0; i < 100; i++ {
		builder.WriteString(fmt.Sprintf("# Database configuration detail line %d\n", i))
	}

	builder.WriteString("database:\n")
	builder.WriteString("  host: localhost\n")
	builder.WriteString("  port: 5432\n")
	builder.WriteString("  username: dbuser\n")
	builder.WriteString("  password: reference:secrets:db.password\n\n")

	builder.WriteString("###############################################################################\n")
	builder.WriteString("# 4. API CONFIGURATION\n")
	builder.WriteString("###############################################################################\n")
	builder.WriteString("#\n")
	builder.WriteString("# External API integration settings including endpoints, authentication,\n")
	builder.WriteString("# retry policies, and timeout configurations.\n")
	builder.WriteString("#\n")

	for i := 0; i < 100; i++ {
		builder.WriteString(fmt.Sprintf("# API configuration detail line %d\n", i))
	}

	builder.WriteString("api:\n")
	builder.WriteString("  endpoint: https://api.example.com\n")
	builder.WriteString("  version: v2\n")
	builder.WriteString("  timeout: 60\n\n")

	builder.WriteString("###############################################################################\n")
	builder.WriteString("# 5. CACHING STRATEGY\n")
	builder.WriteString("###############################################################################\n")
	builder.WriteString("#\n")
	builder.WriteString("# Cache configuration for improving application performance and reducing\n")
	builder.WriteString("# load on backend services.\n")
	builder.WriteString("#\n")

	for i := 0; i < 100; i++ {
		builder.WriteString(fmt.Sprintf("# Cache configuration detail line %d\n", i))
	}

	builder.WriteString("cache:\n")
	builder.WriteString("  type: redis\n")
	builder.WriteString("  host: localhost\n")
	builder.WriteString("  port: 6379\n\n")

	builder.WriteString("###############################################################################\n")
	builder.WriteString("# 6. LOGGING SETTINGS\n")
	builder.WriteString("###############################################################################\n")
	builder.WriteString("#\n")
	builder.WriteString("# Centralized logging configuration for application observability.\n")
	builder.WriteString("#\n")

	for i := 0; i < 100; i++ {
		builder.WriteString(fmt.Sprintf("# Logging configuration detail line %d\n", i))
	}

	builder.WriteString("logging:\n")
	builder.WriteString("  level: info\n")
	builder.WriteString("  format: json\n\n")

	builder.WriteString("###############################################################################\n")
	builder.WriteString("# 7. FEATURE FLAGS\n")
	builder.WriteString("###############################################################################\n")
	builder.WriteString("#\n")
	builder.WriteString("# Feature toggles for enabling/disabling functionality without redeployment.\n")
	builder.WriteString("#\n")

	for i := 0; i < 100; i++ {
		builder.WriteString(fmt.Sprintf("# Feature flags detail line %d\n", i))
	}

	builder.WriteString("features:\n")
	builder.WriteString("  enableNewUI: true\n")
	builder.WriteString("  enableBetaFeatures: false\n\n")

	builder.WriteString("###############################################################################\n")
	builder.WriteString("# END OF CONFIGURATION\n")
	builder.WriteString("###############################################################################\n")

	for i := 0; i < 100; i++ {
		builder.WriteString(fmt.Sprintf("# Footer documentation line %d\n", i))
	}

	source := builder.String()
	commentCount := strings.Count(source, "#")
	b.Logf("File size: %d bytes, Comment count: %d", len(source), commentCount)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader([]byte(source))
		_, err := parser.Parse(reader, "test.csl")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseListSimple benchmarks parsing a simple list fixture.
func BenchmarkParseListSimple(b *testing.B) {
	path := "testdata/fixtures/lists/simple_list.csl"
	data, err := os.ReadFile(path)
	if err != nil {
		b.Fatalf("failed to read fixture %s: %v", path, err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(data)
		_, err := parser.Parse(reader, path)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseListNested benchmarks parsing a nested list fixture.
func BenchmarkParseListNested(b *testing.B) {
	path := "testdata/fixtures/lists/nested_lists.csl"
	data, err := os.ReadFile(path)
	if err != nil {
		b.Fatalf("failed to read fixture %s: %v", path, err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(data)
		_, err := parser.Parse(reader, path)
		if err != nil {
			b.Fatal(err)
		}
	}
}
