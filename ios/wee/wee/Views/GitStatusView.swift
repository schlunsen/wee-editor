//
//  GitStatusView.swift
//  wee
//
//  View for displaying git status and diff for a session's project
//

import SwiftUI

struct GitStatusView: View {
    let sessionID: String

    @State private var gitStatus: GitStatus?
    @State private var diffData: GitDiffData?
    @State private var remoteInfo: GitRemoteInfo?
    @State private var isLoading = true
    @State private var errorMessage: String?
    @State private var showDiff = false

    private let apiClient = WeeAPIClient.shared

    var body: some View {
        ScrollView {
            LazyVStack(spacing: 12) {
                if isLoading {
                    loadingView
                } else if let error = errorMessage {
                    errorView(error)
                } else {
                    if let status = gitStatus {
                        statusCard(status)
                    }
                    if let remote = remoteInfo {
                        remoteCard(remote)
                    }
                    if let diff = diffData {
                        diffSection(diff)
                    }
                }
            }
            .padding(.top, 8)
            .padding(.bottom, 16)
        }
        .background(Color(.systemGroupedBackground))
        .navigationTitle("Git")
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
            Text("Loading git info...")
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
            Text("Unable to Load Git Info")
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

    // MARK: - Status Card

    private func statusCard(_ status: GitStatus) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            // Branch header
            HStack(spacing: 8) {
                Image(systemName: "branch.fill")
                    .font(.system(size: 14))
                    .foregroundStyle(WeeColors.accent)
                Text(status.branch ?? "Unknown Branch")
                    .font(.title3.weight(.bold))
                    .foregroundStyle(.primary)
            }

            Divider()

            // Clean/dirty status
            HStack(spacing: 16) {
                statusChip(
                    label: "Status",
                    value: status.clean == true ? "Clean" : "Dirty",
                    color: status.clean == true ? WeeColors.success : WeeColors.warning,
                    icon: status.clean == true ? "checkmark.circle.fill" : "exclamationmark.triangle.fill"
                )

                if let ahead = status.ahead, ahead > 0 {
                    statusChip(
                        label: "Ahead",
                        value: "\(ahead)",
                        color: WeeColors.accent,
                        icon: "arrow.up.circle.fill"
                    )
                }

                if let behind = status.behind, behind > 0 {
                    statusChip(
                        label: "Behind",
                        value: "\(behind)",
                        color: WeeColors.error,
                        icon: "arrow.down.circle.fill"
                    )
                }
            }

            // File lists
            if let staged = status.staged, !staged.isEmpty {
                fileListSection(title: "Staged", files: staged, color: WeeColors.success)
            }

            if let modified = status.modified, !modified.isEmpty {
                fileListSection(title: "Modified", files: modified, color: WeeColors.warning)
            }

            if let untracked = status.untracked, !untracked.isEmpty {
                fileListSection(title: "Untracked", files: untracked, color: WeeColors.accent)
            }

            if let deleted = status.deleted, !deleted.isEmpty {
                fileListSection(title: "Deleted", files: deleted, color: WeeColors.error)
            }
        }
        .padding(16)
        .background(Color(.secondarySystemGroupedBackground))
        .clipShape(RoundedRectangle(cornerRadius: 12))
        .padding(.horizontal, 16)
    }

    private func statusChip(label: String, value: String, color: Color, icon: String) -> some View {
        VStack(spacing: 4) {
            Image(systemName: icon)
                .font(.system(size: 16))
                .foregroundStyle(color)
            Text(value)
                .font(.caption.weight(.bold))
                .foregroundStyle(color)
            Text(label)
                .font(.caption2)
                .foregroundStyle(.tertiary)
        }
        .frame(maxWidth: .infinity)
    }

    private func fileListSection(title: String, files: [String], color: Color) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(spacing: 6) {
                Circle()
                    .fill(color)
                    .frame(width: 6, height: 6)
                Text(title)
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(.secondary)
                Text("(\(files.count))")
                    .font(.caption2)
                    .foregroundStyle(.tertiary)
            }

            ForEach(files.prefix(10), id: \.self) { file in
                HStack(spacing: 6) {
                    Image(systemName: "doc.fill")
                        .font(.caption2)
                        .foregroundStyle(color)
                    Text(file)
                        .font(.caption)
                        .foregroundStyle(.primary)
                        .lineLimit(1)
                }
                .padding(.vertical, 2)
            }

            if files.count > 10 {
                Text("+ \(files.count - 10) more")
                    .font(.caption2)
                    .foregroundStyle(.tertiary)
            }
        }
        .padding(.top, 8)
    }

    // MARK: - Remote Card

    private func remoteCard(_ remote: GitRemoteInfo) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack(spacing: 8) {
                Image(systemName: "network")
                    .font(.system(size: 14))
                    .foregroundStyle(WeeColors.accent)
                Text("Remote")
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(.primary)
            }

            if let htmlUrl = remote.html_url, !htmlUrl.isEmpty {
                Button(action: {
                    if let url = URL(string: htmlUrl) {
                        UIApplication.shared.open(url)
                    }
                }) {
                    HStack(spacing: 6) {
                        Image(systemName: "safari.fill")
                            .font(.caption)
                            .foregroundStyle(WeeColors.accent)
                        Text(htmlUrl)
                            .font(.system(.caption, design: .monospaced))
                            .foregroundStyle(WeeColors.accent)
                            .lineLimit(1)
                    }
                }
            }

            if let owner = remote.owner, let repo = remote.repo {
                HStack(spacing: 6) {
                    Image(systemName: "person.fill")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    Text("\(owner)/\(repo)")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }
        }
        .padding(16)
        .background(Color(.secondarySystemGroupedBackground))
        .clipShape(RoundedRectangle(cornerRadius: 12))
        .padding(.horizontal, 16)
    }

    // MARK: - Diff Section

    private func diffSection(_ diff: GitDiffData) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack {
                Image(systemName: "doc.text.magnifyingglass")
                    .font(.system(size: 12))
                    .foregroundStyle(WeeColors.accent)
                Text("Diff")
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(.secondary)
                Spacer()

                if let stats = diff.stats {
                    Text("+\(stats.additions ?? 0) / -\(stats.deletions ?? 0) in \(stats.filesChanged ?? 0) files")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }
            .padding(.horizontal, 16)

            // File changes list
            if let files = diff.files, !files.isEmpty {
                ForEach(files, id: \.path) { file in
                    diffFileCard(file)
                }
            }

            // Full diff text (collapsible)
            if let files = diff.files {
                let allDiffs = files.compactMap { $0.diff }.filter { !$0.isEmpty }.joined(separator: "\n\n")
                if !allDiffs.isEmpty {
                    Button(action: { showDiff.toggle() }) {
                        HStack {
                            Image(systemName: showDiff ? "chevron.up" : "chevron.down")
                                .font(.caption)
                            Text(showDiff ? "Hide Full Diff" : "Show Full Diff")
                                .font(.caption.weight(.medium))
                        }
                        .foregroundStyle(WeeColors.accent)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 8)
                    }
                    .padding(.horizontal, 16)

                    if showDiff {
                        diffTextView(allDiffs)
                    }
                }
            }
        }
    }

    private func diffFileCard(_ file: GitDiffFile) -> some View {
        HStack(spacing: 10) {
            // Status icon
            Circle()
                .fill(diffStatusColor(file.status))
                .frame(width: 8, height: 8)

            VStack(alignment: .leading, spacing: 2) {
                Text(file.path ?? "Unknown")
                    .font(.caption.weight(.medium))
                    .foregroundStyle(.primary)
                    .lineLimit(1)

                if let status = file.status {
                    Text(status.capitalized)
                        .font(.caption2)
                        .foregroundStyle(diffStatusColor(status))
                }
            }

            Spacer()

            if let diffText = file.diff, !diffText.isEmpty {
                Image(systemName: "doc.text")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 8)
        .background(Color(.secondarySystemGroupedBackground))
        .clipShape(RoundedRectangle(cornerRadius: 8))
        .padding(.horizontal, 16)
    }

    private func diffTextView(_ text: String) -> some View {
        ScrollView {
            Text(diffAttributedString(text))
                .font(.system(.caption, design: .monospaced))
                .textSelection(.enabled)
                .frame(maxWidth: .infinity, alignment: .leading)
                .padding(12)
        }
        .frame(maxHeight: 400)
        .background(Color(.systemGray6))
        .clipShape(RoundedRectangle(cornerRadius: 8))
        .padding(.horizontal, 16)
    }

    // MARK: - Helpers

    private func diffStatusColor(_ status: String?) -> Color {
        switch status?.lowercased() {
        case "added", "new":
            return WeeColors.success
        case "modified", "changed":
            return WeeColors.warning
        case "deleted", "removed":
            return WeeColors.error
        default:
            return .gray
        }
    }

    private func diffAttributedString(_ text: String) -> AttributedString {
        var result = AttributedString()
        let lines = text.components(separatedBy: "\n")

        for line in lines {
            var lineAttr = AttributedString(line + "\n")
            if line.hasPrefix("+") && !line.hasPrefix("+++") {
                lineAttr.foregroundColor = WeeColors.success
            } else if line.hasPrefix("-") && !line.hasPrefix("---") {
                lineAttr.foregroundColor = WeeColors.error
            } else if line.hasPrefix("@@") {
                lineAttr.foregroundColor = UIColor(red: 0.486, green: 0.227, blue: 0.929, alpha: 1)
            } else if line.hasPrefix("diff ") || line.hasPrefix("index ") {
                lineAttr.foregroundColor = .secondary
            } else {
                lineAttr.foregroundColor = .primary
            }
            result.append(lineAttr)
        }

        return result
    }

    // MARK: - Data Loading

    private func loadData() async {
        isLoading = true
        errorMessage = nil

        do {
            async let statusResult = try await apiClient.getGitStatus(sessionID: sessionID)
            async let diffResult = try await apiClient.getGitDiff(sessionID: sessionID)
            async let remoteResult = try await apiClient.getGitRemote(sessionID: sessionID)

            // Fetch all in parallel
            let status = statusResult
            let diff = diffResult
            let remote = remoteResult

            await MainActor.run {
                self.gitStatus = status
                self.diffData = diff
                self.remoteInfo = remote
                self.isLoading = false
            }
        } catch {
            await MainActor.run {
                self.errorMessage = error.localizedDescription
                self.isLoading = false
            }
        }
    }
}

#Preview {
    NavigationStack {
        GitStatusView(sessionID: "example-session-id")
    }
}
