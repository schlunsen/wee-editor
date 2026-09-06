//
//  ViewTransitions.swift
//  wee
//
//  Reusable transition modifiers for smooth page and content changes
//

import SwiftUI

// MARK: - Transition Animation Configuration

enum TransitionStyle {
    /// Fade in/out effect - subtle and clean
    case fade
    /// Fade + scale effect - adds depth
    case fadeScale
    /// Fade + move from bottom - cards sliding up
    case slideUp
    /// Fade + move from sides - directional entry
    case slideHorizontal(edge: Edge)
    /// Complex animation - fade + scale + move
    case complex
}

// MARK: - Custom Transition Extension

extension AnyTransition {
    /// Fade transition (opacity only)
    static var fadeTransition: AnyTransition {
        .opacity
    }

    /// Fade with scale - good for content appearing/disappearing
    static var fadeScaleTransition: AnyTransition {
        .asymmetric(
            insertion: .opacity.combined(with: .scale(scale: 0.95)),
            removal: .opacity.combined(with: .scale(scale: 0.95))
        )
    }

    /// Slide up from bottom - good for cards
    static var slideUpTransition: AnyTransition {
        .asymmetric(
            insertion: .opacity.combined(with: .move(edge: .bottom)).combined(with: .scale(scale: 0.95)),
            removal: .opacity.combined(with: .move(edge: .bottom))
        )
    }

    /// Slide from leading edge - good for list items
    static var slideLeadingTransition: AnyTransition {
        .asymmetric(
            insertion: .opacity.combined(with: .move(edge: .leading)).combined(with: .scale(scale: 0.95)),
            removal: .opacity.combined(with: .move(edge: .leading))
        )
    }

    /// Slide from trailing edge - good for sequential items
    static var slideTrailingTransition: AnyTransition {
        .asymmetric(
            insertion: .opacity.combined(with: .move(edge: .trailing)).combined(with: .scale(scale: 0.95)),
            removal: .opacity.combined(with: .move(edge: .trailing))
        )
    }

    /// Complex transition with multiple effects
    static var complexTransition: AnyTransition {
        .asymmetric(
            insertion: .opacity.combined(with: .scale(scale: 0.85)).combined(with: .move(edge: .bottom)),
            removal: .opacity.combined(with: .scale(scale: 0.9))
        )
    }

    /// Get transition for a specific style
    static func transition(for style: TransitionStyle) -> AnyTransition {
        switch style {
        case .fade:
            return .fadeTransition
        case .fadeScale:
            return .fadeScaleTransition
        case .slideUp:
            return .slideUpTransition
        case .slideHorizontal(let edge):
            switch edge {
            case .leading:
                return .slideLeadingTransition
            case .trailing:
                return .slideTrailingTransition
            case .top:
                return .asymmetric(
                    insertion: .opacity.combined(with: .move(edge: .top)),
                    removal: .opacity
                )
            case .bottom:
                return .slideUpTransition
            }
        case .complex:
            return .complexTransition
        }
    }
}

// MARK: - View Extension for Easy Application

extension View {
    /// Apply smooth transition with animation
    func smoothTransition(_ style: TransitionStyle = .fade, duration: Double = 0.3) -> some View {
        self
            .transition(.transition(for: style))
            .animation(.easeInOut(duration: duration), value: UUID())
    }

    /// Apply fade transition
    func fadeTransition(duration: Double = 0.3) -> some View {
        self
            .transition(.fadeTransition)
            .animation(.easeInOut(duration: duration), value: UUID())
    }

    /// Apply fade + scale transition (good for state changes)
    func fadeScaleTransition(duration: Double = 0.3) -> some View {
        self
            .transition(.fadeScaleTransition)
            .animation(.easeInOut(duration: duration), value: UUID())
    }

    /// Apply slide up transition (good for cards)
    func slideUpTransition(duration: Double = 0.3) -> some View {
        self
            .transition(.slideUpTransition)
            .animation(.easeInOut(duration: duration), value: UUID())
    }

    /// Apply slide from edge transition
    func slideHorizontalTransition(_ edge: Edge, duration: Double = 0.3) -> some View {
        self
            .transition(.transition(for: .slideHorizontal(edge: edge)))
            .animation(.easeInOut(duration: duration), value: UUID())
    }

    /// Apply complex transition for maximum visual impact
    func complexTransition(duration: Double = 0.4) -> some View {
        self
            .transition(.complexTransition)
            .animation(.easeInOut(duration: duration), value: UUID())
    }
}

// MARK: - Tab Transition Helpers

extension Binding {
    /// Create animated binding for tab changes
    /// Usage: TabView(selection: $selectedTab.withTabAnimation())
    func withTabAnimation(duration: Double = 0.3) -> Binding<Value> {
        Binding(
            get: { self.wrappedValue },
            set: { newValue in
                withAnimation(.easeInOut(duration: duration)) {
                    self.wrappedValue = newValue
                }
            }
        )
    }

    /// Add haptic feedback on value change.
    /// Usage: $value.withHaptics()
    func withHaptics() -> Binding<Value> where Value: Equatable {
        Binding(
            get: { self.wrappedValue },
            set: { newValue in
                if newValue != self.wrappedValue {
                    WeeHaptics.selectionChanged()
                }
                self.wrappedValue = newValue
            }
        )
    }
}

// MARK: - Custom Animation Curves

extension Animation {
    /// Smooth easing for transitions
    static var smoothTransition: Animation {
        .easeInOut(duration: 0.3)
    }

    /// Snappier transition for UI feedback
    static var snappyTransition: Animation {
        .easeOut(duration: 0.2)
    }

    /// Springy transition for playful feel
    static var springyTransition: Animation {
        .spring(response: 0.4, dampingFraction: 0.8)
    }

    /// Slower transition for deliberate, careful feel
    static var slowTransition: Animation {
        .easeInOut(duration: 0.5)
    }
}

// MARK: - Conditional Transitions

extension View {
    /// Apply transition only when condition is true
    @ViewBuilder
    func transitionIf(_ condition: Bool, _ transition: AnyTransition) -> some View {
        if condition {
            self.transition(transition)
        } else {
            self
        }
    }

    /// Apply different transitions based on condition
    func conditionalTransition(_ transition1: AnyTransition, _ transition2: AnyTransition, condition: Bool) -> some View {
        self.transition(condition ? transition1 : transition2)
    }
}

// MARK: - Staggered List Transitions

/// Helper to create staggered transitions for list items
struct StaggeredTransitionModifier: ViewModifier {
    let index: Int
    let totalCount: Int
    let style: TransitionStyle
    let staggerDelay: Double

    func body(content: Content) -> some View {
        let delay = Double(index) * staggerDelay
        content
            .transition(.transition(for: style))
            .animation(.easeInOut(duration: 0.3).delay(delay), value: UUID())
    }
}

extension View {
    /// Apply staggered transition for list items
    /// Usage: ForEach(items.indices) { i in
    ///   ItemView(items[i])
    ///     .staggeredTransition(index: i, totalCount: items.count)
    /// }
    func staggeredTransition(
        index: Int,
        totalCount: Int,
        style: TransitionStyle = .slideUp,
        staggerDelay: Double = 0.05
    ) -> some View {
        modifier(StaggeredTransitionModifier(
            index: index,
            totalCount: totalCount,
            style: style,
            staggerDelay: staggerDelay
        ))
    }
}
