package server

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
)

// LoadOrGenerateEncryptionKey loads an existing encryption key or generates a new one.
// The key is stored securely in ~/.claude/wee/mfa_encryption.key with 0600 permissions.
// This ensures each installation has a unique encryption key for MFA secrets.
func LoadOrGenerateEncryptionKey(configDir string) ([]byte, error) {
	keyPath := filepath.Join(configDir, "mfa_encryption.key")

	// Try to load existing key
	if data, err := os.ReadFile(keyPath); err == nil {
		// Validate key length
		if len(data) != 32 {
			return nil, fmt.Errorf("invalid encryption key length: expected 32 bytes, got %d", len(data))
		}
		return data, nil
	}

	// Generate new 256-bit (32-byte) key for AES-256
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate encryption key: %w", err)
	}

	// Ensure config directory exists with restricted permissions (user only)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Save key with restrictive permissions (user read/write only)
	if err := os.WriteFile(keyPath, key, 0600); err != nil {
		return nil, fmt.Errorf("failed to save encryption key: %w", err)
	}

	return key, nil
}
