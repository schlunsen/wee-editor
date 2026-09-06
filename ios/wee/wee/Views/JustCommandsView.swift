//
//  JustCommandsView.swift
//  wee
//
//  View for listing and running justfile recipes for a project
//

import SwiftUI

struct JustCommandsView: View {
    let projectID: String

    @State private var recipes: [JustRecipe] = []
    @State private var hasJustfile = false
    @State private var isLoading = true
    @State private var errorMessage: String?
    @State private var jobs: [JustJob] = []
    @State private var runningRecipeName: String?
    @State private var runningJobId: String?
    @State private var jobOutput: String = ""
    @State private var jobStatus: String = ""
    @State private var streamingTask: Task<Void, Never>?

    // Filter state
    @State private var recipeFilter: String = ""

    // Job detail sheet state
    @State private var selectedJob: JustJob?
    @State private var showJobDetail = false

    // Edit mode for deleting jobs
    @State private var isEditingJobs = false

    private let apiClient = WeeAPIClient.shared

    var filteredRecipes: [JustRecipe] {
        if recipeFilter.isEmpty {
            return recipes
        }
        return recipes.filter { recipe in
            recipe.name.localizedCaseInsensitiveContains(recipeFilter) ||
            recipe.description.localizedCaseInsensitiveContains(recipeFilter)
        }
    }

    var body: some View {
        VStack(spacing: 0) {
            if isLoading {
                VStack {
                    ProgressView()
                        .tint(WeeColors.accent)
                    Text("Loading recipes...")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                        .padding(.top, 8)
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if let error = errorMessage {
                errorView(error)
            } else if !hasJustfile {
                emptyStateView
            } else {
                content
            }
        }
        .background(Color(.systemGroupedBackground))
        .navigationTitle("Just Commands")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                HStack(spacing: 12) {
                    // Edit button for jobs (only show if there are completed jobs)
                    let completedJobs = jobs.filter { $0.status != "running" }
                    if !completedJobs.isEmpty {
                        Button(action: { isEditingJobs.toggle() }) {
                            Text(isEditingJobs ? "Done" : "Edit")
                                .font(.system(size: 14, weight: .semibold))
                        }
                    }

                    Button(action: { Task { await loadData() } }) {
                        Image(systemName: "arrow.clockwise")
                            .font(.system(size: 14, weight: .semibold))
                    }
                }
            }
        }
        .task {
            await loadData()
        }
        .onDisappear {
            streamingTask?.cancel()
        }
        .sheet(isPresented: $showJobDetail) {
            if let job = selectedJob {
                JobDetailSheet(job: job, onDelete: {
                    Task {
                        await deleteJob(job)
                    }
                })
            }
        }
    }

    // MARK: - Content

    @ViewBuilder
    private var content: some View {
        ScrollView {
            LazyVStack(spacing: 12) {
                // Filter search bar
                filterSearchBar

                // Recent jobs
                let completedJobs = jobs.filter { $0.status != "running" }
                if !completedJobs.isEmpty {
                    jobsSection(completedJobs)
                }

                // Recipes section
                if !recipes.isEmpty {
                    recipesSection
                }
            }
            .padding(.top, 8)
            .padding(.bottom, 16)
        }
    }

    // MARK: - Filter Search Bar

    private var filterSearchBar: some View {
        HStack(spacing: 8) {
            Image(systemName: "magnifyingglass")
                .font(.system(size: 14))
                .foregroundStyle(.secondary)

            TextField("Filter recipes...", text: $recipeFilter)
                .font(.subheadline)

            if !recipeFilter.isEmpty {
                Button(action: { recipeFilter = "" }) {
                    Image(systemName: "xmark.circle.fill")
                        .font(.system(size: 16))
                        .foregroundStyle(.secondary)
                }
            }
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 10)
        .background(Color(.secondarySystemGroupedBackground))
        .clipShape(RoundedRectangle(cornerRadius: 10))
        .padding(.horizontal, 16)
    }

    // MARK: - Recipes Section

    private var recipesSection: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack {
                Text("Recipes")
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(.secondary)
                Spacer()
                Text("\(filteredRecipes.count)")
                    .font(.caption)
                    .foregroundStyle(.tertiary)
            }
            .padding(.horizontal, 16)

            if filteredRecipes.isEmpty && !recipeFilter.isEmpty {
                // No matches state
                HStack {
                    Spacer()
                    VStack(spacing: 8) {
                        Image(systemName: "magnifyingglass")
                            .font(.system(size: 24))
                            .foregroundStyle(.quaternary)
                        Text("No recipes match '\(recipeFilter)'")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                    .padding(.vertical, 20)
                    Spacer()
                }
                .padding(.horizontal, 16)
            } else {
                ForEach(filteredRecipes) { recipe in
                    recipeCard(recipe)
                }
            }
        }
    }

    private func recipeCard(_ recipe: JustRecipe) -> some View {
        let isRunning = runningRecipeName == recipe.name

        return VStack(alignment: .leading, spacing: 0) {
            HStack {
                VStack(alignment: .leading, spacing: 4) {
                    HStack(spacing: 6) {
                        Image(systemName: "terminal.fill")
                            .font(.caption)
                            .foregroundStyle(WeeColors.accent)

                        Text(recipe.name)
                            .font(.subheadline.weight(.semibold))
                            .foregroundStyle(.primary)
                    }

                    if !recipe.description.isEmpty {
                        Text(recipe.description)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                            .lineLimit(2)
                    }

                    if let params = recipe.parameters, !params.isEmpty {
                        HStack(spacing: 4) {
                            ForEach(params, id: \.self) { param in
                                Text(param)
                                    .font(.caption2)
                                    .foregroundStyle(WeeColors.accent)
                                    .padding(.horizontal, 6)
                                    .padding(.vertical, 2)
                                    .background(WeeColors.accentLight)
                                    .clipShape(Capsule())
                            }
                        }
                    }
                }

                Spacer()

                if isRunning {
                    ProgressView()
                        .tint(WeeColors.accent)
                        .padding(.trailing, 4)
                } else if runningRecipeName != nil {
                    Text("Run")
                        .font(.caption.weight(.medium))
                        .foregroundStyle(.tertiary)
                } else {
                    Button(action: { runRecipe(recipe) }) {
                        Text("Run")
                            .font(.caption.weight(.medium))
                            .padding(.horizontal, 12)
                            .padding(.vertical, 6)
                            .background(WeeColors.accent)
                            .foregroundStyle(.white)
                            .clipShape(Capsule())
                    }
                }
            }
            .padding(12)

            // Inline output viewer for running/completed job
            if isRunning {
                VStack(alignment: .leading, spacing: 6) {
                    if !jobOutput.isEmpty {
                        ScrollViewReader { proxy in
                            ScrollView {
                                Text(jobOutput)
                                    .font(.system(.caption, design: .monospaced))
                                    .foregroundStyle(.primary)
                                    .textSelection(.enabled)
                                    .frame(maxWidth: .infinity, alignment: .leading)
                                    .id("output-end")
                            }
                            .frame(maxHeight: 200)
                            .onChange(of: jobOutput) {
                                proxy.scrollTo("output-end", anchor: .bottom)
                            }
                        }
                        .padding(8)
                        .background(Color(.systemGray6))
                        .clipShape(RoundedRectangle(cornerRadius: 6))
                    } else {
                        HStack(spacing: 6) {
                            ProgressView()
                                .scaleEffect(0.7)
                            Text("Waiting for output…")
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
                        .padding(8)
                    }

                    // Status bar
                    HStack {
                        if jobStatus == "running" {
                            Button(action: {
                                if let jobId = runningJobId {
                                    stopJobById(jobId)
                                }
                            }) {
                                HStack(spacing: 4) {
                                    Image(systemName: "stop.fill")
                                        .font(.caption2)
                                    Text("Stop")
                                        .font(.caption2.weight(.medium))
                                }
                                .padding(.horizontal, 8)
                                .padding(.vertical, 4)
                                .background(WeeColors.error)
                                .foregroundStyle(.white)
                                .clipShape(Capsule())
                            }
                        } else if jobStatus == "completed" {
                            Label("Completed", systemImage: "checkmark.circle.fill")
                                .font(.caption2)
                                .foregroundStyle(WeeColors.success)
                        } else if jobStatus == "failed" {
                            Label("Failed", systemImage: "xmark.circle.fill")
                                .font(.caption2)
                                .foregroundStyle(WeeColors.error)
                        }
                        Spacer()
                    }
                }
                .padding(.horizontal, 12)
                .padding(.bottom, 12)
            }
        }
        .background(Color(.secondarySystemGroupedBackground))
        .clipShape(RoundedRectangle(cornerRadius: 10))
        .padding(.horizontal, 16)
    }

    // MARK: - Jobs Section

    private func jobsSection(_ jobs: [JustJob]) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack {
                Text("Recent Jobs")
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(.secondary)
                Spacer()
                Text("\(jobs.count)")
                    .font(.caption)
                    .foregroundStyle(.tertiary)
            }
            .padding(.horizontal, 16)

            ForEach(jobs.prefix(10)) { job in
                jobRow(job)
            }

            // Clear all button (only in edit mode with multiple jobs)
            if isEditingJobs && jobs.count > 1 {
                Button(action: {
                    Task {
                        await clearAllJobs()
                    }
                }) {
                    Label("Clear All Jobs", systemImage: "trash")
                        .font(.caption.weight(.medium))
                        .foregroundStyle(WeeColors.error)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 8)
                }
                .padding(.horizontal, 16)
                .padding(.top, 8)
            }
        }
    }

    private func jobRow(_ job: JustJob) -> some View {
        HStack(spacing: 10) {
            // Status icon
            Circle()
                .fill(job.status == "completed" ? WeeColors.success : WeeColors.error)
                .frame(width: 8, height: 8)

            VStack(alignment: .leading, spacing: 2) {
                Text(job.recipe)
                    .font(.subheadline.weight(.medium))
                    .foregroundStyle(.primary)
                HStack(spacing: 6) {
                    Text(job.status.capitalized)
                        .font(.caption2)
                        .foregroundStyle(job.status == "completed" ? WeeColors.success : WeeColors.error)
                    if job.exitCode != 0 {
                        Text("Exit: \(job.exitCode)")
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                    }
                    if let finishedAt = job.finishedAt, !finishedAt.isEmpty {
                        Text("·")
                            .font(.caption2)
                            .foregroundStyle(.tertiary)
                        Text(formatTimeAgo(finishedAt))
                            .font(.caption2)
                            .foregroundStyle(.tertiary)
                    }
                }
            }

            Spacer()

            if isEditingJobs {
                // Delete button in edit mode
                Button(action: {
                    Task {
                        await deleteJob(job)
                    }
                }) {
                    Image(systemName: "minus.circle.fill")
                        .font(.system(size: 22))
                        .foregroundStyle(WeeColors.error)
                }
            } else {
                // View chevron
                Image(systemName: "chevron.right")
                    .font(.caption.weight(.medium))
                    .foregroundStyle(.quaternary)
            }
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 8)
        .background(Color(.secondarySystemGroupedBackground))
        .clipShape(RoundedRectangle(cornerRadius: 8))
        .padding(.horizontal, 16)
        .contentShape(Rectangle())
        .onTapGesture {
            if !isEditingJobs {
                selectedJob = job
                showJobDetail = true
            }
        }
    }

    // MARK: - Empty & Error States

    private var emptyStateView: some View {
        VStack(spacing: 16) {
            Spacer().frame(height: 40)
            Image(systemName: "terminal")
                .font(.system(size: 40, weight: .light))
                .foregroundStyle(.quaternary)
            Text("No Justfile Found")
                .font(.title3.weight(.semibold))
                .foregroundStyle(.primary)
            Text("This project doesn't have a justfile.\nCreate one to define runnable commands.")
                .font(.subheadline)
                .foregroundStyle(.secondary)
                .multilineTextAlignment(.center)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    private func errorView(_ error: String) -> some View {
        VStack(spacing: 16) {
            Spacer().frame(height: 40)
            Image(systemName: "exclamationmark.triangle.fill")
                .font(.system(size: 36))
                .foregroundStyle(WeeColors.warning)
            Text("Unable to Load Recipes")
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

    // MARK: - Actions

    private func loadData() async {
        isLoading = true
        errorMessage = nil

        do {
            async let recipesResult = apiClient.getJustRecipes(projectID: projectID)
            async let jobsResult = apiClient.getJustJobs(projectID: projectID)

            let (loadedRecipes, hasFile) = try await recipesResult
            let loadedJobs = try await jobsResult

            await MainActor.run {
                self.recipes = loadedRecipes
                self.hasJustfile = hasFile
                self.jobs = loadedJobs
                self.isLoading = false
            }
        } catch {
            await MainActor.run {
                self.errorMessage = error.localizedDescription
                self.isLoading = false
            }
        }
    }

    private func deleteJob(_ job: JustJob) async {
        do {
            try await apiClient.deleteJustJob(projectID: projectID, jobID: job.id)
            await MainActor.run {
                jobs.removeAll { $0.id == job.id }
            }
        } catch {
            #if DEBUG
            print("Failed to delete job: \(error)")
            #endif
        }
    }

    private func clearAllJobs() async {
        let completedJobs = jobs.filter { $0.status != "running" }
        for job in completedJobs {
            await deleteJob(job)
        }
        await MainActor.run {
            isEditingJobs = false
        }
    }

    private func runRecipe(_ recipe: JustRecipe) {
        Task {
            do {
                let job = try await apiClient.runJustRecipe(projectID: projectID, recipe: recipe.name)
                self.runningRecipeName = recipe.name
                self.runningJobId = job.id
                self.jobOutput = ""
                self.jobStatus = "running"
                startStreaming(jobId: job.id)
            } catch {
                self.errorMessage = "Failed to run recipe: \(error.localizedDescription)"
                self.runningRecipeName = nil
                self.runningJobId = nil
            }
        }
    }

    private func stopJobById(_ jobId: String) {
        Task {
            do {
                _ = try await apiClient.stopJustJob(projectID: projectID, jobID: jobId)
                streamingTask?.cancel()
                runningRecipeName = nil
                runningJobId = nil
                await loadData()
            } catch {
                // Error stopping job
            }
        }
    }

    private func startStreaming(jobId: String) {
        streamingTask?.cancel()
        streamingTask = Task {
            guard let baseURL = apiClient.baseURL else { return }

            let path = "api/projects/\(projectID)/just/jobs/\(jobId)/stream"
            guard let url = URL(string: path, relativeTo: baseURL) else { return }
            var request = URLRequest(url: url)
            request.httpMethod = "GET"
            request.setValue("text/event-stream", forHTTPHeaderField: "Accept")
            request.setValue("no-cache", forHTTPHeaderField: "Cache-Control")
            if let token = apiClient.sessionToken {
                request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
            }
            request.timeoutInterval = 300 // 5 min timeout for long jobs

            do {
                let (bytes, response) = try await URLSession.shared.bytes(for: request)
                guard let httpResponse = response as? HTTPURLResponse, httpResponse.statusCode == 200 else {
                    // Fallback to polling if SSE fails
                    startPolling(jobId: jobId)
                    return
                }

                for try await line in bytes.lines {
                    if Task.isCancelled { break }

                    if line.hasPrefix("data: ") {
                        let jsonString = String(line.dropFirst(6))
                        if let jsonData = jsonString.data(using: .utf8),
                           let event = try? JSONSerialization.jsonObject(with: jsonData) as? [String: Any] {
                            let type = event["type"] as? String ?? ""

                            if type == "output", let data = event["data"] as? String {
                                await MainActor.run {
                                    self.jobOutput += data
                                }
                            } else if type == "done" {
                                let status = event["status"] as? String ?? "completed"
                                await MainActor.run {
                                    self.jobStatus = status
                                    self.runningRecipeName = nil
                                    self.runningJobId = nil
                                    Task { await loadData() }
                                }
                                break
                            }
                        }
                    } else if line.isEmpty {
                        // SSE event separator, continue
                        continue
                    }
                }
            } catch {
                // SSE failed, fallback to polling
                if !Task.isCancelled {
                    startPolling(jobId: jobId)
                }
            }
        }
    }

    private func startPolling(jobId: String) {
        streamingTask?.cancel()
        streamingTask = Task {
            while !Task.isCancelled {
                do {
                    let job = try await apiClient.getJustJob(projectID: projectID, jobID: jobId)
                    await MainActor.run {
                        self.jobOutput = job.output
                        self.jobStatus = job.status
                        if job.status != "running" {
                            self.runningRecipeName = nil
                            self.runningJobId = nil
                            Task { await loadData() }
                            return
                        }
                    }
                    try? await Task.sleep(nanoseconds: 1_000_000_000)
                } catch {
                    return
                }
            }
        }
    }

    // MARK: - Helpers

    private func formatTimeAgo(_ dateString: String) -> String {
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

// MARK: - Job Detail Sheet

struct JobDetailSheet: View {
    let job: JustJob
    let onDelete: () -> Void
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    // Job info header
                    jobInfoHeader

                    Divider()

                    // Output section
                    outputSection
                }
                .padding()
            }
            .navigationTitle(job.recipe)
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Done") {
                        dismiss()
                    }
                }

                ToolbarItem(placement: .topBarLeading) {
                    Button(action: {
                        onDelete()
                        dismiss()
                    }) {
                        Image(systemName: "trash")
                            .foregroundStyle(WeeColors.error)
                    }
                }
            }
        }
    }

    private var jobInfoHeader: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Label(
                    job.status.capitalized,
                    systemImage: job.status == "completed" ? "checkmark.circle.fill" : "xmark.circle.fill"
                )
                .font(.subheadline.weight(.medium))
                .foregroundStyle(job.status == "completed" ? WeeColors.success : WeeColors.error)

                Spacer()

                if job.exitCode != 0 {
                    Text("Exit Code: \(job.exitCode)")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                        .padding(.horizontal, 8)
                        .padding(.vertical, 4)
                        .background(Color(.systemGray5))
                        .clipShape(Capsule())
                }
            }

            if !job.startedAt.isEmpty {
                HStack(spacing: 4) {
                    Image(systemName: "calendar")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    Text("Started: \(formatDate(job.startedAt))")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }

            if let finishedAt = job.finishedAt, !finishedAt.isEmpty {
                HStack(spacing: 4) {
                    Image(systemName: "clock")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    Text("Finished: \(formatDate(finishedAt))")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }
        }
    }

    private var outputSection: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Text("Output")
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(.secondary)

                Spacer()

                // Copy button
                if !job.output.isEmpty {
                    Button(action: {
                        UIPasteboard.general.string = job.output
                    }) {
                        Image(systemName: "doc.on.doc")
                            .font(.caption)
                    }
                }
            }

            if job.output.isEmpty {
                Text("No output captured")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .frame(maxWidth: .infinity, alignment: .center)
                    .padding(.vertical, 20)
            } else {
                ScrollView {
                    Text(job.output)
                        .font(.system(.caption, design: .monospaced))
                        .foregroundStyle(.primary)
                        .textSelection(.enabled)
                        .frame(maxWidth: .infinity, alignment: .leading)
                }
                .frame(maxHeight: .infinity)
                .padding(8)
                .background(Color(.systemGray6))
                .clipShape(RoundedRectangle(cornerRadius: 8))
            }
        }
    }

    private func formatDate(_ dateString: String) -> String {
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
                let outputFormatter = DateFormatter()
                outputFormatter.dateStyle = .medium
                outputFormatter.timeStyle = .medium
                return outputFormatter.string(from: date)
            }
        }
        return dateString
    }
}

// MARK: - Preview

#Preview {
    NavigationStack {
        JustCommandsView(projectID: "example-project-id")
    }
}
