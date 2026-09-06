package server

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/schlunsen/wee-editor/internal/database"
)

// AuthHandler handles all authentication endpoints (login, logout, password change, user management)
// This consolidates auth, MFA, and OAuth functionality into a single cohesive handler
type AuthHandler struct {
	userStore          *UserStore
	mfaManager         *MFAManager
	oauthStateManager  *OAuthStateManager
	accessControl      *AccessValidator
	db                 *database.Database
	repo               *database.Repository
	config             *Config
	tlsEnabled         bool
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(
	userStore *UserStore,
	mfaManager *MFAManager,
	oauthStateManager *OAuthStateManager,
	accessControl *AccessValidator,
	db *database.Database,
	repo *database.Repository,
	config *Config,
	tlsEnabled bool,
) *AuthHandler {
	return &AuthHandler{
		userStore:         userStore,
		mfaManager:        mfaManager,
		oauthStateManager: oauthStateManager,
		accessControl:     accessControl,
		db:                db,
		repo:              repo,
		config:            config,
		tlsEnabled:        tlsEnabled,
	}
}

// HandleLogin handles user login
func (h *AuthHandler) HandleLogin(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// SECURITY: Check MFA status BEFORE creating any session tokens
	// This prevents session token leakage when MFA is enabled
	mfaConfig, err := h.repo.GetMFAConfig(req.Username)
	if err != nil {
		// Log error but continue (MFA might not be set up)
		fmt.Printf("Error checking MFA config during login: %v\n", err)
	}

	// If MFA is enabled, verify password WITHOUT creating session token
	if mfaConfig != nil && mfaConfig.IsEnabled {
		// Verify credentials without creating a session
		user, verifyErr := h.userStore.VerifyPassword(req.Username, req.Password)
		if verifyErr != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid username or password",
			})
		}

		// Validate user access via access control
		accessValidator := NewAccessValidator(h.config.Auth.AccessControl, h.db)
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

		if err := h.repo.SaveMFATemporaryToken(temporaryToken); err != nil {
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
		_ = h.repo.LogMFAAudit(auditLog)

		return c.JSON(LoginResponse{
			Username:         req.Username,
			MFARequired:      true,
			TemporaryToken:   tempToken,
			TemporaryExpires: expiresAt,
		})
	}

	// No MFA required, proceed with normal login
	// Verify credentials first
	user, authErr := h.userStore.VerifyPassword(req.Username, req.Password)
	if authErr != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid username or password",
		})
	}

	// Validate user access via access control
	accessValidator := NewAccessValidator(h.config.Auth.AccessControl, h.db)
	if err := accessValidator.ValidateUser(user.Username, user.Email); err != nil {
		// Log access denial
		_ = accessValidator.LogAccessDenied(user.Username, user.Email, "password_login_denied", err.Error(), c.IP(), c.Get("User-Agent"))
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": accessValidator.GetDenyMessage(),
		})
	}

	// Now it's safe to create a session token (both password and access control validated)
	token, sessionErr := h.userStore.generateSessionTokenForUser(user.Username)
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
		Secure:   h.tlsEnabled || c.Get("X-Forwarded-Proto") == "https",
		SameSite: "Lax",
		Expires:  time.Now().Add(SessionDuration),
	})

	return c.JSON(LoginResponse{
		Username:  req.Username,
		ExpiresAt: time.Now().Add(SessionDuration),
	})
}

// HandleLogout handles user logout
func (h *AuthHandler) HandleLogout(c *fiber.Ctx) error {
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
		h.userStore.RevokeSession(token)
	}

	// Clear cookie
	c.ClearCookie("session_token")

	return c.JSON(fiber.Map{
		"message": "Logged out successfully",
	})
}

// HandleAuthStatus returns the current authentication status
func (h *AuthHandler) HandleAuthStatus(c *fiber.Ctx) error {
	// Get session token
	token := c.Cookies("session_token")
	if token == "" {
		authHeader := c.Get("Authorization")
		if authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token = authHeader[7:]
		}
	}

	// Check if users exist to determine if initial setup is needed
	hasUsers := h.userStore.HasUsers()
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
	if !h.config.Auth.UserAuthEnabled {
		return c.JSON(fiber.Map{
			"enabled":       false,
			"authenticated": false,
			"requireLogin":  false,
			"needs_setup":   false,
		})
	}

	// Check if OAuth is enabled
	oauthEnabled := h.config.Auth.OAuth != nil && h.config.Auth.OAuth.Enabled
	oauthProviders := make([]string, 0)
	if oauthEnabled && h.config.Auth.OAuth.Providers != nil {
		for providerName, provider := range h.config.Auth.OAuth.Providers {
			if provider.Enabled {
				oauthProviders = append(oauthProviders, providerName)
			}
		}
	}

	// Validate session
	if token != "" {
		user, err := h.userStore.ValidateSession(token)
		if err == nil {
			// Get session to check MFA status
			session := h.userStore.GetSession(token)

			// Prepare response
			response := fiber.Map{
				"enabled":        true,
				"authenticated":  true,
				"requireLogin":   h.config.Auth.RequireLogin,
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
				avatar, err := h.repo.GetAvatarByID(*user.AvatarID)
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
		"requireLogin":   h.config.Auth.RequireLogin,
		"oauthEnabled":   oauthEnabled,
		"oauthProviders": oauthProviders,
		"needs_setup":    false, // Users exist if we reach here
	})
}

// HandleChangePassword handles password change
func (h *AuthHandler) HandleChangePassword(c *fiber.Ctx) error {
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
	if err := h.userStore.UpdatePassword(user.Username, req.OldPassword, req.NewPassword); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Password updated successfully",
	})
}

// HandleCreateUser handles user creation (admin only)
func (h *AuthHandler) HandleCreateUser(c *fiber.Ctx) error {
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

	// Validate username format
	if err := ValidateUsername(req.Username); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Create user
	if err := h.userStore.CreateUser(req.Username, req.Password, req.IsAdmin); err != nil {
		log.Printf("AUDIT: Admin %q failed to create user %q: %v", user.Username, req.Username, err)
		// Return safe error messages for known validation errors; generic message otherwise
		safeMsg := err.Error()
		if !isUserFacingError(err) {
			safeMsg = "Failed to create user"
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": safeMsg,
		})
	}

	log.Printf("AUDIT: Admin %q created user %q (admin=%v)", user.Username, req.Username, req.IsAdmin)

	return c.JSON(fiber.Map{
		"message":  "User created successfully",
		"username": req.Username,
	})
}

// HandleListUsers lists all users (admin only)
func (h *AuthHandler) HandleListUsers(c *fiber.Ctx) error {
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

	users := h.userStore.ListUsersDetailed()

	return c.JSON(fiber.Map{
		"users": users,
		"count": len(users),
	})
}

// HandleUpdateUser updates a user's properties (admin only)
func (h *AuthHandler) HandleUpdateUser(c *fiber.Ctx) error {
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

	var req struct {
		IsAdmin *bool `json:"is_admin"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate target username format
	if err := ValidateUsername(username); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid username",
		})
	}

	if req.IsAdmin != nil {
		// Prevent removing own admin status
		if username == user.Username && !*req.IsAdmin {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Cannot remove your own admin status",
			})
		}

		if err := h.userStore.SetUserAdmin(username, *req.IsAdmin); err != nil {
			log.Printf("AUDIT: Admin %q failed to update user %q: %v", user.Username, username, err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Failed to update user",
			})
		}
	}

	return c.JSON(fiber.Map{
		"message":  "User updated successfully",
		"username": username,
	})
}

// HandleDeleteUser deletes a user (admin only)
func (h *AuthHandler) HandleDeleteUser(c *fiber.Ctx) error {
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

	// Validate target username format
	if err := ValidateUsername(username); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid username",
		})
	}

	// Prevent deleting self
	if username == user.Username {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot delete your own account",
		})
	}

	// Delete user
	if err := h.userStore.DeleteUser(username); err != nil {
		log.Printf("AUDIT: Admin %q failed to delete user %q: %v", user.Username, username, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete user",
		})
	}

	log.Printf("AUDIT: Admin %q deleted user %q", user.Username, username)

	return c.JSON(fiber.Map{
		"message": "User deleted successfully",
	})
}

// isUserFacingError returns true if the error message is safe to show to the user
// (i.e., it's a validation error, not an internal/database error)
func isUserFacingError(err error) bool {
	msg := err.Error()
	safeMessages := []string{
		"username cannot be empty",
		"username must be",
		"password must be at least",
		"user already exists",
		"user not found",
	}
	for _, safe := range safeMessages {
		if strings.Contains(msg, safe) {
			return true
		}
	}
	return false
}
