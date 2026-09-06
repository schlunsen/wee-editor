//
//  CreateSessionView.swift
//  wee
//
//  Modal for creating a new agent session
//

import SwiftUI

struct CreateSessionView: View {
    let projectId: String?
    let workingDirectory: String?
    @StateObject private var viewModel: CreateSessionViewModel
    @ObservedObject var webSocketService = WebSocketService.shared

    @Environment(\.dismiss) var dismiss
    @State private var isCreatingSession = false
    @State private var creationStatusMessage = "Creating session..."

    private let apiClient = WeeAPIClient.shared

    init(projectId: String? = nil, workingDirectory: String? = nil) {
        self.projectId = projectId
        self.workingDirectory = workingDirectory
        _viewModel = StateObject(wrappedValue: CreateSessionViewModel(
            preSelectedProjectId: projectId,
            workingDirectory: workingDirectory
        ))
    }

    var body: some View {
        ZStack {
            NavigationStack {
                Form {
                // Connection Status
                Section {
                    HStack {
                        Circle()
                            .fill(webSocketService.isConnected ? WeeColors.success : WeeColors.error)
                            .frame(width: 8, height: 8)

                        Text(webSocketService.isConnected ? "Connected" : "Not Connected")
                            .foregroundStyle(webSocketService.isConnected ? WeeColors.success : WeeColors.error)

                        Spacer()
                    }
                    .padding(.vertical, 4)
                } header: {
                    Text("Connection Status")
                }

                // Project Selection
                Section {
                    if viewModel.availableProjects.isEmpty {
                        Text("No projects available")
                            .foregroundStyle(.secondary)
                    } else {
                        Picker("Project", selection: $viewModel.selectedProject) {
                            ForEach(viewModel.availableProjects, id: \.id) { project in
                                Text(project.name)
                                    .tag(project as Project?)
                            }
                        }
                    }
                } header: {
                    Text("Project")
                } footer: {
                    Text("Select the project where the session will be created")
                }

                // Working Directory
                Section {
                    TextField("Working Directory", text: $viewModel.workingDirectory)
                        .textContentType(.URL)
                        .keyboardType(.URL)
                } header: {
                    Text("Working Directory")
                } footer: {
                    Text("The directory where the agent will work (e.g., /Users/user/projects/myapp)")
                }

                // Provider Selection
                Section {
                    if viewModel.availableProviders.isEmpty {
                        if viewModel.isLoading {
                            HStack {
                                ProgressView()
                                    .scaleEffect(0.8)
                                Text("Loading providers...")
                                    .foregroundStyle(.secondary)
                            }
                        } else {
                            Text("No providers available")
                                .foregroundStyle(.secondary)
                        }
                    } else {
                        Picker("Provider", selection: $viewModel.selectedProvider) {
                            ForEach(viewModel.availableProviders, id: \.id) { provider in
                                Text(provider.name)
                                    .tag(provider.id)
                            }
                        }
                        .onChange(of: viewModel.selectedProvider) { oldValue, newValue in
                            viewModel.updateModelsForProvider(newValue)
                        }
                    }
                } header: {
                    Text("Model Provider")
                } footer: {
                    Text("Choose an AI model provider")
                }

                // Model Selection
                Section {
                    if viewModel.availableModels.isEmpty {
                        Text("Select a provider first")
                            .foregroundStyle(.secondary)
                    } else {
                        Picker("Model", selection: $viewModel.selectedModel) {
                            ForEach(viewModel.availableModels, id: \.self) { model in
                                Text(model)
                                    .tag(model)
                            }
                        }
                    }
                } header: {
                    Text("Model")
                } footer: {
                    Text("Select the AI model to use")
                }

                // Permission Mode
                Section {
                    Picker("Permission Mode", selection: $viewModel.permissionMode) {
                        Text("Ask for Permissions").tag("default")
                        Text("Allow All").tag("allow-all")
                        Text("Read Only").tag("read-only")
                    }
                } header: {
                    Text("Permissions")
                } footer: {
                    Text("Control what permissions the agent has")
                }

                // YOLO Mode Toggle
                Section {
                    Toggle("Enable YOLO Mode", isOn: $viewModel.yoloMode)
                } header: {
                    Text("YOLO Mode")
                } footer: {
                    Text("Bypass all permission checks (use with caution)")
                }

                // Tools Selection
                Section {
                    VStack(alignment: .leading, spacing: 12) {
                        Group {
                            Toggle("Read", isOn: Binding(
                                get: { viewModel.selectedTools.contains("Read") },
                                set: { if $0 { viewModel.selectedTools.insert("Read") } else { viewModel.selectedTools.remove("Read") } }
                            ))

                            Toggle("Write", isOn: Binding(
                                get: { viewModel.selectedTools.contains("Write") },
                                set: { if $0 { viewModel.selectedTools.insert("Write") } else { viewModel.selectedTools.remove("Write") } }
                            ))

                            Toggle("Edit", isOn: Binding(
                                get: { viewModel.selectedTools.contains("Edit") },
                                set: { if $0 { viewModel.selectedTools.insert("Edit") } else { viewModel.selectedTools.remove("Edit") } }
                            ))

                            Toggle("Bash", isOn: Binding(
                                get: { viewModel.selectedTools.contains("Bash") },
                                set: { if $0 { viewModel.selectedTools.insert("Bash") } else { viewModel.selectedTools.remove("Bash") } }
                            ))
                        }
                    }
                } header: {
                    Text("Available Tools")
                } footer: {
                    Text("Select which tools the agent can use")
                }

                // System Prompt
                Section {
                    TextEditor(text: $viewModel.systemPrompt)
                        .frame(height: 100)
                } header: {
                    Text("System Prompt")
                } footer: {
                    Text("Custom instructions for the agent")
                }

                // Error Message
                if let errorMessage = viewModel.errorMessage {
                    Section {
                        Text(errorMessage)
                            .foregroundStyle(WeeColors.error)
                    }
                }

                // Success Message
                if let successMessage = viewModel.successMessage {
                    Section {
                        Text(successMessage)
                            .foregroundStyle(WeeColors.success)
                    }
                }
            }
            .navigationTitle("Create New Session")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button("Cancel") {
                        dismiss()
                    }
                }

                ToolbarItem(placement: .topBarTrailing) {
                    Button(action: createSession) {
                        if viewModel.isSavingSession {
                            ProgressView()
                                .scaleEffect(0.8)
                        } else {
                            Text("Create")
                                .fontWeight(.semibold)
                        }
                    }
                    .disabled(!viewModel.isFormValid || viewModel.isSavingSession)
                }
            }
        }
        .task {
            // Ensure WebSocket is connected
            if !webSocketService.isConnected {
                #if DEBUG
                print("🔌 Connecting WebSocket for session creation...")
                #endif
                await webSocketService.connect()
                try? await Task.sleep(nanoseconds: 1_000_000_000)  // 1 second for connection to establish
            }

            await viewModel.loadProjects()
            await viewModel.loadProviders()
            await viewModel.selectRandomAvatar()
        }

        // Loading overlay
        if isCreatingSession {
            ZStack {
                Color.black.opacity(0.4)
                    .ignoresSafeArea()

                VStack(spacing: 20) {
                    ProgressView()
                        .scaleEffect(1.5)
                        .tint(.white)

                    Text(creationStatusMessage)
                        .font(.headline)
                        .foregroundStyle(.white)
                        .multilineTextAlignment(.center)
                        .padding(.horizontal, 40)
                }
                .padding(40)
                .background(Color(.systemGray))
                .cornerRadius(16)
                .shadow(radius: 20)
            }
            .transition(.opacity)
        }
    }
    }

    private func createSession() {
        Task {
            // Show loading overlay
            await MainActor.run {
                isCreatingSession = true
                creationStatusMessage = "Creating session..."
            }

            let success = await viewModel.createSession()
            if success {
                // Update status message
                await MainActor.run {
                    creationStatusMessage = "Waiting for backend..."
                }

                // Wait for session to be created on backend
                #if DEBUG
                print("⏳ Waiting 1.5 seconds for session to be created...")
                #endif
                try? await Task.sleep(nanoseconds: 1_500_000_000)  // 1.5 seconds

                // Fetch sessions to find the newly created one
                guard let projectId = viewModel.selectedProject?.id else {
                    #if DEBUG
                    print("❌ No project ID found")
                    #endif
                    await MainActor.run {
                        isCreatingSession = false
                    }
                    dismiss()
                    return
                }

                await MainActor.run {
                    creationStatusMessage = "Finding your session..."
                }

                #if DEBUG
                print("🔍 Fetching sessions for project: \(projectId)")
                #endif
                do {
                    let sessions = try await apiClient.getProjectSessions(projectID: projectId)
                    #if DEBUG
                    print("📋 Found \(sessions.count) sessions")
                    #endif

                    // Sort by created_at descending and find the newest
                    let sortedSessions = sessions.sorted { a, b in
                        // Compare as strings in ISO format (they should sort correctly)
                        return a.created_at > b.created_at
                    }

                    if let newestSession = sortedSessions.first {
                        #if DEBUG
                        print("✅ Found newest session: \(newestSession.id)")
                        #endif
                        #if DEBUG
                        print("📝 Session title: \(newestSession.model_name ?? "Unknown")")
                        #endif

                        await MainActor.run {
                            creationStatusMessage = "Opening chat..."
                        }

                        // Post notifications:
                        // 1. SessionCreated - triggers session list refresh in ProjectDetailView
                        // 2. NavigateToSession - triggers navigation to ChatView
                        #if DEBUG
                        print("📢 Posting SessionCreated notification for project: \(projectId)")
                        #endif
                        NotificationCenter.default.post(
                            name: NSNotification.Name("SessionCreated"),
                            object: nil,
                            userInfo: ["projectId": projectId]
                        )

                        #if DEBUG
                        print("📢 Posting NavigateToSession notification")
                        #endif
                        NotificationCenter.default.post(
                            name: NSNotification.Name("NavigateToSession"),
                            object: nil,
                            userInfo: [
                                "sessionId": newestSession.id,
                                "sessionTitle": newestSession.model_name ?? "Unknown"
                            ]
                        )

                        // Hide loading overlay
                        await MainActor.run {
                            isCreatingSession = false
                        }

                        // Dismiss the modal
                        #if DEBUG
                        print("🚪 Dismissing create session modal...")
                        #endif
                        dismiss()
                    } else {
                        #if DEBUG
                        print("❌ No sessions found in list")
                        #endif
                        await MainActor.run {
                            isCreatingSession = false
                        }
                        dismiss()
                    }
                } catch {
                    #if DEBUG
                    print("❌ Failed to fetch sessions: \(error)")
                    #endif
                    await MainActor.run {
                        isCreatingSession = false
                    }
                    dismiss()
                }
            } else {
                #if DEBUG
                print("❌ Session creation failed")
                #endif
                await MainActor.run {
                    isCreatingSession = false
                }
            }
        }
    }
}

#Preview {
    CreateSessionView()
}
