package server

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/schlunsen/wee-editor/internal/database"
	"github.com/schlunsen/wee-editor/internal/logging"
)

// OAuthState represents OAuth state token for CSRF protection
type OAuthState struct {
	State        string
	Provider     string
	Nonce        string
	CodeVerifier string // PKCE code verifier (AUTH-VULN-05)
	CreatedAt    time.Time
	ExpiresAt    time.Time
}

// OAuthTokenResponse represents the token response from OAuth provider
type OAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
}

// OAuthUserInfo represents user info from OAuth provider
type OAuthUserInfo struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	// Google-specific fields
	Picture string `json:"picture,omitempty"`
	Sub     string `json:"sub,omitempty"`
	// Generic fields
	GivenName  string `json:"given_name,omitempty"`
	FamilyName string `json:"family_name,omitempty"`
}

// OAuthStateManager manages OAuth state tokens for CSRF protection
type OAuthStateManager struct {
	db *database.Database // Database connection for persistent state storage
}

// NewOAuthStateManager creates a new OAuth state manager
func NewOAuthStateManager(db *database.Database) *OAuthStateManager {
	return &OAuthStateManager{
		db: db,
	}
}

// GenerateState generates a new OAuth state token for CSRF protection
// SECURITY: Also generates PKCE code_verifier for authorization code interception protection (AUTH-VULN-05)
func (osm *OAuthStateManager) GenerateState(provider string) (string, error) {
	// Generate random bytes for state token
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate state token: %w", err)
	}
	state := hex.EncodeToString(bytes)

	// Generate nonce for additional security
	nonceBytes := make([]byte, 16)
	if _, err := rand.Read(nonceBytes); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}
	nonce := hex.EncodeToString(nonceBytes)

	// SECURITY: Generate PKCE code_verifier (RFC 7636)
	// 32 bytes = 43 base64url chars, meeting the 43-128 char requirement
	codeVerifierBytes := make([]byte, 32)
	if _, err := rand.Read(codeVerifierBytes); err != nil {
		return "", fmt.Errorf("failed to generate PKCE code verifier: %w", err)
	}
	codeVerifier := base64.RawURLEncoding.EncodeToString(codeVerifierBytes)

	// Save state to database (including code_verifier)
	expiresAt := time.Now().Add(15 * time.Minute)
	query := `
		INSERT INTO oauth_state_tokens (state, provider, nonce, code_verifier, expires_at)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err := osm.db.GetDB().Exec(query, state, provider, nonce, codeVerifier, expiresAt)
	if err != nil {
		return "", fmt.Errorf("failed to store state token: %w", err)
	}

	return state, nil
}

// generateCodeChallenge generates the PKCE code_challenge from a code_verifier using S256 method
func generateCodeChallenge(codeVerifier string) string {
	hash := sha256.Sum256([]byte(codeVerifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// ValidateState validates an OAuth state token and returns the provider, nonce, and PKCE code_verifier
// Returns (provider, nonce, codeVerifier, error)
func (osm *OAuthStateManager) ValidateState(state string) (string, string, string, error) {
	// Query state from database (including code_verifier for PKCE)
	query := `
		SELECT provider, nonce, COALESCE(code_verifier, ''), expires_at
		FROM oauth_state_tokens
		WHERE state = ?
	`

	var provider, nonce, codeVerifier string
	var expiresAt time.Time
	err := osm.db.GetDB().QueryRow(query, state).Scan(&provider, &nonce, &codeVerifier, &expiresAt)

	if err == sql.ErrNoRows {
		return "", "", "", errors.New("invalid state token: token not found or expired")
	}
	if err != nil {
		return "", "", "", fmt.Errorf("failed to validate state token: %w", err)
	}

	// Check if state has expired
	if time.Now().After(expiresAt) {
		// Clean up expired token
		deleteQuery := `DELETE FROM oauth_state_tokens WHERE state = ?`
		osm.db.GetDB().Exec(deleteQuery, state)
		return "", "", "", errors.New("invalid state token: token expired")
	}

	// Clean up state (one-time use)
	deleteQuery := `DELETE FROM oauth_state_tokens WHERE state = ?`
	osm.db.GetDB().Exec(deleteQuery, state)

	return provider, nonce, codeVerifier, nil
}

// handleOAuthLogin handles the OAuth login request
// Returns the OAuth authorization URL that the client should redirect to
func (s *Server) handleOAuthLogin(c *fiber.Ctx) error {
	// Get provider from query parameter
	provider := c.Query("provider")
	if provider == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "OAuth provider is required",
		})
	}

	// Check if OAuth is enabled
	if s.config.Auth.OAuth == nil || !s.config.Auth.OAuth.Enabled {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "OAuth is not enabled",
		})
	}

	// Get provider configuration
	oauthProvider, exists := s.config.Auth.OAuth.Providers[provider]
	if !exists || !oauthProvider.Enabled {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("OAuth provider '%s' is not enabled", provider),
		})
	}

	// Generate state token for CSRF protection
	state, err := s.oauthStateManager.GenerateState(provider)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate state token",
		})
	}

	// Retrieve the nonce and code_verifier from the state manager
	// (they were just created in GenerateState)
	var nonce, codeVerifier string
	nonceQuery := `
		SELECT nonce, COALESCE(code_verifier, '') FROM oauth_state_tokens WHERE state = ?
	`
	nonceErr := s.oauthStateManager.db.GetDB().QueryRow(nonceQuery, state).Scan(&nonce, &codeVerifier)

	// Build authorization URL
	authParams := url.Values{}
	authParams.Add("client_id", oauthProvider.ClientID)
	authParams.Add("redirect_uri", oauthProvider.RedirectURI)
	authParams.Add("response_type", "code")
	authParams.Add("state", state)
	authParams.Add("scope", strings.Join(oauthProvider.Scopes, " "))

	if nonceErr == nil && nonce != "" {
		authParams.Add("nonce", nonce)
	}

	// SECURITY: Add PKCE code_challenge to prevent authorization code interception (AUTH-VULN-05)
	if codeVerifier != "" {
		codeChallenge := generateCodeChallenge(codeVerifier)
		authParams.Add("code_challenge", codeChallenge)
		authParams.Add("code_challenge_method", "S256")
	}

	authURL := fmt.Sprintf("%s?%s", oauthProvider.AuthURL, authParams.Encode())

	return c.JSON(fiber.Map{
		"auth_url": authURL,
		"provider": provider,
		"state":    state,
	})
}

// handleOAuthCallback handles the OAuth callback from provider
// This is called after the user authorizes the app and is redirected back
func (s *Server) handleOAuthCallback(c *fiber.Ctx) error {
	// Get authorization code and state from query parameters
	code := c.Query("code")
	state := c.Query("state")
	errorMsg := c.Query("error")
	errorDescription := c.Query("error_description")

	// Debug: Log the raw code as received
	logging.Info("OAuth Callback received:")
	logging.Info("  Raw code from query: %s", code)
	logging.Info("  Code length: %d", len(code))

	// Check for OAuth provider error response
	if errorMsg != "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":       errorMsg,
			"description": errorDescription,
		})
	}

	// Validate state token (CSRF protection)
	if state == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Missing state parameter",
		})
	}

	provider, _, codeVerifier, err := s.oauthStateManager.ValidateState(state)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Validate code
	if code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Missing authorization code",
		})
	}

	// Check if OAuth is enabled
	if s.config.Auth.OAuth == nil || !s.config.Auth.OAuth.Enabled {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "OAuth is not enabled",
		})
	}

	// Get provider configuration
	oauthProvider, exists := s.config.Auth.OAuth.Providers[provider]
	if !exists || !oauthProvider.Enabled {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("OAuth provider '%s' is not configured", provider),
		})
	}

	// Exchange authorization code for access token (with PKCE code_verifier)
	accessToken, err := s.exchangeCodeForToken(code, oauthProvider, codeVerifier)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to exchange code for token: %v", err),
		})
	}

	// Fetch user info using access token
	userInfo, err := s.fetchUserInfo(accessToken, oauthProvider)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to fetch user info: %v", err),
		})
	}

	// Validate user access (check email domain/whitelist)
	accessValidator := NewAccessValidator(s.config.Auth.AccessControl, s.db)
	if err := accessValidator.ValidateOAuthEmail(userInfo.Email); err != nil {
		// Log access denial
		_ = accessValidator.LogAccessDenied("", userInfo.Email, "oauth_login_denied", err.Error(), c.IP(), c.Get("User-Agent"))
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": accessValidator.GetDenyMessage(),
		})
	}

	// Check if user already exists
	existingUser := s.userStore.FindUserByOAuthID(provider, userInfo.ID)

	var username string
	var isAdmin bool

	if existingUser != nil {
		// User already exists, just create session
		username = existingUser.Username
		isAdmin = existingUser.IsAdmin
	} else {
		// New OAuth user - check if auto-create is enabled
		if !oauthProvider.AutoCreateUser {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "User account creation is not enabled for this OAuth provider",
			})
		}

		// Generate username from email
		username = s.generateUsernameFromEmail(userInfo.Email)

		// SECURITY: Check if username exists using lock-safe method (not direct map access)
		for {
			if !s.userStore.UserExists(username) {
				break
			}
			// Username already exists, append a number
			username = fmt.Sprintf("%s_%d", username, time.Now().UnixNano()%10000)
		}

		// Check if this should be an admin user
		isAdmin = accessValidator.IsAdmin(username, userInfo.Email, s.config.Auth.AdminUsers)

		// Create new OAuth user
		if err := s.userStore.CreateOAuthUser(username, userInfo.Email, provider, userInfo.ID, isAdmin); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to create user account: %v", err),
			})
		}
	}

	// Log successful authentication
	_ = accessValidator.LogAuthSuccess(username, userInfo.Email, "oauth_login_success", c.IP(), c.Get("User-Agent"))

	// ========== SECURITY FIX: Check if user has MFA enabled ==========
	// This prevents session token leakage when MFA is enabled
	repo := database.NewRepository(s.db)
	mfaConfig, err := repo.GetMFAConfig(username)
	if err != nil {
		// Log error but continue (MFA might not be set up)
		fmt.Printf("Error checking MFA config during OAuth: %v\n", err)
	}

	// If MFA is enabled, create a partial session with MFA pending
	if mfaConfig != nil && mfaConfig.IsEnabled {
		// Generate temporary token for MFA verification
		tempToken := GenerateTemporaryToken()
		expiresAt := CalculateTokenExpiration(300) // 5 minutes

		temporaryToken := &database.MFATemporaryToken{
			Token:       tempToken,
			Username:    username,
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
			Username:  username,
			Action:    "oauth_mfa_required",
			Method:    "totp",
			Success:   true,
			IPAddress: c.IP(),
			UserAgent: c.Get("User-Agent"),
		}
		_ = repo.LogMFAAudit(auditLog)

		// Create a partial session with MFA pending (not yet verified)
		sessionToken, err := s.userStore.createMFAPendingSession(username, tempToken)
		if err != nil {
			fmt.Printf("[OAUTH MFA] Error creating MFA pending session: %v\n", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to create session",
			})
		}

		// Determine if request is over HTTPS (check both TLS config and X-Forwarded-Proto header for proxy support)
		isHTTPS := s.config.TLS.Enabled || c.Get("X-Forwarded-Proto") == "https"

		// Set session cookie
		c.Cookie(&fiber.Cookie{
			Name:     "session_token",
			Value:    sessionToken,
			Path:     "/",
			HTTPOnly: true,
			Secure:   isHTTPS,
			SameSite: "Lax",
			Expires:  expiresAt, // Shorter expiry for MFA pending sessions
		})

		logging.Info("OAuth MFA login - User %s requires MFA, cookie set (HTTPS: %v)", username, isHTTPS)
		// Redirect to dashboard - frontend middleware will detect MFA requirement
		return c.Redirect("/", fiber.StatusSeeOther)
	}
	// ========== END: MFA check ==========

	// No MFA required, proceed with session creation (existing behavior)
	// Create session token
	token, err := s.userStore.generateSessionTokenForUser(username)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create session",
		})
	}

	// Determine if request is over HTTPS (check both TLS config and X-Forwarded-Proto header for proxy support)
	isHTTPS := s.config.TLS.Enabled || c.Get("X-Forwarded-Proto") == "https"

	// Set session cookie
	c.Cookie(&fiber.Cookie{
		Name:     "session_token",
		Value:    token,
		Path:     "/",
		HTTPOnly: true,
		Secure:   isHTTPS,
		SameSite: "Lax",
		Expires:  time.Now().Add(SessionDuration),
	})

	logging.Info("OAuth login successful for user: %s, cookie set (HTTPS: %v)", username, isHTTPS)

	// Redirect to dashboard after successful OAuth login
	return c.Redirect("/", fiber.StatusSeeOther)
}

// exchangeCodeForToken exchanges an authorization code for an access token
// SECURITY: Includes PKCE code_verifier to prove the same client that initiated the flow (AUTH-VULN-05)
func (s *Server) exchangeCodeForToken(code string, provider *OAuthProvider, codeVerifier string) (string, error) {
	// SECURITY: Only log non-sensitive OAuth request metadata
	logging.Info("OAuth Token Exchange: redirect_uri=%s", provider.RedirectURI)

	// Prepare token request
	tokenParams := url.Values{}
	tokenParams.Add("grant_type", "authorization_code")
	tokenParams.Add("code", code)
	tokenParams.Add("redirect_uri", provider.RedirectURI)
	tokenParams.Add("client_id", provider.ClientID)
	tokenParams.Add("client_secret", provider.ClientSecret)

	// SECURITY: Include PKCE code_verifier for authorization code interception protection
	if codeVerifier != "" {
		tokenParams.Add("code_verifier", codeVerifier)
	}

	// Make token request
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	requestBody := tokenParams.Encode()

	req, err := http.NewRequestWithContext(ctx, "POST", provider.TokenURL, strings.NewReader(requestBody))
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to exchange code for token: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		logging.Error("OAuth Token Exchange Failed!")
		logging.Error("  Status: %d", resp.StatusCode)
		logging.Error("  Response: %s", string(body))

		// Parse error for better diagnostics
		var errResp struct {
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
		}
		if json.Unmarshal(body, &errResp) == nil && errResp.Error != "" {
			logging.Error("OAuth Error Type: %s", errResp.Error)
			if errResp.Error == "invalid_client" {
				logging.Error("INVALID_CLIENT error - This usually means:")
				logging.Error("  1. Client Secret doesn't match Google Cloud Console")
				logging.Error("  2. Client ID doesn't match Google Cloud Console")
				logging.Error("  3. OAuth client was deleted/disabled in Google Cloud Console")
				logging.Error("  4. Check https://console.cloud.google.com/apis/credentials")
			}
		}

		return "", fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse token response
	var tokenResp OAuthTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", errors.New("no access token in response")
	}

	return tokenResp.AccessToken, nil
}

// fetchUserInfo fetches user information from the OAuth provider
func (s *Server) fetchUserInfo(accessToken string, provider *OAuthProvider) (*OAuthUserInfo, error) {
	// Make user info request
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", provider.UserInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create user info request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("user info request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse user info response
	var userInfo OAuthUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to parse user info: %w", err)
	}

	// Normalize ID field (Google uses 'sub' field for OpenID Connect)
	if userInfo.ID == "" && userInfo.Sub != "" {
		userInfo.ID = userInfo.Sub
	}

	// Validate required fields
	if userInfo.ID == "" {
		return nil, errors.New("user ID not found in OAuth response")
	}
	if userInfo.Email == "" {
		return nil, errors.New("user email not found in OAuth response")
	}

	return &userInfo, nil
}

// generateUsernameFromEmail generates a username from an email address
// Converts email to username format (e.g., "john.doe@example.com" -> "john.doe")
func (s *Server) generateUsernameFromEmail(email string) string {
	// Extract local part of email (before @)
	parts := strings.Split(email, "@")
	if len(parts) > 0 {
		username := parts[0]
		// Replace common separators with underscores for valid usernames
		username = strings.NewReplacer(
			".", "_",
			"+", "_",
			"-", "_",
		).Replace(username)
		return username
	}
	return "oauth_user"
}

// OAuth-related error types
type OAuthError struct {
	Code        string
	Description string
	Timestamp   time.Time
}

// HandleOAuthError handles OAuth-related errors with proper logging
func HandleOAuthError(c *fiber.Ctx, code string, description string) error {
	oauthErr := &OAuthError{
		Code:        code,
		Description: description,
		Timestamp:   time.Now(),
	}

	// Log error to database for audit trail
	// This would be implemented in the database layer

	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
		"error":       oauthErr.Code,
		"description": oauthErr.Description,
		"timestamp":   oauthErr.Timestamp,
	})
}

// Predefined OAuth providers with standard configurations
var StandardOAuthProviders = map[string]*OAuthProvider{
	"google": {
		Enabled:     false, // Disabled by default, must be explicitly enabled
		ClientID:    "", // Must be set via config or env var
		ClientSecret: "", // Must be set via env var (WEE_OAUTH_GOOGLE_CLIENT_SECRET)
		RedirectURI: "http://localhost:3333/api/oauth/callback/google",
		Scopes: []string{
			"openid",
			"email",
			"profile",
		},
		AuthURL:        "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:       "https://oauth2.googleapis.com/token",
		UserInfoURL:    "https://openidconnect.googleapis.com/v1/userinfo",
		AutoCreateUser: true,
		AutoEnableMFA:  false,
	},
}
