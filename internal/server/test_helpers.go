package server

import (
	"crypto/rand"
	"testing"
	"time"

	"github.com/schlunsen/wee-editor/internal/database"
	"golang.org/x/crypto/bcrypt"
)

// NewInMemoryUserStore creates an in-memory user store for testing
func NewInMemoryUserStore() *UserStore {
	return &UserStore{
		users:    make(map[string]*User),
		sessions: make(map[string]*Session),
		db:       nil,
	}
}

// NewTestDB is imported from database package for testing
// This is just a convenience wrapper
func NewTestDB(t *testing.T) *database.Database {
	return database.NewTestDB(t)
}

// CreateTestUser creates a user in both the UserStore and database
// This is needed for MFA tests since MFA tables have foreign key constraints
func CreateTestUser(t *testing.T, db *database.Database, username, password string, isAdmin bool) *User {
	t.Helper()

	// Hash the password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	// Insert directly into database
	_, err = db.GetDB().Exec(`
		INSERT INTO users (username, password_hash, is_admin, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, username, string(passwordHash), isAdmin, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("Failed to create test user in database: %v", err)
	}

	return &User{
		Username:     username,
		PasswordHash: string(passwordHash),
		IsAdmin:      isAdmin,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

// CreateTestSession creates a test session for a user
func CreateTestSession(t *testing.T, server *Server, username string) string {
	t.Helper()

	// Generate session token
	token := GenerateTemporaryToken()
	expiresAt := CalculateTokenExpiration(int(SessionDuration.Seconds()))

	// Insert session into database using correct table name (user_sessions) and column name (token)
	_, err := server.db.GetDB().Exec(`
		INSERT INTO user_sessions (token, username, expires_at, created_at)
		VALUES (?, ?, ?, ?)
	`, token, username, expiresAt, time.Now())
	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	return token
}

// CreateTestUserWithMFA creates a user with MFA enabled
func CreateTestUserWithMFA(t *testing.T, db *database.Database, username, password string, isAdmin bool, mfaSecret string) *User {
	t.Helper()

	// Create regular user first
	user := CreateTestUser(t, db, username, password, isAdmin)

	// Create MFA manager with random encryption key for testing
	encryptionKey := make([]byte, 32)
	_, err := rand.Read(encryptionKey)
	if err != nil {
		t.Fatalf("Failed to generate encryption key: %v", err)
	}

	mfaManager, err := NewMFAManager(encryptionKey)
	if err != nil {
		t.Fatalf("Failed to create MFA manager: %v", err)
	}

	// Encrypt the TOTP secret
	encryptedSecret, iv, err := mfaManager.EncryptSecret(mfaSecret)
	if err != nil {
		t.Fatalf("Failed to encrypt TOTP secret: %v", err)
	}

	// Insert MFA configuration with encrypted secret and IV
	_, err = db.GetDB().Exec(`
		INSERT INTO user_mfa_config (username, is_enabled, totp_secret, totp_secret_iv, mfa_enabled_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, username, true, encryptedSecret, iv, time.Now(), time.Now())
	if err != nil {
		t.Fatalf("Failed to create MFA config for test user: %v", err)
	}

	return user
}
