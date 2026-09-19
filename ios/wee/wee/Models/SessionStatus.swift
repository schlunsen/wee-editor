//
//  SessionStatus.swift
//  wee
//
//  Session status enumeration with UI helpers
//

import SwiftUI

enum SessionStatus: String, CaseIterable {
    case idle
    case processing
    case thinking
    case completed
    case error

    init(_ string: String) {
        self = SessionStatus(rawValue: string.lowercased()) ?? .idle
    }

    /// The color representing this status
    var color: Color {
        switch self {
        case .processing, .thinking:
            return WeeColors.accent
        case .completed:
            return WeeColors.success
        case .error:
            return WeeColors.error
        case .idle:
            return .gray
        }
    }

    /// Human-readable label for the status
    var label: String {
        switch self {
        case .processing, .thinking:
            return "Processing..."
        case .completed, .idle:
            return "Idle"
        case .error:
            return "Error"
        }
    }

    /// Whether the session is currently active/processing
    var isActive: Bool {
        self == .processing || self == .thinking
    }
}
