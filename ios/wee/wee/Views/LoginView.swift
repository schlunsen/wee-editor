//
//  LoginView.swift
//  wee
//
//  Modern login screen with self-signed certificate support
//

import SwiftUI

struct LoginView: View {
    @EnvironmentObject var viewModel: AuthenticationViewModel
    @State private var serverURL = LocalConfig.defaultServerURL
    @State private var username = ""
    @State private var password = ""
    @State private var showServerError = false
    @State private var selectedHostId: String? = nil
    @State private var showNewHostForm = false
    @State private var formAppeared = false

    var body: some View {
        NavigationStack {
            ZStack {
                // Background gradient
                WeeColors.background
                    .ignoresSafeArea()

                VStack(spacing: 0) {
                    // Header
                    headerView
                        .padding(.vertical, 32)

                    // Form
                    ScrollView {
                        VStack(spacing: 24) {
                            // Saved Hosts Section
                            if !viewModel.savedHosts.isEmpty {
                                savedHostsSection
                            }

                            // Show either saved host login OR new host form
                            if !showNewHostForm && selectedHostId != nil {
                                savedHostLoginForm
                            } else {
                                newHostForm
                            }

                            // Error Message
                            if let error = viewModel.errorMessage {
                                errorBanner(error)
                            }

                            // Login Button
                            loginButton
                        }
                        .padding(.horizontal, 24)
                        .padding(.vertical, 24)
                    }

                    Spacer()

                    // Footer
                    footerView
                        .padding(.vertical, 16)
                        .padding(.horizontal, 24)
                }
                .opacity(formAppeared ? 1 : 0)
                .offset(y: formAppeared ? 0 : 20)
            }
        }
        .onAppear {
            withAnimation(.spring(response: 0.6, dampingFraction: 0.8).delay(0.1)) {
                formAppeared = true
            }
        }
    }

    // MARK: - New Subviews

    private var savedHostsSection: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text("Saved Hosts")
                .font(.subheadline.bold())
                .foregroundStyle(.secondary)

            VStack(spacing: 8) {
                ForEach(viewModel.savedHosts, id: \.id) { host in
                    hostSelectionButton(host)
                        .swipeActions(edge: .trailing) {
                            Button(role: .destructive) {
                                Task {
                                    try? await viewModel.deleteHost(hostId: host.id)
                                }
                            } label: {
                                Label("Delete", systemImage: "trash.fill")
                            }
                        }
                }
            }

            HStack(spacing: 8) {
                Button(action: {
                    showNewHostForm = true
                    selectedHostId = nil
                    resetForm()
                }) {
                    HStack {
                        Image(systemName: "plus.circle.fill")
                        Text("Add New Host")
                    }
                    .font(.subheadline.bold())
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 10)
                    .background(Color(.systemGray6))
                    .foregroundStyle(WeeColors.accent)
                    .cornerRadius(8)
                }
            }
            .padding(.top, 4)
        }
    }

    private func hostSelectionButton(_ host: Host) -> some View {
        Button(action: {
            selectedHostId = host.id
            username = host.username ?? ""
            password = ""
            showNewHostForm = false
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

                if selectedHostId == host.id {
                    Image(systemName: "checkmark.circle.fill")
                        .foregroundStyle(WeeColors.accent)
                }
            }
            .padding(12)
            .background(selectedHostId == host.id ? WeeColors.accentLight : WeeColors.surface)
            .cornerRadius(8)
            .overlay(
                RoundedRectangle(cornerRadius: 8)
                    .stroke(selectedHostId == host.id ? WeeColors.accent : Color(.systemGray3), lineWidth: 1)
            )
        }
    }

    private var savedHostLoginForm: some View {
        VStack(spacing: 24) {
            // Username Input (pre-filled)
            formField(
                label: "Username",
                placeholder: "Enter your username",
                text: $username,
                isSecure: false,
                helpText: "Your Wee account username"
            )

            // Password Input
            formField(
                label: "Password",
                placeholder: "Enter your password",
                text: $password,
                isSecure: true,
                helpText: "Your Wee account password"
            )

            Button(action: {
                showNewHostForm = true
                resetForm()
            }) {
                Text("Use Different Host")
                    .font(.caption.bold())
                    .foregroundStyle(WeeColors.accent)
            }
        }
    }

    private var newHostForm: some View {
        VStack(spacing: 24) {
            // Server URL Input
            formField(
                label: "Server URL",
                placeholder: "localhost:3333",
                text: $serverURL,
                isSecure: false,
                helpText: "The address of your Wee analytics server"
            )

            // Username Input
            formField(
                label: "Username",
                placeholder: "Enter your username",
                text: $username,
                isSecure: false,
                helpText: "Your Wee account username"
            )

            // Password Input
            formField(
                label: "Password",
                placeholder: "Enter your password",
                text: $password,
                isSecure: true,
                helpText: "Your Wee account password"
            )
        }
    }

    // MARK: - Subviews

    private var headerView: some View {
        VStack(spacing: 12) {
            Image("HackerCat")
                .resizable()
                .aspectRatio(contentMode: .fill)
                .frame(width: 100, height: 100)
                .clipShape(RoundedRectangle(cornerRadius: 24))
                .shadow(color: WeeColors.accent.opacity(0.5), radius: 16, y: 6)

            Text("Wee")
                .font(.largeTitle.bold())
                .foregroundStyle(WeeColors.textPrimary)

            Text("Analytics & Agent Dashboard")
                .font(.subheadline)
                .foregroundStyle(WeeColors.textSecondary)
        }
        .multilineTextAlignment(.center)
    }

    private var loginButton: some View {
        Button(action: {
            WeeHaptics.tap()
            Task {
                if let hostId = selectedHostId {
                    // Login to saved host
                    await viewModel.login(username: username, password: password, hostId: hostId)
                } else {
                    // Login to new host
                    do {
                        let hostName = extractHostName(from: serverURL)

                        // Add the host first
                        let host = try await viewModel.addNewHost(
                            name: hostName,
                            url: serverURL,
                            username: username
                        )

                        // Configure API client
                        _ = try viewModel.configureServer(url: serverURL)

                        // Then attempt login
                        await viewModel.login(username: username, password: password, hostId: host.id)
                    } catch {
                        viewModel.errorMessage = "Configuration failed: \(error.localizedDescription)"
                    }
                }
            }
        }) {
            if viewModel.isLoading {
                ProgressView()
                    .tint(.white)
            } else {
                Text("Login")
                    .font(.headline)
                    .fontWeight(.semibold)
            }
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 14)
        .background(WeeGradients.loginButton)
        .foregroundStyle(.white)
        .cornerRadius(10)
        .shadow(color: WeeColors.accent.opacity(0.3), radius: 8, y: 4)
        .disabled(viewModel.isLoading || (selectedHostId == nil && (serverURL.isEmpty || username.isEmpty)) || password.isEmpty)
        .weePrimaryStyle()
    }

    private func extractHostName(from url: String) -> String {
        // Extract hostname from URL like "localhost:3333" or "https://my-server.com"
        var cleaned = url
            .replacingOccurrences(of: "https://", with: "")
            .replacingOccurrences(of: "http://", with: "")

        if let colonIndex = cleaned.firstIndex(of: ":") {
            cleaned = String(cleaned[..<colonIndex])
        }

        // Capitalize first letter
        return cleaned.prefix(1).uppercased() + cleaned.dropFirst()
    }

    private func resetForm() {
        serverURL = LocalConfig.defaultServerURL
        username = ""
        password = ""
        showNewHostForm = false
        selectedHostId = nil
    }

    private var footerView: some View {
        VStack(spacing: 8) {
            Text("Secure Login")
                .font(.caption)
                .foregroundStyle(.secondary)

            HStack(spacing: 12) {
                securityBadge("🔐 Self-Signed", "Supports self-signed")
                securityBadge("🔑 Keychain", "Secure storage")
            }
            .font(.caption2)
            .foregroundStyle(.secondary)
        }
        .multilineTextAlignment(.center)
    }

    // MARK: - Helper Views

    private func formField(
        label: String,
        placeholder: String,
        text: Binding<String>,
        isSecure: Bool,
        helpText: String
    ) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Label(label, systemImage: isSecure ? "key.fill" : "globe")
                .font(.subheadline.bold())
                .foregroundStyle(.primary)

            if isSecure {
                SecureField(placeholder, text: text)
                    .textContentType(.password)
                    .textInputAutocapitalization(.never)
            } else {
                TextField(placeholder, text: text)
                    .textContentType(.URL)
                    .keyboardType(.URL)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
            }

            Text(helpText)
                .font(.caption)
                .foregroundStyle(.secondary)
        }
        .padding(14)
        .background(WeeColors.surface)
        .cornerRadius(10)
        .overlay(
            RoundedRectangle(cornerRadius: 10)
                .stroke(Color(.systemGray4), lineWidth: 0.5)
        )
    }

    private func errorBanner(_ message: String) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack(spacing: 12) {
                Image(systemName: "exclamationmark.circle.fill")
                    .foregroundStyle(.red)
                Text(message)
                    .font(.subheadline)
                    .foregroundStyle(.red)
                Spacer()
            }
        }
        .padding(12)
        .background(Color(.systemRed).opacity(0.1))
        .cornerRadius(8)
    }

    private func securityBadge(_ title: String, _ description: String) -> some View {
        VStack(spacing: 2) {
            Text(title)
            Text(description)
                .font(.caption2)
        }
        .padding(8)
        .frame(maxWidth: .infinity)
        .background(Color(.systemGray6))
        .cornerRadius(6)
    }

    private func stepView(number: String, title: String, command: String? = nil, description: String? = nil) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(spacing: 12) {
                Circle()
                    .fill(WeeColors.accent)
                    .frame(width: 24, height: 24)
                    .overlay(
                        Text(number)
                            .font(.caption.bold())
                            .foregroundStyle(.white)
                    )

                Text(title)
                    .font(.subheadline.bold())

                Spacer()
            }

            if let command = command {
                codeBlock(command)
            }

            if let description = description {
                Text(description)
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .padding(.leading, 36)
            }
        }
    }

    private func codeBlock(_ code: String) -> some View {
        HStack(spacing: 12) {
            Text(code)
                .font(.system(.caption, design: .monospaced))
                .foregroundStyle(WeeColors.accent)

            Spacer()

            Button(action: {
                UIPasteboard.general.string = code
            }) {
                Image(systemName: "doc.on.doc")
                    .font(.caption)
                    .foregroundStyle(WeeColors.accent)
            }
        }
        .padding(12)
        .background(Color(.systemGray6))
        .cornerRadius(8)
    }
}

#Preview {
    LoginView()
        .environmentObject(AuthenticationViewModel())
}
