package routes

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/schlunsen/wee-editor/internal/server/handlers"
)

// AuthHandlers groups all auth-related handler functions
type AuthHandlers struct {
	// AuthHandler methods
	HandleLogin          fiber.Handler
	HandleLogout         fiber.Handler
	HandleAuthStatus     fiber.Handler
	HandleChangePassword fiber.Handler
	HandleCreateUser     fiber.Handler
	HandleListUsers      fiber.Handler
	HandleUpdateUser     fiber.Handler
	HandleDeleteUser     fiber.Handler
	HandleInitialSetup   fiber.Handler // Initial setup endpoint

	// MFA handler methods (still Server methods until refactored)
	HandleMFASetupStart       fiber.Handler
	HandleMFASetupVerify      fiber.Handler
	HandleMFAVerify           fiber.Handler
	HandleMFAStatus           fiber.Handler
	HandleMFADisable          fiber.Handler
	HandleMFABackupCodesRegen fiber.Handler
	HandleMFAAuditLog         fiber.Handler

	// OAuth handler methods (still Server methods until refactored)
	HandleOAuthLogin    fiber.Handler
	HandleOAuthCallback fiber.Handler
}

// AuthConfig contains auth configuration
type AuthConfig struct {
	UserAuthEnabled bool
	OAuthEnabled    bool
}

// RegisterAuth configures all authentication-related endpoints
func RegisterAuth(api fiber.Router, authConfig AuthConfig, authHandlers *AuthHandlers, userProfileHandler *handlers.UserProfileHandler) {
	auth := api.Group("/auth")

	// Setup endpoint is always available (used for initial user creation)
	// Rate limiter for setup endpoint (10 attempts per hour per IP)
	setupLimiter := limiter.New(limiter.Config{
		Max:        10,
		Expiration: 60 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":   "Too many setup attempts",
				"message": "You have exceeded the maximum number of setup attempts. Please try again in 1 hour.",
			})
		},
	})
	auth.Post("/setup", setupLimiter, authHandlers.HandleInitialSetup)

	// Status endpoint is always available (even when auth is disabled)
	// This is needed to check if setup is required
	auth.Get("/status", authHandlers.HandleAuthStatus)

	// Return early if user auth is not enabled
	if !authConfig.UserAuthEnabled {
		return
	}

	// Rate limiter for login endpoint (5 attempts per 15 minutes per IP)
	loginLimiter := limiter.New(limiter.Config{
		Max:        5,
		Expiration: 15 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP() // Rate limit by IP address
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":   "Too many login attempts",
				"message": "You have exceeded the maximum number of login attempts. Please try again in 15 minutes.",
			})
		},
	})

	// Rate limiter for password change (3 attempts per 15 minutes per IP)
	passwordLimiter := limiter.New(limiter.Config{
		Max:        3,
		Expiration: 15 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":   "Too many password change attempts",
				"message": "You have exceeded the maximum number of password change attempts. Please try again in 15 minutes.",
			})
		},
	})

	// Auth endpoints (using AuthHandler)
	auth.Post("/login", loginLimiter, authHandlers.HandleLogin)
	auth.Post("/logout", authHandlers.HandleLogout)
	auth.Get("/status", authHandlers.HandleAuthStatus)
	auth.Post("/change-password", passwordLimiter, authHandlers.HandleChangePassword)

	// MFA endpoints
	auth.Post("/mfa/setup", authHandlers.HandleMFASetupStart)
	auth.Post("/mfa/setup/verify", authHandlers.HandleMFASetupVerify)
	auth.Post("/mfa/verify", authHandlers.HandleMFAVerify)
	auth.Get("/mfa/status", authHandlers.HandleMFAStatus)
	auth.Post("/mfa/disable", authHandlers.HandleMFADisable)
	auth.Post("/mfa/backup-codes/regenerate", authHandlers.HandleMFABackupCodesRegen)
	auth.Get("/mfa/audit-log", authHandlers.HandleMFAAuditLog)

	// Rate limiter for admin user management (30 attempts per 15 minutes per IP)
	adminLimiter := limiter.New(limiter.Config{
		Max:        30,
		Expiration: 15 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":   "Too many requests",
				"message": "You have exceeded the maximum number of admin requests. Please try again in 15 minutes.",
			})
		},
	})

	// Admin-only endpoints (using AuthHandler)
	auth.Post("/users", adminLimiter, authHandlers.HandleCreateUser)
	auth.Get("/users", adminLimiter, authHandlers.HandleListUsers)
	auth.Patch("/users/:username", adminLimiter, authHandlers.HandleUpdateUser)
	auth.Delete("/users/:username", adminLimiter, authHandlers.HandleDeleteUser)

	// OAuth endpoints (if OAuth is enabled)
	if authConfig.OAuthEnabled {
		auth.Get("/oauth/authorize", authHandlers.HandleOAuthLogin)
		auth.Post("/oauth/callback", authHandlers.HandleOAuthCallback)
		auth.Get("/oauth/callback/google", authHandlers.HandleOAuthCallback) // Google uses GET for callback
	}

	// User profile endpoints
	auth.Get("/user/profile", userProfileHandler.HandleGetUserProfile)
	auth.Put("/user/profile", userProfileHandler.HandleUpdateUserProfile)
}
