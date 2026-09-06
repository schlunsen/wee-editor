//
//  weeApp.swift
//  wee
//
//  Created by Rasmus Schlunsen on 2/11/25.
//

import SwiftUI

// MARK: - Appearance Mode

enum AppearanceMode: String, CaseIterable, Identifiable {
    case system = "system"
    case light = "light"
    case dark = "dark"

    var id: String { rawValue }

    var label: String {
        switch self {
        case .system: return "System"
        case .light: return "Light"
        case .dark: return "Dark"
        }
    }

    var icon: String {
        switch self {
        case .system: return "circle.lefthalf.filled"
        case .light: return "sun.max.fill"
        case .dark: return "moon.fill"
        }
    }

    var colorScheme: ColorScheme? {
        switch self {
        case .system: return nil
        case .light: return .light
        case .dark: return .dark
        }
    }
}

@main
struct weeApp: App {
    @UIApplicationDelegateAdaptor(AppDelegate.self) var appDelegate
    @StateObject private var authViewModel = AuthenticationViewModel()
    @AppStorage("appearanceMode") private var appearanceMode: String = AppearanceMode.system.rawValue
    private let avatarManager = AvatarManager.shared

    private var selectedColorScheme: ColorScheme? {
        AppearanceMode(rawValue: appearanceMode)?.colorScheme
    }

    var body: some Scene {
        WindowGroup {
            if authViewModel.isAuthenticated {
                MainTabView()
                    .environmentObject(authViewModel)
                    .environmentObject(avatarManager)
                    .preferredColorScheme(selectedColorScheme)
                    .task {
                        // Load avatars when authenticated
                        await avatarManager.loadAvatars()
                    }
            } else {
                LoginView()
                    .environmentObject(authViewModel)
                    .preferredColorScheme(selectedColorScheme)
            }
        }
    }
}
