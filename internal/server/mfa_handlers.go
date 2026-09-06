package server

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/schlunsen/wee-editor/internal/database"
)

// handleMFASetupStart starts the MFA setup process
// POST /api/auth/mfa/setup
func (s *Server) handleMFASetupStart(c *fiber.Ctx) error {
	// Get current user from context
	user, ok := c.Locals("user").(*User)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Not authenticated",
		})
	}

	// Check if MFA is already enabled
	repo := database.NewRepository(s.db)
	mfaConfig, err := repo.GetMFAConfig(user.Username)
	if err != nil {
		// Log the error for debugging
		fmt.Printf("Error getting MFA config: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check MFA status",
			"details": err.Error(),
		})
	}

	if mfaConfig != nil && mfaConfig.IsEnabled {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "MFA is already enabled for this account",
		})
	}

	// Generate TOTP secret
	key, err := s.mfaManager.GenerateTOTPSecret(user.Username, "Wee")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate TOTP secret",
		})
	}

	// Generate QR code
	qrCode, err := s.mfaManager.GenerateQRCode(key)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate QR code",
		})
	}

	// Generate backup codes
	plainCodes, err := s.mfaManager.GenerateBackupCodes(10)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate backup codes",
		})
	}

	// Generate temporary token for setup
	tempToken := GenerateTemporaryToken()
	expiresAt := CalculateTokenExpiration(300) // 5 minutes

	temporaryToken := &database.MFATemporaryToken{
		Token:       tempToken,
		Username:    user.Username,
		TokenType:   "setup",
		TOTPSecret:  key.Secret(), // Store the TOTP secret for verification
		Verified:    false,
		Attempts:    0,
		MaxAttempts: 5,
		CreatedAt:   time.Now(),
		ExpiresAt:   expiresAt,
		IPAddress:   c.IP(),
		UserAgent:   c.Get("User-Agent"),
	}

	if err := repo.SaveMFATemporaryToken(temporaryToken); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save temporary token",
		})
	}

	// Log audit event
	auditLog := &database.MFAAuditLog{
		Username:  user.Username,
		Action:    "setup_started",
		Method:    "totp",
		Success:   true,
		IPAddress: c.IP(),
		UserAgent: c.Get("User-Agent"),
	}
	_ = repo.LogMFAAudit(auditLog)

	response := database.TOTPSetupResponse{
		QRCode:         qrCode,
		Secret:         key.Secret(), // Base32-encoded secret
		BackupCodes:    plainCodes,
		TemporaryToken: tempToken,
	}

	return c.JSON(response)
}

// handleMFASetupVerify verifies and completes MFA setup
// POST /api/auth/mfa/setup/verify
func (s *Server) handleMFASetupVerify(c *fiber.Ctx) error {
	// Get current user from context
	user, ok := c.Locals("user").(*User)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Not authenticated",
		})
	}

	var req database.MFASetupVerifyRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate TOTP code format
	validator := &MFAConfigValidator{}
	if !validator.ValidateTOTPCode(req.Code) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid TOTP code format (must be 6 digits)",
		})
	}

	repo := database.NewRepository(s.db)

	// Get temporary token
	tempToken, err := repo.GetMFATemporaryToken(req.TemporaryToken)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to validate temporary token",
		})
	}

	if tempToken == nil || tempToken.Username != user.Username || tempToken.TokenType != "setup" {
		auditLog := &database.MFAAuditLog{
			Username:  user.Username,
			Action:    "setup_failed",
			Method:    "totp",
			Success:   false,
			IPAddress: c.IP(),
			UserAgent: c.Get("User-Agent"),
			Reason:    "Invalid or expired temporary token",
		}
		_ = repo.LogMFAAudit(auditLog)

		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid or expired setup token",
		})
	}

	// Check if token has expired
	if !VerifyTokenNotExpired(tempToken.ExpiresAt) {
		auditLog := &database.MFAAuditLog{
			Username:  user.Username,
			Action:    "setup_failed",
			Method:    "totp",
			Success:   false,
			IPAddress: c.IP(),
			UserAgent: c.Get("User-Agent"),
			Reason:    "Temporary token expired",
		}
		_ = repo.LogMFAAudit(auditLog)

		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Setup token has expired",
		})
	}

	// Verify the TOTP code against the stored secret
	if tempToken.TOTPSecret == "" {
		auditLog := &database.MFAAuditLog{
			Username:  user.Username,
			Action:    "setup_failed",
			Method:    "totp",
			Success:   false,
			IPAddress: c.IP(),
			UserAgent: c.Get("User-Agent"),
			Reason:    "No TOTP secret found in temporary token",
		}
		_ = repo.LogMFAAudit(auditLog)

		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid setup session",
		})
	}

	// Verify the TOTP code
	valid, err := s.mfaManager.VerifyTOTPCode(tempToken.TOTPSecret, req.Code)
	if err != nil || !valid {
		auditLog := &database.MFAAuditLog{
			Username:  user.Username,
			Action:    "setup_failed",
			Method:    "totp",
			Success:   false,
			IPAddress: c.IP(),
			UserAgent: c.Get("User-Agent"),
			Reason:    "Invalid TOTP code",
		}
		_ = repo.LogMFAAudit(auditLog)

		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid verification code. Please check your authenticator app and try again.",
		})
	}

	// TOTP code verified successfully! Now generate backup codes and enable MFA
	// Generate backup codes collection
	plainCodes, err := s.mfaManager.GenerateBackupCodes(10)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate backup codes",
		})
	}

	backupCodeCollection, err := s.mfaManager.NewBackupCodeCollection(plainCodes)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to process backup codes",
		})
	}

	backupCodesJSON, err := s.mfaManager.SerializeBackupCodes(backupCodeCollection)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to serialize backup codes",
		})
	}

	// Get or create MFA config
	mfaConfig, err := repo.GetMFAConfig(user.Username)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve MFA config",
		})
	}

	if mfaConfig == nil {
		mfaConfig = &database.MFAConfig{
			Username: user.Username,
			CreatedAt: time.Now(),
		}
	}

	// Encrypt the TOTP secret for permanent storage
	encryptedSecret, iv, err := s.mfaManager.EncryptSecret(tempToken.TOTPSecret)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to encrypt TOTP secret",
		})
	}

	// Update MFA config
	now := time.Now()
	mfaConfig.IsEnabled = true
	mfaConfig.TOTPSecret = sql.NullString{String: encryptedSecret, Valid: true}
	mfaConfig.TOTPSecretIV = sql.NullString{String: iv, Valid: true}
	mfaConfig.BackupCodes = sql.NullString{String: backupCodesJSON, Valid: true}
	mfaConfig.MFAEnabledAt = &now
	mfaConfig.UpdatedAt = now

	if err := repo.SaveMFAConfig(mfaConfig); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save MFA configuration",
		})
	}

	// Log audit event
	auditLog := &database.MFAAuditLog{
		Username:  user.Username,
		Action:    "setup_completed",
		Method:    "totp",
		Success:   true,
		IPAddress: c.IP(),
		UserAgent: c.Get("User-Agent"),
	}
	_ = repo.LogMFAAudit(auditLog)

	return c.JSON(fiber.Map{
		"message": "MFA setup completed successfully",
		"status":  "enabled",
	})
}

// handleMFAVerify verifies MFA code during login
// POST /api/auth/mfa/verify
func (s *Server) handleMFAVerify(c *fiber.Ctx) error {
	var req struct {
		TemporaryToken string `json:"temporary_token"`
		Code           string `json:"code"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	repo := database.NewRepository(s.db)

	// Get temporary token
	tempToken, err := repo.GetMFATemporaryToken(req.TemporaryToken)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to validate temporary token",
		})
	}

	if tempToken == nil || tempToken.TokenType != "login" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid temporary token",
		})
	}

	// SECURITY: Log IP address mismatch for audit but don't block (AUTH-VULN-04)
	// Strict IP binding can cause DoS when users are behind load balancers, CDNs,
	// or have changing IPs (mobile networks, IPv4/IPv6 dual-stack). We log the
	// mismatch for security monitoring instead of blocking the request.
	if tempToken.IPAddress != c.IP() {
		auditLog := &database.MFAAuditLog{
			Username:  tempToken.Username,
			Action:    "verification_ip_mismatch",
			Method:    "totp",
			Success:   true,
			IPAddress: c.IP(),
			UserAgent: c.Get("User-Agent"),
			Reason:    fmt.Sprintf("IP changed: login from %s, verify from %s (allowed, logged for audit)", tempToken.IPAddress, c.IP()),
		}
		_ = repo.LogMFAAudit(auditLog)
		// Continue processing — don't block
	}

	// Check if token has expired
	if !VerifyTokenNotExpired(tempToken.ExpiresAt) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Login token has expired",
		})
	}

	// Check rate limiting
	rateLimiter := &RateLimitChecker{
		MaxAttempts:     5,
		LockoutDuration: 5 * time.Minute,
		AttemptWindow:   5 * time.Minute,
	}

	allowed, _, _ := rateLimiter.CheckRateLimit(tempToken.Attempts, &tempToken.CreatedAt)
	if !allowed {
		auditLog := &database.MFAAuditLog{
			Username:  tempToken.Username,
			Action:    "verification_failed",
			Method:    "totp",
			Success:   false,
			IPAddress: c.IP(),
			UserAgent: c.Get("User-Agent"),
			Reason:    "Rate limited - too many attempts",
		}
		_ = repo.LogMFAAudit(auditLog)

		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error": "Too many verification attempts. Please try again later.",
		})
	}

	// Increment attempt counter
	tempToken.Attempts++

	// Get MFA config
	mfaConfig, err := repo.GetMFAConfig(tempToken.Username)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve MFA config",
		})
	}

	if mfaConfig == nil || !mfaConfig.IsEnabled {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "MFA is not enabled for this account",
		})
	}

	validator := &MFAConfigValidator{}

	// Try TOTP code first
	if validator.ValidateTOTPCode(req.Code) {
		// Verify the TOTP code against the stored encrypted secret
		if !mfaConfig.TOTPSecret.Valid || mfaConfig.TOTPSecret.String == "" {
			auditLog := &database.MFAAuditLog{
				Username:  tempToken.Username,
				Action:    "verification_failed",
				Method:    "totp",
				Success:   false,
				IPAddress: c.IP(),
				UserAgent: c.Get("User-Agent"),
				Reason:    "No TOTP secret stored in MFA config",
			}
			_ = repo.LogMFAAudit(auditLog)
			_ = repo.UpdateMFATemporaryToken(tempToken)

			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "MFA configuration error",
			})
		}

		// Decrypt the TOTP secret
		decryptedSecret, err := s.mfaManager.DecryptSecret(mfaConfig.TOTPSecret.String, mfaConfig.TOTPSecretIV.String)
		if err != nil {
			auditLog := &database.MFAAuditLog{
				Username:  tempToken.Username,
				Action:    "verification_failed",
				Method:    "totp",
				Success:   false,
				IPAddress: c.IP(),
				UserAgent: c.Get("User-Agent"),
				Reason:    "Failed to decrypt TOTP secret",
			}
			_ = repo.LogMFAAudit(auditLog)
			_ = repo.UpdateMFATemporaryToken(tempToken)

			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "MFA configuration error",
			})
		}

		// Verify the TOTP code
		valid, err := s.mfaManager.VerifyTOTPCode(decryptedSecret, req.Code)
		if err != nil || !valid {
			// Invalid TOTP code
			auditLog := &database.MFAAuditLog{
				Username:  tempToken.Username,
				Action:    "verification_failed",
				Method:    "totp",
				Success:   false,
				IPAddress: c.IP(),
				UserAgent: c.Get("User-Agent"),
				Reason:    "Invalid TOTP code",
			}
			_ = repo.LogMFAAudit(auditLog)
			_ = repo.UpdateMFATemporaryToken(tempToken)

			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid verification code. Please check your authenticator app and try again.",
			})
		}

		// TOTP code is valid!
		tempToken.Verified = true
		_ = repo.UpdateMFATemporaryToken(tempToken)
		// SECURITY: Delete the temporary token immediately to prevent replay within the 5-min window
		_ = repo.DeleteMFATemporaryToken(tempToken.Token)

		auditLog := &database.MFAAuditLog{
			Username:  tempToken.Username,
			Action:    "verification_success",
			Method:    "totp",
			Success:   true,
			IPAddress: c.IP(),
			UserAgent: c.Get("User-Agent"),
		}
		_ = repo.LogMFAAudit(auditLog)

		now := time.Now()
		mfaConfig.LastVerifiedAt = &now
		mfaConfig.LastVerificationMethod = sql.NullString{String: "totp", Valid: true}
		_ = repo.SaveMFAConfig(mfaConfig)

		// Create a session token for the user
		sessionToken, err := s.userStore.generateSessionTokenForUser(tempToken.Username)
		if err != nil {
			fmt.Printf("Error generating session token: %v\n", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to create session",
			})
		}

		// Set session cookie
		c.Cookie(&fiber.Cookie{
			Name:     "session_token",
			Value:    sessionToken,
			Path:     "/",
			HTTPOnly: true,
			Secure:   s.config.TLS.Enabled || c.Get("X-Forwarded-Proto") == "https",
			SameSite: "Lax",
			Expires:  time.Now().Add(SessionDuration),
		})

		// SECURITY: Session token not included in response body - cookie only
		return c.JSON(fiber.Map{
			"message": "MFA verification successful",
			"status":  "verified",
		})
	}

	// Try backup code
	if validator.ValidateBackupCodeFormat(req.Code) {
		if !mfaConfig.BackupCodes.Valid || mfaConfig.BackupCodes.String == "" {
			auditLog := &database.MFAAuditLog{
				Username:  tempToken.Username,
				Action:    "verification_failed",
				Method:    "backup_code",
				Success:   false,
				IPAddress: c.IP(),
				UserAgent: c.Get("User-Agent"),
				Reason:    "No backup codes available",
			}
			_ = repo.LogMFAAudit(auditLog)

			_ = repo.UpdateMFATemporaryToken(tempToken)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "No backup codes available",
			})
		}

		collection, err := s.mfaManager.DeserializeBackupCodes(mfaConfig.BackupCodes.String)
		if err != nil {
			_ = repo.UpdateMFATemporaryToken(tempToken)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to process backup codes",
			})
		}

		valid, updatedCodes, err := s.mfaManager.VerifyAndUseBackupCode(collection, req.Code)
		if !valid || err != nil {
			auditLog := &database.MFAAuditLog{
				Username:  tempToken.Username,
				Action:    "verification_failed",
				Method:    "backup_code",
				Success:   false,
				IPAddress: c.IP(),
				UserAgent: c.Get("User-Agent"),
				Reason:    "Invalid or already used backup code",
			}
			_ = repo.LogMFAAudit(auditLog)

			_ = repo.UpdateMFATemporaryToken(tempToken)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid or already used backup code",
			})
		}

		// Update backup codes in config
		mfaConfig.BackupCodes = sql.NullString{String: updatedCodes, Valid: true}

		tempToken.Verified = true
		_ = repo.UpdateMFATemporaryToken(tempToken)
		// SECURITY: Delete the temporary token immediately to prevent replay
		_ = repo.DeleteMFATemporaryToken(tempToken.Token)

		auditLog := &database.MFAAuditLog{
			Username:  tempToken.Username,
			Action:    "verification_success",
			Method:    "backup_code",
			Success:   true,
			IPAddress: c.IP(),
			UserAgent: c.Get("User-Agent"),
		}
		_ = repo.LogMFAAudit(auditLog)

		now := time.Now()
		mfaConfig.LastVerifiedAt = &now
		mfaConfig.LastVerificationMethod = sql.NullString{String: "backup_code", Valid: true}
		_ = repo.SaveMFAConfig(mfaConfig)

		// Create a session token for the user
		sessionToken, err := s.userStore.generateSessionTokenForUser(tempToken.Username)
		if err != nil {
			fmt.Printf("Error generating session token: %v\n", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to create session",
			})
		}

		// Set session cookie
		c.Cookie(&fiber.Cookie{
			Name:     "session_token",
			Value:    sessionToken,
			Path:     "/",
			HTTPOnly: true,
			Secure:   s.config.TLS.Enabled || c.Get("X-Forwarded-Proto") == "https",
			SameSite: "Lax",
			Expires:  time.Now().Add(SessionDuration),
		})

		// SECURITY: Session token not included in response body - cookie only
		return c.JSON(fiber.Map{
			"message":         "MFA verification successful with backup code",
			"status":          "verified",
			"backup_codes_remaining": s.mfaManager.CountUnusedBackupCodes(collection),
		})
	}

	// Invalid code format
	_ = repo.UpdateMFATemporaryToken(tempToken)

	auditLog := &database.MFAAuditLog{
		Username:  tempToken.Username,
		Action:    "verification_failed",
		Method:    "totp",
		Success:   false,
		IPAddress: c.IP(),
		UserAgent: c.Get("User-Agent"),
		Reason:    "Invalid code format",
	}
	_ = repo.LogMFAAudit(auditLog)

	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"error": "Invalid verification code format",
	})
}

// handleMFAStatus returns MFA status
// GET /api/auth/mfa/status
func (s *Server) handleMFAStatus(c *fiber.Ctx) error {
	// Get current user from context
	user, ok := c.Locals("user").(*User)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Not authenticated",
		})
	}

	repo := database.NewRepository(s.db)
	mfaConfig, err := repo.GetMFAConfig(user.Username)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve MFA status",
		})
	}

	status := &database.MFAStatus{
		Enabled: false,
	}

	if mfaConfig != nil && mfaConfig.IsEnabled {
		status.Enabled = true
		status.Method = "totp"
		status.LastVerifiedAt = mfaConfig.LastVerifiedAt
		if mfaConfig.LastVerificationMethod.Valid {
			status.LastVerificationMethod = mfaConfig.LastVerificationMethod.String
		}

		if mfaConfig.BackupCodes.Valid && mfaConfig.BackupCodes.String != "" {
			collection, _ := s.mfaManager.DeserializeBackupCodes(mfaConfig.BackupCodes.String)
			if collection != nil {
				status.BackupCodesCount = s.mfaManager.CountUnusedBackupCodes(collection)
			}
		}
	}

	return c.JSON(status)
}

// handleMFADisable disables MFA for the user
// POST /api/auth/mfa/disable
func (s *Server) handleMFADisable(c *fiber.Ctx) error {
	// Get current user from context
	user, ok := c.Locals("user").(*User)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Not authenticated",
		})
	}

	var req struct {
		Password string `json:"password"` // Require password to disable MFA
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// SECURITY: Use VerifyPassword instead of Authenticate to avoid creating orphaned sessions
	_, err := s.userStore.VerifyPassword(user.Username, req.Password)
	if err != nil {
		auditLog := &database.MFAAuditLog{
			Username:  user.Username,
			Action:    "disable_failed",
			Method:    "totp",
			Success:   false,
			IPAddress: c.IP(),
			UserAgent: c.Get("User-Agent"),
			Reason:    "Invalid password",
		}
		repo := database.NewRepository(s.db)
		_ = repo.LogMFAAudit(auditLog)

		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid password",
		})
	}

	repo := database.NewRepository(s.db)

	// Disable MFA
	if err := repo.DisableMFAForUser(user.Username); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to disable MFA",
		})
	}

	// Log audit event
	auditLog := &database.MFAAuditLog{
		Username:  user.Username,
		Action:    "mfa_disabled",
		Method:    "totp",
		Success:   true,
		IPAddress: c.IP(),
		UserAgent: c.Get("User-Agent"),
	}
	_ = repo.LogMFAAudit(auditLog)

	return c.JSON(fiber.Map{
		"message": "MFA disabled successfully",
		"status":  "disabled",
	})
}

// handleMFABackupCodesRegenerate generates new backup codes
// POST /api/auth/mfa/backup-codes/regenerate
func (s *Server) handleMFABackupCodesRegenerate(c *fiber.Ctx) error {
	// Get current user from context
	user, ok := c.Locals("user").(*User)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Not authenticated",
		})
	}

	var req struct {
		Password string `json:"password"` // Require password to regenerate codes
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// SECURITY: Use VerifyPassword instead of Authenticate to avoid creating orphaned sessions
	_, err := s.userStore.VerifyPassword(user.Username, req.Password)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid password",
		})
	}

	repo := database.NewRepository(s.db)

	// Get current MFA config
	mfaConfig, err := repo.GetMFAConfig(user.Username)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve MFA config",
		})
	}

	if mfaConfig == nil || !mfaConfig.IsEnabled {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "MFA is not enabled for this account",
		})
	}

	// Generate new backup codes
	plainCodes, err := s.mfaManager.GenerateBackupCodes(10)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate backup codes",
		})
	}

	collection, err := s.mfaManager.NewBackupCodeCollection(plainCodes)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to process backup codes",
		})
	}

	backupCodesJSON, err := s.mfaManager.SerializeBackupCodes(collection)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to serialize backup codes",
		})
	}

	// Update config
	mfaConfig.BackupCodes = sql.NullString{String: backupCodesJSON, Valid: true}
	mfaConfig.UpdatedAt = time.Now()

	if err := repo.SaveMFAConfig(mfaConfig); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save new backup codes",
		})
	}

	// Log audit event
	auditLog := &database.MFAAuditLog{
		Username:  user.Username,
		Action:    "backup_codes_regenerated",
		Method:    "totp",
		Success:   true,
		IPAddress: c.IP(),
		UserAgent: c.Get("User-Agent"),
	}
	_ = repo.LogMFAAudit(auditLog)

	response := fiber.Map{
		"message":      "Backup codes regenerated successfully",
		"backup_codes": plainCodes,
	}

	return c.JSON(response)
}

// handleMFAAuditLog retrieves MFA audit log
// GET /api/auth/mfa/audit-log
func (s *Server) handleMFAAuditLog(c *fiber.Ctx) error {
	// Get current user from context
	user, ok := c.Locals("user").(*User)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Not authenticated",
		})
	}

	limit := c.QueryInt("limit", 50)
	offset := c.QueryInt("offset", 0)

	if limit > 500 {
		limit = 500 // Cap limit to prevent abuse
	}

	repo := database.NewRepository(s.db)
	logs, err := repo.GetMFAAuditLog(user.Username, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve audit log",
		})
	}

	return c.JSON(fiber.Map{
		"logs":  logs,
		"count": len(logs),
	})
}
