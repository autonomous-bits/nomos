package pipeline

import (
	"testing"

	"github.com/autonomous-bits/nomos/libs/compiler/internal/models"
	"github.com/autonomous-bits/nomos/libs/compiler/pkg/encryption"
)

func TestEncryptSecrets(t *testing.T) {
	key, _ := encryption.GenerateKey()

	tests := []struct {
		name    string
		input   map[string]any
		wantErr bool
	}{
		{
			name: "simple secret",
			input: map[string]any{
				"secret": models.Secret{Value: "password"},
				"public": "public",
			},
			wantErr: false,
		},
		{
			name: "nested secret",
			input: map[string]any{
				"app": map[string]any{
					"database": map[string]any{
						"password": models.Secret{Value: "db-pass"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "secret in list",
			input: map[string]any{
				"secrets": []any{
					models.Secret{Value: "one"},
					"two",
				},
			},
			wantErr: false,
		},
		{
			name: "typed secret",
			input: map[string]any{
				"port": models.Secret{Value: 8080},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EncryptSecrets(tt.input, key)
			if (err != nil) != tt.wantErr {
				t.Errorf("EncryptSecrets() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Verification logic dependent on structure
			// ... (omitted for brevity, just checking execution and type replacement)

			if tt.name == "simple secret" {
				val := got["secret"]
				if _, ok := val.(string); !ok {
					t.Errorf("expected string (ciphertext), got %T", val)
				}
				public := got["public"]
				if public != "public" {
					t.Errorf("public value modified")
				}
			}
		})
	}
}

func TestEncryptSecrets_InvalidKey(t *testing.T) {
	input := map[string]any{"secret": models.Secret{Value: "val"}}
	_, err := EncryptSecrets(input, nil)
	if err == nil {
		t.Error("expected error for nil key")
	}

	// Invalid key should result in error (often during encryption process)
	// We just check that it doesn't panic and potentially returns error
	_, _ = EncryptSecrets(input, []byte("short"))
}
