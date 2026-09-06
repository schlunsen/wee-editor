package server

import (
	"crypto/subtle"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/pterm/pterm"
)

// AuthMiddleware creates a middleware for API key authentication
type AuthMiddleware struct {
	apiKey  string
	enabled bool
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(apiKey string, enabled bool) *AuthMiddleware {
	return &AuthMiddleware{
		apiKey:  apiKey,
		enabled: enabled,
	}
}

// Handler returns the Fiber middleware handler
func (am *AuthMiddleware) Handler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip if authentication is disabled
		if !am.enabled {
			return c.Next()
		}

		// Public endpoints that don't require authentication
		publicEndpoints := map[string]bool{
			// Health and version endpoints
			"/health":     true,
			"/version":    true,
			"/api/health": true,
			"/api/version": true,
			// Auth endpoints
			"/api/auth/login":    true,
			"/api/auth/status":   true,
			"/api/auth/logout":   true,
			"/api/auth/setup":    true, // Initial setup endpoint
			"/api/auth/mfa/verify": true,
			// OAuth endpoints
			"/api/auth/oauth/authorize":       true,
			"/api/auth/oauth/callback":        true,
			"/api/auth/oauth/callback/google": true,
		}

		// Check if this is a public endpoint
		path := c.Path()
		if publicEndpoints[path] {
			return c.Next()
		}

		// Allow login page and related assets without authentication
		if path == "/login" || path == "/favicon.ico" || path == "/favicon.svg" ||
			strings.HasPrefix(path, "/assets/") || strings.HasPrefix(path, "/_nuxt/") {
			return c.Next()
		}

		// For all other frontend routes, check if user is authenticated (via sessionAuthMiddleware)
		// If not authenticated AND not an API request, redirect to login
		if !strings.HasPrefix(path, "/api/") && !strings.HasPrefix(path, "/ws") {
			// Check if user was authenticated by sessionAuthMiddleware
			user := c.Locals("user")
			if user == nil {
				// No authenticated user, redirect to login page
				return c.Redirect("/login", fiber.StatusFound)
			}
			// User is authenticated, allow access
			return c.Next()
		}

		// Check for WebSocket upgrade request
		// WebSocket upgrades use GET with Connection: Upgrade header
		isWebSocketUpgrade := c.Get("Connection") == "Upgrade" && c.Get("Upgrade") == "websocket"

		// Extract token from either header or query parameter
		token := extractToken(c)

		// If no token found, return unauthorized error
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Missing Authorization header or token",
				"message": "API key required. Include 'Authorization: Bearer <api-key>' header or '?token=<api-key>' query parameter",
			})
		}

		// Validate token using constant-time comparison to prevent timing attacks
		if subtle.ConstantTimeCompare([]byte(token), []byte(am.apiKey)) != 1 {
			pterm.Warning.Printf("Unauthorized API request from %s (WebSocket: %v)\n", c.IP(), isWebSocketUpgrade)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Invalid API key",
				"message": "The provided API key is not valid",
			})
		}

		// Token is valid, continue
		return c.Next()
	}
}

// extractToken extracts the API token from either the Authorization header or query parameter
// Supports both header-based tokens and query parameter tokens for WebSocket compatibility
func extractToken(c *fiber.Ctx) string {
	// Try Authorization header first (preferred method)
	authHeader := c.Get("Authorization")
	if authHeader != "" {
		// Parse Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && parts[0] == "Bearer" {
			return parts[1]
		}
	}

	// Fall back to query parameter for WebSocket connections
	// WebSocket cannot send custom headers during the initial upgrade request
	token := c.Query("token")
	if token != "" {
		return token
	}

	return ""
}

// ProtectEndpoint creates a one-off middleware to protect specific endpoints
func ProtectEndpoint(apiKey string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authentication required",
			})
		}

		parts := strings.SplitN(authHeader, " ", 2)
		// Use constant-time comparison to prevent timing attacks
		if len(parts) != 2 || parts[0] != "Bearer" || subtle.ConstantTimeCompare([]byte(parts[1]), []byte(apiKey)) != 1 {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid credentials",
			})
		}

		return c.Next()
	}
}
