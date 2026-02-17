package encryption

import (
	"encoding/base64"
	"os"
	"testing"
)

func TestGenerateKey(t *testing.T) {
	key, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}
	if len(key) != KeySize {
		t.Errorf("expected key length %d, got %d", KeySize, len(key))
	}

	// Check randomness (basic check)
	key2, _ := GenerateKey()
	if string(key) == string(key2) {
		t.Error("GenerateKey returned identical keys")
	}
}

func TestEncryptDecrypt(t *testing.T) {
	key, _ := GenerateKey()
	plaintext := []byte("secret message")

	// Encrypt
	ciphertext, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	if ciphertext == "" {
		t.Fatal("ciphertext is empty")
	}

	// Decrypt
	decrypted, err := Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("expected '%s', got '%s'", plaintext, decrypted)
	}
}

func TestDecrypt_InvalidInput(t *testing.T) {
	key, _ := GenerateKey()

	// Invalid base64
	_, err := Decrypt("invalid-base64!@#", key)
	if err == nil {
		t.Error("expected error for invalid base64")
	}

	// Ciphertext too short
	shortCiphertext := base64.StdEncoding.EncodeToString([]byte("short"))
	_, err = Decrypt(shortCiphertext, key)
	if err == nil {
		t.Error("expected error for short ciphertext")
	}

	// Wrong key
	wrongKey, _ := GenerateKey()
	ciphertext, _ := Encrypt([]byte("data"), key)
	_, err = Decrypt(ciphertext, wrongKey)
	if err == nil {
		t.Error("expected error for wrong key")
	}
}

func TestLoadKey(t *testing.T) {
	// Create temp key file
	key, _ := GenerateKey()
	encoded := base64.StdEncoding.EncodeToString(key)
	tmpFile, err := os.CreateTemp("", "key")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	if _, err := tmpFile.WriteString(encoded); err != nil {
		t.Fatal(err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatal(err)
	}

	// Load valid key
	loadedKey, err := LoadKey(tmpFile.Name())
	if err != nil {
		t.Fatalf("LoadKey failed: %v", err)
	}
	if string(loadedKey) != string(key) {
		t.Error("loaded key mismatch")
	}

	// Load non-existent file
	_, err = LoadKey("non-existent-file")
	if err == nil {
		t.Error("expected error for non-existent file")
	}

	// Load invalid content
	invalidFile, _ := os.CreateTemp("", "invalid")
	defer func() { _ = os.Remove(invalidFile.Name()) }()
	if _, err := invalidFile.WriteString("invalid-base64"); err != nil {
		t.Fatal(err)
	}
	_ = invalidFile.Close()

	_, err = LoadKey(invalidFile.Name())
	if err == nil {
		t.Error("expected error for invalid key file content")
	}

	// Load wrong size key
	shortKey := make([]byte, 10)
	shortEncoded := base64.StdEncoding.EncodeToString(shortKey)
	shortFile, _ := os.CreateTemp("", "short")
	defer func() { _ = os.Remove(shortFile.Name()) }()
	if _, err := shortFile.WriteString(shortEncoded); err != nil {
		t.Fatal(err)
	}
	_ = shortFile.Close()

	_, err = LoadKey(shortFile.Name())
	if err == nil {
		t.Error("expected error for wrong key size")
	}
}
