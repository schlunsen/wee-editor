//
//  String+Date.swift
//  wee
//
//  Date parsing and formatting utilities
//

import Foundation

extension String {
    /// Parses an ISO8601 date string using multiple format variations
    func parseISO8601Date() -> Date? {
        let formatters: [ISO8601DateFormatter] = [
            ISO8601DateFormatter(),
            {
                let formatter = ISO8601DateFormatter()
                formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
                return formatter
            }(),
            {
                let formatter = ISO8601DateFormatter()
                formatter.formatOptions = [.withInternetDateTime]
                return formatter
            }()
        ]

        for formatter in formatters {
            if let date = formatter.date(from: self) {
                return date
            }
        }

        // Fallback: try to parse with standard formatter
        let dateFormatter = DateFormatter()
        dateFormatter.dateFormat = "yyyy-MM-dd'T'HH:mm:ss.SSSSSSZ"
        if let date = dateFormatter.date(from: self) {
            return date
        }

        return nil
    }

    /// Formats an ISO8601 date string to a short time format
    func formattedTime() -> String {
        guard let date = parseISO8601Date() else {
            return "Unknown"
        }

        let timeFormatter = DateFormatter()
        timeFormatter.timeStyle = .short
        return timeFormatter.string(from: date)
    }
}
