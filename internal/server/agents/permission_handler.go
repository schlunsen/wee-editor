package agents

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	fiberws "github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"github.com/schlunsen/wee-editor/internal/logging"
)

func (h *AgentHandler) handleFiberPermissionResponse(c *fiberws.Conn, rawMsg map[string]interface{}) error {
	logging.Info("📥 RAW PERMISSION RESPONSE from frontend: %+v", rawMsg)

	var msg PermissionResponseMessage
	msgBytes, _ := json.Marshal(rawMsg)
	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		return fmt.Errorf("invalid permission_response message: %w", err)
	}

	// SECURITY: Validate session ownership before allowing permission approval/denial
	// This prevents unauthorized users from approving dangerous operations on other users' sessions
	authenticatedUser := h.GetConnectionUser(c)
	if err := h.validateSessionOwnership(msg.SessionID, authenticatedUser); err != nil {
		h.sendFiberError(c, err.Error())
		return err
	}

	logging.Info("📥 PARSED permission response: sessionID=%s, permissionID='%s', approved=%v",
		msg.SessionID, msg.PermissionID, msg.Approved)

	// Get the session
	session, err := h.SessionManager.GetSession(msg.SessionID)
	if err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	// Find the pending permission request
	// NOTE: Frontend doesn't send permission_id, so we look for any pending permission in this session
	session.permMu.Lock()
	var responseChan chan PermissionResponse
	var exists bool

	// Try to find by permission ID first (if frontend sends it)
	if msg.PermissionID != "" {
		responseChan, exists = session.pendingPermissions[msg.PermissionID]
	}

	// If not found or no ID provided, get the first (and should be only) pending permission
	if !exists {
		for _, ch := range session.pendingPermissions {
			responseChan = ch
			exists = true
			break
		}
	}
	session.permMu.Unlock()

	if !exists {
		logging.Warning("No pending permission request found for session %s (permission_id='%s')", msg.SessionID, msg.PermissionID)
		return fmt.Errorf("no pending permission request found for ID: %s", msg.PermissionID)
	}

	logging.Info("✅ Found pending permission, sending response to callback")

	// Send response to the callback
	response := PermissionResponse{
		Approved:    msg.Approved,
		DenyMessage: "User denied permission",
	}

	select {
	case responseChan <- response:
		log.Printf("Permission response delivered to callback: %s", msg.PermissionID)
	case <-time.After(5 * time.Second):
		log.Printf("ERROR: Timeout delivering permission response to callback")
		return fmt.Errorf("timeout delivering permission response")
	}

	// Send acknowledgement to frontend
	ack := BaseMessage{Type: MessageTypePermissionAcknowledged}
	return h.safeWriteJSON(c, ack)
}

func (h *AgentHandler) handleFiberAddAlwaysAllowRule(c *fiberws.Conn, rawMsg map[string]interface{}) error {
	var msg AddAlwaysAllowRuleMessage
	msgBytes, _ := json.Marshal(rawMsg)
	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		return fmt.Errorf("invalid add_always_allow_rule message: %w", err)
	}

	// SECURITY: Validate session ownership before allowing rule addition
	// This prevents unauthorized users from adding always-allow rules to other users' sessions
	authenticatedUser := h.GetConnectionUser(c)
	if err := h.validateSessionOwnership(msg.SessionID, authenticatedUser); err != nil {
		h.sendFiberError(c, err.Error())
		return err
	}

	logging.Info("Adding always-allow rule to session %s: %s (mode: %s)", msg.SessionID, msg.Rule.Description, msg.Rule.MatchMode)

	// Get session to find working directory
	session, err := h.SessionManager.GetSession(msg.SessionID)
	if err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	// Get working directory for this session
	workingDir := "."
	if session.Options.WorkingDirectory != nil {
		workingDir = *session.Options.WorkingDirectory
	}

	// Create settings manager
	settingsManager := NewClaudeSettingsManager(workingDir)

	// Format permission string based on match mode
	var permissionStr string
	if msg.Rule.MatchMode == RuleMatchExact {
		// For exact mode, format the exact parameters as a literal permission
		permissionStr = FormatExactPermissionString(msg.Rule.Tool, msg.Rule.Parameters)
		logging.Info("📝 Adding exact permission to settings.local.json: %s", permissionStr)
	} else {
		// For pattern mode, format the pattern rule
		permissionStr = FormatPermissionString(msg.Rule.Tool, msg.Rule.Pattern)
		logging.Info("📝 Adding pattern permission to settings.local.json: %s", permissionStr)
	}

	// SECURITY: Never allow Bash(*) as a project permission (too dangerous)
	// This would allow any arbitrary command to run without approval
	if permissionStr == "Bash(*)" {
		logging.Error("❌ BLOCKED: Bash(*) is not allowed as a project permission (security risk)")
		return fmt.Errorf("Bash(*) is not allowed as a project permission. Please specify a command prefix like Bash(git:*) instead")
	}

	// Add to settings.local.json
	if err := settingsManager.AddPermission(permissionStr); err != nil {
		logging.Error("Failed to add permission to settings: %v", err)
		return fmt.Errorf("failed to add permission: %w", err)
	}

	// Generate ID and timestamp for response
	if msg.Rule.ID == "" {
		msg.Rule.ID = uuid.New().String()
	}
	msg.Rule.CreatedAt = time.Now()

	// Add rule to in-memory session for immediate effect
	h.SessionManager.mu.Lock()
	session.Options.AlwaysAllowRules = append(session.Options.AlwaysAllowRules, msg.Rule)
	totalRules := len(session.Options.AlwaysAllowRules)
	session.UpdatedAt = time.Now()
	h.SessionManager.mu.Unlock()

	logging.Info("✅ Rule added to session in-memory cache (total rules: %d)", totalRules)
	logging.Info("   Rule details: tool=%s, mode=%s, pattern=%v", msg.Rule.Tool, msg.Rule.MatchMode, msg.Rule.Pattern)

	// Persist updated session to database to preserve all options (including working_directory)
	if err := h.SessionManager.updateSessionInDB(&session.Session); err != nil {
		logging.Error("Failed to update session in database: %v", err)
		// Don't fail the request - the in-memory session is updated
	} else {
		logging.Info("💾 Session persisted to database with updated rules")
	}

	// IMPORTANT: If there's a pending permission request (from the UI that triggered this),
	// we need to approve it FIRST, let Claude process it, THEN reload to pick up the new rule
	if msg.PermissionID != "" {
		logging.Info("📤 Approving pending permission request: %s", msg.PermissionID)

		// Find the pending permission request
		session.permMu.Lock()
		responseChan, exists := session.pendingPermissions[msg.PermissionID]
		session.permMu.Unlock()

		if exists {
			// Send simple approval first (without the rule update)
			// This lets Claude continue with the current request
			select {
			case responseChan <- PermissionResponse{
				Approved:    true,
				DenyMessage: "",
			}:
				logging.Info("✅ Permission approved - Claude will continue processing")
				// Clean up the pending permission immediately after approval
				session.permMu.Lock()
				delete(session.pendingPermissions, msg.PermissionID)
				session.permMu.Unlock()

				// Set flag to reload after next message
				// This ensures Claude completes the current action before we reload
				session.pendingReloadMu.Lock()
				session.pendingReload = true
				session.pendingReloadMu.Unlock()
				logging.Info("📋 Marked session for reload after next message")

			case <-time.After(3 * time.Second):
				logging.Warning("⚠️ Timeout sending permission approval to SDK")
			}
		} else {
			logging.Warning("⚠️ No pending permission found for ID: %s", msg.PermissionID)
			// No pending permission, so just reload the session immediately
			logging.Info("🔄 Reloading session settings to apply new always-allow rule")
			if err := h.SessionManager.ReloadSessionSettings(msg.SessionID); err != nil {
				logging.Error("Failed to reload session settings: %v", err)
			} else {
				go func() {
					time.Sleep(200 * time.Millisecond)
					if err := h.SessionManager.SendPromptSilent(msg.SessionID, "continue"); err != nil {
						logging.Error("Failed to auto-continue session: %v", err)
					}
				}()
			}
		}
	} else {
		// No permission ID provided - just reload the session immediately
		logging.Info("🔄 Reloading session settings to apply new always-allow rule")
		if err := h.SessionManager.ReloadSessionSettings(msg.SessionID); err != nil {
			logging.Error("Failed to reload session settings: %v", err)
		} else {
			go func() {
				time.Sleep(200 * time.Millisecond)
				if err := h.SessionManager.SendPromptSilent(msg.SessionID, "continue"); err != nil {
					logging.Error("Failed to auto-continue session: %v", err)
				}
			}()
		}
	}

	// Send confirmation with the full rule (including generated ID)
	response := AlwaysAllowRulesListMessage{
		BaseMessage: BaseMessage{Type: MessageTypeAlwaysAllowRulesList},
		SessionID:   msg.SessionID,
		Rules:       session.Options.AlwaysAllowRules,
	}

	return h.safeWriteJSON(c, response)
}

func (h *AgentHandler) handleFiberRemoveAlwaysAllowRule(c *fiberws.Conn, rawMsg map[string]interface{}) error {
	var msg RemoveAlwaysAllowRuleMessage
	msgBytes, _ := json.Marshal(rawMsg)
	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		return fmt.Errorf("invalid remove_always_allow_rule message: %w", err)
	}

	// SECURITY: Validate session ownership before allowing rule removal
	authenticatedUser := h.GetConnectionUser(c)
	if err := h.validateSessionOwnership(msg.SessionID, authenticatedUser); err != nil {
		h.sendFiberError(c, err.Error())
		return err
	}

	logging.Info("Removing always-allow rule %s from session %s", msg.RuleID, msg.SessionID)

	// Get session
	session, err := h.SessionManager.GetSession(msg.SessionID)
	if err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	// Get working directory for this session
	workingDir := "."
	if session.Options.WorkingDirectory != nil {
		workingDir = *session.Options.WorkingDirectory
	}

	// Create settings manager
	settingsManager := NewClaudeSettingsManager(workingDir)

	// Find the rule to remove and format its permission string
	h.SessionManager.mu.Lock()
	var ruleToRemove *AlwaysAllowRule
	newRules := []AlwaysAllowRule{}
	for _, rule := range session.Options.AlwaysAllowRules {
		if rule.ID != msg.RuleID {
			newRules = append(newRules, rule)
		} else {
			ruleToRemove = &rule
		}
	}
	session.Options.AlwaysAllowRules = newRules
	session.UpdatedAt = time.Now()
	h.SessionManager.mu.Unlock()

	// Remove from settings.local.json
	if ruleToRemove != nil {
		permissionStr := FormatPermissionString(ruleToRemove.Tool, ruleToRemove.Pattern)
		logging.Info("🗑️ Removing permission from settings.local.json: %s", permissionStr)
		if err := settingsManager.RemovePermission(permissionStr); err != nil {
			logging.Error("Failed to remove permission from settings: %v", err)
		}
	}

	// Persist updated session to database to preserve all options (including working_directory)
	if err := h.SessionManager.updateSessionInDB(&session.Session); err != nil {
		logging.Error("Failed to update session in database: %v", err)
		// Don't fail the request - the in-memory session is updated
	} else {
		logging.Info("💾 Session persisted to database after removing rule")
	}

	// Send updated rules list
	response := AlwaysAllowRulesListMessage{
		BaseMessage: BaseMessage{Type: MessageTypeAlwaysAllowRulesList},
		SessionID:   msg.SessionID,
		Rules:       session.Options.AlwaysAllowRules,
	}

	return h.safeWriteJSON(c, response)
}

func (h *AgentHandler) handleFiberListAlwaysAllowRules(c *fiberws.Conn, rawMsg map[string]interface{}) error {
	var msg ListAlwaysAllowRulesMessage
	msgBytes, _ := json.Marshal(rawMsg)
	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		return fmt.Errorf("invalid list_always_allow_rules message: %w", err)
	}

	// SECURITY: Validate session ownership before allowing rule listing
	authenticatedUser := h.GetConnectionUser(c)
	if err := h.validateSessionOwnership(msg.SessionID, authenticatedUser); err != nil {
		h.sendFiberError(c, err.Error())
		return err
	}

	// Get session
	session, err := h.SessionManager.GetSession(msg.SessionID)
	if err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	// Send rules list
	response := AlwaysAllowRulesListMessage{
		BaseMessage: BaseMessage{Type: MessageTypeAlwaysAllowRulesList},
		SessionID:   msg.SessionID,
		Rules:       session.Options.AlwaysAllowRules,
	}

	return h.safeWriteJSON(c, response)
}

func (h *AgentHandler) handleFiberToggleYOLOMode(c *fiberws.Conn, rawMsg map[string]interface{}) error {
	var msg ToggleYOLOModeMessage
	msgBytes, _ := json.Marshal(rawMsg)
	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		return fmt.Errorf("invalid toggle_yolo_mode message: %w", err)
	}

	// SECURITY: Validate session ownership before allowing YOLO mode toggle
	// This prevents unauthorized users from disabling all permission checks on other users' sessions
	authenticatedUser := h.GetConnectionUser(c)
	if err := h.validateSessionOwnership(msg.SessionID, authenticatedUser); err != nil {
		h.sendFiberError(c, err.Error())
		return err
	}

	logging.Info("🎯 YOLO Mode toggle request: session=%s, enabled=%v", msg.SessionID, msg.Enabled)

	// Create new options with YOLO flags
	newOptions := SessionOptions{
		DangerouslySkipPermissions:      &msg.Enabled,
		AllowDangerouslySkipPermissions: &msg.Enabled,
	}

	// Reload session with new options (this interrupts and restarts with --resume)
	if err := h.SessionManager.ReloadSessionWithOptions(msg.SessionID, newOptions); err != nil {
		logging.Error("❌ Failed to toggle YOLO mode: %v", err)
		h.sendFiberError(c, fmt.Sprintf("Failed to toggle YOLO mode: %v", err))
		return err
	}

	logging.Info("✅ YOLO Mode toggled successfully")

	// Send confirmation to frontend
	statusText := "enabled"
	if !msg.Enabled {
		statusText = "disabled"
	}

	response := YOLOModeToggledMessage{
		BaseMessage: BaseMessage{Type: MessageTypeYOLOModeToggled},
		SessionID:   msg.SessionID,
		Enabled:     msg.Enabled,
		Message:     fmt.Sprintf("YOLO Mode %s - session reloaded with new permissions", statusText),
	}

	return h.safeWriteJSON(c, response)
}

