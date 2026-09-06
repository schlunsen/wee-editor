//
//  String+Formatting.swift
//  wee
//
//  Text formatting and truncation utilities
//

import Foundation

extension String {
    /// Counts the number of lines in the string
    func lineCount() -> Int {
        return components(separatedBy: .newlines).count
    }

    /// Truncates text to a maximum number of lines
    /// - Returns: The truncated string, or nil if no truncation is needed
    func truncated(to maxLines: Int = 4) -> String? {
        let lines = components(separatedBy: .newlines)
        guard lines.count > maxLines else { return nil }
        return lines.prefix(maxLines).joined(separator: "\n")
    }

    /// Returns initials from a name (e.g., "John Smith" -> "JS", "Bob" -> "BO")
    var initials: String {
        let components = split(separator: " ")
        if components.count >= 2 {
            return String(components[0].prefix(1)) + String(components[components.count - 1].prefix(1))
        } else if !isEmpty {
            return String(prefix(2)).uppercased()
        }
        return "??"
    }
}
