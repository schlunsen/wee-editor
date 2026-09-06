//
//  String+Image.swift
//  wee
//
//  Image decoding extensions for base64 data URLs
//

import UIKit

extension String {
    /// Decodes a base64 image string (data URL or plain base64) into a UIImage
    func decodeBase64Image() -> UIImage? {
        var base64String = self

        // If it's a data URL, extract the base64 part
        if contains(",") {
            let components = components(separatedBy: ",")
            if components.count >= 2 {
                base64String = components[1]
            } else {
                return nil
            }
        }

        // Trim whitespace and newlines
        base64String = base64String.trimmingCharacters(in: .whitespacesAndNewlines)

        // Try to decode base64
        guard let data = Data(base64Encoded: base64String, options: .ignoreUnknownCharacters) else {
            // If standard decoding fails, try with padding
            let padding = (4 - (base64String.count % 4)) % 4
            let paddedString = base64String + String(repeating: "=", count: padding)

            guard let paddedData = Data(base64Encoded: paddedString, options: .ignoreUnknownCharacters) else {
                return nil
            }

            return UIImage(data: paddedData)
        }

        // Create UIImage from data
        return UIImage(data: data)
    }
}
