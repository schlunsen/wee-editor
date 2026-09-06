//
//  ProvidersView.swift
//  wee
//
//  Provider configuration management view
//

import SwiftUI

struct ProvidersView: View {
    @StateObject private var viewModel = ProvidersViewModel()
    @State private var showConfigSheet = false

    var body: some View {
        List {
            // Status messages
            if let error = viewModel.errorMessage {
                Section {
                    Label(error, systemImage: "exclamationmark.triangle.fill")
                        .font(.subheadline)
                        .foregroundStyle(WeeColors.error)
                }
                .listRowBackground(WeeColors.error.opacity(0.1))
            }

            if let success = viewModel.successMessage {
                Section {
                    Label(success, systemImage: "checkmark.circle.fill")
                        .font(.subheadline)
                        .foregroundStyle(WeeColors.success)
                }
                .listRowBackground(WeeColors.success.opacity(0.1))
            }

            // Provider list
            Section {
                if viewModel.isLoading {
                    HStack {
                        Spacer()
                        ProgressView()
                            .padding()
                        Spacer()
                    }
                } else if viewModel.providers.isEmpty {
                    ContentUnavailableView(
                        "No Providers",
                        systemImage: "cpu",
                        description: Text("Could not load providers from server.")
                    )
                } else {
                    ForEach(viewModel.providers) { provider in
                        ProviderRow(provider: provider) {
                            viewModel.startEditing(provider)
                            showConfigSheet = true
                        } onSetDefault: {
                            Task { await viewModel.setDefault(provider) }
                        } onDelete: {
                            Task { await viewModel.deleteProviderConfig(provider) }
                        }
                    }
                }
            } header: {
                Text("AI Providers")
            } footer: {
                Text("Configure API keys and models for each provider. The default provider is used for new agent sessions.")
                    .font(.caption)
                    .foregroundStyle(WeeColors.textTertiary)
            }
        }
        .navigationTitle("Providers")
        .refreshable {
            await viewModel.loadProviders()
        }
        .task {
            await viewModel.loadProviders()
        }
        .sheet(isPresented: $showConfigSheet) {
            if let provider = viewModel.editingProvider {
                ProviderConfigSheet(viewModel: viewModel, provider: provider)
            }
        }
    }
}

// MARK: - Provider Row

private struct ProviderRow: View {
    let provider: ProviderInfo
    let onConfigure: () -> Void
    let onSetDefault: () -> Void
    let onDelete: () -> Void

    @State private var showDeleteConfirmation = false

    var body: some View {
        VStack(alignment: .leading, spacing: WeeSpacing.sm) {
            // Header: icon, name, status badge
            HStack(spacing: WeeSpacing.sm) {
                Text(provider.icon)
                    .font(.title2)

                VStack(alignment: .leading, spacing: 2) {
                    Text(provider.name)
                        .font(.subheadline.weight(.semibold))

                    if let description = provider.description {
                        Text(description)
                            .font(.caption)
                            .foregroundStyle(WeeColors.textSecondary)
                            .lineLimit(1)
                    }
                }

                Spacer()

                // Status badge
                Text(provider.statusLabel)
                    .font(WeeTypography.badge)
                    .padding(.horizontal, 8)
                    .padding(.vertical, 3)
                    .background(provider.statusColor.opacity(0.15))
                    .foregroundStyle(provider.statusColor)
                    .clipShape(Capsule())
            }

            // Details
            VStack(alignment: .leading, spacing: 4) {
                if let baseURL = provider.baseURL, !baseURL.isEmpty {
                    DetailRow(label: "Base URL", value: baseURL)
                }

                if let model = provider.configuredModel, !model.isEmpty {
                    DetailRow(label: "Model", value: model)
                } else if let defaultModel = provider.defaultModel, !defaultModel.isEmpty {
                    DetailRow(label: "Default Model", value: defaultModel)
                }

                if provider.isConfigured, provider.apiKey != nil {
                    DetailRow(label: "API Key", value: "Configured")
                }
            }

            // Actions
            HStack(spacing: WeeSpacing.sm) {
                Button(action: onConfigure) {
                    Label(provider.isConfigured ? "Edit" : "Configure", systemImage: provider.isConfigured ? "pencil" : "key.fill")
                        .font(.caption.weight(.medium))
                }
                .buttonStyle(.bordered)
                .tint(WeeColors.accent)

                if provider.isConfigured && !provider.isCurrent {
                    Button(action: onSetDefault) {
                        Label("Set Default", systemImage: "star")
                            .font(.caption.weight(.medium))
                    }
                    .buttonStyle(.bordered)
                    .tint(WeeColors.secondary)
                }

                if provider.isConfigured {
                    Spacer()

                    Button(role: .destructive) {
                        showDeleteConfirmation = true
                    } label: {
                        Image(systemName: "trash")
                            .font(.caption)
                    }
                    .buttonStyle(.bordered)
                    .tint(WeeColors.error)
                    .confirmationDialog(
                        "Remove \(provider.name) configuration?",
                        isPresented: $showDeleteConfirmation,
                        titleVisibility: .visible
                    ) {
                        Button("Remove", role: .destructive) {
                            onDelete()
                        }
                        Button("Cancel", role: .cancel) {}
                    } message: {
                        Text("This will remove the API key and configuration for this provider.")
                    }
                }
            }
            .padding(.top, 4)
        }
        .padding(.vertical, WeeSpacing.xs)
    }
}

// MARK: - Detail Row

private struct DetailRow: View {
    let label: String
    let value: String

    var body: some View {
        HStack(spacing: 6) {
            Text(label)
                .font(.caption)
                .foregroundStyle(WeeColors.textTertiary)
            Text(value)
                .font(.caption)
                .foregroundStyle(WeeColors.textSecondary)
                .lineLimit(1)
        }
    }
}

// MARK: - Provider Config Sheet

private struct ProviderConfigSheet: View {
    @ObservedObject var viewModel: ProvidersViewModel
    let provider: ProviderInfo
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            Form {
                // Provider info header
                Section {
                    HStack(spacing: WeeSpacing.md) {
                        Text(provider.icon)
                            .font(.largeTitle)

                        VStack(alignment: .leading, spacing: 2) {
                            Text(provider.name)
                                .font(.headline)
                            if let description = provider.description {
                                Text(description)
                                    .font(.caption)
                                    .foregroundStyle(WeeColors.textSecondary)
                            }
                        }
                    }
                    .padding(.vertical, 4)
                }

                // API Key
                Section {
                    SecureField("API Key", text: $viewModel.editAPIKey)
                        .textContentType(.password)
                        .autocorrectionDisabled()
                        .textInputAutocapitalization(.never)
                } header: {
                    Text("API Key")
                } footer: {
                    if provider.id == "claude" {
                        Text("Optional for Claude. Falls back to ANTHROPIC_API_KEY environment variable.")
                            .font(.caption)
                    } else {
                        Text("Required. Your API key for \(provider.name).")
                            .font(.caption)
                    }
                }

                // Custom URL (only for custom provider)
                if provider.id == "custom" {
                    Section {
                        TextField("https://api.example.com/v1", text: $viewModel.editCustomURL)
                            .keyboardType(.URL)
                            .autocorrectionDisabled()
                            .textInputAutocapitalization(.never)
                    } header: {
                        Text("Custom Base URL")
                    } footer: {
                        Text("Required. The API endpoint URL for your custom provider.")
                            .font(.caption)
                    }
                } else if let baseURL = provider.baseURL, !baseURL.isEmpty {
                    // Show read-only base URL for preset providers
                    Section {
                        HStack {
                            Text(baseURL)
                                .font(.subheadline)
                                .foregroundStyle(WeeColors.textSecondary)
                            Spacer()
                            Image(systemName: "lock.fill")
                                .font(.caption)
                                .foregroundStyle(WeeColors.textTertiary)
                        }
                    } header: {
                        Text("Base URL")
                    }
                }

                // Model selection
                Section {
                    if !provider.models.isEmpty {
                        Picker("Model", selection: $viewModel.editModelName) {
                            Text("Select a model").tag("")
                            ForEach(provider.models, id: \.self) { model in
                                Text(model)
                                    .tag(model)
                            }
                        }
                    } else {
                        TextField("Model name", text: $viewModel.editModelName)
                            .autocorrectionDisabled()
                            .textInputAutocapitalization(.never)
                    }
                } header: {
                    Text("Model")
                } footer: {
                    if let defaultModel = provider.defaultModel, !defaultModel.isEmpty {
                        Text("Default: \(defaultModel)")
                            .font(.caption)
                    }
                }
            }
            .navigationTitle("Configure \(provider.name)")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") {
                        dismiss()
                    }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Save") {
                        Task {
                            await viewModel.saveProvider()
                            if viewModel.errorMessage == nil {
                                dismiss()
                            }
                        }
                    }
                    .disabled(viewModel.isSaving || !isFormValid)
                    .overlay {
                        if viewModel.isSaving {
                            ProgressView()
                        }
                    }
                }
            }
        }
        .presentationDetents([.medium, .large])
    }

    private var isFormValid: Bool {
        // Claude doesn't require API key
        if provider.id == "claude" { return true }
        // Custom requires both API key and URL
        if provider.id == "custom" {
            return !viewModel.editAPIKey.trimmingCharacters(in: .whitespaces).isEmpty &&
                   !viewModel.editCustomURL.trimmingCharacters(in: .whitespaces).isEmpty
        }
        // Others just need API key
        return !viewModel.editAPIKey.trimmingCharacters(in: .whitespaces).isEmpty
    }
}
