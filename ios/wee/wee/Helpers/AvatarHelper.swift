//
//  AvatarHelper.swift
//  wee
//
//  Avatar helper for mapping session names to character avatars
//

import Foundation
import SwiftUI

// Character information for avatar display
struct CharacterInfo {
    let name: String
    let avatar: String
    let color: Color
}

// Avatar helper for iOS - mirrors the frontend useCharacterAvatar logic
class AvatarHelper {
    // Character mapping (case-insensitive)
    private static let characterMap: [String: CharacterInfo] = [
        "stan": CharacterInfo(name: "Stan Marsh", avatar: "stan", color: Color(red: 0.29, green: 0.56, blue: 0.89)),
        "kyle": CharacterInfo(name: "Kyle Broflovski", avatar: "kyle", color: Color(red: 0.15, green: 0.68, blue: 0.38)),
        "cartman": CharacterInfo(name: "Eric Cartman", avatar: "cartman", color: Color(red: 0.91, green: 0.30, blue: 0.24)),
        "kenny": CharacterInfo(name: "Kenny McCormick", avatar: "kenny", color: Color(red: 0.95, green: 0.61, blue: 0.07)),
        "butters": CharacterInfo(name: "Butters Stotch", avatar: "butters", color: Color(red: 0.98, green: 0.85, blue: 0.11)),
        "ike": CharacterInfo(name: "Ike Broflovski", avatar: "ike", color: Color(red: 0.61, green: 0.35, blue: 0.71)),
        "lke": CharacterInfo(name: "Ike Broflovski", avatar: "ike", color: Color(red: 0.61, green: 0.35, blue: 0.71)),
        "token": CharacterInfo(name: "Token Black", avatar: "token", color: Color(red: 0.20, green: 0.29, blue: 0.37)),
        "wendy": CharacterInfo(name: "Wendy Testaburger", avatar: "wendy", color: Color(red: 0.91, green: 0.12, blue: 0.39)),
        "timmy": CharacterInfo(name: "Timmy Burch", avatar: "timmy", color: Color(red: 0.21, green: 0.60, blue: 0.87)),
        "jimmy": CharacterInfo(name: "Jimmy Valmer", avatar: "jimmy", color: Color(red: 0.09, green: 0.63, blue: 0.52)),
        "randy": CharacterInfo(name: "Randy Marsh", avatar: "randy", color: Color(red: 0.56, green: 0.27, blue: 0.68)),
        "tweek": CharacterInfo(name: "Tweek Tweak", avatar: "tweek", color: Color(red: 0.84, green: 0.64, blue: 0.09)),
        "craig": CharacterInfo(name: "Craig Tucker", avatar: "craig", color: Color(red: 0.17, green: 0.24, blue: 0.31)),
        "sheila": CharacterInfo(name: "Sheila Broflovski", avatar: "sheila", color: Color(red: 0.90, green: 0.49, blue: 0.13)),
        "sharon": CharacterInfo(name: "Sharon Marsh", avatar: "sharon", color: Color(red: 0.75, green: 0.22, blue: 0.17)),
        "chef": CharacterInfo(name: "Chef", avatar: "chef", color: Color(red: 0.55, green: 0.27, blue: 0.08)),
        "mr-garrison": CharacterInfo(name: "Mr. Garrison", avatar: "mr-garrison", color: Color(red: 0.50, green: 0.55, blue: 0.56)),
        "mr-mackey": CharacterInfo(name: "Mr. Mackey", avatar: "mr-mackey", color: Color(red: 0.63, green: 0.32, blue: 0.18)),
        "mackey": CharacterInfo(name: "Mr. Mackey", avatar: "mr-mackey", color: Color(red: 0.63, green: 0.32, blue: 0.18)),
        "bebe": CharacterInfo(name: "Bebe Stevens", avatar: "bebe", color: Color(red: 1.0, green: 0.41, blue: 0.71)),
        "clyde": CharacterInfo(name: "Clyde Donovan", avatar: "clyde", color: Color(red: 0.36, green: 0.68, blue: 0.89)),
        "pc-principal": CharacterInfo(name: "PC Principal", avatar: "pc-principal", color: Color(red: 0.10, green: 0.73, blue: 0.61)),
        "towelie": CharacterInfo(name: "Towelie", avatar: "towelie", color: Color(red: 0.20, green: 0.29, blue: 0.37)),
        "mr-hankey": CharacterInfo(name: "Mr. Hankey", avatar: "mr-hankey", color: Color(red: 0.55, green: 0.27, blue: 0.08)),
        "big-gay-al": CharacterInfo(name: "Big Gay Al", avatar: "big-gay-al", color: Color(red: 0.91, green: 0.12, blue: 0.39)),
        "satan": CharacterInfo(name: "Satan", avatar: "satan", color: Color(red: 0.75, green: 0.22, blue: 0.17))
    ]

    // Ordered list of unique characters matching frontend order (Object.values in JavaScript maintains insertion order)
    private static let orderedUniqueCharacters: [CharacterInfo] = [
        CharacterInfo(name: "Stan Marsh", avatar: "stan", color: Color(red: 0.29, green: 0.56, blue: 0.89)),
        CharacterInfo(name: "Kyle Broflovski", avatar: "kyle", color: Color(red: 0.15, green: 0.68, blue: 0.38)),
        CharacterInfo(name: "Eric Cartman", avatar: "cartman", color: Color(red: 0.91, green: 0.30, blue: 0.24)),
        CharacterInfo(name: "Kenny McCormick", avatar: "kenny", color: Color(red: 0.95, green: 0.61, blue: 0.07)),
        CharacterInfo(name: "Butters Stotch", avatar: "butters", color: Color(red: 0.98, green: 0.85, blue: 0.11)),
        CharacterInfo(name: "Ike Broflovski", avatar: "ike", color: Color(red: 0.61, green: 0.35, blue: 0.71)),
        CharacterInfo(name: "Token Black", avatar: "token", color: Color(red: 0.20, green: 0.29, blue: 0.37)),
        CharacterInfo(name: "Wendy Testaburger", avatar: "wendy", color: Color(red: 0.91, green: 0.12, blue: 0.39)),
        CharacterInfo(name: "Timmy Burch", avatar: "timmy", color: Color(red: 0.21, green: 0.60, blue: 0.87)),
        CharacterInfo(name: "Jimmy Valmer", avatar: "jimmy", color: Color(red: 0.09, green: 0.63, blue: 0.52)),
        CharacterInfo(name: "Randy Marsh", avatar: "randy", color: Color(red: 0.56, green: 0.27, blue: 0.68)),
        CharacterInfo(name: "Tweek Tweak", avatar: "tweek", color: Color(red: 0.84, green: 0.64, blue: 0.09)),
        CharacterInfo(name: "Craig Tucker", avatar: "craig", color: Color(red: 0.17, green: 0.24, blue: 0.31)),
        CharacterInfo(name: "Sheila Broflovski", avatar: "sheila", color: Color(red: 0.90, green: 0.49, blue: 0.13)),
        CharacterInfo(name: "Sharon Marsh", avatar: "sharon", color: Color(red: 0.75, green: 0.22, blue: 0.17)),
        CharacterInfo(name: "Chef", avatar: "chef", color: Color(red: 0.55, green: 0.27, blue: 0.08)),
        CharacterInfo(name: "Mr. Garrison", avatar: "mr-garrison", color: Color(red: 0.50, green: 0.55, blue: 0.56)),
        CharacterInfo(name: "Mr. Mackey", avatar: "mr-mackey", color: Color(red: 0.63, green: 0.32, blue: 0.18)),
        CharacterInfo(name: "Bebe Stevens", avatar: "bebe", color: Color(red: 1.0, green: 0.41, blue: 0.71)),
        CharacterInfo(name: "Clyde Donovan", avatar: "clyde", color: Color(red: 0.36, green: 0.68, blue: 0.89)),
        CharacterInfo(name: "PC Principal", avatar: "pc-principal", color: Color(red: 0.10, green: 0.73, blue: 0.61)),
        CharacterInfo(name: "Towelie", avatar: "towelie", color: Color(red: 0.20, green: 0.29, blue: 0.37)),
        CharacterInfo(name: "Mr. Hankey", avatar: "mr-hankey", color: Color(red: 0.55, green: 0.27, blue: 0.08)),
        CharacterInfo(name: "Big Gay Al", avatar: "big-gay-al", color: Color(red: 0.91, green: 0.12, blue: 0.39)),
        CharacterInfo(name: "Satan", avatar: "satan", color: Color(red: 0.75, green: 0.22, blue: 0.17))
    ]

    private static let defaultCharacter = CharacterInfo(
        name: "Unknown",
        avatar: "default",
        color: Color(red: 0.58, green: 0.65, blue: 0.65)
    )

    /// FNV-1a hash function for better distribution
    /// Mirrors the frontend's improvedHash function
    private static func improvedHash(_ str: String) -> UInt32 {
        let FNV_OFFSET_BASIS: UInt32 = 2166136261
        let FNV_PRIME: UInt32 = 16777619

        var hash = FNV_OFFSET_BASIS

        for char in str.utf8 {
            hash ^= UInt32(char)
            hash = hash &* FNV_PRIME
        }

        return hash
    }

    /// Get all available characters (in consistent order matching frontend)
    static func getAllCharacters() -> [CharacterInfo] {
        return orderedUniqueCharacters
    }

    /// Get character avatar information for a session name or ID
    /// - Parameter sessionName: The session name/ID to look up
    /// - Returns: Character information including name, avatar path, and color
    static func getCharacterAvatar(for sessionName: String?) -> CharacterInfo {
        guard let sessionName = sessionName, !sessionName.isEmpty else {
            return defaultCharacter
        }

        // Normalize session name (lowercase, trim whitespace)
        let normalizedName = sessionName.lowercased().trimmingCharacters(in: .whitespaces)

        // First, try direct character name lookup
        if let character = characterMap[normalizedName] {
            return character
        }

        // If not found, use hash to map session ID to a character
        let characters = getAllCharacters()
        guard !characters.isEmpty else {
            return defaultCharacter
        }

        let hash = improvedHash(sessionName)
        let index = Int(hash) % characters.count

        return characters[index]
    }

    /// Check if a session name has a mapped character
    /// - Parameter sessionName: The session name to check
    /// - Returns: True if character exists, false otherwise
    static func hasCharacter(_ sessionName: String?) -> Bool {
        guard let sessionName = sessionName, !sessionName.isEmpty else {
            return false
        }

        let normalizedName = sessionName.lowercased().trimmingCharacters(in: .whitespaces)
        return characterMap[normalizedName] != nil
    }

    /// Convert backend Avatar to CharacterInfo
    /// - Parameter avatar: The Avatar from the backend
    /// - Returns: CharacterInfo for UI display
    static func avatarToCharacterInfo(_ avatar: Avatar) -> CharacterInfo {
        let color = parseHexColor(avatar.color) ?? Color(red: 0.58, green: 0.65, blue: 0.65)
        return CharacterInfo(
            name: avatar.name,
            avatar: (avatar.image_path ?? avatar.name).lowercased(),
            color: color
        )
    }

    /// Parse hex color string to Color
    /// Supports formats like "#FF69B4", "FF69B4", or fallback to default
    private static func parseHexColor(_ hexString: String?) -> Color? {
        guard let hexString = hexString else { return nil }

        var hex = hexString.trimmingCharacters(in: .whitespaces)
        if hex.hasPrefix("#") {
            hex.removeFirst()
        }

        guard hex.count == 6 else { return nil }

        let scanner = Scanner(string: hex)
        var rgbValue: UInt64 = 0

        guard scanner.scanHexInt64(&rgbValue) else { return nil }

        let r = Double((rgbValue >> 16) & 0xFF) / 255.0
        let g = Double((rgbValue >> 8) & 0xFF) / 255.0
        let b = Double(rgbValue & 0xFF) / 255.0

        return Color(red: r, green: g, blue: b)
    }

    /// Get avatar system name for SwiftUI Image
    /// Returns a safe system name or a fallback
    static func getAvatarImageName(for character: CharacterInfo) -> String {
        // For iOS, we'll use SF Symbols for avatars
        // Map character names to appropriate SF Symbols
        let symbolMap: [String: String] = [
            "stan": "person.fill",
            "kyle": "person.fill",
            "cartman": "person.fill",
            "kenny": "person.fill",
            "butters": "person.fill",
            "ike": "person.fill",
            "token": "person.fill",
            "wendy": "person.fill",
            "timmy": "person.fill",
            "jimmy": "person.fill",
            "randy": "person.fill",
            "tweek": "person.fill",
            "craig": "person.fill",
            "sheila": "person.fill",
            "sharon": "person.fill",
            "chef": "person.fill",
            "mr-garrison": "person.fill",
            "mr-mackey": "person.fill",
            "bebe": "person.fill",
            "clyde": "person.fill",
            "pc-principal": "person.fill",
            "towelie": "person.fill",
            "mr-hankey": "person.fill",
            "big-gay-al": "person.fill",
            "satan": "person.fill",
            "default": "person.crop.circle"
        ]

        return symbolMap[character.avatar] ?? "person.crop.circle"
    }
}
