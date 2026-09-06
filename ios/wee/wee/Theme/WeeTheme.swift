//
//  WeeTheme.swift
//  wee
//
//  Centralized design tokens for the Wee iOS app.
//  Bridges the web dashboard's dark purple aesthetic with iOS native patterns.
//

import SwiftUI
import UIKit

// MARK: - Colors

enum WeeColors {

    // MARK: Primary Accent (Violet)

    /// Brand violet — the primary accent color, adaptive for light/dark mode.
    static let accent = Color(UIColor { traits in
        traits.userInterfaceStyle == .dark
            ? UIColor(red: 0.655, green: 0.545, blue: 0.98, alpha: 1)   // #a78bfa
            : UIColor(red: 0.486, green: 0.227, blue: 0.929, alpha: 1)  // #7c3aed
    })

    /// Lighter violet for subtle backgrounds, highlights.
    static let accentLight = Color(UIColor { traits in
        traits.userInterfaceStyle == .dark
            ? UIColor(red: 0.655, green: 0.545, blue: 0.98, alpha: 0.15)
            : UIColor(red: 0.486, green: 0.227, blue: 0.929, alpha: 0.10)
    })

    // MARK: Secondary Accent (Cyan)

    static let secondary = Color(UIColor { traits in
        traits.userInterfaceStyle == .dark
            ? UIColor(red: 0.404, green: 0.910, blue: 0.976, alpha: 1)  // #67e8f9
            : UIColor(red: 0.031, green: 0.569, blue: 0.698, alpha: 1)  // #0891b2
    })

    // MARK: Semantic Status Colors

    static let success = Color(UIColor { traits in
        traits.userInterfaceStyle == .dark
            ? UIColor(red: 0.29, green: 0.87, blue: 0.50, alpha: 1)     // #4ade80
            : UIColor(red: 0.086, green: 0.639, blue: 0.290, alpha: 1)  // #16a34a
    })

    static let warning = Color(UIColor { traits in
        traits.userInterfaceStyle == .dark
            ? UIColor(red: 0.984, green: 0.749, blue: 0.141, alpha: 1)  // #fbbf24
            : UIColor(red: 0.851, green: 0.467, blue: 0.024, alpha: 1)  // #d97706
    })

    static let error = Color(UIColor { traits in
        traits.userInterfaceStyle == .dark
            ? UIColor(red: 0.973, green: 0.443, blue: 0.443, alpha: 1)  // #f87171
            : UIColor(red: 0.863, green: 0.149, blue: 0.149, alpha: 1)  // #dc2626
    })

    // MARK: Surface Colors

    /// Page background — near-black with a slight purple shift in dark mode.
    static let background = Color(UIColor { traits in
        traits.userInterfaceStyle == .dark
            ? UIColor(red: 0.04, green: 0.04, blue: 0.06, alpha: 1)     // ~#0a0a0f
            : UIColor(red: 0.98, green: 0.98, blue: 1.0, alpha: 1)      // #fafaff
    })

    /// Card/elevated surface — subtle elevation in dark mode.
    static let surface = Color(UIColor { traits in
        traits.userInterfaceStyle == .dark
            ? UIColor(red: 0.078, green: 0.078, blue: 0.09, alpha: 1)   // ~#141417
            : UIColor(red: 1.0, green: 1.0, blue: 1.0, alpha: 1)
    })

    /// Tertiary surface for nested cards, input backgrounds.
    static let surfaceTertiary = Color(UIColor { traits in
        traits.userInterfaceStyle == .dark
            ? UIColor(red: 0.11, green: 0.11, blue: 0.14, alpha: 1)     // ~#1c1c24
            : UIColor(red: 0.96, green: 0.96, blue: 0.98, alpha: 1)
    })

    // MARK: Text Colors

    static let textPrimary = Color(UIColor { traits in
        traits.userInterfaceStyle == .dark
            ? UIColor(red: 0.95, green: 0.95, blue: 0.97, alpha: 1)
            : UIColor(red: 0.10, green: 0.10, blue: 0.12, alpha: 1)
    })

    static let textSecondary = Color(UIColor { traits in
        traits.userInterfaceStyle == .dark
            ? UIColor(red: 0.65, green: 0.65, blue: 0.70, alpha: 1)
            : UIColor(red: 0.40, green: 0.40, blue: 0.45, alpha: 1)
    })

    static let textTertiary = Color(UIColor { traits in
        traits.userInterfaceStyle == .dark
            ? UIColor(red: 0.45, green: 0.45, blue: 0.50, alpha: 1)
            : UIColor(red: 0.60, green: 0.60, blue: 0.65, alpha: 1)
    })
}

// MARK: - Gradients

enum WeeGradients {

    /// User message bubble gradient — violet to indigo.
    static let userBubble = LinearGradient(
        colors: [
            Color(red: 0.35, green: 0.45, blue: 0.95),
            Color(red: 0.5, green: 0.35, blue: 0.9)
        ],
        startPoint: .topLeading,
        endPoint: .bottomTrailing
    )

    /// Send button gradient — matches user bubble.
    static let sendButton = userBubble

    /// Assistant avatar background gradient.
    static let assistantAvatar = LinearGradient(
        colors: [
            Color.purple.opacity(0.7),
            Color.indigo.opacity(0.6)
        ],
        startPoint: .topLeading,
        endPoint: .bottomTrailing
    )

    /// Login button gradient.
    static let loginButton = LinearGradient(
        colors: [
            Color(red: 0.35, green: 0.45, blue: 0.95),
            Color(red: 0.5, green: 0.35, blue: 0.9)
        ],
        startPoint: .leading,
        endPoint: .trailing
    )

    /// Login background gradient (dark mode).
    static let loginBackground = LinearGradient(
        gradient: Gradient(colors: [
            Color(red: 0.06, green: 0.04, blue: 0.12),
            Color(red: 0.03, green: 0.03, blue: 0.05)
        ]),
        startPoint: .topLeading,
        endPoint: .bottomTrailing
    )

    /// Login background gradient (light mode).
    static let loginBackgroundLight = LinearGradient(
        gradient: Gradient(colors: [
            Color(red: 0.98, green: 0.97, blue: 1.0),
            Color(red: 0.95, green: 0.95, blue: 0.98)
        ]),
        startPoint: .topLeading,
        endPoint: .bottomTrailing
    )

    /// Empty state icon gradient.
    static let emptyState = LinearGradient(
        colors: [.indigo, .purple],
        startPoint: .topLeading,
        endPoint: .bottomTrailing
    )
}

// MARK: - Spacing

enum WeeSpacing {
    static let xxs: CGFloat = 2
    static let xs: CGFloat = 4
    static let sm: CGFloat = 8
    static let md: CGFloat = 12
    static let lg: CGFloat = 16
    static let xl: CGFloat = 24
    static let xxl: CGFloat = 32

    /// Standard card horizontal margin.
    static let cardMargin: CGFloat = 16
    /// Standard card internal horizontal padding.
    static let cardPaddingH: CGFloat = 14
    /// Standard card internal vertical padding.
    static let cardPaddingV: CGFloat = 12
}

// MARK: - Typography Helpers

enum WeeTypography {
    static let mono = Font.system(.caption2, design: .monospaced)
    static let monoSmall = Font.system(.caption2, design: .monospaced).weight(.medium)
    static let badge = Font.caption2.weight(.bold)
}

// MARK: - Status Color Helper

extension WeeColors {
    /// Map a status string to the appropriate brand color.
    static func statusColor(for status: String) -> Color {
        switch status.lowercased() {
        case "active", "running", "processing":
            return accent
        case "pending", "planning":
            return secondary
        case "completed":
            return success
        case "failed", "error":
            return error
        case "paused":
            return warning
        case "idle":
            return Color.gray
        default:
            return Color.gray
        }
    }
}
