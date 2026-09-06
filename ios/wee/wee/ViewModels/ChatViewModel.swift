//
//  ChatViewModel.swift
//  wee
//
//  ViewModel for ChatView - manages state and business logic
//

import SwiftUI
import Combine

@MainActor
class ChatViewModel: ObservableObject {
    // MARK: - Dependencies
    private let webSocketViewModel: SessionsWebSocketViewModel
    private let apiClient: WeeAPIClient
    private let avatarManager: AvatarManager

    // MARK: - Published Properties
    @Published var messages: [AgentMessage] = []
    @Published var userInput = ""
    @Published var isLoading = false
    @Published var showImageViewer = false
    @Published var selectedImageDataUrl: String?
    @Published var selectedMessageId: String?
    @Published var showMessageDetail = false
    @Published var currentSessionStatus: SessionStatus = .idle
    @Published var currentSession: AgentSession?
    @Published var expandedMessageIds: Set<String> = []

    // Navigation state
    @Published var navigateToJust = false
    @Published var navigateToGit = false
    @Published var navigateToStats = false
    @Published var navigateToZen = false

    // MARK: - Read-only Properties
    let sessionID: String
    let sessionTitle: String

    var projectId: String? {
        currentSession?.project_id ?? webSocketViewModel.sessions.first(where: { $0.id == sessionID })?.project_id
    }

    var hasSelectedMessage: Bool {
        selectedMessageId != nil
    }

    var selectedMessage: AgentMessage? {
        guard let id = selectedMessageId else { return nil }
        return messages.first(where: { $0.id == id })
    }

    // MARK: - Initialization
    init(
        sessionID: String,
        sessionTitle: String,
        webSocketViewModel: SessionsWebSocketViewModel = .shared,
        apiClient: WeeAPIClient = .shared,
        avatarManager: AvatarManager = .shared
    ) {
        self.sessionID = sessionID
        self.sessionTitle = sessionTitle
        self.webSocketViewModel = webSocketViewModel
        self.apiClient = apiClient
        self.avatarManager = avatarManager
    }

    // MARK: - Lifecycle
    func initialize() async {
        if !webSocketViewModel.webSocketConnected {
            await webSocketViewModel.startMonitoring()
        }
        webSocketViewModel.selectSession(sessionID)

        await loadSessionDetails()
        await loadMessages()

        // Sync initial status
        if let status = webSocketViewModel.sessionStatus[sessionID] {
            self.currentSessionStatus = SessionStatus(status)
        } else if let session = currentSession ?? webSocketViewModel.sessions.first(where: { $0.id == sessionID }) {
            self.currentSessionStatus = SessionStatus(session.status)
        }
    }

    func cleanup() {
        // Don't disconnect WebSocket — keep it alive for the app lifecycle.
        // Just deselect the session so we don't keep loading messages for it.
        if webSocketViewModel.selectedSessionId == sessionID {
            webSocketViewModel.selectedSessionId = nil
        }
    }

    // MARK: - Data Loading
    private func loadSessionDetails() async {
        do {
            let sessions = try await apiClient.getAgentSessions()
            if let session = sessions.first(where: { $0.id == sessionID }) {
                self.currentSession = session
                #if DEBUG
                print("✅ Loaded session details for \(sessionID): avatar_id=\(session.selected_avatar_id ?? 0)")
                #endif
            }
        } catch {
            #if DEBUG
            print("❌ Failed to load session details: \(error.localizedDescription)")
            #endif
        }
    }

    func loadMessages() async {
        isLoading = true
        defer { isLoading = false }

        do {
            let loadedMessages = try await apiClient.getSessionMessages(sessionID: sessionID, limit: 50)
            await MainActor.run {
                var updatedMessages = webSocketViewModel.sessionMessages
                updatedMessages[sessionID] = loadedMessages
                webSocketViewModel.sessionMessages = updatedMessages
                self.messages = loadedMessages
            }
        } catch {
            // Messages will be loaded via WebSocket fallback
        }
    }

    func syncMessages(from messagesDict: [String: [AgentMessage]]) {
        if let sessionMessages = messagesDict[sessionID] {
            self.messages = sessionMessages
        }
    }

    func updateStatus(_ status: String) {
        let newStatus = SessionStatus(status)
        if newStatus != currentSessionStatus {
            currentSessionStatus = newStatus
        }
    }

    // MARK: - User Actions
    func sendMessage() {
        let messageText = userInput.trimmingCharacters(in: .whitespaces)
        guard !messageText.isEmpty else { return }

        // Clear input immediately
        userInput = ""

        // Create optimistic message
        let optimisticMessage = AgentMessage(
            id: UUID().uuidString,
            role: "user",
            content: .string(messageText),
            created_at: ISO8601DateFormatter().string(from: Date()),
            thinking: nil,
            tools: nil
        )

        // Add to local messages
        messages.append(optimisticMessage)

        // Update ViewModel for consistency
        var updatedDict = webSocketViewModel.sessionMessages
        var sessionMessages = updatedDict[sessionID] ?? []
        sessionMessages.append(optimisticMessage)
        updatedDict[sessionID] = sessionMessages
        webSocketViewModel.sessionMessages = updatedDict

        // Send via WebSocket
        webSocketViewModel.sendMessage(messageText)
    }

    func handleInterrupt() {
        webSocketViewModel.addSystemMessage("Session interrupted by user", sessionId: sessionID)
        webSocketViewModel.interruptSession()
    }

    func refreshMessages() async {
        await loadMessages()
    }

    // MARK: - Message Detail
    func selectMessage(_ id: String) {
        selectedMessageId = id
        showMessageDetail = true
    }

    func dismissMessageDetail() {
        showMessageDetail = false
        selectedMessageId = nil
    }

    // MARK: - Image Viewer
    func showImage(_ dataUrl: String) {
        selectedImageDataUrl = dataUrl
        showImageViewer = true
    }

    func dismissImageViewer() {
        showImageViewer = false
        selectedImageDataUrl = nil
    }

    // MARK: - Message Expansion
    func toggleMessageExpansion(_ id: String) {
        withAnimation(.easeInOut(duration: 0.2)) {
            if expandedMessageIds.contains(id) {
                expandedMessageIds.remove(id)
            } else {
                expandedMessageIds.insert(id)
            }
        }
    }

    func isMessageExpanded(_ id: String) -> Bool {
        expandedMessageIds.contains(id)
    }

    // MARK: - Navigation
    func navigate(to destination: NavigationDestination) {
        switch destination {
        case .just: navigateToJust = true
        case .git: navigateToGit = true
        case .stats: navigateToStats = true
        case .zen: navigateToZen = true
        }
    }

    func resetNavigation(_ destination: NavigationDestination) {
        switch destination {
        case .just: navigateToJust = false
        case .git: navigateToGit = false
        case .stats: navigateToStats = false
        case .zen: navigateToZen = false
        }
    }

    // MARK: - Message Filtering
    func shouldHideMessage(_ message: AgentMessage) -> Bool {
        // Quick exit for non-assistant messages
        if message.role != "assistant" {
            return false
        }

        let content = message.content.textContent.lowercased().trimmingCharacters(in: .whitespaces)
        let hasTools = message.getTools().count > 0
        let hasThinking = message.thinking != nil && !message.thinking!.isEmpty
        let hasContent = !content.isEmpty

        // Hide assistant messages with NO visible content AND no tools AND no thinking
        if !hasContent && !hasThinking && !hasTools {
            return true
        }

        // Hide tool result messages
        if content.contains("\"tool_use_id\"") && content.contains("tool_result") {
            return true
        }

        // Hide messages that are just tool result indicators
        if content == "[result]" {
            return true
        }

        // Hide messages that are just showing tool names in brackets
        if content.hasPrefix("[") && content.hasSuffix("]") && content.count < 50 {
            let inner = String(content.dropFirst().dropLast()).lowercased()
            let hiddenMarkers = ["result", "output", "response", "tool result", "execution result"]
            if hiddenMarkers.contains(inner) {
                return true
            }
        }

        return false
    }

    // MARK: - Avatar Helpers
    var currentAvatar: Avatar? {
        let session = currentSession ?? webSocketViewModel.sessions.first(where: { $0.id == sessionID })
        return session?.selected_avatar ?? (session?.selected_avatar_id.flatMap { avatarManager.getAvatar(id: $0) })
    }

    var characterName: String {
        currentAvatar?.name ?? "Unknown"
    }

    var characterColor: Color {
        currentAvatar?.color?.parseHexColor() ?? .defaultAvatar
    }

    var avatarImage: UIImage? {
        currentAvatar.flatMap { avatarManager.getAvatarImage(id: $0.id) }
    }

    func loadAvatarImageIfNeeded() async {
        guard let avatar = currentAvatar else { return }
        if avatarManager.getAvatarImage(id: avatar.id) == nil {
            await avatarManager.loadAvatarImageIfNeeded(avatar: avatar)
        }
    }
}

// MARK: - Navigation Destinations

enum NavigationDestination {
    case just
    case git
    case stats
    case zen
}
