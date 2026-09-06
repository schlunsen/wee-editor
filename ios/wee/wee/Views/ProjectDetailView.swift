//
//  ProjectDetailView.swift
//  wee
//
//  Project detail and execution steps view
//

import SwiftUI

// MARK: - Session Card Protocol

/// Protocol for unified session card rendering
protocol SessionCardData: Identifiable {
    var id: String { get }
    var status: String { get }
    var created_at: String { get }
    var cost_value: Double { get }
    var model_name: String? { get }
    var provider: String? { get }
    var selected_avatar_id: Int64? { get }
    var selected_avatar: Avatar? { get }
    var displayMessageCount: Int { get }
}

// Conform AgentSession to the protocol
extension AgentSession: SessionCardData {
    var cost_value: Double {
        cost ?? 0.0
    }

    var displayMessageCount: Int {
        message_count ?? 0
    }
}

// Conform SessionMetadata to the protocol
extension SessionMetadata: SessionCardData {
    var cost_value: Double {
        cost_usd
    }

    var displayMessageCount: Int {
        message_count
    }
}

struct ProjectDetailView: View {
    let projectID: String

    @State private var isLoading = false
    @State private var errorMessage: String?
    @State private var project: Project?
    @State private var projectSessions: [SessionMetadata] = []
    @State private var sessionsLoading = false
    @State private var showCreateSessionModal = false
    @State private var selectedSessionID: String? = nil
    @State private var selectedSessionTitle: String? = nil
    @State private var navigationActive = false
    @State private var isSelectionMode = false
    @State private var selectedSessionIDs = Set<String>()
    @ObservedObject var webSocketViewModel = SessionsWebSocketViewModel.shared
    @ObservedObject var webSocketService = WebSocketService.shared
    @ObservedObject var avatarManager = AvatarManager.shared

    private let apiClient = WeeAPIClient.shared

    // MARK: - Computed Stats

    private var totalCost: Double {
        projectSessions.reduce(0) { $0 + $1.cost_usd }
    }

    private var totalMessages: Int {
        projectSessions.reduce(0) { $0 + $1.message_count }
    }

    private var totalDurationMs: Int64 {
        projectSessions.reduce(Int64(0)) { $0 + $1.duration_ms }
    }

    private var activeSessionCount: Int {
        projectSessions.filter { $0.status.lowercased() == "active" || $0.status.lowercased() == "running" || $0.status.lowercased() == "processing" }.count
    }

    private var completedSessionCount: Int {
        projectSessions.filter { $0.status.lowercased() == "completed" }.count
    }

    private var failedSessionCount: Int {
        projectSessions.filter { $0.status.lowercased() == "failed" || $0.status.lowercased() == "error" }.count
    }

    private var avgCostPerSession: Double {
        projectSessions.isEmpty ? 0 : totalCost / Double(projectSessions.count)
    }

    private var modelBreakdown: [(name: String, count: Int)] {
        var counts: [String: Int] = [:]
        for session in projectSessions {
            let model = session.model_name ?? "Unknown"
            counts[model, default: 0] += 1
        }
        return counts.map { (name: $0.key, count: $0.value) }
            .sorted { $0.count > $1.count }
    }

    private var providerBreakdown: [(name: String, count: Int)] {
        var counts: [String: Int] = [:]
        for session in projectSessions {
            let provider = session.provider ?? "Unknown"
            counts[provider, default: 0] += 1
        }
        return counts.map { (name: $0.key, count: $0.value) }
            .sorted { $0.count > $1.count }
    }

    var body: some View {
        VStack(spacing: 0) {
            if isLoading {
                VStack {
                    ProgressView()
                        .tint(WeeColors.accent)
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else if let error = errorMessage {
                    VStack(spacing: 12) {
                        Image(systemName: "exclamationmark.triangle.fill")
                            .font(.system(size: 36))
                            .foregroundStyle(WeeColors.warning)
                        Text("Unable to Load Project")
                            .font(.headline)
                        Text(error)
                            .font(.subheadline)
                            .foregroundStyle(.secondary)
                            .multilineTextAlignment(.center)
                    }
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else if let project = project {
                    ScrollView {
                        LazyVStack(spacing: 12) {
                            // MARK: - Project Header Card
                            projectHeaderCard(project)

                            // MARK: - Session Stats
                            if !projectSessions.isEmpty {
                                sessionStatsBar
                            }

                            // MARK: - Sessions Section Header
                            if !projectSessions.isEmpty || sessionsLoading {
                                HStack {
                                    Text("Sessions")
                                        .font(.subheadline.weight(.semibold))
                                        .foregroundStyle(.secondary)
                                    Spacer()
                                    Text("\(projectSessions.count) total")
                                        .font(.caption)
                                        .foregroundStyle(.tertiary)
                                }
                                .padding(.horizontal, 16)
                            }

                            // MARK: - Sessions List
                            if sessionsLoading {
                                HStack {
                                    ProgressView()
                                        .tint(WeeColors.accent)
                                    Text("Loading sessions...")
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                }
                                .frame(maxWidth: .infinity)
                                .padding(16)
                            } else if !projectSessions.isEmpty {
                                ForEach(projectSessions, id: \.id) { session in
                                    HStack(spacing: 0) {
                                        // Selection indicator
                                        if isSelectionMode {
                                            Image(systemName: selectedSessionIDs.contains(session.id) ? "checkmark.circle.fill" : "circle")
                                                .foregroundStyle(selectedSessionIDs.contains(session.id) ? WeeColors.accent : .gray)
                                                .font(.system(size: 22))
                                                .padding(.trailing, 10)
                                        }

                                        redesignedSessionCard(session)
                                    }
                                    .padding(.horizontal, 16)
                                    .contentShape(Rectangle())
                                    .onTapGesture {
                                        if isSelectionMode {
                                            if selectedSessionIDs.contains(session.id) {
                                                selectedSessionIDs.remove(session.id)
                                            } else {
                                                selectedSessionIDs.insert(session.id)
                                            }
                                        } else {
                                            selectedSessionID = session.id
                                            selectedSessionTitle = session.model_name ?? "Unknown"
                                            navigationActive = true
                                        }
                                    }
                                    .simultaneousGesture(
                                        LongPressGesture(minimumDuration: 0.5)
                                            .onEnded { _ in
                                                if !isSelectionMode {
                                                    isSelectionMode = true
                                                    selectedSessionIDs.insert(session.id)
                                                    let generator = UIImpactFeedbackGenerator(style: .medium)
                                                    generator.impactOccurred()
                                                }
                                            }
                                    )
                                    .contextMenu {
                                        if !isSelectionMode {
                                            Button(role: .destructive, action: {
                                                deleteSession(session.id)
                                            }) {
                                                Label("Delete", systemImage: "trash.fill")
                                            }
                                        }
                                    }
                                }
                            } else {
                                // Empty state
                                VStack(spacing: 12) {
                                    Spacer().frame(height: 32)
                                    Image(systemName: "text.bubble")
                                        .font(.system(size: 32, weight: .light))
                                        .foregroundStyle(.quaternary)
                                    Text("No Sessions Yet")
                                        .font(.subheadline.weight(.medium))
                                        .foregroundStyle(.secondary)
                                    Text("Tap + to start a new agent session")
                                        .font(.caption)
                                        .foregroundStyle(.tertiary)
                                }
                                .frame(maxWidth: .infinity)
                                .padding(24)
                                .transition(.opacity)
                            }

                            // MARK: - Bottom Metrics Panel
                            if !projectSessions.isEmpty {
                                bottomMetricsPanel
                            }

                            // Bottom spacer for scroll comfort
                            Spacer().frame(height: 16)
                        }
                        .padding(.top, 8)
                    }
                } else {
                    VStack(spacing: 12) {
                        Image(systemName: "folder.fill.badge.questionmark")
                            .font(.system(size: 36))
                            .foregroundStyle(.gray)
                        Text("Project Not Found")
                            .font(.headline)
                    }
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                }
            }
            .background(Color(.systemGroupedBackground))
            .navigationTitle("Project Details")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                if isSelectionMode {
                    ToolbarItem(placement: .topBarLeading) {
                        Button("Cancel") {
                            isSelectionMode = false
                            selectedSessionIDs.removeAll()
                        }
                    }

                    ToolbarItem(placement: .topBarTrailing) {
                        HStack(spacing: 12) {
                            Button(action: {
                                if selectedSessionIDs.count == projectSessions.count {
                                    selectedSessionIDs.removeAll()
                                } else {
                                    selectedSessionIDs = Set(projectSessions.map { $0.id })
                                }
                            }) {
                                Text(selectedSessionIDs.count == projectSessions.count ? "Deselect All" : "Select All")
                            }

                            Button(role: .destructive, action: {
                                deleteSelectedSessions()
                            }) {
                                Label("\(selectedSessionIDs.count)", systemImage: "trash.fill")
                            }
                            .disabled(selectedSessionIDs.isEmpty)
                        }
                    }
                } else {
                    ToolbarItem(placement: .topBarLeading) {
                        Button("Select") {
                            isSelectionMode = true
                        }
                    }

                    ToolbarItem(placement: .topBarTrailing) {
                        HStack(spacing: 12) {
                            Button(action: { showCreateSessionModal = true }) {
                                Image(systemName: "plus.circle.fill")
                                    .font(.system(size: 14, weight: .semibold))
                            }

                            Button(action: {
                                Task {
                                    await loadProject()
                                }
                            }) {
                                Image(systemName: "arrow.clockwise")
                                    .font(.system(size: 14, weight: .semibold))
                            }
                        }
                    }
                }
            }
            .sheet(isPresented: $showCreateSessionModal) {
                CreateSessionView(projectId: projectID, workingDirectory: project?.path)
            }
            .navigationDestination(isPresented: $navigationActive) {
                if let sessionID = selectedSessionID,
                   let sessionTitle = selectedSessionTitle {
                    ChatView(sessionID: sessionID, sessionTitle: sessionTitle)
                }
            }
        .task {
            await loadProject()
            if avatarManager.avatarMap.isEmpty {
                try? await Task.sleep(nanoseconds: 500_000_000)
            }
            await loadProjectSessions()
        }
        .onChange(of: avatarManager.avatarMap) {
            Task {
                await loadProjectSessions()
            }
        }
        .onReceive(NotificationCenter.default.publisher(for: NSNotification.Name("SessionCreated"))) { notification in
            if let userInfo = notification.userInfo,
               let projectId = userInfo["projectId"] as? String,
               projectId == projectID {
                Task {
                    try? await Task.sleep(nanoseconds: 500_000_000)
                    await loadProjectSessions()
                }
            }
        }
        .onReceive(NotificationCenter.default.publisher(for: NSNotification.Name("NavigateToSession"))) { notification in
            if let userInfo = notification.userInfo,
               let sessionId = userInfo["sessionId"] as? String,
               let sessionTitle = userInfo["sessionTitle"] as? String {
                selectedSessionID = sessionId
                selectedSessionTitle = sessionTitle
                navigationActive = true
            }
        }
        // Deselect session when user navigates back from ChatView to this project view.
        // This replaces ChatView's .onDisappear which incorrectly fired when pushing to
        // sub-views (ZenView, GitStatus, etc.) within the same navigation stack.
        .onChange(of: navigationActive) { _, isActive in
            if !isActive {
                SessionsWebSocketViewModel.shared.selectedSessionId = nil
            }
        }
    }

    // MARK: - Project Header Card

    private func projectHeaderCard(_ project: Project) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack(alignment: .center) {
                VStack(alignment: .leading, spacing: 4) {
                    Text(project.name)
                        .font(.title3.weight(.bold))
                        .foregroundStyle(.primary)

                    if let description = project.description, !description.isEmpty {
                        Text(description)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                            .lineLimit(2)
                    }
                }

                Spacer()

                // Status pill
                HStack(spacing: 5) {
                    Circle()
                        .fill(project.is_active ? WeeColors.success : WeeColors.warning.opacity(0.6))
                        .frame(width: 7, height: 7)

                    Text(project.is_active ? "Active" : "Inactive")
                        .font(.caption.weight(.medium))
                        .foregroundStyle(project.is_active ? WeeColors.success : WeeColors.warning)
                }
                .padding(.horizontal, 10)
                .padding(.vertical, 5)
                .background((project.is_active ? WeeColors.success : WeeColors.warning).opacity(0.1))
                .clipShape(Capsule())
            }

            // Project path in a subtle box
            HStack(spacing: 6) {
                Image(systemName: "folder")
                    .font(.caption)
                    .foregroundStyle(.tertiary)

                Text(project.path)
                    .font(.system(.caption, design: .monospaced))
                    .foregroundStyle(.secondary)
                    .lineLimit(2)
            }
        }
        .padding(16)
        .background(Color(.secondarySystemGroupedBackground))
        .clipShape(RoundedRectangle(cornerRadius: 12))
        .padding(.horizontal, 16)
    }

    // MARK: - Session Stats Bar

    private var sessionStatsBar: some View {
        HStack(spacing: 0) {
            statItem(
                value: "\(activeSessionCount)",
                label: "Active",
                icon: "play.circle.fill",
                color: WeeColors.success
            )

            Divider()
                .frame(height: 28)
                .background(Color(.systemGray4))

            statItem(
                value: "\(completedSessionCount)",
                label: "Completed",
                icon: "checkmark.circle.fill",
                color: .gray
            )

            Divider()
                .frame(height: 28)
                .background(Color(.systemGray4))

            statItem(
                value: String(format: "$%.2f", totalCost),
                label: "Total Cost",
                icon: "dollarsign.circle.fill",
                color: WeeColors.accent
            )
        }
        .padding(.vertical, 10)
        .background(Color(.secondarySystemGroupedBackground))
        .clipShape(RoundedRectangle(cornerRadius: 12))
        .padding(.horizontal, 16)
    }

    private func statItem(value: String, label: String, icon: String, color: Color) -> some View {
        VStack(spacing: 4) {
            HStack(spacing: 4) {
                Image(systemName: icon)
                    .font(.system(size: 12))
                    .foregroundStyle(color)
                Text(value)
                    .font(.caption.weight(.bold))
                    .foregroundStyle(.primary)
            }
            Text(label)
                .font(.caption2)
                .foregroundStyle(.tertiary)
        }
        .frame(maxWidth: .infinity)
    }

    // MARK: - Redesigned Session Card

    private func redesignedSessionCard<T: SessionCardData>(_ session: T) -> some View {
        let avatar = session.selected_avatar ??
                    (session.selected_avatar_id.flatMap { AvatarManager.shared.getAvatar(id: $0) })
        let avatarImage = avatar.flatMap { AvatarManager.shared.getAvatarImage(id: $0.id) }
        let characterName = avatar?.name ?? "Unknown"
        let characterColor = parseHexColor(avatar?.color) ?? Color(red: 0.58, green: 0.65, blue: 0.65)
        let statusColor = statusColor(session.status)

        return HStack(spacing: 0) {
            // Left status accent bar
            RoundedRectangle(cornerRadius: 2)
                .fill(statusColor)
                .frame(width: 3)
                .padding(.vertical, 4)

            VStack(spacing: 0) {
                // Main content
                HStack(spacing: 12) {
                    // Avatar
                    ZStack {
                        Circle()
                            .fill(characterColor.opacity(0.15))
                            .frame(width: 40, height: 40)

                        if let image = avatarImage {
                            Image(uiImage: image)
                                .resizable()
                                .scaledToFill()
                                .frame(width: 40, height: 40)
                                .clipShape(Circle())
                        } else {
                            Text(getInitials(characterName))
                                .foregroundStyle(characterColor)
                                .font(.system(size: 14, weight: .bold))
                        }
                    }
                    .task(id: avatar?.id) {
                        if let avatar = avatar, AvatarManager.shared.getAvatarImage(id: avatar.id) == nil {
                            await AvatarManager.shared.loadAvatarImageIfNeeded(avatar: avatar)
                        }
                    }

                    // Info
                    VStack(alignment: .leading, spacing: 3) {
                        Text(characterName)
                            .font(.subheadline.weight(.semibold))
                            .foregroundStyle(.primary)
                            .lineLimit(1)

                        HStack(spacing: 6) {
                            Text(session.model_name ?? "Unknown")
                                .font(.caption)
                                .foregroundStyle(.secondary)

                            if let provider = session.provider {
                                Text("·")
                                    .font(.caption)
                                    .foregroundStyle(.tertiary)

                                Text(provider)
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            }
                        }
                    }

                    Spacer()

                    // Right side: status + time
                    VStack(alignment: .trailing, spacing: 6) {
                        Text(session.status.capitalized)
                            .font(.caption2.weight(.bold))
                            .padding(.horizontal, 7)
                            .padding(.vertical, 3)
                            .background(statusColor.opacity(0.12))
                            .foregroundStyle(statusColor)
                            .clipShape(Capsule())

                        Text(formatDateRelative(session.created_at))
                            .font(.caption2)
                            .foregroundStyle(.tertiary)
                    }
                }
                .padding(.horizontal, 14)
                .padding(.top, 12)
                .padding(.bottom, 10)

                // Bottom metadata row
                HStack(spacing: 0) {
                    // Session ID chip
                    Text(String(session.id.prefix(8)))
                        .font(.system(.caption2, design: .monospaced))
                        .foregroundStyle(.secondary)
                        .padding(.horizontal, 6)
                        .padding(.vertical, 2)
                        .background(Color(.systemGray6))
                        .clipShape(RoundedRectangle(cornerRadius: 3))

                    Spacer()

                    // Message count
                    Label("\(session.displayMessageCount)", systemImage: "message")
                        .font(.caption2)
                        .foregroundStyle(.tertiary)

                    // Cost
                    Text(String(format: "$%.4f", session.cost_value))
                        .font(.caption2.weight(.medium))
                        .foregroundStyle(.secondary)
                        .padding(.leading, 8)
                }
                .padding(.horizontal, 14)
                .padding(.bottom, 10)
                .background(Color(.systemGray6).opacity(0.3))
            }
        }
        .background(Color(.secondarySystemGroupedBackground))
        .clipShape(RoundedRectangle(cornerRadius: 12))
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    // MARK: - Bottom Metrics Panel (inspired by web MetricsSidebar)

    private var bottomMetricsPanel: some View {
        VStack(spacing: 0) {
            // Section Header
            HStack {
                Image(systemName: "chart.bar.fill")
                    .font(.system(size: 12))
                    .foregroundStyle(WeeColors.accent)
                Text("Project Analytics")
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(.secondary)
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(.horizontal, 16)
            .padding(.bottom, 8)

            // Main metrics card
            VStack(spacing: 0) {
                // ── Status Distribution ──
                metricsSectionHeader(icon: "circle.hexagongrid.fill", title: "STATUS DISTRIBUTION")

                HStack(spacing: 8) {
                    statusDistributionBar(
                        active: activeSessionCount,
                        completed: completedSessionCount,
                        failed: failedSessionCount,
                        other: projectSessions.count - activeSessionCount - completedSessionCount - failedSessionCount
                    )
                }
                .padding(.horizontal, 16)
                .padding(.bottom, 12)

                // Status legend
                HStack(spacing: 12) {
                    if activeSessionCount > 0 {
                        legendItem(color: WeeColors.success, label: "Active (\(activeSessionCount))")
                    }
                    if completedSessionCount > 0 {
                        legendItem(color: .gray, label: "Done (\(completedSessionCount))")
                    }
                    if failedSessionCount > 0 {
                        legendItem(color: WeeColors.error, label: "Failed (\(failedSessionCount))")
                    }
                    let otherCount = projectSessions.count - activeSessionCount - completedSessionCount - failedSessionCount
                    if otherCount > 0 {
                        legendItem(color: WeeColors.warning, label: "Other (\(otherCount))")
                    }
                }
                .font(.caption2)
                .padding(.horizontal, 16)
                .padding(.bottom, 16)

                // ── Divider ──
                Divider().padding(.horizontal, 16)

                // ── Cost & Usage Grid ──
                metricsSectionHeader(icon: "dollarsign.circle.fill", title: "COST & USAGE")

                LazyVGrid(columns: [
                    GridItem(.flexible()),
                    GridItem(.flexible())
                ], spacing: 12) {
                    metricTile(
                        icon: "dollarsign.circle",
                        value: String(format: "$%.4f", totalCost),
                        label: "Total Cost",
                        color: WeeColors.accent
                    )
                    metricTile(
                        icon: "chart.line.downtrend.xyaxis",
                        value: String(format: "$%.4f", avgCostPerSession),
                        label: "Avg / Session",
                        color: WeeColors.accent
                    )
                    metricTile(
                        icon: "message.fill",
                        value: "\(totalMessages)",
                        label: "Total Messages",
                        color: WeeColors.success
                    )
                    metricTile(
                        icon: "clock.fill",
                        value: formatDuration(totalDurationMs),
                        label: "Total Duration",
                        color: WeeColors.warning
                    )
                }
                .padding(.horizontal, 16)
                .padding(.bottom, 16)

                // ── Divider ──
                Divider().padding(.horizontal, 16)

                // ── Model Breakdown ──
                if !modelBreakdown.isEmpty {
                    metricsSectionHeader(icon: "cpu", title: "MODELS")

                    VStack(spacing: 8) {
                        ForEach(modelBreakdown.prefix(4), id: \.name) { model in
                            modelBreakdownRow(
                                name: model.name,
                                count: model.count,
                                total: projectSessions.count
                            )
                        }
                    }
                    .padding(.horizontal, 16)
                    .padding(.bottom, 16)
                }

                // ── Divider ──
                if !providerBreakdown.isEmpty && providerBreakdown.count > 1 {
                    Divider().padding(.horizontal, 16)

                    // ── Provider Breakdown ──
                    metricsSectionHeader(icon: "server.rack", title: "PROVIDERS")

                    HStack(spacing: 6) {
                        ForEach(providerBreakdown.prefix(4), id: \.name) { provider in
                            providerBadge(name: provider.name, count: provider.count)
                        }
                    }
                    .padding(.horizontal, 16)
                    .padding(.bottom, 16)
                }
            }
            .background(Color(.secondarySystemGroupedBackground))
            .clipShape(RoundedRectangle(cornerRadius: 12))
            .padding(.horizontal, 16)
        }
    }

    // MARK: Metrics Panel Sub-components

    private func metricsSectionHeader(icon: String, title: String) -> some View {
        HStack(spacing: 6) {
            Image(systemName: icon)
                .font(.system(size: 10))
                .foregroundStyle(.secondary)
            Text(title)
                .font(.system(size: 10, weight: .bold))
                .foregroundStyle(.secondary)
                .tracking(0.5)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(.horizontal, 16)
        .padding(.top, 14)
        .padding(.bottom, 10)
    }

    private func statusDistributionBar(active: Int, completed: Int, failed: Int, other: Int) -> some View {
        let total = CGFloat(active + completed + failed + other)

        return GeometryReader { geometry in
            HStack(spacing: 2) {
                if active > 0 {
                    RoundedRectangle(cornerRadius: 3)
                        .fill(WeeColors.success)
                        .frame(width: max(4, geometry.size.width * CGFloat(active) / total))
                }
                if completed > 0 {
                    RoundedRectangle(cornerRadius: 3)
                        .fill(Color.gray.opacity(0.6))
                        .frame(width: max(4, geometry.size.width * CGFloat(completed) / total))
                }
                if failed > 0 {
                    RoundedRectangle(cornerRadius: 3)
                        .fill(WeeColors.error.opacity(0.7))
                        .frame(width: max(4, geometry.size.width * CGFloat(failed) / total))
                }
                if other > 0 {
                    RoundedRectangle(cornerRadius: 3)
                        .fill(WeeColors.warning.opacity(0.5))
                        .frame(width: max(4, geometry.size.width * CGFloat(other) / total))
                }
            }
        }
        .frame(height: 8)
        .clipShape(RoundedRectangle(cornerRadius: 4))
    }

    private func legendItem(color: Color, label: String) -> some View {
        HStack(spacing: 4) {
            Circle()
                .fill(color)
                .frame(width: 6, height: 6)
            Text(label)
                .foregroundStyle(.secondary)
        }
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

    private func modelBreakdownRow(name: String, count: Int, total: Int) -> some View {
        let pct = total > 0 ? Double(count) / Double(total) : 0

        return VStack(spacing: 6) {
            HStack {
                Text(name)
                    .font(.caption.weight(.medium))
                    .foregroundStyle(.primary)
                    .lineLimit(1)

                Spacer()

                Text("\(count) session\(count != 1 ? "s" : "")")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }

            // Progress bar
            GeometryReader { geometry in
                ZStack(alignment: .leading) {
                    RoundedRectangle(cornerRadius: 2)
                        .fill(Color(.systemGray5))
                        .frame(height: 4)

                    RoundedRectangle(cornerRadius: 2)
                        .fill(WeeColors.accent.opacity(0.6))
                        .frame(width: geometry.size.width * pct, height: 4)
                }
            }
            .frame(height: 4)
        }
    }

    private func providerBadge(name: String, count: Int) -> some View {
        HStack(spacing: 4) {
            Circle()
                .fill(WeeColors.accent.opacity(0.6))
                .frame(width: 6, height: 6)
            Text(name.capitalized)
                .font(.caption.weight(.medium))
                .foregroundStyle(.primary)
            Text("(\(count))")
                .font(.caption2)
                .foregroundStyle(.secondary)
        }
        .padding(.horizontal, 10)
        .padding(.vertical, 6)
        .background(Color(.systemGray6).opacity(0.5))
        .clipShape(Capsule())
    }

    private func loadProject() async {
        isLoading = true
        errorMessage = nil

        defer { isLoading = false }

        do {
            // Fetch projects list and find the one with matching ID
            let projects = try await apiClient.getProjects()
            if let foundProject = projects.first(where: { $0.id == projectID }) {
                self.project = foundProject
            } else {
                errorMessage = "Project not found"
            }
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    private func loadProjectSessions() async {
        sessionsLoading = true

        do {
            // Fetch sessions specific to this project
            let sessions = try await apiClient.getProjectSessions(projectID: projectID)
            self.projectSessions = sessions
        } catch {
            // Don't fail the entire view, just show no sessions
            self.projectSessions = []
        }

        sessionsLoading = false
    }

    private func metadataRow(label: String, value: String) -> some View {
        HStack {
            Text(label)
                .font(.subheadline)
                .foregroundStyle(.secondary)
            Spacer()
            Text(value)
                .font(.subheadline.bold())
                .lineLimit(2)
                .multilineTextAlignment(.trailing)
        }
    }

    private func statusBadge(_ status: String) -> some View {
        Text(status.capitalized)
            .font(.caption2.bold())
            .padding(.horizontal, 8)
            .padding(.vertical, 4)
            .background(statusColor(status).opacity(0.2))
            .foregroundStyle(statusColor(status))
            .cornerRadius(4)
    }

    private func statusColor(_ status: String) -> Color {
        switch status.lowercased() {
        case "active", "running", "processing":
            return WeeColors.success
        case "pending", "planning":
            return WeeColors.accent
        case "completed":
            return .gray
        case "failed", "error":
            return WeeColors.error
        case "paused":
            return WeeColors.warning
        default:
            return WeeColors.warning
        }
    }

    private func formatDateRelative(_ dateString: String) -> String {
        guard let date = parseISO8601Date(dateString) else {
            return "Unknown"
        }

        let relativeFormatter = RelativeDateTimeFormatter()
        relativeFormatter.unitsStyle = .abbreviated
        return relativeFormatter.localizedString(for: date, relativeTo: Date())
    }

    private func formatDateDetailed(_ dateString: String) -> String {
        guard let date = parseISO8601Date(dateString) else {
            return "Unknown"
        }

        let formatter = DateFormatter()
        formatter.dateStyle = .medium
        formatter.timeStyle = .short
        return formatter.string(from: date)
    }

    private func parseISO8601Date(_ dateString: String) -> Date? {
        // Try multiple ISO8601 formats to handle different variations
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

        // Fallback: try to parse with standard formatter
        let dateFormatter = DateFormatter()
        dateFormatter.dateFormat = "yyyy-MM-dd'T'HH:mm:ss.SSSSSSZ"
        if let date = dateFormatter.date(from: dateString) {
            return date
        }

        return nil
    }

    private func formatDate(_ dateString: String) -> String {
        // Legacy function for backwards compatibility
        return formatDateRelative(dateString)
    }

    private func formatDuration(_ milliseconds: Int64) -> String {
        let seconds = milliseconds / 1000
        let minutes = seconds / 60
        let hours = minutes / 60

        if hours > 0 {
            return "\(hours)h \(minutes % 60)m"
        } else if minutes > 0 {
            return "\(minutes)m"
        } else {
            return "\(seconds)s"
        }
    }

    private func getInitials(_ name: String) -> String {
        let components = name.split(separator: " ")
        if components.count >= 2 {
            // Return first letter of first and last name (e.g., "John Smith" -> "JS")
            return String(components[0].prefix(1)) + String(components[components.count - 1].prefix(1))
        } else if !name.isEmpty {
            // Return first two letters if single word
            let shortened = name.prefix(2)
            return String(shortened).uppercased()
        }
        return "??"
    }

    private func deleteSession(_ sessionId: String) {
        #if DEBUG
        print("🗑️  Deleting session: \(sessionId)")
        #endif

        Task {
            // Check if WebSocket is connected, reconnect if needed
            if !webSocketService.isConnected {
                #if DEBUG
                print("❌ WebSocket not connected. Attempting to connect...")
                #endif
                await webSocketService.connect()
                try? await Task.sleep(nanoseconds: 500_000_000)  // 0.5 second for connection

                if !webSocketService.isConnected {
                    errorMessage = "WebSocket not connected. Unable to delete session."
                    return
                }
            }

            #if DEBUG
            print("📤 Sending delete_session message...")
            #endif
            #if DEBUG
            print("   Session ID: \(sessionId)")
            #endif

            let success = webSocketService.send([
                "type": "delete_session",
                "session_id": sessionId
            ])

            if success {
                #if DEBUG
                print("✅ Delete session message sent")
                #endif
                // Reload sessions after deletion
                #if DEBUG
                print("⏳ Waiting for deletion to complete...")
                #endif
                try? await Task.sleep(nanoseconds: 1_000_000_000)  // 1 second
                await loadProjectSessions()
            } else {
                #if DEBUG
                print("❌ Failed to send delete session message")
                #endif
                errorMessage = "Failed to send delete message. Please try again."
            }
        }
    }

    private func deleteSelectedSessions() {
        #if DEBUG
        print("🗑️  Deleting \(selectedSessionIDs.count) sessions")
        #endif

        Task {
            // Check if WebSocket is connected, reconnect if needed
            if !webSocketService.isConnected {
                #if DEBUG
                print("❌ WebSocket not connected. Attempting to connect...")
                #endif
                await webSocketService.connect()
                try? await Task.sleep(nanoseconds: 500_000_000)  // 0.5 second for connection

                if !webSocketService.isConnected {
                    errorMessage = "WebSocket not connected. Unable to delete sessions."
                    return
                }
            }

            // Delete each selected session
            for sessionId in selectedSessionIDs {
                #if DEBUG
                print("📤 Sending delete_session message for: \(sessionId)")
                #endif

                let success = webSocketService.send([
                    "type": "delete_session",
                    "session_id": sessionId
                ])

                if success {
                    #if DEBUG
                    print("✅ Delete session message sent for: \(sessionId)")
                    #endif
                } else {
                    #if DEBUG
                    print("❌ Failed to send delete session message for: \(sessionId)")
                    #endif
                }

                // Small delay between deletions
                try? await Task.sleep(nanoseconds: 100_000_000)  // 0.1 second
            }

            // Exit selection mode and clear selection
            await MainActor.run {
                isSelectionMode = false
                selectedSessionIDs.removeAll()
            }

            // Reload sessions after all deletions
            #if DEBUG
            print("⏳ Waiting for deletions to complete...")
            #endif
            try? await Task.sleep(nanoseconds: 1_000_000_000)  // 1 second
            await loadProjectSessions()
        }
    }

    private func parseHexColor(_ hexString: String?) -> Color? {
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
}

// MARK: - Supporting Views

struct Badge: View {
    let label: String
    let icon: String
    let color: Color

    var body: some View {
        HStack(spacing: 4) {
            Image(systemName: icon)
                .font(.system(size: 8))
            Text(label.capitalized)
                .font(.caption)
        }
        .padding(.horizontal, 8)
        .padding(.vertical, 4)
        .background(color.opacity(0.2))
        .foregroundStyle(color)
        .cornerRadius(4)
    }
}

#Preview {
    ProjectDetailView(projectID: "lp_example")
}
