//
//  ProvidersViewModel.swift
//  wee
//
//  ViewModel for managing AI provider configurations
//

import SwiftUI
import Combine

// MARK: - Provider Configuration Model

struct ProviderInfo: Identifiable, Equatable {
    let id: String
    let name: String
    let icon: String
    let baseURL: String?
    let models: [String]
    let defaultModel: String?
    let description: String?
    var isCurrent: Bool
    var isConfigured: Bool
    var apiKey: String?
    var customURL: String?
    var configuredModel: String?

    var statusLabel: String {
        if isCurrent { return "Default" }
        if isConfigured { return "Configured" }
        return "Available"
    }

    var statusColor: Color {
        if isCurrent { return WeeColors.accent }
        if isConfigured { return WeeColors.success }
        return WeeColors.textTertiary
    }
}

// MARK: - ViewModel

@MainActor
class ProvidersViewModel: ObservableObject {
    // MARK: - Published Properties

    @Published var providers: [ProviderInfo] = []
    @Published var isLoading = false
    @Published var errorMessage: String?
    @Published var successMessage: String?

    // Edit sheet state
    @Published var editingProvider: ProviderInfo?
    @Published var editAPIKey = ""
    @Published var editCustomURL = ""
    @Published var editModelName = ""
    @Published var isSaving = false

    private let apiClient = WeeAPIClient.shared
    private let urlSession = createWeeURLSession()

    // MARK: - Load Providers

    func loadProviders() async {
        isLoading = true
        errorMessage = nil

        defer { isLoading = false }

        let response = await fetch("/api/providers")
        guard let providersArray = response["providers"] as? [[String: Any]] else {
            errorMessage = "Failed to load providers"
            return
        }

        providers = providersArray.compactMap { dict -> ProviderInfo? in
            guard let id = dict["id"] as? String,
                  let name = dict["name"] as? String else { return nil }

            return ProviderInfo(
                id: id,
                name: name,
                icon: dict["icon"] as? String ?? "",
                baseURL: dict["base_url"] as? String,
                models: dict["models"] as? [String] ?? [],
                defaultModel: dict["default_model"] as? String,
                description: dict["description"] as? String,
                isCurrent: dict["is_current"] as? Bool ?? false,
                isConfigured: dict["is_configured"] as? Bool ?? false,
                apiKey: dict["api_key"] as? String,
                customURL: dict["custom_url"] as? String,
                configuredModel: dict["configured_model"] as? String
            )
        }
    }

    // MARK: - Configure Provider

    func startEditing(_ provider: ProviderInfo) {
        editingProvider = provider
        editAPIKey = provider.apiKey ?? ""
        editCustomURL = provider.customURL ?? ""
        editModelName = provider.configuredModel ?? provider.defaultModel ?? ""
    }

    func saveProvider() async {
        guard let provider = editingProvider else { return }

        isSaving = true
        errorMessage = nil
        successMessage = nil

        defer { isSaving = false }

        var body: [String: Any] = [
            "provider_id": provider.id,
            "api_key": editAPIKey,
        ]

        if !editModelName.isEmpty {
            body["model_name"] = editModelName
        }

        if provider.id == "custom" && !editCustomURL.isEmpty {
            body["custom_url"] = editCustomURL
        }

        let result = await post("/api/providers", body: body)

        if let success = result["success"] as? Bool, success {
            successMessage = "\(provider.name) configured successfully"
            editingProvider = nil
            // Reload providers to reflect changes
            await loadProviders()
        } else {
            errorMessage = result["error"] as? String ?? "Failed to save provider"
        }
    }

    // MARK: - Set Default Provider

    func setDefault(_ provider: ProviderInfo) async {
        errorMessage = nil
        successMessage = nil

        let result = await post("/api/providers/\(provider.id)/set-default", body: [:])

        if let success = result["success"] as? Bool, success {
            successMessage = "\(provider.name) set as default"
            await loadProviders()
        } else {
            errorMessage = result["error"] as? String ?? "Failed to set default provider"
        }
    }

    // MARK: - Delete Provider Config

    func deleteProviderConfig(_ provider: ProviderInfo) async {
        errorMessage = nil
        successMessage = nil

        let result = await delete("/api/providers/\(provider.id)")

        if let success = result["success"] as? Bool, success {
            successMessage = "\(provider.name) configuration removed"
            await loadProviders()
        } else {
            errorMessage = result["error"] as? String ?? "Failed to remove provider configuration"
        }
    }

    // MARK: - Network Helpers

    private func fetch(_ endpoint: String) async -> [String: Any] {
        do {
            guard let baseURL = apiClient.baseURL else { return [:] }
            let url = baseURL.appendingPathComponent(endpoint)

            var request = URLRequest(url: url)
            request.httpMethod = "GET"
            request.timeoutInterval = 30

            if let token = apiClient.sessionToken {
                request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
            }

            let (data, response) = try await urlSession.data(for: request)
            guard let httpResponse = response as? HTTPURLResponse, httpResponse.statusCode == 200 else {
                return [:]
            }

            return (try? JSONSerialization.jsonObject(with: data) as? [String: Any]) ?? [:]
        } catch {
            return [:]
        }
    }

    private func post(_ endpoint: String, body: [String: Any]) async -> [String: Any] {
        do {
            guard let baseURL = apiClient.baseURL else { return [:] }
            let url = baseURL.appendingPathComponent(endpoint)

            var request = URLRequest(url: url)
            request.httpMethod = "POST"
            request.timeoutInterval = 30
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")

            if let token = apiClient.sessionToken {
                request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
            }

            request.httpBody = try JSONSerialization.data(withJSONObject: body)

            let (data, response) = try await urlSession.data(for: request)
            guard let httpResponse = response as? HTTPURLResponse else { return [:] }

            let json = (try? JSONSerialization.jsonObject(with: data) as? [String: Any]) ?? [:]

            if httpResponse.statusCode >= 200 && httpResponse.statusCode < 300 {
                return json.merging(["success": true]) { _, new in new }
            } else {
                return json
            }
        } catch {
            return ["error": error.localizedDescription]
        }
    }

    private func delete(_ endpoint: String) async -> [String: Any] {
        do {
            guard let baseURL = apiClient.baseURL else { return [:] }
            let url = baseURL.appendingPathComponent(endpoint)

            var request = URLRequest(url: url)
            request.httpMethod = "DELETE"
            request.timeoutInterval = 30

            if let token = apiClient.sessionToken {
                request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
            }

            let (data, response) = try await urlSession.data(for: request)
            guard let httpResponse = response as? HTTPURLResponse else { return [:] }

            let json = (try? JSONSerialization.jsonObject(with: data) as? [String: Any]) ?? [:]

            if httpResponse.statusCode >= 200 && httpResponse.statusCode < 300 {
                return json.merging(["success": true]) { _, new in new }
            } else {
                return json
            }
        } catch {
            return ["error": error.localizedDescription]
        }
    }
}
