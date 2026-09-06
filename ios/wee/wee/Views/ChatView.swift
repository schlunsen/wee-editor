//
//  ChatView.swift
//  wee
//
//  Chat view for displaying agent session messages
//

import SwiftUI

struct ChatView: View {
    let sessionID: String
    let sessionTitle: String

    @StateObject private var viewModel: ChatViewModel
    @ObservedObject private var webSocketViewModel: SessionsWebSocketViewModel
    @ObservedObject private var avatarManager = AvatarManager.shared
    @FocusState private var isInputFocused: Bool

    // Convenience initializer
    init(sessionID: String, sessionTitle: String, viewModel: SessionsWebSocketViewModel? = nil) {
        self.sessionID = sessionID
        self.sessionTitle = sessionTitle
        let wsViewModel = viewModel ?? SessionsWebSocketViewModel.shared
        self._viewModel = StateObject(wrappedValue: ChatViewModel(sessionID: sessionID, sessionTitle: sessionTitle, webSocketViewModel: wsViewModel))
        self.webSocketViewModel = wsViewModel
    }

    var body: some View {
        ZStack {
            chatContent
            if viewModel.showImageViewer, let dataUrl = viewModel.selectedImageDataUrl {
                ImageViewerOverlay(dataUrl: dataUrl, onClose: viewModel.dismissImageViewer)
            }
        }
        .task { await viewModel.initialize() }
        .onReceive(webSocketViewModel.$sessionMessages) { viewModel.syncMessages(from: $0) }
        .onChange(of: webSocketViewModel.sessionStatus) { _, newStatus in
            if let newSessionStatus = newStatus[sessionID] {
                viewModel.updateStatus(newSessionStatus)
            }
        }
        .onChange(of: viewModel.selectedMessageId) { _, newValue in
            if newValue != nil { viewModel.showMessageDetail = true }
        }
        .onChange(of: viewModel.showMessageDetail) { _, newValue in
            if !newValue { viewModel.selectedMessageId = nil }
        }
    }

    // MARK: - Main Content

    private var chatContent: some View {
        VStack(spacing: 0) {
            messagesArea
            Spacer()
            inputArea
        }
        .navigationTitle(sessionTitle)
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .principal) {
                StatusHeaderView(viewModel: viewModel)
            }
            ToolbarItemGroup(placement: .topBarTrailing) {
                ToolbarButtonsView(viewModel: viewModel)
            }
        }
        .sheet(isPresented: $viewModel.showMessageDetail) {
            if let message = viewModel.selectedMessage {
                MessageDetailView(message: message)
                    .id(message.id)
                    .presentationDetents([.medium, .large])
                    .presentationDragIndicator(.visible)
            }
        }
        .navigationDestination(isPresented: $viewModel.navigateToJust) {
            if let pid = viewModel.projectId {
                JustCommandsView(projectID: pid)
            }
        }
        .navigationDestination(isPresented: $viewModel.navigateToGit) {
            GitStatusView(sessionID: sessionID)
        }
        .navigationDestination(isPresented: $viewModel.navigateToStats) {
            SessionStatsView(sessionID: sessionID)
        }
        .navigationDestination(isPresented: $viewModel.navigateToZen) {
            ZenView(sessionID: sessionID, sessionTitle: sessionTitle, viewModel: webSocketViewModel)
        }
    }

    // MARK: - Messages Area

    @ViewBuilder
    private var messagesArea: some View {
        if viewModel.isLoading {
            LoadingMessagesView()
        } else if let error = webSocketViewModel.errorMessage {
            ErrorMessagesView(error: error)
        } else if viewModel.messages.isEmpty {
            EmptyMessagesView()
        } else {
            MessagesScrollView(viewModel: viewModel, isInputFocused: $isInputFocused)
        }
    }

    // MARK: - Input Area

    private var inputArea: some View {
        VStack(spacing: 0) {
            // Subtle top shadow instead of hard divider
            Rectangle()
                .fill(
                    LinearGradient(
                        colors: [Color.black.opacity(0.06), Color.clear],
                        startPoint: .top,
                        endPoint: .bottom
                    )
                )
                .frame(height: 8)
                .allowsHitTesting(false)

            HStack(alignment: .bottom, spacing: 10) {
                MessageTextField(text: $viewModel.userInput, isFocused: $isInputFocused, onSubmit: sendMessage)
                SendButton(isEnabled: !viewModel.userInput.isEmpty, action: sendMessage)
            }
            .padding(.horizontal, 14)
            .padding(.top, 8)
            .padding(.bottom, 12)
            .background(.ultraThinMaterial)
        }
    }

    // MARK: - Actions

    private func sendMessage() {
        isInputFocused = false
        viewModel.sendMessage()
    }
}

// MARK: - Status Header View

struct StatusHeaderView: View {
    @ObservedObject var viewModel: ChatViewModel

    var body: some View {
        VStack(spacing: 4) {
            HStack(spacing: 8) {
                AvatarView(
                    image: viewModel.avatarImage,
                    name: viewModel.characterName,
                    color: viewModel.characterColor
                )
                .task {
                    await viewModel.loadAvatarImageIfNeeded()
                }

                VStack(alignment: .leading, spacing: 2) {
                    Text(viewModel.characterName)
                        .font(.caption.bold())
                        .foregroundStyle(.secondary)
                    Text(viewModel.sessionTitle)
                        .font(.headline)
                }

                Spacer()
            }

            StatusIndicatorView(status: viewModel.currentSessionStatus)
        }
    }
}

struct AvatarView: View {
    let image: UIImage?
    let name: String
    let color: Color
    var size: CGFloat = 32

    var body: some View {
        ZStack {
            Circle()
                .fill(color)
                .frame(width: size, height: size)

            if let image = image {
                Image(uiImage: image)
                    .resizable()
                    .scaledToFill()
                    .frame(width: size, height: size)
                    .clipShape(Circle())
            } else {
                Text(name.initials)
                    .foregroundStyle(.white)
                    .font(.system(size: size * 0.375, weight: .semibold))
            }
        }
    }
}

struct StatusIndicatorView: View {
    let status: SessionStatus
    @State private var pulse = false

    var body: some View {
        HStack(spacing: 5) {
            ZStack {
                if status.isActive {
                    Circle()
                        .fill(status.color.opacity(0.3))
                        .frame(width: 10, height: 10)
                        .scaleEffect(pulse ? 1.4 : 1.0)
                        .opacity(pulse ? 0.0 : 0.6)
                        .animation(.easeInOut(duration: 1.2).repeatForever(autoreverses: false), value: pulse)
                }
                Circle()
                    .fill(status.color)
                    .frame(width: 6, height: 6)
            }
            Text(status.label)
                .font(.caption2.weight(.medium))
                .foregroundStyle(.secondary)
        }
        .onAppear {
            if status.isActive { pulse = true }
        }
        .onChange(of: status.isActive) { _, active in
            pulse = active
        }
    }
}

// MARK: - Toolbar Buttons

struct ToolbarButtonsView: View {
    @ObservedObject var viewModel: ChatViewModel

    var body: some View {
        Menu {
            Button(action: { viewModel.navigate(to: .just) }) {
                Label("Just Commands", systemImage: "terminal.fill")
            }
            .disabled(viewModel.projectId == nil)

            Button(action: { viewModel.navigate(to: .git) }) {
                Label("Git", systemImage: "branch")
            }

            Divider()

            Button(action: { viewModel.navigate(to: .stats) }) {
                Label("Session Stats", systemImage: "chart.bar.fill")
            }

            Button(action: { viewModel.navigate(to: .zen) }) {
                Label("Zen View", systemImage: "eye")
            }
        } label: {
            Image(systemName: "ellipsis.circle")
                .font(.system(size: 16, weight: .semibold))
        }

        Button(action: { viewModel.handleInterrupt() }) {
            Image(systemName: "stop.circle.fill")
                .font(.system(size: 16, weight: .semibold))
                .foregroundStyle(.red)
        }

        Button(action: { Task { await viewModel.refreshMessages() } }) {
            Image(systemName: "arrow.clockwise")
                .font(.system(size: 14, weight: .semibold))
        }
    }
}

// MARK: - Messages Scroll View

struct MessagesScrollView: View {
    @ObservedObject var viewModel: ChatViewModel
    @FocusState.Binding var isInputFocused: Bool

    var body: some View {
        ScrollViewReader { scrollProxy in
            ScrollView {
                LazyVStack(alignment: .leading, spacing: 6) {
                    ForEach(visibleMessages, id: \.id) { message in
                        MessageBubbleView(
                            message: message,
                            isExpanded: viewModel.isMessageExpanded(message.id),
                            avatarImage: viewModel.avatarImage,
                            avatarName: viewModel.characterName,
                            avatarColor: viewModel.characterColor,
                            onToggleExpand: { viewModel.toggleMessageExpansion(message.id) },
                            onTap: { viewModel.selectMessage(message.id) },
                            onImageTap: { viewModel.showImage($0) }
                        )
                        .id(message.id)
                        .transition(.asymmetric(
                            insertion: .opacity.combined(with: .move(edge: .bottom)),
                            removal: .opacity
                        ))
                    }

                    if viewModel.currentSessionStatus.isActive {
                        ProcessingIndicator()
                    }
                }
                .padding(.horizontal, 12)
                .padding(.vertical, 12)
                .frame(maxWidth: .infinity, alignment: .leading)
            }
            .background(
                LinearGradient(
                    colors: [
                        Color(.systemGroupedBackground),
                        Color(.systemBackground)
                    ],
                    startPoint: .top,
                    endPoint: .bottom
                )
            )
            .onTapGesture { isInputFocused = false }
            .onAppear {
                scrollToLastMessage(using: scrollProxy, animated: false)
            }
            .onChange(of: viewModel.messages.count) { oldCount, _ in
                let delay: UInt64 = oldCount == 0 ? 100_000_000 : 0
                Task {
                    try? await Task.sleep(nanoseconds: delay)
                    scrollToLastMessage(using: scrollProxy, animated: true)
                }
            }
        }
    }

    private var visibleMessages: [AgentMessage] {
        viewModel.messages.filter { !viewModel.shouldHideMessage($0) }
    }

    private func scrollToLastMessage(using proxy: ScrollViewProxy, animated: Bool) {
        if let lastId = visibleMessages.last?.id {
            proxy.scrollTo(lastId, anchor: .bottom)
        }
    }
}

struct ProcessingIndicator: View {
    @State private var animate = false

    var body: some View {
        HStack(spacing: 8) {
            HStack(spacing: 4) {
                ForEach(0..<3) { index in
                    Circle()
                        .fill(Color.indigo.opacity(0.6))
                        .frame(width: 6, height: 6)
                        .offset(y: animate ? -4 : 2)
                        .animation(
                            .easeInOut(duration: 0.5)
                            .repeatForever(autoreverses: true)
                            .delay(Double(index) * 0.15),
                            value: animate
                        )
                }
            }
            Text("Processing…")
                .font(.caption.weight(.medium))
                .foregroundStyle(.secondary)
        }
        .padding(.vertical, 10)
        .padding(.horizontal, 14)
        .background(Color(.systemGray6).opacity(0.6))
        .clipShape(Capsule())
        .id("processing-spinner")
        .onAppear { animate = true }
    }
}

// MARK: - Input Components

struct MessageTextField: View {
    @Binding var text: String
    var isFocused: FocusState<Bool>.Binding
    let onSubmit: () -> Void

    var body: some View {
        TextField("Message...", text: $text, axis: .vertical)
            .textFieldStyle(.plain)
            .lineLimit(1...4)
            .padding(.horizontal, 16)
            .padding(.vertical, 10)
            .font(.subheadline)
            .focused(isFocused)
            .background(
                RoundedRectangle(cornerRadius: 22, style: .continuous)
                    .fill(Color(.systemGray6).opacity(0.8))
            )
            .overlay(
                RoundedRectangle(cornerRadius: 22, style: .continuous)
                    .strokeBorder(Color(.systemGray4).opacity(0.3), lineWidth: 0.5)
            )
            .onSubmit {
                if !text.trimmingCharacters(in: .whitespaces).isEmpty {
                    onSubmit()
                }
            }
    }
}

struct SendButton: View {
    let isEnabled: Bool
    let action: () -> Void

    var body: some View {
        Button(action: {
            if isEnabled {
                WeeHaptics.tap()
                action()
            }
        }) {
            ZStack {
                Circle()
                    .fill(
                        isEnabled
                        ? WeeGradients.sendButton
                        : LinearGradient(
                            colors: [Color(.systemGray4), Color(.systemGray4)],
                            startPoint: .topLeading,
                            endPoint: .bottomTrailing
                        )
                    )
                    .shadow(color: isEnabled ? WeeColors.accent.opacity(0.3) : .clear, radius: 4, y: 2)

                Image(systemName: "arrow.up")
                    .font(.system(size: 15, weight: .bold))
                    .foregroundStyle(.white)
            }
            .frame(width: 36, height: 36)
        }
        .disabled(!isEnabled)
        .animation(.easeInOut(duration: 0.2), value: isEnabled)
    }
}

// MARK: - State Views

struct LoadingMessagesView: View {
    var body: some View {
        VStack(spacing: 16) {
            ProgressView()
                .scaleEffect(1.1)
                .tint(.indigo)
            Text("Loading messages…")
                .font(.caption)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(Color(.systemGroupedBackground))
    }
}

struct ErrorMessagesView: View {
    let error: String

    var body: some View {
        VStack(spacing: 16) {
            Image(systemName: "wifi.exclamationmark")
                .font(.system(size: 40, weight: .light))
                .foregroundStyle(.red.opacity(0.7))
            Text("Connection Issue")
                .font(.headline.weight(.semibold))
            Text(error)
                .font(.subheadline)
                .foregroundStyle(.secondary)
                .multilineTextAlignment(.center)
                .padding(.horizontal, 40)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(Color(.systemGroupedBackground))
    }
}

struct EmptyMessagesView: View {
    var body: some View {
        VStack(spacing: 16) {
            Image(systemName: "sparkles")
                .font(.system(size: 40, weight: .light))
                .foregroundStyle(
                    LinearGradient(
                        colors: [.indigo, .purple],
                        startPoint: .topLeading,
                        endPoint: .bottomTrailing
                    )
                )
            Text("No Messages Yet")
                .font(.headline.weight(.semibold))
            Text("Start a conversation below")
                .font(.subheadline)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(Color(.systemGroupedBackground))
    }
}

// MARK: - Preview

#Preview {
    NavigationStack {
        ChatView(sessionID: "example-id", sessionTitle: "Claude")
    }
}
