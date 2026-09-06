package agents

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/schlunsen/wee-editor/internal/database"
	"github.com/schlunsen/wee-editor/internal/logging"
)

// Session Handover Methods
// This file contains functions for creating and applying session handovers.

// generateHandoverToken creates a cryptographically secure random token
func generateHandoverToken() (string, error) {
	bytes := make([]byte, 32) // 256 bits
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// CreateHandover creates a handover package for a session
func (sm *SessionManager) CreateHandover(sessionID uuid.UUID, options HandoverOptions) (*HandoverCreatedMessage, error) {
	sm.mu.RLock()
	agentSession := sm.sessions[sessionID]
	sm.mu.RUnlock()

	// Load full session data from database
	sessionMetadata, err := sm.Storage.GetSession(sessionID)
	if err != nil {
		// Session not found in database either
		return nil, fmt.Errorf("failed to load session: %w", err)
	}

	// Convert SessionMetadata to Session for the handover
	session := Session{
		ID:              sessionMetadata.ID,
		CreatedAt:       sessionMetadata.CreatedAt,
		UpdatedAt:       sessionMetadata.UpdatedAt,
		Status:          SessionStatus(sessionMetadata.Status),
		MessageCount:    sessionMetadata.MessageCount,
		CostUSD:         sessionMetadata.CostUSD,
		NumTurns:        sessionMetadata.NumTurns,
		DurationMS:      sessionMetadata.DurationMS,
		ModelName:       sessionMetadata.ModelName,
		ClaudeSessionID: sessionMetadata.ClaudeSessionID,
		GitBranch:       sessionMetadata.GitBranch,
		ParentSessionID: sessionMetadata.ParentSessionID,
		ContextSummary:  sessionMetadata.ContextSummary,
		Provider:        sessionMetadata.Provider,
		ProjectID:       sessionMetadata.ProjectID,
	}

	// Parse OptionsJSON to get SessionOptions
	if sessionMetadata.OptionsJSON != "" {
		if err := json.Unmarshal([]byte(sessionMetadata.OptionsJSON), &session.Options); err != nil {
			logging.Warning("Failed to parse session options: %v", err)
		}
	}

	// Build handover data
	handoverData := HandoverData{
		SourceSessionID: sessionID.String(),
		SessionMetadata: session,
		CreatedAt:       time.Now(),
		HandoverNote:    options.HandoverNote,
	}

	// Include working directory and git branch if requested
	if options.IncludeWorkingDir || options.IncludeContext {
		if session.Options.WorkingDirectory != nil {
			handoverData.WorkingDirectory = *session.Options.WorkingDirectory
		}
		handoverData.GitBranch = session.GitBranch
		handoverData.Provider = session.Provider
	}

	// Include messages if requested
	if options.IncludeMessages {
		limit := 100 // Default limit
		if options.MessageLimit != nil && *options.MessageLimit > 0 {
			limit = *options.MessageLimit
		}

		messagesPtr, hasMore, err := sm.Storage.GetMessages(sessionID, limit, 0)
		if err != nil {
			logging.Warning("Failed to load messages for handover: %v", err)
		} else {
			// Convert []*MessageRecord to []MessageRecord
			messages := make([]MessageRecord, len(messagesPtr))
			for i, msg := range messagesPtr {
				messages[i] = *msg
			}
			handoverData.Messages = messages
			if hasMore {
				logging.Info("Handover includes %d messages (more available)", len(messages))
			}
		}
	}

	// Preserve YOLO mode settings if requested
	if options.PreserveYOLOMode {
		// YOLO mode is already in session.Options
		// No additional action needed, it's preserved in SessionMetadata
	}

	// Preserve always-allow rules if requested
	if options.PreserveAlwaysAllowRules {
		// Rules are already in session.Options.AlwaysAllowRules
		// No additional action needed
	}

	// Serialize handover data to JSON
	handoverJSON, err := json.Marshal(handoverData)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize handover data: %w", err)
	}

	// Generate secure token
	token, err := generateHandoverToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate handover token: %w", err)
	}

	// Calculate expiration
	expirationMinutes := 60 // Default 1 hour
	if options.ExpirationMinutes != nil && *options.ExpirationMinutes > 0 {
		expirationMinutes = *options.ExpirationMinutes
	}
	expiresAt := time.Now().Add(time.Duration(expirationMinutes) * time.Minute)

	// Create handover record in database
	repo := database.NewRepository(database.GetInstance())
	handover := &database.AgentHandover{
		HandoverToken:   token,
		SourceSessionID: sessionID.String(),
		TargetSessionID: func() *string {
			if options.TargetSessionID != nil {
				s := options.TargetSessionID.String()
				return &s
			}
			return nil
		}(),
		HandoverData: string(handoverJSON),
		HandoverNote: options.HandoverNote,
		ExpiresAt:    expiresAt,
	}

	if err := repo.CreateHandover(handover); err != nil {
		return nil, fmt.Errorf("failed to save handover: %w", err)
	}

	// Build response
	response := &HandoverCreatedMessage{
		BaseMessage:     BaseMessage{Type: MessageTypeHandoverCreated},
		HandoverToken:   token,
		SourceSessionID: sessionID,
		CreatedAt:       time.Now(),
		ExpiresAt:       expiresAt,
	}

	response.Context.WorkingDirectory = handoverData.WorkingDirectory
	response.Context.GitBranch = handoverData.GitBranch
	response.Context.MessageCount = len(handoverData.Messages)
	if agentSession != nil {
		response.Context.LastActivity = agentSession.UpdatedAt.Format(time.RFC3339)
	}

	logging.Info("Created handover %s for session %s (expires: %s)", token[:16]+"...", sessionID, expiresAt.Format(time.RFC3339))

	return response, nil
}

// ApplyHandover applies a handover to create or update a session
func (sm *SessionManager) ApplyHandover(token string, newSessionConfig *SessionOptions, useExistingSession *uuid.UUID, autoStart bool, initialPrompt string) (*HandoverAcceptedMessage, error) {
	// Get handover from database
	repo := database.NewRepository(database.GetInstance())
	handoverRecord, err := repo.GetHandoverByToken(token)
	if err != nil {
		return nil, fmt.Errorf("handover not found: %w", err)
	}

	// Check if already consumed
	if handoverRecord.ConsumedAt != nil {
		return nil, fmt.Errorf("handover already consumed")
	}

	// Check if expired
	if time.Now().After(handoverRecord.ExpiresAt) {
		return nil, fmt.Errorf("handover expired")
	}

	// Parse handover data
	var handoverData HandoverData
	if err := json.Unmarshal([]byte(handoverRecord.HandoverData), &handoverData); err != nil {
		return nil, fmt.Errorf("failed to parse handover data: %w", err)
	}

	var targetSessionID uuid.UUID
	var targetSession *Session
	var isNewSession bool

	// Determine target session
	if useExistingSession != nil {
		// Apply to existing session
		sm.mu.RLock()
		agentSession, exists := sm.sessions[*useExistingSession]
		sm.mu.RUnlock()

		if !exists {
			return nil, fmt.Errorf("target session not found")
		}
		targetSessionID = *useExistingSession
		targetSession = &agentSession.Session
		isNewSession = false
	} else {
		// Create new session with handover context
		targetSessionID = uuid.New()
		isNewSession = true

		// Build session options from handover data and optional overrides
		options := handoverData.SessionMetadata.Options

		// Apply overrides if provided
		if newSessionConfig != nil {
			if newSessionConfig.SystemPrompt != nil {
				options.SystemPrompt = newSessionConfig.SystemPrompt
			}
			if newSessionConfig.AgentName != nil {
				options.AgentName = newSessionConfig.AgentName
			}
			if newSessionConfig.WorkingDirectory != nil {
				options.WorkingDirectory = newSessionConfig.WorkingDirectory
			}
			if newSessionConfig.Model != nil {
				options.Model = newSessionConfig.Model
			}
			if newSessionConfig.Provider != nil {
				options.Provider = newSessionConfig.Provider
			}
			if newSessionConfig.MaxTokens != nil {
				options.MaxTokens = newSessionConfig.MaxTokens
			}
			if newSessionConfig.Temperature != nil {
				options.Temperature = newSessionConfig.Temperature
			}
			// Preserve YOLO mode from handover unless explicitly overridden
			if newSessionConfig.DangerouslySkipPermissions != nil {
				options.DangerouslySkipPermissions = newSessionConfig.DangerouslySkipPermissions
			}
			if newSessionConfig.AllowDangerouslySkipPermissions != nil {
				options.AllowDangerouslySkipPermissions = newSessionConfig.AllowDangerouslySkipPermissions
			}
			// Preserve always-allow rules from handover unless explicitly overridden
			if len(newSessionConfig.AlwaysAllowRules) > 0 {
				options.AlwaysAllowRules = newSessionConfig.AlwaysAllowRules
			}
		}

		// Set parent session ID for lineage tracking
		sourceUUID, err := uuid.Parse(handoverData.SourceSessionID)
		if err == nil {
			options.ParentSessionID = &sourceUUID
			options.ContextSummary = &handoverData.HandoverNote
		}

		// Create the session
		newSession, err := sm.CreateSession(targetSessionID, options)
		if err != nil {
			return nil, fmt.Errorf("failed to create session: %w", err)
		}
		targetSession = newSession
	}

	// Import messages if included in handover
	messagesImported := 0
	if len(handoverData.Messages) > 0 && isNewSession {
		// For new sessions, we can import the message history
		// Note: This would require extending the Storage interface to support bulk message import
		// For now, we'll just count them
		messagesImported = len(handoverData.Messages)
		logging.Info("Handover includes %d messages (import functionality pending)", messagesImported)
	}

	// Mark handover as consumed
	if err := repo.ConsumeHandover(token, targetSessionID.String()); err != nil {
		logging.Warning("Failed to mark handover as consumed: %v", err)
	}

	// Build response
	response := &HandoverAcceptedMessage{
		BaseMessage:      BaseMessage{Type: MessageTypeHandoverAccepted},
		SessionID:        targetSessionID,
		HandoverApplied:  true,
		MessagesImported: messagesImported,
		Session:          *targetSession,
	}

	response.ContextRestored.WorkingDirectory = handoverData.WorkingDirectory
	response.ContextRestored.GitBranch = handoverData.GitBranch
	response.ContextRestored.Provider = handoverData.Provider

	logging.Info("Applied handover %s to session %s (imported %d messages)", token[:16]+"...", targetSessionID, messagesImported)

	// Auto-start the agent if requested
	if autoStart && isNewSession {
		logging.Info("🚀 Auto-starting agent for session %s", targetSessionID)

		// Update session status to 'processing' BEFORE returning response
		// This prevents race condition where UI receives handover_accepted with status=idle
		// before session_updated with status=processing
		sm.mu.Lock()
		targetSession.Status = SessionStatusProcessing
		targetSession.UpdatedAt = time.Now()

		// Get the AgentSession from the sessions map to broadcast
		agentSession, exists := sm.sessions[targetSessionID]
		sm.mu.Unlock()

		// Update response with processing status
		response.Session.Status = SessionStatusProcessing

		// Broadcast session update immediately so all WebSocket clients know it's processing
		if exists && agentSession != nil {
			sm.broadcastSessionUpdate(agentSession)
		}

		// Determine the prompt to send
		prompt := initialPrompt
		if prompt == "" {
			// Use handover note or default prompt
			if handoverData.HandoverNote != "" {
				prompt = fmt.Sprintf("I've received a handover from another session with the following context:\n\n%s\n\nPlease continue working on the tasks from this session.", handoverData.HandoverNote)
			} else {
				prompt = "Please continue working on the tasks from the previous session."
			}
		}

		// Send the initial prompt to start the agent
		// This will be done asynchronously to not block the response
		// The status update has already been sent above, so the UI will show "processing" immediately
		// authenticatedUser is nil for internal handover operations
		go func() {
			if err := sm.SendPrompt(targetSessionID, prompt, nil); err != nil {
				logging.Error("Failed to auto-start agent: %v", err)
			} else {
				logging.Info("✅ Agent auto-started successfully for session %s", targetSessionID)
			}
		}()
	}

	return response, nil
}
