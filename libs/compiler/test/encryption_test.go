package test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/autonomous-bits/nomos/libs/compiler"
	"github.com/autonomous-bits/nomos/libs/compiler/pkg/encryption"
	"github.com/autonomous-bits/nomos/libs/compiler/testutil"
)

// TestCompile_Encryption_EndToEnd tests the encryption pipeline.
func TestCompile_Encryption_EndToEnd(t *testing.T) {
	// Generate a key
	key, err := encryption.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create a configuration file with a marked secret
	configContent := `config:
	secret: "my-secret-value"!
	public: "public-value"
`
	if err := os.WriteFile(tmpDir+"/config.csl", []byte(configContent), 0600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Setup compiler options with encryption key
	ctx := context.Background()
	opts := compiler.Options{
		Path:             tmpDir,
		ProviderRegistry: testutil.NewFakeProviderRegistry(),
		EncryptionKey:    key,
		Timeouts: compiler.OptionsTimeouts{
			PerProviderFetch: 5 * time.Second,
		},
	}

	// Compile
	result := compiler.Compile(ctx, opts)
	if result.HasErrors() {
		for _, e := range result.Errors() {
			t.Logf("Compile Error: %v", e)
		}
		t.Fatalf("compilation failed")
	}

	// Inspect the output data
	data := result.Snapshot.Data
	config, ok := data["config"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'config' map in data")
	}

	// Check public value (should be plaintext)
	publicVal, ok := config["public"].(string)
	if !ok || publicVal != "public-value" {
		t.Errorf("expected public='public-value', got '%v'", config["public"])
	}

	// Check secret value (should be encrypted string)
	secretVal, ok := config["secret"].(string)
	if !ok {
		t.Fatalf("expected secret to be a string (ciphertext), got %T", config["secret"])
	}

	if secretVal == "my-secret-value" {
		t.Fatal("secret value was NOT encrypted (matches plaintext)")
	}

	// Try to decrypt it to verify correctness
	decrypted, err := encryption.Decrypt(secretVal, key)
	if err != nil {
		t.Fatalf("failed to decrypt secret: %v", err)
	}

	if string(decrypted) != "my-secret-value" {
		t.Errorf("decrypted value mismatch: expected 'my-secret-value', got '%s'", string(decrypted))
	}
}
