//
//  ImageViewerOverlay.swift
//  wee
//
//  Full-screen image viewer overlay with zoom support
//

import SwiftUI

struct ImageViewerOverlay: View {
    let dataUrl: String
    let onClose: () -> Void

    var body: some View {
        ZStack {
            Color.black.opacity(0.5)
                .ignoresSafeArea()
                .onTapGesture(perform: onClose)
            imageViewerSheet
                .frame(maxWidth: .infinity, maxHeight: .infinity)
        }
        .transition(.opacity)
    }

    @ViewBuilder
    private var imageViewerSheet: some View {
        let decodedImage = dataUrl.decodeBase64Image()

        VStack(spacing: 0) {
            // Header
            HStack {
                Button("Done", action: onClose)
                    .font(.body.bold())

                Spacer()

                Text("Image")
                    .font(.headline)

                Spacer()

                // Placeholder for alignment
                Button("Done", action: onClose)
                    .font(.body.bold())
                    .hidden()
            }
            .padding(16)
            .background(Color(.systemBackground))
            .border(Color(.systemGray3), width: 1)

            // Content
            if let uiImage = decodedImage {
                ZoomableImageView(image: uiImage)
                    .background(Color(.systemBackground))
            } else {
                ErrorImageView(dataUrl: dataUrl)
            }
        }
    }
}

struct ErrorImageView: View {
    let dataUrl: String

    var body: some View {
        VStack(spacing: 16) {
            Image(systemName: "exclamationmark.triangle.fill")
                .font(.system(size: 40))
                .foregroundStyle(.orange)

            Text("Could not decode image")
                .font(.headline)

            Text("Data URL length: \(dataUrl.count)")
                .font(.caption)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(Color(.systemBackground))
    }
}
