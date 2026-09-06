//
//  ZenView.swift
//  wee
//
//  Immersive ambient visualization of agent session activity
//  with floating particles, message ripples, role-based lighting effects,
//  central avatar, topic extraction, and persistent messages
//

import SwiftUI
import Combine

// MARK: - Topic Types

enum TopicCategory: String, CaseIterable {
    case code = "code"
    case debug = "debug"
    case refactor = "refactor"
    case design = "design"
    case config = "config"
    case test = "test"
    case deploy = "deploy"
    case docs = "docs"
    case question = "question"
    case general = "general"

    var color: Color {
        switch self {
        case .code: return Color(hex: "#8B5CF6") ?? .purple
        case .debug: return Color(hex: "#EF4444") ?? .red
        case .refactor: return Color(hex: "#F59E0B") ?? .orange
        case .design: return Color(hex: "#EC4899") ?? .pink
        case .config: return Color(hex: "#3B82F6") ?? .blue
        case .test: return Color(hex: "#10B981") ?? .green
        case .deploy: return Color(hex: "#06B6D4") ?? .cyan
        case .docs: return Color(hex: "#FBBF24") ?? .yellow
        case .question: return Color(hex: "#A78BFA") ?? .purple
        case .general: return Color(hex: "#6B7280") ?? .gray
        }
    }

    var hue: Double {
        switch self {
        case .code: return 0.75
        case .debug: return 0.0
        case .refactor: return 0.09
        case .design: return 0.92
        case .config: return 0.58
        case .test: return 0.4
        case .deploy: return 0.5
        case .docs: return 0.15
        case .question: return 0.7
        case .general: return 0.0
        }
    }

    var keywords: [String] {
        switch self {
        case .code: return ["fn", "=>", "{}", "[]", "const", "let", "async"]
        case .debug: return ["bug", "fix", "error", "log", "trace"]
        case .refactor: return ["clean", "move", "rename", "simplify"]
        case .design: return ["ui", "ux", "css", "layout", "style"]
        case .config: return ["env", "json", "setup", "install"]
        case .test: return ["test", "spec", "expect", "assert"]
        case .deploy: return ["build", "ship", "push", "deploy"]
        case .docs: return ["doc", "readme", "guide", "api"]
        case .question: return ["?", "why", "how", "what"]
        case .general: return ["...", ">>", "->", "=>"]
        }
    }
}

struct ExtractedTopic {
    let category: TopicCategory
    let keywords: [String]
    let intensity: Double // 0-1
}

// MARK: - Particle Model

struct Particle: Identifiable {
    let id = UUID()
    var x: CGFloat
    var y: CGFloat
    var vx: CGFloat
    var vy: CGFloat
    var size: CGFloat
    var opacity: Double
    var hue: Double
    var life: Double
    var maxLife: Double
}

// MARK: - Ripple Effect Model

struct Ripple: Identifiable {
    let id = UUID()
    var x: CGFloat
    var y: CGFloat
    var radius: CGFloat
    var maxRadius: CGFloat
    var opacity: Double
    var hue: Double
    var lineWidth: CGFloat
}

// MARK: - Topic Keyword Particle

struct TopicKeyword: Identifiable {
    let id = UUID()
    let text: String
    var x: CGFloat
    var y: CGFloat
    var opacity: Double
    var scale: CGFloat
}

// MARK: - Zen View

struct ZenView: View {
    let sessionID: String
    let sessionTitle: String

    @ObservedObject var webSocketViewModel: SessionsWebSocketViewModel
    @ObservedObject var avatarManager = AvatarManager.shared
    @State private var messages: [AgentMessage] = []
    @State private var lastMessageRole: String? = nil
    @State private var userInput = ""
    @FocusState private var isInputFocused: Bool

    // Particle system state
    @State private var particles: [Particle] = []
    @State private var ripples: [Ripple] = []
    @State private var topicKeywords: [TopicKeyword] = []
    @State private var messageGlowIntensity: Double = 0
    @State private var messageGlowHue: Double = 0.08
    @State private var currentTime: Double = 0

    // Topic extraction state
    @State private var currentTopic: ExtractedTopic? = nil
    @State private var topicPulseIntensity: Double = 0

    // Persistent message state (5s display)
    @State private var persistentUserMessage: String = ""
    @State private var persistentAssistantMessage: String = ""
    @State private var showPersistentUserMsg: Bool = false
    @State private var showPersistentAsstMsg: Bool = false
    private let messagePersistSeconds: Double = 5.0

    private let apiClient = WeeAPIClient.shared
    private let particleCount = 60
    private let timer = Timer.publish(every: 0.016, on: .main, in: .common).autoconnect()

    init(sessionID: String, sessionTitle: String, viewModel: SessionsWebSocketViewModel? = nil) {
        self.sessionID = sessionID
        self.sessionTitle = sessionTitle
        self.webSocketViewModel = viewModel ?? SessionsWebSocketViewModel.shared
    }

    // MARK: - Session Helpers

    private var currentSession: AgentSession? {
        // Look in both the WebSocket sessions and try to get the latest from view model
        let session = webSocketViewModel.sessions.first(where: { $0.id == sessionID })
        return session
    }

    private var sessionStatus: String {
        webSocketViewModel.sessionStatus[sessionID]
            ?? currentSession?.status
            ?? "idle"
    }

    private var isProcessing: Bool {
        sessionStatus == "processing"
    }

    private var sessionAvatar: Avatar? {
        // Try selected_avatar first, then fallback to looking up by selected_avatar_id
        currentSession?.selected_avatar ??
            (currentSession?.selected_avatar_id.flatMap { AvatarManager.shared.getAvatar(id: $0) })
    }

    @State private var sessionAvatarImage: UIImage? = nil

    private func loadAvatarImage() {
        #if DEBUG
        print("📥 ZenView.loadAvatarImage called")
        #endif
        #if DEBUG
        print("📥 ZenView: currentSession id = \(currentSession?.id ?? "nil")")
        #endif
        #if DEBUG
        print("📥 ZenView: currentSession selected_avatar_id = \(currentSession?.selected_avatar_id.map(String.init) ?? "nil")")
        #endif
        #if DEBUG
        print("📥 ZenView: currentSession selected_avatar = \(currentSession?.selected_avatar.map { "\($0.name) (id:\($0.id))" } ?? "nil")")
        #endif

        guard let avatar = sessionAvatar else {
            #if DEBUG
            print("⚠️ ZenView: No session avatar found (sessionAvatar computed property returned nil)")
            #endif
            sessionAvatarImage = nil
            return
        }
        #if DEBUG
        print("✅ ZenView: Found avatar \(avatar.name) (ID: \(avatar.id), image_path: \(avatar.image_path ?? "nil"), image_url: \(avatar.image_url ?? "nil"))")
        #endif

        // First check memory cache
        if let cached = avatarManager.getAvatarImage(id: avatar.id) {
            #if DEBUG
            print("✅ ZenView: Avatar image cached in memory")
            #endif
            sessionAvatarImage = cached
            return
        }

        #if DEBUG
        print("⚠️ ZenView: Avatar not in memory cache, loading from disk/server...")
        #endif

        // Otherwise load async
        Task {
            await avatarManager.loadAvatarImageIfNeeded(avatar: avatar)
            // Update after loading
            await MainActor.run {
                let loadedImage = avatarManager.getAvatarImage(id: avatar.id)
                sessionAvatarImage = loadedImage
                if loadedImage != nil {
                    #if DEBUG
                    print("✅ ZenView: Avatar image loaded successfully")
                    #endif
                } else {
                    #if DEBUG
                    print("❌ ZenView: Failed to load avatar image")
                    #endif
                }
            }
        }
    }

    private var sessionAvatarColor: Color {
        guard let colorStr = sessionAvatar?.color else { return .purple }
        return Color(hex: colorStr) ?? .purple
    }

    private var lastAssistantMessage: String? {
        let assistantMessages = messages.filter { $0.role == "assistant" }
        guard let last = assistantMessages.last else { return nil }
        let text = last.content.textContent
        let truncated = String(text.prefix(200))
        return truncated.isEmpty ? nil : truncated
    }

    private var lastUserMessage: String? {
        let userMessages = messages.filter { $0.role == "user" }
        guard let last = userMessages.last else { return nil }
        let text = last.content.textContent
        return text.isEmpty ? nil : String(text.prefix(200))
    }

    // MARK: - Body

    var body: some View {
        GeometryReader { geometry in
            ZStack {
                // Dynamic background with topic color influence
                backgroundLayer(size: geometry.size)

                // Particle layer
                particleLayer(size: geometry.size)

                // Ripple layer
                rippleLayer(size: geometry.size)

                // Topic keywords floating
                topicKeywordsLayer(size: geometry.size)

                // Vignette
                vignetteLayer

                // Central Avatar Visualization
                centralAvatarLayer(size: geometry.size)

                // UI Overlay
                overlayContent
            }
        }
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .principal) {
                Text("Zen")
                    .font(.system(size: 14, weight: .medium, design: .monospaced))
                    .foregroundStyle(.white.opacity(0.6))
            }
        }
        .toolbarBackground(.hidden, for: .navigationBar)
        .toolbarColorScheme(.dark, for: .navigationBar)
        .onAppear {
            initializeParticles()
            loadSessionAvatar()
        }
        .task {
            await initializeView()
            // Also try loading avatar after a delay to ensure AvatarManager has loaded
            try? await Task.sleep(nanoseconds: 500_000_000) // 0.5 seconds
            await MainActor.run {
                loadAvatarImage()
            }
        }
        .onReceive(webSocketViewModel.$sessionMessages) { syncMessages($0) }
        .onReceive(webSocketViewModel.$sessions) { _ in
            loadAvatarImage()
        }
        .onReceive(timer) { _ in
            updateAnimation()
        }
        .onReceive(avatarManager.$avatarImages) { _ in
            loadAvatarImage()
        }
        .onChange(of: messageCount) { oldCount, newCount in
            if newCount > oldCount {
                handleNewMessage()
            }
        }
        .onChange(of: currentSession?.selected_avatar_id) { _, _ in
            loadAvatarImage()
        }
        .onChange(of: currentSession?.id) { _, _ in
            loadAvatarImage()
        }
    }

    // MARK: - Avatar Loading

    private func loadSessionAvatar() {
        loadAvatarImage()
    }

    // MARK: - Topic Extraction

    private func extractTopic(from text: String) -> ExtractedTopic? {
        let lowerText = text.lowercased()

        // Detect category based on keywords
        var detectedCategory: TopicCategory = .general
        var maxScore = 0

        let categoryPatterns: [(TopicCategory, [String])] = [
            (.code, ["write", "create", "implement", "add", "function", "component", "class", "code for", "code to"]),
            (.debug, ["fix", "bug", "error", "crash", "broken", "not working", "fail", "debug", "exception"]),
            (.refactor, ["refactor", "clean", "rewrite", "restructure", "optimize", "improve", "simplify", "move", "extract", "rename"]),
            (.design, ["design", "layout", "ui", "ux", "style", "theme", "color", "font", "animation", "appearance", "visual"]),
            (.config, ["config", "setting", "env", "environment", "variable", "json", "yaml", "setup", "install", "package"]),
            (.test, ["test", "spec", "jest", "vitest", "cypress", "unit", "e2e", "coverage", "mock", "assert", "expect"]),
            (.deploy, ["deploy", "build", "release", "publish", "push", "pipeline", "ci/cd", "server", "production"]),
            (.docs, ["doc", "readme", "comment", "documentation", "explain", "describe", "guide", "tutorial", "jsdoc"]),
            (.question, ["what", "how", "why", "when", "where", "which", "can you", "could you", "would you"]),
        ]

        for (category, patterns) in categoryPatterns {
            let score = patterns.reduce(0) { count, pattern in
                lowerText.contains(pattern) ? count + 1 : count
            }
            if score > maxScore {
                maxScore = score
                detectedCategory = category
            }
        }

        // Extract keywords from the text
        let allKeywords = detectedCategory.keywords
        let foundKeywords = allKeywords.filter { kw in
            lowerText.contains(kw.lowercased())
        }

        // Intensity based on text length and keyword matches
        let intensity = min(Double(text.count) / 200.0 + Double(foundKeywords.count) * 0.1, 1.0)

        // If no strong signal, use general category
        if maxScore == 0 {
            detectedCategory = .general
        }

        return ExtractedTopic(
            category: detectedCategory,
            keywords: foundKeywords.isEmpty ? Array(allKeywords.prefix(3)) : foundKeywords,
            intensity: intensity
        )
    }

    // MARK: - Animation Update

    private func updateAnimation() {
        currentTime += 0.016

        // Decay glow
        if messageGlowIntensity > 0 {
            messageGlowIntensity = max(0, messageGlowIntensity - 0.015)
        }

        // Decay topic pulse
        if topicPulseIntensity > 0 {
            topicPulseIntensity = max(0, topicPulseIntensity - 0.02)
        }

        // Update particles
        updateParticles()

        // Update ripples
        updateRipples()

        // Update topic keywords
        updateTopicKeywords()
    }

    // MARK: - Background Layer

    private func backgroundLayer(size: CGSize) -> some View {
        ZStack {
            // Base dark color that shifts with topic or message glow
            let baseHue = currentTopic?.category.hue ?? messageGlowHue
            Color(
                hue: baseHue,
                saturation: 0.5,
                brightness: 0.08 + (messageGlowIntensity * 0.05)
            )
            .ignoresSafeArea()

            // Animated orbs with topic color influence
            Canvas { context, canvasSize in
                let time = currentTime
                let orbCount = 3

                for i in 0..<orbCount {
                    let offset = Double(i) * (2.0 * .pi / Double(orbCount))
                    let x = canvasSize.width * 0.5 + cos(time * 0.3 + offset) * canvasSize.width * 0.25
                    let y = canvasSize.height * 0.5 + sin(time * 0.25 + offset) * canvasSize.height * 0.25

                    let orbRect = CGRect(
                        x: x - canvasSize.width * 0.35,
                        y: y - canvasSize.height * 0.35,
                        width: canvasSize.width * 0.7,
                        height: canvasSize.height * 0.7
                    )

                    let orbHue = (currentTopic?.category.hue ?? messageGlowHue) + (Double(i) * 0.03)
                    let orbColor = Color(
                        hue: orbHue,
                        saturation: 0.6,
                        brightness: 0.25
                    ).opacity(0.08 + messageGlowIntensity * 0.08)

                    context.fill(
                        Ellipse().path(in: orbRect),
                        with: .color(orbColor)
                    )
                }
            }
            .ignoresSafeArea()
        }
    }

    // MARK: - Particle Layer

    private func particleLayer(size: CGSize) -> some View {
        Canvas { context, _ in
            for particle in particles {
                let lifeRatio = particle.life / particle.maxLife
                let alpha = particle.opacity * lifeRatio

                // Skip nearly invisible particles
                guard alpha > 0.01 else { continue }

                // Use topic hue if available
                let particleHue = currentTopic?.category.hue ?? particle.hue

                // Outer glow
                let outerRect = CGRect(
                    x: particle.x - particle.size * 1.2,
                    y: particle.y - particle.size * 1.2,
                    width: particle.size * 2.4,
                    height: particle.size * 2.4
                )

                context.fill(
                    Circle().path(in: outerRect),
                    with: .color(
                        Color(
                            hue: particleHue,
                            saturation: 0.6,
                            brightness: 0.8
                        ).opacity(alpha * 0.12)
                    )
                )

                // Middle glow
                let middleRect = CGRect(
                    x: particle.x - particle.size * 0.6,
                    y: particle.y - particle.size * 0.6,
                    width: particle.size * 1.2,
                    height: particle.size * 1.2
                )

                context.fill(
                    Circle().path(in: middleRect),
                    with: .color(
                        Color(
                            hue: particleHue,
                            saturation: 0.7,
                            brightness: 0.9
                        ).opacity(alpha * 0.25)
                    )
                )

                // Core
                let coreRect = CGRect(
                    x: particle.x - particle.size * 0.25,
                    y: particle.y - particle.size * 0.25,
                    width: particle.size * 0.5,
                    height: particle.size * 0.5
                )

                context.fill(
                    Circle().path(in: coreRect),
                    with: .color(
                        Color(
                            hue: particleHue,
                            saturation: 0.4,
                            brightness: 1.0
                        ).opacity(alpha)
                    )
                )
            }
        }
        .ignoresSafeArea()
    }

    // MARK: - Ripple Layer

    private func rippleLayer(size: CGSize) -> some View {
        Canvas { context, _ in
            for ripple in ripples {
                guard ripple.opacity > 0.01 else { continue }

                let rect = CGRect(
                    x: ripple.x - ripple.radius,
                    y: ripple.y - ripple.radius,
                    width: ripple.radius * 2,
                    height: ripple.radius * 2
                )

                let rippleHue = currentTopic?.category.hue ?? ripple.hue
                let rippleColor = Color(
                    hue: rippleHue,
                    saturation: 0.8,
                    brightness: 1.0
                ).opacity(ripple.opacity)

                // Stroke
                context.stroke(
                    Circle().path(in: rect),
                    with: .color(rippleColor),
                    lineWidth: ripple.lineWidth
                )

                // Inner fill
                context.fill(
                    Circle().path(in: rect),
                    with: .color(rippleColor.opacity(ripple.opacity * 0.08))
                )
            }
        }
        .ignoresSafeArea()
    }

    // MARK: - Topic Keywords Layer

    private func topicKeywordsLayer(size: CGSize) -> some View {
        ZStack {
            ForEach(topicKeywords) { keyword in
                Text(keyword.text)
                    .font(.system(size: 12, weight: .medium, design: .monospaced))
                    .foregroundColor((currentTopic?.category.color ?? .purple).opacity(keyword.opacity))
                    .scaleEffect(keyword.scale)
                    .position(x: keyword.x, y: keyword.y)
            }
        }
        .ignoresSafeArea()
    }

    // MARK: - Vignette Layer

    private var vignetteLayer: some View {
        GeometryReader { geometry in
            RadialGradient(
                colors: [
                    .clear,
                    .black.opacity(0.4),
                    .black.opacity(0.7)
                ],
                center: .center,
                startRadius: geometry.size.width * 0.25,
                endRadius: geometry.size.width * 0.9
            )
        }
        .ignoresSafeArea()
        .allowsHitTesting(false)
    }

    // MARK: - Central Avatar Layer

    private func centralAvatarLayer(size: CGSize) -> some View {
        VStack(spacing: 16) {
            // Avatar circle with topic-colored ring
            ZStack {
                // Outer glow ring
                Circle()
                    .stroke(
                        (currentTopic?.category.color ?? sessionAvatarColor).opacity(0.6),
                        lineWidth: 2
                    )
                    .frame(width: 100, height: 100)
                    .blur(radius: topicPulseIntensity * 10)
                    .scaleEffect(1 + topicPulseIntensity * 0.1)

                // Main avatar ring
                Circle()
                    .stroke(
                        currentTopic?.category.color ?? sessionAvatarColor,
                        lineWidth: 2
                    )
                    .frame(width: 90, height: 90)
                    .background(
                        Circle()
                            .fill(Color.black.opacity(0.4))
                    )

                // Avatar image or fallback
                if let avatarImage = sessionAvatarImage {
                    Image(uiImage: avatarImage)
                        .resizable()
                        .scaledToFit()
                        .frame(width: 80, height: 80)
                        .clipShape(Circle())
                } else {
                    Image(systemName: "person.fill")
                        .font(.system(size: 32))
                        .foregroundColor(.white.opacity(0.5))
                }

                // Processing indicator ring
                if isProcessing {
                    ProcessingRings()
                        .frame(width: 110, height: 110)
                }
            }

            // Avatar name
            if let avatar = sessionAvatar {
                Text(avatar.name)
                    .font(.system(size: 16, weight: .medium))
                    .foregroundColor(currentTopic?.category.color ?? sessionAvatarColor)
            }

            // Topic indicator
            if let topic = currentTopic {
                VStack(spacing: 8) {
                    Text(topic.category.rawValue.uppercased())
                        .font(.system(size: 10, weight: .semibold, design: .monospaced))
                        .foregroundColor(topic.category.color)
                        .padding(.horizontal, 12)
                        .padding(.vertical, 4)
                        .background(
                            Capsule()
                                .fill(topic.category.color.opacity(0.15))
                                .overlay(
                                    Capsule()
                                        .stroke(topic.category.color.opacity(0.3), lineWidth: 1)
                                )
                        )

                    HStack(spacing: 8) {
                        ForEach(topic.keywords.prefix(3), id: \.self) { keyword in
                            Text("#\(keyword)")
                                .font(.system(size: 10, weight: .medium, design: .monospaced))
                                .foregroundColor(topic.category.color.opacity(0.8))
                        }
                    }
                }
            }
        }
        .position(x: size.width / 2, y: size.height / 2)
    }

    // MARK: - Overlay Content

    private var overlayContent: some View {
        GeometryReader { geo in
            VStack(spacing: 0) {
                topBar

                Spacer()

                // Position the message text in the upper-center area,
                // above the central avatar which sits at the vertical center.
                centerMessage
                    .frame(maxWidth: .infinity)

                // Push the message text above the avatar by reserving the
                // bottom half of the available space (avatar + name + gap).
                Spacer()
                    .frame(minHeight: geo.size.height * 0.35)

                inputBar
            }
        }
    }

    // MARK: - Top Bar

    private var topBar: some View {
        HStack {
            HStack(spacing: 8) {
                ZStack {
                    Circle()
                        .fill(statusColor)
                        .frame(width: 14, height: 14)
                        .opacity(isProcessing ? 0.25 : 0)
                        .scaleEffect(isProcessing ? 2.0 : 1.0)

                    Circle()
                        .fill(statusColor)
                        .frame(width: 7, height: 7)
                }
                .animation(isProcessing ? .easeOut(duration: 1.0).repeatForever(autoreverses: false) : .default, value: isProcessing)

                Text(statusLabel)
                    .font(.system(size: 11, weight: .medium, design: .monospaced))
                    .foregroundStyle(.white.opacity(0.6))
            }

            Spacer()

            if let session = currentSession, let project = session.project_id {
                Text(project)
                    .font(.system(size: 11, weight: .medium, design: .monospaced))
                    .foregroundStyle(.white.opacity(0.4))
                    .lineLimit(1)
            }
        }
        .padding(.horizontal, 20)
        .padding(.top, 8)
    }

    private var statusColor: Color {
        switch sessionStatus {
        case "processing": return .orange
        case "idle": return .green.opacity(0.8)
        case "completed": return .blue.opacity(0.8)
        default: return .gray.opacity(0.6)
        }
    }

    private var statusLabel: String {
        switch sessionStatus {
        case "processing": return "Processing"
        case "idle": return "Idle"
        case "completed": return "Done"
        default: return sessionStatus.capitalized
        }
    }

    // MARK: - Center Message

    private var centerMessage: some View {
        Group {
            if showPersistentAsstMsg && !persistentAssistantMessage.isEmpty {
                // Show persistent assistant message
                VStack(spacing: 16) {
                    HStack(spacing: 6) {
                        Circle()
                            .fill(currentTopic?.category.color ?? Color(hue: messageGlowHue, saturation: 0.7, brightness: 0.9))
                            .frame(width: 6, height: 6)

                        Text("Assistant")
                            .font(.system(size: 10, weight: .medium, design: .monospaced))
                            .foregroundStyle(.white.opacity(0.4))
                            .textCase(.uppercase)
                    }

                    Text(persistentAssistantMessage)
                        .font(.system(size: 18, weight: .light, design: .serif))
                        .foregroundStyle(.white.opacity(0.85))
                        .multilineTextAlignment(.center)
                        .lineLimit(6)
                        .shadow(
                            color: (currentTopic?.category.color ?? Color(hue: messageGlowHue, saturation: 0.5, brightness: 0.5))
                                .opacity(messageGlowIntensity * 0.4),
                            radius: 20
                        )
                        .animation(.easeOut(duration: 0.4), value: persistentAssistantMessage)
                }
                .padding(.horizontal, 32)
            } else if showPersistentUserMsg && !persistentUserMessage.isEmpty {
                // Show persistent user message
                VStack(spacing: 16) {
                    HStack(spacing: 6) {
                        Image(systemName: "person.fill")
                            .font(.system(size: 8))
                            .foregroundStyle(.white.opacity(0.4))

                        Text("You")
                            .font(.system(size: 10, weight: .medium, design: .monospaced))
                            .foregroundStyle(.white.opacity(0.4))
                            .textCase(.uppercase)
                    }

                    Text(persistentUserMessage)
                        .font(.system(size: 18, weight: .light, design: .serif))
                        .foregroundStyle(.white.opacity(0.7))
                        .multilineTextAlignment(.center)
                        .lineLimit(4)
                }
                .padding(.horizontal, 32)
            } else if isProcessing {
                // Processing state shown in central avatar layer with rings
                EmptyView()
            } else {
                VStack(spacing: 12) {
                    IdleBreathingOrb()
                        .frame(width: 40, height: 40)

                    Text("Session Active")
                        .font(.system(size: 16, weight: .ultraLight, design: .serif))
                        .foregroundStyle(.white.opacity(0.25))
                }
            }
        }
    }

    // MARK: - Input Bar

    private var inputBar: some View {
        HStack(spacing: 12) {
            ZStack(alignment: .topLeading) {
                RoundedRectangle(cornerRadius: 20)
                    .fill(Color.white.opacity(0.08))
                    .overlay(
                        RoundedRectangle(cornerRadius: 20)
                            .stroke(Color.white.opacity(0.1), lineWidth: 1)
                    )

                TextField("Message…", text: $userInput, axis: .vertical)
                    .textFieldStyle(.plain)
                    .lineLimit(1...3)
                    .padding(.horizontal, 16)
                    .padding(.vertical, 10)
                    .font(.body)
                    .foregroundStyle(.white)
                    .focused($isInputFocused)
                    .onSubmit {
                        if !userInput.trimmingCharacters(in: .whitespaces).isEmpty {
                            sendMessage()
                        }
                    }
            }
            .frame(minHeight: 40, maxHeight: 80)

            ZStack {
                Circle()
                    .fill(userInput.isEmpty ? Color.clear : Color.orange.opacity(0.3))
                    .frame(width: 50, height: 50)
                    .blur(radius: 10)
                    .opacity(userInput.isEmpty ? 0 : 1)

                Circle()
                    .fill(userInput.isEmpty ? Color.white.opacity(0.1) : Color.orange.opacity(0.9))

                Image(systemName: "arrow.up")
                    .font(.system(size: 14, weight: .bold))
                    .foregroundStyle(.white)
            }
            .frame(width: 36, height: 36)
            .onTapGesture {
                if !userInput.isEmpty {
                    sendMessage()
                }
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
        .background(
            LinearGradient(
                colors: [.clear, .black.opacity(0.5), .black.opacity(0.8)],
                startPoint: .top,
                endPoint: .bottom
            )
            .blur(radius: 20)
        )
    }

    // MARK: - Message Count

    private var messageCount: Int {
        messages.count
    }

    // MARK: - Actions

    private func initializeView() async {
        if !webSocketViewModel.webSocketConnected {
            await webSocketViewModel.startMonitoring()
        }
        webSocketViewModel.selectSession(sessionID)

        // Load avatar if needed
        loadSessionAvatar()
    }

    private func syncMessages(_ messagesDict: [String: [AgentMessage]]) {
        if let sessionMessages = messagesDict[sessionID] {
            if let lastMsg = sessionMessages.last {
                lastMessageRole = lastMsg.role
            }
            self.messages = sessionMessages
        }
    }

    private func sendMessage() {
        let text = userInput.trimmingCharacters(in: .whitespaces)
        guard !text.isEmpty else { return }

        userInput = ""
        isInputFocused = false

        let optimisticMessage = AgentMessage(
            id: UUID().uuidString,
            role: "user",
            content: .string(text),
            created_at: ISO8601DateFormatter().string(from: Date()),
            thinking: nil,
            tools: nil
        )
        self.messages.append(optimisticMessage)
        lastMessageRole = "user"

        var updatedDict = webSocketViewModel.sessionMessages
        var sessionMessages = updatedDict[sessionID] ?? []
        sessionMessages.append(optimisticMessage)
        updatedDict[sessionID] = sessionMessages
        webSocketViewModel.sessionMessages = updatedDict

        triggerUserMessageEffect()
        webSocketViewModel.sendMessage(text)
    }

    // MARK: - Persistent Message Logic

    private func setPersistentUserMessage(_ text: String) {
        persistentUserMessage = text
        showPersistentUserMsg = true

        // Hide after 5 seconds
        DispatchQueue.main.asyncAfter(deadline: .now() + messagePersistSeconds) { [self] in
            showPersistentUserMsg = false
        }
    }

    private func setPersistentAssistantMessage(_ text: String) {
        persistentAssistantMessage = text
        showPersistentAsstMsg = true

        // Extract topic from the message
        if let topic = extractTopic(from: text) {
            currentTopic = topic
            topicPulseIntensity = 1.0
            spawnTopicKeywords(topic: topic)
        }

        // Hide after 5 seconds
        DispatchQueue.main.asyncAfter(deadline: .now() + messagePersistSeconds) { [self] in
            showPersistentAsstMsg = false
        }
    }

    // MARK: - Topic Keywords

    private func spawnTopicKeywords(topic: ExtractedTopic) {
        guard let window = UIApplication.shared.windows.first else { return }
        let size = window.bounds.size

        // Create floating keywords
        for (i, keyword) in topic.keywords.enumerated() {
            DispatchQueue.main.asyncAfter(deadline: .now() + Double(i) * 0.1) {
                let keyword = TopicKeyword(
                    text: keyword,
                    x: size.width / 2 + CGFloat.random(in: -100...100),
                    y: size.height / 2 + CGFloat.random(in: -80...80),
                    opacity: 0.8,
                    scale: 1.0
                )
                topicKeywords.append(keyword)

                // Remove after animation
                DispatchQueue.main.asyncAfter(deadline: .now() + 2.0) {
                    topicKeywords.removeAll { $0.id == keyword.id }
                }
            }
        }
    }

    private func updateTopicKeywords() {
        for i in topicKeywords.indices {
            topicKeywords[i].opacity -= 0.01
            topicKeywords[i].y -= 0.5 // Float upward
            topicKeywords[i].scale += 0.002 // Slight grow
        }
        topicKeywords.removeAll { $0.opacity <= 0 }
    }

    // MARK: - Message Effects

    private func handleNewMessage() {
        guard let role = lastMessageRole else { return }

        if role == "user" {
            if let msg = lastUserMessage {
                setPersistentUserMessage(msg)
            }
            triggerUserMessageEffect()
        } else if role == "assistant" {
            if let msg = lastAssistantMessage {
                setPersistentAssistantMessage(msg)
            }
            triggerAssistantMessageEffect()
        }
    }

    private func triggerUserMessageEffect() {
        // Use topic hue if available, otherwise blue/cyan
        let hue = currentTopic?.category.hue ?? 0.58
        messageGlowHue = hue
        messageGlowIntensity = 1.0

        guard let window = UIApplication.shared.windows.first else { return }
        let screenSize = window.bounds.size

        // Ripple from bottom
        createRipple(
            x: screenSize.width / 2,
            y: screenSize.height - 120,
            hue: hue,
            intensity: 0.7
        )

        // Burst particles upward
        for i in particles.indices {
            particles[i].vy -= Double.random(in: 2...5)
            particles[i].hue = hue
            particles[i].opacity = min(particles[i].opacity + 0.4, 1.0)
        }
    }

    private func triggerAssistantMessageEffect() {
        // Use topic hue if available, otherwise purple/magenta
        let hue = currentTopic?.category.hue ?? 0.82
        messageGlowHue = hue
        messageGlowIntensity = 1.0

        guard let window = UIApplication.shared.windows.first else { return }
        let screenSize = window.bounds.size
        let centerX = screenSize.width / 2
        let centerY = screenSize.height / 2 - 50

        // Multiple ripples
        createRipple(x: centerX, y: centerY, hue: hue, intensity: 0.9)

        DispatchQueue.main.asyncAfter(deadline: .now() + 0.12) {
            createRipple(x: centerX, y: centerY, hue: hue, intensity: 0.5)
        }

        // Burst outward
        for i in particles.indices {
            let dx = particles[i].x - centerX
            let dy = particles[i].y - centerY
            let dist = max(sqrt(dx * dx + dy * dy), 1)
            let burstForce: Double = 4.5

            particles[i].vx += (dx / dist) * burstForce
            particles[i].vy += (dy / dist) * burstForce
            particles[i].hue = hue
            particles[i].opacity = min(particles[i].opacity + 0.5, 1.0)
        }
    }

    // MARK: - Ripple System

    private func createRipple(x: CGFloat, y: CGFloat, hue: Double, intensity: Double) {
        guard let window = UIApplication.shared.windows.first else { return }
        let maxR = min(window.bounds.width, window.bounds.height) * 0.5

        let ripple = Ripple(
            x: x,
            y: y,
            radius: 5,
            maxRadius: maxR,
            opacity: intensity,
            hue: hue,
            lineWidth: 2.5
        )
        ripples.append(ripple)
    }

    private func updateRipples() {
        for i in (0..<ripples.count).reversed() {
            var r = ripples[i]
            r.radius += r.maxRadius * 0.025
            let progress = r.radius / r.maxRadius
            r.opacity *= (1 - progress * 0.04)
            r.lineWidth = 3 * (1 - progress)

            if r.opacity < 0.01 || r.radius >= r.maxRadius {
                ripples.remove(at: i)
            } else {
                ripples[i] = r
            }
        }
    }

    // MARK: - Particle System

    private func initializeParticles() {
        guard let window = UIApplication.shared.windows.first else { return }
        let size = window.bounds.size

        particles = (0..<particleCount).map { _ in
            createRandomParticle(screenSize: size)
        }
    }

    private func createRandomParticle(screenSize: CGSize) -> Particle {
        Particle(
            x: CGFloat.random(in: 0...screenSize.width),
            y: CGFloat.random(in: 0...screenSize.height),
            vx: Double.random(in: -0.4...0.4),
            vy: Double.random(in: -0.6...(-0.15)),
            size: CGFloat.random(in: 2...6),
            opacity: Double.random(in: 0.25...0.7),
            hue: Double.random(in: 0.05...0.15),
            life: Double.random(in: 0.5...1.0),
            maxLife: 1.0
        )
    }

    private func updateParticles() {
        guard let window = UIApplication.shared.windows.first else { return }
        let size = window.bounds.size
        let time = currentTime
        let speedMult = isProcessing ? 2.2 : 1.0

        for i in particles.indices {
            var p = particles[i]

            // Move
            var newX = p.x + CGFloat(p.vx * speedMult)
            var newY = p.y + CGFloat(p.vy * speedMult)

            // Sine wave drift
            let sineOffset = sin(time * 0.5 + Double(i) * 0.1) * 0.4
            newX += CGFloat(sineOffset)

            // Processing orbital effect
            if isProcessing {
                let centerX = size.width / 2
                let centerY = size.height / 2 - 50
                let dx = newX - centerX
                let dy = newY - centerY
                let angle = atan2(dy, dx) + 0.01
                let dist = sqrt(dx * dx + dy * dy)
                newX = centerX + cos(angle) * dist * 0.995
                newY = centerY + sin(angle) * dist * 0.995
            }

            // Wrap edges
            if newY < -20 {
                newY = size.height + 20
                newX = CGFloat.random(in: 0...size.width)
            }
            if newX < -20 { newX = size.width + 20 }
            if newX > size.width + 20 { newX = -20 }

            // Life decay
            p.life -= 0.001
            if p.life <= 0 {
                particles[i] = createRandomParticle(screenSize: size)
                particles[i].y = size.height + 20
                continue
            }

            // Damping and drift restore
            p.vx *= 0.985
            p.vy *= 0.985
            if p.vy > -0.15 {
                p.vy -= 0.008
            }

            // Return to base hue or use topic hue
            let targetHue = currentTopic?.category.hue ?? 0.1
            if !isProcessing && messageGlowIntensity < 0.1 {
                p.hue += (targetHue - p.hue) * 0.008
            }

            particles[i].x = newX
            particles[i].y = newY
            particles[i].life = p.life
            particles[i].vx = p.vx
            particles[i].vy = p.vy
            particles[i].hue = p.hue
        }
    }
}

// MARK: - Processing Rings

struct ProcessingRings: View {
    @State private var rotation: Double = 0

    var body: some View {
        ZStack {
            Circle()
                .stroke(
                    AngularGradient(
                        colors: [.orange.opacity(0), .orange.opacity(0.5), .orange],
                        center: .center,
                        startAngle: .degrees(0),
                        endAngle: .degrees(360)
                    ),
                    lineWidth: 2
                )
                .rotationEffect(.degrees(rotation))

            Circle()
                .stroke(
                    AngularGradient(
                        colors: [.purple.opacity(0), .purple.opacity(0.3), .purple],
                        center: .center,
                        startAngle: .degrees(180),
                        endAngle: .degrees(540)
                    ),
                    lineWidth: 1.5
                )
                .rotationEffect(.degrees(-rotation * 1.5))
                .scaleEffect(0.7)

            Circle()
                .fill(.orange.opacity(0.8))
                .frame(width: 8, height: 8)
                .scaleEffect(1 + sin(rotation * 0.1) * 0.2)
        }
        .onAppear {
            withAnimation(.linear(duration: 3).repeatForever(autoreverses: false)) {
                rotation = 360
            }
        }
    }
}

// MARK: - Idle Breathing Orb

struct IdleBreathingOrb: View {
    @State private var scale: CGFloat = 1.0
    @State private var opacity: Double = 0.3

    var body: some View {
        ZStack {
            Circle()
                .fill(
                    RadialGradient(
                        colors: [.orange.opacity(0.2), .clear],
                        center: .center,
                        startRadius: 0,
                        endRadius: 20
                    )
                )
                .scaleEffect(scale * 1.5)
                .opacity(opacity)

            Circle()
                .fill(.orange.opacity(0.4))
                .scaleEffect(scale)
        }
        .onAppear {
            withAnimation(.easeInOut(duration: 3).repeatForever(autoreverses: true)) {
                scale = 1.2
                opacity = 0.6
            }
        }
    }
}

// MARK: - Color Extension

extension Color {
    init?(hex: String) {
        let hex = hex.trimmingCharacters(in: CharacterSet.alphanumerics.inverted)
        var int: UInt64 = 0
        Scanner(string: hex).scanHexInt64(&int)
        let a, r, g, b: UInt64
        switch hex.count {
        case 3: // RGB (12-bit)
            (a, r, g, b) = (255, (int >> 8) * 17, (int >> 4 & 0xF) * 17, (int & 0xF) * 17)
        case 6: // RGB (24-bit)
            (a, r, g, b) = (255, int >> 16, int >> 8 & 0xFF, int & 0xFF)
        case 8: // ARGB (32-bit)
            (a, r, g, b) = (int >> 24, int >> 16 & 0xFF, int >> 8 & 0xFF, int & 0xFF)
        default:
            return nil
        }
        self.init(
            .sRGB,
            red: Double(r) / 255,
            green: Double(g) / 255,
            blue: Double(b) / 255,
            opacity: Double(a) / 255
        )
    }
}

#Preview {
    NavigationStack {
        ZenView(sessionID: "example", sessionTitle: "Test Session")
    }
    .preferredColorScheme(.dark)
}
