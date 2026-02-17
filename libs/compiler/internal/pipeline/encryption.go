package pipeline

import (
	"encoding/json"
	"fmt"

	"github.com/autonomous-bits/nomos/libs/compiler/internal/models"
	"github.com/autonomous-bits/nomos/libs/compiler/pkg/encryption"
)

// EncryptSecrets traverses the data map and replaces models.Secret values
// with encrypted strings using the provided key.
func EncryptSecrets(data map[string]any, key []byte) (map[string]any, error) {
	if len(key) == 0 {
		return nil, fmt.Errorf("encryption key is required")
	}

	result, err := traverseAndEncrypt(data, key)
	if err != nil {
		return nil, err
	}

	return result.(map[string]any), nil
}

func traverseAndEncrypt(val any, key []byte) (any, error) {
	switch v := val.(type) {
	case models.Secret:
		// Encrypt the inner value
		// First, serialize value to functionality equivalent bytes (e.g. JSON)
		// Usually secrets are strings, but could be anything.
		var plaintext []byte
		var err error

		if str, ok := v.Value.(string); ok {
			plaintext = []byte(str)
		} else {
			plaintext, err = json.Marshal(v.Value)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal secret value: %w", err)
			}
		}

		ciphertext, err := encryption.Encrypt(plaintext, key)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt secret: %w", err)
		}
		return ciphertext, nil

	case map[string]any:
		newMap := make(map[string]any, len(v))
		for k, val := range v {
			encryptedVal, err := traverseAndEncrypt(val, key)
			if err != nil {
				return nil, err
			}
			newMap[k] = encryptedVal
		}
		return newMap, nil

	case []any:
		newSlice := make([]any, len(v))
		for i, val := range v {
			encryptedVal, err := traverseAndEncrypt(val, key)
			if err != nil {
				return nil, err
			}
			newSlice[i] = encryptedVal
		}
		return newSlice, nil

	default:
		return val, nil
	}
}
