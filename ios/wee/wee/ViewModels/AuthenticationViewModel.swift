//
//  AuthenticationViewModel.swift
//  wee
//
//  ViewModel for managing authentication state with multi-host support
//

import SwiftUI
import Combine

@MainActor
class AuthenticationViewModel: ObservableObject {
    @Published var isAuthenticated = false
    @Published var isLoading = false
    @Published var errorMessage: String?

    @Published var savedHosts: [Host] = []
    @Published var currentHost: Host?

    private let apiClient = WeeAPIClient.shared
    private let hostManager = HostManager.shared
    private var sessionExpiredObserver: Any?

    init() {
        // Listen for session expiration from WebSocket
        sessionExpiredObserver = NotificationCenter.default.addObserver(
            forName: .sessionExpired, object: nil, queue: .main
        ) { [weak self] _ in
            Task { @MainActor in
                self?.isAuthenticated = false
                self?.errorMessage = "Session expired. Please log in again."
            }
        }

        Task {
            await loadSavedHosts()

            // Check if we have stored credentials and session token
            if apiClient.baseURL != nil && apiClient.sessionToken != nil {
                isAuthenticated = true
                currentHost = hostManager.currentHost
            }
        }
    }

    deinit {
        if let observer = sessionExpiredObserver {
            NotificationCenter.default.removeObserver(observer)
        }
    }

    // MARK: - Host Management

    private func loadSavedHosts() async {
        self.savedHosts = hostManager.hosts
        self.currentHost = hostManager.currentHost
    }

    /// Add a new host and optionally authenticate
    func addNewHost(name: String, url: String, username: String? = nil) async throws -> Host {
        let host = try await hostManager.addHost(name: name, url: url, username: username)
        await loadSavedHosts()
        return host
    }

    /// Switch to a saved host
    func switchHost(hostId: String) async throws {
        try await apiClient.switchHost(hostId: hostId)
        try await hostManager.switchHost(id: hostId)
        await loadSavedHosts()
        // Note: User needs to login again after switching
        isAuthenticated = false
        errorMessage = "Switched to \(hostManager.currentHost?.name ?? "selected host"). Please login again."
    }

    /// Delete a saved host
    func deleteHost(hostId: String) async throws {
        try await hostManager.deleteHost(id: hostId)
        await loadSavedHosts()

        // If deleted host was current, logout
        if apiClient.currentHostId == hostId {
            logout()
        }
    }

    // MARK: - Authentication Methods

    /// Configure server URL for a new host (called from LoginView when adding new host)
    func configureServer(url: String) throws -> URL {
        var urlString = url.trimmingCharacters(in: .whitespaces)

        // Add https:// if not present
        if !urlString.hasPrefix("http://") && !urlString.hasPrefix("https://") {
            urlString = "https://\(urlString)"
        }

        guard let parsedURL = URL(string: urlString) else {
            throw AuthError.invalidURL
        }

        try apiClient.configureServer(baseURL: parsedURL)
        return parsedURL
    }

    /// Attempt to login with username and password to a specific host
    func login(username: String, password: String, hostId: String? = nil) async {
        isLoading = true
        errorMessage = nil

        defer { isLoading = false }

        // Validate inputs
        guard !username.trimmingCharacters(in: .whitespaces).isEmpty else {
            errorMessage = "Username cannot be empty"
            return
        }

        guard !password.isEmpty else {
            errorMessage = "Password cannot be empty"
            return
        }

        // If switching to a specific host, switch first
        if let hostId = hostId {
            do {
                try await switchHost(hostId: hostId)
            } catch {
                errorMessage = "Failed to switch host: \(error.localizedDescription)"
                return
            }
        }

        // Attempt login
        do {
            _ = try await apiClient.login(username: username.trimmingCharacters(in: .whitespaces), password: password)
            isAuthenticated = true
            errorMessage = nil
            WeeHaptics.success()
            await loadSavedHosts()
        } catch {
            WeeHaptics.error()
            errorMessage = "Login failed: \(error.localizedDescription)"
            // Clear credentials on failed login
            try? apiClient.clearCredentials()
        }
    }

    /// Logout user
    func logout() {
        do {
            try apiClient.clearCredentials()
            isAuthenticated = false
            errorMessage = nil
        } catch {
            errorMessage = "Failed to logout: \(error.localizedDescription)"
        }
    }

    /// Refresh authentication status
    func refreshAuthStatus() async {
        guard isAuthenticated else { return }

        do {
            _ = try await apiClient.testConnection()
            // Still authenticated
        } catch {
            // Connection failed, might be offline - that's okay
            // Only logout if explicitly unauthorized
            if case WeeAPIClient.WeeAPIError.unauthorized = error {
                isAuthenticated = false
            }
        }
    }
}

// MARK: - Error Types

enum AuthError: LocalizedError {
    case invalidURL

    var errorDescription: String? {
        switch self {
        case .invalidURL:
            return "Invalid server URL format"
        }
    }
}
