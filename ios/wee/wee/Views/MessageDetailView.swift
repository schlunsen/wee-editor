//
//  MessageDetailView.swift
//  wee
//
//  Full message detail view for displaying complete message content
//

import SwiftUI

struct MessageDetailView: View {
    @Environment(\.dismiss) var dismiss
    let message: AgentMessage

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                // Header
                VStack(alignment: .leading, spacing: 8) {
                    HStack {
                        Label(message.role.capitalized, systemImage: roleIcon(message.role))
                            .font(.headline)
                            .foregroundStyle(roleColor(message.role))

                        Spacer()

                        if let createdAt = message.created_at {
                            Text(formatDate(createdAt))
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
                    }

                    Divider()
                }
                .padding(16)
                .background(Color(.systemGray6))

                // Content
                ScrollView {
                    VStack(alignment: .leading, spacing: 16) {
                        // Thinking section
                        if let thinking = message.thinking, !thinking.isEmpty {
                            VStack(alignment: .leading, spacing: 8) {
                                Label("Thinking", systemImage: "lightbulb.fill")
                                    .font(.caption.bold())
                                    .foregroundStyle(WeeColors.warning)

                                Text(thinking)
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                                    .textSelection(.enabled)
                                    .padding(12)
                                    .background(Color(.systemGray6))
                                    .cornerRadius(8)
                            }
                        }

                        // Text content section
                        let textContent = message.content.textContent
                        if !textContent.isEmpty {
                            VStack(alignment: .leading, spacing: 8) {
                                Label("Message", systemImage: "bubble.left.fill")
                                    .font(.caption.bold())
                                    .foregroundStyle(WeeColors.accent)

                                Text(textContent)
                                    .font(.body)
                                    .textSelection(.enabled)
                                    .padding(12)
                                    .background(Color(.systemGray6))
                                    .cornerRadius(8)
                            }
                        }

                        // Images section
                        let imageBlocks = message.content.imageBlocks
                        if !imageBlocks.isEmpty {
                            VStack(alignment: .leading, spacing: 8) {
                                Label("Images", systemImage: "photo.fill")
                                    .font(.caption.bold())
                                    .foregroundStyle(WeeColors.accent)

                                VStack(spacing: 12) {
                                    ForEach(imageBlocks) { image in
                                        if let uiImage = decodeBase64Image(image.dataUrl) {
                                            Image(uiImage: uiImage)
                                                .resizable()
                                                .scaledToFit()
                                                .frame(maxWidth: .infinity)
                                                .cornerRadius(8)
                                                .overlay(
                                                    RoundedRectangle(cornerRadius: 8)
                                                        .stroke(Color(.systemGray3), lineWidth: 1)
                                                )
                                        } else {
                                            VStack(spacing: 8) {
                                                Image(systemName: "photo.slash")
                                                    .font(.system(size: 32))
                                                    .foregroundStyle(.gray)
                                                Text("Image unavailable")
                                                    .font(.subheadline)
                                                    .foregroundStyle(.secondary)
                                            }
                                            .frame(maxWidth: .infinity)
                                            .padding(20)
                                            .background(Color(.systemGray6))
                                            .cornerRadius(8)
                                        }
                                    }
                                }
                            }
                        }

                        // Tools section
                        let tools = message.getTools()
                        if !tools.isEmpty {
                            VStack(alignment: .leading, spacing: 12) {
                                Label("Tools Used", systemImage: "wrench.and.screwdriver.fill")
                                    .font(.caption.bold())
                                    .foregroundStyle(WeeColors.success)

                                VStack(alignment: .leading, spacing: 12) {
                                    ForEach(tools, id: \.id) { tool in
                                        // Special formatting for Bash commands
                                        if tool.name.lowercased() == "bash", let command = tool.bashCommand {
                                            bashCommandCard(tool: tool, command: command)
                                        } else if tool.name.lowercased() == "read", let filePath = tool.readFilePath {
                                            readFileCard(tool: tool, filePath: filePath, limit: tool.readLimit)
                                        } else {
                                            // Default tool card
                                            VStack(alignment: .leading, spacing: 8) {
                                                HStack(spacing: 8) {
                                                    Text(tool.statusIcon)
                                                        .font(.title3)

                                                    VStack(alignment: .leading, spacing: 2) {
                                                        Text(tool.displayName)
                                                            .font(.headline)
                                                            .foregroundStyle(.primary)

                                                        if !tool.inputDescription.isEmpty {
                                                            Text("Input: \(tool.inputDescription)")
                                                                .font(.caption)
                                                                .foregroundStyle(.secondary)
                                                                .textSelection(.enabled)
                                                        }
                                                    }

                                                    Spacer()
                                                }
                                            }
                                            .padding(12)
                                            .background(Color(.systemGray6))
                                            .cornerRadius(8)
                                        }
                                    }
                                }
                            }
                        }

                        // Empty state
                        if message.content.textContent.isEmpty &&
                           message.content.imageBlocks.isEmpty &&
                           message.getTools().isEmpty &&
                           (message.thinking == nil || message.thinking!.isEmpty) {
                            VStack(spacing: 12) {
                                Image(systemName: "bubble.slash.fill")
                                    .font(.system(size: 36))
                                    .foregroundStyle(.gray)
                                Text("No Content")
                                    .font(.headline)
                                Text("This message has no visible content")
                                    .font(.subheadline)
                                    .foregroundStyle(.secondary)
                            }
                            .frame(maxWidth: .infinity)
                            .padding(32)
                        }
                    }
                    .padding(16)
                }
                .background(Color(.systemBackground))
            }
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Done") {
                        dismiss()
                    }
                    .font(.body.bold())
                }
            }
        }
    }

    // MARK: - Helper Methods

    @ViewBuilder
    private func bashCommandCard(tool: ToolUseInfo, command: String) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            // Header with icon and status
            HStack(spacing: 8) {
                Text(tool.statusIcon)
                    .font(.title3)

                VStack(alignment: .leading, spacing: 2) {
                    Text(tool.displayName)
                        .font(.headline)
                        .foregroundStyle(.primary)

                    if let description = tool.bashDescription, !description.isEmpty {
                        Text(description)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                }

                Spacer()
            }

            // Command display with monospace font
            VStack(alignment: .leading, spacing: 6) {
                Label("Command", systemImage: "terminal.fill")
                    .font(.caption.bold())
                    .foregroundStyle(WeeColors.accent)

                Text(command)
                    .font(.caption.monospaced())
                    .foregroundStyle(.primary)
                    .textSelection(.enabled)
                    .padding(10)
                    .background(Color(.black).opacity(0.05))
                    .cornerRadius(6)
                    .frame(maxWidth: .infinity, alignment: .leading)
            }
        }
        .padding(12)
        .background(Color(.systemGray6))
        .cornerRadius(8)
    }

    @ViewBuilder
    private func readFileCard(tool: ToolUseInfo, filePath: String, limit: Int?) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            // Header with icon and status
            HStack(spacing: 8) {
                Text(tool.statusIcon)
                    .font(.title3)

                VStack(alignment: .leading, spacing: 2) {
                    Text(tool.displayName)
                        .font(.headline)
                        .foregroundStyle(.primary)

                    Text("Read file contents")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }

                Spacer()
            }

            // File path display
            VStack(alignment: .leading, spacing: 6) {
                Label("File Path", systemImage: "doc.text.fill")
                    .font(.caption.bold())
                    .foregroundStyle(WeeColors.accent)

                Text(filePath)
                    .font(.caption.monospaced())
                    .foregroundStyle(.primary)
                    .textSelection(.enabled)
                    .padding(10)
                    .background(Color(.black).opacity(0.05))
                    .cornerRadius(6)
                    .frame(maxWidth: .infinity, alignment: .leading)
            }

            // Show limit if provided
            if let limit = limit, limit > 0 {
                VStack(alignment: .leading, spacing: 6) {
                    Label("Limit", systemImage: "line.3")
                        .font(.caption.bold())
                        .foregroundStyle(.secondary)

                    Text("\(limit) lines")
                        .font(.caption)
                        .foregroundStyle(.primary)
                }
            }
        }
        .padding(12)
        .background(Color(.systemGray6))
        .cornerRadius(8)
    }

    private func decodeBase64Image(_ dataUrl: String) -> UIImage? {
        var base64String = dataUrl

        if dataUrl.contains(",") {
            let components = dataUrl.components(separatedBy: ",")
            if components.count >= 2 {
                base64String = components[1]
            } else {
                return nil
            }
        }

        base64String = base64String.trimmingCharacters(in: .whitespacesAndNewlines)

        guard let data = Data(base64Encoded: base64String, options: .ignoreUnknownCharacters) else {
            let padding = (4 - (base64String.count % 4)) % 4
            let paddedString = base64String + String(repeating: "=", count: padding)

            guard let paddedData = Data(base64Encoded: paddedString, options: .ignoreUnknownCharacters) else {
                return nil
            }

            return UIImage(data: paddedData)
        }

        return UIImage(data: data)
    }

    private func roleIcon(_ role: String) -> String {
        switch role.lowercased() {
        case "user":
            return "person.fill"
        case "assistant":
            return "sparkles"
        case "system":
            return "gear"
        default:
            return "message.fill"
        }
    }

    private func roleColor(_ role: String) -> Color {
        switch role.lowercased() {
        case "user":
            return WeeColors.accent
        case "assistant":
            return WeeColors.success
        case "system":
            return .gray
        default:
            return .gray
        }
    }

    private func formatDate(_ dateString: String) -> String {
        guard let date = parseISO8601Date(dateString) else {
            return "Unknown"
        }

        let formatter = DateFormatter()
        formatter.timeStyle = .short
        formatter.dateStyle = .short
        return formatter.string(from: date)
    }

    private func parseISO8601Date(_ dateString: String) -> Date? {
        let formatters = [
            ISO8601DateFormatter(),
            {
                let formatter = ISO8601DateFormatter()
                formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
                return formatter
            }(),
            {
                let formatter = ISO8601DateFormatter()
                formatter.formatOptions = [.withInternetDateTime]
                return formatter
            }()
        ]

        for formatter in formatters {
            if let date = formatter.date(from: dateString) {
                return date
            }
        }

        let dateFormatter = DateFormatter()
        dateFormatter.dateFormat = "yyyy-MM-dd'T'HH:mm:ss.SSSSSSZ"
        if let date = dateFormatter.date(from: dateString) {
            return date
        }

        return nil
    }
}

#Preview {
    let exampleMessage = AgentMessage(
        id: "msg-123",
        role: "assistant",
        content: .string("This is an example message with full content."),
        created_at: ISO8601DateFormatter().string(from: Date()),
        thinking: "Let me analyze this request and provide a comprehensive response.",
        tools: nil
    )

    MessageDetailView(message: exampleMessage)
}
