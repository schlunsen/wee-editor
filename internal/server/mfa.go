package server

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/skip2/go-qrcode"
	"golang.org/x/crypto/bcrypt"
)

// MFAManager handles all MFA operations
type MFAManager struct {
	encryptionKey []byte // 32-byte key for AES-256
}

// NewMFAManager creates a new MFA manager with encryption key
func NewMFAManager(encryptionKey []byte) (*MFAManager, error) {
	if len(encryptionKey) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes for AES-256")
	}
	return &MFAManager{
		encryptionKey: encryptionKey,
	}, nil
}

// GenerateTOTPSecret generates a new TOTP secret
func (m *MFAManager) GenerateTOTPSecret(username, issuer string) (*otp.Key, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: username,
		Period:      30, // Standard 30-second period per RFC 6238
		SecretSize:  32, // 256 bits for strong security
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate TOTP secret: %w", err)
	}
	return key, nil
}

// GenerateQRCode creates a QR code image from the TOTP key
func (m *MFAManager) GenerateQRCode(key *otp.Key) (string, error) {
	uri := key.String()
	qr, err := qrcode.New(uri, qrcode.High)
	if err != nil {
		return "", fmt.Errorf("failed to generate QR code: %w", err)
	}

	// Convert to PNG and encode as base64 data URL
	pngData, err := qr.PNG(200)
	if err != nil {
		return "", fmt.Errorf("failed to encode QR code: %w", err)
	}

	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngData)
	return dataURL, nil
}

// VerifyTOTPCode verifies a 6-digit TOTP code
// Uses a window of ±1 to account for time skew (RFC 6238)
func (m *MFAManager) VerifyTOTPCode(secret string, code string) (bool, error) {
	// totp.ValidateCustom expects the base32-encoded secret directly
	// No need to decode it first
	valid, err := totp.ValidateCustom(code, secret, time.Now(), totp.ValidateOpts{
		Period:    30,
		Digits:    6,
		Algorithm: otp.AlgorithmSHA1,
		Skew:      1, // Allow ±1 time step
	})
	if err != nil {
		return false, fmt.Errorf("failed to validate TOTP code: %w", err)
	}

	return valid, nil
}

// EncryptSecret encrypts a TOTP secret using AES-256-GCM
// Returns encrypted data and IV as base64 strings
func (m *MFAManager) EncryptSecret(secret string) (encrypted, iv string, err error) {
	block, err := aes.NewCipher(m.encryptionKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM cipher
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate random nonce (IV)
	nonce := make([]byte, gcm.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt the secret (nil dst creates fresh slice without prepending nonce)
	ciphertext := gcm.Seal(nil, nonce, []byte(secret), nil)

	// Return both encrypted data and IV as base64 strings
	return base64.StdEncoding.EncodeToString(ciphertext),
		base64.StdEncoding.EncodeToString(nonce),
		nil
}

// DecryptSecret decrypts an encrypted TOTP secret
func (m *MFAManager) DecryptSecret(encrypted, ivStr string) (string, error) {
	block, err := aes.NewCipher(m.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Decode base64 strings
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	nonce, err := base64.StdEncoding.DecodeString(ivStr)
	if err != nil {
		return "", fmt.Errorf("failed to decode IV: %w", err)
	}

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// GenerateBackupCodes generates N backup codes
// Each code is 8 characters long (alphanumeric) for readability
func (m *MFAManager) GenerateBackupCodes(count int) (codes []string, err error) {
	codes = make([]string, count)
	for i := 0; i < count; i++ {
		code := generateRandomCode(8)
		codes[i] = code
	}
	return codes, nil
}

// HashBackupCode hashes a backup code for storage
func (m *MFAManager) HashBackupCode(code string) (string, error) {
	// Use bcrypt with cost of 12
	hash, err := bcrypt.GenerateFromPassword([]byte(code), 12)
	if err != nil {
		return "", fmt.Errorf("failed to hash backup code: %w", err)
	}
	return string(hash), nil
}

// VerifyBackupCode verifies a backup code against its hash
func (m *MFAManager) VerifyBackupCode(code string, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(code))
	return err == nil
}

// BackupCodeCollection manages a collection of backup codes with usage tracking
type BackupCodeCollection struct {
	Codes []BackupCodeEntry `json:"codes"`
}

// BackupCodeEntry represents a single backup code with usage tracking
type BackupCodeEntry struct {
	Hash   string     `json:"hash"`   // Hashed code
	Used   bool       `json:"used"`   // Whether code has been used
	UsedAt *time.Time `json:"used_at,omitempty"`
}

// NewBackupCodeCollection creates a new backup code collection from plain codes
func (m *MFAManager) NewBackupCodeCollection(plainCodes []string) (*BackupCodeCollection, error) {
	collection := &BackupCodeCollection{
		Codes: make([]BackupCodeEntry, len(plainCodes)),
	}

	for i, code := range plainCodes {
		hash, err := m.HashBackupCode(code)
		if err != nil {
			return nil, err
		}
		collection.Codes[i] = BackupCodeEntry{
			Hash: hash,
			Used: false,
		}
	}

	return collection, nil
}

// SerializeBackupCodes serializes backup codes to JSON for storage
func (m *MFAManager) SerializeBackupCodes(collection *BackupCodeCollection) (string, error) {
	data, err := json.Marshal(collection)
	if err != nil {
		return "", fmt.Errorf("failed to marshal backup codes: %w", err)
	}
	return string(data), nil
}

// DeserializeBackupCodes deserializes backup codes from JSON
func (m *MFAManager) DeserializeBackupCodes(jsonStr string) (*BackupCodeCollection, error) {
	collection := &BackupCodeCollection{}
	err := json.Unmarshal([]byte(jsonStr), collection)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal backup codes: %w", err)
	}
	return collection, nil
}

// VerifyAndUseBackupCode verifies a backup code and marks it as used
// Returns (true, serialized codes) if valid, (false, serialized codes) if invalid/already used
func (m *MFAManager) VerifyAndUseBackupCode(collection *BackupCodeCollection, code string) (bool, string, error) {
	now := time.Now()

	for i := range collection.Codes {
		if collection.Codes[i].Used {
			continue // Skip already used codes
		}

		if m.VerifyBackupCode(code, collection.Codes[i].Hash) {
			// Mark as used
			collection.Codes[i].Used = true
			collection.Codes[i].UsedAt = &now

			// Serialize and return
			serialized, err := m.SerializeBackupCodes(collection)
			if err != nil {
				return false, "", err
			}
			return true, serialized, nil
		}
	}

	// Code not found or already used
	serialized, err := m.SerializeBackupCodes(collection)
	if err != nil {
		return false, "", err
	}
	return false, serialized, nil
}

// CountUnusedBackupCodes counts unused backup codes in a collection
func (m *MFAManager) CountUnusedBackupCodes(collection *BackupCodeCollection) int {
	count := 0
	for _, code := range collection.Codes {
		if !code.Used {
			count++
		}
	}
	return count
}

// GenerateTemporaryToken generates a temporary token for the MFA flow
func GenerateTemporaryToken() string {
	return uuid.New().String()
}

// CalculateTokenExpiration calculates when a token should expire
func CalculateTokenExpiration(ttlSeconds int) time.Time {
	return time.Now().Add(time.Duration(ttlSeconds) * time.Second)
}

// VerifyTokenNotExpired checks if a token has expired
func VerifyTokenNotExpired(expiresAt time.Time) bool {
	return time.Now().Before(expiresAt)
}

// helper function to generate random codes
func generateRandomCode(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		num := make([]byte, 1)
		_, err := rand.Read(num)
		if err != nil {
			// Fallback to deterministic generation if random fails
			num[0] = byte((i * 7) % 256)
		}
		b[i] = charset[int(num[0])%len(charset)]
	}
	return string(b)
}

// RateLimitChecker checks if too many verification attempts have been made
type RateLimitChecker struct {
	MaxAttempts      int           // Maximum attempts allowed
	LockoutDuration  time.Duration // Duration to lock out after max attempts
	AttemptWindow    time.Duration // Time window for counting attempts
}

// CheckRateLimit checks if an attempt should be rate limited
// Returns (allowed bool, attemptsRemaining int, lockoutUntil *time.Time)
func (r *RateLimitChecker) CheckRateLimit(attempts int, lastAttemptTime *time.Time) (bool, int, *time.Time) {
	now := time.Now()

	// If within attempt window and at max attempts, reject
	if lastAttemptTime != nil && now.Sub(*lastAttemptTime) < r.AttemptWindow && attempts >= r.MaxAttempts {
		lockoutUntil := lastAttemptTime.Add(r.LockoutDuration)
		remaining := 0
		return false, remaining, &lockoutUntil
	}

	// If outside attempt window, reset counter
	if lastAttemptTime != nil && now.Sub(*lastAttemptTime) >= r.AttemptWindow {
		attempts = 0
	}

	attemptsRemaining := r.MaxAttempts - attempts
	return true, attemptsRemaining, nil
}

// MFAConfigValidator validates MFA configuration
type MFAConfigValidator struct {
	MinBackupCodes int
	MaxBackupCodes int
}

// ValidateTOTPCode validates the format of a TOTP code
func (v *MFAConfigValidator) ValidateTOTPCode(code string) bool {
	// Must be exactly 6 digits
	if len(code) != 6 {
		return false
	}

	for _, ch := range code {
		if ch < '0' || ch > '9' {
			return false
		}
	}

	return true
}

// ValidateBackupCodeFormat validates the format of a backup code
func (v *MFAConfigValidator) ValidateBackupCodeFormat(code string) bool {
	// Backup codes should be uppercase alphanumeric, 8 characters
	code = strings.ToUpper(code)
	if len(code) != 8 {
		return false
	}

	for _, ch := range code {
		if (ch < 'A' || ch > 'Z') && (ch < '0' || ch > '9') {
			return false
		}
	}

	return true
}

// CalculateTOTPProgressPercent calculates how much time remains in the current TOTP window (0-100)
func CalculateTOTPProgressPercent() float64 {
	now := time.Now()
	secondsInPeriod := now.Unix() % 30
	return (float64(secondsInPeriod) / 30.0) * 100.0
}

// GetNextTOTPRefreshTime returns when the next TOTP code will be generated
func GetNextTOTPRefreshTime() time.Duration {
	now := time.Now()
	secondsInPeriod := now.Unix() % 30
	secondsUntilRefresh := 30 - secondsInPeriod
	return time.Duration(secondsUntilRefresh) * time.Second
}

// EncryptionHelper provides utility functions for encryption
type EncryptionHelper struct {
	key []byte
}

// NewEncryptionHelper creates a new encryption helper
func NewEncryptionHelper(key []byte) (*EncryptionHelper, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("key must be 32 bytes")
	}
	return &EncryptionHelper{key: key}, nil
}

// DeriveKeyFromPassword derives a 32-byte key from a password
// This can be used if you want encryption keys based on user passwords
func (e *EncryptionHelper) DeriveKeyFromPassword(password string) [32]byte {
	// For production, use scrypt or PBKDF2
	// This is a simple example
	var key [32]byte
	hash := make([]byte, 32)

	// Use bcrypt to derive key from password
	bcryptHash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MaxCost)
	copy(hash, bcryptHash[:32])
	copy(key[:], hash)

	return key
}

// MFAMetrics provides statistics about MFA usage
type MFAMetrics struct {
	TotalUsersWithMFA          int
	TotalVerifications         int
	SuccessfulVerifications    int
	FailedVerifications        int
	BackupCodeUsages           int
	LockedOutAccounts          int
	AverageSetupTime           time.Duration
	LastMetricsUpdateTime      time.Time
}

// CalculateSuccessRate returns the success rate as a percentage
func (m *MFAMetrics) CalculateSuccessRate() float64 {
	if m.TotalVerifications == 0 {
		return 0
	}
	return (float64(m.SuccessfulVerifications) / float64(m.TotalVerifications)) * 100
}

// CalculateBackupCodeUsageRate returns the rate of backup code usage vs TOTP
func (m *MFAMetrics) CalculateBackupCodeUsageRate() float64 {
	if m.SuccessfulVerifications == 0 {
		return 0
	}
	return (float64(m.BackupCodeUsages) / float64(m.SuccessfulVerifications)) * 100
}
