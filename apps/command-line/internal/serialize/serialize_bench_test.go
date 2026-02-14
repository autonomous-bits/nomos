package serialize

import (
	"fmt"
	"testing"
	"time"

	"github.com/autonomous-bits/nomos/libs/compiler"
)

// generateSmallSnapshot creates a snapshot with ~10 keys representing a simple configuration.
func generateSmallSnapshot() compiler.Snapshot {
	now := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)

	return compiler.Snapshot{
		Data: map[string]any{
			"environment": "production",
			"region":      "us-west-2",
			"version":     "1.2.3",
			"count":       5,
			"enabled":     true,
			"database": map[string]any{
				"host": "localhost",
				"port": 5432,
			},
			"tags":     []any{"app", "backend", "production"},
			"timeout":  30.5,
			"replicas": 3,
		},
		Metadata: compiler.Metadata{
			InputFiles:      []string{"config.csl"},
			ProviderAliases: []string{"file"},
			StartTime:       now,
			EndTime:         now.Add(100 * time.Millisecond),
			Errors:          []string{},
			Warnings:        []string{},
			PerKeyProvenance: map[string]compiler.Provenance{
				"environment": {Source: "config.csl", ProviderAlias: "file"},
				"region":      {Source: "config.csl", ProviderAlias: "file"},
			},
		},
	}
}

// generateMediumSnapshot creates a snapshot with ~100 keys representing a moderately complex configuration.
func generateMediumSnapshot() compiler.Snapshot {
	now := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)

	data := map[string]any{
		"environment": "production",
		"region":      "us-west-2",
		"version":     "1.2.3",
	}

	// Add networking configuration (20 keys)
	networking := map[string]any{
		"vpc_cidr":    "10.0.0.0/16",
		"vpn_enabled": true,
	}
	for i := 0; i < 10; i++ {
		networking[fmt.Sprintf("subnet_%d", i)] = map[string]any{
			"cidr":              fmt.Sprintf("10.0.%d.0/24", i),
			"availability_zone": fmt.Sprintf("us-west-2%c", 'a'+i%3),
		}
	}
	data["networking"] = networking

	// Add database configuration (30 keys)
	databases := map[string]any{}
	for i := 0; i < 10; i++ {
		databases[fmt.Sprintf("db_%d", i)] = map[string]any{
			"host":     fmt.Sprintf("db%d.example.com", i),
			"port":     5432 + i,
			"replicas": i%3 + 1,
		}
	}
	data["databases"] = databases

	// Add service configuration (30 keys)
	services := map[string]any{}
	for i := 0; i < 10; i++ {
		services[fmt.Sprintf("service_%d", i)] = map[string]any{
			"name":     fmt.Sprintf("svc-%d", i),
			"port":     8000 + i,
			"replicas": i%5 + 1,
		}
	}
	data["services"] = services

	// Add monitoring configuration (20 keys)
	monitoring := map[string]any{
		"enabled":  true,
		"interval": 60,
	}
	for i := 0; i < 9; i++ {
		monitoring[fmt.Sprintf("metric_%d", i)] = map[string]any{
			"name":      fmt.Sprintf("metric_%d", i),
			"threshold": float64(i) * 10.5,
		}
	}
	data["monitoring"] = monitoring

	// Build provenance map
	provenance := map[string]compiler.Provenance{}
	for key := range data {
		provenance[key] = compiler.Provenance{
			Source:        "config.csl",
			ProviderAlias: "file",
		}
	}

	return compiler.Snapshot{
		Data: data,
		Metadata: compiler.Metadata{
			InputFiles:       []string{"config.csl", "network.csl", "db.csl"},
			ProviderAliases:  []string{"file", "vault"},
			StartTime:        now,
			EndTime:          now.Add(500 * time.Millisecond),
			Errors:           []string{},
			Warnings:         []string{},
			PerKeyProvenance: provenance,
		},
	}
}

// generateLargeSnapshot creates a snapshot with ~1000 keys representing a complex real-world configuration.
func generateLargeSnapshot() compiler.Snapshot {
	now := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)

	data := map[string]any{
		"environment": "production",
		"region":      "us-west-2",
		"version":     "1.2.3",
	}

	// Add 100 compute instances (300 keys)
	compute := map[string]any{}
	for i := 0; i < 100; i++ {
		compute[fmt.Sprintf("instance_%d", i)] = map[string]any{
			"id":    fmt.Sprintf("i-%08d", i),
			"type":  "t3.medium",
			"state": "running",
		}
	}
	data["compute"] = compute

	// Add 100 network configurations (300 keys)
	networks := map[string]any{}
	for i := 0; i < 100; i++ {
		networks[fmt.Sprintf("network_%d", i)] = map[string]any{
			"cidr":       fmt.Sprintf("10.%d.0.0/16", i),
			"gateway":    fmt.Sprintf("10.%d.0.1", i),
			"dns_server": "8.8.8.8",
		}
	}
	data["networks"] = networks

	// Add 100 storage volumes (300 keys)
	storage := map[string]any{}
	for i := 0; i < 100; i++ {
		storage[fmt.Sprintf("volume_%d", i)] = map[string]any{
			"size":      i*10 + 100,
			"type":      "gp3",
			"encrypted": true,
		}
	}
	data["storage"] = storage

	// Add 50 security groups (100 keys)
	security := map[string]any{}
	for i := 0; i < 50; i++ {
		security[fmt.Sprintf("sg_%d", i)] = map[string]any{
			"name": fmt.Sprintf("security-group-%d", i),
			"rules": []any{
				map[string]any{"port": 443, "protocol": "tcp"},
				map[string]any{"port": 80, "protocol": "tcp"},
			},
		}
	}
	data["security"] = security

	// Build provenance map
	provenance := map[string]compiler.Provenance{}
	for key := range data {
		provenance[key] = compiler.Provenance{
			Source:        "config.csl",
			ProviderAlias: "file",
		}
	}

	return compiler.Snapshot{
		Data: data,
		Metadata: compiler.Metadata{
			InputFiles:       []string{"config.csl", "network.csl", "compute.csl", "storage.csl"},
			ProviderAliases:  []string{"file", "vault", "aws"},
			StartTime:        now,
			EndTime:          now.Add(2 * time.Second),
			Errors:           []string{},
			Warnings:         []string{"Large configuration may impact performance"},
			PerKeyProvenance: provenance,
		},
	}
}

// BenchmarkToJSON_Small benchmarks JSON serialization with small dataset (~10 keys).
func BenchmarkToJSON_Small(b *testing.B) {
	snapshot := generateSmallSnapshot()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := ToJSON(snapshot, true)
		if err != nil {
			b.Fatalf("ToJSON failed: %v", err)
		}
	}
}

// BenchmarkToJSON_Medium benchmarks JSON serialization with medium dataset (~100 keys).
func BenchmarkToJSON_Medium(b *testing.B) {
	snapshot := generateMediumSnapshot()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := ToJSON(snapshot, true)
		if err != nil {
			b.Fatalf("ToJSON failed: %v", err)
		}
	}
}

// BenchmarkToJSON_Large benchmarks JSON serialization with large dataset (~1000 keys).
func BenchmarkToJSON_Large(b *testing.B) {
	snapshot := generateLargeSnapshot()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := ToJSON(snapshot, true)
		if err != nil {
			b.Fatalf("ToJSON failed: %v", err)
		}
	}
}

// BenchmarkToYAML_Small benchmarks YAML serialization with small dataset (~10 keys).
func BenchmarkToYAML_Small(b *testing.B) {
	snapshot := generateSmallSnapshot()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := ToYAML(snapshot, true)
		if err != nil {
			b.Fatalf("ToYAML failed: %v", err)
		}
	}
}

// BenchmarkToYAML_Medium benchmarks YAML serialization with medium dataset (~100 keys).
func BenchmarkToYAML_Medium(b *testing.B) {
	snapshot := generateMediumSnapshot()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := ToYAML(snapshot, true)
		if err != nil {
			b.Fatalf("ToYAML failed: %v", err)
		}
	}
}

// BenchmarkToYAML_Large benchmarks YAML serialization with large dataset (~1000 keys).
func BenchmarkToYAML_Large(b *testing.B) {
	snapshot := generateLargeSnapshot()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := ToYAML(snapshot, true)
		if err != nil {
			b.Fatalf("ToYAML failed: %v", err)
		}
	}
}

// BenchmarkToTfvars_Small benchmarks tfvars serialization with small dataset (~10 keys).
func BenchmarkToTfvars_Small(b *testing.B) {
	snapshot := generateSmallSnapshot()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := ToTfvars(snapshot, true)
		if err != nil {
			b.Fatalf("ToTfvars failed: %v", err)
		}
	}
}

// BenchmarkToTfvars_Medium benchmarks tfvars serialization with medium dataset (~100 keys).
func BenchmarkToTfvars_Medium(b *testing.B) {
	snapshot := generateMediumSnapshot()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := ToTfvars(snapshot, true)
		if err != nil {
			b.Fatalf("ToTfvars failed: %v", err)
		}
	}
}

// BenchmarkToTfvars_Large benchmarks tfvars serialization with large dataset (~1000 keys).
func BenchmarkToTfvars_Large(b *testing.B) {
	snapshot := generateLargeSnapshot()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := ToTfvars(snapshot, true)
		if err != nil {
			b.Fatalf("ToTfvars failed: %v", err)
		}
	}
}
