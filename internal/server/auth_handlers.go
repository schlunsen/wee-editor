package server

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/schlunsen/wee-editor/internal/database"
)

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents a login response
// SECURITY: Token is deliberately excluded - session tokens are delivered via HttpOnly cookies only
type LoginResponse struct {
	Username         string    `json:"username"`
	ExpiresAt        time.Time `json:"expires_at,omitempty"`
	MFARequired      bool      `json:"mfa_required,omitempty"`
	TemporaryToken   string    `json:"temporary_token,omitempty"`
	TemporaryExpires time.Time `json:"temporary_expires,omitempty"`
}

// ChangePasswordRequest represents a password change request
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// CreateUserRequest represents a user creation request
type CreateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	IsAdmin  bool   `json:"is_admin"`
}

// SetupRequest represents an initial setup request
type SetupRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// SetupResponse represents an initial setup response
// SECURITY: Token is deliberately excluded - session tokens are delivered via HttpOnly cookies only
type SetupResponse struct {
	Username  string    `json:"username"`
	ExpiresAt time.Time `json:"expires_at"`
	Message   string    `json:"message"`
}

// SessionAuthMiddleware creates a middleware for session authentication
type SessionAuthMiddleware struct {
	userStore    *UserStore
	enabled      bool
	requireLogin bool
}

// NewSessionAuthMiddleware creates a new session authentication middleware
func NewSessionAuthMiddleware(userStore *UserStore, enabled bool, requireLogin bool) *SessionAuthMiddleware {
	return &SessionAuthMiddleware{
		userStore:    userStore,
		enabled:      enabled,
		requireLogin: requireLogin,
	}
}

// handleUnauthorized handles unauthorized requests by either redirecting (for browser requests)
// or returning 401 JSON (for API requests)
func (sam *SessionAuthMiddleware) handleUnauthorized(c *fiber.Ctx) error {
	// Check if this is a browser request
	accept := c.Get("Accept", "")
	isBrowserRequest := strings.Contains(accept, "text/html")

	if isBrowserRequest {
		// Redirect to home page
		return c.Redirect("/", fiber.StatusSeeOther) // 303 See Other
	}

	// For API requests, return 401 JSON
	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
		"error": "Authentication required",
	})
}

// Handler returns the Fiber middleware handler
func (sam *SessionAuthMiddleware) Handler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		path := c.Path()

		// Enhanced debug logging for ALL requests
		if strings.Contains(path, "oauth") || strings.Contains(path, "callback") {
			fmt.Printf("\n===== SESSION AUTH MIDDLEWARE =====\n")
			fmt.Printf("Path: %s\n", path)
			fmt.Printf("Method: %s\n", c.Method())
			fmt.Printf("Enabled: %v\n", sam.enabled)
			fmt.Printf("RequireLogin: %v\n", sam.requireLogin)
		}

		// Skip if authentication is disabled
		if !sam.enabled {
			if strings.Contains(path, "oauth") {
				fmt.Printf("✅ Auth disabled - allowing\n")
				fmt.Printf("===================================\n\n")
			}
			return c.Next()
		}

		// SECURITY: Setup endpoint is only allowed when no users exist
		if path == "/api/auth/setup" {
			if sam.userStore.HasUsers() {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "Setup already completed. Users already exist.",
				})
			}
			return c.Next()
		}

		// Always allow other auth endpoints
		if path == "/api/auth/login" || path == "/api/auth/status" || path == "/api/auth/logout" ||
		   path == "/api/auth/mfa/verify" || path == "/api/auth/oauth/authorize" ||
		   path == "/api/auth/oauth/callback" || path == "/api/auth/oauth/callback/google" {
			if strings.Contains(path, "oauth") || strings.Contains(path, "callback") {
				fmt.Printf("✅ Path in exemption list - ALLOWING\n")
				fmt.Printf("===================================\n\n")
			}
			return c.Next()
		}

		if strings.Contains(path, "oauth") || strings.Contains(path, "callback") {
			fmt.Printf("❌ Path NOT in exemption list - checking auth\n")
		}

		// Always allow health, version, and service worker endpoints
		if path == "/api/health" || path == "/api/version" ||
		   path == "/sw.js" || strings.HasPrefix(path, "/workbox-") {
			return c.Next()
		}

		method := c.Method()

		// If require_login is false, allow most GET/OPTIONS requests without auth
		// This lets users browse the dashboard but protects sensitive endpoints
		if !sam.requireLogin && (method == "GET" || method == "OPTIONS") {
			// Always allow static assets
			if path == "/" || strings.HasPrefix(path, "/assets/") ||
				strings.HasPrefix(path, "/_nuxt/") || path == "/favicon.ico" || path == "/favicon.svg" {
				return c.Next()
			}
			// SECURITY: Sensitive API endpoints that ALWAYS require auth,
			// even when require_login is false.
			// All /api/auth/ sub-endpoints (except already-exempted login/status/logout)
			// need session auth to set c.Locals("user") for admin/MFA checks.
			isSensitive := strings.HasPrefix(path, "/api/auth/") ||
				path == "/api/config/api-key"
			if !isSensitive {
				// Allow non-sensitive API GET requests without auth
				return c.Next()
			}
			// Fall through to auth check for sensitive endpoints
		}

		// If require_login is true, only allow static assets without auth
		if sam.requireLogin && (method == "GET" || method == "OPTIONS") {
			// Allow static assets only
			if path == "/" || strings.HasPrefix(path, "/assets/") ||
				strings.HasPrefix(path, "/_nuxt/") || path == "/favicon.ico" || path == "/favicon.svg" {
				return c.Next()
			}
			// All other GET requests require auth when require_login is true
		}

		// Get session token from cookie or Authorization header
		token := c.Cookies("session_token")
		authHeader := c.Get("Authorization")
		if token == "" && authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token = authHeader[7:]
		}

		if token == "" {
			if strings.Contains(path, "oauth") || strings.Contains(path, "callback") {
				fmt.Printf("❌ No token found - returning 401\n")
				fmt.Printf("===================================\n\n")
			}
			return sam.handleUnauthorized(c)
		}

		// Try to validate as a session token first
		user, err := sam.userStore.ValidateSession(token)
		if err == nil {
			// SECURITY: Check if MFA verification is required but not yet completed
			// This prevents users from bypassing MFA by using a pre-MFA session token
			session := sam.userStore.GetSession(token)
			if session != nil && session.MFARequired && !session.MFAVerified {
				// Only allow MFA verification endpoint and auth status while MFA is pending
				if path != "/api/auth/mfa/verify" && path != "/api/auth/status" {
					return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
						"error":        "MFA verification required",
						"mfa_required": true,
					})
				}
			}

			// Valid session token with MFA completed (or not required)
			// Store user in context
			c.Locals("user", user)
			c.Locals("username", user.Username)
			return c.Next()
		}

		// If session validation failed, check if it's an API key
		// API keys can be used for direct API access without user sessions
		isValidAPIKey := sam.userStore.ValidateAPIKey(token)
		if isValidAPIKey {
			// Valid API key - allow access without storing user
			c.Locals("authenticated_via_api_key", true)
			return c.Next()
		}

		// Clear invalid cookie
		c.ClearCookie("session_token")
		return sam.handleUnauthorized(c)
	}
}

// handleLogin handles user login
func (s *Server) handleLogin(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// SECURITY: Check MFA status BEFORE creating any session tokens
	// This prevents session token leakage when MFA is enabled
	repo := database.NewRepository(s.db)
	mfaConfig, err := repo.GetMFAConfig(req.Username)
	if err != nil {
		// Log error but continue (MFA might not be set up)
		fmt.Printf("Error checking MFA config during login: %v\n", err)
	}

	// If MFA is enabled, verify password WITHOUT creating session token
	if mfaConfig != nil && mfaConfig.IsEnabled {
		// Verify credentials without creating a session
		user, verifyErr := s.userStore.VerifyPassword(req.Username, req.Password)
		if verifyErr != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid username or password",
			})
		}

		// Validate user access via access control
		accessValidator := NewAccessValidator(s.config.Auth.AccessControl, s.db)
		if err := accessValidator.ValidateUser(user.Username, user.Email); err != nil {
			// Log access denial
			_ = accessValidator.LogAccessDenied(user.Username, user.Email, "password_login_denied", err.Error(), c.IP(), c.Get("User-Agent"))
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": accessValidator.GetDenyMessage(),
			})
		}

		// Generate temporary token for MFA verification
		tempToken := GenerateTemporaryToken()
		expiresAt := CalculateTokenExpiration(300) // 5 minutes

		temporaryToken := &database.MFATemporaryToken{
			Token:       tempToken,
			Username:    req.Username,
			TokenType:   "login",
			Verified:    false,
			Attempts:    0,
			MaxAttempts: 5,
			CreatedAt:   time.Now(),
			ExpiresAt:   expiresAt,
			IPAddress:   c.IP(),
			UserAgent:   c.Get("User-Agent"),
		}

		if err := repo.SaveMFATemporaryToken(temporaryToken); err != nil {
			fmt.Printf("Error saving temporary token: %v\n", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to initiate MFA verification",
			})
		}

		// Log audit event
		auditLog := &database.MFAAuditLog{
			Username:  req.Username,
			Action:    "login_mfa_required",
			Method:    "totp",
			Success:   true,
			IPAddress: c.IP(),
			UserAgent: c.Get("User-Agent"),
		}
		_ = repo.LogMFAAudit(auditLog)

		return c.JSON(LoginResponse{
			Username:         req.Username,
			MFARequired:      true,
			TemporaryToken:   tempToken,
			TemporaryExpires: expiresAt,
		})
	}

	// No MFA required, proceed with normal login
	// Verify credentials first
	user, authErr := s.userStore.VerifyPassword(req.Username, req.Password)
	if authErr != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid username or password",
		})
	}

	// Validate user access via access control
	accessValidator := NewAccessValidator(s.config.Auth.AccessControl, s.db)
	if err := accessValidator.ValidateUser(user.Username, user.Email); err != nil {
		// Log access denial
		_ = accessValidator.LogAccessDenied(user.Username, user.Email, "password_login_denied", err.Error(), c.IP(), c.Get("User-Agent"))
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": accessValidator.GetDenyMessage(),
		})
	}

	// Now it's safe to create a session token (both password and access control validated)
	token, sessionErr := s.userStore.generateSessionTokenForUser(user.Username)
	if sessionErr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create session",
		})
	}

	// Log successful authentication
	_ = accessValidator.LogAuthSuccess(user.Username, user.Email, "password_login_success", c.IP(), c.Get("User-Agent"))

	// Set session cookie
	c.Cookie(&fiber.Cookie{
		Name:     "session_token",
		Value:    token,
		Path:     "/",
		HTTPOnly: true,
		Secure:   s.config.TLS.Enabled || c.Get("X-Forwarded-Proto") == "https",
		SameSite: "Lax",
		Expires:  time.Now().Add(SessionDuration),
	})

	return c.JSON(LoginResponse{
		Username:  req.Username,
		ExpiresAt: time.Now().Add(SessionDuration),
	})
}

// handleLogout handles user logout
func (s *Server) handleLogout(c *fiber.Ctx) error {
	// Get session token
	token := c.Cookies("session_token")
	if token == "" {
		authHeader := c.Get("Authorization")
		if authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token = authHeader[7:]
		}
	}

	if token != "" {
		// Revoke session
		s.userStore.RevokeSession(token)
	}

	// Clear cookie
	c.ClearCookie("session_token")

	// SECURITY: Prevent browser/proxy caching of logout response (AUTH-VULN-06)
	c.Set("Cache-Control", "no-store, no-cache, must-revalidate")
	c.Set("Pragma", "no-cache")

	return c.JSON(fiber.Map{
		"message": "Logged out successfully",
	})
}

// handleAuthStatus returns the current authentication status
func (s *Server) handleAuthStatus(c *fiber.Ctx) error {
	// Get session token
	token := c.Cookies("session_token")
	if token == "" {
		authHeader := c.Get("Authorization")
		if authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token = authHeader[7:]
		}
	}

	// Check if users exist to determine if initial setup is needed
	hasUsers := s.userStore.HasUsers()
	needsSetup := !hasUsers

	// If setup is needed, enable auth temporarily for the setup flow
	if needsSetup {
		return c.JSON(fiber.Map{
			"enabled":       true,  // Auth is enabled during setup
			"authenticated": false, // Can't be authenticated during setup
			"requireLogin":  false, // Don't require login during setup
			"needs_setup":   true,  // Signal that setup is required
		})
	}

	// Check if user auth is enabled
	if !s.config.Auth.UserAuthEnabled {
		return c.JSON(fiber.Map{
			"enabled":       false,
			"authenticated": false,
			"requireLogin":  false,
			"needs_setup":   false,
		})
	}

	// Check if OAuth is enabled
	oauthEnabled := s.config.Auth.OAuth != nil && s.config.Auth.OAuth.Enabled
	oauthProviders := make([]string, 0)
	if oauthEnabled && s.config.Auth.OAuth.Providers != nil {
		for providerName, provider := range s.config.Auth.OAuth.Providers {
			if provider.Enabled {
				oauthProviders = append(oauthProviders, providerName)
			}
		}
	}

	// Validate session
	if token != "" {
		user, err := s.userStore.ValidateSession(token)
		if err == nil {
			// Get session to check MFA status
			session := s.userStore.GetSession(token)

			// Prepare response
			response := fiber.Map{
				"enabled":        true,
				"authenticated":  true,
				"requireLogin":   s.config.Auth.RequireLogin,
				"username":       user.Username,
				"isAdmin":        user.IsAdmin,
				"oauthEnabled":   oauthEnabled,
				"oauthProviders": oauthProviders,
				"needs_setup":    false, // Users exist if we're authenticated
			}

			// Add MFA status if session exists
			if session != nil {
				response["mfa_required"] = session.MFARequired
				response["mfa_verified"] = session.MFAVerified
				// If MFA required but not verified, include temp token
				if session.MFARequired && !session.MFAVerified {
					response["mfa_temp_token"] = session.MFATempToken
				}
			}

			// Add avatar data if user has an avatar
			if user.AvatarID != nil {
				avatar, err := s.repo.GetAvatarByID(*user.AvatarID)
				if err == nil && avatar != nil {
					response["avatar_id"] = avatar.ID
					response["avatar_name"] = avatar.Name
					response["avatar_color"] = avatar.Color

					// Set avatar image
					if avatar.ImagePath != "" {
						if !strings.HasPrefix(avatar.ImagePath, "http") && !strings.HasPrefix(avatar.ImagePath, "/") {
							response["avatar_image"] = fmt.Sprintf("/api/avatars/%d/image", avatar.ID)
						} else {
							response["avatar_image"] = avatar.ImagePath
						}
					}
				}
			}

			return c.JSON(response)
		}
	}

	return c.JSON(fiber.Map{
		"enabled":        true,
		"authenticated":  false,
		"requireLogin":   s.config.Auth.RequireLogin,
		"oauthEnabled":   oauthEnabled,
		"oauthProviders": oauthProviders,
		"needs_setup":    false, // Users exist if we reach here
	})
}

// handleChangePassword handles password change
func (s *Server) handleChangePassword(c *fiber.Ctx) error {
	// Get current user from context
	user, ok := c.Locals("user").(*User)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Not authenticated",
		})
	}

	var req ChangePasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Update password
	if err := s.userStore.UpdatePassword(user.Username, req.OldPassword, req.NewPassword); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Password updated successfully",
	})
}

// handleCreateUser handles user creation (admin only)
func (s *Server) handleCreateUser(c *fiber.Ctx) error {
	// Get current user from context
	user, ok := c.Locals("user").(*User)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Not authenticated",
		})
	}

	// Check if user is admin
	if !user.IsAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Admin access required",
		})
	}

	var req CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Create user
	if err := s.userStore.CreateUser(req.Username, req.Password, req.IsAdmin); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message":  "User created successfully",
		"username": req.Username,
	})
}

// handleListUsers lists all users (admin only)
func (s *Server) handleListUsers(c *fiber.Ctx) error {
	// Get current user from context
	user, ok := c.Locals("user").(*User)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Not authenticated",
		})
	}

	// Check if user is admin
	if !user.IsAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Admin access required",
		})
	}

	users := s.userStore.ListUsers()

	return c.JSON(fiber.Map{
		"users": users,
		"count": len(users),
	})
}

// handleDeleteUser deletes a user (admin only)
func (s *Server) handleDeleteUser(c *fiber.Ctx) error {
	// Get current user from context
	user, ok := c.Locals("user").(*User)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Not authenticated",
		})
	}

	// Check if user is admin
	if !user.IsAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Admin access required",
		})
	}

	username := c.Params("username")
	if username == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Username is required",
		})
	}

	// Prevent deleting self
	if username == user.Username {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot delete your own account",
		})
	}

	// Delete user
	if err := s.userStore.DeleteUser(username); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "User deleted successfully",
	})
}

// handleInitialSetup handles initial setup when no users exist
func (s *Server) handleInitialSetup(c *fiber.Ctx) error {
	// SECURITY: Check if users already exist
	if s.userStore.HasUsers() {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Setup already completed. Users already exist.",
		})
	}

	// Parse request
	var req SetupRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate username
	if req.Username == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Username is required",
		})
	}

	// Validate password
	if len(req.Password) < 8 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Password must be at least 8 characters",
		})
	}

	// Create the first admin user
	if err := s.userStore.CreateUser(req.Username, req.Password, true); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to create admin user: %v", err),
		})
	}

	// Enable user auth in config if not already enabled
	if !s.config.Auth.UserAuthEnabled {
		configManager := NewConfigManager(s.claudeDir)
		if err := configManager.EnableUserAuth(false); err != nil {
			// Log error but don't fail the setup
			fmt.Printf("Warning: Failed to enable user auth in config: %v\n", err)
		}
		// Update server config in memory
		s.config.Auth.UserAuthEnabled = true
	}

	// Create session token for the new user
	token, err := s.userStore.generateSessionTokenForUser(req.Username)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "User created but failed to create session",
		})
	}

	// Set session cookie
	c.Cookie(&fiber.Cookie{
		Name:     "session_token",
		Value:    token,
		Path:     "/",
		HTTPOnly: true,
		Secure:   s.config.TLS.Enabled || c.Get("X-Forwarded-Proto") == "https",
		SameSite: "Lax",
		Expires:  time.Now().Add(SessionDuration),
	})

	// Return success response (SECURITY: token not included in response body - cookie only)
	expiresAt := time.Now().Add(SessionDuration)
	return c.JSON(SetupResponse{
		Username:  req.Username,
		ExpiresAt: expiresAt,
		Message:   "Initial setup completed successfully. Welcome to Wee!",
	})
}
