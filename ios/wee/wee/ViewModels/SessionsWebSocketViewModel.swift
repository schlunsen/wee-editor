//
//  SessionsWebSocketViewModel.swift
//  wee
//
//  ViewModel for real-time session management via WebSocket
//  Combines REST API with WebSocket event streaming
//

import SwiftUI
import Combine

@MainActor
class SessionsWebSocketViewModel: NSObject, ObservableObject, WebSocketEventHandler {
    // MARK: - Singleton

    static let shared = SessionsWebSocketViewModel()

    // MARK: - Published Properties

    @Published var sessions: [AgentSession] = []
    @Published var isLoading = false
    @Published var errorMessage: String?
    @Published var selectedSessionId: String?
    @Published var webSocketConnected = false  // Track WebSocket connection status

    // Real-time message streaming
    @Published var sessionMessages: [String: [AgentMessage]] = [:]
    @Published var messagesLoading: [String: Bool] = [:]
    @Published var sessionStatus: [String: String] = [:]  // Track status per session

    private let apiClient = WeeAPIClient.shared
    private let webSocket = WebSocketService.shared

    private override init() {
        super.init()
        webSocket.eventHandler = self
    }

    // MARK: - Initialization & Connection

    /// Start real-time session monitoring
    func startMonitoring() async {

        // Connect WebSocket
        await webSocket.connect()

        // Request initial session list
        _ = webSocket.requestSessions()

        // Also load sessions via REST API as fallback
        await loadSessions()
    }

    /// Stop real-time monitoring
    func stopMonitoring() {
        webSocket.disconnect()
    }

    // MARK: - Session Management

    /// Load sessions via REST API (fallback)
    func loadSessions() async {
        isLoading = true
        errorMessage = nil

        defer { isLoading = false }

        do {
            sessions = try await apiClient.getAgentSessions()
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    /// Select a session and load its messages.
    /// Idempotent — skips WebSocket calls if already subscribed to this session with messages loaded.
    func selectSession(_ sessionId: String) {
        // Skip if already selected AND messages are loaded (prevents redundant subscribe/load
        // when navigating between ChatView ↔ ZenView within the same navigation stack)
        if selectedSessionId == sessionId && sessionMessages[sessionId] != nil {
            return
        }

        selectedSessionId = sessionId

        // Subscribe to real-time updates for this session (CRITICAL for receiving new messages)
        _ = webSocket.subscribeToSession(sessionId: sessionId)

        // Load historical messages via WebSocket (limit to 50 to avoid oversized WS frames
        // that crash the connection on mobile devices over ngrok)
        _ = webSocket.loadMessages(sessionId: sessionId, limit: 50, offset: 0)
    }

    /// Force-reload messages for a session (used by the toolbar refresh button).
    /// Unlike selectSession(), this always sends the load_messages command.
    func reloadSessionMessages(_ sessionId: String) {
        _ = webSocket.loadMessages(sessionId: sessionId, limit: 50, offset: 0)
    }

    // MARK: - WebSocket Event Handlers

    nonisolated func onSessionCreated(_ data: [String: Any]) {
        Task { @MainActor in

            if let session = self.parseSession(from: data) {
                // Add to front of list
                self.sessions.insert(session, at: 0)

                // Initialize message array
                if self.sessionMessages[session.id] == nil {
                    self.sessionMessages[session.id] = []
                }

                // Select this session
                self.selectedSessionId = session.id

            }
        }
    }

    nonisolated func onSessionUpdated(_ data: [String: Any]) {
        Task { @MainActor in
            #if DEBUG
            print("📥 onSessionUpdated called with keys: \(data.keys.sorted())")
            #endif

            // First try to parse the full session object from the message
            // Backend sends session data nested under "session" key
            if let sessionData = data["session"] as? [String: Any] {
                #if DEBUG
                print("📥 Session data keys: \(sessionData.keys.sorted())")
                #endif
                #if DEBUG
                print("📥 Session data selected_avatar: \(sessionData["selected_avatar"] ?? "nil")")
                #endif

                if let parsedSession = self.parseSession(from: sessionData) {
                    #if DEBUG
                    print("✅ Parsed session: id=\(parsedSession.id), avatar=\(parsedSession.selected_avatar?.name ?? "none")")
                    #endif

                    if let index = self.sessions.firstIndex(where: { $0.id == parsedSession.id }) {
                        self.sessions[index] = parsedSession
                    } else {
                        // Session not in list, add it
                        self.sessions.append(parsedSession)
                    }

                    // Update status tracking
                    var newStatusDict = self.sessionStatus
                    newStatusDict[parsedSession.id] = parsedSession.status
                    self.sessionStatus = newStatusDict

                    #if DEBUG
                    print("✅ Session updated with avatar: \(parsedSession.selected_avatar?.name ?? "none")")
                    #endif
                } else {
                    #if DEBUG
                    print("❌ Failed to parse session from session data")
                    #endif
                }
            } else if let sessionId = data["session_id"] as? String,
                      let index = self.sessions.firstIndex(where: { $0.id == sessionId }) {
                // Fallback: update just the status from top-level fields
                let currentSession = self.sessions[index]

                let updatedSession = AgentSession(
                    id: currentSession.id,
                    status: (data["status"] as? String) ?? currentSession.status,
                    model_name: currentSession.model_name,
                    provider: currentSession.provider,
                    git_branch: (data["git_branch"] as? String) ?? currentSession.git_branch,
                    working_directory: currentSession.working_directory,
                    created_at: currentSession.created_at,
                    updated_at: currentSession.updated_at,
                    message_count: (data["message_count"] as? Int) ?? currentSession.message_count,
                    cost: (data["cost"] as? Double) ?? currentSession.cost,
                    project_id: currentSession.project_id,
                    selected_avatar_id: currentSession.selected_avatar_id,
                    selected_avatar: currentSession.selected_avatar
                )

                if let status = data["status"] as? String {
                    var newStatusDict = self.sessionStatus
                    newStatusDict[sessionId] = status
                    self.sessionStatus = newStatusDict
                }

                self.sessions[index] = updatedSession
            } else {
                // Even if session not in list, update status tracking (for new sessions)
                if let sessionId = data["session_id"] as? String,
                   let status = data["status"] as? String {
                    var newStatusDict = self.sessionStatus
                    newStatusDict[sessionId] = status
                    self.sessionStatus = newStatusDict
                }
            }
        }
    }

    nonisolated func onAgentMessage(_ data: [String: Any]) {
        Task { @MainActor in
            guard let sessionId = data["session_id"] as? String else {
                return
            }

            // Initialize messages array if needed
            if self.sessionMessages[sessionId] == nil {
                self.sessionMessages[sessionId] = []
            }

            // Parse message
            if let message = self.parseMessage(from: data) {

                // CRITICAL: Create a completely new array (not just modify existing)
                var currentMessages = self.sessionMessages[sessionId] ?? []
                currentMessages.append(message)

                // CRITICAL: Create a completely new dictionary
                var newDict = [String: [AgentMessage]]()
                for (key, value) in self.sessionMessages {
                    newDict[key] = value
                }
                newDict[sessionId] = currentMessages

                // CRITICAL: Replace entire dictionary to trigger @Published
                self.sessionMessages = newDict
            }

            // Update session status (create new dictionary to trigger @Published)
            if let status = data["status"] as? String {
                var newStatusDict = self.sessionStatus
                newStatusDict[sessionId] = status
                self.sessionStatus = newStatusDict
            }

        }
    }

    nonisolated func onAgentThinking(_ data: [String: Any]) {
        Task { @MainActor in
            guard let sessionId = data["session_id"] as? String else { return }

            // Update session status to processing (create new dictionary to trigger @Published)
            if let isThinking = data["thinking"] as? Bool, isThinking {
                var newStatusDict = self.sessionStatus
                newStatusDict[sessionId] = "processing"
                self.sessionStatus = newStatusDict
            }
        }
    }

    nonisolated func onAgentToolUse(_ data: [String: Any]) {
        Task { @MainActor in
            guard let sessionId = data["session_id"] as? String,
                  data["tool"] is String else { return }

            // Update session status (create new dictionary to trigger @Published)
            var newStatusDict = self.sessionStatus
            newStatusDict[sessionId] = "processing"
            self.sessionStatus = newStatusDict
        }
    }

    nonisolated func onAgentError(_ data: [String: Any]) {
        Task { @MainActor in
            guard let sessionId = data["session_id"] as? String else { return }

            let errorMessage = data["error"] as? String ?? "Unknown error"

            // Update session status (create new dictionary to trigger @Published)
            var newStatusDict = self.sessionStatus
            newStatusDict[sessionId] = "error"
            self.sessionStatus = newStatusDict

            // Add error message to chat
            if self.sessionMessages[sessionId] == nil {
                self.sessionMessages[sessionId] = []
            }

            let errorMsg = AgentMessage(
                id: UUID().uuidString,
                role: "error",
                content: ContentBlockOrString.string(errorMessage),
                created_at: ISO8601DateFormatter().string(from: Date()),
                thinking: nil,
                tools: nil
            )
            self.sessionMessages[sessionId]?.append(errorMsg)
        }
    }

    nonisolated func onPermissionRequest(_ data: [String: Any]) {
        // Permission requests are handled by the chat view directly
    }

    nonisolated func onPermissionAcknowledged(_ data: [String: Any]) {
        // Permission acknowledgements are handled by the chat view directly
    }

    nonisolated func onSessionsList(_ data: [String: Any]) {
        Task { @MainActor in

            if let sessionsList = data["sessions"] as? [[String: Any]] {
                var newSessions: [AgentSession] = []

                for sessionData in sessionsList {
                    if let session = self.parseSession(from: sessionData) {
                        newSessions.append(session)
                    }
                }

                // Replace sessions list (deduplicating)
                self.sessions = newSessions
            }
        }
    }

    nonisolated func onMessagesLoaded(_ data: [String: Any]) {
        Task { @MainActor in
            guard let sessionId = data["session_id"] as? String else { return }

            // Initialize messages array
            if self.sessionMessages[sessionId] == nil {
                self.sessionMessages[sessionId] = []
            }

            // Parse messages
            if let messagesList = data["messages"] as? [[String: Any]] {
                var messages: [AgentMessage] = []

                for msgData in messagesList {
                    if let message = self.parseMessage(from: msgData) {
                        messages.append(message)
                    }
                }

                // Set messages (replace, not append - these are historical)
                self.sessionMessages[sessionId] = messages
                self.messagesLoading[sessionId] = false

            }
        }
    }

    nonisolated func onSessionDeleted(_ data: [String: Any]) {
        Task { @MainActor in
            if let sessionId = data["session_id"] as? String {

                // Remove from lists
                self.sessions.removeAll { $0.id == sessionId }
                self.sessionMessages.removeValue(forKey: sessionId)

                // Remove from status (create new dictionary to trigger @Published)
                var newStatusDict = self.sessionStatus
                newStatusDict.removeValue(forKey: sessionId)
                self.sessionStatus = newStatusDict

                // Deselect if it was selected
                if self.selectedSessionId == sessionId {
                    self.selectedSessionId = nil
                }
            }
        }
    }

    nonisolated func onConnectionStatusChanged(_ isConnected: Bool) {
        Task { @MainActor in
            // Update connection status
            self.webSocketConnected = isConnected

            if !isConnected {
                self.errorMessage = "WebSocket connection lost"
            } else {
                self.errorMessage = nil
                // Request session list when reconnected
                _ = self.webSocket.requestSessions()

                // Re-select the current session to re-subscribe and reload messages.
                // Without this, the ChatView stays stuck on "Unable to Load Messages"
                // because messages were cleared during the disconnect.
                if let sessionId = self.selectedSessionId {
                    #if DEBUG
                    print("[WS] reconnected — re-subscribing to session \(sessionId)")
                    #endif
                    _ = self.webSocket.subscribeToSession(sessionId: sessionId)
                    _ = self.webSocket.loadMessages(sessionId: sessionId, limit: 50, offset: 0)
                }
            }
        }
    }

    nonisolated func onError(_ error: String) {
        Task { @MainActor in
            self.errorMessage = error
        }
    }

    // MARK: - Helper Methods

    /// Parse AgentSession from dictionary
    private func parseSession(from data: [String: Any]) -> AgentSession? {
        guard let id = data["id"] as? String ?? data["session_id"] as? String else {
            #if DEBUG
            print("❌ parseSession: No id found in data keys: \(data.keys.sorted())")
            #endif
            return nil
        }

        #if DEBUG
        print("📥 parseSession: Parsing session id=\(id)")
        #endif
        #if DEBUG
        print("📥 parseSession: selected_avatar type = \(type(of: data["selected_avatar"]))")
        #endif

        // Parse avatar if present
        var selectedAvatar: Avatar? = nil
        if let avatarData = data["selected_avatar"] as? [String: Any] {
            #if DEBUG
            print("📥 parseSession: avatarData keys = \(avatarData.keys.sorted())")
            #endif
            if let avatarId = avatarData["id"] as? Int64 ?? (avatarData["id"] as? NSNumber)?.int64Value {
                #if DEBUG
                print("📥 parseSession: Creating avatar id=\(avatarId), name=\(avatarData["name"] as? String ?? "unknown")")
                #endif
                selectedAvatar = Avatar(
                    id: avatarId,
                    theme_id: (avatarData["theme_id"] as? Int64) ?? (avatarData["theme_id"] as? NSNumber)?.int64Value ?? 0,
                    name: avatarData["name"] as? String ?? "Unknown",
                    type: avatarData["type"] as? String,
                    image_path: avatarData["image_path"] as? String,
                    image_url: avatarData["image_url"] as? String,
                    style: avatarData["style"] as? String,
                    seed: avatarData["seed"] as? String,
                    color: avatarData["color"] as? String,
                    created_at: avatarData["created_at"] as? String ?? ISO8601DateFormatter().string(from: Date())
                )
            } else {
                #if DEBUG
                print("❌ parseSession: avatarData has no valid id")
                #endif
            }
        } else {
            #if DEBUG
            print("📥 parseSession: No selected_avatar in data")
            #endif
        }

        return AgentSession(
            id: id,
            status: data["status"] as? String ?? "idle",
            model_name: data["model_name"] as? String,
            provider: data["provider"] as? String,
            git_branch: data["git_branch"] as? String,
            working_directory: data["working_directory"] as? String,
            created_at: data["created_at"] as? String ?? ISO8601DateFormatter().string(from: Date()),
            updated_at: data["updated_at"] as? String ?? ISO8601DateFormatter().string(from: Date()),
            message_count: data["message_count"] as? Int,
            cost: data["cost"] as? Double,
            project_id: data["project_id"] as? String,
            selected_avatar_id: (data["selected_avatar_id"] as? Int64) ?? (data["selected_avatar_id"] as? NSNumber)?.int64Value ?? (data["avatar_id"] as? Int64) ?? (data["avatar_id"] as? NSNumber)?.int64Value,
            selected_avatar: selectedAvatar
        )
    }

    /// Parse AgentMessage from dictionary
    private func parseMessage(from data: [String: Any]) -> AgentMessage? {
        guard let id = data["id"] as? String else {
            return nil
        }

        let role = data["role"] as? String ?? "assistant"
        var contentStr = ""
        var thinkingStr: String? = nil
        var createdAt: String? = data["created_at"] as? String

        // Also check metadata for created_at if not found at top level
        if createdAt == nil, let metadata = data["metadata"] as? [String: Any] {
            createdAt = metadata["created_at"] as? String
        }

        // Handle content structure from Go backend
        var toolsData: [Any] = []
        if let contentData = data["content"] {
            if let contentDict = contentData as? [String: Any] {
                let contentType = contentDict["type"] as? String ?? "unknown"

                // Handle different message structure types
                if contentType == "user" {
                    // For user messages, the actual content is in the "content" field
                    if let userContent = contentDict["content"] {
                        if let contentStr_val = userContent as? String {
                            contentStr = contentStr_val
                        } else {
                            contentStr = String(describing: userContent)
                        }
                    }
                } else if contentType == "tool_result" {
                    // Tool result messages should be hidden - they're responses to tool calls, not user-visible
                    contentStr = ""
                } else if contentType == "assistant" {
                    // For assistant messages, extract from "text" field
                    if let text = contentDict["text"] {
                        if let textStr = text as? String {
                            contentStr = textStr
                        } else if let textArray = text as? [String] {
                            contentStr = textArray.joined(separator: "\n")
                        } else if let textArray = text as? [Any] {
                            // Handle [Any] which might contain strings
                            let strings = textArray.compactMap { $0 as? String }
                            contentStr = strings.joined(separator: "\n")
                        }
                    }
                } else {
                    // Unknown content type - try to extract sensible content
                    if let text = contentDict["text"] as? String {
                        contentStr = text
                    } else if let content = contentDict["content"] as? String {
                        contentStr = content
                    } else {
                        contentStr = ""
                    }
                }

                // Extract thinking content (could be string or array of strings)
                if let thinking = contentDict["thinking"] {
                    if let thinkingStrValue = thinking as? String {
                        thinkingStr = thinkingStrValue
                    } else if let thinkingArray = thinking as? [String] {
                        thinkingStr = thinkingArray.joined(separator: "\n")
                    }
                }

                // Extract tools from the same contentDict
                if let tools = contentDict["tools"] as? [Any] {
                    toolsData = tools
                }

                // Note: Removed fallback that created "[{type}]" since we now properly handle
                // both user and assistant message structures above
            } else if let contentStrValue = contentData as? String {
                // Fallback for simple string content (from REST API)
                contentStr = contentStrValue
            } else {
                contentStr = String(describing: contentData)
            }
        } else {
            contentStr = ""
        }

        // Extract thinking from top-level if not found in content
        if thinkingStr == nil, let topThinking = data["thinking"] as? String {
            thinkingStr = topThinking
        }

        // Decode tools array into ToolUseInfo objects
        var decodedTools: [ToolUseInfo]? = nil
        if !toolsData.isEmpty {
            do {
                let toolsJSON = try JSONSerialization.data(withJSONObject: toolsData)
                let decoder = JSONDecoder()
                decodedTools = try decoder.decode([ToolUseInfo].self, from: toolsJSON)
            } catch {
                decodedTools = nil
            }
        }

        let agentMessage = AgentMessage(
            id: id,
            role: role,
            content: ContentBlockOrString.string(contentStr),
            created_at: createdAt,
            thinking: thinkingStr,
            tools: decodedTools
        )

        return agentMessage
    }

    // MARK: - User Actions

    /// Send a message to the selected session
    func sendMessage(_ prompt: String) {
        guard let sessionId = selectedSessionId else {
            errorMessage = "No session selected"
            return
        }

        _ = webSocket.sendPrompt(sessionId, prompt: prompt)
    }

    /// Interrupt the current session
    func interruptSession() {
        guard let sessionId = selectedSessionId else {
            errorMessage = "No session selected"
            return
        }

        _ = webSocket.interruptSession(sessionId)
    }

    // MARK: - Optimistic Message Mutations

    /// Add an optimistic user message immediately (before server confirms).
    /// Gives instant UI feedback while the message travels to the backend.
    func addOptimisticUserMessage(_ text: String, sessionId: String) {
        let message = AgentMessage(
            id: UUID().uuidString,
            role: "user",
            content: .string(text),
            created_at: ISO8601DateFormatter().string(from: Date()),
            thinking: nil,
            tools: nil
        )

        var updatedDict = sessionMessages
        var sessionMsgs = updatedDict[sessionId] ?? []
        sessionMsgs.append(message)
        updatedDict[sessionId] = sessionMsgs
        sessionMessages = updatedDict
    }

    /// Add a system message (e.g., "Session interrupted by user").
    func addSystemMessage(_ text: String, sessionId: String) {
        let message = AgentMessage(
            id: UUID().uuidString,
            role: "system",
            content: .string(text),
            created_at: ISO8601DateFormatter().string(from: Date()),
            thinking: nil,
            tools: nil
        )

        var updatedDict = sessionMessages
        var sessionMsgs = updatedDict[sessionId] ?? []
        sessionMsgs.append(message)
        updatedDict[sessionId] = sessionMsgs
        sessionMessages = updatedDict
    }
}

// MARK: - Preview Helper

#if DEBUG
extension SessionsWebSocketViewModel {
    static let preview: SessionsWebSocketViewModel = {
        let vm = SessionsWebSocketViewModel.shared
        vm.sessions = [
            AgentSession(
                id: "session-1",
                status: "processing",
                model_name: "claude-sonnet-4-5",
                provider: "claude",
                git_branch: "main",
                working_directory: "/path/to/project",
                created_at: ISO8601DateFormatter().string(from: Date()),
                updated_at: ISO8601DateFormatter().string(from: Date()),
                message_count: 5,
                cost: 0.05,
                project_id: "lp_example",
                selected_avatar_id: 19,
                selected_avatar: Avatar(id: 19, theme_id: 1, name: "Bebe Stevens", type: "preset", image_path: "bebe", image_url: nil, style: nil, seed: nil, color: "#FF69B4", created_at: "2024-01-01T00:00:00Z")
            )
        ]

        vm.sessionMessages["session-1"] = [
            AgentMessage(
                id: "msg-1",
                role: "user",
                content: .string("Hello"),
                created_at: nil,
                thinking: nil,
                tools: nil
            ),
            AgentMessage(
                id: "msg-2",
                role: "assistant",
                content: .string("Hi! How can I help?"),
                created_at: nil,
                thinking: nil,
                tools: nil
            )
        ]

        return vm
    }()
}
#endif
