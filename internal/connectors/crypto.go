// Package connectors provides connector management for external service integrations.
// This file implements AES-256-GCM encryption for securely storing API keys.
package connectors

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// EncryptionPrefix marks encrypted values in the database
const EncryptionPrefix = "enc:"

// LoadOrGenerateConnectorKey loads or generates an AES-256 encryption key for connectors.
// The key is stored at ~/.claude/wee/connector_encryption.key with 0600 permissions.
func LoadOrGenerateConnectorKey(configDir string) ([]byte, error) {
	keyPath := filepath.Join(configDir, "connector_encryption.key")

	// Try to load existing key
	if data, err := os.ReadFile(keyPath); err == nil {
		if len(data) != 32 {
			return nil, fmt.Errorf("invalid connector encryption key length: expected 32 bytes, got %d", len(data))
		}
		return data, nil
	}

	// Generate new 256-bit key
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate connector encryption key: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Save with restrictive permissions
	if err := os.WriteFile(keyPath, key, 0600); err != nil {
		return nil, fmt.Errorf("failed to save connector encryption key: %w", err)
	}

	return key, nil
}

// Encrypt encrypts a plaintext string using AES-256-GCM and returns a base64-encoded ciphertext.
func Encrypt(plaintext string, key []byte) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	return EncryptionPrefix + base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a base64-encoded AES-256-GCM ciphertext.
func Decrypt(encrypted string, key []byte) (string, error) {
	if encrypted == "" {
		return "", nil
	}

	// Strip encryption prefix if present
	data := encrypted
	if len(data) > len(EncryptionPrefix) && data[:len(EncryptionPrefix)] == EncryptionPrefix {
		data = data[len(EncryptionPrefix):]
	}

	ciphertext, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		// If it can't be decoded, it might be a plain-text value (migration case)
		return encrypted, nil
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		// Decryption failed - might be plain text stored before encryption was added
		return encrypted, nil
	}

	return string(plaintext), nil
}

// MaskAPIKey masks an API key for display, showing only first 3 and last 3 characters.
func MaskAPIKey(apiKey string) string {
	if len(apiKey) <= 8 {
		return "***"
	}
	return apiKey[:3] + "..." + apiKey[len(apiKey)-3:]
}
