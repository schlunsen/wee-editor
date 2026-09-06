//
//  SecureStorage.swift
//  wee
//
//  Secure storage for credentials using Keychain
//

import Foundation
import Security

/// Service for securely storing and retrieving sensitive data using Keychain
class SecureStorage {
    static let shared = SecureStorage()

    private let service = "com.wee.app"
    private let hostCredentialPrefix = "host_credential_"

    private init() {}

    /// Store a value securely in Keychain
    func store(value: String, forKey key: String) throws {
        let data = value.data(using: .utf8)!

        // Delete existing value first
        let deleteQuery: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: key,
        ]

        SecItemDelete(deleteQuery as CFDictionary)

        // Add new value
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: key,
            kSecValueData as String: data,
            kSecAttrAccessible as String: kSecAttrAccessibleWhenUnlockedThisDeviceOnly,
        ]

        let status = SecItemAdd(query as CFDictionary, nil)

        guard status == errSecSuccess else {
            throw KeychainError.storeFailed(status)
        }
    }

    /// Retrieve a value from Keychain
    func retrieve(key: String) throws -> String? {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: key,
            kSecReturnData as String: true,
        ]

        var result: AnyObject?
        let status = SecItemCopyMatching(query as CFDictionary, &result)

        guard status == errSecSuccess else {
            if status == errSecItemNotFound {
                return nil
            }
            throw KeychainError.retrieveFailed(status)
        }

        guard let data = result as? Data else {
            return nil
        }

        return String(data: data, encoding: .utf8)
    }

    /// Delete a value from Keychain
    func delete(key: String) throws {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: key,
        ]

        let status = SecItemDelete(query as CFDictionary)

        guard status == errSecSuccess || status == errSecItemNotFound else {
            throw KeychainError.deleteFailed(status)
        }
    }

    /// Clear all stored credentials
    func deleteAll() throws {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
        ]

        let status = SecItemDelete(query as CFDictionary)

        guard status == errSecSuccess || status == errSecItemNotFound else {
            throw KeychainError.deleteFailed(status)
        }
    }

    // MARK: - Host-Specific Credential Methods

    /// Store session token for a specific host
    func storeHostCredential(hostId: String, sessionToken: String, expiresAt: Date? = nil) throws {
        let credential = HostCredential(hostId: hostId, sessionToken: sessionToken, expiresAt: expiresAt)
        let encoder = JSONEncoder()
        let data = try encoder.encode(credential)
        let jsonString = String(data: data, encoding: .utf8) ?? ""

        try store(value: jsonString, forKey: "\(hostCredentialPrefix)\(hostId)")
    }

    /// Retrieve session token for a specific host
    func retrieveHostCredential(hostId: String) throws -> HostCredential? {
        guard let jsonString = try retrieve(key: "\(hostCredentialPrefix)\(hostId)") else {
            return nil
        }

        guard let data = jsonString.data(using: .utf8) else {
            return nil
        }

        let decoder = JSONDecoder()
        return try decoder.decode(HostCredential.self, from: data)
    }

    /// Delete session token for a specific host
    func deleteHostCredential(hostId: String) throws {
        try delete(key: "\(hostCredentialPrefix)\(hostId)")
    }

    /// List all stored host IDs with credentials
    func getAllHostCredentialIds() -> [String] {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecReturnAttributes as String: true,
            kSecMatchLimit as String: kSecMatchLimitAll,
        ]

        var result: AnyObject?
        let status = SecItemCopyMatching(query as CFDictionary, &result)

        guard status == errSecSuccess else {
            return []
        }

        guard let items = result as? [[String: Any]] else {
            return []
        }

        return items.compactMap { item in
            guard let account = item[kSecAttrAccount as String] as? String else {
                return nil
            }
            if account.hasPrefix(hostCredentialPrefix) {
                return String(account.dropFirst(hostCredentialPrefix.count))
            }
            return nil
        }
    }
}

// MARK: - Error Types

enum KeychainError: LocalizedError {
    case storeFailed(OSStatus)
    case retrieveFailed(OSStatus)
    case deleteFailed(OSStatus)

    var errorDescription: String? {
        switch self {
        case .storeFailed(let status):
            return "Failed to store credential: \(SecCopyErrorMessageString(status, nil) as String? ?? "Unknown error")"
        case .retrieveFailed(let status):
            return "Failed to retrieve credential: \(SecCopyErrorMessageString(status, nil) as String? ?? "Unknown error")"
        case .deleteFailed(let status):
            return "Failed to delete credential: \(SecCopyErrorMessageString(status, nil) as String? ?? "Unknown error")"
        }
    }
}
