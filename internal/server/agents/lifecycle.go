package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/schlunsen/claude-agent-sdk-go/types"
	"github.com/schlunsen/wee-editor/internal/database"
	"github.com/schlunsen/wee-editor/internal/logging"
	"github.com/schlunsen/wee-editor/internal/providers"
)

// Session Lifecycle Management
// This file contains functions for creating, ending, and deleting sessions.

// isYOLOMode returns true if the permission mode means bypass all permissions.
func isYOLOMode(mode string) bool {
	m := strings.ToLower(strings.TrimSpace(mode))
	return m == "yolo" || m == "bypasspermissions"
}

// applyConfigDefaults fills in SessionOptions from server config when not explicitly set by the client.
func (sm *SessionManager) applyConfigDefaults(options *SessionOptions) {
	// Default provider
	if options.Provider == nil && sm.config.DefaultProvider != "" {
		options.Provider = &sm.config.DefaultProvider
	}

	// Default model
	if options.Model == nil && sm.config.DefaultModel != "" {
		options.Model = &sm.config.DefaultModel
	}

	// Default permission mode (YOLO)
	if options.DangerouslySkipPermissions == nil && isYOLOMode(sm.config.PermissionMode) {
		t := true
		options.DangerouslySkipPermissions = &t
		options.AllowDangerouslySkipPermissions = &t
	}
}

func (sm *SessionManager) CreateSession(sessionID uuid.UUID, options SessionOptions) (*Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	logging.Debug("CreateSession called for session: %s", sessionID)

	// Apply server config defaults for any options not explicitly set by the client
	sm.applyConfigDefaults(&options)

	// Check if session already exists in memory (from loadSessionsFromDB or previous create)
	if existingSession, exists := sm.sessions[sessionID]; exists {
		logging.Info("Session already exists in memory: %s - returning existing session", sessionID)
		// Return the existing session without error
		// This allows frontend reconnects after page refresh to work seamlessly
		return &existingSession.Session, nil
	}

	// Check if session exists in database (from previous server run or browser refresh)
	existingMeta, err := sm.Storage.GetSession(sessionID)
	if err == nil && existingMeta != nil {
		logging.Info("Restoring session from database: %s (Claude session: %s)", sessionID, existingMeta.ClaudeSessionID)

		// Restore options from database (if available), otherwise use request options
		restoredOptions := options // Start with request options as fallback
		if existingMeta.OptionsJSON != "" {
			var dbOptions SessionOptions
			if err := json.Unmarshal([]byte(existingMeta.OptionsJSON), &dbOptions); err == nil {
				logging.Debug("Restored options from database for session %s", sessionID)
				restoredOptions = dbOptions
			} else {
				logging.Warning("Failed to deserialize session options from DB for session %s: %v", sessionID, err)
			}
		}

		// Re-inject project area context if session has a project_area_id
		// This ensures the context prompt is always up-to-date even after restoration
		if existingMeta.ProjectAreaID != nil && *existingMeta.ProjectAreaID != "" {
			db := database.GetInstance()
			repo := database.NewRepository(db)
			if area, err := repo.GetProjectArea(*existingMeta.ProjectAreaID); err == nil {
				// Inject area context into system prompt if provided
				if area.ContextPrompt != "" {
					if restoredOptions.SystemPrompt == nil {
						restoredOptions.SystemPrompt = &area.ContextPrompt
						logging.Info("Injected project area context into restored session %s", sessionID)
					} else {
						// Check if context is already present (avoid duplication)
						if !strings.Contains(*restoredOptions.SystemPrompt, area.ContextPrompt) {
							// Prepend area context to existing system prompt
							combined := area.ContextPrompt + "\n\n" + *restoredOptions.SystemPrompt
							restoredOptions.SystemPrompt = &combined
							logging.Info("Prepended project area context to restored session %s", sessionID)
						} else {
							logging.Debug("Project area context already present in restored session %s", sessionID)
						}
					}
				}

				// Update working directory to area path if needed
				if project, err := repo.GetProject(area.ProjectID); err == nil {
					areaPath := filepath.Join(project.Path, area.RelativePath)
					if restoredOptions.WorkingDirectory == nil || *restoredOptions.WorkingDirectory != areaPath {
						restoredOptions.WorkingDirectory = &areaPath
						logging.Info("Updated working directory for restored session %s: %s", sessionID, areaPath)
					}
				}
			} else {
				logging.Warning("Failed to load project area %s for restored session %s: %v", *existingMeta.ProjectAreaID, sessionID, err)
			}
		}

		// CRITICAL FIX: Ensure the stored ModelName takes precedence over Options.Model
		// This prevents the TUI default model from overriding the restored session's model
		if existingMeta.ModelName != "" {
			if restoredOptions.Model == nil || *restoredOptions.Model != existingMeta.ModelName {
				logging.Info("Updating session options model to match stored model: %s -> %s",
					func() string {
						if restoredOptions.Model == nil {
							return "nil"
						}
						return *restoredOptions.Model
					}(), existingMeta.ModelName)
				restoredOptions.Model = &existingMeta.ModelName
			}
		}

		// Also ensure the provider is consistent with the stored model if not explicitly set
		if existingMeta.ModelName != "" && (restoredOptions.Provider == nil || *restoredOptions.Provider == "") {
			// Infer provider from stored model name
			var inferredProvider string
			if strings.HasPrefix(existingMeta.ModelName, "glm-") {
				inferredProvider = "glm"
			} else if strings.HasPrefix(existingMeta.ModelName, "claude-") {
				inferredProvider = "claude"
			} else if strings.Contains(existingMeta.ModelName, "gpt-") {
				inferredProvider = "openai"
			} else if strings.Contains(existingMeta.ModelName, "deepseek") {
				inferredProvider = "deepseek"
			}

			if inferredProvider != "" {
				logging.Info("Inferred provider from stored model: %s (model: %s)", inferredProvider, existingMeta.ModelName)
				restoredOptions.Provider = &inferredProvider
			}
		}

		// Detect git branch if working directory is provided
		gitBranch := existingMeta.GitBranch // Use stored value first
		if gitBranch == "" && restoredOptions.WorkingDirectory != nil && *restoredOptions.WorkingDirectory != "" {
			// If not stored, try to detect it now
			gitBranch = GetGitBranch(*restoredOptions.WorkingDirectory)
		}

		// Restore session to memory with data from database
		session := &AgentSession{
			Session: Session{
				ID:               existingMeta.ID,
				CreatedAt:        existingMeta.CreatedAt,
				UpdatedAt:        time.Now(), // Update to current time
				Status:           SessionStatus(existingMeta.Status),
				Options:          restoredOptions, // Use restored options from database
				MessageCount:     existingMeta.MessageCount,
				CostUSD:          existingMeta.CostUSD,
				NumTurns:         existingMeta.NumTurns,
				DurationMS:       existingMeta.DurationMS,
				ModelName:        existingMeta.ModelName,
				ClaudeSessionID:  existingMeta.ClaudeSessionID, // CRITICAL: Restore Claude session ID
				GitBranch:        gitBranch,
				ParentSessionID:  existingMeta.ParentSessionID,
				ContextSummary:   existingMeta.ContextSummary,
				Provider:         existingMeta.Provider,
				ProjectID:        existingMeta.ProjectID,
				ProjectAreaID:    existingMeta.ProjectAreaID,
				SelectedAvatarID: existingMeta.SelectedAvatarID,
			},
			active: true,
		}

		if existingMeta.ErrorMessage != "" {
			session.ErrorMessage = &existingMeta.ErrorMessage
		}

		// Create context for restored session
		session.ctx, session.cancel = context.WithCancel(context.Background())

		// Create response and permission channels
		session.responseChan = make(chan types.Message, 10)
		session.permissionReqChan = make(chan *PermissionRequest, 10)
		session.permissionRespChan = make(chan *PermissionResponse, 10)
		session.pendingPermissions = make(map[string]chan PermissionResponse)
		session.questionReqChan = make(chan *UserQuestionRequest, 10)
		session.pendingQuestions = make(map[string]chan UserQuestionAnswerResponse)
		session.pendingQuestionData = make(map[string]*UserQuestionMessage)
		session.subscribedProjects = make(map[string]bool)
		session.wsConnected = false // Will be set to true when WebSocket connects

		sm.sessions[sessionID] = session

		logging.Info("Session restored from database: %s (total sessions: %d)", sessionID, len(sm.sessions))
		return &session.Session, nil
	}

	// Session doesn't exist anywhere, create new one
	logging.Debug("Creating new session: %s", sessionID)
	now := time.Now()

	// Validate working directory if provided
	if options.WorkingDirectory != nil && *options.WorkingDirectory != "" {
		if err := ValidateWorkingDirectory(*options.WorkingDirectory); err != nil {
			return nil, fmt.Errorf("invalid working directory: %w", err)
		}
	}

	// Detect git branch if working directory is provided
	gitBranch := ""
	if options.WorkingDirectory != nil && *options.WorkingDirectory != "" {
		gitBranch = GetGitBranch(*options.WorkingDirectory)
	}

	// Determine model to use: session-specific > provider default > config default
	modelToUse := ""
	if options.Model != nil && *options.Model != "" {
		modelToUse = *options.Model
		logging.Info("DEBUG: Using session-specific model from options: %s", modelToUse)
	} else {
		// Try to get the current provider's default model
		if options.Provider != nil && *options.Provider != "" {
			if provider := providers.GetProviderByID(*options.Provider); provider != nil {
				modelToUse = provider.DefaultModel
				logging.Info("DEBUG: Using provider default model: %s (provider: %s)", modelToUse, *options.Provider)
			} else {
				logging.Warning("DEBUG: Provider not found: %s", *options.Provider)
			}
		} else {
			logging.Warning("DEBUG: No provider specified in options")
		}

		// Fall back to session manager config default
		if modelToUse == "" {
			modelToUse = sm.config.Model
			logging.Info("DEBUG: Using session manager default model: %s", modelToUse)
		}
	}

	// Determine provider (use session-specific > fallback)
	provider := ""
	if options.Provider != nil && *options.Provider != "" {
		provider = *options.Provider
		logging.Debug("Using session-specific provider: %s", provider)
	} else {
		// Infer provider from model if possible, or use fallback
		if strings.Contains(modelToUse, "glm-") {
			provider = "glm"
		} else if strings.Contains(strings.ToLower(modelToUse), "codex") {
			provider = CodexProviderID
		} else if strings.Contains(modelToUse, "claude-") {
			provider = "claude"
		} else if strings.Contains(modelToUse, "gpt-") {
			provider = "openai"
		} else if strings.Contains(modelToUse, "deepseek") {
			provider = "deepseek"
		} else {
			provider = "claude" // Safe fallback
		}
		logging.Debug("Inferred provider from model: %s -> %s", modelToUse, provider)
	}

	// Extract parent session ID and context summary from options
	var parentSessionID *uuid.UUID
	var contextSummary string
	if options.ParentSessionID != nil {
		parentSessionID = options.ParentSessionID
	}
	if options.ContextSummary != nil {
		contextSummary = *options.ContextSummary
	}

	// Extract avatar ID from options
	var selectedAvatarID *int64
	if options.SelectedAvatarID != nil {
		selectedAvatarID = options.SelectedAvatarID
	}

	// Auto-detect or use explicit project_id
	projectID := sm.detectOrCreateProject(&options)

	// Inject project default skills and system prompt if available
	if projectID != nil && *projectID != "" {
		db := database.GetInstance()
		repo := database.NewRepository(db)
		if project, err := repo.GetProject(*projectID); err == nil {
			// Inject default skills if no skills were explicitly provided
			if len(options.EnabledSkillIDs) == 0 && project.DefaultSkillIDs != "" {
				var defaultSkills []string
				if err := json.Unmarshal([]byte(project.DefaultSkillIDs), &defaultSkills); err == nil && len(defaultSkills) > 0 {
					options.EnabledSkillIDs = defaultSkills
					logging.Info("Injected %d default skills from project %s into session %s", len(defaultSkills), *projectID, sessionID)
				}
			}

			// Inject project system prompt (custom instructions)
			if project.SystemPrompt != "" {
				if options.SystemPrompt == nil {
					options.SystemPrompt = &project.SystemPrompt
				} else {
					// Prepend project instructions to existing system prompt
					combined := project.SystemPrompt + "\n\n" + *options.SystemPrompt
					options.SystemPrompt = &combined
				}
				logging.Info("Injected project system prompt from project %s into session %s", *projectID, sessionID)
			}
		}
	}

	// Handle project area ID (if provided)
	var projectAreaID *string
	if options.ProjectAreaID != nil && *options.ProjectAreaID != "" {
		projectAreaID = options.ProjectAreaID

		// If a project area is specified, update working directory to area path
		if projectAreaID != nil {
			// Get database instance and create repository to access project area data
			db := database.GetInstance()
			repo := database.NewRepository(db)
			if area, err := repo.GetProjectArea(*projectAreaID); err == nil {
				// Get project to construct full path
				if project, err := repo.GetProject(area.ProjectID); err == nil {
					// Update working directory to project path + area relative path
					areaPath := filepath.Join(project.Path, area.RelativePath)
					options.WorkingDirectory = &areaPath

					// Re-detect git branch for the area path
					gitBranch = GetGitBranch(areaPath)

					// Inject area context into system prompt if provided
					if area.ContextPrompt != "" {
						if options.SystemPrompt == nil {
							options.SystemPrompt = &area.ContextPrompt
						} else {
							// Prepend area context to existing system prompt
							combined := area.ContextPrompt + "\n\n" + *options.SystemPrompt
							options.SystemPrompt = &combined
						}
					}

					logging.Info("Session using project area: %s (path: %s)", area.Name, areaPath)
				}
			}
		}
	}

	// Handle git worktree creation if requested
	var worktreeID *string
	var worktreePath string
	if options.UseWorktree != nil && *options.UseWorktree && options.WorkingDirectory != nil && *options.WorkingDirectory != "" {
		wtBranch := ""
		if options.WorktreeBranch != nil && *options.WorktreeBranch != "" {
			wtBranch = *options.WorktreeBranch
		} else {
			// Generate a branch name from the session ID
			wtBranch = fmt.Sprintf("session/%s", sessionID.String()[:8])
		}

		repoDir := *options.WorkingDirectory
		shortID := sessionID.String()[:8]
		wtPath := GenerateWorktreePath(repoDir, shortID, wtBranch)

		createOpts := WorktreeCreateOptions{
			BranchName: wtBranch,
			NewBranch:  true, // Default to creating a new branch
		}

		// If worktree branch already exists, don't create a new one
		if options.WorktreeBranch != nil && *options.WorktreeBranch != "" {
			// Check if the branch exists
			checkCmd := fmt.Sprintf("git rev-parse --verify %s", wtBranch)
			_ = checkCmd // We'll try with NewBranch=true first, and fall back
		}

		if err := CreateWorktree(repoDir, wtPath, createOpts); err != nil {
			// Try again without NewBranch (branch might already exist)
			createOpts.NewBranch = false
			if err2 := CreateWorktree(repoDir, wtPath, createOpts); err2 != nil {
				logging.Warning("Failed to create worktree for session %s: %v (also tried existing branch: %v)", sessionID, err, err2)
				// Don't fail session creation, just skip worktree
			} else {
				// Success with existing branch
				options.WorkingDirectory = &wtPath
				gitBranch = GetGitBranch(wtPath)
				wtID := uuid.New().String()
				worktreeID = &wtID
				worktreePath = wtPath
				logging.Info("Created worktree for session %s at %s (existing branch: %s)", sessionID, wtPath, wtBranch)
			}
		} else {
			// Success with new branch
			options.WorkingDirectory = &wtPath
			gitBranch = GetGitBranch(wtPath)
			wtID := uuid.New().String()
			worktreeID = &wtID
			worktreePath = wtPath
			logging.Info("Created worktree for session %s at %s (new branch: %s)", sessionID, wtPath, wtBranch)
		}

		// Save worktree record to database
		if worktreeID != nil {
			db := database.GetInstance()
			repo := database.NewRepository(db)
			sessionIDStr := sessionID.String()

			var sourceBranch *string
			currentBranch := GetGitBranch(repoDir)
			if currentBranch != "" {
				sourceBranch = &currentBranch
			}

			wt := &database.Worktree{
				ID:            *worktreeID,
				ProjectID:     func() string { if projectID != nil { return *projectID }; return "" }(),
				SessionID:     &sessionIDStr,
				WorktreePath:  worktreePath,
				BranchName:    wtBranch,
				SourceBranch:  sourceBranch,
				IsAutoCreated: true,
				AutoCleanup:   true,
			}

			if err := repo.CreateWorktree(wt); err != nil {
				logging.Warning("Failed to save worktree record: %v", err)
			}
		}
	} else if options.WorktreeID != nil && *options.WorktreeID != "" {
		// Use an existing worktree
		db := database.GetInstance()
		repo := database.NewRepository(db)
		if wt, err := repo.GetWorktree(*options.WorktreeID); err == nil {
			options.WorkingDirectory = &wt.WorktreePath
			gitBranch = GetGitBranch(wt.WorktreePath)
			worktreeID = &wt.ID
			worktreePath = wt.WorktreePath

			// Update session association
			sessionIDStr := sessionID.String()
			_ = repo.Worktree.UpdateWorktreeSession(wt.ID, &sessionIDStr)

			logging.Info("Session %s using existing worktree %s at %s", sessionID, wt.ID, wt.WorktreePath)
		} else {
			logging.Warning("Worktree %s not found: %v", *options.WorktreeID, err)
		}
	}

	session := &AgentSession{
		Session: Session{
			ID:               sessionID,
			CreatedAt:        now,
			UpdatedAt:        now,
			Status:           SessionStatusIdle,
			Options:          options,
			MessageCount:     0,
			CostUSD:          0.0,
			NumTurns:         0,
			DurationMS:       0,
			ModelName:        modelToUse,
			GitBranch:        gitBranch,
			ParentSessionID:  parentSessionID,
			ContextSummary:   contextSummary,
			Provider:         provider,
			ProjectID:        projectID,
			ProjectAreaID:    projectAreaID,
			SelectedAvatarID: selectedAvatarID,
			WorktreeID:       worktreeID,
			WorktreePath:     worktreePath,
		},
		active: true,
	}

	// Create context for session
	session.ctx, session.cancel = context.WithCancel(context.Background())

	// Create response and permission channels
	session.responseChan = make(chan types.Message, 10)
	session.permissionReqChan = make(chan *PermissionRequest, 10)
	session.permissionRespChan = make(chan *PermissionResponse, 10)
	session.pendingPermissions = make(map[string]chan PermissionResponse)
	session.questionReqChan = make(chan *UserQuestionRequest, 10)
	session.pendingQuestions = make(map[string]chan UserQuestionAnswerResponse)
	session.pendingQuestionData = make(map[string]*UserQuestionMessage)
	session.subscribedProjects = make(map[string]bool)
	session.wsConnected = false // Will be set to true when WebSocket connects

	sm.sessions[sessionID] = session

	// Save to database
	if err := sm.saveSessionToDB(&session.Session); err != nil {
		logging.Error("Failed to save session to database: %v", err)
		// Don't fail the creation, just log the error
	}

	// Auto-enable connectors for the new session
	sm.autoEnableConnectors(sessionID.String(), options.Connectors)

	logging.Info("Session created: %s (total sessions: %d)", sessionID, len(sm.sessions))

	return &session.Session, nil
}

// autoEnableConnectors enables connectors for a newly created session.
// If specific connector slugs are provided, only those are enabled (if the user has active connections).
// If no slugs are provided (nil or empty), all active user connections are auto-enabled.
func (sm *SessionManager) autoEnableConnectors(sessionID string, connectorSlugs []string) {
	db := database.GetInstance()
	if db == nil {
		return
	}

	connectorRepo := database.NewConnectorRepository(db)

	var count int
	var err error

	if len(connectorSlugs) > 0 {
		// Enable only the specifically requested connectors
		count, err = connectorRepo.EnableSpecificConnectors(sessionID, connectorSlugs)
		if err != nil {
			logging.Warning("Failed to enable specific connectors for session %s: %v", sessionID, err)
			return
		}
		logging.Info("🔌 Enabled %d/%d requested connectors for session %s", count, len(connectorSlugs), sessionID)
	} else {
		// No specific connectors requested — auto-enable all active connections
		count, err = connectorRepo.EnableAllActiveConnectors(sessionID)
		if err != nil {
			logging.Warning("Failed to auto-enable connectors for session %s: %v", sessionID, err)
			return
		}
		if count > 0 {
			logging.Info("🔌 Auto-enabled %d active connectors for session %s", count, sessionID)
		}
	}
}

// NOTE: Git watching is now handled by ProjectWatcherManager and ProjectGitWatcher
// Sessions subscribe to project updates via subscribe_project WebSocket messages
// See agent_handler.go handleFiberSubscribeProject for the subscription flow

func (sm *SessionManager) InterruptSession(sessionID uuid.UUID) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	logging.Info("Interrupting session %s (status: %s)", sessionID, session.Status)

	// Loop mode: interrupting a session also stops any running autonomous loop so
	// it doesn't re-prompt after the user takes back control. We set the flags
	// directly (rather than via StopLoop) because sm.mu is already held here.
	//
	// loopStopped must be latched too. The interrupt itself produces a final
	// "Stopped." turn, and that turn's completion would otherwise be read as the
	// start of a brand new loop.
	session.loopMu.Lock()
	session.loopActive = false
	session.loopStopped = true
	session.loopMu.Unlock()

	// Close the streaming client BEFORE cancelling context
	// This ensures the client can clean up properly
	session.mu.Lock()
	if session.client != nil {
		logging.Info("Closing client for interrupted session %s", sessionID)
		// Use a background context for closing, not the about-to-be-cancelled session context
		closeCtx := context.Background()
		session.client.Close(closeCtx)
		session.client = nil
	}
	session.mu.Unlock()

	// Close cached MCP client — its context is about to be cancelled
	session.directMCPMu.Lock()
	if session.directMCPClient != nil {
		session.directMCPClient.Close()
		session.directMCPClient = nil
	}
	session.directMCPMu.Unlock()

	// Now cancel current context (this will interrupt any ongoing agent operations)
	if session.cancel != nil {
		session.cancel()
	}

	// CRITICAL FIX: Replace the response channel THEN close the old one.
	// After an interrupt, the old streamFiberResponses goroutine is still blocked
	// reading from responseChan (waiting for either a "result" message or channel close).
	// If we don't close the old channel, the old goroutine leaks and will steal
	// messages from the new streamFiberResponses goroutine on the next prompt,
	// since both goroutines would be reading from the same channel.
	//
	// Order matters: replace session.responseChan FIRST, then close the old one.
	// This ensures that if receiveQueryResponses enters its select after the replace,
	// it uses the new channel (safe). The old streamFiberResponses holds the old
	// channel by parameter, so closing it causes its `for msg := range` to exit.
	swapResponseChan(session)

	// Reset activeStreamerCount to 0 to ensure clean state for the next prompt.
	// The old streamFiberResponses goroutine's defer will try to decrement, but
	// we don't want a stale count to affect message routing (activeStreamerCount > 0
	// causes receiveQueryResponses to send to the channel instead of broadcasting directly).
	if old := atomic.SwapInt32(&session.activeStreamerCount, 0); old != 0 {
		logging.Warning("Session %s: Reset stale activeStreamerCount from %d to 0 during interrupt", session.ID, old)
	}

	// Create a new context for future operations
	session.ctx, session.cancel = context.WithCancel(context.Background())

	// Update status back to idle
	session.Status = SessionStatusIdle
	session.UpdatedAt = time.Now()

	// Update in database
	if err := sm.updateSessionInDB(&session.Session); err != nil {
		logging.Error("Failed to update interrupted session in database: %v", err)
	}

	// Save interruption message to database for conversation history
	// Only save once per processing cycle to prevent duplicates when user presses ESC multiple times
	session.interruptionMu.Lock()
	if !session.interruptionSaved {
		session.MessageCount++
		interruptionSequence := session.MessageCount
		if err := sm.saveMessageToDB(sessionID, interruptionSequence, "system", "⚠️ Session interrupted by user", "", nil, nil); err != nil {
			logging.Error("Failed to save interruption message to database: %v", err)
		} else {
			logging.Debug("Saved interruption message to database (sequence: %d)", interruptionSequence)
			session.interruptionSaved = true
		}
	} else {
		logging.Debug("Skipping duplicate interruption message (already saved for this cycle)")
	}
	session.interruptionMu.Unlock()

	logging.Info("Session interrupted successfully: %s (client closed, context refreshed)", sessionID)
	return nil
}

func (sm *SessionManager) EndSession(sessionID uuid.UUID) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Close streaming client if exists
	session.mu.Lock()
	if session.client != nil {
		session.client.Close(session.ctx)
		session.client = nil
	}
	session.mu.Unlock()

	// Close cached MCP client for direct provider sessions
	session.directMCPMu.Lock()
	if session.directMCPClient != nil {
		session.directMCPClient.Close()
		session.directMCPClient = nil
	}
	session.directMCPMu.Unlock()

	// Cancel context (will stop any ongoing queries)
	if session.cancel != nil {
		session.cancel()
	}

	// Update status
	session.Status = SessionStatusEnded
	session.UpdatedAt = time.Now()
	session.active = false

	// Calculate duration
	session.DurationMS = time.Since(session.CreatedAt).Milliseconds()

	// Update in database
	if err := sm.updateSessionInDB(&session.Session); err != nil {
		logging.Error("Failed to update ended session in database: %v", err)
	}

	// Clean up worktree if auto-cleanup is enabled
	if session.WorktreeID != nil && session.WorktreePath != "" {
		sm.cleanupSessionWorktree(session)
	}

	// Remove from active sessions map
	delete(sm.sessions, sessionID)

	logging.Info("Session ended: %s (duration: %dms, messages: %d)",
		sessionID, session.DurationMS, session.MessageCount)

	return nil
}

func (sm *SessionManager) DeleteSession(sessionID uuid.UUID) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// If session is still active, end it first
	if session, exists := sm.sessions[sessionID]; exists {
		// Close streaming client if exists
		session.mu.Lock()
		if session.client != nil {
			session.client.Close(session.ctx)
			session.client = nil
		}
		session.mu.Unlock()

		// Close cached MCP client for direct provider sessions
		session.directMCPMu.Lock()
		if session.directMCPClient != nil {
			session.directMCPClient.Close()
			session.directMCPClient = nil
		}
		session.directMCPMu.Unlock()

		// Cancel context
		if session.cancel != nil {
			session.cancel()
		}

		// Remove from active sessions
		delete(sm.sessions, sessionID)
	}

	// Delete from database
	if err := sm.Storage.DeleteSession(sessionID); err != nil {
		return fmt.Errorf("failed to delete session from storage: %w", err)
	}

	logging.Info("Session deleted: %s", sessionID)
	return nil
}

// cleanupSessionWorktree removes the worktree associated with a session if auto-cleanup is enabled
func (sm *SessionManager) cleanupSessionWorktree(session *AgentSession) {
	if session.WorktreeID == nil || session.WorktreePath == "" {
		return
	}

	db := database.GetInstance()
	repo := database.NewRepository(db)

	wt, err := repo.GetWorktree(*session.WorktreeID)
	if err != nil {
		logging.Warning("Failed to get worktree %s for cleanup: %v", *session.WorktreeID, err)
		return
	}

	if !wt.AutoCleanup {
		logging.Info("Worktree %s has auto_cleanup=false, skipping cleanup", wt.ID)
		return
	}

	// Get the main repo dir to run git worktree remove
	mainRepoDir, err := GetMainWorktreePath(wt.WorktreePath)
	if err != nil {
		logging.Warning("Failed to find main repo for worktree %s: %v", wt.WorktreePath, err)
		// Try using the project path instead
		if wt.ProjectID != "" {
			if project, err := repo.GetProject(wt.ProjectID); err == nil {
				mainRepoDir = project.Path
			}
		}
		if mainRepoDir == "" {
			return
		}
	}

	// Remove the worktree (force to handle any uncommitted changes)
	if err := RemoveWorktree(mainRepoDir, wt.WorktreePath, true); err != nil {
		logging.Warning("Failed to remove worktree %s: %v", wt.WorktreePath, err)
	} else {
		logging.Info("Cleaned up worktree %s for ended session %s", wt.WorktreePath, session.ID)
	}

	// Mark as removed in database
	if err := repo.MarkWorktreeRemoved(wt.ID); err != nil {
		logging.Warning("Failed to mark worktree %s as removed: %v", wt.ID, err)
	}
}

func (sm *SessionManager) EndAllSessions(projectID *string) int {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	count := 0
	for sessionID, session := range sm.sessions {
		// Skip if project filter is specified and doesn't match
		if projectID != nil && *projectID != "" {
			session.mu.Lock()
			sessionProjectID := session.Session.ProjectID
			session.mu.Unlock()

			if sessionProjectID == nil || *sessionProjectID != *projectID {
				continue
			}
		}

		// Close cached MCP client for direct provider sessions
		session.directMCPMu.Lock()
		if session.directMCPClient != nil {
			session.directMCPClient.Close()
			session.directMCPClient = nil
		}
		session.directMCPMu.Unlock()

		if session.cancel != nil {
			session.cancel()
		}
		delete(sm.sessions, sessionID)
		count++
	}

	return count
}

func (sm *SessionManager) DeleteAllSessions(projectID *string) (int, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// First, end active sessions (with project filter if specified)
	for sessionID, session := range sm.sessions {
		// Skip if project filter is specified and doesn't match
		if projectID != nil && *projectID != "" {
			session.mu.Lock()
			sessionProjectID := session.Session.ProjectID
			session.mu.Unlock()

			if sessionProjectID == nil || *sessionProjectID != *projectID {
				continue
			}
		}

		// Close streaming client if exists
		session.mu.Lock()
		if session.client != nil {
			session.client.Close(session.ctx)
			session.client = nil
		}
		session.mu.Unlock()

		// Close cached MCP client for direct provider sessions
		session.directMCPMu.Lock()
		if session.directMCPClient != nil {
			session.directMCPClient.Close()
			session.directMCPClient = nil
		}
		session.directMCPMu.Unlock()

		// Cancel context
		if session.cancel != nil {
			session.cancel()
		}

		// Remove from active sessions
		delete(sm.sessions, sessionID)
	}

	// Get all sessions from database
	allSessions, err := sm.Storage.ListSessions("all")
	if err != nil {
		return 0, fmt.Errorf("failed to list sessions: %w", err)
	}

	count := 0
	// Delete each session from database (with project filter if specified)
	for _, session := range allSessions {
		// Skip if project filter is specified and doesn't match
		if projectID != nil && *projectID != "" {
			if session.ProjectID == nil || *session.ProjectID != *projectID {
				continue
			}
		}

		if err := sm.Storage.DeleteSession(session.ID); err != nil {
			logging.Error("Failed to delete session %s: %v", session.ID, err)
			continue
		}
		count++
	}

	if projectID != nil && *projectID != "" {
		logging.Info("Deleted %d sessions for project %s from database", count, *projectID)
	} else {
		logging.Info("Deleted %d sessions from database", count)
	}
	return count, nil
}
