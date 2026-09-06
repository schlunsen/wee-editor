//
//  WeeAPIClient.swift
//  wee
//
//  API client for communicating with Wee backend
//

import Foundation

// MARK: - API Models

struct LoginRequest: Codable {
    let username: String
    let password: String
}

struct LoginResponse: Codable {
    let username: String
    let expires_at: String?
    let mfa_required: Bool?
    let temporary_token: String?
    let temporary_expires: String?
}

// Empty response for endpoints that return no data
struct EmptyResponse: Codable {}

struct AgentSession: Codable, Identifiable {
    let id: String
    let status: String
    let model_name: String?
    let provider: String?
    let git_branch: String?
    let working_directory: String?
    let created_at: String
    let updated_at: String
    let message_count: Int?
    let cost: Double?
    let project_id: String?
    let selected_avatar_id: Int64?
    let selected_avatar: Avatar?
}

struct SessionMetadata: Codable, Identifiable {
    let id: String
    let status: String
    let created_at: String
    let updated_at: String
    let ended_at: String?
    let message_count: Int
    let cost_usd: Double
    let num_turns: Int
    let duration_ms: Int64
    let error_message: String?
    let model_name: String?
    let git_branch: String?
    let provider: String?
    let project_id: String?
    let selected_avatar_id: Int64?
    let selected_avatar: Avatar?

    /// Shortened session ID (first 8 characters)
    var shortID: String {
        return String(id.prefix(8))
    }
}

struct Avatar: Codable, Identifiable, Equatable {
    let id: Int64
    let theme_id: Int64
    let name: String
    let type: String? // "preset", "ai_generated"
    let image_path: String?
    let image_url: String?
    let style: String?
    let seed: String?
    let color: String?
    let created_at: String

    static func == (lhs: Avatar, rhs: Avatar) -> Bool {
        lhs.id == rhs.id
    }
}

struct AvatarTheme: Codable, Identifiable {
    let id: Int64
    let name: String
}

struct ContentBlockSource: Codable {
    let type: String
    let media_type: String
    let data: String
}

struct ContentBlock: Codable {
    let type: String  // "text", "image"
    let text: String?
    let source: ContentBlockSource?
}

struct AgentMessage: Codable, Identifiable {
    let id: String
    let role: String  // "user", "assistant", "system"
    let content: ContentBlockOrString
    let created_at: String?
    let thinking: String?
    let tools: [ToolUseInfo]? // Backend sends as array directly

    enum CodingKeys: String, CodingKey {
        case id, role, content, created_at
        case thinking = "thinking_content"  // Backend uses snake_case
        case tools = "tool_uses"  // Backend field name
    }
}

struct ToolUseInfo: Codable {
    let id: String
    let name: String
    let status: String? // Optional - backend doesn't always include this
    private let input: AnyCodable? // Flexible type that accepts dict or string

    var displayName: String {
        return name.replacingOccurrences(of: "_", with: " ").capitalized
    }

    var statusIcon: String {
        guard let status = status else { return "🔧" }
        switch status {
        case "running": return "⏳"
        case "completed": return "✅"
        case "error": return "❌"
        default: return "🔧"
        }
    }

    // MARK: - Input Extraction Methods

    /// Get the raw input dictionary
    var inputDict: [String: Any]? {
        guard let input = input else { return nil }
        return input.value as? [String: Any]
    }

    /// Helper to extract string from value that might be wrapped in AnyCodable
    private func extractString(from value: Any?) -> String? {
        guard let value = value else { return nil }

        // If it's already a String, return it
        if let stringValue = value as? String {
            return stringValue
        }

        // If it's AnyCodable, try to unwrap it
        if let anyCodable = value as? AnyCodable {
            if let stringValue = anyCodable.value as? String {
                return stringValue
            }
        }

        return nil
    }

    /// Helper to extract int from value that might be wrapped in AnyCodable
    private func extractInt(from value: Any?) -> Int? {
        guard let value = value else { return nil }

        // If it's already an Int, return it
        if let intValue = value as? Int {
            return intValue
        }

        // If it's a String, try to parse it
        if let stringValue = value as? String, let intValue = Int(stringValue) {
            return intValue
        }

        // If it's AnyCodable, try to unwrap it
        if let anyCodable = value as? AnyCodable {
            if let intValue = anyCodable.value as? Int {
                return intValue
            }
            if let stringValue = anyCodable.value as? String, let intValue = Int(stringValue) {
                return intValue
            }
        }

        return nil
    }

    /// Extract command from input (for Bash tool)
    var bashCommand: String? {
        guard name.lowercased() == "bash", let dict = inputDict else { return nil }
        return extractString(from: dict["command"])
    }

    /// Extract description from input (for Bash tool)
    var bashDescription: String? {
        guard name.lowercased() == "bash", let dict = inputDict else { return nil }
        return extractString(from: dict["description"])
    }

    /// Extract file path from input (for Read tool)
    var readFilePath: String? {
        guard name.lowercased() == "read", let dict = inputDict else { return nil }
        return extractString(from: dict["file_path"])
    }

    /// Extract limit from input (for Read tool)
    var readLimit: Int? {
        guard name.lowercased() == "read", let dict = inputDict else { return nil }
        return extractInt(from: dict["limit"])
    }

    /// Get a simplified string representation of the input
    var inputDescription: String {
        guard let input = input else { return "" }

        // If it's already a string, return it
        if let stringValue = input.value as? String {
            return stringValue
        }

        // If it's a dictionary, try to extract meaningful fields
        if let dictValue = input.value as? [String: Any] {
            // Try common fields
            if let path = dictValue["file_path"] as? String {
                return path
            }
            if let pattern = dictValue["pattern"] as? String {
                return pattern
            }
            if let command = dictValue["command"] as? String {
                // For bash commands, show the full command on first view
                return command.count > 100 ? String(command.prefix(100)) + "..." : command
            }

            // Fallback: show key-value pairs
            let pairs = dictValue.compactMap { key, value in
                "\(key): \(value)"
            }.prefix(2).joined(separator: ", ")

            return pairs.isEmpty ? "" : pairs
        }

        return ""
    }
}

extension AgentMessage {
    /// Get tools array (already decoded from JSON)
    func getTools() -> [ToolUseInfo] {
        guard let toolsArray = tools, !toolsArray.isEmpty else {
            return []
        }
        return toolsArray
    }

    /// Check if message has tools
    var hasTools: Bool {
        return tools != nil && !tools!.isEmpty
    }
}

// Helper to decode content as either string or array of ContentBlocks
enum ContentBlockOrString: Codable {
    case string(String)
    case blocks([ContentBlock])

    init(from decoder: Decoder) throws {
        let container = try decoder.singleValueContainer()

        // Try to decode as array of content blocks first (direct JSON)
        if let blocks = try? container.decode([ContentBlock].self) {
            self = .blocks(blocks)
            return
        }

        // Try to decode as string
        if let stringValue = try? container.decode(String.self) {
            // If it's a string, try to parse it as JSON content blocks
            if let jsonData = stringValue.data(using: .utf8) {
                let decoder = JSONDecoder()
                if let blocks = try? decoder.decode([ContentBlock].self, from: jsonData) {
                    self = .blocks(blocks)
                    return
                }
            }

            // Otherwise treat it as plain text
            self = .string(stringValue)
            return
        }

        throw DecodingError.dataCorruptedError(in: container, debugDescription: "Cannot decode content")
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.singleValueContainer()
        switch self {
        case .string(let str):
            try container.encode(str)
        case .blocks(let blocks):
            try container.encode(blocks)
        }
    }

    var textContent: String {
        switch self {
        case .string(let str):
            return str
        case .blocks(let blocks):
            return blocks
                .compactMap { $0.text }
                .joined(separator: "\n")
        }
    }

    var imageBlocks: [ImageBlockData] {
        switch self {
        case .string:
            return []
        case .blocks(let blocks):
            return blocks.compactMap { block in
                guard block.type == "image", let source = block.source else { return nil }
                return ImageBlockData(
                    mediaType: source.media_type,
                    base64Data: source.data
                )
            }
        }
    }
}

struct ImageBlockData: Identifiable {
    let id = UUID()
    let mediaType: String
    let base64Data: String

    var dataUrl: String {
        "data:\(mediaType);base64,\(base64Data)"
    }
}

struct HealthResponse: Codable {
    let status: String
    let version: String?
    let uptime: Int?
}

struct TranscriptionResponse: Codable {
    let text: String
    let language: String?
    let duration: Double?
}

struct ConversationData: Codable {
    let id: String
    let status: String
    let total_commands: Int?
    let created_at: String
    let updated_at: String?
}

struct Project: Codable, Identifiable, Hashable {
    let id: String
    let name: String
    let path: String
    let description: String?
    let default_model: String?
    let default_provider: String?
    let settings: String?
    let is_active: Bool
    let created_at: String
    let updated_at: String

    func hash(into hasher: inout Hasher) {
        hasher.combine(id)
    }

    static func == (lhs: Project, rhs: Project) -> Bool {
        lhs.id == rhs.id
    }
}

struct SiteProject: Codable, Identifiable {
    let id: String
    let user_description: String
    let category: String?
    let style_preferences: String?
    let additional_notes: String?
    let provider: String
    let model: String
    let status: String
    let current_step: String?
    let orchestrator_plan: String?
    let workspace_path: String?
    let created_at: String
    let updated_at: String
    let completed_at: String?
    let error_message: String?
}

struct APIResponse<T: Codable>: Codable {
    let data: T?
    let error: String?
    let message: String?
}

// MARK: - API Client

class WeeAPIClient: NSObject {
    static let shared = WeeAPIClient()

    private var urlSession: URLSession
    private let secureStorage = SecureStorage.shared
    private let hostManager = HostManager.shared

    // Configuration
    private(set) var baseURL: URL?
    private(set) var sessionToken: String?  // Changed from apiKey to sessionToken
    private(set) var currentHostId: String?

    override private init() {
        self.urlSession = createWeeURLSession()
        super.init()
        Task {
            await loadStoredCredentials()
        }
    }

    // MARK: - Configuration

    /// Configure the API client with server URL (for a new host or current host)
    func configureServer(baseURL: URL, hostId: String? = nil) throws {
        self.baseURL = baseURL
        if let hostId = hostId {
            self.currentHostId = hostId
        }
    }

    /// Set session token after successful login
    func setSessionToken(_ token: String) throws {
        self.sessionToken = token

        // Store per-host credential
        if let hostId = currentHostId {
            try secureStorage.storeHostCredential(hostId: hostId, sessionToken: token)
        }
    }

    /// Switch to a different host
    func switchHost(hostId: String) async throws {
        guard let host = hostManager.hosts.first(where: { $0.id == hostId }) else {
            throw WeeAPIError.notFound
        }

        try configureServer(baseURL: URL(string: host.url) ?? URL(string: "https://localhost")!, hostId: hostId)

        // Load the session token for this host if available
        if let credential = try secureStorage.retrieveHostCredential(hostId: hostId),
           !isCredentialExpired(credential) {
            self.sessionToken = credential.sessionToken
        } else {
            self.sessionToken = nil
        }

        // Record host usage
        await hostManager.recordHostUse(id: hostId)
    }

    /// Load stored credentials from Keychain
    private func loadStoredCredentials() async {
        // Load current host preference from HostManager
        if let host = hostManager.currentHost {
            if let url = URL(string: host.url) {
                self.baseURL = url
                self.currentHostId = host.id

                // Try to load session token for this host
                if let credential = try? secureStorage.retrieveHostCredential(hostId: host.id),
                   !isCredentialExpired(credential) {
                    self.sessionToken = credential.sessionToken
                }
            }
        }
    }

    /// Clear stored credentials (logout from current host)
    func clearCredentials() throws {
        // Clear session token for current host
        if let hostId = currentHostId {
            try secureStorage.deleteHostCredential(hostId: hostId)
        }

        self.baseURL = nil
        self.sessionToken = nil
        self.currentHostId = nil
    }

    /// Check if a credential has expired
    private func isCredentialExpired(_ credential: HostCredential) -> Bool {
        if let expiresAt = credential.expiresAt {
            return Date() > expiresAt
        }
        return false
    }

    // MARK: - API Key (for WebSocket auth)

    /// Fetch the server API key needed for WebSocket authentication.
    /// This is different from the session token — the session token authenticates the user,
    /// while the API key authenticates the WebSocket connection to the agent handler.
    func fetchAPIKey() async throws -> String {
        guard let baseURL = baseURL else {
            throw WeeAPIError.notConfigured
        }

        let url = baseURL.appendingPathComponent("api/config/api-key")
        var request = URLRequest(url: url)
        request.httpMethod = "GET"
        request.timeoutInterval = 10

        // Use session token for authentication
        if let token = sessionToken {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }

        let (data, response) = try await urlSession.data(for: request)

        guard let httpResponse = response as? HTTPURLResponse,
              (200...299).contains(httpResponse.statusCode) else {
            throw WeeAPIError.invalidResponse
        }

        struct APIKeyResponse: Codable {
            let apiKey: String
        }

        let decoded = try JSONDecoder().decode(APIKeyResponse.self, from: data)
        return decoded.apiKey
    }

    // MARK: - HTTP Methods

    /// Login with username and password
    /// The backend delivers the session token via HttpOnly Set-Cookie header.
    /// We extract it from the response headers and store it for Bearer auth on subsequent requests.
    func login(username: String, password: String) async throws -> LoginResponse {
        guard let baseURL = baseURL else {
            throw WeeAPIError.notConfigured
        }

        let url = baseURL.appendingPathComponent("api/auth/login")
        let request = LoginRequest(username: username, password: password)

        let encoder = JSONEncoder()
        let body = try encoder.encode(request)

        var URLRequest = URLRequest(url: url)
        URLRequest.httpMethod = "POST"
        URLRequest.timeoutInterval = 30
        URLRequest.setValue("application/json", forHTTPHeaderField: "Content-Type")
        URLRequest.httpBody = body

        let (data, response) = try await urlSession.data(for: URLRequest)

        guard let httpResponse = response as? HTTPURLResponse else {
            throw WeeAPIError.invalidResponse
        }

        switch httpResponse.statusCode {
        case 200...299:
            // Extract session token from Set-Cookie header
            if let token = extractSessionToken(from: httpResponse) {
                try setSessionToken(token)
            }

            // Decode the JSON response
            do {
                return try JSONDecoder().decode(LoginResponse.self, from: data)
            } catch {
                throw WeeAPIError.decodingError(error)
            }

        case 401:
            throw WeeAPIError.unauthorized

        case 404:
            throw WeeAPIError.notFound

        case 500...599:
            throw WeeAPIError.serverError(httpResponse.statusCode)

        default:
            throw WeeAPIError.httpError(httpResponse.statusCode)
        }
    }

    /// Extract session_token from Set-Cookie response headers
    private func extractSessionToken(from response: HTTPURLResponse) -> String? {
        // Primary: check HTTPCookieStorage (most reliable for HttpOnly cookies with URLSession)
        if let url = response.url,
           let cookies = HTTPCookieStorage.shared.cookies(for: url),
           let sessionCookie = cookies.first(where: { $0.name == "session_token" }) {
            return sessionCookie.value
        }

        // Fallback: parse Set-Cookie header directly
        if let setCookies = response.value(forHTTPHeaderField: "Set-Cookie") {
            return parseSessionTokenFromCookieHeader(setCookies)
        }

        return nil
    }

    /// Parse the session_token value from a Set-Cookie header string
    private func parseSessionTokenFromCookieHeader(_ header: String) -> String? {
        // Format: "session_token=<value>; Path=/; HttpOnly; ..."
        let parts = header.components(separatedBy: ";")
        for part in parts {
            let trimmed = part.trimmingCharacters(in: .whitespaces)
            if trimmed.hasPrefix("session_token=") {
                let token = String(trimmed.dropFirst("session_token=".count))
                return token.isEmpty ? nil : token
            }
        }
        return nil
    }

    /// Test connectivity to the Wee server
    func testConnection() async throws -> HealthResponse {
        guard let baseURL = baseURL else {
            throw WeeAPIError.notConfigured
        }

        let url = baseURL.appendingPathComponent("api/health")
        return try await performRequest(url: url, method: "GET", body: nil)
    }

    /// Get all agent sessions
    func getAgentSessions() async throws -> [AgentSession] {
        guard let baseURL = baseURL else {
            throw WeeAPIError.notConfigured
        }

        let url = baseURL.appendingPathComponent("api/agent/sessions")

        struct SessionsResponse: Codable {
            let sessions: [AgentSession]?
        }

        let response: SessionsResponse = try await performRequest(url: url, method: "GET", body: nil)
        return response.sessions ?? []
    }

    /// Get conversations data
    func getConversations() async throws -> [ConversationData] {
        guard let baseURL = baseURL else {
            throw WeeAPIError.notConfigured
        }

        let url = baseURL.appendingPathComponent("api/conversations")

        struct ConversationsResponse: Codable {
            let conversations: [ConversationData]?
        }

        let response: ConversationsResponse = try await performRequest(url: url, method: "GET", body: nil)
        return response.conversations ?? []
    }

    /// Get main projects list
    func getProjects() async throws -> [Project] {
        guard let baseURL = baseURL else {
            throw WeeAPIError.notConfigured
        }

        let url = baseURL.appendingPathComponent("api/projects")

        struct ProjectsResponse: Codable {
            let projects: [Project]?
        }

        let response: ProjectsResponse = try await performRequest(url: url, method: "GET", body: nil)
        return response.projects ?? []
    }

    /// Get site projects list
    func getSiteProjects(limit: Int = 50, offset: Int = 0) async throws -> [SiteProject] {
        guard let baseURL = baseURL else {
            throw WeeAPIError.notConfigured
        }

        var urlComponents = URLComponents(url: baseURL.appendingPathComponent("api/site/projects"), resolvingAgainstBaseURL: false)
        urlComponents?.queryItems = [
            URLQueryItem(name: "limit", value: String(limit)),
            URLQueryItem(name: "offset", value: String(offset))
        ]

        guard let url = urlComponents?.url else {
            throw WeeAPIError.invalidURL
        }

        struct SiteProjectsResponse: Codable {
            let projects: [SiteProject]?
            let limit: Int?
            let offset: Int?
        }

        let response: SiteProjectsResponse = try await performRequest(url: url, method: "GET", body: nil)
        return response.projects ?? []
    }

    /// Get site project details by ID
    func getSiteProjectDetails(projectID: String) async throws -> SiteProject? {
        guard let baseURL = baseURL else {
            throw WeeAPIError.notConfigured
        }

        let url = baseURL.appendingPathComponent("api/site/projects/\(projectID)")

        let response: SiteProject = try await performRequest(url: url, method: "GET", body: nil)
        return response
    }

    /// Get sessions for a specific project
    func getProjectSessions(projectID: String) async throws -> [SessionMetadata] {
        guard let baseURL = baseURL else {
            throw WeeAPIError.notConfigured
        }

        let url = baseURL.appendingPathComponent("api/projects/\(projectID)/sessions")

        struct ProjectSessionsResponse: Codable {
            let sessions: [SessionMetadata]?
        }

        let response: ProjectSessionsResponse = try await performRequest(url: url, method: "GET", body: nil)
        return response.sessions ?? []
    }

    /// Get messages for a specific agent session
    func getSessionMessages(sessionID: String, limit: Int = 50, beforeSequence: Int = 0) async throws -> [AgentMessage] {
        guard let baseURL = baseURL else {
            throw WeeAPIError.notConfigured
        }

        var urlComponents = URLComponents(url: baseURL.appendingPathComponent("api/agent/sessions/\(sessionID)/messages"), resolvingAgainstBaseURL: false)
        var queryItems = [
            URLQueryItem(name: "limit", value: String(limit))
        ]
        if beforeSequence > 0 {
            queryItems.append(URLQueryItem(name: "before_sequence", value: String(beforeSequence)))
        }
        urlComponents?.queryItems = queryItems

        guard let url = urlComponents?.url else {
            throw WeeAPIError.invalidURL
        }

        struct MessagesResponse: Codable {
            let messages: [AgentMessage]?
        }

        let response: MessagesResponse = try await performRequest(url: url, method: "GET", body: nil)
        return response.messages ?? []
    }

    // MARK: - Audio Transcription

    func transcribeAudio(audioData: Data) async throws -> String {
        guard let baseURL = baseURL else {
            throw WeeAPIError.notConfigured
        }

        let url = baseURL.appendingPathComponent("api/transcribe")

        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.timeoutInterval = 60  // Longer timeout for audio processing

        // Create multipart form data
        let boundary = UUID().uuidString
        request.setValue("multipart/form-data; boundary=\(boundary)", forHTTPHeaderField: "Content-Type")

        // Add authentication header if we have a session token
        if let token = sessionToken {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }

        // Build multipart body
        var body = Data()

        // Add audio file
        body.append("--\(boundary)\r\n".data(using: .utf8)!)
        body.append("Content-Disposition: form-data; name=\"audio\"; filename=\"audio.m4a\"\r\n".data(using: .utf8)!)
        body.append("Content-Type: audio/mp4\r\n\r\n".data(using: .utf8)!)
        body.append(audioData)
        body.append("\r\n".data(using: .utf8)!)

        // End boundary
        body.append("--\(boundary)--\r\n".data(using: .utf8)!)

        request.httpBody = body


        let (data, response) = try await urlSession.data(for: request)

        guard let httpResponse = response as? HTTPURLResponse else {
            throw WeeAPIError.invalidResponse
        }


        switch httpResponse.statusCode {
        case 200...299:
            do {
                let decoder = JSONDecoder()
                let transcriptionResponse = try decoder.decode(TranscriptionResponse.self, from: data)
                return transcriptionResponse.text
            } catch {
                // Try parsing as simple JSON response with text key
                if let jsonDict = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                   let text = jsonDict["text"] as? String {
                    return text
                }
                throw WeeAPIError.decodingError(error)
            }

        case 401:
            throw WeeAPIError.unauthorized

        case 404:
            throw WeeAPIError.notFound

        case 500...599:
            throw WeeAPIError.serverError(httpResponse.statusCode)

        default:
            throw WeeAPIError.httpError(httpResponse.statusCode)
        }
    }

    // MARK: - Internal Methods

    private func performRequest<T: Codable>(url: URL, method: String, body: Data?) async throws -> T {
        var request = URLRequest(url: url)
        request.httpMethod = method
        request.timeoutInterval = 30

        // Add headers
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")

        // Add authentication header if we have a session token
        if let token = sessionToken {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        } else {
        }

        if let body = body {
            request.httpBody = body
        }

        let (data, response) = try await urlSession.data(for: request)

        // Check HTTP response
        guard let httpResponse = response as? HTTPURLResponse else {
            throw WeeAPIError.invalidResponse
        }


        // Handle various status codes
        switch httpResponse.statusCode {
        case 200...299:
            // Success
            do {
                return try JSONDecoder().decode(T.self, from: data)
            } catch {
                throw WeeAPIError.decodingError(error)
            }

        case 401:
            throw WeeAPIError.unauthorized

        case 404:
            throw WeeAPIError.notFound

        case 500...599:
            throw WeeAPIError.serverError(httpResponse.statusCode)

        default:
            throw WeeAPIError.httpError(httpResponse.statusCode)
        }
    }

    /// Get all avatar themes
    func getAvatarThemes() async throws -> [AvatarTheme] {
        guard let baseURL = baseURL else {
            throw WeeAPIError.notConfigured
        }

        let url = baseURL.appendingPathComponent("api/avatars/themes")

        struct ThemesResponse: Codable {
            let themes: [AvatarTheme]?
        }

        let response: ThemesResponse = try await performRequest(url: url, method: "GET", body: nil)
        return response.themes ?? []
    }

    /// Get avatar theme by ID
    func getAvatarTheme(themeId: Int64) async throws -> AvatarTheme {
        guard let baseURL = baseURL else {
            throw WeeAPIError.notConfigured
        }

        let url = baseURL.appendingPathComponent("api/avatars/themes/\(themeId)")

        struct ThemeResponse: Codable {
            let id: Int64
            let name: String
        }

        let response: ThemeResponse = try await performRequest(url: url, method: "GET", body: nil)
        return AvatarTheme(id: response.id, name: response.name)
    }

    /// Get avatars for a theme
    func getThemeAvatars(themeId: Int64) async throws -> [Avatar] {
        guard let baseURL = baseURL else {
            throw WeeAPIError.notConfigured
        }

        let url = baseURL.appendingPathComponent("api/avatars/themes/\(themeId)/avatars")

        struct AvatarsResponse: Codable {
            let avatars: [Avatar]?
        }

        let response: AvatarsResponse = try await performRequest(url: url, method: "GET", body: nil)
        return response.avatars ?? []
    }

    /// Get avatar image by avatar ID using the /api/avatars/:id/image endpoint
    func getAvatarImage(avatarId: Int64) async throws -> Data {
        guard let baseURL = baseURL else {
            throw WeeAPIError.notConfigured
        }

        // Use the backend's unified image endpoint
        // Serves avatar images from the filesystem
        let url = baseURL.appendingPathComponent("api/avatars/\(avatarId)/image")

        var request = URLRequest(url: url)
        request.timeoutInterval = 30

        // Add authentication header if we have a session token
        if let token = sessionToken {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }

        let (data, response) = try await URLSession.shared.data(for: request)

        guard let httpResponse = response as? HTTPURLResponse else {
            throw WeeAPIError.invalidResponse
        }

        guard httpResponse.statusCode == 200 else {
            throw WeeAPIError.httpError(httpResponse.statusCode)
        }

        return data
    }

    /// Download image from an external URL
    func downloadImage(urlString: String) async throws -> Data {
        guard let url = URL(string: urlString) else {
            throw WeeAPIError.invalidURL
        }

        var request = URLRequest(url: url)
        request.timeoutInterval = 30

        let (data, response) = try await URLSession.shared.data(for: request)

        guard let httpResponse = response as? HTTPURLResponse else {
            throw WeeAPIError.invalidResponse
        }

        guard httpResponse.statusCode == 200 else {
            throw WeeAPIError.httpError(httpResponse.statusCode)
        }

        return data
    }

    // MARK: - Just Commands API

    func getJustRecipes(projectID: String) async throws -> (recipes: [JustRecipe], hasJustfile: Bool) {
        guard let baseURL = baseURL else {
            throw WeeAPIError.notConfigured
        }

        let url = baseURL.appendingPathComponent("api/projects/\(projectID)/just/recipes")

        struct RecipesResponse: Codable {
            let recipes: [JustRecipe]?
            let has_justfile: Bool?
        }

        let response: RecipesResponse = try await performRequest(url: url, method: "GET", body: nil)
        return (response.recipes ?? [], response.has_justfile ?? false)
    }

    func runJustRecipe(projectID: String, recipe: String, args: [String]? = nil) async throws -> JustJob {
        guard let baseURL = baseURL else {
            throw WeeAPIError.notConfigured
        }

        let url = baseURL.appendingPathComponent("api/projects/\(projectID)/just/run")

        struct RunRequest: Codable {
            let recipe: String
            let args: [String]?
        }

        struct RunResponse: Codable {
            let job: JustJob?
        }

        let body = try JSONEncoder().encode(RunRequest(recipe: recipe, args: args))
        let response: RunResponse = try await performRequest(url: url, method: "POST", body: body)
        guard let job = response.job else {
            throw WeeAPIError.decodingError(NSError(domain: "WeeAPI", code: -1, userInfo: [NSLocalizedDescriptionKey: "Missing job in response"]))
        }
        return job
    }

    func getJustJobs(projectID: String) async throws -> [JustJob] {
        guard let baseURL = baseURL else {
            throw WeeAPIError.notConfigured
        }

        let url = baseURL.appendingPathComponent("api/projects/\(projectID)/just/jobs")

        struct JobsResponse: Codable {
            let jobs: [JustJob]?
        }

        let response: JobsResponse = try await performRequest(url: url, method: "GET", body: nil)
        return response.jobs ?? []
    }

    func getJustJob(projectID: String, jobID: String) async throws -> JustJob {
        guard let baseURL = baseURL else {
            throw WeeAPIError.notConfigured
        }

        let url = baseURL.appendingPathComponent("api/projects/\(projectID)/just/jobs/\(jobID)")

        struct JobResponse: Codable {
            let job: JustJob?
        }

        let response: JobResponse = try await performRequest(url: url, method: "GET", body: nil)
        guard let job = response.job else {
            throw WeeAPIError.notFound
        }
        return job
    }

    func stopJustJob(projectID: String, jobID: String) async throws -> JustJob {
        guard let baseURL = baseURL else {
            throw WeeAPIError.notConfigured
        }

        let url = baseURL.appendingPathComponent("api/projects/\(projectID)/just/jobs/\(jobID)/stop")

        struct JobResponse: Codable {
            let job: JustJob?
        }

        let response: JobResponse = try await performRequest(url: url, method: "POST", body: nil)
        guard let job = response.job else {
            throw WeeAPIError.notFound
        }
        return job
    }

    func deleteJustJob(projectID: String, jobID: String) async throws {
        guard let baseURL = baseURL else {
            throw WeeAPIError.notConfigured
        }

        let url = baseURL.appendingPathComponent("api/projects/\(projectID)/just/jobs/\(jobID)")
        _ = try await performRequest(url: url, method: "DELETE", body: nil) as EmptyResponse
    }

    // MARK: - Session Context API

    func getSessionContext(sessionID: String) async throws -> SessionContextInfo? {
        guard let baseURL = baseURL else {
            throw WeeAPIError.notConfigured
        }

        let url = baseURL.appendingPathComponent("api/agent/sessions/\(sessionID)/context")
        return try await performRequest(url: url, method: "GET", body: nil)
    }

    // MARK: - Git API

    func getGitStatus(sessionID: String) async throws -> GitStatus {
        guard let baseURL = baseURL else {
            throw WeeAPIError.notConfigured
        }

        let url = baseURL.appendingPathComponent("api/agent/sessions/\(sessionID)/git-status")
        return try await performRequest(url: url, method: "GET", body: nil)
    }

    func getGitDiff(sessionID: String) async throws -> GitDiffData {
        guard let baseURL = baseURL else {
            throw WeeAPIError.notConfigured
        }

        let url = baseURL.appendingPathComponent("api/agent/sessions/\(sessionID)/git/diff")
        return try await performRequest(url: url, method: "GET", body: nil)
    }

    func getGitRemote(sessionID: String) async throws -> GitRemoteInfo {
        guard let baseURL = baseURL else {
            throw WeeAPIError.notConfigured
        }

        let url = baseURL.appendingPathComponent("api/agent/sessions/\(sessionID)/git-remote")
        return try await performRequest(url: url, method: "GET", body: nil)
    }

    // MARK: - Error Types

    enum WeeAPIError: LocalizedError {
        case notConfigured
        case invalidURL
        case invalidResponse
        case unauthorized
        case notFound
        case decodingError(Error)
        case serverError(Int)
        case httpError(Int)
        case networkError(Error)

        var errorDescription: String? {
            switch self {
            case .notConfigured:
                return "API client not configured. Set baseURL and apiKey first."
            case .invalidURL:
                return "Invalid URL provided"
            case .invalidResponse:
                return "Invalid response from server"
            case .unauthorized:
                return "Unauthorized. Check your API key."
            case .notFound:
                return "Resource not found on server"
            case .decodingError(let error):
                return "Failed to decode response: \(error.localizedDescription)"
            case .serverError(let code):
                return "Server error: \(code)"
            case .httpError(let code):
                return "HTTP error: \(code)"
            case .networkError(let error):
                return "Network error: \(error.localizedDescription)"
            }
        }
    }
}

// MARK: - Justfile Models

struct JustRecipe: Codable, Identifiable {
    let name: String
    let description: String
    let parameters: [String]?

    var id: String { name }
}

struct JustJob: Codable, Identifiable {
    let id: String
    let projectId: String
    let recipe: String
    let args: [String]?
    let status: String // "running", "completed", "failed"
    let exitCode: Int
    let output: String
    let startedAt: String
    let finishedAt: String?

    enum CodingKeys: String, CodingKey {
        case id
        case projectId = "project_id"
        case recipe, args, status
        case exitCode = "exit_code"
        case output
        case startedAt = "started_at"
        case finishedAt = "finished_at"
    }
}

// MARK: - Git Models

struct GitStatus: Codable {
    let branch: String?
    let ahead: Int?
    let behind: Int?
    let staged: [String]?
    let modified: [String]?
    let untracked: [String]?
    let deleted: [String]?
    let clean: Bool?
}

struct GitDiffFile: Codable {
    let path: String?
    let status: String? // "modified", "added", "deleted", "renamed"
    let additions: Int?
    let deletions: Int?
    let diff: String?
}

struct GitDiffStats: Codable {
    let filesChanged: Int?
    let additions: Int?
    let deletions: Int?
}

struct GitDiffData: Codable {
    let stats: GitDiffStats?
    let files: [GitDiffFile]?
}

struct GitRemoteInfo: Codable {
    let remote_url: String?
    let html_url: String?
    let owner: String?
    let repo: String?
}
