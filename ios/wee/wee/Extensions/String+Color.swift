//
//  String+Color.swift
//  wee
//
//  Color parsing utilities for hex strings
//

import SwiftUI

extension String {
    /// Parses a hex color string (e.g., "#FF69B4" or "FF69B4") into a SwiftUI Color
    func parseHexColor() -> Color? {
        var hex = trimmingCharacters(in: .whitespaces)
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
}

extension Color {
    /// Default avatar color when no specific color is provided
    static let defaultAvatar = Color(red: 0.58, green: 0.65, blue: 0.65)
}
