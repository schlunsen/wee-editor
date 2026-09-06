//
//  LocalConfig.swift
//  wee
//
//  Loads local configuration from LocalConfig.plist (gitignored).
//  Copy LocalConfig.template.plist → LocalConfig.plist and fill in your values.
//

import Foundation

enum LocalConfig {
    private static let config: [String: Any]? = {
        guard let url = Bundle.main.url(forResource: "LocalConfig", withExtension: "plist"),
              let data = try? Data(contentsOf: url),
              let dict = try? PropertyListSerialization.propertyList(from: data, format: nil) as? [String: Any]
        else { return nil }
        return dict
    }()

    /// Default server URL shown in the login form.
    /// Set via LocalConfig.plist key "DefaultServerURL", falls back to empty string.
    static var defaultServerURL: String {
        (config?["DefaultServerURL"] as? String) ?? ""
    }
}
