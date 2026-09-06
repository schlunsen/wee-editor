//
//  AvatarManager.swift
//  wee
//
//  Manages avatar caching and lookup with persistent disk storage
//

import Foundation
import SwiftUI
import Combine
import UIKit

@MainActor
class AvatarManager: NSObject, ObservableObject {
    static let shared = AvatarManager()

    @Published var avatarMap: [Int64: Avatar] = [:]
    @Published var avatarImages: [Int64: UIImage] = [:]
    @Published var isLoading = false
    @Published var errorMessage: String?

    private let apiClient = WeeAPIClient.shared
    private let cacheDirectory: URL

    private override init() {
        // Set up cache directory in Caches folder
        let paths = FileManager.default.urls(for: .cachesDirectory, in: .userDomainMask)
        self.cacheDirectory = paths[0].appendingPathComponent("AvatarImages")

        // Create cache directory if it doesn't exist
        try? FileManager.default.createDirectory(at: cacheDirectory, withIntermediateDirectories: true)

        super.init()
    }

    /// Load avatars from all themes and cache them
    func loadAvatars() async {
        isLoading = true
        errorMessage = nil

        defer { isLoading = false }

        do {
            // Get all available themes
            let themes = try await apiClient.getAvatarThemes()

            // Create a map for quick lookup by ID
            var map: [Int64: Avatar] = [:]

            // Load avatars for each theme
            for theme in themes {
                let avatars = try await apiClient.getThemeAvatars(themeId: theme.id)
                for avatar in avatars {
                    map[avatar.id] = avatar
                }
            }

            self.avatarMap = map

            // Load images for all avatars in background (with disk caching)
            await loadAvatarImages(for: Array(map.values))
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    /// Load avatar images for given avatars (with disk caching)
    private func loadAvatarImages(for avatars: [Avatar]) async {
        var images: [Int64: UIImage] = [:]

        #if DEBUG
        print("🖼️ Starting to load \(avatars.count) avatar images...")
        #endif

        for avatar in avatars {
            // Try to load from disk cache first
            if let cachedImage = loadImageFromDisk(avatarId: avatar.id) {
                images[avatar.id] = cachedImage
                continue
            }

            // Download from API
            do {
                let imageData: Data

                imageData = try await apiClient.getAvatarImage(avatarId: avatar.id)

                if let uiImage = UIImage(data: imageData) {
                    images[avatar.id] = uiImage
                    // Save to disk cache
                    saveImageToDisk(imageData, avatarId: avatar.id)
                } else {
                    #if DEBUG
                    print("⚠️ Failed to create UIImage from data for \(avatar.name) (ID: \(avatar.id)), data size: \(imageData.count)")
                    #endif
                }
            } catch {
                #if DEBUG
                print("❌ Error loading avatar image for \(avatar.name) (ID: \(avatar.id)): \(error.localizedDescription)")
                #endif
                continue
            }
        }

        #if DEBUG
        print("📊 Total avatar images loaded: \(images.count) / \(avatars.count)")
        #endif
        self.avatarImages = images
    }

    /// Save image to disk cache
    private func saveImageToDisk(_ imageData: Data, avatarId: Int64) {
        let fileURL = cacheDirectory.appendingPathComponent("\(avatarId).png")
        try? imageData.write(to: fileURL)
    }

    /// Load image from disk cache
    private func loadImageFromDisk(avatarId: Int64) -> UIImage? {
        let fileURL = cacheDirectory.appendingPathComponent("\(avatarId).png")
        guard let imageData = try? Data(contentsOf: fileURL) else {
            return nil
        }
        return UIImage(data: imageData)
    }

    /// Get avatar by ID from cache
    func getAvatar(id: Int64) -> Avatar? {
        return avatarMap[id]
    }

    /// Get avatar image by ID from memory cache (already loaded)
    func getAvatarImage(id: Int64) -> UIImage? {
        return avatarImages[id]
    }

    /// Load a single avatar image on-demand (from disk cache or server).
    /// Call this when you have an avatar from the session but it's not in the memory cache yet.
    func loadAvatarImageIfNeeded(avatar: Avatar) async {
        // Already in memory cache
        if avatarImages[avatar.id] != nil { return }

        // Try disk cache
        if let cached = loadImageFromDisk(avatarId: avatar.id) {
            avatarImages[avatar.id] = cached
            return
        }

        do {
            let imageData: Data

            imageData = try await apiClient.getAvatarImage(avatarId: avatar.id)

            if let uiImage = UIImage(data: imageData) {
                avatarImages[avatar.id] = uiImage
                saveImageToDisk(imageData, avatarId: avatar.id)
            }
        } catch {
            #if DEBUG
            print("❌ Failed to load avatar image for \(avatar.name): \(error.localizedDescription)")
            #endif
        }
    }
}
