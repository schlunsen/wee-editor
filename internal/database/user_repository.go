// Package database provides data access methods for user management.
// This repository handles all CRUD operations related to user accounts,
// including password updates, avatar management, and OAuth lookups.
package database

import (
	"database/sql"
	"time"
)

// UserRepository provides data access methods for user operations
type UserRepository struct {
	db *Database
}

// NewUserRepository creates a new user repository instance
func NewUserRepository(db *Database) *UserRepository {
	return &UserRepository{db: db}
}

// ============================================
// User CRUD Operations
// ============================================

// CreateUser creates a new user in the database
func (r *UserRepository) CreateUser(user *DBUser) error {
	query := `
		INSERT INTO users (username, password_hash, email, auth_method, oauth_provider_id, oauth_provider, is_admin, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.db.Exec(query, user.Username, user.PasswordHash, user.Email, user.AuthMethod, user.OAuthProviderID, user.OAuthProvider, user.IsAdmin, user.CreatedAt, user.UpdatedAt)
	return err
}

// GetUser retrieves a user by username
func (r *UserRepository) GetUser(username string) (*DBUser, error) {
	query := `SELECT id, username, password_hash, email, auth_method, oauth_provider_id, oauth_provider, is_admin, avatar_id, created_at, updated_at FROM users WHERE username = ?`

	user := &DBUser{}
	err := r.db.db.QueryRow(query, username).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.Email,
		&user.AuthMethod,
		&user.OAuthProviderID,
		&user.OAuthProvider,
		&user.IsAdmin,
		&user.AvatarID,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return user, nil
}

// GetUserByIDOrUsername retrieves a user by either UUID or username
// This is used for resolving user_id from messages (which can be either format)
func (r *UserRepository) GetUserByIDOrUsername(userID string) (*DBUser, error) {
	query := `
		SELECT id, username, password_hash, email, auth_method, oauth_provider_id, oauth_provider, is_admin, avatar_id, created_at, updated_at
		FROM users
		WHERE id = ? OR username = ?
		LIMIT 1
	`

	user := &DBUser{}
	err := r.db.db.QueryRow(query, userID, userID).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.Email,
		&user.AuthMethod,
		&user.OAuthProviderID,
		&user.OAuthProvider,
		&user.IsAdmin,
		&user.AvatarID,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return user, nil
}

// ListUsers returns all users
func (r *UserRepository) ListUsers() ([]*DBUser, error) {
	query := `SELECT username, password_hash, email, auth_method, oauth_provider_id, oauth_provider, is_admin, avatar_id, created_at, updated_at FROM users ORDER BY username`

	rows, err := r.db.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*DBUser
	for rows.Next() {
		user := &DBUser{}
		err := rows.Scan(&user.Username, &user.PasswordHash, &user.Email, &user.AuthMethod, &user.OAuthProviderID, &user.OAuthProvider, &user.IsAdmin, &user.AvatarID, &user.CreatedAt, &user.UpdatedAt)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, rows.Err()
}

// UpdateUserPassword updates a user's password hash
func (r *UserRepository) UpdateUserPassword(username, newPasswordHash string) error {
	query := `UPDATE users SET password_hash = ?, updated_at = ? WHERE username = ?`
	_, err := r.db.db.Exec(query, newPasswordHash, time.Now(), username)
	return err
}

// UpdateUserAvatar updates a user's selected avatar
func (r *UserRepository) UpdateUserAvatar(username string, avatarID *int64) error {
	query := `UPDATE users SET avatar_id = ?, updated_at = ? WHERE username = ?`
	_, err := r.db.db.Exec(query, avatarID, time.Now(), username)
	return err
}

// SetUserAdmin updates the admin status of a user
func (r *UserRepository) SetUserAdmin(username string, isAdmin bool) error {
	query := `UPDATE users SET is_admin = ?, updated_at = ? WHERE username = ?`
	_, err := r.db.db.Exec(query, isAdmin, time.Now(), username)
	return err
}

// DeleteUser deletes a user from the database
func (r *UserRepository) DeleteUser(username string) error {
	query := `DELETE FROM users WHERE username = ?`
	_, err := r.db.db.Exec(query, username)
	return err
}

// ============================================
// User Lookup Operations
// ============================================

// FindUserByOAuthID finds a user by OAuth provider and provider ID
func (r *UserRepository) FindUserByOAuthID(provider, providerID string) (*DBUser, error) {
	query := `SELECT username, password_hash, COALESCE(email, ''), auth_method, COALESCE(oauth_provider_id, ''), COALESCE(oauth_provider, ''), is_admin, avatar_id, created_at, updated_at FROM users WHERE oauth_provider = ? AND oauth_provider_id = ?`

	user := &DBUser{}
	err := r.db.db.QueryRow(query, provider, providerID).Scan(
		&user.Username,
		&user.PasswordHash,
		&user.Email,
		&user.AuthMethod,
		&user.OAuthProviderID,
		&user.OAuthProvider,
		&user.IsAdmin,
		&user.AvatarID,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return user, nil
}

// FindUserByEmail finds a user by email address
func (r *UserRepository) FindUserByEmail(email string) (*DBUser, error) {
	query := `SELECT username, password_hash, COALESCE(email, ''), auth_method, COALESCE(oauth_provider_id, ''), COALESCE(oauth_provider, ''), is_admin, avatar_id, created_at, updated_at FROM users WHERE email = ?`

	user := &DBUser{}
	err := r.db.db.QueryRow(query, email).Scan(
		&user.Username,
		&user.PasswordHash,
		&user.Email,
		&user.AuthMethod,
		&user.OAuthProviderID,
		&user.OAuthProvider,
		&user.IsAdmin,
		&user.AvatarID,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return user, nil
}
