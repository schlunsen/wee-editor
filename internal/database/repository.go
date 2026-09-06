// Package database provides data access methods for querying and persisting data.
// This file implements the Repository pattern as a composition of specialized repositories
// for different domains: commands, settings, projects, handovers, site generation, terminals,
// MFA, users, and authentication auditing.
package database

// Repository provides composed access to all specialized data repositories
type Repository struct {
	db *Database

	// Command History Repositories
	ShellCommand   *ShellCommandRepository
	ClaudeCommand  *ClaudeCommandRepository
	CommandStats   *CommandStatsRepository

	// Configuration Repositories
	Provider  *ProviderRepository
	Connector *ConnectorRepository
	Settings  *SettingsRepository

	// Project Management Repositories
	Project *ProjectRepository

	// Agent Handover Repository
	Handover *HandoverRepository

	// Site Generation Repositories
	SiteProject *SiteProjectRepository

	// Terminal Repository
	Terminal *TerminalRepository

	// Authentication & User Repositories
	MFA       *MFARepository
	User      *UserRepository
	AuthAudit *AuthAuditRepository
	Session   *SessionRepository

	// History Management Repository
	History *HistoryRepository

	// Git Worktree Repository
	Worktree *WorktreeRepository

	// Skills & Hooks Repositories
	Skill *SkillRepository
	Hook  *HookRepository

	// Memory Palace Repository
	Memory *MemoryRepository
}

// NewRepository creates a new repository instance with all specialized sub-repositories
func NewRepository(db *Database) *Repository {
	return &Repository{
		db: db,

		// Initialize command history repositories
		ShellCommand:  NewShellCommandRepository(db),
		ClaudeCommand: NewClaudeCommandRepository(db),
		CommandStats:  NewCommandStatsRepository(db),

		// Initialize configuration repositories
		Provider:  NewProviderRepository(db),
		Connector: NewConnectorRepository(db),
		Settings:  NewSettingsRepository(db),

		// Initialize project management repositories
		Project: NewProjectRepository(db),

		// Initialize handover repository
		Handover: NewHandoverRepository(db),

		// Initialize site generation repositories
		SiteProject: NewSiteProjectRepository(db),

		// Initialize terminal repository
		Terminal: NewTerminalRepository(db),

		// Initialize authentication & user repositories
		MFA:       NewMFARepository(db),
		User:      NewUserRepository(db),
		AuthAudit: NewAuthAuditRepository(db),
		Session:   NewSessionRepository(db),

		// Initialize history repository
		History: NewHistoryRepository(db),

		// Initialize worktree repository
		Worktree: NewWorktreeRepository(db),

		// Initialize skills & hooks repositories
		Skill: NewSkillRepository(db),
		Hook:  NewHookRepository(db),

		// Initialize Memory Palace repository
		Memory: NewMemoryRepository(db),
	}
}

// Deprecated: Direct method access on Repository. Use the specialized sub-repositories instead.
// These methods are kept for backward compatibility during migration.

// RecordShellCommand is deprecated. Use repo.ShellCommand.RecordShellCommand instead.
func (r *Repository) RecordShellCommand(cmd *ShellCommand) error {
	return r.ShellCommand.RecordShellCommand(cmd)
}

// GetShellCommands is deprecated. Use repo.ShellCommand.GetShellCommands instead.
func (r *Repository) GetShellCommands(query *CommandHistoryQuery) ([]*ShellCommand, error) {
	return r.ShellCommand.GetShellCommands(query)
}

// DeleteAllShellCommands is deprecated. Use repo.ShellCommand.DeleteAllShellCommands instead.
func (r *Repository) DeleteAllShellCommands() error {
	return r.ShellCommand.DeleteAllShellCommands()
}

// RecordClaudeCommand is deprecated. Use repo.ClaudeCommand.RecordClaudeCommand instead.
func (r *Repository) RecordClaudeCommand(cmd *ClaudeCommand) error {
	return r.ClaudeCommand.RecordClaudeCommand(cmd)
}

// GetClaudeCommands is deprecated. Use repo.ClaudeCommand.GetClaudeCommands instead.
func (r *Repository) GetClaudeCommands(query *CommandHistoryQuery) ([]*ClaudeCommand, error) {
	return r.ClaudeCommand.GetClaudeCommands(query)
}

// DeleteAllClaudeCommands is deprecated. Use repo.ClaudeCommand.DeleteAllClaudeCommands instead.
func (r *Repository) DeleteAllClaudeCommands() error {
	return r.ClaudeCommand.DeleteAllClaudeCommands()
}

// GetCommandStats is deprecated. Use repo.CommandStats.GetCommandStats instead.
func (r *Repository) GetCommandStats(commandType string, limit int) ([]*CommandStat, error) {
	return r.CommandStats.GetCommandStats(commandType, limit)
}

// SaveProvider is deprecated. Use repo.Provider.SaveProvider instead.
func (r *Repository) SaveProvider(provider *ProviderConfig) error {
	return r.Provider.SaveProvider(provider)
}

// GetProvider is deprecated. Use repo.Provider.GetProvider instead.
func (r *Repository) GetProvider(providerID string) (*ProviderConfig, error) {
	return r.Provider.GetProvider(providerID)
}

// GetCurrentProvider is deprecated. Use repo.Provider.GetCurrentProvider instead.
func (r *Repository) GetCurrentProvider() (*ProviderConfig, error) {
	return r.Provider.GetCurrentProvider()
}

// GetAllProviders is deprecated. Use repo.Provider.GetAllProviders instead.
func (r *Repository) GetAllProviders() ([]*ProviderConfig, error) {
	return r.Provider.GetAllProviders()
}

// ListProviders retrieves all providers from database
func (r *Repository) ListProviders() ([]*ProviderConfig, error) {
	return r.Provider.ListProviders()
}

// DeleteProvider is deprecated. Use repo.Provider.DeleteProvider instead.
func (r *Repository) DeleteProvider(providerID string) error {
	return r.Provider.DeleteProvider(providerID)
}

// DeleteAllProviders is deprecated. Use repo.Provider.DeleteAllProviders instead.
func (r *Repository) DeleteAllProviders() error {
	return r.Provider.DeleteAllProviders()
}

// UpdateProviderModels updates only model metadata for a provider (preserves user config).
func (r *Repository) UpdateProviderModels(providerID string, modelsJSON string, defaultModel string, name string, description string) error {
	return r.Provider.UpdateProviderModels(providerID, modelsJSON, defaultModel, name, description)
}

// GetUserSetting is deprecated. Use repo.Settings.GetUserSetting instead.
func (r *Repository) GetUserSetting(key string) (*UserSetting, error) {
	return r.Settings.GetUserSetting(key)
}

// GetAllUserSettings is deprecated. Use repo.Settings.GetAllUserSettings instead.
func (r *Repository) GetAllUserSettings() ([]UserSetting, error) {
	return r.Settings.GetAllUserSettings()
}

// SetUserSetting is deprecated. Use repo.Settings.SetUserSetting instead.
func (r *Repository) SetUserSetting(setting *UserSetting) error {
	return r.Settings.SetUserSetting(setting)
}

// DeleteUserSetting is deprecated. Use repo.Settings.DeleteUserSetting instead.
func (r *Repository) DeleteUserSetting(key string) error {
	return r.Settings.DeleteUserSetting(key)
}

// CreateProject is deprecated. Use repo.Project.CreateProject instead.
func (r *Repository) CreateProject(project *Project) error {
	return r.Project.CreateProject(project)
}

// GetProject is deprecated. Use repo.Project.GetProject instead.
func (r *Repository) GetProject(id string) (*Project, error) {
	return r.Project.GetProject(id)
}

// GetProjectByPath is deprecated. Use repo.Project.GetProjectByPath instead.
func (r *Repository) GetProjectByPath(path string) (*Project, error) {
	return r.Project.GetProjectByPath(path)
}

// GetAllProjects is deprecated. Use repo.Project.GetAllProjects instead.
func (r *Repository) GetAllProjects() ([]*Project, error) {
	return r.Project.GetAllProjects()
}

// GetActiveProjects is deprecated. Use repo.Project.GetActiveProjects instead.
func (r *Repository) GetActiveProjects() ([]*Project, error) {
	return r.Project.GetActiveProjects()
}

// UpdateProject is deprecated. Use repo.Project.UpdateProject instead.
func (r *Repository) UpdateProject(project *Project) error {
	return r.Project.UpdateProject(project)
}

// DeleteProject is deprecated. Use repo.Project.DeleteProject instead.
func (r *Repository) DeleteProject(id string) error {
	return r.Project.DeleteProject(id)
}

// GetProjectStats is deprecated. Use repo.Project.GetProjectStats instead.
func (r *Repository) GetProjectStats(projectID string) (*ProjectStats, error) {
	return r.Project.GetProjectStats(projectID)
}

// CreateProjectArea is deprecated. Use repo.Project.CreateProjectArea instead.
func (r *Repository) CreateProjectArea(area *ProjectArea) error {
	return r.Project.CreateProjectArea(area)
}

// GetProjectArea is deprecated. Use repo.Project.GetProjectArea instead.
func (r *Repository) GetProjectArea(areaID string) (*ProjectArea, error) {
	return r.Project.GetProjectArea(areaID)
}

// GetProjectAreas is deprecated. Use repo.Project.GetProjectAreas instead.
func (r *Repository) GetProjectAreas(projectID string) ([]*ProjectArea, error) {
	return r.Project.GetProjectAreas(projectID)
}

// UpdateProjectArea is deprecated. Use repo.Project.UpdateProjectArea instead.
func (r *Repository) UpdateProjectArea(area *ProjectArea) error {
	return r.Project.UpdateProjectArea(area)
}

// DeleteProjectArea is deprecated. Use repo.Project.DeleteProjectArea instead.
func (r *Repository) DeleteProjectArea(areaID string) error {
	return r.Project.DeleteProjectArea(areaID)
}

// CreateHandover is deprecated. Use repo.Handover.CreateHandover instead.
func (r *Repository) CreateHandover(handover *AgentHandover) error {
	return r.Handover.CreateHandover(handover)
}

// GetHandoverByToken is deprecated. Use repo.Handover.GetHandoverByToken instead.
func (r *Repository) GetHandoverByToken(token string) (*AgentHandover, error) {
	return r.Handover.GetHandoverByToken(token)
}

// ConsumeHandover is deprecated. Use repo.Handover.ConsumeHandover instead.
func (r *Repository) ConsumeHandover(token, consumedBySessionID string) error {
	return r.Handover.ConsumeHandover(token, consumedBySessionID)
}

// CleanupExpiredHandovers is deprecated. Use repo.Handover.CleanupExpiredHandovers instead.
func (r *Repository) CleanupExpiredHandovers() (int64, error) {
	return r.Handover.CleanupExpiredHandovers()
}

// GetHandoversBySourceSession is deprecated. Use repo.Handover.GetHandoversBySourceSession instead.
func (r *Repository) GetHandoversBySourceSession(sourceSessionID string) ([]*AgentHandover, error) {
	return r.Handover.GetHandoversBySourceSession(sourceSessionID)
}

// CreateSiteProject is deprecated. Use repo.SiteProject.CreateSiteProject instead.
func (r *Repository) CreateSiteProject(project *SiteProject) error {
	return r.SiteProject.CreateSiteProject(project)
}

// GetSiteProject is deprecated. Use repo.SiteProject.GetSiteProject instead.
func (r *Repository) GetSiteProject(projectID string) (*SiteProject, error) {
	return r.SiteProject.GetSiteProject(projectID)
}

// ListSiteProjects is deprecated. Use repo.SiteProject.ListSiteProjects instead.
func (r *Repository) ListSiteProjects(limit, offset int) ([]*SiteProject, error) {
	return r.SiteProject.ListSiteProjects(limit, offset)
}

// UpdateSiteProject is deprecated. Use repo.SiteProject.UpdateSiteProject instead.
func (r *Repository) UpdateSiteProject(project *SiteProject) error {
	return r.SiteProject.UpdateSiteProject(project)
}

// DeleteSiteProject is deprecated. Use repo.SiteProject.DeleteSiteProject instead.
func (r *Repository) DeleteSiteProject(projectID string) error {
	return r.SiteProject.DeleteSiteProject(projectID)
}

// CreateSiteStep is deprecated. Use repo.SiteProject.CreateSiteStep instead.
func (r *Repository) CreateSiteStep(step *SiteStep) error {
	return r.SiteProject.CreateSiteStep(step)
}

// GetSiteSteps is deprecated. Use repo.SiteProject.GetSiteSteps instead.
func (r *Repository) GetSiteSteps(projectID string) ([]*SiteStep, error) {
	return r.SiteProject.GetSiteSteps(projectID)
}

// UpdateSiteStep is deprecated. Use repo.SiteProject.UpdateSiteStep instead.
func (r *Repository) UpdateSiteStep(step *SiteStep) error {
	return r.SiteProject.UpdateSiteStep(step)
}

// SaveSiteArtifact is deprecated. Use repo.SiteProject.SaveSiteArtifact instead.
func (r *Repository) SaveSiteArtifact(artifact *SiteArtifact) error {
	return r.SiteProject.SaveSiteArtifact(artifact)
}

// GetSiteArtifacts is deprecated. Use repo.SiteProject.GetSiteArtifacts instead.
func (r *Repository) GetSiteArtifacts(projectID string) ([]*SiteArtifact, error) {
	return r.SiteProject.GetSiteArtifacts(projectID)
}

// GetSiteArtifact is deprecated. Use repo.SiteProject.GetSiteArtifact instead.
func (r *Repository) GetSiteArtifact(artifactID int64) (*SiteArtifact, error) {
	return r.SiteProject.GetSiteArtifact(artifactID)
}

// CacheLandingPageTemplate is deprecated. Use repo.SiteProject.CacheLandingPageTemplate instead.
func (r *Repository) CacheLandingPageTemplate(template *AvailableTemplate) error {
	return r.SiteProject.CacheLandingPageTemplate(template)
}

// GetLandingPageTemplates is deprecated. Use repo.SiteProject.GetLandingPageTemplates instead.
func (r *Repository) GetLandingPageTemplates(limit int) ([]*AvailableTemplate, error) {
	return r.SiteProject.GetLandingPageTemplates(limit)
}

// GetAvatarThemes is deprecated. Use repo.SiteProject.GetAvatarThemes instead.
func (r *Repository) GetAvatarThemes() ([]*AvatarTheme, error) {
	return r.SiteProject.GetAvatarThemes()
}

// GetAvatarTheme is deprecated. Use repo.SiteProject.GetAvatarTheme instead.
func (r *Repository) GetAvatarTheme(themeID int64) (*AvatarThemeDetail, error) {
	return r.SiteProject.GetAvatarTheme(themeID)
}

// GetThemeAvatars is deprecated. Use repo.SiteProject.GetThemeAvatars instead.
func (r *Repository) GetThemeAvatars(themeID int64) ([]*Avatar, error) {
	return r.SiteProject.GetThemeAvatars(themeID)
}

// DeleteAvatarTheme is deprecated. Use repo.SiteProject.DeleteAvatarTheme instead.
func (r *Repository) DeleteAvatarTheme(themeID int64) error {
	return r.SiteProject.DeleteAvatarTheme(themeID)
}

// CreateAvatarTheme is deprecated. Use repo.SiteProject.CreateAvatarTheme instead.
func (r *Repository) CreateAvatarTheme(theme *AvatarTheme) error {
	return r.SiteProject.CreateAvatarTheme(theme)
}

// CreateAvatar is deprecated. Use repo.SiteProject.CreateAvatar instead.
func (r *Repository) CreateAvatar(avatar *Avatar) error {
	return r.SiteProject.CreateAvatar(avatar)
}

// UpdateAvatarTheme is deprecated. Use repo.SiteProject.UpdateAvatarTheme instead.
func (r *Repository) UpdateAvatarTheme(theme *AvatarTheme) error {
	return r.SiteProject.UpdateAvatarTheme(theme)
}

// GetAvatarByID is deprecated. Use repo.SiteProject.GetAvatarByID instead.
func (r *Repository) GetAvatarByID(avatarID int64) (*Avatar, error) {
	return r.SiteProject.GetAvatarByID(avatarID)
}

// DeleteAvatar is deprecated. Use repo.SiteProject.DeleteAvatar instead.
func (r *Repository) DeleteAvatar(avatarID int64) error {
	return r.SiteProject.DeleteAvatar(avatarID)
}

// UpdateAvatar is deprecated. Use repo.SiteProject.UpdateAvatar instead.
func (r *Repository) UpdateAvatar(avatarID int64, name string) error {
	return r.SiteProject.UpdateAvatar(avatarID, name)
}

func (r *Repository) SetAvatarThemeDisabled(themeID int64, disabled bool) error {
	return r.SiteProject.SetAvatarThemeDisabled(themeID, disabled)
}

// SaveTerminalSession is deprecated. Use repo.Terminal.SaveTerminalSession instead.
func (r *Repository) SaveTerminalSession(id, agentSessionID, shell, workingDir string, rows, cols int) error {
	return r.Terminal.SaveTerminalSession(id, agentSessionID, shell, workingDir, rows, cols)
}

// GetTerminalSession is deprecated. Use repo.Terminal.GetTerminalSession instead.
func (r *Repository) GetTerminalSession(id string) (map[string]interface{}, error) {
	return r.Terminal.GetTerminalSession(id)
}

// ListTerminalsByAgent is deprecated. Use repo.Terminal.ListTerminalsByAgent instead.
func (r *Repository) ListTerminalsByAgent(agentSessionID string) ([]map[string]interface{}, error) {
	return r.Terminal.ListTerminalsByAgent(agentSessionID)
}

// GetActiveTerminalsByAgent is deprecated. Use repo.Terminal.GetActiveTerminalsByAgent instead.
func (r *Repository) GetActiveTerminalsByAgent(agentSessionID string) ([]map[string]interface{}, error) {
	return r.Terminal.GetActiveTerminalsByAgent(agentSessionID)
}

// UpdateTerminalActivity is deprecated. Use repo.Terminal.UpdateTerminalActivity instead.
func (r *Repository) UpdateTerminalActivity(id string) error {
	return r.Terminal.UpdateTerminalActivity(id)
}

// UpdateTerminalEnd is deprecated. Use repo.Terminal.UpdateTerminalEnd instead.
func (r *Repository) UpdateTerminalEnd(id string, exitCode *int) error {
	return r.Terminal.UpdateTerminalEnd(id, exitCode)
}

// RecordTerminalCommand is deprecated. Use repo.Terminal.RecordTerminalCommand instead.
func (r *Repository) RecordTerminalCommand(terminalID, command string) error {
	return r.Terminal.RecordTerminalCommand(terminalID, command)
}

// GetTerminalCommandHistory is deprecated. Use repo.Terminal.GetTerminalCommandHistory instead.
func (r *Repository) GetTerminalCommandHistory(terminalID string, limit int) ([]string, error) {
	return r.Terminal.GetTerminalCommandHistory(terminalID, limit)
}

// GetTerminalStats is deprecated. Use repo.Terminal.GetTerminalStats instead.
func (r *Repository) GetTerminalStats() (map[string]interface{}, error) {
	return r.Terminal.GetTerminalStats()
}

// DeleteTerminalSession is deprecated. Use repo.Terminal.DeleteTerminalSession instead.
func (r *Repository) DeleteTerminalSession(id string) error {
	return r.Terminal.DeleteTerminalSession(id)
}

// SaveMFAConfig is deprecated. Use repo.MFA.SaveMFAConfig instead.
func (r *Repository) SaveMFAConfig(config *MFAConfig) error {
	return r.MFA.SaveMFAConfig(config)
}

// GetMFAConfig is deprecated. Use repo.MFA.GetMFAConfig instead.
func (r *Repository) GetMFAConfig(username string) (*MFAConfig, error) {
	return r.MFA.GetMFAConfig(username)
}

// DisableMFAForUser is deprecated. Use repo.MFA.DisableMFAForUser instead.
func (r *Repository) DisableMFAForUser(username string) error {
	return r.MFA.DisableMFAForUser(username)
}

// LogMFAAudit is deprecated. Use repo.MFA.LogMFAAudit instead.
func (r *Repository) LogMFAAudit(log *MFAAuditLog) error {
	return r.MFA.LogMFAAudit(log)
}

// GetMFAAuditLog is deprecated. Use repo.MFA.GetMFAAuditLog instead.
func (r *Repository) GetMFAAuditLog(username string, limit int, offset int) ([]*MFAAuditLog, error) {
	return r.MFA.GetMFAAuditLog(username, limit, offset)
}

// SaveMFATemporaryToken is deprecated. Use repo.MFA.SaveMFATemporaryToken instead.
func (r *Repository) SaveMFATemporaryToken(token *MFATemporaryToken) error {
	return r.MFA.SaveMFATemporaryToken(token)
}

// GetMFATemporaryToken is deprecated. Use repo.MFA.GetMFATemporaryToken instead.
func (r *Repository) GetMFATemporaryToken(token string) (*MFATemporaryToken, error) {
	return r.MFA.GetMFATemporaryToken(token)
}

// UpdateMFATemporaryToken is deprecated. Use repo.MFA.UpdateMFATemporaryToken instead.
func (r *Repository) UpdateMFATemporaryToken(token *MFATemporaryToken) error {
	return r.MFA.UpdateMFATemporaryToken(token)
}

// DeleteMFATemporaryToken deletes a specific temporary token.
func (r *Repository) DeleteMFATemporaryToken(token string) error {
	return r.MFA.DeleteMFATemporaryToken(token)
}

// DeleteExpiredMFATemporaryTokens is deprecated. Use repo.MFA.DeleteExpiredMFATemporaryTokens instead.
func (r *Repository) DeleteExpiredMFATemporaryTokens() error {
	return r.MFA.DeleteExpiredMFATemporaryTokens()
}

// GetMFAStats is deprecated. Use repo.MFA.GetMFAStats instead.
func (r *Repository) GetMFAStats() (map[string]interface{}, error) {
	return r.MFA.GetMFAStats()
}

// CreateUser is deprecated. Use repo.User.CreateUser instead.
func (r *Repository) CreateUser(user *DBUser) error {
	return r.User.CreateUser(user)
}

// GetUser is deprecated. Use repo.User.GetUser instead.
func (r *Repository) GetUser(username string) (*DBUser, error) {
	return r.User.GetUser(username)
}

// GetUserByIDOrUsername is deprecated. Use repo.User.GetUserByIDOrUsername instead.
func (r *Repository) GetUserByIDOrUsername(userID string) (*DBUser, error) {
	return r.User.GetUserByIDOrUsername(userID)
}

// ListUsers is deprecated. Use repo.User.ListUsers instead.
func (r *Repository) ListUsers() ([]*DBUser, error) {
	return r.User.ListUsers()
}

// UpdateUserPassword is deprecated. Use repo.User.UpdateUserPassword instead.
func (r *Repository) UpdateUserPassword(username, newPasswordHash string) error {
	return r.User.UpdateUserPassword(username, newPasswordHash)
}

// UpdateUserAvatar is deprecated. Use repo.User.UpdateUserAvatar instead.
func (r *Repository) UpdateUserAvatar(username string, avatarID *int64) error {
	return r.User.UpdateUserAvatar(username, avatarID)
}

// SetUserAdmin is deprecated. Use repo.User.SetUserAdmin instead.
func (r *Repository) SetUserAdmin(username string, isAdmin bool) error {
	return r.User.SetUserAdmin(username, isAdmin)
}

// DeleteUser is deprecated. Use repo.User.DeleteUser instead.
func (r *Repository) DeleteUser(username string) error {
	return r.User.DeleteUser(username)
}

// FindUserByOAuthID is deprecated. Use repo.User.FindUserByOAuthID instead.
func (r *Repository) FindUserByOAuthID(provider, providerID string) (*DBUser, error) {
	return r.User.FindUserByOAuthID(provider, providerID)
}

// FindUserByEmail is deprecated. Use repo.User.FindUserByEmail instead.
func (r *Repository) FindUserByEmail(email string) (*DBUser, error) {
	return r.User.FindUserByEmail(email)
}

// LogAuthAudit is deprecated. Use repo.AuthAudit.LogAuthAudit instead.
func (r *Repository) LogAuthAudit(log *AuthAuditLog) error {
	return r.AuthAudit.LogAuthAudit(log)
}

// ListAuthAuditLogs is deprecated. Use repo.AuthAudit.ListAuthAuditLogs instead.
func (r *Repository) ListAuthAuditLogs(limit, offset int) ([]*AuthAuditLog, error) {
	return r.AuthAudit.ListAuthAuditLogs(limit, offset)
}

// CreateSession is deprecated. Use repo.Session.CreateSession instead.
func (r *Repository) CreateSession(session *DBSession) error {
	return r.Session.CreateSession(session)
}

// GetSession is deprecated. Use repo.Session.GetSession instead.
func (r *Repository) GetSession(token string) (*DBSession, error) {
	return r.Session.GetSession(token)
}

// ListValidSessions retrieves all non-expired sessions from the database
func (r *Repository) ListValidSessions() ([]*DBSession, error) {
	return r.Session.ListValidSessions()
}

// DeleteSession is deprecated. Use repo.Session.DeleteSession instead.
func (r *Repository) DeleteSession(token string) error {
	return r.Session.DeleteSession(token)
}

// CleanupExpiredSessions is deprecated. Use repo.Session.CleanupExpiredSessions instead.
func (r *Repository) CleanupExpiredSessions() error {
	return r.Session.CleanupExpiredSessions()
}

// DeleteAllUserMessages is deprecated. Use repo.History.DeleteAllUserMessages instead.
func (r *Repository) DeleteAllUserMessages() error {
	return r.History.DeleteAllUserMessages()
}

// DeleteAllHistory is deprecated. Use repo.History.DeleteAllHistory instead.
func (r *Repository) DeleteAllHistory() error {
	return r.History.DeleteAllHistory()
}

// CreateWorktree creates a new worktree record. Use repo.Worktree.CreateWorktree instead.
func (r *Repository) CreateWorktree(wt *Worktree) error {
	return r.Worktree.CreateWorktree(wt)
}

// GetWorktree retrieves a worktree by ID. Use repo.Worktree.GetWorktree instead.
func (r *Repository) GetWorktree(id string) (*Worktree, error) {
	return r.Worktree.GetWorktree(id)
}

// GetWorktreeBySessionID retrieves the worktree for a session. Use repo.Worktree.GetWorktreeBySessionID instead.
func (r *Repository) GetWorktreeBySessionID(sessionID string) (*Worktree, error) {
	return r.Worktree.GetWorktreeBySessionID(sessionID)
}

// GetProjectWorktrees retrieves all active worktrees for a project. Use repo.Worktree.GetProjectWorktrees instead.
func (r *Repository) GetProjectWorktrees(projectID string) ([]*Worktree, error) {
	return r.Worktree.GetProjectWorktrees(projectID)
}

// MarkWorktreeRemoved marks a worktree as removed. Use repo.Worktree.MarkWorktreeRemoved instead.
func (r *Repository) MarkWorktreeRemoved(id string) error {
	return r.Worktree.MarkWorktreeRemoved(id)
}
