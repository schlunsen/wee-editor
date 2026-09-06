//
//  CreateSessionViewModel.swift
//  wee
//
//  ViewModel for managing session creation form
//

import SwiftUI
import Combine

@MainActor
class CreateSessionViewModel: ObservableObject {
    // MARK: - Published Properties

    @Published var workingDirectory = ""
    @Published var selectedProvider = ""
    @Published var selectedModel = ""
    @Published var selectedProject: Project? = nil
    @Published var permissionMode = "default"
    @Published var systemPrompt = "You are a helpful AI assistant."
    @Published var selectedTools: Set<String> = ["Read", "Write", "Edit", "Bash"]
    @Published var yoloMode = false

    @Published var availableProviders: [Provider] = []
    @Published var availableModels: [String] = []
    @Published var availableProjects: [Project] = []
    @Published var selectedAvatarId: Int64? = nil
    @Published var selectedAvatarThemeId: Int64? = nil
    @Published var isLoading = false
    @Published var isSavingSession = false
    @Published var errorMessage: String?
    @Published var successMessage: String?

    private let apiClient = WeeAPIClient.shared
    let webSocketService = WebSocketService.shared
    private var preSelectedProjectId: String?
    private var preFilledWorkingDirectory: String?

    // MARK: - Initialization

    init(preSelectedProjectId: String? = nil, workingDirectory: String? = nil) {
        self.preSelectedProjectId = preSelectedProjectId
        self.preFilledWorkingDirectory = workingDirectory
        if let workingDirectory = workingDirectory {
            self.workingDirectory = workingDirectory
        }
    }

    // MARK: - Computed Properties

    var isFormValid: Bool {
        !workingDirectory.trimmingCharacters(in: .whitespaces).isEmpty &&
        !selectedProvider.isEmpty &&
        !selectedModel.isEmpty &&
        selectedProject != nil
    }

    // MARK: - Methods

    func loadProjects() async {
        isLoading = true
        errorMessage = nil

        defer { isLoading = false }

        do {
            availableProjects = try await apiClient.getProjects()

            // If a project was pre-selected, use it
            if let preSelectedId = preSelectedProjectId,
               let preSelectedProject = availableProjects.first(where: { $0.id == preSelectedId }) {
                selectedProject = preSelectedProject
            } else if let firstActiveProject = availableProjects.first(where: { $0.is_active }) {
                // Otherwise auto-select first active project
                selectedProject = firstActiveProject
            } else if !availableProjects.isEmpty {
                selectedProject = availableProjects[0]
            }
        } catch {
            errorMessage = "Failed to load projects: \(error.localizedDescription)"
        }
    }

    func loadProviders() async {
        isLoading = true
        errorMessage = nil

        defer { isLoading = false }

        let response = await fetch("/api/providers")
        if let providers = response["providers"] as? [[String: Any]] {
            availableProviders = providers.compactMap { dict in
                guard let id = dict["id"] as? String,
                      let name = dict["name"] as? String else { return nil }
                let models = dict["models"] as? [String] ?? []
                return Provider(id: id, name: name, models: models)
            }

            // Auto-select first provider if available
            if !availableProviders.isEmpty {
                selectedProvider = availableProviders[0].id
                // Populate available models for the selected provider
                availableModels = availableProviders[0].models
                // Try to select "haiku" model, fall back to first available model
                if let haikuModel = availableProviders[0].models.first(where: { $0.lowercased().contains("haiku") }) {
                    selectedModel = haikuModel
                } else {
                    selectedModel = availableProviders[0].models.first ?? ""
                }
            }
        }
    }

    func updateModelsForProvider(_ providerId: String) {
        if let provider = availableProviders.first(where: { $0.id == providerId }) {
            availableModels = provider.models
            // Try to select "haiku" model, fall back to first available model
            if let haikuModel = provider.models.first(where: { $0.lowercased().contains("haiku") }) {
                selectedModel = haikuModel
            } else {
                selectedModel = provider.models.first ?? ""
            }
        }
    }

    func selectRandomAvatar() async {
        // Fetch all avatar themes
        let themes = await fetch("/api/avatars/themes")
        guard let themesArray = themes["themes"] as? [[String: Any]] else { return }

        // Pick a random theme
        guard let randomThemeDict = themesArray.randomElement() else { return }
        guard let themeId = randomThemeDict["id"] as? Int64 else { return }

        // Fetch avatars for this theme
        let avatarsResponse = await fetch("/api/avatars/themes/\(themeId)/avatars")
        guard let avatarsArray = avatarsResponse["avatars"] as? [[String: Any]] else { return }

        // Pick a random avatar
        guard let randomAvatarDict = avatarsArray.randomElement() else { return }
        guard let avatarId = randomAvatarDict["id"] as? Int64 else { return }

        // Set the selected avatar
        self.selectedAvatarThemeId = themeId
        self.selectedAvatarId = avatarId
    }

    func createSession() async -> Bool {
        guard isFormValid else {
            errorMessage = "Please fill in all required fields"
            return false
        }

        isSavingSession = true
        errorMessage = nil
        successMessage = nil

        defer { isSavingSession = false }

        // Check if WebSocket is connected
        if !webSocketService.isConnected {
            #if DEBUG
            print("❌ WebSocket not connected. Attempting to connect...")
            #endif
            errorMessage = "WebSocket not connected. Please wait a moment and try again."
            return false
        }

        #if DEBUG
        print("📤 Sending create_session message...")
        #endif
        #if DEBUG
        print("   Working Dir: \(workingDirectory)")
        #endif
        #if DEBUG
        print("   Provider: \(selectedProvider)")
        #endif
        #if DEBUG
        print("   Model: \(selectedModel)")
        #endif
        #if DEBUG
        print("   Project: \(selectedProject?.name ?? "none")")
        #endif
        #if DEBUG
        print("   Permission Mode: \(yoloMode ? "yolo" : permissionMode)")
        #endif
        #if DEBUG
        print("   YOLO Mode: \(yoloMode)")
        #endif

        // Send create_session message via WebSocket
        // When YOLO mode is enabled, use "yolo" as permission mode
        let effectivePermissionMode = yoloMode ? "yolo" : permissionMode

        let success = webSocketService.createSession(
            workingDirectory: workingDirectory.trimmingCharacters(in: .whitespaces),
            modelProvider: selectedProvider,
            model: selectedModel,
            permissionMode: effectivePermissionMode,
            tools: Array(selectedTools),
            systemPrompt: systemPrompt,
            projectId: selectedProject?.id,
            selectedAvatarId: selectedAvatarId,
            yoloMode: yoloMode
        )

        if success {
            #if DEBUG
            print("✅ Session creation message sent successfully")
            #endif
            successMessage = "Session created! Check the Projects tab to monitor progress."

            // NOTE: Notification is now posted from CreateSessionView after session is confirmed to exist
            // This prevents ProjectDetailView from fetching too early

            // Don't reset form here - let the View handle closing and then resetting
            return true
        } else {
            #if DEBUG
            print("❌ Failed to send session creation message")
            #endif
            errorMessage = "Failed to send create session message. Please check your connection."
            return false
        }
    }

    func resetForm() {
        workingDirectory = ""
        systemPrompt = "You are a helpful AI assistant."
        selectedTools = ["Read", "Write", "Edit", "Bash"]
        permissionMode = "default"
        yoloMode = false
        selectedAvatarId = nil
        selectedAvatarThemeId = nil
        errorMessage = nil
        successMessage = nil
    }

    // MARK: - Helper Methods

    private func fetch(_ endpoint: String) async -> [String: Any] {
        do {
            let url = apiClient.baseURL?.appendingPathComponent(endpoint) ?? URL(string: "http://localhost")!

            var request = URLRequest(url: url)
            request.httpMethod = "GET"
            request.timeoutInterval = 30

            if let token = apiClient.sessionToken {
                request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
            }

            let (data, response) = try await URLSession.shared.data(for: request)

            guard let httpResponse = response as? HTTPURLResponse, httpResponse.statusCode == 200 else {
                return [:]
            }

            if let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any] {
                return json
            }
            return [:]
        } catch {
            errorMessage = "Network error: \(error.localizedDescription)"
            return [:]
        }
    }
}

// MARK: - Supporting Types

struct Provider {
    let id: String
    let name: String
    let models: [String]
}
