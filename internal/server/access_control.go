package server

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/schlunsen/wee-editor/internal/database"
)

// AccessValidator validates user access against configured rules
type AccessValidator struct {
	config *AccessControlConfig
	db     *database.Database
}

// NewAccessValidator creates a new access validator
func NewAccessValidator(config *AccessControlConfig, db *database.Database) *AccessValidator {
	return &AccessValidator{
		config: config,
		db:     db,
	}
}

// ValidateUser checks if a user is allowed based on username or email
// Returns nil if allowed, error if denied
func (av *AccessValidator) ValidateUser(username, email string) error {
	// If access control is disabled, allow all
	if av.config == nil || !av.config.Enabled {
		return nil
	}

	// Check if user is in allowed_users list (exact match on username or email)
	for _, allowedUser := range av.config.AllowedUsers {
		if strings.EqualFold(username, allowedUser) || strings.EqualFold(email, allowedUser) {
			return nil
		}
	}

	// Check if user's email domain is in allowed_domains list
	if email != "" {
		if av.matchesAnyDomain(email, av.config.AllowedDomains) {
			return nil
		}
	}

	// If we get here, user is not in allowed list
	reason := "User is not in the allowed users or domains list"
	denyMessage := av.config.DenyMessage
	if denyMessage == "" {
		denyMessage = reason
	}

	return errors.New(denyMessage)
}

// ValidateOAuthEmail checks if an OAuth user's email is allowed
// Returns nil if allowed, error if denied
func (av *AccessValidator) ValidateOAuthEmail(email string) error {
	// If access control is disabled, allow all
	if av.config == nil || !av.config.Enabled {
		return nil
	}

	// Check if email is in allowed_users list (exact match)
	for _, allowedUser := range av.config.AllowedUsers {
		if strings.EqualFold(email, allowedUser) {
			return nil
		}
	}

	// Check if email domain is in allowed_domains list
	if av.matchesAnyDomain(email, av.config.AllowedDomains) {
		return nil
	}

	// If we get here, user is not in allowed list
	reason := "Your email domain or address is not authorized for this service"
	denyMessage := av.config.DenyMessage
	if denyMessage == "" {
		denyMessage = reason
	}

	return errors.New(denyMessage)
}

// matchesDomainPattern checks if an email matches a domain pattern
// Pattern formats:
//   - "*@domain.com" - matches any user at domain.com (case-insensitive)
//   - "user@domain.com" - exact email match (case-insensitive)
func (av *AccessValidator) matchesDomainPattern(email, pattern string) bool {
	// Normalize to lowercase for case-insensitive comparison
	email = strings.ToLower(email)
	pattern = strings.ToLower(pattern)

	// Exact email match
	if !strings.Contains(pattern, "*") {
		return email == pattern
	}

	// Domain pattern match: "*@domain.com"
	if strings.HasPrefix(pattern, "*@") {
		domainPattern := strings.TrimPrefix(pattern, "*@")
		// Extract domain from email (everything after @)
		parts := strings.Split(email, "@")
		if len(parts) != 2 {
			return false
		}
		userDomain := parts[1]

		// Allow subdomain matching: *@company.com matches user@dept.company.com
		return userDomain == domainPattern || strings.HasSuffix(userDomain, "."+domainPattern)
	}

	return false
}

// matchesAnyDomain checks if an email matches any pattern in the domain list
func (av *AccessValidator) matchesAnyDomain(email string, patterns []string) bool {
	for _, pattern := range patterns {
		if av.matchesDomainPattern(email, pattern) {
			return true
		}
	}
	return false
}

// ShouldAllowFirstUser returns true if the first user should bypass access control
func (av *AccessValidator) ShouldAllowFirstUser() bool {
	if av.config == nil {
		return true
	}
	return av.config.AllowFirstUser
}

// GetDenyMessage returns the configured deny message or a default one
func (av *AccessValidator) GetDenyMessage() string {
	if av.config == nil || av.config.DenyMessage == "" {
		return "Access denied: Your account is not authorized to access this service"
	}
	return av.config.DenyMessage
}

// IsAdmin checks if a user should be created as an admin
// Returns true if the username/email matches the admin_users list in auth config
func (av *AccessValidator) IsAdmin(username, email string, adminUsers []string) bool {
	for _, admin := range adminUsers {
		if strings.EqualFold(username, admin) || strings.EqualFold(email, admin) {
			return true
		}
	}
	return false
}

// LogAccessDenied logs an access denial event
func (av *AccessValidator) LogAccessDenied(username, email, action, reason, ipAddress, userAgent string) error {
	if av.db == nil {
		return nil // Skip logging if DB not available
	}

	repo := database.NewRepository(av.db)
	log := &database.AuthAuditLog{
		Username:  username,
		Email:     email,
		Action:    action,
		Reason:    reason,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		CreatedAt: time.Now(),
	}

	return repo.LogAuthAudit(log)
}

// LogAuthSuccess logs a successful authentication event
func (av *AccessValidator) LogAuthSuccess(username, email, action, ipAddress, userAgent string) error {
	if av.db == nil {
		return nil // Skip logging if DB not available
	}

	repo := database.NewRepository(av.db)
	log := &database.AuthAuditLog{
		Username:  username,
		Email:     email,
		Action:    action,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		CreatedAt: time.Now(),
	}

	return repo.LogAuthAudit(log)
}

// ValidateFirstUser checks if this is the first user and allows or denies based on config
// Returns (allowed, isFirstUser, error)
func (av *AccessValidator) ValidateFirstUser() (bool, error) {
	// Check if any users exist yet
	if av.db == nil {
		return false, errors.New("database not available")
	}

	repo := database.NewRepository(av.db)
	users, err := repo.ListUsers()
	if err != nil {
		return false, fmt.Errorf("failed to check existing users: %w", err)
	}

	isFirstUser := len(users) == 0

	// If no users exist yet
	if isFirstUser {
		// Check if we should allow the first user
		if av.config == nil || !av.config.Enabled || av.config.AllowFirstUser {
			return true, nil
		}
		// Access control enabled and AllowFirstUser is false
		return false, errors.New("access control is enabled but no users have been created yet. Cannot create first user")
	}

	return false, nil
}
