//
//  WeeButtonStyles.swift
//  wee
//
//  Custom button styles for tactile, branded interactions.
//

import SwiftUI

// MARK: - Card Press Style

/// Subtle scale-down on press for card-like buttons. Adds haptic feedback.
struct CardPressStyle: ButtonStyle {
    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .scaleEffect(configuration.isPressed ? 0.97 : 1.0)
            .opacity(configuration.isPressed ? 0.9 : 1.0)
            .animation(.easeOut(duration: 0.15), value: configuration.isPressed)
    }
}

// MARK: - Primary Button Style

/// Branded button with scale + shadow animation (for Login, Send, etc.)
struct WeePrimaryButtonStyle: ButtonStyle {
    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .scaleEffect(configuration.isPressed ? 0.96 : 1.0)
            .opacity(configuration.isPressed ? 0.85 : 1.0)
            .animation(.spring(response: 0.25, dampingFraction: 0.7), value: configuration.isPressed)
    }
}

// MARK: - Metric Pill Style

/// For tappable metric pills — subtle brightness shift.
struct MetricPillStyle: ButtonStyle {
    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .brightness(configuration.isPressed ? 0.1 : 0)
            .scaleEffect(configuration.isPressed ? 0.95 : 1.0)
            .animation(.easeOut(duration: 0.12), value: configuration.isPressed)
    }
}

// MARK: - View Extension

extension View {
    func cardPressStyle() -> some View {
        self.buttonStyle(CardPressStyle())
    }

    func weePrimaryStyle() -> some View {
        self.buttonStyle(WeePrimaryButtonStyle())
    }
}
