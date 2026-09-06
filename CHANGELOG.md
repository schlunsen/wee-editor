# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.12.0] - Sandbox Apps & Agent Defaults - Kite - 2026-04-06

### Added
- **Sandbox app subdomains** - deploy apps as subdomains (e.g. `app.xyz2.wee.cat`) with reverse proxy
- **MCP tools for sandbox deployment** - register, list, and remove sandbox apps via MCP
- **Persistent sandbox apps** - sandbox app registrations survive server restarts (SQLite-backed)
- **Configurable agent defaults** - set default provider, model, and permission mode in config
- **Custom provider auto-config** - auto-configure custom providers from environment variables
- **Default /app project** - seed a default app project at `/app` on first boot
- **Agent defaults in frontend** - expose agent defaults to frontend via providers API
- **Redesigned homepage** - status board layout with help page and duplicate message fix

### Fixed
- **Sandbox app proxy before auth** - deployed apps are publicly accessible without auth
- **Localhost guard** - use RemoteAddr instead of c.IP() for localhost detection
- **Unregistered subdomain fallthrough** - fall through to wee dashboard for unknown app subdomains
- **MCP config format** - wrap config in `mcpServers` key for `--mcp-config` format
- **MCP config robustness** - handle empty or invalid `.mcp.json` files gracefully
- **ANTHROPIC_SMALL_FAST_MODEL** - set to session model for custom providers
- **Justfile parse errors** - show errors in command palette instead of 500
- **YOLO mode alias** - handle 'yolo' alias in permission mode switch

## [1.10.1] - Animated Cat Logo & Navbar Cleanup - Jet - 2026-04-06

### Added
- **Animated WeeLogo component** - new animated cat logo in navbar, login, and setup pages
- **Cat logo on homepage** - welcome section now shows the cat terminal logo
- **Animated SVG favicon** - dynamic favicon matching the cat branding
- **Settings dropdown** - clean navbar with consolidated settings menu
- **Cat activity indicator** - animated cat logo with project label and activity states
- **AI provider configuration (iOS)** - configure AI providers from the iOS app

### Fixed
- **WebSocket auto-resubscribe** - auto-resubscribe to active session after reconnect
- **Concurrent prompt corruption** - prevent orphaned responseChan and concurrent prompt issues
- **WebSocket auth** - fix auth when only session auth is enabled
- **Processing state race conditions** - added tests for race condition scenarios

### Changed
- **Navbar redesign** - cleaned up navbar with centered logo and settings dropdown
- **Tracked cat logo assets** - logo SVG and Lottie JSON committed to repo

## [1.10.0] - iOS App Store & Dashboard Navigation - Ion - 2026-04-06

### Added
- **JSONL fallback for context usage** - fallback when SDK method unsupported
- **iOS App Store preparation** - app icon, display name, and submission readiness
- **Dashboard session navigation** - navigate to project tab when tapping dashboard session
- **iOS theme system** - centralized WeeColors, dark/light mode toggle
- **Chat avatar bubbles** - use session avatar in chat message bubbles

### Fixed
- **App icon alpha channel** - flatten alpha channel for App Store validation
- **Xcode CFBundleDisplayName** - deduplicate display name entries

## [1.9.1] - YOLO Mode Fix - Ignis - 2026-04-05

### Fixed
- **YOLO mode permission flags** - set DangerouslySkipPermissions in buildSDKOptions and ensureClientConnected

## [1.9.0] - iOS Multi-Device & Session Stability - Horizon - 2026-04-05

### Fixed
- **iOS multi-device sessions** - improved session handling across multiple devices
- **WebSocket authentication performance** - optimized auth perf for better responsiveness
- **iOS chat bubbles** - added timestamps to chat bubbles
- **YOLO mode permissions** - respect YOLO mode permissions for restored sessions

## [1.8.0] - Admin Management & Auto-Handoff - Flare - 2026-04-03

### Added
- **Admin user management page** - new administrative interface for user management
- **Project default skills** - set default skills per project
- **Auto-restore last session** - automatically restore the last active session on startup
- **Auto-handoff system** - infinite agent chaining with smart context transfer

## [1.7.0] - Project Skills & Auto-Handoff - Gale - 2026-04-03

### Added
- **Project default skills** - configure default skills per project
- **Auto-restore last session** - resume the last active session automatically
- **Auto-handoff** - infinite agent chaining with intelligent context transfer between agents

## [1.6.0] - GLM Models & Security Fixes - Flare - 2026-04-02

### Added
- **GLM-5/5.1 models** - support for new GLM model versions
- **Inline image previews** - display image previews directly in messages
- **Images tab** - dedicated tab for browsing message images
- **Message filters** - filter messages by type and content
- **TTS & CSP improvements** - enhanced text-to-speech and content security policy

### Fixed
- **Critical security findings** from pentest - 1 Critical, 3 High, 1 Medium, 2 Low vulnerabilities
- **Inline image preview streaming** - fix streaming of image previews in Read tool output
- **Escape key in project modal** - prevent Escape key from interrupting active sessions
- **Full-width git page** - improve layout and visibility

## [1.5.2] - Bug Fixes & Maintenance - 2026-04-01

### Fixed
- **Process Manager sidebar icon** — replaced mdi:server-outline Icon with inline SVG to fix rendering
- **Skill loading** — prevent redundant Skill tool calls and add auto-discovery

### Changed
- **Website component cleanup** — extracted website components and added shared styles
- **Security hardening** — fixed critical shell injection and CSP vulnerabilities

## [1.5.1] - Security Hardening & Mobile Polish - 2026-04-01

### Fixed
- **Security: Critical vulnerabilities** for public ngrok exposure — path traversal, CSWSH, timing attacks, DoS vectors
- **Security: HIGH vulnerabilities** from second security scan — XSS, rate limiting, API key protection
- **Security: Session ownership** validation added to all WebSocket handlers
- **Security: Self-review fixes** in security hardening layer
- **Mobile UI** — prevent horizontal scroll overflow on mobile navbar
- **Mobile responsive layout** — compact spacing and cleaner UI on small screens

## [1.5.0] - Tunnels & Subagents - 2026-03-31

### Added
- **Ngrok Tunnel Support** — first-class ngrok tunnel integration for public session exposure
- **Auto-load .env** — automatically load `.env` file on startup
- **UI Layout Improvements** — improved project selector and chat components

### Fixed
- **Subagent persistence** across project switches and app restarts
- **VPL visualization freezes** caused by infinite loop and resource leaks
- **Avatar manager** delete dialog and creative avatar naming
- **Tunnel provider default** backfill for existing configs

## [1.4.0] - Avatars & Visualizations - 2026-03-29

### Added
- **Avatar Manager** — dedicated page for managing and generating avatars
- **Avatar Theme Toggle** — ability to disable/enable avatar themes
- **Alternative Zen Visualizations** — new visualization modes for ZenMode
- **Per-Project Pack Enablement** — enable packs per-project with message persistence fixes

### Fixed
- **Parakeet FP32 Encoder** — restore FP32 encoder for WebGPU to improve transcription speed
- **Session Connectors** — auto-enable connectors on agent session creation
- **macOS dylib bundling** — bundle sherpa-onnx dylibs with macOS binaries for Homebrew

### Changed
- Refreshed website and README with new screenshots and Three.js background
- Zen mode lifecycle refactored for cleaner resource management

## [1.3.0] - Skills, Hooks & Zen Mode - 2026-03-27

### Added
- **Skills & Hooks System** - Full skill/hook management with SKILL.md parser, validator, discovery, database schema, REST API, WebSocket injection, and MCP ecosystem tools (`list_skills`, `invoke_skill`, `get_hook_status`, `create_skill_from_handover`)
- **Skill Templates Gallery** - 18 embedded skill templates with categorized gallery UI, auto-show when few skills installed
- **Packs System** - Pre-built skill and hook packs (including GStack pack) with one-click install UI
- **Debug Logging Panel** - Log broadcaster infrastructure for real-time debug visibility
- **Per-Project Library Browser** - Replaces flaky Batteries page with project-scoped library
- **Zen Mode** - Immersive session monitoring with Three.js wave animation, ambient audio engine, TTS with Kokoro browser synthesis, voice transcription overlay, KITT voice clone, reverb/glitch audio effects, water-drop ripples, per-agent colors, fullscreen mode, session switcher, message history, and live git stats
- **Browser TTS** - Kokoro browser TTS with macOS `say` fallback, F5-TTS WebGPU inference, voice selector, auto-read
- **WebLLM Auto-Tagging** - Auto-tag sessions using in-browser WebLLM after 3 messages
- **File Browser** - Browse project files alongside chat and terminal in sessions
- **Session Connectors** - Hot-pluggable connector access with junction table and env mapping
- **Hugging Face Connector** - New connector definition and migration
- **Project Colors** - Auto-assign random color to new projects with visual indicators
- **Just Integration** - Native justfile integration with command palette
- **Tauri Desktop Enhancements** - Global hotkey for skill invocation, native notification bridge, faster splash screen animations
- **Release Skill** - `/release` Claude Code skill for automated version bumping and release orchestration

### Fixed
- Security hardening: XSS, path traversal, timing attacks, CSWSH, DoS, WebSocket rate limiting, session ownership validation, API key protection, MFA IP binding
- Race conditions in session management and concurrency fixes
- Parakeet crash from unbounded audio buffer when app goes to background
- External links now open in default browser in Tauri desktop app
- Audio resource leaks and zen mode timer/watcher cleanup
- TTS audio overlap prevention with speak lock

### Changed
- **Rebranded from CCT to Wee** across all source, docs, iOS tests, and static assets
- Live view is now the default instead of Zen
- Parakeet voice model cached on filesystem instead of IndexedDB
- Unified version numbers across all 6 version locations (Go, Node, Tauri, Cargo, Website)
- Renamed Homebrew formula from `cct.rb` to `wee.rb`

## [1.1.0] - Worktree Isolation & Git Branch UI - 2026-03-15

### Added
- Git worktree integration for isolated parallel agent sessions
- Worktree UI controls in the frontend
- GitHub link button for quick repository access
- Git-remote API endpoint
- Git branch name display in sidebar session cards
- DiceBear fallback avatars with click-to-change avatar in sidebar

### Changed
- Worktree isolation enabled by default for new agent sessions

### Fixed
- Use DiceBear fallback avatars instead of broken avatar images
- Remove legacy South Park/Trailer Park Boys avatar themes from database on startup
- Remove experimental site generator link from sidebar
- Remove broken dashboard image from README

### Docs
- Rewrite README for v1.0.0 with concise product-focused overview

## [1.0.0] - Stable Release - 2026-03-15

### Added
- Parakeet v3 streaming speech-to-text engine with improved download progress tracking and UX
- Background agent toast notifications for real-time agent status updates
- Agent question handling with user interaction modals (AskUserQuestion)
- Session tags with metrics sidebar integration
- Initial prompt banner and sidebar prompt preview
- Activity pulse on sidebar sessions when messages arrive
- Agent name display under avatar in right sidebar
- Session toolbar with enhanced controls
- Kimi K2.5 model with vision support

### Changed
- Replaced South Park/Trailer Park Boys avatars with DiceBear fallback avatars
- Default session cleanup to enabled with 30-day retention
- Require Go 1.25+ (removed Go 1.24 from test matrix)

### Fixed
- Nil map panic in AskUserQuestion handler
- Broken avatar images and raw JSON prompt text handling
- Input not clearing after speech recording send
- Consistent session order maintained across page refreshes
- Auto-stop dictation when sending a message
- Toolbar rendering issues

### Performance
- Checkpoint WAL on startup for large databases
- Cap agent message content at 100KB on insert
- Move FixMessageSequences to one-time migration

## [0.17.2] - SDK v0.5.1 Upgrade - 2025-12-31

### Changed
- Upgraded claude-agent-sdk-go to v0.5.1 for latest SDK improvements and bug fixes
- Updated Go module dependencies to ensure compatibility with SDK v0.5.1

## [0.17.1] - SDK v0.5.0 Upgrade - 2025-12-31

### Changed
- Upgraded claude-agent-sdk-go to v0.5.0 for enhanced performance and latest SDK features
- Updated Go module dependencies to ensure compatibility with SDK v0.5.0

## [0.17.0] - SDK Upgrade & Agent Orchestration - 2025-12-31

### Added
- Upgraded to claude-agent-sdk-go v0.4.0 from v0.3.1 for enhanced agent capabilities and performance improvements
- Comprehensive sub-agent task test suite (sub_agent_task_test.go) demonstrating advanced orchestration patterns
- Sub-agent task simulation with real-time progress tracking and status updates
- Support for multiple parallel sub-agent tasks with independent execution and monitoring
- Robust error handling and recovery mechanisms for sub-agent task failures
- Context cancellation support with graceful task termination and cleanup
- Full output streaming capabilities for sub-agent task results

### Changed
- Enhanced agent orchestration framework to support complex multi-agent workflows
- Improved SDK integration for better agent interoperability and communication

### Fixed
- Sub-agent task error handling with proper context propagation
- Improved task result streaming performance and reliability

## [0.16.0] - Git Diff Viewer & Branch Comparison - 2025-12-02

### Added
- Git diff viewer page at /agents/:id/git-diff with comprehensive diff visualization
- Keyboard navigation with Option + arrow keys (macOS) / Alt + arrow keys (other platforms) for navigating between changed files
- Clickable file navigation from Git Status sidebar for quick access to specific file diffs
- Automatic branch comparison showing diff vs main when working tree is clean
- View Branch Diff button on feature branches for comparing against main branch
- Full file content display for newly added files with syntax highlighting
- Fixed header with scrollable diff content for better usability on long diffs
- Syntax-highlighted diffs with line numbers and proper formatting
- Added/removed line indicators with color-coded backgrounds (green for additions, red for deletions)
- Hunk headers showing context for each change section
- Real-time Git status integration showing file change types (modified, added, deleted, renamed)

### Changed
- Git status sidebar now supports clickable file navigation to jump directly to file diffs
- Enhanced Git status display with branch information and file change indicators

### Fixed
- Improved diff parsing to handle various Git diff formats correctly
- Better handling of binary files and large diffs

## [0.15.1] - Provider Configuration UI & Management - 2025-11-28

### Added
- Web-based provider configuration UI with interactive modal interface for managing AI providers
- Complete CRUD operations for provider management through REST API
- Provider search and filtering capabilities for easy discovery
- Database-backed provider storage with seed data migration for initial setup
- Dedicated /api/providers/:id/set-default endpoint for setting default provider
- Provider validation and error handling throughout the stack
- Support for multiple configured providers with single default selection
- Real-time form validation with success/error toast notifications
- Environment variables preview in configuration modal
- API key input with support for custom provider URLs
- Model selection dropdowns for providers with predefined model lists
- Model name input field for custom provider configurations
- Configure/Edit buttons on provider cards for easy access
- Delete functionality with confirmation for provider removal
- DEFAULT badge indicator showing currently active provider
- Provider state management in database schema

### Changed
- Enhanced provider repository with comprehensive update and delete operations
- Improved provider configuration API handlers with full CRUD support
- Updated authentication system to support provider-specific API keys
- Modified HandleSaveProvider to preserve existing default status instead of always setting IsCurrent=true
- Renamed ACTIVE label to DEFAULT in provider status badges for clarity
- Fixed isConfigured() to check is_configured flag instead of checking current provider status

### Fixed
- Provider configuration now correctly identifies configured vs unconfigured providers
- Multiple providers can be configured simultaneously while maintaining single default

## [0.15.0] - Analytics Default & Web Setup Flow - 2025-11-27

### Added
- Web-based initial admin user setup flow with browser-based account creation when no users exist
- Automatic browser opening when running ./wee to improve user experience
- Silent message recovery on idle transition for better conversation continuity
- Setup status endpoint to detect when initial admin user creation is needed

### Changed
- Default behavior: ./wee now launches analytics dashboard with verbose mode by default
- TUI mode now available via --tui flag (analytics is the new default)
- Removed --analytics flag as it is now the default behavior
- Setup endpoint restricted to only work when no users exist for improved security
- Status and setup endpoints always registered regardless of auth settings
- Improved MCP installation process with detailed logging

### Fixed
- Browser detection now uses runtime.GOOS instead of unreliable environment variables
- Setup endpoint access control enforced at middleware level for defense-in-depth security

## [0.14.1] - TUI Migration & Server Shutdown Improvements - 2025-11-10

### Added
- TUI functionality integrated into frontend web interface for unified management experience

### Changed
- Improved server shutdown behavior with better cleanup and graceful termination
- Process manager icon updated to use Nuxt icon component for better consistency

### Removed
- Standalone TUI mode removed in favor of web-based interface
- Profile link removed from sidebar for streamlined navigation

## [0.14.0] - Theming & Frontend Architecture - 2025-11-09

### Added
- Dark and Light themes with a maroon accent color (#7C4D4A) and Questrial typography
- Modular CSS theme architecture with separate theme files (neon.css, nord.css, dracula.css, southpark.css) for better maintainability
- Session persistence across server restarts enabling reliable session recovery and state management
- Smooth page transitions with fading effects and subtle vertical slide animation (200ms duration) for enhanced navigation UX
- Randomized loading spinners with 4 different animation styles (Rings, Dots, Gradient, Laser) creating dynamic loading experiences
- 25 creative loading messages with playful, tech-inspired, and poetic variations for personality in the UI
- Comprehensive frontend documentation including Pinia stores README and Phase 4 refactoring guide
- themes/README.md with complete theming documentation and extension guide

### Changed
- Complete refactoring of ChatArea.vue into focused, maintainable components through Phase 4 decomposition
- Enhanced CSS theme system with modular architecture and clear separation of concerns
- Unified branding updates across documentation (CLAUDE.md, README.md), Vue components, login pages, and iOS app styling
- Improved chat area loading spinner with themed CSS variables and proper light/dark mode support
- Enhanced scroll behavior with instant positioning before hiding spinner to prevent visible animation artifacts
- Loading spinners and messages now randomize on every load instead of once per component mount

### Fixed
- CSS @import ordering issue resolved by ensuring imports precede other rules for proper stylesheet loading
- Context command now only called when agent is not processing to prevent conflicts
- PWA module temporarily disabled to resolve OAuth callback interference issues
- Chat area scroll behavior improved with disabled auto-scroll during manual positioning
- Loading spinner theming now uses CSS variables (--bg-primary, --text-primary) instead of hardcoded colors

### Refactored
- Agent handler decomposed through comprehensive Phase 3 refactoring for improved code organization
- Spinner animations extracted into dedicated components for better reusability
- Theme CSS structure reorganized with _variables.css for base definitions

## [0.13.2] - GitHub Trending Repositories & Enhanced Git Features - 2025-11-05

### Added
- Trending repositories tab to the /git page with language filtering, REST API caching, and pagination support
- New /api/git/trending endpoint that uses GitHub REST API directly with 1-hour in-memory caching
- Language filtering for trending repositories (Go, Python, Rust, TypeScript, JavaScript, Java, C++, C#, Swift, Kotlin)
- Thread-safe in-memory cache for trending repositories to reduce GitHub API calls
- Default clone path setting to settings page for customizable project organization
- Clone options modal with custom path and depth support for flexible repository management
- Shows repositories with >50 stars for high-quality discovery

### Changed
- Trending repositories tab uses same UI as search and organization tabs for consistent user experience
- Full pagination support for browsing through trending repositories

### Fixed
- Undefined clone path in modal by adding proper settings configuration
- Project search now handles multi-word queries correctly for better search accuracy

## [0.13.1] - WebSocket & Message Display Improvements - 2025-11-05

### Added
- Tool uses now display in WebSocket messages within thinking sections for better visibility
- Context cache store for improved session management and UX
- Glowing ring animation to avatar during processing for visual feedback
- Multi-select deletion for sessions in iOS app

### Changed
- Simplified ProjectGitWatcher to use polling instead of fsnotify for better reliability
- Store full project object in Pinia instead of just ID for improved state management
- Enhanced session sidebar UX with better project context handling

### Fixed
- Tool-only messages now display correctly in chat interface
- Text and thinking content properly extracted from WebSocket message objects
- WebSocket authentication now properly enforced for all endpoints including GET requests
- Multi-select gesture conflicts with context menu resolved in iOS app

### Removed
- Dead code and stub flag implementations cleaned up for better maintainability
- Unused conversations, user_messages, and notifications database tables

## [0.13.0] - iOS Companion App & WebSocket Consolidation - 2025-11-03

### Added
- Native iOS companion app with SwiftUI interface for real-time agent session monitoring and control
- WebSocket-based live updates with production-grade service including ping/pong keep-alive, network monitoring, and auto-reconnection
- Voice recording capability using Apple's Speech framework with instant transcription and offline support
- ngrok integration for secure remote analytics access with automatic tunneling to a reserved ngrok domain
- iOS host management UI with menu button for managing multiple Wee server hosts
- Session interrupt functionality in iOS app with visual confirmation and warning icons
- Message detail lightbox view for viewing full message content, images, tool parameters, and thinking content
- Optimistic UI updates for instant message display when sending from iOS
- Tool overlay tooltips showing full command text with modern themed CSS styling
- Session status indicators in iOS chat header with color-coded processing/idle/error states
- Confirmation modal for session deletion with proper cleanup
- Real-time WebSocket session subscriptions with immediate status broadcasting

### Changed
- Consolidated WebSocket libraries to use only Fiber WebSocket, removing Gorilla WebSocket dependency
- Improved WebSocket message handling with proper schema matching between backend and iOS
- Enhanced authentication state tracking and error messaging in iOS app
- Refactored session deletion to use Pinia store for proper state management
- Tool overlay auto-dismiss timeout reduced to 5 seconds for cleaner UI
- Per-connection WebSocket locks replace global mutex for better multi-client concurrency
- Empty assistant messages and tool result markers filtered from UI display

### Fixed
- Tool message display in iOS app with proper field mapping (tool_uses, thinking_content)
- User message content parsing from WebSocket to show actual message text instead of placeholders
- Session status updates in iOS chat view toolbar now properly reflect real-time processing state
- Empty message boxes no longer appear after thinking messages
- WebSocket connection race conditions with 0.5s stabilization wait
- Message persistence with proper WebSocket subscription and status broadcasting
- ngrok URL handling with dynamic port determination based on host
- iOS field name mapping to match backend schema (tools -> tool_uses, thinking -> thinking_content)

### Removed
- Gorilla WebSocket dependency (913 lines removed) in favor of unified Fiber WebSocket implementation
- Duplicate iOS code including SessionsViewModel and excessive transition documentation
- Verbose debug logging spam in both Go backend and Swift iOS app
- Unused launcher.go file from internal/agents package

## [0.12.1] - Chat Navigation & Component Architecture - 2025-11-01

### Added
- Chat message history navigation with arrow keys enabling quick access to previous messages without scrolling
- Comprehensive test suite for message components with over 70% coverage for improved code quality

### Changed
- MessageBubble component refactored into modular sub-components (MessageContent, MessageHeader, MessageImages, ThinkingSection, ToolUseIndicators) for better maintainability
- Enhanced component architecture with dedicated composables for message classification, copy functionality, keyboard navigation, and event handling
- Improved accessibility helpers and message formatting utilities for better user experience

### Fixed
- Message component modularity improved through systematic decomposition into focused, testable units

## [0.12.0] - Site Generator & Extended Thinking Mode - 2025-11-01

### Added
- Nuxt UI Site Generator with streamlined 3-step workflow enabling automated static site generation from natural language descriptions
- Multi-agent orchestration system with Designer, Implementer, and Orchestrator specialists working in sequence
- Extended thinking mode display in agent chat interface showing real-time cognitive process visibility
- Per-project session defaults and persistence for consistent agent configuration across project sessions
- Real-time context usage estimation with dual-fill avatar and percentage display for token tracking
- Beautiful UI for plan messages and folder picker with enhanced visual presentation
- HTTP buffer size increases to prevent header field size errors in large payload scenarios

### Changed
- Session defaults now configurable on per-project basis with automatic persistence across restarts
- Folder picker simplified to use native browser file picker for improved user experience
- Project path input streamlined to text field for faster project configuration

### Fixed
- HTTP header field size errors resolved through increased buffer sizes for better stability
- Session configuration persistence improved across project boundaries

## [0.11.1] - Search & UI Improvements - 2025-10-30

### Added
- Message search functionality to agents page enabling quick filtering and location of specific messages within agent conversations
- Full-text search across message content with instant filtering for improved navigation of long conversation histories

### Changed
- Stats page enhanced with inner scrollable area and fixed header maintaining consistent navigation while browsing detailed statistics
- Improved scrolling behavior for better usability on pages with extensive content

## [0.11.0] - Project Areas Management & AI Context Generation - 2025-10-30

### Added
- Project areas management system enabling organization of codebase into focused work zones for better context management
- AI-powered context generation automatically creating semantic summaries from project areas for Claude agent sessions
- Detection caching for improved performance when loading project areas repeatedly
- Auto-loading project areas during agent session creation for streamlined workflow initialization
- Enhanced session creation with pre-populated contextual information from defined project areas

### Changed
- Session creation workflow enhanced with automatic context detection from project areas
- Improved project organization capabilities with structured area definitions

### Fixed
- Session ID persistence to database for reliable session resumption
- Missing ID and Timestamp fields restored in MessageRecord for proper message tracking

## [0.10.9] - Draggable Metric Sections & UI Refinements - 2025-10-29

### Added
- Draggable metric sidebar sections with persistent positioning allowing users to customize dashboard layout
- Drag and drop functionality for all metric sections enabling personalized organization
- Section order persistence across sessions maintaining user-configured layout preferences
- Enhanced provider and model display in metrics sidebar with improved visual hierarchy

### Changed
- Metric sections now fully customizable via drag-and-drop interface for better user control
- Provider and model information display refined for clearer presentation and readability
- Improved scroll behavior in agent session interface for smoother navigation experience

### Removed
- Resume session button removed from interface for streamlined user experience

## [0.10.8] - Chat Area Transition Animations - 2025-10-29

### Added
- Smooth fade-in animations for chat area when selecting agent sessions providing elegant visual feedback
- Delayed auto-scroll activation preventing janky scroll behavior during message loading
- Instant hide transition when switching or exiting sessions for immediate visual response
- Enhanced message scroll composable with auto-scroll enable/disable control for better UX coordination
- Orchestrated transition sequence ensuring messages load before visual presentation begins

### Changed
- Chat area now displays with 300ms fade-in after messages load instead of immediate appearance
- Auto-scroll disabled during chat area transitions and enabled after fade completes
- Session selection workflow improved with clean separation of entrance and exit animations
- Chat area exits instantly (opacity 0) while entering smoothly for optimized user experience

### Fixed
- Auto-scroll no longer fires during message load preventing visible scrolling artifacts
- Chat area transitions now properly coordinated with message loading state

## [0.10.7] - Git Status Metrics Dashboard - 2025-10-28

### Added
- Git Status section in metrics sidebar displaying real-time repository state including current branch, uncommitted changes, and file statuses
- Auto-refresh functionality for git status metrics with 30-second interval ensuring up-to-date repository information
- GitHub Pull Request detection showing active PRs and their status directly in the metrics sidebar
- Enhanced git status visualization with color-coded file changes (added, modified, deleted) for quick repository overview
- Clickable file paths in git status enabling easy navigation to modified files

## [0.10.6] - Enhanced Bash Tool Visualization - 2025-10-28

### Added
- BashToolPreview component for improved Bash tool display in message detail modal with terminal-style formatting
- Exit code indicators with visual success/error states for command execution results
- Collapsible stdout/stderr sections with syntax highlighting for better output readability
- Copy functionality for command outputs enabling easy clipboard access to execution results
- Enhanced command metadata display showing working directory, timeout settings, and background execution status
- Terminal-style UI with monospace fonts and code block styling for authentic command-line experience

### Changed
- Message detail modal now uses dedicated BashToolPreview component instead of generic tool display
- Bash tool execution results now display with improved visual hierarchy and organization
- Command output sections are now collapsible by default for cleaner UI presentation

## [0.10.5] - WebSocket Broadcast Fix - 2025-10-28

### Fixed
- WebSocket messages now broadcast to all active connections for each session, ensuring all clients receive updates correctly
- Improved session message delivery with proper connection tracking per session

## [0.10.4] - Agent Session Handover & UI Improvements - 2025-10-28

### Added
- Agent Session Handover API enabling context transfer between agent sessions for task delegation and collaboration
- MCP integration for handover operations with native tool support for Claude Code agents
- Comprehensive handover token system with secure, time-limited, single-use tokens
- Database tracking for all handovers with full audit trail

### Changed
- Improved avatar hash distribution using FNV-1a algorithm for better visual variety
- Enhanced sidebar navigation with reorganized menu structure and clickable logo
- Settings page now scrollable for improved user experience with long configuration lists

### Fixed
- Avatar hash distribution now provides more balanced color variety across sessions
- Duplicate messages prevented when reloading agent sessions
- Context usage messages properly filtered from database persistence
- System messages correctly filtered from display in loaded sessions

## [0.10.3] - State Management & Session Isolation - 2025-10-27

### Added
- Unified State Management Architecture with Pinia for centralized application state management
- Real-time session state synchronization ensuring consistent state across all components
- Project-specific session management with proper isolation between different projects

### Changed
- Delete all sessions functionality now project-specific, preventing accidental deletion of sessions from other projects
- Improved session management architecture with better state consistency and reactivity

### Fixed
- Tool overlay notifications now display correctly with proper positioning and visibility
- Session isolation improved to prevent cross-project data leakage
- Enhanced state synchronization for more reliable real-time updates

## [0.10.2] - Release Workflow Fix - 2025-10-27

### Fixed
- GitHub Actions release workflow now explicitly lists binary files to include, preventing Windows binaries from being included in releases
- Enhanced release reliability by specifying exact binary files (wee-darwin-amd64, wee-darwin-arm64, wee-linux-amd64, wee-linux-arm64)

## [0.10.1] - Development Time Analytics & Documentation Improvements - 2025-10-27

### Added
- Development time estimation command with comprehensive git history analysis for project analytics
- Enhanced documentation showcasing analytics features and updated project focus

### Fixed
- Removed unnecessary files to streamline project structure and improve maintainability
- Updated showcase image to better represent current project features

### Changed
- Documentation updated to remove Docker support references and highlight new analytics capabilities
- Improved README focus on analytics dashboard and real-time monitoring features

## [0.10.0] - Projects Management & UI Enhancements - 2025-10-27

### Added
- Complete projects management system with auto-detection and enhanced UI for better project organization
- Comprehensive DeepSeek model support with multiple model variants for expanded AI provider options
- Custom dropdown components in session creation modal replacing standard select elements for improved UX
- Close all button for tool notifications enabling batch dismissal of notification overlays

### Changed
- Combined Delete All Sessions and Kill All Agents buttons into unified action for cleaner UI
- Enhanced tool notifications to display actual parameter values instead of parameter names for better clarity
- Improved session creation workflow with modern dropdown components

### Fixed
- Tool notification parameter display now shows actual values instead of parameter names

## [0.9.4] - Trusted TLS Certificates & PWA Installation - 2025-10-26

### Added
- mkcert support for trusted TLS certificates enabling seamless PWA installation without browser security warnings
- Automatic mkcert certificate generation and management for locally-trusted HTTPS certificates
- PWA installation support for analytics dashboard with trusted certificates
- Comprehensive certificate management with auto-detection of best available certificate type
- Certificate regeneration commands (`wee cert --regenerate --mkcert`) for certificate lifecycle management
- Enhanced TLS startup messages showing certificate type and trust status

### Changed
- Improved TLS certificate system with automatic fallback from mkcert to self-signed certificates
- Enhanced PWA icon support with correctly resized dimensions for various display requirements
- Streamlined PWA installation experience by removing custom install button component

### Fixed
- PWA icon dimensions corrected to proper specifications for reliable installation
- Certificate trust status display improved for better user awareness

### Security
- Locally-trusted mkcert certificates provide enhanced security without browser warnings
- Automatic certificate validation and trust status reporting
- Secure certificate storage in `~/.claude/analytics/certs/` directory

## [0.9.3] - Kimi Models & Agent UI Enhancements - 2025-10-26

### Added
- New Kimi model variants (kimi-k2-0905, kimi-k2-turbo-preview) to Moonshot provider configuration
- Enhanced visual feedback for agent session states with improved user experience

### Changed
- Improved agent session UI by adding grayscale filter to avatars when sessions are in idle/ended/error states
- Enhanced visual distinction between active and inactive agent sessions for better user experience

## [0.9.2] - Slash Command Autocomplete System - 2025-10-26

### Added
- Sleek slash command autocomplete system with real-time dropdown suggestions
- Comprehensive keyboard navigation support for slash commands with arrow keys, Enter, and Escape
- Support for essential slash commands: /handoff, /clear, /help, /settings, /new, /theme
- Command categorization system with visual badges (Navigation, Session, UI, System)
- Parameter hints for commands that accept arguments with dynamic input guidance
- Beautiful UI with smooth transitions, hover states, and modern design aesthetics
- Auto-triggering autocomplete dropdown when user types '/' in chat input

### Changed
- Enhanced chat input UX with intelligent slash command discovery and execution
- Improved user workflow efficiency through keyboard-driven command navigation

### Removed
- Unused session handoff feature template file to streamline codebase

### Fixed
- Removed debug console.log statements from slash command autocomplete implementation

## [0.9.1] - Enhanced Chat Input Experience - 2025-10-26

### Added
- Auto-expanding textarea for chat input in Live Agents interface that dynamically adjusts height as users type
- Fullscreen editor button for large mode text editing with improved UX for lengthy prompts
- Smooth animations for textarea expansion and paste operations using requestAnimationFrame
- Increased maximum height to 400px for better handling of multi-line prompts

### Changed
- Chat input field now automatically grows from single line to multi-line based on content
- Improved visual feedback during text paste operations with smooth animation transitions

## [0.9.0] - Intelligent Session Handoff - 2025-10-26

### Added
- AI-powered session summarization for intelligent context transfer between agent conversations
- `/handoff` slash command for seamless session transitions with auto-generated summaries
- Session lineage tracking via `parent_session_id` field to maintain conversation history relationships
- Context summary storage for rich handoff information between sessions
- Loading indicator with animated dots during AI summary generation
- Database migration 8 for session handoff schema additions (parent_session_id, context_summary, provider fields)
- Handoff UI with purple-themed badge and editable context summary in create session modal
- Graceful fallback to rule-based summarization if AI generation fails
- 30-second timeout with comprehensive error handling for summary generation

### Changed
- Session creation now supports inheriting working directory and permission mode from parent sessions
- Sessions can now be created with pre-filled context summaries from handoff operations
- Summarize endpoint now uses session's own Claude agent for intelligent context-aware summaries
- Session metadata now includes provider information for better tracking across handoffs

### Fixed
- NULL value handling in notification stats SQL query by adding COALESCE() wrapper to SUM() aggregates
- Index creation timing for parent_session_id moved to migration to prevent "no such column" errors
- Authentication header properly included in summarize API calls using useAuthenticatedFetch composable

## [0.8.3] - Session UX & Permission Enhancements - 2025-10-26

### Added
- YOLO Mode (permission bypass) for agent sessions allowing unrestricted operations when enabled
- Directory validation for agent session creation to prevent invalid workspace configurations
- Styled browser console branding for improved developer experience

### Changed
- Simplified session creation dialog by removing redundant permission bypass checkboxes for cleaner UX
- Project permissions now correctly loaded from session's working directory instead of server default

### Fixed
- Duplicate session items in sidebar prevented through defense-in-depth validation approach
- Debug console.log statements removed from frontend production code

## [0.8.2] - Agent Session Stability & Context Filtering - 2025-10-26

### Fixed
- GLM provider base URL corrected from 'https://open.bigmodel.cn/api/paas/v4/' to 'https://api.z.ai/api/anthropic' to resolve 404 errors
- Duplicate path issue causing '/v4/v1/messages' error resolved for GLM sessions
- Context commands ('/context' and 'continue') now automatically filtered from chat history to prevent system commands from appearing in user conversations
- Session model persistence improved across page refreshes with better provider inference from stored model names
- Database purge operation now includes agent_sessions and agent_messages tables for complete cleanup
- Session restoration logic enhanced to prevent model override during page reloads

### Changed
- Auto-detected system commands now route through silent mode (saveToDb=false) to keep database clean from internal UI commands
- Database purge confirmation dialog updated to include agent sessions in the description
- API success message for purge operations updated to reflect all deleted data types
- Session manager consistency improved between session_manager.go and providers.json configuration

## [0.8.1] - Project Permission Management & Stability Improvements - 2025-10-25

### Added
- Project-level permissions management UI for granular control over project-specific permission rules
- API endpoints for managing project permissions independently from global settings
- Enhanced permission request handling with support for exact pattern matching

### Changed
- Permission UI improvements for better clarity when managing project vs global permissions
- Auto-continue message behavior refined to hide notification after permission approval

### Fixed
- Concurrent WebSocket write panics resolved with mutex protection for thread-safe operations
- Permission approval flow stability improvements preventing race conditions
- Auto-continue message now properly dismissed after user action

## [0.8.0] - Agent UX & Message Visualization - 2025-10-25

### Added
- Universal Message Modal Viewer for detailed inspection of agent messages with full metadata display
- Interactive tool use visualization with clickable pills and comprehensive diff modal
- Live session context API endpoint providing real-time working directory and git branch information
- Enhanced message bubble UI with expandable tool results and thinking process visibility
- Session context awareness with automatic git branch refresh when loading agent sessions

### Changed
- Improved chat area UX with better overflow handling and automatic session selection
- Enhanced message ordering system using sequence numbers for reliable conversation flow
- Tool execution display now features interactive pills with modal details instead of inline expansion
- Message modal provides structured view of thinking process, tool uses, and full content

### Fixed
- Git branch now always refreshes correctly when loading agent sessions
- Message sequence numbers ensure proper ordering in agent conversations
- ContentBlocks in user messages now handled correctly for database persistence
- Duplicate permissions no longer added to settings.local.json
- Chat area overflow issues resolved for better scrolling behavior

### Technical
- Added comprehensive Claude settings utilities with 192 test cases
- Enhanced WebSocket handlers with improved message detail tracking
- Refined session state management with better provider/model persistence
- Improved rule matching system for permission management

## [0.7.1] - CI/CD Infrastructure & Permission Flow Improvements - 2025-10-25

### Fixed
- Permission approval flow now completes immediately, with session reload deferred until after Claude finishes current action
- "Allow Similar" permission requests no longer interrupt Claude's execution, improving workflow smoothness

### Added
- Comprehensive CI/CD pipeline with GitHub Actions for automated testing and linting
- Test matrix covering 3 operating systems (Ubuntu, macOS, Windows) × 2 Go versions (1.23.x, 1.24.x)
- Race condition detection enabled in test runs for improved reliability
- Coverage tracking with automatic report generation
- golangci-lint integration with proper fallback to `go install` when apt installation fails
- Enhanced CONTRIBUTING.md documentation with CI/CD workflow details

### Changed
- GitHub Actions workflow README added to document CI/CD pipeline configuration

## [0.7.0] - Permission Management & Database Consolidation - 2025-10-25

### Added
- Advanced permission management system with "Allow Similar" functionality for streamlined agent permissions
- Support for allowing similar commands/patterns without requiring approval each time
- Permission profiles for common command patterns (git operations, npm commands, etc.)

### Changed
- Consolidated database architecture to single unified schema for improved performance and maintainability
- Reorganized metrics sidebar with collapsible sections for better space utilization
- Enhanced session state management with provider/model settings persistence
- Improved UI layout with cleaner session list presentation

### Fixed
- Session options now persist to database when adding/removing always-allow rules
- Provider and model settings correctly preserved when resetting session form
- Session list items no longer show redundant message count for cleaner UI

### Documentation
- Comprehensive database architecture documentation added to CLAUDE.md
- Enhanced git workflow documentation with branch protection policy

## [0.6.0] - Agent Session Control & UI Polish - 2025-10-24

### Added
- Interrupt session functionality for Live Agents allowing users to stop running agent sessions
- Manual close buttons for tool overlays providing better control over UI elements
- Enhanced metrics sidebar with provider badge for visual provider identification

### Changed
- Reorganized metrics sidebar layout with improved provider information display
- Enhanced tool overlay user experience with manual dismissal controls

### Fixed
- Permission request handling improvements for more reliable agent interactions
- WebSocket reconnection logic enhanced for better connection stability

## [0.5.15] - Browser-Based Voice Recording & Transcription - 2025-10-23

### Added
- Browser-based voice recording with Whisper.js transcription for hands-free agent interaction
- Three Whisper model options: Tiny (~35MB, fast), Base (~75MB, balanced), Small (~150MB, accurate)
- Voice recording settings UI with model selection and visual size/speed/accuracy indicators
- Auto-start recording after Whisper model loads for streamlined voice input workflow
- Keyboard shortcuts for voice recording: Ctrl/Cmd+R to start, Space to stop, Escape to cancel
- Real-time progress tracking during Whisper model downloads with per-file progress calculation
- Vertical button layout in voice recording modal optimized for mobile UX
- Visual feedback showing keyboard shortcuts during recording session
- Smart model switching that avoids reloading when same model already loaded
- Complete documentation in VOICE_RECORDING.md with architecture and usage guide

### Changed
- Voice recording UX improved with immediate auto-start after model initialization
- Recording modal buttons stacked vertically for better mobile accessibility
- Progress bar now accurately reflects multi-file Whisper model download status
- Button labels enhanced to display keyboard shortcut hints

### Technical
- Integrated Whisper.js (@xenova/transformers) for client-side speech recognition
- Dynamic model loading based on user preferences stored in settings
- Per-file progress tracking with overall percentage calculation
- Model caching to prevent unnecessary redownloads

## [0.5.14] - Username/Password Authentication System - 2025-10-23

### Added
- Comprehensive username/password authentication system with session-based login
- Admin user management interface accessible via TUI ('U' key)
- Dedicated login page with beautiful UI and form validation
- User menu dropdown in navbar showing username and admin badge
- Configurable authentication modes: disabled, optional, or required
- Session-based authentication with 24-hour token expiration
- Password change functionality for authenticated users
- Admin-only user management endpoints (create, list, delete users)
- Rate limiting on authentication endpoints (5 login attempts, 3 password changes per 15min)
- Interactive setup wizard for creating first admin user via TUI
- useAuth composable for managing authentication state across frontend
- Automatic login prompt when authentication is required
- Comprehensive USER_AUTHENTICATION.md documentation

### Security
- Bcrypt password hashing with cost factor 12 for secure password storage
- 256-bit random session tokens for secure session management
- HttpOnly session cookies to prevent XSS attacks
- Secure flag for HTTPS to prevent man-in-the-middle attacks
- Timing attack protection using crypto/subtle constant-time comparison for API keys
- File permissions: 0600 for sensitive files (users.json, sessions.json)
- Minimum 8 character password requirement
- Session middleware protecting endpoints based on configuration
- .gitignore updated to exclude sensitive files (users.json, sessions.json, .secret, *.key, *.crt)

### Changed
- Authentication middleware now properly enforces authentication based on require_login setting
- Static assets (/, /assets/, /_nuxt/, favicon) allowed without authentication
- API access properly blocked when user authentication is enabled
- Improved middleware logic for strict vs optional authentication modes
- User menu appears in top-right navbar when authenticated with logout option

### API
- POST /api/auth/login - Login with username/password credentials
- POST /api/auth/logout - Logout current session
- GET /api/auth/status - Check authentication status
- POST /api/auth/change-password - Change user password
- POST /api/auth/users - Create user (admin only)
- GET /api/auth/users - List all users (admin only)
- DELETE /api/auth/users/:username - Delete user (admin only)

### Files
- ~/.claude/analytics/users/users.json - User account storage
- ~/.claude/analytics/users/sessions.json - Active session tracking
- USER_AUTHENTICATION.md - Complete authentication documentation

## [0.5.13] - Settings UI & Unified Diff Display - 2025-10-23

### Added
- Comprehensive Settings UI page for user preference management
- Professional unified diff algorithm with contextual display using jsdiff library
- Smart context-aware diff with automatic collapsing for large file edits
- Conditional diff display based on user preferences (inline chat vs overlay)
- Diff statistics showing additions/deletions/context lines
- Side-by-side diff support infrastructure for future enhancements

### Changed
- Edit diff display now defaults to 'In Chat' for better inline visibility
- Removed 'In Panel' option from diff display settings for simplified UX
- Edit tool data now persists on messages for reliable diff display
- Diff display shows 3 lines of context around changes matching git diff standards
- Improved diff rendering with proper line-by-line matching instead of simple before/after

### Fixed
- Resolved variable shadowing bug in useDiff composable causing initialization errors
- Fixed Edit diff persistence issue by storing data directly on message objects
- Corrected default diff display values across database schema and TypeScript types
- Edit diffs now display reliably even after tool dismissal

### Removed
- Verbose debug console logging across frontend codebase for better performance
- Removed debug logs from TodoWrite parser and WebSocket connection handling
- Cleaned up console noise while preserving error and warning visibility

## [0.5.12] - Sidebar Improvements & Session State Persistence - 2025-10-23

### Added
- Enhanced sidebar with improved session state persistence for better user experience
- Session state now reliably persists across page refreshes and navigation
- Improved sidebar UI with better visual feedback and interaction patterns

### Changed
- Sidebar architecture refactored for more reliable state management
- Enhanced session tracking to maintain user context during navigation
- Improved sidebar responsiveness and performance

## [0.5.11] - Image Support & Agent UI Enhancements - 2025-10-20

### Added
- Image support in agent chat with drag-and-drop and paste functionality for visual asset sharing
- Lightbox viewer for full-screen image viewing with zoom and navigation controls
- Image gallery support in agent conversations for better visual context

### Changed
- Improved agent session creation preserving provider/model settings from TUI configuration
- Complete modularization of agents.vue component achieving 83% size reduction for better performance
- Enhanced UI responsiveness and maintainability through component refactoring

### Fixed
- Provider and model selection now correctly persists when creating new agent sessions from TUI
- Session creation workflow now respects TUI-configured settings instead of reverting to defaults

## [0.5.10] - Maintenance Release - 2025-10-20

### Changed
- Version bump for maintenance release
- Updated internal version numbers and dependencies

## [0.5.9] - Character Avatar Enhancement & Session Management - 2025-10-20

### Added
- Deterministic character avatar assignment for session IDs using hash-based mapping
- Character avatars now displayed in session selector on agents page for visual identification
- Session start time tracking and display in dropdown selectors with relative timestamps
- Enhanced session metadata showing both character name and session ID for better clarity

### Changed
- Session selector dropdown now shows character names instead of raw session IDs for better UX
- Character avatar composable enhanced with fallback logic: direct lookup first, then hash-based assignment
- Sessions now sorted by most recent first in activity history and selector dropdowns
- Activity history displays character names as primary label with session ID as metadata

### Improved
- Consistent visual identification across all session displays (agents page, activity history, dropdowns)
- Better user experience with memorable character-based session naming system
- Enhanced session management with character avatars providing visual cues throughout UI

## [0.5.8] - Multi-Provider Support & UI Enhancements - 2025-10-20

### Added
- Multi-provider support with custom model selection for flexible AI provider configuration
- Custom model input allowing users to specify any model name beyond predefined options
- Enhanced statistics page with comprehensive provider metrics and usage insights
- Interactive website features section with show/hide screenshots for better documentation
- New website screenshots showcasing analytics, live agents, shortcuts, statistics, and themes

### Changed
- Improved provider configuration with dynamic model selection across multiple AI providers
- Enhanced session metrics component with comprehensive conversation visibility
- Upgraded website UI with optimized screenshot gallery and improved navigation
- Better state calculator with tool result message type distinction

### Fixed
- Fixed viewport layout to provide desktop app experience with proper scroll behavior
- Tool results now correctly distinguished from user messages in analytics
- Provider model selection now persists custom model choices correctly
- Enhanced chat scroll behavior with deep watch mode for improved reactivity

## [0.5.7] - South Park Theme & Theme Carousel Enhancement - 2025-10-20

### Added
- South Park Dark and Light themes with Comic Sans typography and South Park-inspired color palette
- Theme selector component in header for quick theme switching
- Comprehensive theme carousel navigation page with all 5 theme families (Default, Neon, Nord, Dracula, South Park)
- Theme documentation with THEMES.md describing all theme options and customization
- Enhanced theme composable with support for theme persistence and system-wide theme management

### Changed
- Increased theme carousel scroll amount from 320px to 650px for faster navigation and improved user experience
- Improved theme toggle component with better integration for all theme families
- Enhanced sidebar with theme selector for easy access to theme customization

## [0.5.6] - Historical Message Loading & Session Deletion - 2025-10-19

### Added
- Historical message loading for agent sessions to view complete conversation history
- Session deletion functionality allowing users to remove agent sessions from the dashboard
- Deep watch mode for agent chat auto-scroll ensuring smooth scrolling behavior

### Fixed
- Agent chat auto-scroll now uses deep watch for improved reactivity and reliability
- Session management improved with proper cleanup when sessions are deleted

## [0.5.5] - Permission Streaming & Logging Enhancements - 2025-10-19

### Fixed
- Agent permission callbacks now use streaming mode for improved real-time responsiveness
- Enhanced logging for permission request handling with detailed debugging information
- Improved error handling for missing permission_id in frontend responses
- Better visibility of permission request flow through enhanced logging

### Changed
- Switched permission callbacks from non-streaming to streaming mode for better performance
- Improved logging infrastructure for permission request lifecycle tracking

## [0.5.4] - Permission Callback Improvements - 2025-10-19

### Changed
- Updated claude-agent-sdk-go dependency from v0.2.1 to v0.2.2 for improved stability

### Fixed
- Permission popup now displays tool descriptions correctly instead of showing 'undefined'
- Agent permission callbacks switched to streaming mode for better real-time responsiveness
- Enhanced permission request UI with proper tool information display
- Removed local replace directive for claude-agent-sdk-go to use official package version

## [0.5.3] - SQLite Agent Session Persistence - 2025-10-19

### Added
- SQLite-based agent session persistence for reliable session state management across server restarts
- Database-backed session storage providing durability and crash recovery
- Persistent session tracking for long-running agent conversations
- Enhanced session recovery capabilities after server interruptions

### Changed
- Agent sessions now stored in SQLite database instead of in-memory only
- Improved session reliability with persistent storage backend
- Enhanced session lifecycle management with database integration

## [0.5.2] - Agent Session Timeout Improvement - 2025-10-19

### Changed
- Increased agent session timeout from 60 seconds to 5 minutes for better stability with long-running agent operations
- Enhanced session management to accommodate complex tasks requiring extended execution time

## [0.5.1] - Global Keyboard Shortcuts - 2025-01-19

### Added
- Global keyboard shortcuts system for enhanced accessibility and efficiency
- Shortcuts dialog (press '?' key) displaying all available keyboard commands
- Comprehensive keyboard navigation support across all dashboard screens
- Help modal with organized shortcut reference grouped by functionality

### Changed
- Improved user experience with discoverable keyboard shortcuts
- Enhanced navigation workflow with consistent keyboard controls

## [0.5.0] - Unified Go Server & Enhanced Logging - 2025-01-19

### Added
- Comprehensive logging system with structured logging for enhanced debugging and monitoring
- Enhanced tool tracking with detailed execution information and metrics
- New internal/logging package providing centralized logging infrastructure
- Debug logging throughout agent server for full conversation visibility
- TodoWrite and tool execution event overlays in agents page for real-time visibility
- API key validation warnings on server startup for better troubleshooting

### Changed
- **BREAKING**: Migrated agent server from Python FastAPI to Go with native implementation
- Complete unified server implementation integrating analytics and agent functionality
- Agent server now uses Gorilla WebSocket and claude-agent-sdk-go natively in Go
- Improved session management with Go-native implementation
- Enhanced agent handler with comprehensive logging and error tracking
- Streamlined server architecture with single-process unified server
- Converted all log.Printf statements to internal/logging package for consistent verbose output

### Removed
- Python-based agent server (internal/agents/agents_server/)
- Python dependencies and virtual environment management
- FastAPI and Python FastAPI WebSocket implementation
- Legacy Python agent manager, auth, and session modules (~2500 lines of Python code)

### Fixed
- **CRITICAL**: Downgraded claude-agent-sdk-go from v0.2.0 to v0.1.3 for stability and compatibility
- **CRITICAL**: Restored WithVerbose option that was removed in SDK v0.2.0 for better debugging
- Agent connection issues caused by SDK v0.2.0 incompatibilities
- Server tests updated to match NewServerWithOptions signature with verbose parameter
- Improved error handling and logging throughout agent lifecycle
- Better signal handling and cleanup for agent server process lifecycle
- Enhanced verbose logging provides comprehensive session and tool execution visibility

## [0.4.4] - Real-Time Session Metrics Dashboard - 2025-01-16

### Added
- Real-time session metrics dashboard in agents page with comprehensive conversation visibility
- SessionMetrics.vue component displaying live session statistics:
  - Session status and duration tracking
  - Message count tracking with visual progress bars
  - Tool usage statistics and breakdown by tool type
  - Permission approval/denial rates with visual indicators
  - Working directory and configuration details
- Comprehensive debug logging to agent server for full conversation visibility
- Live tracking of tool executions, permissions, and message counts
- Detailed tool execution information extraction (files, commands, patterns) in execution bars
- Status updates during message streaming and tool execution phases
- Responsive design supporting desktop, tablet, and mobile views

### Changed
- Enhanced agents page with integrated metrics sidebar for better monitoring
- Improved real-time WebSocket updates for session metrics

## [0.4.3] - MCP Server Integration - 2025-10-16

### Added
- MCP (Model Context Protocol) server integration for agents_server to extend Claude's capabilities with custom tools and resources
- Support for stdio-based MCP server processes with configurable commands, arguments, and environment variables
- MCPServerConfig model for managing MCP server configurations
- Comprehensive MCP tool permission handling integrated with existing permission system
- MCP tool naming convention: mcp__<server_name>__<tool_name>
- Configuration options for per-server permission requirements via require_permission flag
- Documentation and examples for calculator and GitHub MCP servers

### Changed
- Enhanced agent_manager.py with _build_mcp_servers() method to register and manage MCP servers
- Improved models.py with MCP server configuration data structures

## [0.4.2] - Live Agent TodoWrite & Tool Tracking - 2025-10-16

### Added
- Real-time TodoWrite event streaming for live agent sessions in analytics dashboard
- Tool execution tracking with live updates showing tool calls and results
- Enhanced TodoWrite parsing to capture structured task updates from agents
- Live visualization of agent task progress with status indicators
- Tool event timeline showing execution history and results

### Changed
- Improved agent manager to emit TodoWrite and tool execution events via WebSocket
- Enhanced agents.vue page with dedicated TodoWrite panel showing live task updates
- Refined todo auto-hide timing with 2-second delay after completion for better visibility

### Fixed
- TodoWrite parsing now handles new agent SDK input format correctly
- Todo cleanup logic improved to prevent premature hiding of active tasks
- Better signal handling and cleanup for agent server process lifecycle

## [0.4.1] - UI Default Value Cleanup - 2025-01-16

### Fixed
- Removed hardcoded user-specific working directory defaults from agent session forms
- Working directory fields now use empty string defaults instead of a hardcoded home-directory path
- Improved UI generalization for all users

## [0.4.0] - Agent Server & Live Agent Integration - 2025-10-16

### Added
- 🤖 **Agent Server**: New Python FastAPI WebSocket server for real-time Claude agent conversations
- Full Claude Agent SDK integration with WebSocket support
- Session management for multiple concurrent agent conversations
- Embedded Python runtime for agent server functionality
- Automatic Python dependency management via virtual environments
- Comprehensive tool support (Read, Write, Edit, Bash, etc.)
- Real-time agent communication with streaming responses

### Changed
- Enhanced release workflow documentation with descriptive release names in CHANGELOG format
- Improved release process guide in workflow documentation for better clarity
- Streamlined project structure to support agent server integration
- Updated CLI commands to support agent server management

### Fixed
- Disabled Windows builds in GitHub Actions release workflow due to compilation issues
- Improved process management for agent server lifecycle

### Security
- API key authentication for agent server
- Secure session management
- Isolated Python virtual environment for dependencies

## [0.3.5] - TLS/HTTPS Security & API Authentication - 2025-10-16

### Added
- TLS/HTTPS encryption enabled by default for analytics server with auto-generated self-signed certificates
- API key authentication system protecting write operations to analytics endpoints
- Automatic API key generation and storage in `~/.claude/analytics/.secret`
- TLS certificate auto-generation with 1-year validity and expiration warnings
- Comprehensive security configuration in `~/.claude/analytics/config.json`
- Enhanced hook scripts with automatic API key authentication and TLS support
- Security documentation covering TLS, API keys, and best practices

### Changed
- Analytics server now runs on HTTPS by default (https://localhost:3333)
- All hooks updated to use API key authentication via Authorization header
- TUI analytics dashboard URLs updated to use HTTPS protocol
- Analytics server configuration now supports enabling/disabling TLS and auth independently
- Hook scripts enhanced with self-signed certificate support (-k flag for curl)

### Security
- All POST/PUT/DELETE/PATCH requests now require API key authentication
- GET requests remain unauthenticated for browser access
- Server binds to localhost (127.0.0.1) by default for security
- Self-signed certificates stored in `~/.claude/analytics/certs/`

### Fixed
- Analytics header UI cleaned up by removing non-functional "Open Dashboard" button

## [0.3.4] - AI Model Provider Tracking - 2025-10-15

### Added
- AI model provider tracking with color-coded badges in analytics dashboard
- Provider badges showing AI service (Anthropic, OpenAI, Google, etc.) with distinct colors
- Enhanced analytics UI with visual provider identification for conversations

### Changed
- Bumped GitHub Pages deployment version for improved website stability

## [0.3.3] - Browser Integration and Character Avatars - 2025-10-15

### Added
- Browser integration: Press 'O' in TUI menu to open analytics dashboard in default browser (when analytics is enabled)
- South Park character avatars for session names in analytics dashboard (26 optimized character images)
- Modern session selector dropdown in analytics with character avatars and session metadata
- Session start time tracking and sorting (most recent sessions first)
- Character avatar composable with 25+ South Park character mappings

### Changed
- Enhanced TUI help text to show 'O: Open Dashboard' when analytics is enabled
- Improved ActivityHistory component with visual session identification using character avatars
- Analytics dashboard now displays session info with avatars, IDs, and relative timestamps

## [0.3.2] - Website Branding Updates - 2025-10-15

### Changed
- Updated website favicons for improved branding consistency
- Enhanced website header with current version display (v0.3.1)

## [0.3.1] - Hook Installation Improvements - 2025-10-15

### Fixed
- Hook scripts now embedded in binary using Go's embed package for portability
- `wee --install-all-hooks` now works from any directory without requiring hook source files
- Created hooks package with embedded .sh files for reliable hook installation

## [0.3.0] - PostToolUse Hooks & Test Coverage - 2025-10-15

### Added
- Nuxt-based documentation website with modern UI at website/
- Lightbox viewer for screenshot galleries in documentation
- GitHub Actions workflow for automated website deployment
- Enhanced website UI with responsive design and improved navigation
- COVERAGE.md file documenting test coverage status

### Changed
- Improved test coverage across analytics, file watcher, and reset tracker modules
- Enhanced mobile responsiveness for documentation website
- Hero section spacing and badge alignment improvements
- Refactored version management to dedicated internal/version package

### Removed
- Non-working wrapper functionality and related scripts (internal/wrapper, scripts/install-wrapper.sh)
- Deprecated wrapper tests that were no longer functional

### Fixed
- Mobile responsive layout issues in documentation website
- Hero section spacing and component alignment

## [0.2.20] - MCP Metadata Tracking - 2025-10-14

### Added
- MCP metadata tracking system (.mcp-metadata.json) that maintains install name to server keys mapping
- Reliable MCP uninstall using metadata for exact server key removal from .mcp.json
- Comprehensive test suite with 15 new tests for MCP metadata and detection functionality
- Backward compatibility with legacy MCP installs through substring matching fallback

### Changed
- Improved TUI installation status detection for MCPs with complex names (e.g., deepgraph-vue → DeepGraph Vue MCP)
- Enhanced MCP detection with bidirectional name matching between install names and server keys

### Fixed
- MCP uninstall now accurately removes correct server entries using metadata tracking
- Installation status indicator in TUI now correctly identifies installed MCPs regardless of name complexity

## [0.2.19] - Batch Component Operations - 2025-10-14

### Added
- Multi-component selection support with Space key for batch operations
- Action chooser screen to select install/uninstall for multiple components
- Auto-refresh component list after install/remove operations to show updated status
- Visual selection indicators (checkmark) and improved help text

### Changed
- Component operations now support batch install/uninstall workflows
- Silent skip for non-installed components instead of errors during uninstall

### Fixed
- MCP removal now properly cleans up all servers from .mcp.json configuration
- Fixed broken string matching for MCP server removal

## [0.2.18] - Claude CLI Detection Improvements - 2025-10-14

### Fixed
- Claude CLI detection now works when installed in ~/.local/bin but not in PATH
- TUI launcher now properly finds Claude binary in common installation locations (/usr/local/bin, /opt/homebrew/bin, ~/.local/bin)
- FindClaudePath() function enhanced to check common locations beyond PATH

### Testing
- Added comprehensive tests for FindClaudePath() function in wrapper package
- Added 160+ lines of test coverage for Claude binary detection
- Tests cover PATH detection, common location fallback, and error cases

## [0.2.17] - Automatic Claude CLI Installer - 2025-10-14

### Added
- Automatic Claude CLI installer with native binary and npm fallback support
- New `--install-claude` flag for automated Claude CLI installation
- Node.js version detection and compatibility checking (v18+ required for npm fallback)
- TUI integration for Claude CLI installation detection and prompts
- Interactive installation prompts with progress feedback
- Detection of existing Claude installations to avoid redundant installs

### Changed
- TUI claude_launcher now detects missing Claude CLI and suggests installation
- Improved user experience with automatic installation option instead of manual setup

### Documentation
- Updated README with comprehensive installation documentation
- Added automatic installer benefits and usage examples

### Testing
- Added 40+ new tests for installer functionality
- Docker-based testing environment for clean installation verification
- Comprehensive test coverage for edge cases and error handling

## [0.2.16] - Test Coverage Infrastructure - 2025-10-14

### Added
- Comprehensive test coverage for core packages (analytics, server, websocket, components)
- Testing infrastructure with coverage thresholds and CI integration
- Scheduled CI runs for continuous test validation

### Changed
- Improved test coverage from minimal to 60%+ across critical packages
- Enhanced CI workflow with test coverage reporting

## [0.2.15] - Persistent Provider Storage - 2025-10-14

### Added
- Persistent provider token storage with SQLite database for secure credential management
- Custom model input support for AI providers allowing users to specify any model name
- Responsive compact UI for provider configuration that adapts to terminal size
- Enhanced provider configuration screen with improved layout and usability

### Changed
- Provider tokens and configurations now persist across sessions in ~/.claude/wee/wee.db
- Provider UI now displays in compact mode on smaller terminals for better accessibility
- Improved provider model selection with custom input option

## [0.2.14] - AI Provider Configuration - 2025-10-13

### Added
- AI provider configuration and management system for flexible model selection
- Support for multiple AI providers (Anthropic, OpenAI, Google, Mistral, etc.)
- Provider configuration UI in TUI for easy setup
- Model selection and API key management per provider

## [0.2.13] - Multi-Source Permissions - 2025-10-13

### Added
- Multi-source permissions management with tabbed UI for better control over Claude Code permissions
- GitHub CLI (gh) commands now included in Git Commands permission category

### Improved
- Enhanced permissions management interface with tabbed navigation between different permission sources

## [0.2.12] - Local Permissions Fix - 2025-10-13

### Fixed
- Permissions management now uses local `.claude/settings.local.json` instead of global settings file for better project isolation
- Empty permissions object is now properly removed from settings when all permissions are disabled

## [0.2.11] - Permissions Improvements - 2025-10-13

### Fixed
- Permissions management improvements

## [0.2.10] - Session Launcher & Documentation - 2025-10-13

### Added
- 'Launch last Claude session' menu option in TUI for quick access to recent conversations
- Comprehensive godoc package comments across all internal packages for better code documentation

### Fixed
- Search bar state persistence issue when navigating between screens in TUI

## [0.2.9] - Security & Stability Improvements - 2025-10-13

### Added
- Memory limits to conversation parser to prevent excessive memory usage
- HTTP timeout protection to GitHub downloads for improved reliability
- Graceful shutdown support to analytics server

### Fixed
- WebSocket hub deadlock issue with proper graceful shutdown
- File watcher resource leaks and race conditions
- Command injection vulnerability in process detector
- Performance issue by replacing bubble sort with stdlib sort.Slice

### Security
- Enhanced process detector security to prevent command injection attacks

## [0.2.8] - Homebrew CGO Fix - 2025-10-12

### Fixed
- Enabled CGO in GitHub Actions workflow to fix SQLite database support in Homebrew installations
- Analytics now works correctly when installed via Homebrew

## [0.2.7] - Analytics Startup Fix - 2025-10-12

### Fixed
- Analytics server now loads conversation data synchronously on startup to ensure data is available before server starts
- Improved initial page load experience with pre-loaded conversation data

## [0.2.6] - Command History Database - 2025-10-12

### Added
- SQLite database for command history tracking
- Automatic command history recording via conversation parsing
- Command history UI with search and filtering capabilities
- User message interception with wrapper script
- User message recording in database

### Security
- Added strict file permissions for database files

### Changed
- Simplified wrapper script implementation
- Enhanced command history tracking with persistent storage

## [0.2.5] - Integrated Analytics - 2025-10-12

### Added
- Analytics server now enabled by default in TUI mode
- Toggle shortcut (Ctrl+A) to start/stop analytics dashboard from TUI
- Quiet mode for analytics server to reduce console output
- Analytics server management integrated into TUI model lifecycle
- Dashboard screenshot in documentation

### Changed
- Analytics server runs automatically when TUI is launched (can be toggled off)
- Improved analytics server lifecycle management with graceful shutdown
- Enhanced TUI experience with integrated analytics control

### Fixed
- Removed debug print statements from WebSocket handler
- Improved analytics server startup/shutdown reliability

## [0.2.4] - Background Shell Detection - 2025-10-12

### Added
- Background shell detection in analytics dashboard
- Process monitoring enhancements for tracking Claude CLI background operations

### Changed
- Improved analytics dashboard to identify and display background shell processes

## [0.2.3] - Analytics Enhancements - 2025-10-12

### Added
- Analytics dashboard enhancements for better background process detection

## [0.2.2] - Docker Build Fix - 2025-10-12

### Fixed
- Docker build process now automatically builds wee binary before image build
- Resolved issue where Docker image build would fail if wee binary was missing

## [0.2.1] - Component Removal - 2025-10-12

### Added
- Component removal functionality - ability to uninstall agents, commands, and MCPs
- Interactive component removal in TUI with confirmation prompts

### Changed
- TUI rebranded to "Wee" with updated branding throughout interface
- Improved component management workflow with removal capabilities

## [0.2.0] - Wee Rebrand & Docker Support - 2025-10-12

### Changed - BREAKING
- **Rebrand to Wee**: Project renamed to better reflect its role as a comprehensive control center for Claude Code
- **Module path changed**: module path updated to `github.com/schlunsen/wee-editor`
- **Repository moved**: `github.com/schlunsen/claude-templates-go` → `github.com/schlunsen/wee-editor`
- All import paths updated across 25 Go files
- CLI descriptions updated to position Wee as "control center and wrapper" for Claude Code

### Added - Docker Support
- **Complete Docker integration** for containerizing Claude Code environments
- New `internal/docker/` package with 3 core modules (~700 lines):
  - `docker.go`: Docker operations (build, run, stop, logs, exec)
  - `dockerfile_generator.go`: Generate 4 types of Dockerfiles
  - `compose_generator.go`: Generate docker-compose.yml templates
- **9 new CLI commands**:
  - `--docker-init`: Generate Dockerfile + .dockerignore
  - `--docker-build`: Build Docker image
  - `--docker-run`: Run containerized Claude environment
  - `--docker-stop`: Stop Docker container
  - `--docker-logs`: View container logs
  - `--docker-compose`: Generate docker-compose.yml
  - `--docker-type`: Select type (base/claude/analytics/full)
  - `--docker-mcps`: Include MCPs in container (comma-separated)
  - `--docker-command`: Custom command to run in container
- **4 Dockerfile templates**:
  - `base`: Minimal Wee-only image
  - `claude`: Full environment (Node.js + Claude CLI + Wee + MCPs)
  - `analytics`: Optimized for analytics dashboard
  - `full`: Complete dev environment with all tools
- **4 docker-compose templates**:
  - `simple`: Claude + Wee
  - `analytics`: Claude + Analytics dashboard
  - `database`: Claude + PostgreSQL
  - `full`: All services (Claude + Analytics + PostgreSQL + Redis)
- MCP integration in Docker containers
- Automatic .dockerignore and .env.example generation

### Improved
- Enhanced TUI with installation status indicators ([G]=Global, [P]=Project)
- Simplified component selection to single-select on Enter
- Improved navigation flow (Enter/Esc returns to list from completion)
- Enhanced "Launch Claude" menu item visibility with special styling
- Better UX showing installation status before installing

### Documentation
- Complete README overhaul with Docker section and migration guide
- Updated CLAUDE.md with new project overview and Docker architecture
- All documentation files updated with new repository URLs
- GitHub workflows updated with new repository references

## [0.1.0] - Stable TUI Release - 2025-10-12

### Changed
- Minor version bump to 0.1.0 marking stable TUI and core functionality

## [0.0.9] - Claude CLI Launcher - 2025-10-12

### Added
- Claude CLI launcher integration in TUI for direct conversation launching
- Launch Claude Code conversations from selected agents or components
- Interactive component selection with conversation context

## [0.0.8] - MCP Registration Fix - 2025-10-12

### Fixed
- TUI MCP installer now properly registers MCP servers in .mcp.json configuration file
- MCPs installed via TUI now work correctly in Claude Code

### Changed
- Added .mcp.json to .gitignore to prevent committing local MCP configurations

## [0.0.7] - Component Preview - 2025-10-12

### Added
- Preview functionality for agents, commands, and MCPs via --preview/-p flag
- Interactive preview screen in TUI with scrollable content viewing
- Preview methods for all component installers
- Ability to view component content before installation in both CLI and TUI modes
- Keyboard navigation in TUI preview: arrow keys, PgUp/PgDn, g/G for top/bottom
- Direct install from preview screen with I key in TUI
- P key to preview selected component from list in TUI

### Fixed
- MCP registration in TUI now uses proper project scope

## [0.0.6] - MCP Configuration - 2025-10-12

### Added
- MCP installation now properly registers servers in Claude Code configuration files
- Support for project-local vs user-global MCP installation via --scope flag
- Configuration utilities for reading/writing MCP config files

### Changed
- Automated release process with Claude agent integration in justfile

### Fixed
- MCPs not showing up in Claude Code's /mcp command after installation
- MCP servers not being properly registered in .mcp.json or ~/.claude/config.json

## [0.0.5] - TUI Improvements - 2025-10-12

### Added
- Active filter display with contextual hints when search is not focused
- Two-step Esc behavior: first clears filter, second returns to main screen

### Changed
- Dynamic viewport calculation based on terminal height
- Component list now adapts to any terminal size (min 5, max 20 items)
- Compact help text for terminals with height < 20 lines
- Centered cursor positioning in viewport for better navigation

### Fixed
- TUI elements being cut off in small terminal windows
- Search filter state unclear after exiting search mode
- Help text and component lists truncated in limited height terminals

## [0.0.4] - Navigation & Documentation - 2025-10-12

### Added
- Page up/down navigation support in component lists

### Changed
- Organized documentation files into docs/ directory
- Streamlined changelog to follow Keep a Changelog format

### Removed
- Old test scripts from project root

## [0.0.3] - Modern TUI - 2025-10-12

### Added
- Modern interactive TUI with theme support
- Bubbles/Bubbletea-based component selection interface
- Visual theme with gradients and modern styling

### Fixed
- Homebrew formula generation in release workflow
- Installation documentation accuracy

## [0.0.2] - Homebrew & Documentation - 2025-10-12

### Added
- Homebrew formula generation to release workflow
- Automated release commands to justfile
- Professional README improvements
- LICENSE file with MIT License
- CONTRIBUTING.md with development guidelines
- GitHub issue templates (bug report, feature request, question)
- Pull request template
- Badges to README (Go version, license, build status, release)
- Table of contents to README

### Changed
- Streamlined README for professional appearance
- Updated repository URLs from placeholders to actual repository
- Enhanced code blocks with language labels

## [0.0.1] - Initial Go Port - 2025-10-12

### Added
- Initial Go implementation
- CLI with Cobra framework and Pterm terminal UI
- Fiber web server with WebSocket support for real-time updates
- Analytics dashboard with embedded frontend
- Smart category search for agents, commands, and MCPs across 50+ categories
- Cross-platform builds (Linux, macOS, Windows on amd64/arm64)
- File system watching with fsnotify for real-time conversation monitoring
- RESTful API with 6 endpoints (health, data, conversations, processes, stats, refresh)
- Comprehensive test suite with automated category search validation
- Makefile and justfile for build automation
- GitHub Actions workflow for multi-platform releases
- Component management system (agents, commands, MCPs)
- File operations module for template management
- ConversationAnalyzer and FileWatcher modules
- Analytics core modules (StateCalculator, ProcessDetector)

### Fixed
- Component installation 404 errors with comprehensive category search
- Path handling from "cli-tool/templates" to "cli-tool"
- WebSocket unused variable warnings

## Version Comparison Links

[0.14.0]: https://github.com/schlunsen/wee-editor/compare/v0.13.2...v0.14.0
[0.13.2]: https://github.com/schlunsen/wee-editor/compare/v0.13.1...v0.13.2
[0.13.1]: https://github.com/schlunsen/wee-editor/compare/v0.13.0...v0.13.1
[0.13.0]: https://github.com/schlunsen/wee-editor/compare/v0.12.1...v0.13.0
[0.12.1]: https://github.com/schlunsen/wee-editor/compare/v0.12.0...v0.12.1
[0.12.0]: https://github.com/schlunsen/wee-editor/compare/v0.11.1...v0.12.0
[0.11.1]: https://github.com/schlunsen/wee-editor/compare/v0.11.0...v0.11.1
[0.11.0]: https://github.com/schlunsen/wee-editor/compare/v0.10.9...v0.11.0
[0.10.9]: https://github.com/schlunsen/wee-editor/compare/v0.10.8...v0.10.9
[0.10.8]: https://github.com/schlunsen/wee-editor/compare/v0.10.7...v0.10.8
[0.10.7]: https://github.com/schlunsen/wee-editor/compare/v0.10.6...v0.10.7
[0.10.6]: https://github.com/schlunsen/wee-editor/compare/v0.10.5...v0.10.6
[0.10.5]: https://github.com/schlunsen/wee-editor/compare/v0.10.4...v0.10.5
[0.10.4]: https://github.com/schlunsen/wee-editor/compare/v0.10.3...v0.10.4
[0.10.3]: https://github.com/schlunsen/wee-editor/compare/v0.10.2...v0.10.3
[0.10.2]: https://github.com/schlunsen/wee-editor/compare/v0.10.1...v0.10.2
[0.10.1]: https://github.com/schlunsen/wee-editor/compare/v0.10.0...v0.10.1
[0.10.0]: https://github.com/schlunsen/wee-editor/compare/v0.9.4...v0.10.0
[0.9.4]: https://github.com/schlunsen/wee-editor/compare/v0.9.3...v0.9.4
[0.9.3]: https://github.com/schlunsen/wee-editor/compare/v0.9.2...v0.9.3
[0.9.2]: https://github.com/schlunsen/wee-editor/compare/v0.9.1...v0.9.2
[0.9.1]: https://github.com/schlunsen/wee-editor/compare/v0.9.0...v0.9.1
[0.9.0]: https://github.com/schlunsen/wee-editor/compare/v0.8.3...v0.9.0
[0.8.3]: https://github.com/schlunsen/wee-editor/compare/v0.8.2...v0.8.3
[0.8.2]: https://github.com/schlunsen/wee-editor/compare/v0.8.1...v0.8.2
[0.8.1]: https://github.com/schlunsen/wee-editor/compare/v0.8.0...v0.8.1
[0.8.0]: https://github.com/schlunsen/wee-editor/compare/v0.7.1...v0.8.0
[0.7.1]: https://github.com/schlunsen/wee-editor/compare/v0.7.0...v0.7.1
[0.7.0]: https://github.com/schlunsen/wee-editor/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/schlunsen/wee-editor/compare/v0.5.15...v0.6.0
[0.5.15]: https://github.com/schlunsen/wee-editor/compare/v0.5.14...v0.5.15
[0.5.14]: https://github.com/schlunsen/wee-editor/compare/v0.5.13...v0.5.14
[0.5.13]: https://github.com/schlunsen/wee-editor/compare/v0.5.12...v0.5.13
[0.5.12]: https://github.com/schlunsen/wee-editor/compare/v0.5.11...v0.5.12
[0.5.11]: https://github.com/schlunsen/wee-editor/compare/v0.5.10...v0.5.11
[0.5.10]: https://github.com/schlunsen/wee-editor/compare/v0.5.9...v0.5.10
[0.5.9]: https://github.com/schlunsen/wee-editor/compare/v0.5.8...v0.5.9
[0.5.8]: https://github.com/schlunsen/wee-editor/compare/v0.5.7...v0.5.8
[0.5.7]: https://github.com/schlunsen/wee-editor/compare/v0.5.6...v0.5.7
[0.5.6]: https://github.com/schlunsen/wee-editor/compare/v0.5.5...v0.5.6
[0.5.5]: https://github.com/schlunsen/wee-editor/compare/v0.5.4...v0.5.5
[0.5.4]: https://github.com/schlunsen/wee-editor/compare/v0.5.3...v0.5.4
[0.5.3]: https://github.com/schlunsen/wee-editor/compare/v0.5.2...v0.5.3
[0.5.2]: https://github.com/schlunsen/wee-editor/compare/v0.5.1...v0.5.2
[0.5.1]: https://github.com/schlunsen/wee-editor/compare/v0.5.0...v0.5.1
[0.5.0]: https://github.com/schlunsen/wee-editor/compare/v0.4.4...v0.5.0
[0.4.4]: https://github.com/schlunsen/wee-editor/compare/v0.4.3...v0.4.4
[0.4.3]: https://github.com/schlunsen/wee-editor/compare/v0.4.2...v0.4.3
[0.4.2]: https://github.com/schlunsen/wee-editor/compare/v0.4.1...v0.4.2
[0.4.1]: https://github.com/schlunsen/wee-editor/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/schlunsen/wee-editor/compare/v0.3.5...v0.4.0
[0.3.5]: https://github.com/schlunsen/wee-editor/compare/v0.3.4...v0.3.5
[0.3.4]: https://github.com/schlunsen/wee-editor/compare/v0.3.3...v0.3.4
[0.3.3]: https://github.com/schlunsen/wee-editor/compare/v0.3.2...v0.3.3
[0.3.2]: https://github.com/schlunsen/wee-editor/compare/v0.3.1...v0.3.2
[0.3.1]: https://github.com/schlunsen/wee-editor/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/schlunsen/wee-editor/compare/v0.2.20...v0.3.0
[0.2.20]: https://github.com/schlunsen/wee-editor/compare/v0.2.19...v0.2.20
[0.2.19]: https://github.com/schlunsen/wee-editor/compare/v0.2.18...v0.2.19
[0.2.18]: https://github.com/schlunsen/wee-editor/compare/v0.2.17...v0.2.18
[0.2.17]: https://github.com/schlunsen/wee-editor/compare/v0.2.16...v0.2.17
[0.2.16]: https://github.com/schlunsen/wee-editor/compare/v0.2.15...v0.2.16
[0.2.15]: https://github.com/schlunsen/wee-editor/compare/v0.2.14...v0.2.15
[0.2.14]: https://github.com/schlunsen/wee-editor/compare/v0.2.13...v0.2.14
[0.2.13]: https://github.com/schlunsen/wee-editor/compare/v0.2.12...v0.2.13
[0.2.12]: https://github.com/schlunsen/wee-editor/compare/v0.2.11...v0.2.12
[0.2.11]: https://github.com/schlunsen/wee-editor/compare/v0.2.10...v0.2.11
[0.2.10]: https://github.com/schlunsen/wee-editor/compare/v0.2.9...v0.2.10
[0.2.9]: https://github.com/schlunsen/wee-editor/compare/v0.2.8...v0.2.9
[0.2.8]: https://github.com/schlunsen/wee-editor/compare/v0.2.7...v0.2.8
[0.2.7]: https://github.com/schlunsen/wee-editor/compare/v0.2.6...v0.2.7
[0.2.6]: https://github.com/schlunsen/wee-editor/compare/v0.2.5...v0.2.6
[0.2.5]: https://github.com/schlunsen/wee-editor/compare/v0.2.4...v0.2.5
[0.2.4]: https://github.com/schlunsen/wee-editor/compare/v0.2.3...v0.2.4
[0.2.3]: https://github.com/schlunsen/wee-editor/compare/v0.2.2...v0.2.3
[0.2.2]: https://github.com/schlunsen/wee-editor/compare/v0.2.1...v0.2.2
[0.2.1]: https://github.com/schlunsen/wee-editor/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/schlunsen/wee-editor/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/schlunsen/wee-editor/compare/v0.0.9...v0.1.0
[0.0.9]: https://github.com/schlunsen/wee-editor/compare/v0.0.8...v0.0.9
[0.0.8]: https://github.com/schlunsen/wee-editor/compare/v0.0.7...v0.0.8
[0.0.7]: https://github.com/schlunsen/wee-editor/compare/v0.0.6...v0.0.7
[0.0.6]: https://github.com/schlunsen/wee-editor/compare/v0.0.5...v0.0.6
[0.0.5]: https://github.com/schlunsen/wee-editor/compare/v0.0.4...v0.0.5
[0.0.4]: https://github.com/schlunsen/wee-editor/compare/v0.0.3...v0.0.4
[0.0.3]: https://github.com/schlunsen/wee-editor/compare/v0.0.2...v0.0.3
[0.0.2]: https://github.com/schlunsen/wee-editor/compare/v0.0.1...v0.0.2
[0.0.1]: https://github.com/schlunsen/wee-editor/releases/tag/v0.0.1
