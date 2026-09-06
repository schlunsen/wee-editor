//
//  WrappingHStack.swift
//  wee
//
//  Compact tool chip view with wrapping layout
//

import SwiftUI

struct WrappingHStack: View {
    let tools: [ToolUseInfo]

    var body: some View {
        FlowLayout(spacing: 4) {
            ForEach(tools, id: \.id) { tool in
                ToolChipView(tool: tool)
            }
        }
    }
}

struct ToolChipView: View {
    let tool: ToolUseInfo

    private var chipColor: Color {
        switch tool.name.lowercased() {
        case "bash": return Color(red: 0.3, green: 0.7, blue: 0.5)
        case "read", "glob", "grep": return Color(red: 0.4, green: 0.6, blue: 0.85)
        case "edit", "write": return Color(red: 0.85, green: 0.55, blue: 0.3)
        default: return Color(red: 0.6, green: 0.5, blue: 0.8)
        }
    }

    var body: some View {
        HStack(spacing: 5) {
            Text(tool.statusIcon)
                .font(.system(size: 9))

            Image(systemName: toolIcon)
                .font(.system(size: 8, weight: .semibold))
                .foregroundStyle(.white.opacity(0.8))

            Text(tool.displayName)
                .font(.system(size: 10, weight: .medium))
                .foregroundStyle(.white.opacity(0.95))

            // Show brief input for bash/read
            if tool.name.lowercased() == "bash", let command = tool.bashCommand {
                Text(command.count > 30 ? String(command.prefix(30)) + "…" : command)
                    .font(.system(size: 9, design: .monospaced))
                    .foregroundStyle(.white.opacity(0.6))
                    .lineLimit(1)
            } else if tool.name.lowercased() == "read", let filePath = tool.readFilePath {
                Text(filePath.count > 30 ? String(filePath.prefix(30)) + "…" : filePath)
                    .font(.system(size: 9, design: .monospaced))
                    .foregroundStyle(.white.opacity(0.6))
                    .lineLimit(1)
            }
        }
        .padding(.horizontal, 8)
        .padding(.vertical, 5)
        .background(chipColor.opacity(0.85))
        .clipShape(Capsule())
    }

    private var toolIcon: String {
        switch tool.name.lowercased() {
        case "bash": return "terminal"
        case "read": return "doc.text"
        case "edit", "write": return "pencil"
        case "glob": return "folder.badge.gearshape"
        case "grep": return "magnifyingglass"
        default: return "wrench"
        }
    }
}
