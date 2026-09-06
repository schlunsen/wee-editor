//
//  WebSocketService.swift
//  wee
//
//  Real-time WebSocket service for agent session updates
//  Production-ready with ping/pong, network monitoring, and proper reconnection
//
//  Uses Starscream for WebSocket transport (HTTP/1.1 compatible, avoids
//  URLSessionWebSocketTask HTTP/2 upgrade issues through proxies like ngrok).
//
//  Key features:
//  - Ping/pong keep-alive every 30s (detects silent drops)
//  - Network monitoring with NWPathMonitor (auto-reconnect)
//  - Task-based reconnection with exponential backoff + jitter
//  - Typed error handling with recovery strategies
//

import Foundation
import Combine
import Network
import Starscream
import UIKit

// MARK: - Notifications

extension Notification.Name {
    /// Posted when the WebSocket receives an auth_error (session expired/invalid).
    /// Observers should reset auth state and redirect to login.
    static let sessionExpired = Notification.Name("sessionExpired")
}

// MARK: - WebSocket Message Types

/// Base message structure for all WebSocket communications
/// Matches the backend's AgentMessageResponse and related message types
struct WebSocketMessage: Codable {
    let type: String
    let session_id: String?

    // For agent_message, agent_thinking, agent_tool_use, etc. (Claude SDK messages)
    let id: String?          // Message ID from backend
    let content: AnyCodable? // Message content (text, thinking, tools, etc.)
    let role: String?        // Message role (user, assistant, system)
    let metadata: AnyCodable? // Optional metadata

    // For legacy data wrapper (for backward compatibility)
    let data: AnyCodable?

    // For error messages
    let message: String?
    let error: String?

    // For status/state messages
    let status: String?
    let sessions: [AnyCodable]?

    enum CodingKeys: String, CodingKey {
        case type, session_id, id, content, role, metadata, data, message, error, status, sessions
    }
}

// MARK: - WebSocket Error Handling

/// Typed WebSocket errors with recovery classification
enum WebSocketError: Error, LocalizedError {
    case connectionFailed(String)
    case messageSendFailed(String)
    case messageDecodeFailed(String)
    case networkLost
    case timeout
    case invalidURL
    case unauthorized
    case serverError(String)

    var errorDescription: String? {
        switch self {
        case .connectionFailed(let msg):
            return "Connection failed: \(msg)"
        case .messageSendFailed(let msg):
            return "Failed to send message: \(msg)"
        case .messageDecodeFailed(let msg):
            return "Failed to decode message: \(msg)"
        case .networkLost:
            return "Network connection lost"
        case .timeout:
            return "Connection timeout - no response for 30 seconds"
        case .invalidURL:
            return "Invalid WebSocket URL"
        case .unauthorized:
            return "Unauthorized - check API key"
        case .serverError(let msg):
            return "Server error: \(msg)"
        }
    }

    var isRecoverable: Bool {
        switch self {
        case .networkLost, .timeout, .connectionFailed:
            return true
        case .unauthorized, .invalidURL, .messageDecodeFailed:
            return false
        case .messageSendFailed, .serverError:
            return true
        }
    }
}

// MARK: - Network Monitor

/// Monitors network connectivity status and triggers reconnection when network is restored
class NetworkMonitor: NSObject, ObservableObject {
    @Published var isConnected = true

    private let monitor = NWPathMonitor()
    private let queue = DispatchQueue(label: "NetworkMonitor")
    private var isMonitoring = false

    override init() {
        super.init()
        setupMonitoring()
    }

    private func setupMonitoring() {
        monitor.pathUpdateHandler = { [weak self] path in
            DispatchQueue.main.async {
                let wasConnected = self?.isConnected ?? false
                let isNowConnected = path.status == .satisfied

                if wasConnected != isNowConnected {
                    self?.isConnected = isNowConnected
                }
            }
        }

        monitor.start(queue: queue)
        isMonitoring = true
    }

    deinit {
        monitor.cancel()
    }
}

// MARK: - WebSocket Event Handlers

/// Protocol for handling WebSocket events
protocol WebSocketEventHandler: AnyObject {
    func onSessionCreated(_ data: [String: Any])
    func onSessionUpdated(_ data: [String: Any])
    func onAgentMessage(_ data: [String: Any])
    func onAgentThinking(_ data: [String: Any])
    func onAgentToolUse(_ data: [String: Any])
    func onAgentError(_ data: [String: Any])
    func onPermissionRequest(_ data: [String: Any])
    func onPermissionAcknowledged(_ data: [String: Any])
    func onSessionsList(_ data: [String: Any])
    func onMessagesLoaded(_ data: [String: Any])
    func onSessionDeleted(_ data: [String: Any])
    func onConnectionStatusChanged(_ isConnected: Bool)
    func onError(_ error: String)
}

// MARK: - Starscream Delegate Bridge

/// Bridges Starscream's nonisolated delegate callbacks to the @MainActor WebSocketService.
///
/// Starscream delivers events on a background queue. This class receives them without
/// actor isolation and dispatches to MainActor so WebSocketService state is always
/// mutated on the main thread.
///
/// IMPORTANT: When we tear down a stale socket and create a new one (forceReconnect),
/// the old socket may still fire events (e.g. .cancelled, .error) on a background queue.
/// These arrive *after* the new socket has connected and would destroy the new connection.
/// We use `invalidated` to make the old bridge ignore late-arriving events.
nonisolated
private class WebSocketDelegateBridge: NSObject, Starscream.WebSocketDelegate {
    weak var service: WebSocketService?

    /// When true, this bridge has been replaced by a new one and should ignore all events.
    /// Accessed from background threads — use atomic flag.
    private let _invalidated = NSLock()
    private var _isInvalidated = false
    var isInvalidated: Bool {
        get { _invalidated.lock(); defer { _invalidated.unlock() }; return _isInvalidated }
        set { _invalidated.lock(); defer { _invalidated.unlock() }; _isInvalidated = newValue }
    }

    init(service: WebSocketService) {
        self.service = service
    }

    /// Mark this bridge as stale — all future events will be dropped.
    func invalidate() {
        isInvalidated = true
        service = nil
    }

    nonisolated func didReceive(event: Starscream.WebSocketEvent, client: any Starscream.WebSocketClient) {
        // Drop events from stale sockets (old connection being torn down)
        guard !isInvalidated else {
            return
        }
        let captured = service
        Task { @MainActor in
            captured?.handleStarscreamEvent(event)
        }
    }
}

// MARK: - WebSocket Service

/// Service managing WebSocket connection to the agent server
///
/// Production-ready implementation with:
/// - Starscream WebSocket (HTTP/1.1, proxy-friendly)
/// - Ping/pong keep-alive (prevents silent disconnects)
/// - Network monitoring (auto-reconnect when network changes)
/// - Exponential backoff with jitter (prevents server overload)
/// - Typed error handling (recovery classification)
@MainActor
class WebSocketService: NSObject, ObservableObject {
    // MARK: - Published Properties

    @Published var isConnected = false
    @Published var isAuthenticated = false
    @Published var lastError: String?
    @Published var lastErrorType: WebSocketError?

    // Connection state guards
    private var isConnecting = false
    private var isReconnecting = false

    // MARK: - Constants

    private let PING_INTERVAL: UInt64 = 30_000_000_000  // 30 seconds in nanoseconds
    private let INITIAL_RECONNECT_DELAY: TimeInterval = 1.0
    private let MAX_RECONNECT_DELAY: TimeInterval = 60.0
    private let MAX_RECONNECT_ATTEMPTS = 10

    // MARK: - Private Properties

    private var socket: Starscream.WebSocket?
    private var delegateBridge: WebSocketDelegateBridge?
    private let apiClient = WeeAPIClient.shared
    private let networkMonitor = NetworkMonitor()
    private var reconnectAttempts = 0
    private var reconnectDelay: TimeInterval = 1.0

    // Task management
    private var pingTask: Task<Void, Never>?
    private var networkMonitorTask: Task<Void, Never>?

    // Cached server API key (avoids extra HTTP round trip on reconnect)
    private var cachedServerAPIKey: String?

    // Track which sessions are subscribed so we can re-subscribe after reconnect
    private var subscribedSessionIds: Set<String> = []

    // Pong timeout tracking
    private var lastPongTime: Date = Date()
    private let PONG_TIMEOUT: TimeInterval = 60  // reconnect if no pong in 60s

    // Messages queued before authentication completed
    private var pendingMessages: [[String: Any]] = []

    // App lifecycle observers
    private var didBecomeActiveObserver: NSObjectProtocol?
    private var willResignActiveObserver: NSObjectProtocol?

    /// Event handler for WebSocket messages
    weak var eventHandler: WebSocketEventHandler?

    // MARK: - Singleton

    static let shared = WebSocketService()

    private override init() {
        super.init()
        setupAppLifecycleObservers()
    }

    private func setupAppLifecycleObservers() {
        // Remove existing observers first
        if let observer = didBecomeActiveObserver {
            NotificationCenter.default.removeObserver(observer)
        }
        if let observer = willResignActiveObserver {
            NotificationCenter.default.removeObserver(observer)
        }

        didBecomeActiveObserver = NotificationCenter.default.addObserver(
            forName: UIApplication.didBecomeActiveNotification,
            object: nil,
            queue: .main
        ) { [weak self] _ in
            guard let self = self else { return }
            Task { @MainActor in
                // iOS kills TCP connections almost immediately when the app is
                // suspended (even for very short sleeps like pressing the lock
                // button). Starscream won't fire a disconnect event while
                // suspended, so `isConnected` stays true even though the socket
                // is dead.
                //
                // The safest strategy is to always tear down and reconnect.
                // This adds ~1-2s of reconnection time but guarantees we never
                // get stuck with a zombie socket showing "connection lost".
                if self.isConnected || self.isConnecting {
                    #if DEBUG
                    print("[WS] app became active — force reconnecting (was connected=\(self.isConnected), connecting=\(self.isConnecting))")
                    #endif
                    self.forceReconnect()
                } else {
                    // Not connected and not trying — reconnect now
                    #if DEBUG
                    print("[WS] app became active — not connected, reconnecting")
                    #endif
                    self.reconnectAttempts = 0
                    self.reconnectDelay = self.INITIAL_RECONNECT_DELAY
                    await self.connect()
                }
            }
        }

        willResignActiveObserver = NotificationCenter.default.addObserver(
            forName: UIApplication.willResignActiveNotification,
            object: nil,
            queue: .main
        ) { [weak self] _ in
            guard let self = self else { return }
            Task { @MainActor in
                // Cancel the ping task — it can't run while the app is suspended
                // and its sleep timer will be stale when the app resumes.
                self.pingTask?.cancel()
                self.pingTask = nil
                #if DEBUG
                print("[WS] app will resign active — stopped ping keep-alive")
                #endif
            }
        }
    }

    /// Tear down a stale connection and reconnect immediately.
    /// Used when the app resumes and we detect the socket is dead.
    private func forceReconnect() {
        // Tear down stale socket
        pingTask?.cancel()
        pingTask = nil
        networkMonitorTask?.cancel()
        networkMonitorTask = nil

        // CRITICAL: Invalidate the old delegate bridge BEFORE disconnecting.
        // When we call socket.disconnect(), Starscream queues .cancelled/.error
        // events on a background thread. These arrive via the bridge's
        // didReceive() and would set isConnected=false on the NEW connection.
        // Invalidating the bridge makes it drop all late-arriving events.
        delegateBridge?.invalidate()
        delegateBridge = nil

        socket?.disconnect(closeCode: CloseCode.goingAway.rawValue)
        socket = nil

        isConnected = false
        isAuthenticated = false

        // Reset backoff so we reconnect immediately
        reconnectAttempts = 0
        reconnectDelay = INITIAL_RECONNECT_DELAY
        isReconnecting = false

        eventHandler?.onConnectionStatusChanged(false)

        // Reconnect immediately
        Task {
            await connect()
        }
    }

    deinit {
        // Remove app lifecycle observers
        if let observer = didBecomeActiveObserver {
            NotificationCenter.default.removeObserver(observer)
        }
        if let observer = willResignActiveObserver {
            NotificationCenter.default.removeObserver(observer)
        }

        // Cancel tasks directly -- disconnect() cannot be called from deinit
        // because it requires @MainActor isolation
        pingTask?.cancel()
        networkMonitorTask?.cancel()
        socket?.disconnect(closeCode: CloseCode.goingAway.rawValue)
    }

    // MARK: - Connection Management

    /// Connect to the agent WebSocket endpoint
    ///
    /// IMPORTANT: This function waits for the connection to actually establish,
    /// ensuring the WebSocket is ready before returning. This prevents race conditions
    /// where selectSession() is called before the WebSocket is fully established.
    func connect() async {
        guard !isConnected && !isConnecting else {
            return
        }

        isConnecting = true
        defer { isConnecting = false }

        guard let baseURL = apiClient.baseURL else {
            handleError(.invalidURL)
            return
        }

        // Check if we can try session-based pre-authentication first.
        // When we have a session token, the server's SessionAuthMiddleware may authenticate
        // the WebSocket upgrade request via cookie -- avoiding the API key fetch.
        let hasSessionAuth = apiClient.sessionToken != nil

        // Always resolve the API key (from cache/keychain) so we have a fallback
        // if session pre-auth doesn't work (e.g. ngrok strips cookies).
        var serverAPIKey: String?
        if let cachedKey = cachedServerAPIKey, !cachedKey.isEmpty {
            serverAPIKey = cachedKey
        } else if let storedKey = try? SecureStorage.shared.retrieve(key: "apiKey"), !storedKey.isEmpty {
            serverAPIKey = storedKey
        } else {
            // Fetch from server -- needed as fallback even with session auth
            do {
                let key = try await apiClient.fetchAPIKey()
                cachedServerAPIKey = key
                try? SecureStorage.shared.store(value: key, forKey: "apiKey")
                serverAPIKey = key
                #if DEBUG
                print("[WS] fetched API key successfully")
                #endif
            } catch {
                #if DEBUG
                print("[WS] fetchAPIKey failed: \(error)")
                #endif
                // Only fail if we also don't have session auth
                if !hasSessionAuth {
                    handleError(WebSocketError.unauthorized)
                    return
                }
            }
        }

        // Build WebSocket URL
        let wsScheme = baseURL.scheme == "https" ? "wss" : "ws"
        let host = baseURL.host ?? "localhost"
        let portString: String
        if let port = baseURL.port {
            portString = ":\(port)"
        } else {
            // For localhost, default to 3333; for remote hosts (ngrok), don't add port
            portString = host == "localhost" || host == "127.0.0.1" ? ":3333" : ""
        }

        let urlString = "\(wsScheme)://\(host)\(portString)/agent/ws"
        guard let wsURL = URL(string: urlString) else {
            handleError(.invalidURL)
            return
        }

        // Build the URLRequest for Starscream
        var request = URLRequest(url: wsURL)
        request.timeoutInterval = 30

        // Include session token as cookie for the HTTP upgrade request
        // so SessionAuthMiddleware can identify the authenticated user
        if let sessionToken = apiClient.sessionToken {
            request.setValue("session_token=\(sessionToken)", forHTTPHeaderField: "Cookie")
        }

        // Create Starscream WebSocket
        // For development with self-signed certificates on localhost, pass certPinner: nil
        let isLocalhost = host == "localhost" || host == "127.0.0.1" || host == "::1"
        let ws = Starscream.WebSocket(request: request, certPinner: isLocalhost ? nil : FoundationSecurity())

        // Set up delegate bridge
        let bridge = WebSocketDelegateBridge(service: self)
        ws.delegate = bridge
        self.delegateBridge = bridge
        self.socket = ws

        // Connect
        ws.connect()
        #if DEBUG
        print("[WS] connecting to \(urlString), hasSessionAuth=\(hasSessionAuth), hasAPIKey=\(serverAPIKey != nil)")
        #endif

        // Wait for the connection to establish (or fail)
        // The delegate bridge will set isConnected = true on .connected event
        for _ in 0..<60 {
            if isConnected { break }
            if lastErrorType != nil { return }  // Connection failed
            try? await Task.sleep(nanoseconds: 50_000_000)  // 50ms, up to 3s total
        }

        guard isConnected else {
            handleError(.connectionFailed("Timed out waiting for WebSocket connection"))
            return
        }

        if hasSessionAuth {
            // Try session-based pre-auth first: the server may have already authenticated
            // us via the cookie on the HTTP upgrade -- wait briefly for auth_success.
            for _ in 0..<20 {
                if isAuthenticated { break }
                try? await Task.sleep(nanoseconds: 50_000_000)  // 50ms
            }
        }

        // If pre-auth didn't work, fall back to API key or session token auth
        if !isAuthenticated, let apiKey = serverAPIKey {
            _ = sendRaw([
                "type": "auth",
                "token": apiKey
            ])

            for _ in 0..<50 {
                if isAuthenticated { break }
                try? await Task.sleep(nanoseconds: 100_000_000)  // 100ms
            }
        }

        // Last resort: fetch API key from server if we don't have one yet
        if !isAuthenticated, serverAPIKey == nil, isConnected {
            do {
                let key = try await apiClient.fetchAPIKey()
                cachedServerAPIKey = key
                try? SecureStorage.shared.store(value: key, forKey: "apiKey")

                _ = sendRaw([
                    "type": "auth",
                    "token": key
                ])

                for _ in 0..<50 {
                    if isAuthenticated { break }
                    try? await Task.sleep(nanoseconds: 100_000_000)  // 100ms
                }
            } catch {
                // API key fetch failed
            }
        }

        // Last resort: send session token directly as auth token.
        // The server validates it against the session store.
        if !isAuthenticated, let sessionToken = apiClient.sessionToken, isConnected {
            #if DEBUG
            print("[WS] trying session token auth")
            #endif
            _ = sendRaw([
                "type": "auth",
                "token": sessionToken
            ])

            for _ in 0..<30 {
                if isAuthenticated { break }
                try? await Task.sleep(nanoseconds: 100_000_000)  // 100ms
            }
        }

        #if DEBUG
        print("[WS] auth result -- authenticated=\(isAuthenticated), connected=\(isConnected)")
        #endif

        if isAuthenticated {
            eventHandler?.onConnectionStatusChanged(true)
        } else if isConnected {
            // Auth didn't complete but connection is alive -- notify anyway
            // so callers can still attempt operations
            eventHandler?.onConnectionStatusChanged(true)
        }
    }

    /// Disconnect from WebSocket and cancel all tasks
    func disconnect() {
        isConnected = false
        isAuthenticated = false
        pendingMessages.removeAll()

        // Cancel running tasks
        pingTask?.cancel()
        pingTask = nil

        networkMonitorTask?.cancel()
        networkMonitorTask = nil

        // Invalidate bridge before disconnecting to prevent stale events
        delegateBridge?.invalidate()
        delegateBridge = nil

        // Close WebSocket connection
        socket?.disconnect(closeCode: CloseCode.goingAway.rawValue)
        socket = nil

        eventHandler?.onConnectionStatusChanged(false)
    }

    // MARK: - Starscream Event Handling

    /// Handle events from the Starscream delegate bridge.
    /// Called on MainActor via the bridge's Task dispatch.
    func handleStarscreamEvent(_ event: Starscream.WebSocketEvent) {
        switch event {
        case .connected(let headers):
            #if DEBUG
            print("[WS] connected, headers: \(headers)")
            #endif
            isConnected = true
            reconnectAttempts = 0
            reconnectDelay = INITIAL_RECONNECT_DELAY
            lastErrorType = nil
            lastError = nil
            startPingKeepAlive()
            startNetworkMonitoring()

        case .disconnected(let reason, let code):
            #if DEBUG
            print("[WS] disconnected: \(reason) (code \(code))")
            #endif
            let wasConnected = isConnected
            isConnected = false
            isAuthenticated = false
            if wasConnected {
                eventHandler?.onConnectionStatusChanged(false)
                scheduleReconnect()
            }

        case .text(let text):
            #if DEBUG
            print("[WS] received: \(String(text.prefix(300)))")
            #endif
            Task {
                await handleMessage(text)
            }

        case .binary(let data):
            if let text = String(data: data, encoding: .utf8) {
                Task {
                    await handleMessage(text)
                }
            }

        case .ping(_):
            // Starscream automatically responds with pong
            break

        case .pong(_):
            // Pong received -- connection is alive
            lastPongTime = Date()

        case .viabilityChanged(let viable):
            #if DEBUG
            print("[WS] viability changed: \(viable)")
            #endif
            if !viable && isConnected {
                // Connection is no longer viable
                isConnected = false
                isAuthenticated = false
                eventHandler?.onConnectionStatusChanged(false)
                scheduleReconnect()
            }

        case .reconnectSuggested(let suggested):
            #if DEBUG
            print("[WS] reconnect suggested: \(suggested)")
            #endif
            if suggested && !isConnected {
                scheduleReconnect()
            }

        case .cancelled:
            #if DEBUG
            print("[WS] cancelled")
            #endif
            let wasConnected = isConnected
            isConnected = false
            isAuthenticated = false
            if wasConnected {
                eventHandler?.onConnectionStatusChanged(false)
            }

        case .error(let error):
            #if DEBUG
            print("[WS] error: \(String(describing: error))")
            #endif
            let wasConnected = isConnected
            isConnected = false
            isAuthenticated = false

            if let wsError = error as? WSError {
                handleError(.connectionFailed(wsError.message))
            } else if let error = error {
                handleError(.connectionFailed(error.localizedDescription))
            } else {
                handleError(.connectionFailed("Unknown WebSocket error"))
            }

            if wasConnected {
                eventHandler?.onConnectionStatusChanged(false)
            }
            scheduleReconnect()

        case .peerClosed:
            #if DEBUG
            print("[WS] peer closed")
            #endif
            let wasConnected = isConnected
            isConnected = false
            isAuthenticated = false
            if wasConnected {
                eventHandler?.onConnectionStatusChanged(false)
                scheduleReconnect()
            }
        }
    }

    // MARK: - Keep-Alive (Ping/Pong)

    /// Start ping keep-alive - sends ping every 30s to detect silent disconnects
    private func startPingKeepAlive() {
        pingTask?.cancel()
        lastPongTime = Date()

        pingTask = Task {
            while !Task.isCancelled && isConnected {
                do {
                    let sleepStart = Date()
                    try await Task.sleep(nanoseconds: PING_INTERVAL)
                    let actualSleep = Date().timeIntervalSince(sleepStart)

                    // If the actual sleep was significantly longer than expected,
                    // the app was likely suspended (e.g., phone went to sleep).
                    // In this case, the connection is almost certainly dead.
                    let expectedSleepSeconds = Double(PING_INTERVAL) / 1_000_000_000
                    if actualSleep > expectedSleepSeconds * 2 {
                        #if DEBUG
                        print("[WS] ping task slept \(Int(actualSleep))s (expected \(Int(expectedSleepSeconds))s) — app was likely suspended")
                        #endif
                    }

                    // Check for pong timeout — if no pong in 60s, connection is dead
                    let timeSinceLastPong = Date().timeIntervalSince(lastPongTime)
                    if timeSinceLastPong > PONG_TIMEOUT {
                        #if DEBUG
                        print("[WS] no pong received in \(Int(timeSinceLastPong))s, reconnecting")
                        #endif
                        forceReconnect()
                        return
                    }

                    if isConnected, let ws = socket {
                        ws.write(ping: Data())
                    }
                } catch {
                    // Sleep was cancelled -- exit loop
                    break
                }
            }
        }
    }

    // MARK: - Network Monitoring

    /// Monitor network status and reconnect when network is restored
    private func startNetworkMonitoring() {
        networkMonitorTask?.cancel()

        networkMonitorTask = Task {
            var previousStatus = networkMonitor.isConnected

            // Use AsyncSequence to monitor changes
            for await isNetworkConnected in networkMonitor.$isConnected.values {
                // Network transitioned from unavailable to available
                if !previousStatus && isNetworkConnected {
                    if !self.isConnected {
                        // Network is back — reset reconnect counter so we get fresh attempts
                        self.reconnectAttempts = 0
                        self.reconnectDelay = self.INITIAL_RECONNECT_DELAY
                        await self.connect()
                    }
                }

                // Network transitioned from available to unavailable
                if previousStatus && !isNetworkConnected {
                    if self.isConnected {
                        self.handleError(.networkLost)
                        self.isConnected = false
                        self.socket?.disconnect(closeCode: CloseCode.goingAway.rawValue)
                        self.socket = nil
                        self.delegateBridge = nil
                    }
                }

                previousStatus = isNetworkConnected
            }
        }
    }

    // MARK: - Reconnection

    /// Reconnect with exponential backoff + jitter (prevents thundering herd)
    private func scheduleReconnect() {
        // Don't reconnect if credentials were cleared (session expired)
        guard apiClient.sessionToken != nil || apiClient.baseURL != nil else {
            #if DEBUG
            print("[WS] skipping reconnect — no credentials")
            #endif
            return
        }

        // Don't stack reconnect attempts
        guard !isReconnecting else {
            #if DEBUG
            print("[WS] reconnect already in progress, skipping")
            #endif
            return
        }

        guard reconnectAttempts < MAX_RECONNECT_ATTEMPTS else {
            handleError(.connectionFailed("Max reconnection attempts reached"))
            return
        }

        isReconnecting = true
        reconnectAttempts += 1

        // Calculate delay with jitter (prevents all clients reconnecting simultaneously)
        let jitter = Double.random(in: 0.7...1.3)  // +/-30% random
        let delayWithJitter = reconnectDelay * jitter
        let finalDelay = min(delayWithJitter, MAX_RECONNECT_DELAY)

        Task {
            try? await Task.sleep(nanoseconds: UInt64(finalDelay * 1_000_000_000))

            if !Task.isCancelled && !isConnected {
                await connect()
            }
            isReconnecting = false
        }

        // Exponential backoff: 1s -> 2s -> 4s -> 8s -> 16s -> 32s -> 60s
        reconnectDelay = min(reconnectDelay * 2, MAX_RECONNECT_DELAY)
    }

    // MARK: - Message Handling

    /// Handle incoming WebSocket message
    private func handleMessage(_ text: String) async {

        do {
            guard let data = text.data(using: .utf8) else {
                handleError(.messageDecodeFailed("Failed to convert message to data"))
                return
            }

            let decoder = JSONDecoder()
            let message = try decoder.decode(WebSocketMessage.self, from: data)

            // Route message to appropriate handler
            let messageData = extractMessageData(message)

            switch message.type {
            case "auth_success":
                isAuthenticated = true

                // Re-subscribe to sessions that were active before reconnect
                for sessionId in subscribedSessionIds {
                    _ = subscribeToSession(sessionId: sessionId)
                }

                // Flush any messages that were queued before auth completed.
                // For send_prompt messages, we must re-subscribe to the session first
                // so the server associates our connection with the session.
                let queued = pendingMessages
                pendingMessages.removeAll()
                var subscribedSessions = Set<String>()
                for msg in queued {
                    if let sessionId = msg["session_id"] as? String,
                       !subscribedSessions.contains(sessionId) {
                        _ = subscribeToSession(sessionId: sessionId)
                        subscribedSessions.insert(sessionId)
                    }
                    _ = send(msg)
                }

            case "session_created":
                eventHandler?.onSessionCreated(messageData)

            case "session_updated":
                eventHandler?.onSessionUpdated(messageData)

            case "agent_message":
                eventHandler?.onAgentMessage(messageData)

            case "agent_thinking":
                eventHandler?.onAgentThinking(messageData)

            case "agent_tool_use":
                eventHandler?.onAgentToolUse(messageData)

            case "agent_error":
                eventHandler?.onAgentError(messageData)

            case "permission_request":
                eventHandler?.onPermissionRequest(messageData)

            case "permission_acknowledged":
                eventHandler?.onPermissionAcknowledged(messageData)

            case "sessions_list":
                eventHandler?.onSessionsList(messageData)

            case "messages_loaded":
                eventHandler?.onMessagesLoaded(messageData)

            case "session_deleted":
                eventHandler?.onSessionDeleted(messageData)

            case "session_subscribed":
                break

            case "auth_error":
                let errorMsg = message.error ?? "Authentication failed"
                #if DEBUG
                print("[WS] auth_error: \(errorMsg) — session token is invalid/expired, clearing credentials")
                #endif
                // Clear stale session token so the app redirects to login
                try? apiClient.clearCredentials()
                isAuthenticated = false
                // Notify the auth system that the session is invalid
                NotificationCenter.default.post(name: .sessionExpired, object: nil)
                handleError(.unauthorized)
                eventHandler?.onError("Session expired. Please log in again.")

            case "error":
                let errorMsg = message.message ?? message.error ?? "Unknown error"
                let wsError = WebSocketError.serverError(errorMsg)
                handleError(wsError)
                eventHandler?.onError(errorMsg)

            default:
                break
            }

        } catch {
            handleError(.messageDecodeFailed(error.localizedDescription))
        }
    }

    /// Convert AnyCodable nested value to regular Swift types
    ///
    /// AnyCodable stores arrays as [AnyCodable] and dicts as [String: AnyCodable]
    /// This recursively converts them to [Any] and [String: Any]
    private func convertAnyCodableValue(_ value: AnyCodable) -> Any {
        switch value.value {
        case let arr as [AnyCodable]:
            // Convert [AnyCodable] to [Any]
            return arr.map { convertAnyCodableValue($0) }
        case let dict as [String: AnyCodable]:
            // Convert [String: AnyCodable] to [String: Any]
            var result: [String: Any] = [:]
            for (key, val) in dict {
                result[key] = convertAnyCodableValue(val)
            }
            return result
        default:
            // Primitive types (String, Int, Bool, etc.)
            return value.value
        }
    }

    /// Extract message data from WebSocket message
    ///
    /// The backend sends messages in two formats:
    /// 1. Agent messages: top-level fields (id, content, role, metadata)
    /// 2. Legacy/status messages: wrapped in 'data' field
    ///
    /// This function merges both into a single dictionary for handlers
    private func extractMessageData(_ message: WebSocketMessage) -> [String: Any] {
        var data: [String: Any] = [:]

        // Start with legacy 'data' wrapper if present (for backward compatibility)
        if let legacyData = message.data?.value as? [String: Any] {
            data = legacyData
        }

        // Add top-level agent message fields (these take precedence)
        if let id = message.id {
            data["id"] = id
        }
        if let content = message.content?.value {
            // Convert AnyCodable values to regular values (for nested structures)
            if let contentDict = content as? [String: AnyCodable] {
                var convertedDict: [String: Any] = [:]
                for (key, anyValue) in contentDict {
                    convertedDict[key] = convertAnyCodableValue(anyValue)
                }
                data["content"] = convertedDict
            } else {
                data["content"] = content
            }
        }
        if let role = message.role {
            data["role"] = role
        }
        if let metadata = message.metadata?.value {
            data["metadata"] = metadata
        }
        if let status = message.status {
            data["status"] = status
        }
        if let sessions = message.sessions {
            data["sessions"] = sessions.compactMap { $0.value }
        }

        // Always ensure session_id is included
        if let sessionId = message.session_id {
            data["session_id"] = sessionId
        }

        // Add error/message fields if present
        if let msg = message.message {
            data["message"] = msg
        }
        if let error = message.error {
            data["error"] = error
        }

        return data
    }

    // MARK: - Error Handling

    /// Handle WebSocket errors with recovery classification
    private func handleError(_ error: WebSocketError) {
        lastErrorType = error
        lastError = error.errorDescription
    }

    // MARK: - Message Sending

    /// Send a raw message (bypasses auth check - used for auth message itself)
    private func sendRaw(_ message: [String: Any]) -> Bool {
        guard let ws = socket, isConnected else {
            #if DEBUG
            print("[WS] sendRaw: NOT connected, can't send \(message["type"] ?? "?")")
            #endif
            handleError(.messageSendFailed("WebSocket not connected"))
            return false
        }

        do {
            let jsonData = try JSONSerialization.data(withJSONObject: message)
            guard let jsonString = String(data: jsonData, encoding: .utf8) else {
                handleError(.messageSendFailed("Failed to encode message as UTF-8"))
                return false
            }

            ws.write(string: jsonString)
            #if DEBUG
            print("[WS] sendRaw: sent \(message["type"] ?? "?") OK")
            #endif
            return true
        } catch {
            handleError(.messageSendFailed(error.localizedDescription))
            return false
        }
    }

    /// Send a message to the WebSocket server (requires authentication)
    func send(_ message: [String: Any]) -> Bool {
        guard isConnected else {
            // Not connected -- queue the message and trigger a reconnect
            pendingMessages.append(message)
            Task { await connect() }
            return true
        }

        guard socket != nil else {
            handleError(.messageSendFailed("WebSocket not available"))
            return false
        }

        guard isAuthenticated else {
            // Queue the message to be sent after auth completes
            pendingMessages.append(message)
            return true
        }

        do {
            let jsonData = try JSONSerialization.data(withJSONObject: message)
            guard let jsonString = String(data: jsonData, encoding: .utf8) else {
                handleError(.messageSendFailed("Failed to encode message as UTF-8"))
                return false
            }

            socket?.write(string: jsonString)
            return true
        } catch {
            handleError(.messageSendFailed(error.localizedDescription))
            return false
        }
    }

    /// Request list of sessions
    func requestSessions() -> Bool {
        send(["type": "list_sessions"])
    }

    /// Send a prompt to an active session
    func sendPrompt(_ sessionId: String, prompt: String) -> Bool {
        send([
            "type": "send_prompt",
            "session_id": sessionId,
            "prompt": prompt
        ])
    }

    /// Load messages for a session
    func loadMessages(sessionId: String, limit: Int = 50, offset: Int = 0) -> Bool {
        send([
            "type": "load_messages",
            "session_id": sessionId,
            "limit": limit,
            "offset": offset
        ])
    }

    /// Subscribe to real-time updates for a session
    func subscribeToSession(sessionId: String) -> Bool {
        subscribedSessionIds.insert(sessionId)
        return send([
            "type": "subscribe_session",
            "session_id": sessionId
        ])
    }

    /// Interrupt/stop the current agent session
    func interruptSession(_ sessionId: String) -> Bool {
        return send([
            "type": "interrupt_session",
            "session_id": sessionId
        ])
    }

    /// Create a new agent session with the given options
    func createSession(
        workingDirectory: String,
        modelProvider: String,
        model: String,
        permissionMode: String = "default",
        tools: [String] = [],
        systemPrompt: String? = nil,
        agentName: String? = nil,
        projectId: String? = nil,
        projectAreaId: String? = nil,
        selectedAvatarId: Int64? = nil,
        contextSummary: String? = nil,
        parentSessionId: String? = nil,
        baseURL: String? = nil,
        yoloMode: Bool = false
    ) -> Bool {
        let sessionId = UUID().uuidString

        var options: [String: Any] = [
            "working_directory": workingDirectory,
            "provider": modelProvider,
            "model": model,
            "permission_mode": permissionMode,
            "tools": tools
        ]

        // Add optional fields
        if let systemPrompt = systemPrompt {
            options["system_prompt"] = systemPrompt
        }
        if let agentName = agentName {
            options["agent_name"] = agentName
        }
        if let projectId = projectId {
            options["project_id"] = projectId
        }
        if let projectAreaId = projectAreaId {
            options["project_area_id"] = projectAreaId
        }
        if let selectedAvatarId = selectedAvatarId {
            options["selected_avatar_id"] = selectedAvatarId
        }
        if let contextSummary = contextSummary {
            options["context_summary"] = contextSummary
        }
        if let parentSessionId = parentSessionId {
            options["parent_session_id"] = parentSessionId
        }
        if let baseURL = baseURL {
            options["base_url"] = baseURL
        }

        // YOLO mode
        if yoloMode {
            options["dangerously_skip_permissions"] = true
            options["allow_dangerously_skip_permissions"] = true
        }

        return send([
            "type": "create_session",
            "session_id": sessionId,
            "options": options
        ])
    }
}

// MARK: - Preview Helper

#if DEBUG
extension WebSocketService {
    /// Create a mock WebSocket service for SwiftUI previews
    static let preview = WebSocketService()
}
#endif

// MARK: - Task Sleep Helper

extension Task where Success == Never, Failure == Never {
    /// Sleep for a specified number of seconds
    static func sleep(seconds: TimeInterval) async throws {
        try await sleep(nanoseconds: UInt64(seconds * 1_000_000_000))
    }
}
