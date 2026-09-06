//
//  WeeHaptics.swift
//  wee
//
//  Centralized haptic feedback for consistent tactile responses.
//

import UIKit

enum WeeHaptics {

    // MARK: - Feedback Generators (reuse for performance)

    private static let lightImpact = UIImpactFeedbackGenerator(style: .light)
    private static let mediumImpact = UIImpactFeedbackGenerator(style: .medium)
    private static let heavyImpact = UIImpactFeedbackGenerator(style: .heavy)
    private static let notification = UINotificationFeedbackGenerator()
    private static let selection = UISelectionFeedbackGenerator()

    // MARK: - Semantic Haptics

    /// Light tap — message sent, toggle, expand/collapse.
    static func tap() {
        lightImpact.impactOccurred()
    }

    /// Medium tap — selection, long-press action.
    static func select() {
        mediumImpact.impactOccurred()
    }

    /// Heavy tap — interrupt session, destructive action confirmation.
    static func heavy() {
        heavyImpact.impactOccurred()
    }

    /// Success — login, session created, session completed.
    static func success() {
        notification.notificationOccurred(.success)
    }

    /// Warning — session idle too long, cost threshold.
    static func warning() {
        notification.notificationOccurred(.warning)
    }

    /// Error — login failed, session error, connection lost.
    static func error() {
        notification.notificationOccurred(.error)
    }

    /// Selection change — tab switch, host switch, picker change.
    static func selectionChanged() {
        selection.selectionChanged()
    }

    // MARK: - Prepare (call before expected interaction for zero-latency)

    static func prepare() {
        lightImpact.prepare()
        notification.prepare()
    }
}
