package server

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"strings"
	"time"

	"github.com/pquerna/otp"
)

// MockTOTPService provides a mock implementation of TOTP functionality for testing
type MockTOTPService struct {
	// For deterministic testing
	CurrentTime time.Time
	GenerateCodeFunc func(secret string, t time.Time) (string, error)
	ValidateCodeFunc func(code, secret string, t time.Time) (bool, error)
	GenerateSecretFunc func(issuer, accountName string) (*otp.Key, error)
	QRCodeFunc func(key *otp.Key) (string, error)

	// For tracking calls
	Calls struct {
		GenerateSecret   []MockSecretCall
		GenerateCode     []MockCodeCall
		ValidateCode     []MockValidateCall
		GenerateQRCode   []MockQRCall
	}
}

// MockSecretCall tracks calls to GenerateSecret
type MockSecretCall struct {
	Issuer      string
	AccountName string
	Result      *otp.Key
	Error       error
}

// MockCodeCall tracks calls to GenerateCode
type MockCodeCall struct {
	Secret string
	Time   time.Time
	Code   string
	Error  error
}

// MockValidateCall tracks calls to ValidateCode
type MockValidateCall struct {
	Secret   string
	Code     string
	Time     time.Time
	Valid    bool
	Error    error
}

// MockQRCall tracks calls to GenerateQRCode
type MockQRCall struct {
	Key    *otp.Key
	Result string
	Error  error
}

// NewMockTOTPService creates a new mock TOTP service with default behaviors
func NewMockTOTPService() *MockTOTPService {
	mock := &MockTOTPService{
		CurrentTime: time.Now(),
	}

	// Set up default deterministic behaviors
	mock.GenerateCodeFunc = mock.defaultGenerateCode
	mock.ValidateCodeFunc = mock.defaultValidateCode
	mock.GenerateSecretFunc = mock.defaultGenerateSecret
	mock.QRCodeFunc = mock.defaultGenerateQRCode

	return mock
}

// defaultGenerateSecret creates a deterministic TOTP secret for testing
func (m *MockTOTPService) defaultGenerateSecret(issuer, accountName string) (*otp.Key, error) {
	// Generate deterministic secret based on issuer and account name
	seed := fmt.Sprintf("%s:%s:%d", issuer, accountName, m.CurrentTime.Unix())
	secret := m.generateDeterministicSecret(seed)

	// Create a mock otpauth URL that will generate the desired key
	url := fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=SHA1&digits=6&period=30",
		issuer, accountName, secret, issuer)

	// Use the library's URL parser to create a valid Key
	key, err := otp.NewKeyFromURL(url)
	if err != nil {
		// Fallback: create a minimal working key
		return &otp.Key{}, err
	}

	return key, nil
}

// defaultGenerateCode creates a deterministic 6-digit TOTP code
func (m *MockTOTPService) defaultGenerateCode(secret string, t time.Time) (string, error) {
	// Generate deterministic code based on secret and time
	// Use a simple hash-like function for consistency
	seed := fmt.Sprintf("%s:%d", secret, t.Unix()/30) // 30-second window
	code := m.deterministicCode(seed)
	return code, nil
}

// defaultValidateCode validates a TOTP code using the mock generation logic
func (m *MockTOTPService) defaultValidateCode(code, secret string, t time.Time) (bool, error) {
	// Generate expected code for current time
	expectedCode, err := m.defaultGenerateCode(secret, t)
	if err != nil {
		return false, err
	}

	// Also check codes within ±1 time window for skew handling
	pastTime := t.Add(-30 * time.Second)
	pastCode, err := m.defaultGenerateCode(secret, pastTime)
	if err != nil {
		return false, err
	}

	futureTime := t.Add(30 * time.Second)
	futureCode, err := m.defaultGenerateCode(secret, futureTime)
	if err != nil {
		return false, err
	}

	return code == expectedCode || code == pastCode || code == futureCode, nil
}

// defaultGenerateQRCode creates a mock QR code data URL
func (m *MockTOTPService) defaultGenerateQRCode(key *otp.Key) (string, error) {
	// Return a deterministic mock QR code
	qrData := fmt.Sprintf("data:image/png;base64,MOCK-QR-CODE-FOR-%s-%s",
		key.Issuer(), key.AccountName())
	return qrData, nil
}

// generateDeterministicSecret creates a deterministic base32 secret
func (m *MockTOTPService) generateDeterministicSecret(seed string) string {
	// Simple deterministic function for generating base32 secrets
	// In real implementation, this would use proper cryptographic randomness
	hash := strings.ToUpper(seed)

	// Pad and convert to valid base32 characters
	validChars := "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"
	secret := ""
	for _, char := range hash {
		secret += string(validChars[int(char)%len(validChars)])
		if len(secret) >= 32 {
			break
		}
	}

	// Pad to 32 characters if needed
	for len(secret) < 32 {
		secret += "A"
	}

	return secret
}

// deterministicCode generates a deterministic 6-digit code from a seed
func (m *MockTOTPService) deterministicCode(seed string) string {
	// Simple hash function to generate consistent 6-digit codes
	var sum int32
	for _, char := range seed {
		sum = sum*31 + int32(char)
	}
	if sum < 0 {
		sum = -sum
	}

	// Generate 6 digits
	code := fmt.Sprintf("%06d", sum%1000000)
	return code
}

// MockMFAManager wraps MFAManager with mock TOTP service
type MockMFAManager struct {
	*MFAManager
	TOTPService *MockTOTPService
}

// NewMockMFAManager creates a mock MFA manager for testing
func NewMockMFAManager(encryptionKey []byte) (*MockMFAManager, error) {
	realManager, err := NewMFAManager(encryptionKey)
	if err != nil {
		return nil, err
	}

	return &MockMFAManager{
		MFAManager:  realManager,
		TOTPService: NewMockTOTPService(),
	}, nil
}

// GenerateTOTPSecret overrides the real implementation with mock
func (m *MockMFAManager) GenerateTOTPSecret(username, issuer string) (*otp.Key, error) {
	call := MockSecretCall{
		Issuer:      issuer,
		AccountName: username,
	}

	key, err := m.TOTPService.GenerateSecretFunc(issuer, username)
	call.Result = key
	call.Error = err

	m.TOTPService.Calls.GenerateSecret = append(m.TOTPService.Calls.GenerateSecret, call)
	return key, err
}

// GenerateQRCode overrides the real implementation with mock
func (m *MockMFAManager) GenerateQRCode(key *otp.Key) (string, error) {
	call := MockQRCall{
		Key: key,
	}

	result, err := m.TOTPService.QRCodeFunc(key)
	call.Result = result
	call.Error = err

	m.TOTPService.Calls.GenerateQRCode = append(m.TOTPService.Calls.GenerateQRCode, call)
	return result, err
}

// VerifyTOTPCode overrides the real implementation with mock
func (m *MockMFAManager) VerifyTOTPCode(secret string, code string) (bool, error) {
	call := MockValidateCall{
		Secret: secret,
		Code:   code,
		Time:   m.TOTPService.CurrentTime,
	}

	valid, err := m.TOTPService.ValidateCodeFunc(code, secret, m.TOTPService.CurrentTime)
	call.Valid = valid
	call.Error = err

	m.TOTPService.Calls.ValidateCode = append(m.TOTPService.Calls.ValidateCode, call)
	return valid, err
}

// MockGenerateCode provides access to mock code generation for tests
func (m *MockMFAManager) MockGenerateCode(secret string, t time.Time) (string, error) {
	call := MockCodeCall{
		Secret: secret,
		Time:   t,
	}

	code, err := m.TOTPService.GenerateCodeFunc(secret, t)
	call.Code = code
	call.Error = err

	m.TOTPService.Calls.GenerateCode = append(m.TOTPService.Calls.GenerateCode, call)
	return code, err
}

// SetCurrentTime sets the mock's current time for deterministic testing
func (m *MockMFAManager) SetCurrentTime(t time.Time) {
	m.TOTPService.CurrentTime = t
}

// ResetCalls clears all call tracking
func (m *MockMFAManager) ResetCalls() {
	m.TOTPService.Calls = struct {
		GenerateSecret []MockSecretCall
		GenerateCode   []MockCodeCall
		ValidateCode   []MockValidateCall
		GenerateQRCode []MockQRCall
	}{}
}

// GetCalls returns a copy of all tracked calls
func (m *MockMFAManager) GetCalls() struct {
	GenerateSecret []MockSecretCall
	GenerateCode   []MockCodeCall
	ValidateCode   []MockValidateCall
	GenerateQRCode []MockQRCall
} {
	return m.TOTPService.Calls
}

// Utility function to create a real-looking TOTP secret for tests
func CreateTestTOTPSecret(username, issuer string) string {
	// Create 32-byte base32 secret
	secret := make([]byte, 32)
	rand.Read(secret) // Fallback to real randomness for initial seed

	// Convert to base32
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret)
}

// Utility function to create a real TOTP key for integration tests
func CreateTestTOTPKey(username, issuer string) *otp.Key {
	secret := CreateTestTOTPSecret(username, issuer)

	// Create a valid otpauth URL
	url := fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=SHA1&digits=6&period=30",
		issuer, username, secret, issuer)

	// Use the library's URL parser to create a valid Key
	key, err := otp.NewKeyFromURL(url)
	if err != nil {
		// Return an empty key as fallback
		return &otp.Key{}
	}

	return key
}

// Convenience functions for common test scenarios

// SetupSuccessfulTOTP configures the mock for successful TOTP operations
func (m *MockMFAManager) SetupSuccessfulTOTP(username, issuer string) {
	testKey := CreateTestTOTPKey(username, issuer)

	m.TOTPService.GenerateSecretFunc = func(issuer, accountName string) (*otp.Key, error) {
		return testKey, nil
	}

	m.TOTPService.GenerateCodeFunc = func(secret string, t time.Time) (string, error) {
		// Return a deterministic valid code for testing
		return "123456", nil
	}

	m.TOTPService.ValidateCodeFunc = func(code, secret string, t time.Time) (bool, error) {
		return code == "123456", nil
	}

	m.TOTPService.QRCodeFunc = func(key *otp.Key) (string, error) {
		return "data:image/png;base64,TEST-QR-CODE", nil
	}
}

// SetupFailingTOTP configures the mock for failing TOTP operations
func (m *MockMFAManager) SetupFailingTOTP() {
	m.TOTPService.GenerateSecretFunc = func(issuer, accountName string) (*otp.Key, error) {
		return nil, fmt.Errorf("mock error: failed to generate secret")
	}

	m.TOTPService.GenerateCodeFunc = func(secret string, t time.Time) (string, error) {
		return "", fmt.Errorf("mock error: failed to generate code")
	}

	m.TOTPService.ValidateCodeFunc = func(code, secret string, t time.Time) (bool, error) {
		return false, fmt.Errorf("mock error: validation failed")
	}

	m.TOTPService.QRCodeFunc = func(key *otp.Key) (string, error) {
		return "", fmt.Errorf("mock error: failed to generate QR code")
	}
}

// SetupTimeSkewScenario configures the mock to test time skew handling
func (m *MockMFAManager) SetupTimeSkewScenario(validCode string) {
	m.TOTPService.ValidateCodeFunc = func(code, secret string, t time.Time) (bool, error) {
		// Always return true for the specified valid code regardless of time
		if code == validCode {
			return true, nil
		}
		return false, nil
	}
}