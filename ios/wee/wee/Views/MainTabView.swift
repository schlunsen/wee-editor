//
//  MainTabView.swift
//  wee
//
//  Main tab-based navigation for authenticated users
//

import SwiftUI

struct MainTabView: View {
    @EnvironmentObject var authViewModel: AuthenticationViewModel
    @State private var selectedTab: Tab = .dashboard
    @State private var showHostSwitcher = false
    @State private var projectsNavigationPath = NavigationPath()

    enum Tab {
        case dashboard
        case projects
        case settings
    }

    var body: some View {
        ZStack {
            TabView(selection: $selectedTab.withTabAnimation().withHaptics()) {
                // Dashboard Tab
                DashboardView(selectedTab: $selectedTab, projectsNavigationPath: $projectsNavigationPath)
                    .tabItem {
                        Label("Dashboard", systemImage: "chart.bar.fill")
                    }
                    .tag(Tab.dashboard)
                    .transition(.asymmetric(
                        insertion: .opacity.combined(with: .scale(scale: 0.95)),
                        removal: .opacity
                    ))
                    .toolbar {
                        hostSwitcherToolbarItem
                        menuToolbarItem
                    }

                // Projects Tab
                ProjectsListView(externalNavigationPath: $projectsNavigationPath)
                    .tabItem {
                        Label("Projects", systemImage: "folder.fill")
                    }
                    .tag(Tab.projects)
                    .transition(.asymmetric(
                        insertion: .opacity.combined(with: .scale(scale: 0.95)),
                        removal: .opacity
                    ))
                    .toolbar {
                        hostSwitcherToolbarItem
                        menuToolbarItem
                    }

                // Settings Tab
                SettingsView(authViewModel: authViewModel)
                    .tabItem {
                        Label("Settings", systemImage: "gear")
                    }
                    .tag(Tab.settings)
                    .transition(.asymmetric(
                        insertion: .opacity.combined(with: .scale(scale: 0.95)),
                        removal: .opacity
                    ))
                    .toolbar {
                        hostSwitcherToolbarItem
                        menuToolbarItem
                    }
            }
            .navigationBarBackButtonHidden(true)

            // Host Switcher Menu
            if showHostSwitcher {
                hostSwitcherMenu
            }
        }
    }

    private var hostSwitcherToolbarItem: some ToolbarContent {
        ToolbarItem(placement: .topBarLeading) {
            Button(action: { showHostSwitcher.toggle() }) {
                HStack(spacing: 6) {
                    Image(systemName: "server.rack")
                        .font(.system(size: 14))
                    Text(authViewModel.currentHost?.name ?? "No Host")
                        .font(.caption.bold())
                }
                .padding(.horizontal, 10)
                .padding(.vertical, 6)
                .background(WeeColors.accentLight)
                .foregroundStyle(WeeColors.accent)
                .cornerRadius(6)
            }
        }
    }

    private var menuToolbarItem: some ToolbarContent {
        ToolbarItem(placement: .topBarTrailing) {
            Menu {
                NavigationLink(destination: HostsManagementView(authViewModel: authViewModel)) {
                    Label("Manage Hosts", systemImage: "server.rack")
                }
                Divider()
                Button(action: {
                    Task {
                        await DashboardViewModel().refresh()
                    }
                }) {
                    Label("Refresh", systemImage: "arrow.clockwise")
                }
            } label: {
                Image(systemName: "ellipsis.circle.fill")
                    .font(.system(size: 16))
            }
        }
    }

    private var hostSwitcherMenu: some View {
        VStack(alignment: .leading, spacing: 0) {
            VStack(alignment: .leading, spacing: 12) {
                HStack {
                    Text("Switch Host")
                        .font(.headline)
                    Spacer()
                    Button(action: { showHostSwitcher = false }) {
                        Image(systemName: "xmark.circle.fill")
                            .foregroundStyle(.secondary)
                    }
                }

                Divider()

                VStack(spacing: 8) {
                    ForEach(authViewModel.savedHosts, id: \.id) { host in
                        hostSwitchButton(host)
                    }
                }
            }
            .padding(16)

            Spacer()
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Color(.systemBackground))
        .transition(.move(edge: .leading).combined(with: .opacity))
        .onTapGesture {
            showHostSwitcher = false
        }
    }

    private func hostSwitchButton(_ host: Host) -> some View {
        Button(action: {
            switchToHost(host)
        }) {
            HStack {
                VStack(alignment: .leading, spacing: 4) {
                    Text(host.name)
                        .font(.subheadline.bold())
                        .foregroundStyle(.primary)

                    Text(host.url)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                        .lineLimit(1)
                }

                Spacer()

                if authViewModel.currentHost?.id == host.id {
                    Image(systemName: "checkmark.circle.fill")
                        .foregroundStyle(WeeColors.accent)
                }
            }
            .padding(12)
            .background(authViewModel.currentHost?.id == host.id ? WeeColors.accentLight : Color(.systemGray6))
            .cornerRadius(8)
        }
    }

    private func switchToHost(_ host: Host) {
        showHostSwitcher = false

        Task {
            do {
                try await authViewModel.switchHost(hostId: host.id)
            } catch {
            }
        }
    }
}

// MARK: - Dashboard Tab

struct DashboardView: View {
    @Binding var selectedTab: MainTabView.Tab
    @Binding var projectsNavigationPath: NavigationPath
    @StateObject private var viewModel = DashboardViewModel()
    @State private var showCreateSessionModal = false
    @State private var appearAnimating = false

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                if viewModel.isLoading {
                    VStack {
                        ProgressView()
                            .tint(WeeColors.accent)
                    }
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else if let error = viewModel.errorMessage {
                    errorView(error)
                } else {
                    ScrollView {
                        LazyVStack(spacing: 16) {
                            // MARK: - Summary Metrics
                            summaryMetricsRow

                            // MARK: - Recent Sessions
                            recentSessionsSection

                            // MARK: - Recent Projects
                            recentProjectsSection
                        }
                        .padding(.top, 8)
                        .padding(.bottom, 16)
                    }
                    .refreshable {
                        await viewModel.refresh()
                    }
                }
            }
            .background(WeeColors.background)
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    HStack(spacing: 12) {
                        Button(action: { showCreateSessionModal = true }) {
                            Image(systemName: "plus.circle.fill")
                                .font(.system(size: 14, weight: .semibold))
                        }

                        Button(action: {
                            Task { await viewModel.refresh() }
                        }) {
                            Image(systemName: "arrow.clockwise")
                                .font(.system(size: 14, weight: .semibold))
                        }
                    }
                }
            }
            .sheet(isPresented: $showCreateSessionModal) {
                CreateSessionView()
            }
        }
        .task {
            await viewModel.loadData()
            withAnimation {
                appearAnimating = true
            }
        }
    }

    // MARK: - Summary Metrics

    private var summaryMetricsRow: some View {
        HStack(spacing: 10) {
            metricPill(
                value: "\(viewModel.activeSessions)",
                label: "Active",
                color: WeeColors.accent,
                showPulse: viewModel.activeSessions > 0
            )

            metricPill(
                value: "\(viewModel.totalSessions)",
                label: "Sessions",
                color: WeeColors.textSecondary,
                showPulse: false
            )

            metricPill(
                value: String(format: "$%.2f", viewModel.totalCost),
                label: "Cost",
                color: WeeColors.success,
                showPulse: false
            )
        }
        .padding(.horizontal, 16)
    }

    private func metricPill(value: String, label: String, color: Color, showPulse: Bool) -> some View {
        HStack(spacing: 6) {
            if showPulse {
                Circle()
                    .fill(color)
                    .frame(width: 6, height: 6)
                    .overlay(
                        Circle()
                            .stroke(color.opacity(0.5), lineWidth: 1.5)
                            .scaleEffect(1.6)
                            .opacity(0.8)
                    )
            }

            Text(value)
                .font(.subheadline.weight(.bold))
                .foregroundStyle(color)

            Text(label)
                .font(.caption2)
                .foregroundStyle(WeeColors.textTertiary)
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 8)
        .frame(maxWidth: .infinity)
        .background(WeeColors.surface)
        .clipShape(RoundedRectangle(cornerRadius: 10))
    }

    // MARK: - Recent Sessions Section

    private var recentSessionsSection: some View {
        VStack(alignment: .leading, spacing: 10) {
            sectionHeader(
                title: "Recent Sessions",
                subtitle: viewModel.activeSessions > 0 ? "\(viewModel.activeSessions) active" : nil
            )

            if viewModel.recentSessions.isEmpty {
                emptyStateView(
                    icon: "text.bubble",
                    title: "No Sessions",
                    subtitle: "Sessions will appear here as agents run"
                )
            } else {
                ForEach(Array(viewModel.recentSessions.enumerated()), id: \.element.id) { index, session in
                    Button {
                        WeeHaptics.tap()
                        if let projectId = session.project_id {
                            navigateToProjectAndSession(
                                projectId,
                                sessionId: session.id,
                                sessionTitle: session.model_name ?? "Session"
                            )
                        }
                    } label: {
                        dashboardSessionCard(session)
                    }
                    .cardPressStyle()
                    .opacity(appearAnimating ? 1 : 0)
                    .offset(y: appearAnimating ? 0 : 12)
                    .animation(
                        .spring(response: 0.4, dampingFraction: 0.8)
                        .delay(Double(index) * 0.05),
                        value: appearAnimating
                    )
                }
            }
        }
    }

    // MARK: - Recent Projects Section

    private var recentProjectsSection: some View {
        VStack(alignment: .leading, spacing: 10) {
            sectionHeader(
                title: "Recent Projects",
                subtitle: "\(viewModel.projects.count) total"
            )

            if viewModel.recentProjects.isEmpty {
                emptyStateView(
                    icon: "folder.fill.badge.questionmark",
                    title: "No Projects",
                    subtitle: "Projects you generate will appear here"
                )
            } else {
                ForEach(viewModel.recentProjects, id: \.id) { project in
                    Button {
                        WeeHaptics.tap()
                        navigateToProject(project.id)
                    } label: {
                        dashboardProjectRow(project)
                    }
                    .buttonStyle(.plain)
                }
            }
        }
    }

    // MARK: - Navigation Helpers

    private func navigateToProject(_ projectId: String) {
        projectsNavigationPath = NavigationPath()
        projectsNavigationPath.append(projectId)
        selectedTab = .projects
    }

    private func navigateToProjectAndSession(_ projectId: String, sessionId: String, sessionTitle: String) {
        projectsNavigationPath = NavigationPath()
        projectsNavigationPath.append(projectId)
        selectedTab = .projects

        DispatchQueue.main.asyncAfter(deadline: .now() + 0.3) {
            NotificationCenter.default.post(
                name: NSNotification.Name("NavigateToSession"),
                object: nil,
                userInfo: [
                    "sessionId": sessionId,
                    "sessionTitle": sessionTitle,
                    "projectId": projectId
                ]
            )
        }
    }

    // MARK: - Session Card (matches ProjectDetailView style)

    private func dashboardSessionCard(_ session: AgentSession) -> some View {
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
                            .frame(width: 36, height: 36)

                        if let image = avatarImage {
                            Image(uiImage: image)
                                .resizable()
                                .scaledToFill()
                                .frame(width: 36, height: 36)
                                .clipShape(Circle())
                        } else {
                            Text(getInitials(characterName))
                                .foregroundStyle(characterColor)
                                .font(.system(size: 13, weight: .bold))
                        }
                    }
                    .task(id: avatar?.id) {
                        if let avatar = avatar, AvatarManager.shared.getAvatarImage(id: avatar.id) == nil {
                            await AvatarManager.shared.loadAvatarImageIfNeeded(avatar: avatar)
                        }
                    }

                    // Info
                    VStack(alignment: .leading, spacing: 3) {
                        // Project name if available
                        if let projectId = session.project_id {
                            Text(projectName(for: projectId))
                                .font(.caption)
                                .foregroundStyle(.tertiary)
                                .lineLimit(1)
                        }

                        HStack(spacing: 6) {
                            Text(session.model_name ?? "Unknown")
                                .font(.subheadline.weight(.medium))
                                .foregroundStyle(.primary)
                                .lineLimit(1)

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

                    // Status + time
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

                // Bottom metadata
                HStack(spacing: 0) {
                    Text(String(session.id.prefix(8)))
                        .font(.system(.caption2, design: .monospaced))
                        .foregroundStyle(.secondary)
                        .padding(.horizontal, 6)
                        .padding(.vertical, 2)
                        .background(Color(.systemGray6))
                        .clipShape(RoundedRectangle(cornerRadius: 3))

                    Spacer()

                    if let msgCount = session.message_count, msgCount > 0 {
                        Label("\(msgCount)", systemImage: "message")
                            .font(.caption2)
                            .foregroundStyle(.tertiary)
                    }

                    if let cost = session.cost, cost > 0 {
                        Text(String(format: "$%.4f", cost))
                            .font(.caption2.weight(.medium))
                            .foregroundStyle(.secondary)
                            .padding(.leading, 8)
                    }
                }
                .padding(.horizontal, 14)
                .padding(.bottom, 10)
                .background(Color(.systemGray6).opacity(0.3))
            }
        }
        .background(Color(.secondarySystemGroupedBackground))
        .clipShape(RoundedRectangle(cornerRadius: 12))
        .padding(.horizontal, 16)
    }

    // MARK: - Project Row (compact, matches ProjectsListView style)

    private func dashboardProjectRow(_ project: Project) -> some View {
        HStack(spacing: 10) {
            Circle()
                .fill(project.is_active ? Color.green : Color.orange.opacity(0.6))
                .frame(width: 8, height: 8)

            VStack(alignment: .leading, spacing: 2) {
                Text(project.name)
                    .font(.subheadline.weight(.semibold))
                    .lineLimit(1)
                    .foregroundStyle(.primary)

                HStack(spacing: 6) {
                    if let provider = project.default_provider {
                        Text(provider)
                            .font(.caption2)
                            .foregroundStyle(.tertiary)
                        Text("·")
                            .font(.caption2)
                            .foregroundStyle(.tertiary)
                    }

                    Text(formatDateRelative(project.updated_at))
                        .font(.caption2)
                        .foregroundStyle(.tertiary)
                }
            }

            Spacer()

            Image(systemName: "chevron.right")
                .font(.caption.weight(.medium))
                .foregroundStyle(.quaternary)
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 8)
        .background(Color(.secondarySystemGroupedBackground))
        .clipShape(RoundedRectangle(cornerRadius: 10))
        .padding(.horizontal, 16)
    }

    // MARK: - Shared Components

    private func sectionHeader(title: String, subtitle: String?) -> some View {
        HStack {
            Text(title)
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(.secondary)

            Spacer()

            if let subtitle = subtitle {
                Text(subtitle)
                    .font(.caption)
                    .foregroundStyle(.tertiary)
            }
        }
        .padding(.horizontal, 16)
    }

    private func emptyStateView(icon: String, title: String, subtitle: String) -> some View {
        VStack(spacing: 10) {
            Image(systemName: icon)
                .font(.system(size: 28, weight: .light))
                .foregroundStyle(.quaternary)
            Text(title)
                .font(.subheadline.weight(.medium))
                .foregroundStyle(.secondary)
            Text(subtitle)
                .font(.caption)
                .foregroundStyle(.tertiary)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 24)
        .background(Color(.secondarySystemGroupedBackground))
        .clipShape(RoundedRectangle(cornerRadius: 12))
        .padding(.horizontal, 16)
    }

    private func errorView(_ error: String) -> some View {
        VStack(spacing: 12) {
            Image(systemName: "exclamationmark.triangle.fill")
                .font(.system(size: 36))
                .foregroundStyle(.orange)
            Text("Unable to Load Data")
                .font(.headline)
            Text(error)
                .font(.subheadline)
                .foregroundStyle(.secondary)
                .multilineTextAlignment(.center)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    // MARK: - Helpers

    private func projectName(for projectId: String) -> String {
        viewModel.projects.first(where: { $0.id == projectId })?.name ?? "Unknown Project"
    }

    private func statusColor(_ status: String) -> Color {
        WeeColors.statusColor(for: status)
    }

    private func getInitials(_ name: String) -> String {
        let components = name.split(separator: " ")
        if components.count >= 2 {
            return String(components[0].prefix(1)) + String(components[components.count - 1].prefix(1))
        } else if !name.isEmpty {
            return String(name.prefix(2)).uppercased()
        }
        return "??"
    }

    private func parseHexColor(_ hexString: String?) -> Color? {
        guard let hexString = hexString else { return nil }
        var hex = hexString.trimmingCharacters(in: .whitespaces)
        if hex.hasPrefix("#") { hex.removeFirst() }
        guard hex.count == 6 else { return nil }
        let scanner = Scanner(string: hex)
        var rgbValue: UInt64 = 0
        guard scanner.scanHexInt64(&rgbValue) else { return nil }
        let r = Double((rgbValue >> 16) & 0xFF) / 255.0
        let g = Double((rgbValue >> 8) & 0xFF) / 255.0
        let b = Double(rgbValue & 0xFF) / 255.0
        return Color(red: r, green: g, blue: b)
    }

    private func formatDateRelative(_ dateString: String) -> String {
        let formatters = [
            ISO8601DateFormatter(),
            {
                let f = ISO8601DateFormatter()
                f.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
                return f
            }()
        ]
        for formatter in formatters {
            if let date = formatter.date(from: dateString) {
                let relativeFormatter = RelativeDateTimeFormatter()
                relativeFormatter.unitsStyle = .abbreviated
                return relativeFormatter.localizedString(for: date, relativeTo: Date())
            }
        }
        return dateString
    }
}

// MARK: - Dashboard Navigation

// MARK: - Projects Tab

struct ProjectsListView: View {
    @Binding var externalNavigationPath: NavigationPath
    @StateObject private var viewModel = DashboardViewModel()
    @State private var searchText = ""

    private var filteredProjects: [Project] {
        if searchText.isEmpty {
            return viewModel.projects
                .sorted { $0.updated_at > $1.updated_at }
        }
        let query = searchText.lowercased()
        return viewModel.projects
            .filter {
                $0.name.lowercased().contains(query) ||
                ($0.description?.lowercased().contains(query) ?? false) ||
                ($0.path.lowercased().contains(query)) ||
                ($0.default_provider?.lowercased().contains(query) ?? false)
            }
            .sorted { $0.updated_at > $1.updated_at }
    }

    var body: some View {
        NavigationStack(path: $externalNavigationPath) {
            VStack(spacing: 0) {
                if viewModel.isLoading {
                    VStack {
                        ProgressView()
                            .tint(WeeColors.accent)
                    }
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else if viewModel.projects.isEmpty {
                    VStack(spacing: 12) {
                        Image(systemName: "folder.fill.badge.questionmark")
                            .font(.system(size: 36))
                            .foregroundStyle(.gray)
                        Text("No Projects")
                            .font(.headline)
                        Text("Projects you generate will appear here")
                            .font(.subheadline)
                            .foregroundStyle(.secondary)
                    }
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else {
                    ScrollView {
                        LazyVStack(spacing: 6) {
                            ForEach(filteredProjects, id: \.id) { project in
                                NavigationLink(value: project.id) {
                                    projectRow(project)
                                }
                                .buttonStyle(.plain)
                            }
                        }
                        .padding(.horizontal, 16)
                        .padding(.top, 8)
                    }
                    .refreshable {
                        await viewModel.refresh()
                    }
                }
            }
            .navigationDestination(for: String.self) { projectId in
                ProjectDetailView(projectID: projectId)
            }
            .background(WeeColors.background)
            .searchable(
                text: $searchText,
                placement: .navigationBarDrawer(displayMode: .always),
                prompt: "Search projects..."
            )
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button(action: {
                        Task { await viewModel.refresh() }
                    }) {
                        Image(systemName: "arrow.clockwise")
                            .font(.system(size: 14, weight: .semibold))
                    }
                }
            }
        }
        .task {
            await viewModel.loadData()
        }
    }

    private func projectRow(_ project: Project) -> some View {
        HStack(spacing: 10) {
            Circle()
                .fill(project.is_active ? Color.green : Color.orange.opacity(0.6))
                .frame(width: 8, height: 8)

            VStack(alignment: .leading, spacing: 2) {
                Text(project.name)
                    .font(.subheadline.weight(.semibold))
                    .lineLimit(1)
                    .foregroundStyle(.primary)

                HStack(spacing: 6) {
                    if let provider = project.default_provider {
                        Text(provider)
                            .font(.caption2)
                            .foregroundStyle(.tertiary)
                        Text("·")
                            .font(.caption2)
                            .foregroundStyle(.tertiary)
                    }

                    Text(formatDateRelative(project.updated_at))
                        .font(.caption2)
                        .foregroundStyle(.tertiary)
                }
            }

            Spacer()

            Image(systemName: "chevron.right")
                .font(.caption.weight(.medium))
                .foregroundStyle(.quaternary)
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 8)
        .background(Color(.secondarySystemGroupedBackground))
        .clipShape(RoundedRectangle(cornerRadius: 10))
    }

    private func formatDateRelative(_ dateString: String) -> String {
        let formatters = [
            ISO8601DateFormatter(),
            {
                let f = ISO8601DateFormatter()
                f.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
                return f
            }()
        ]
        for formatter in formatters {
            if let date = formatter.date(from: dateString) {
                let relativeFormatter = RelativeDateTimeFormatter()
                relativeFormatter.unitsStyle = .abbreviated
                return relativeFormatter.localizedString(for: date, relativeTo: Date())
            }
        }
        return dateString
    }
}

// MARK: - Settings Tab

struct SettingsView: View {
    @ObservedObject var authViewModel: AuthenticationViewModel
    @Environment(\.dismiss) var dismiss
    @State private var showHostsManagement = false
    @AppStorage("appearanceMode") private var appearanceMode: String = AppearanceMode.system.rawValue

    private var selectedMode: AppearanceMode {
        AppearanceMode(rawValue: appearanceMode) ?? .system
    }

    var body: some View {
        NavigationStack {
            List {
                // Connection status header
                Section {
                    HStack(spacing: 12) {
                        ZStack {
                            RoundedRectangle(cornerRadius: 10)
                                .fill(WeeColors.accent.opacity(0.15))
                                .frame(width: 40, height: 40)
                            Image(systemName: "server.rack")
                                .font(.system(size: 18))
                                .foregroundStyle(WeeColors.accent)
                        }

                        VStack(alignment: .leading, spacing: 2) {
                            Text(authViewModel.currentHost?.name ?? "No Host")
                                .font(.subheadline.weight(.semibold))
                            HStack(spacing: 4) {
                                Circle()
                                    .fill(WeeColors.success)
                                    .frame(width: 6, height: 6)
                                Text("Connected")
                                    .font(.caption)
                                    .foregroundStyle(WeeColors.textSecondary)
                            }
                        }

                        Spacer()

                        if let currentHost = authViewModel.currentHost {
                            Text(currentHost.url.replacingOccurrences(of: "https://", with: ""))
                                .font(.caption)
                                .foregroundStyle(WeeColors.textTertiary)
                                .lineLimit(1)
                        }
                    }
                    .padding(.vertical, 4)
                }

                Section {
                    NavigationLink(destination: HostsManagementView(authViewModel: authViewModel)) {
                        Label("Manage Hosts", systemImage: "server.rack")
                            .foregroundStyle(.primary)
                    }
                } header: {
                    Text("Server")
                }

                Section {
                    NavigationLink(destination: ProvidersView()) {
                        Label("AI Providers", systemImage: "cpu")
                            .foregroundStyle(.primary)
                    }
                } header: {
                    Text("Providers")
                } footer: {
                    Text("Configure API keys and models for AI providers.")
                        .font(.caption)
                }

                Section {
                    VStack(alignment: .leading, spacing: 12) {
                        Label {
                            Text("Appearance")
                        } icon: {
                            Image(systemName: selectedMode.icon)
                                .foregroundStyle(WeeColors.accent)
                        }

                        Picker("Appearance", selection: $appearanceMode) {
                            ForEach(AppearanceMode.allCases) { mode in
                                Label(mode.label, systemImage: mode.icon)
                                    .tag(mode.rawValue)
                            }
                        }
                        .pickerStyle(.segmented)
                    }
                    .padding(.vertical, 4)
                } header: {
                    Text("Preferences")
                }

                Section {
                    HStack {
                        Label("App Version", systemImage: "info.circle")
                        Spacer()
                        Text("1.0.0")
                            .foregroundStyle(.secondary)
                    }

                    HStack {
                        Label("Build", systemImage: "hammer")
                        Spacer()
                        Text("1")
                            .foregroundStyle(.secondary)
                    }
                } header: {
                    Text("About")
                }

                Section {
                    Button(role: .destructive, action: {
                        authViewModel.logout()
                    }) {
                        HStack {
                            Spacer()
                            Label("Logout", systemImage: "arrow.left.circle.fill")
                                .font(.subheadline.weight(.medium))
                            Spacer()
                        }
                    }
                }
            }
        }
    }
}

#Preview {
    MainTabView()
}
