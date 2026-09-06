//
//  MessageBubbleView.swift
//  wee
//
//  Individual chat message bubble view
//

import SwiftUI

struct MessageBubbleView: View {
    let message: AgentMessage
    let isExpanded: Bool
    let avatarImage: UIImage?
    let avatarName: String
    let avatarColor: Color
    let onToggleExpand: () -> Void
    let onTap: () -> Void
    let onImageTap: (String) -> Void

    private var isUser: Bool { message.role == "user" }

    var body: some View {
        HStack(alignment: .bottom, spacing: 8) {
            if isUser { Spacer(minLength: 48) }

            if !isUser {
                // Assistant avatar - uses session avatar
                AvatarView(
                    image: avatarImage,
                    name: avatarName,
                    color: avatarColor,
                    size: 28
                )
            }

            VStack(alignment: isUser ? .trailing : .leading, spacing: 4) {
                // Show thinking indicator if present
                if let thinking = message.thinking, !thinking.isEmpty {
                    ThinkingIndicatorView(onTap: onTap)
                }

                // Main message content
                let textContent = message.content.textContent
                let imageBlocks = message.content.imageBlocks
                let tools = message.getTools()
                let hasContent = !textContent.isEmpty || !imageBlocks.isEmpty || !tools.isEmpty

                if hasContent {
                    MessageContentView(
                        textContent: textContent,
                        imageBlocks: imageBlocks,
                        tools: tools,
                        isUser: isUser,
                        isExpanded: isExpanded,
                        onToggleExpand: onToggleExpand,
                        onTap: onTap,
                        onImageTap: onImageTap
                    )
                }

                // Timestamp
                if let createdAt = message.created_at {
                    Text(createdAt.formattedTime())
                        .font(.caption2)
                        .foregroundStyle(.tertiary)
                        .padding(.horizontal, 4)
                }
            }
            if !isUser { Spacer(minLength: 48) }
        }
        .padding(.vertical, 2)
    }
}

struct ThinkingIndicatorView: View {
    let onTap: () -> Void
    @State private var animate = false

    var body: some View {
        HStack(spacing: 6) {
            HStack(spacing: 3) {
                ForEach(0..<3) { index in
                    Circle()
                        .fill(Color.purple.opacity(0.5))
                        .frame(width: 5, height: 5)
                        .scaleEffect(animate ? 1.0 : 0.5)
                        .animation(
                            .easeInOut(duration: 0.6)
                            .repeatForever()
                            .delay(Double(index) * 0.2),
                            value: animate
                        )
                }
            }
            Text("Thinking")
                .font(.caption2)
                .foregroundStyle(.secondary)
        }
        .padding(.horizontal, 10)
        .padding(.vertical, 6)
        .background(Color(.systemGray6).opacity(0.8))
        .clipShape(Capsule())
        .onTapGesture(perform: onTap)
        .onAppear { animate = true }
    }
}

struct MessageContentView: View {
    let textContent: String
    let imageBlocks: [ImageBlockData]
    let tools: [ToolUseInfo]
    let isUser: Bool
    let isExpanded: Bool
    let onToggleExpand: () -> Void
    let onTap: () -> Void
    let onImageTap: (String) -> Void

    var body: some View {
        VStack(alignment: isUser ? .trailing : .leading, spacing: 6) {
            // Text content
            if !textContent.isEmpty {
                TextContentView(
                    textContent: textContent,
                    isUser: isUser,
                    isExpanded: isExpanded,
                    onToggleExpand: onToggleExpand
                )
            }

            // Image content
            if !imageBlocks.isEmpty {
                ImageBlocksView(imageBlocks: imageBlocks, onImageTap: onImageTap)
            }

            // Tool usage chips
            if !tools.isEmpty {
                WrappingHStack(tools: tools)
            }
        }
        .padding(.vertical, 10)
        .padding(.horizontal, 14)
        .background(bubbleBackground)
        .clipShape(RoundedRectangle(cornerRadius: 18, style: .continuous))
        .overlay(
            RoundedRectangle(cornerRadius: 18, style: .continuous)
                .strokeBorder(
                    isUser ? Color.clear : Color(.systemGray4).opacity(0.5),
                    lineWidth: 0.5
                )
        )
        .shadow(color: .black.opacity(isUser ? 0.08 : 0.04), radius: isUser ? 4 : 2, y: 1)
        .onTapGesture(perform: onTap)
    }

    @ViewBuilder
    private var bubbleBackground: some View {
        if isUser {
            LinearGradient(
                colors: [Color(red: 0.35, green: 0.45, blue: 0.95), Color(red: 0.5, green: 0.35, blue: 0.9)],
                startPoint: .topLeading,
                endPoint: .bottomTrailing
            )
        } else {
            Color(.systemBackground)
        }
    }
}

struct TextContentView: View {
    let textContent: String
    let isUser: Bool
    let isExpanded: Bool
    let onToggleExpand: () -> Void

    private var truncated: String? {
        isExpanded ? nil : textContent.truncated(to: 3)
    }

    private var shouldShowExpand: Bool {
        textContent.lineCount() > 3
    }

    var body: some View {
        VStack(alignment: isUser ? .trailing : .leading, spacing: 4) {
            Text(isExpanded ? textContent : (truncated ?? textContent))
                .font(.subheadline)
                .fontWeight(isUser ? .regular : .regular)
                .foregroundStyle(isUser ? .white : .primary)
                .textSelection(.enabled)
                .multilineTextAlignment(isUser ? .trailing : .leading)
                .lineSpacing(2)

            // Show expand/collapse button if needed
            if shouldShowExpand {
                ExpandCollapseButton(isExpanded: isExpanded, isUser: isUser, action: onToggleExpand)
            }
        }
    }
}

struct ExpandCollapseButton: View {
    let isExpanded: Bool
    let isUser: Bool
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            HStack(spacing: 4) {
                Text(isExpanded ? "Show less" : "Show more")
                    .font(.caption2.weight(.medium))
                Image(systemName: isExpanded ? "chevron.up" : "chevron.down")
                    .font(.system(size: 8, weight: .semibold))
            }
            .foregroundStyle(isUser ? .white.opacity(0.75) : Color.indigo.opacity(0.8))
            .padding(.top, 2)
        }
    }
}

struct ImageBlocksView: View {
    let imageBlocks: [ImageBlockData]
    let onImageTap: (String) -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            ForEach(imageBlocks) { image in
                MessageImageView(dataUrl: image.dataUrl)
                    .onTapGesture { onImageTap(image.dataUrl) }
            }
        }
    }
}

struct MessageImageView: View {
    let dataUrl: String
    private let maxWidth: CGFloat = 280

    var body: some View {
        if let uiImage = dataUrl.decodeBase64Image() {
            Image(uiImage: uiImage)
                .resizable()
                .scaledToFit()
                .frame(maxWidth: maxWidth, maxHeight: 600)
                .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
                .overlay(
                    RoundedRectangle(cornerRadius: 12, style: .continuous)
                        .stroke(Color(.systemGray4).opacity(0.3), lineWidth: 0.5)
                )
                .shadow(color: .black.opacity(0.1), radius: 4, y: 2)
        } else {
            VStack(spacing: 8) {
                Image(systemName: "photo.slash")
                    .font(.system(size: 28))
                    .foregroundStyle(.quaternary)
                Text("Image unavailable")
                    .font(.caption.weight(.medium))
                    .foregroundStyle(.secondary)
            }
            .frame(maxWidth: maxWidth, minHeight: 80)
            .frame(maxWidth: .infinity)
            .background(Color(.systemGray6))
            .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
        }
    }
}
