//
//  HostManager.swift
//  wee
//
//  Service for managing multiple Wee server hosts
//

import Foundation
import Combine

@MainActor
class HostManager: ObservableObject {
    static let shared = HostManager()

    @Published var hosts: [Host] = []
    @Published var currentHostId: String? {
        didSet {
            Task {
                await saveCurrentHostPreference()
            }
        }
    }

    private let secureStorage = SecureStorage.shared
    private let defaults = UserDefaults.standard
    private let hostsKey = "savedHosts"
    private let currentHostKey = "currentHostId"
    private let defaultHostKey = "defaultHostId"

    private init() {
        Task {
            await loadHosts()
            await loadCurrentHostPreference()
        }
    }

    // MARK: - Host Management

    /// Add a new host configuration
    @discardableResult
    func addHost(name: String, url: String, username: String? = nil) async throws -> Host {
        let normalizedURL = Host.normalizeURL(url)

        // Validate URL format
        guard URL(string: normalizedURL) != nil else {
            throw HostError.invalidURL
        }

        let host = Host(
            name: name,
            url: normalizedURL,
            username: username
        )

        hosts.append(host)
        await saveHosts()

        return host
    }

    /// Update an existing host
    func updateHost(id: String, name: String, url: String, username: String? = nil) async throws {
        let normalizedURL = Host.normalizeURL(url)

        guard URL(string: normalizedURL) != nil else {
            throw HostError.invalidURL
        }

        if let index = hosts.firstIndex(where: { $0.id == id }) {
            var host = hosts[index]
            host = Host(
                id: host.id,
                name: name,
                url: normalizedURL,
                username: username,
                createdAt: host.createdAt,
                updatedAt: Date(),
                lastUsedAt: host.lastUsedAt
            )
            hosts[index] = host
            await saveHosts()
        }
    }

    /// Delete a host configuration
    func deleteHost(id: String) async throws {
        // Don't allow deleting the current host
        if currentHostId == id {
            currentHostId = nil
        }

        hosts.removeAll { $0.id == id }
        await saveHosts()

        // Delete associated credentials
        try secureStorage.deleteHostCredential(hostId: id)
    }

    /// Get current host
    var currentHost: Host? {
        guard let hostId = currentHostId else { return nil }
        return hosts.first { $0.id == hostId }
    }

    /// Switch to a different host
    func switchHost(id: String) async throws {
        guard hosts.contains(where: { $0.id == id }) else {
            throw HostError.hostNotFound
        }
        currentHostId = id
    }

    /// Record that a host was just used
    func recordHostUse(id: String) async {
        if let index = hosts.firstIndex(where: { $0.id == id }) {
            let updatedHost = hosts[index].withUpdatedLastUsed()
            hosts[index] = updatedHost
            await saveHosts()
        }
    }

    /// Set default host for auto-selection
    func setDefaultHost(id: String) async throws {
        guard hosts.contains(where: { $0.id == id }) else {
            throw HostError.hostNotFound
        }
        defaults.set(id, forKey: defaultHostKey)
    }

    /// Get default host
    var defaultHostId: String? {
        defaults.string(forKey: defaultHostKey)
    }

    // MARK: - Persistence

    private func saveHosts() async {
        do {
            let encoder = JSONEncoder()
            let data = try encoder.encode(hosts)
            try secureStorage.store(value: String(data: data, encoding: .utf8) ?? "", forKey: hostsKey)
        } catch {
        }
    }

    private func loadHosts() async {
        do {
            if let jsonString = try secureStorage.retrieve(key: hostsKey),
               let data = jsonString.data(using: .utf8) {
                let decoder = JSONDecoder()
                hosts = try decoder.decode([Host].self, from: data)
            } else {
                hosts = []
            }
        } catch {
            hosts = []
        }
    }

    private func saveCurrentHostPreference() async {
        if let hostId = currentHostId {
            defaults.set(hostId, forKey: currentHostKey)
        } else {
            defaults.removeObject(forKey: currentHostKey)
        }
    }

    private func loadCurrentHostPreference() async {
        if let hostId = defaults.string(forKey: currentHostKey) {
            // Verify the host still exists
            if hosts.contains(where: { $0.id == hostId }) {
                self.currentHostId = hostId
            } else {
                // Host was deleted, load default or first available
                if let defaultId = defaultHostId, hosts.contains(where: { $0.id == defaultId }) {
                    self.currentHostId = defaultId
                } else if !hosts.isEmpty {
                    self.currentHostId = hosts[0].id
                }
            }
        } else if let defaultId = defaultHostId, hosts.contains(where: { $0.id == defaultId }) {
            // No current host set, use default
            self.currentHostId = defaultId
        } else if !hosts.isEmpty {
            // Use first available host
            self.currentHostId = hosts[0].id
        }
    }

    /// Clear all hosts (used on logout)
    func clearAllHosts() async throws {
        hosts.removeAll()
        currentHostId = nil
        defaults.removeObject(forKey: currentHostKey)
        defaults.removeObject(forKey: defaultHostKey)

        try secureStorage.deleteAll()
    }
}

// MARK: - Error Types

enum HostError: LocalizedError {
    case invalidURL
    case hostNotFound
    case failedToLoad
    case failedToSave

    var errorDescription: String? {
        switch self {
        case .invalidURL:
            return "Invalid server URL format"
        case .hostNotFound:
            return "Host configuration not found"
        case .failedToLoad:
            return "Failed to load host configurations"
        case .failedToSave:
            return "Failed to save host configurations"
        }
    }
}
