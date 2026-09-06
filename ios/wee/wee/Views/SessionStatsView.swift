//
//  SessionStatsView.swift
//  wee
//
//  View for displaying session context usage and statistics
//

import SwiftUI

struct SessionStatsView: View {
    let sessionID: String

    @State private var session: AgentSession?
    @State private var contextInfo: SessionContextInfo?
    @State private var messages: [AgentMessage] = []
    @State private var isLoading = true
    @State private var errorMessage: String?

    private let apiClient = WeeAPIClient.shared

    var body: some View {
        ScrollView {
            LazyVStack(spacing: 12) {
                if isLoading {
                    loadingView
                } else if let error = errorMessage {
                    errorView(error)
                } else {
                    if let session = session {
                        sessionHeaderCard(session)
                    }
                    contextCard
                    if let session = session {
                        usageMetricsCard(session)
                    }
                    messagesBreakdownCard
                }
            }
            .padding(.top, 8)
            .padding(.bottom, 16)
        }
        .background(Color(.systemGroupedBackground))
        .navigationTitle("Session Stats")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button(action: { Task { await loadData() } }) {
                    Image(systemName: "arrow.clockwise")
                        .font(.system(size: 14, weight: .semibold))
                }
            }
        }
        .task {
            await loadData()
        }
    }

    // MARK: - Loading & Error

    private var loadingView: some View {
        VStack(spacing: 12) {
            ProgressView()
                .tint(WeeColors.accent)
            Text("Loading session stats...")
                .font(.caption)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .padding(.top, 60)
    }

    private func errorView(_ error: String) -> some View {
        VStack(spacing: 16) {
            Spacer().frame(height: 40)
            Image(systemName: "exclamationmark.triangle.fill")
                .font(.system(size: 36))
                .foregroundStyle(WeeColors.warning)
            Text("Unable to Load Stats")
                .font(.headline)
            Text(error)
                .font(.subheadline)
                .foregroundStyle(.secondary)
                .multilineTextAlignment(.center)
                .padding(.horizontal, 32)

            Button("Retry") {
                Task { await loadData() }
            }
            .buttonStyle(.bordered)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    // MARK: - Session Header Card

    private func sessionHeaderCard(_ session: AgentSession) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack(spacing: 8) {
                Image(systemName: "bubble.left.and.bubble.right.fill")
                    .font(.system(size: 14))
                    .foregroundStyle(WeeColors.accent)
                Text("Session")
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(.primary)
            }

            Divider()

            // Session ID
            HStack(spacing: 8) {
                Text("ID")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .frame(width: 80, alignment: .leading)
                Text(String(session.id.prefix(16)) + "...")
                    .font(.system(.caption, design: .monospaced))
                    .foregroundStyle(.primary)
                    .textSelection(.enabled)
            }

            // Status
            HStack(spacing: 8) {
                Text("Status")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .frame(width: 80, alignment: .leading)
                HStack(spacing: 4) {
                    Circle()
                        .fill(statusColor(session.status))
                        .frame(width: 6, height: 6)
                    Text(session.status.capitalized)
                        .font(.caption.weight(.medium))
                        .foregroundStyle(statusColor(session.status))
                }
            }

            // Model
            if let model = session.model_name {
                HStack(spacing: 8) {
                    Text("Model")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                        .frame(width: 80, alignment: .leading)
                    Text(model)
                        .font(.caption)
                        .foregroundStyle(.primary)
                }
            }

            // Provider
            if let provider = session.provider {
                HStack(spacing: 8) {
                    Text("Provider")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                        .frame(width: 80, alignment: .leading)
                    Text(provider)
                        .font(.caption)
                        .foregroundStyle(.primary)
                }
            }

            // Git branch
            if let branch = session.git_branch {
                HStack(spacing: 8) {
                    Text("Branch")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                        .frame(width: 80, alignment: .leading)
                    HStack(spacing: 4) {
                        Image(systemName: "branch")
                            .font(.caption2)
                        Text(branch)
                            .font(.system(.caption, design: .monospaced))
                    }
                    .foregroundStyle(.primary)
                }
            }

            // Working directory
            if let wd = contextInfo?.working_directory ?? session.working_directory {
                HStack(spacing: 8) {
                    Text("Directory")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                        .frame(width: 80, alignment: .leading)
                    Text(wd)
                        .font(.system(.caption2, design: .monospaced))
                        .foregroundStyle(.primary)
                        .lineLimit(2)
                }
            }

            // Created
            HStack(spacing: 8) {
                Text("Created")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .frame(width: 80, alignment: .leading)
                Text(formatDate(session.created_at))
                    .font(.caption)
                    .foregroundStyle(.primary)
            }
        }
        .padding(16)
        .background(Color(.secondarySystemGroupedBackground))
        .clipShape(RoundedRectangle(cornerRadius: 12))
        .padding(.horizontal, 16)
    }

    // MARK: - Context Card

    private var contextCard: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack(spacing: 8) {
                Image(systemName: "text.alignleft")
                    .font(.system(size: 14))
                    .foregroundStyle(WeeColors.accent)
                Text("Context Usage")
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(.primary)
            }

            Divider()

            // Message count as proxy for context
            let msgCount = messages.count
            let userMessages = messages.filter { $0.role == "user" }.count
            let assistantMessages = messages.filter { $0.role == "assistant" }.count
            let systemMessages = messages.filter { $0.role == "system" }.count

            // Context usage bar (rough estimate based on message count)
            let estimatedContextPercent = min(100, Double(msgCount) * 0.8)

            VStack(spacing: 8) {
                HStack {
                    Text("Estimated Context Window")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    Spacer()
                    Text(String(format: "%.0f%%", estimatedContextPercent))
                        .font(.caption.weight(.bold))
                        .foregroundStyle(contextColor(estimatedContextPercent))
                }

                GeometryReader { geometry in
                    ZStack(alignment: .leading) {
                        RoundedRectangle(cornerRadius: 4)
                            .fill(Color(.systemGray5))
                            .frame(height: 8)

                        RoundedRectangle(cornerRadius: 4)
                            .fill(contextColor(estimatedContextPercent))
                            .frame(width: geometry.size.width * min(1.0, estimatedContextPercent / 100.0), height: 8)
                    }
                }
                .frame(height: 8)
            }

            HStack(spacing: 16) {
                contextChip(value: "\(msgCount)", label: "Messages", icon: "message.fill", color: WeeColors.accent)
                contextChip(value: "\(userMessages)", label: "User", icon: "person.fill", color: WeeColors.success)
                contextChip(value: "\(assistantMessages)", label: "Assistant", icon: "cpu", color: WeeColors.warning)
                if systemMessages > 0 {
                    contextChip(value: "\(systemMessages)", label: "System", icon: "gearshape.fill", color: .gray)
                }
            }
        }
        .padding(16)
        .background(Color(.secondarySystemGroupedBackground))
        .clipShape(RoundedRectangle(cornerRadius: 12))
        .padding(.horizontal, 16)
    }

    private func contextChip(value: String, label: String, icon: String, color: Color) -> some View {
        VStack(spacing: 4) {
            Image(systemName: icon)
                .font(.system(size: 12))
                .foregroundStyle(color)
            Text(value)
                .font(.caption.weight(.bold))
                .foregroundStyle(.primary)
            Text(label)
                .font(.caption2)
                .foregroundStyle(.tertiary)
        }
        .frame(maxWidth: .infinity)
    }

    // MARK: - Usage Metrics Card

    private func usageMetricsCard(_ session: AgentSession) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack(spacing: 8) {
                Image(systemName: "chart.bar.fill")
                    .font(.system(size: 14))
                    .foregroundStyle(WeeColors.success)
                Text("Usage Metrics")
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(.primary)
            }

            Divider()

            LazyVGrid(columns: [
                GridItem(.flexible()),
                GridItem(.flexible())
            ], spacing: 12) {
                metricTile(
                    icon: "dollarsign.circle",
                    value: session.cost != nil ? String(format: "$%.4f", session.cost!) : "—",
                    label: "Cost",
                    color: WeeColors.accent
                )
                metricTile(
                    icon: "message.fill",
                    value: "\(session.message_count ?? messages.count)",
                    label: "Messages",
                    color: WeeColors.success
                )
                metricTile(
                    icon: "clock.fill",
                    value: formatDuration(from: session.created_at, to: session.updated_at),
                    label: "Duration",
                    color: WeeColors.warning
                )
                metricTile(
                    icon: "arrow.triangle.2.circlepath",
                    value: "\(messages.filter { $0.role == "user" }.count)",
                    label: "Turns",
                    color: WeeColors.accent
                )
            }
        }
        .padding(16)
        .background(Color(.secondarySystemGroupedBackground))
        .clipShape(RoundedRectangle(cornerRadius: 12))
        .padding(.horizontal, 16)
    }

    private func metricTile(icon: String, value: String, label: String, color: Color) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            Image(systemName: icon)
                .font(.system(size: 14))
                .foregroundStyle(color)

            Text(value)
                .font(.subheadline.weight(.bold))
                .foregroundStyle(.primary)
                .lineLimit(1)
                .minimumScaleFactor(0.7)

            Text(label)
                .font(.caption2)
                .foregroundStyle(.tertiary)
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Color(.systemGray6).opacity(0.5))
        .clipShape(RoundedRectangle(cornerRadius: 8))
    }

    // MARK: - Messages Breakdown Card

    private var messagesBreakdownCard: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack(spacing: 8) {
                Image(systemName: "list.bullet.clipboard")
                    .font(.system(size: 14))
                    .foregroundStyle(WeeColors.warning)
                Text("Messages Breakdown")
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(.primary)
            }

            Divider()

            let userMsgs = messages.filter { $0.role == "user" }
            let assistantMsgs = messages.filter { $0.role == "assistant" }
            let systemMsgs = messages.filter { $0.role == "system" }

            breakdownRow(
                label: "User",
                count: userMsgs.count,
                total: messages.count,
                color: WeeColors.accent
            )
            breakdownRow(
                label: "Assistant",
                count: assistantMsgs.count,
                total: messages.count,
                color: WeeColors.success
            )
            if !systemMsgs.isEmpty {
                breakdownRow(
                    label: "System",
                    count: systemMsgs.count,
                    total: messages.count,
                    color: .gray
                )
            }

            // Tools used count
            let toolsCount = assistantMsgs.reduce(0) { count, msg in
                count + msg.getTools().count
            }
            if toolsCount > 0 {
                Divider()
                HStack(spacing: 6) {
                    Image(systemName: "wrench.and.screwdriver")
                        .font(.caption)
                        .foregroundStyle(WeeColors.accent)
                    Text("Tool invocations:")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    Text("\(toolsCount)")
                        .font(.caption.weight(.bold))
                        .foregroundStyle(.primary)
                }
            }
        }
        .padding(16)
        .background(Color(.secondarySystemGroupedBackground))
        .clipShape(RoundedRectangle(cornerRadius: 12))
        .padding(.horizontal, 16)
    }

    private func breakdownRow(label: String, count: Int, total: Int, color: Color) -> some View {
        let pct = total > 0 ? Double(count) / Double(total) : 0

        return VStack(spacing: 6) {
            HStack {
                Circle()
                    .fill(color)
                    .frame(width: 8, height: 8)
                Text(label)
                    .font(.caption.weight(.medium))
                    .foregroundStyle(.primary)
                Spacer()
                Text("\(count) (\(Int(pct * 100))%)")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }

            GeometryReader { geometry in
                ZStack(alignment: .leading) {
                    RoundedRectangle(cornerRadius: 2)
                        .fill(Color(.systemGray5))
                        .frame(height: 4)

                    RoundedRectangle(cornerRadius: 2)
                        .fill(color.opacity(0.7))
                        .frame(width: geometry.size.width * pct, height: 4)
                }
            }
            .frame(height: 4)
        }
    }

    // MARK: - Helpers

    private func statusColor(_ status: String) -> Color {
        switch status.lowercased() {
        case "active", "running", "processing":
            return WeeColors.success
        case "completed":
            return .gray
        case "failed", "error":
            return WeeColors.error
        case "idle":
            return .gray
        default:
            return WeeColors.warning
        }
    }

    private func contextColor(_ percent: Double) -> Color {
        if percent < 50 {
            return WeeColors.success
        } else if percent < 80 {
            return WeeColors.warning
        } else {
            return WeeColors.error
        }
    }

    private func formatDate(_ dateString: String) -> String {
        guard let date = parseISO8601Date(dateString) else {
            return "Unknown"
        }
        let formatter = DateFormatter()
        formatter.dateStyle = .medium
        formatter.timeStyle = .short
        return formatter.string(from: date)
    }

    private func formatDuration(from startStr: String, to endStr: String) -> String {
        guard let start = parseISO8601Date(startStr),
              let end = parseISO8601Date(endStr) else {
            return "—"
        }
        let interval = end.timeIntervalSince(start)
        let hours = Int(interval) / 3600
        let minutes = Int(interval) % 3600 / 60
        let seconds = Int(interval) % 60

        if hours > 0 {
            return "\(hours)h \(minutes)m"
        } else if minutes > 0 {
            return "\(minutes)m \(seconds)s"
        } else {
            return "\(seconds)s"
        }
    }

    private func parseISO8601Date(_ dateString: String) -> Date? {
        let formatters = [
            ISO8601DateFormatter(),
            {
                let f = ISO8601DateFormatter()
                f.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
                return f
            }(),
            {
                let f = ISO8601DateFormatter()
                f.formatOptions = [.withInternetDateTime]
                return f
            }()
        ]
        for formatter in formatters {
            if let date = formatter.date(from: dateString) {
                return date
            }
        }
        return nil
    }

    // MARK: - Data Loading

    private func loadData() async {
        isLoading = true
        errorMessage = nil

        do {
            async let sessionsResult: [AgentSession] = apiClient.getAgentSessions()
            async let messagesResult: [AgentMessage] = apiClient.getSessionMessages(sessionID: sessionID, limit: 500)

            // Context info from the context endpoint
            let contextResult: SessionContextInfo? = await (try? await fetchContextInfo())

            let sessions = try await sessionsResult
            let loadedMessages = try await messagesResult

            await MainActor.run {
                self.session = sessions.first(where: { $0.id == sessionID })
                self.messages = loadedMessages
                self.contextInfo = contextResult
                self.isLoading = false
            }
        } catch {
            await MainActor.run {
                self.errorMessage = error.localizedDescription
                self.isLoading = false
            }
        }
    }

    private func fetchContextInfo() async -> SessionContextInfo? {
        return try? await apiClient.getSessionContext(sessionID: sessionID)
    }
}

// Session context info model
struct SessionContextInfo: Codable {
    let session_id: String?
    let working_directory: String?
    let git_branch: String?
    let timestamp: String?
    let fallback_used: Bool?
}

#Preview {
    NavigationStack {
        SessionStatsView(sessionID: "example-session-id")
    }
}
