package agents

import (
	"encoding/json"
	"fmt"

	fiberws "github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"github.com/schlunsen/wee-editor/internal/logging"
)

// handleFiberListSessionSkills returns the skills currently enabled for a session
func (h *AgentHandler) handleFiberListSessionSkills(c *fiberws.Conn, rawMsg map[string]interface{}) error {
	sessionIDStr, ok := rawMsg["session_id"].(string)
	if !ok {
		return fmt.Errorf("missing session_id")
	}
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		return fmt.Errorf("invalid session_id: %w", err)
	}

	// Validate ownership
	authenticatedUser := h.GetConnectionUser(c)
	if err := h.validateSessionOwnership(sessionID, authenticatedUser); err != nil {
		h.sendFiberError(c, err.Error())
		return err
	}

	session, err := h.SessionManager.GetSession(sessionID)
	if err != nil {
		return err
	}

	// Fetch full skill details if injector available
	type SkillInfo struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
	}

	var skills []SkillInfo
	if h.SessionManager.skillInjector != nil && len(session.Options.EnabledSkillIDs) > 0 {
		_, loadedSkills, _ := h.SessionManager.skillInjector.BuildSkillContext(session.Options.EnabledSkillIDs)
		for _, s := range loadedSkills {
			skills = append(skills, SkillInfo{
				ID:          s.ID,
				Name:        s.Name,
				Description: s.Description,
			})
		}
	}

	response := map[string]interface{}{
		"type":       MessageTypeSessionSkillsList,
		"session_id": sessionID,
		"skills":     skills,
	}

	return h.safeWriteJSON(c, response)
}

// handleFiberAddSessionSkill adds a skill to a session's enabled skills
func (h *AgentHandler) handleFiberAddSessionSkill(c *fiberws.Conn, rawMsg map[string]interface{}) error {
	sessionIDStr, ok := rawMsg["session_id"].(string)
	if !ok {
		return fmt.Errorf("missing session_id")
	}
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		return fmt.Errorf("invalid session_id: %w", err)
	}

	skillID, ok := rawMsg["skill_id"].(string)
	if !ok {
		return fmt.Errorf("missing skill_id")
	}

	// Validate ownership
	authenticatedUser := h.GetConnectionUser(c)
	if err := h.validateSessionOwnership(sessionID, authenticatedUser); err != nil {
		h.sendFiberError(c, err.Error())
		return err
	}

	session, err := h.SessionManager.GetSession(sessionID)
	if err != nil {
		return err
	}

	// Check if skill is already enabled
	for _, id := range session.Options.EnabledSkillIDs {
		if id == skillID {
			return h.safeWriteJSON(c, map[string]interface{}{
				"type":       MessageTypeSessionSkillsUpdated,
				"session_id": sessionID,
				"action":     "already_enabled",
				"skill_id":   skillID,
			})
		}
	}

	// Add skill to session options (lock, re-fetch session, modify, unlock)
	h.SessionManager.mu.Lock()
	// Re-fetch session inside lock to avoid stale data
	session, err = h.SessionManager.getSessionLocked(sessionID)
	if err != nil {
		h.SessionManager.mu.Unlock()
		return err
	}
	session.Options.EnabledSkillIDs = append(session.Options.EnabledSkillIDs, skillID)
	optionsJSON, _ := json.Marshal(session.Options)
	h.SessionManager.mu.Unlock()

	// Persist updated options to database
	if err := h.SessionManager.Storage.UpdateSessionOptions(sessionID, string(optionsJSON)); err != nil {
		logging.Warning("Failed to persist skill addition for session %s: %v", sessionID, err)
	}

	logging.Info("Added skill %s to session %s", skillID, sessionID)

	// Respond with update
	return h.safeWriteJSON(c, map[string]interface{}{
		"type":       MessageTypeSessionSkillsUpdated,
		"session_id": sessionID,
		"action":     "added",
		"skill_id":   skillID,
		"skill_ids":  session.Options.EnabledSkillIDs,
	})
}

// handleFiberRemoveSessionSkill removes a skill from a session's enabled skills
func (h *AgentHandler) handleFiberRemoveSessionSkill(c *fiberws.Conn, rawMsg map[string]interface{}) error {
	sessionIDStr, ok := rawMsg["session_id"].(string)
	if !ok {
		return fmt.Errorf("missing session_id")
	}
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		return fmt.Errorf("invalid session_id: %w", err)
	}

	skillID, ok := rawMsg["skill_id"].(string)
	if !ok {
		return fmt.Errorf("missing skill_id")
	}

	// Validate ownership
	authenticatedUser := h.GetConnectionUser(c)
	if err := h.validateSessionOwnership(sessionID, authenticatedUser); err != nil {
		h.sendFiberError(c, err.Error())
		return err
	}

	session, err := h.SessionManager.GetSession(sessionID)
	if err != nil {
		return err
	}

	// Remove skill from session options (lock, re-fetch session, modify, unlock)
	h.SessionManager.mu.Lock()
	// Re-fetch session inside lock to avoid stale data
	session, err = h.SessionManager.getSessionLocked(sessionID)
	if err != nil {
		h.SessionManager.mu.Unlock()
		return err
	}
	newSkills := make([]string, 0, len(session.Options.EnabledSkillIDs))
	found := false
	for _, id := range session.Options.EnabledSkillIDs {
		if id == skillID {
			found = true
			continue
		}
		newSkills = append(newSkills, id)
	}
	session.Options.EnabledSkillIDs = newSkills
	removeOptionsJSON, _ := json.Marshal(session.Options)
	h.SessionManager.mu.Unlock()

	if !found {
		return h.safeWriteJSON(c, map[string]interface{}{
			"type":       MessageTypeSessionSkillsUpdated,
			"session_id": sessionID,
			"action":     "not_found",
			"skill_id":   skillID,
		})
	}

	// Persist updated options to database
	if err := h.SessionManager.Storage.UpdateSessionOptions(sessionID, string(removeOptionsJSON)); err != nil {
		logging.Warning("Failed to persist skill removal for session %s: %v", sessionID, err)
	}

	logging.Info("Removed skill %s from session %s", skillID, sessionID)

	// Respond with update
	return h.safeWriteJSON(c, map[string]interface{}{
		"type":       MessageTypeSessionSkillsUpdated,
		"session_id": sessionID,
		"action":     "removed",
		"skill_id":   skillID,
		"skill_ids":  session.Options.EnabledSkillIDs,
	})
}
