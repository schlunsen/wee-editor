# Wee - Just Commands
# Install just: https://github.com/casey/just
# Works on macOS, Linux, and Windows (PowerShell)

set windows-shell := ["powershell.exe", "-NoLogo", "-Command"]

# Binary name changes based on OS
wee := if os_family() == "windows" { "wee.exe" } else { "wee" }
frontend_dir := "internal/server/frontend"

# Default recipe to display help
default:
    @just --list

# Build the application (with frontend)
build: build-frontend build-go

# Build frontend only
[unix]
build-frontend:
    @echo "Building Nuxt frontend..."
    @if [ ! -d "{{frontend_dir}}/node_modules" ]; then \
        echo "📦 Installing frontend dependencies..."; \
        npm --prefix {{frontend_dir}} install; \
    fi
    @npm --prefix {{frontend_dir}} run generate
    @echo "✅ Frontend build complete"

[windows]
build-frontend:
    @echo "Building Nuxt frontend..."
    @if (-not (Test-Path "{{frontend_dir}}/node_modules")) { echo "📦 Installing frontend dependencies..."; npm --prefix {{frontend_dir}} install }
    @npm --prefix {{frontend_dir}} run generate
    @echo "✅ Frontend build complete"

# Build Go binary only (assumes frontend already built)
[unix]
build-go:
    @echo "Building wee..."
    @go build -o {{wee}} ./cmd/wee
    @echo "✅ Build complete: ./{{wee}}"

[windows]
build-go:
    @echo "Building wee..."
    @$env:CGO_ENABLED = "1"; go build -o {{wee}} ./cmd/wee
    @echo "✅ Build complete: {{wee}}"

# Run the application
run:
    @go run ./cmd/wee

# Run with analytics flag (web dashboard)
analytics:
    @go run ./cmd/wee --analytics

# Run Nuxt frontend in development mode
frontend-dev:
    @echo "Starting Nuxt development server..."
    @npm --prefix {{frontend_dir}} run dev

# Build Nuxt frontend for production (alias for build-frontend)
frontend-build: build-frontend

# Install Nuxt frontend dependencies
frontend-install:
    @echo "Installing frontend dependencies..."
    @npm --prefix {{frontend_dir}} install
    @echo "✅ Frontend dependencies installed"

# Full development workflow: build frontend then start analytics
dev-full: build-frontend build-go analytics

# Run with agents flag
agents:
    @go run ./cmd/wee --agents

# Run with chats flag
chats:
    @go run ./cmd/wee --chats

# Run help
help:
    @go run ./cmd/wee --help

# Install specific agent
install-agent agent:
    @go run ./cmd/wee --agent {{agent}}

# Install specific command
install-command command:
    @go run ./cmd/wee --command {{command}}

# Install specific MCP
install-mcp mcp:
    @go run ./cmd/wee --mcp {{mcp}}

# Setup TTS: download Pocket TTS model for local KITT voice clone
[unix]
setup-tts:
    @echo "🔊 Setting up local TTS (Sherpa-ONNX Pocket TTS)..."
    @mkdir -p models
    @if [ -d "models/sherpa-onnx-pocket-tts-int8-2026-01-26" ]; then \
        echo "✅ Pocket TTS model already downloaded"; \
    else \
        echo "📥 Downloading Pocket TTS model (~94MB)..."; \
        curl -sL "https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/sherpa-onnx-pocket-tts-int8-2026-01-26.tar.bz2" -o models/pocket-tts.tar.bz2; \
        cd models && tar xjf pocket-tts.tar.bz2 && rm pocket-tts.tar.bz2; \
        echo "✅ Pocket TTS model downloaded"; \
    fi
    @echo ""
    @echo "🎙️  KITT voice reference:"
    @if [ -f "$HOME/.claude/assets/kitt-voice-ref.wav" ]; then \
        echo "   ✅ Found at ~/.claude/assets/kitt-voice-ref.wav"; \
    else \
        echo "   ⚠️  Not found. Place kitt-voice-ref.wav in ~/.claude/assets/ for voice cloning"; \
    fi
    @echo ""
    @echo "🎉 TTS ready! The server will auto-detect the model on startup."
    @echo "   API: POST /api/tts/generate {\"text\": \"Hello\", \"backend\": \"sherpa\"}"

[windows]
setup-tts:
    @echo "🔊 Setting up local TTS (Sherpa-ONNX Pocket TTS)..."
    @if (-not (Test-Path "models")) { New-Item -ItemType Directory -Path "models" | Out-Null }
    @if (Test-Path "models/sherpa-onnx-pocket-tts-int8-2026-01-26") { echo "✅ Pocket TTS model already downloaded" } else { echo "📥 Downloading Pocket TTS model (~94MB)..."; Invoke-WebRequest -Uri "https://github.com/k2-fsa/sherpa-onnx/releases/download/tts-models/sherpa-onnx-pocket-tts-int8-2026-01-26.tar.bz2" -OutFile "models/pocket-tts.tar.bz2"; tar xjf "models/pocket-tts.tar.bz2" -C models; Remove-Item "models/pocket-tts.tar.bz2"; echo "✅ Pocket TTS model downloaded" }
    @echo "🎉 TTS ready! The server will auto-detect the model on startup."

# Setup F5-TTS: download ONNX models for browser-based TTS with voice cloning
[unix]
setup-f5-tts:
    @echo "🧠 Setting up F5-TTS (browser WebGPU inference)..."
    @mkdir -p models/f5-tts
    @if [ -f "models/f5-tts/F5_Preprocess.onnx" ]; then \
        echo "✅ F5-TTS models already downloaded"; \
    else \
        echo "📥 Downloading F5-TTS ONNX models from HuggingFace (~711MB total)..."; \
        echo "   F5_Preprocess.onnx (17MB)..."; \
        curl -sL "https://huggingface.co/huggingfacess/F5-TTS-ONNX/resolve/main/F5_Preprocess.onnx" -o models/f5-tts/F5_Preprocess.onnx; \
        echo "   F5_Decode.onnx (30MB)..."; \
        curl -sL "https://huggingface.co/huggingfacess/F5-TTS-ONNX/resolve/main/F5_Decode.onnx" -o models/f5-tts/F5_Decode.onnx; \
        echo "   F5_Transformer.onnx (664MB — this takes a while)..."; \
        curl -L "https://huggingface.co/huggingfacess/F5-TTS-ONNX/resolve/main/F5_Transformer.onnx" -o models/f5-tts/F5_Transformer.onnx; \
        echo "✅ F5-TTS models downloaded"; \
    fi
    @echo ""
    @echo "🎉 F5-TTS ready for browser inference via WebGPU!"
    @echo "   Models served at: /api/tts/models/f5-tts/"

# Check TTS status
tts-status:
    @echo "🔊 TTS Backend Status:"
    @echo ""
    @echo "Pocket TTS model:"
    @if [ -d "models/sherpa-onnx-pocket-tts-int8-2026-01-26" ]; then echo "  ✅ Installed"; else echo "  ❌ Not found (run: just setup-tts)"; fi
    @echo ""
    @echo "KITT voice reference:"
    @if [ -f "$HOME/.claude/assets/kitt-voice-ref.wav" ]; then echo "  ✅ Found"; else echo "  ⚠️  Not found"; fi
    @echo ""
    @echo "macOS say:"
    @which say > /dev/null 2>&1 && echo "  ✅ Available" || echo "  ❌ Not available"

# Clean build artifacts
[unix]
clean:
    @echo "Cleaning..."
    @rm -f wee
    @rm -rf dist/
    @echo "✅ Clean complete"

[windows]
clean:
    @echo "Cleaning..."
    @if (Test-Path wee.exe) { Remove-Item wee.exe }
    @if (Test-Path dist) { Remove-Item -Recurse -Force dist }
    @echo "✅ Clean complete"

# Install to GOPATH/bin
install:
    @echo "Installing wee..."
    @go install ./cmd/wee
    @echo "✅ Installed to GOPATH/bin"

# Run all tests
test:
    @echo "Running tests..."
    @go test -v ./...

# Run tests with coverage
[unix]
test-coverage:
    @echo "Running tests with coverage..."
    @go test -v -coverprofile=coverage.out -covermode=atomic ./...
    @go tool cover -html=coverage.out -o coverage.html
    @echo ""
    @echo "📊 Coverage Summary:"
    @echo "   Total Coverage: $(go tool cover -func=coverage.out | tail -1 | grep -oE '[0-9]+\.[0-9]+%')"
    @echo ""
    @echo "✅ Coverage report generated:"
    @echo "   HTML: coverage.html"
    @echo "   Data: coverage.out"

[windows]
test-coverage:
    @echo "Running tests with coverage..."
    @go test -v -coverprofile=coverage.out -covermode=atomic ./...
    @go tool cover -html=coverage.out -o coverage.html
    @echo ""
    @echo "✅ Coverage report generated:"
    @echo "   HTML: coverage.html"
    @echo "   Data: coverage.out"

# Build for all platforms
[unix]
build-all: build-frontend
    @echo "Building for multiple platforms..."
    @mkdir -p dist
    @GOOS=linux GOARCH=amd64 go build -o dist/wee-linux-amd64 ./cmd/wee
    @GOOS=linux GOARCH=arm64 go build -o dist/wee-linux-arm64 ./cmd/wee
    @GOOS=darwin GOARCH=amd64 go build -o dist/wee-darwin-amd64 ./cmd/wee
    @GOOS=darwin GOARCH=arm64 go build -o dist/wee-darwin-arm64 ./cmd/wee
    @GOOS=windows GOARCH=amd64 go build -o dist/wee-windows-amd64.exe ./cmd/wee
    @echo "✅ Build complete for all platforms in ./dist/"

[windows]
build-all: build-frontend
    @echo "Building for multiple platforms..."
    @if (-not (Test-Path dist)) { New-Item -ItemType Directory -Path dist | Out-Null }
    @$env:GOOS="linux"; $env:GOARCH="amd64"; go build -o dist/wee-linux-amd64 ./cmd/wee
    @$env:GOOS="linux"; $env:GOARCH="arm64"; go build -o dist/wee-linux-arm64 ./cmd/wee
    @$env:GOOS="darwin"; $env:GOARCH="amd64"; go build -o dist/wee-darwin-amd64 ./cmd/wee
    @$env:GOOS="darwin"; $env:GOARCH="arm64"; go build -o dist/wee-darwin-arm64 ./cmd/wee
    @$env:GOOS="windows"; $env:GOARCH="amd64"; go build -o dist/wee-windows-amd64.exe ./cmd/wee
    @echo "✅ Build complete for all platforms in ./dist/"

# Format code
fmt:
    @echo "Formatting code..."
    @go fmt ./...
    @echo "✅ Format complete"

# Format code (alias for fmt)
format: fmt

# Lint code
[unix]
lint:
    @echo "Linting code..."
    @golangci-lint run || go vet ./...
    @echo "✅ Lint complete"

[windows]
lint:
    @echo "Linting code..."
    @go vet ./...
    @echo "✅ Lint complete"

# Download and tidy dependencies
deps:
    @echo "Downloading dependencies..."
    @go mod download
    @go mod tidy
    @echo "✅ Dependencies updated"

# Run the app with verbose logging
verbose:
    @go run ./cmd/wee --verbose

# Quick test - build and run help
[unix]
quick: build
    @./wee --help

[windows]
quick: build
    @./wee.exe --help

# Development mode - build and test
dev: fmt build test
    @echo "✅ Development checks passed"

# ──────────────────────────────────────────────────────
# macOS/Linux only recipes below (release, deploy, etc.)
# ──────────────────────────────────────────────────────

# Development time estimation (unix only)
[unix]
time-used:
    @echo "🕒 Analyzing development time from git history..."
    @./scripts/time-used.sh

# Build and start the web dashboard
[unix]
start: build
    @echo "🚀 Starting wee..."
    @./wee -v --analytics

[windows]
start: build
    @echo "🚀 Starting wee..."
    @./wee.exe -v --analytics

# Create a new release (tags, builds, updates Homebrew formula) — macOS/Linux only
[unix]
release version release_name:
    #!/usr/bin/env bash
    set -euo pipefail
    \
    echo "🚀 Creating release v{{version}} - {{release_name}}..."; \
    \
    if [[ "{{version}}" != v* ]]; then \
        VERSION="v{{version}}"; \
        VERSION_NUM="{{version}}"; \
    else \
        VERSION="{{version}}"; \
        VERSION_NUM="${VERSION#v}"; \
    fi; \
    \
    if [[ -n $(git status -s) ]]; then \
        echo "❌ Working directory is not clean. Please commit or stash changes first."; \
        exit 1; \
    fi; \
    \
    TODAY=$(date +%Y-%m-%d); \
    echo "🤖 Starting Claude agent to update version and create git commit..."; \
    claude --auto-edit "Please perform the following release tasks for version $VERSION_NUM with release name '{{release_name}}': 1) Update the version constant in internal/cmd/root.go to $VERSION_NUM. 2) Update CHANGELOG.md by moving content from [Unreleased] section to a new [$VERSION_NUM] - {{release_name}} section with today's date ($TODAY), and ensure the version comparison links at the bottom are updated. The format should be: ## [$VERSION_NUM] - {{release_name}} - $TODAY. 3) Create a git commit with message 'chore: bump version to $VERSION_NUM'. 4) Create a git tag $VERSION with the message '$VERSION - {{release_name}}'. 5) Push both the commit and tag to origin."; \
    \
    echo "✅ Version updated and tagged by Claude agent"; \
    \
    echo "⏳ Waiting 60 seconds for GitHub Actions to build binaries..."; \
    sleep 60; \
    \
    echo "📦 Downloading binaries and calculating checksums..."; \
    mkdir -p /tmp/wee-release; \
    cd /tmp/wee-release; \
    \
    curl -sL "https://github.com/schlunsen/claude-templates-go/releases/download/$VERSION/wee-darwin-arm64" -o wee-darwin-arm64; \
    curl -sL "https://github.com/schlunsen/claude-templates-go/releases/download/$VERSION/wee-darwin-amd64" -o wee-darwin-amd64; \
    curl -sL "https://github.com/schlunsen/claude-templates-go/releases/download/$VERSION/wee-linux-arm64" -o wee-linux-arm64; \
    curl -sL "https://github.com/schlunsen/claude-templates-go/releases/download/$VERSION/wee-linux-amd64" -o wee-linux-amd64; \
    \
    SHA_DARWIN_ARM64=$(shasum -a 256 wee-darwin-arm64 | awk '{print $1}'); \
    SHA_DARWIN_AMD64=$(shasum -a 256 wee-darwin-amd64 | awk '{print $1}'); \
    SHA_LINUX_ARM64=$(shasum -a 256 wee-linux-arm64 | awk '{print $1}'); \
    SHA_LINUX_AMD64=$(shasum -a 256 wee-linux-amd64 | awk '{print $1}'); \
    \
    echo "✅ Checksums calculated"; \
    \
    echo "🍺 Updating Homebrew formula..."; \
    VERSION_NUM="${VERSION#v}"; \
    \
    cd "${WEE_HOMEBREW_REPO:-$HOME/projects/homebrew-wee}"; \
    \
    printf '%s\n' \
        'class Wee '"<"' Formula' \
        '  desc "High-performance CLI tool for Claude Code component templates and analytics"' \
        '  homepage "https://github.com/schlunsen/claude-templates-go"' \
        "  version \"$VERSION_NUM\"" \
        '' \
        '  # This is a precompiled binary, no build tools required' \
        '  uses_from_macos "unzip" '"=>"' :build' \
        '' \
        '  on_macos do' \
        '    if Hardware::CPU.arm?' \
        "      url \"https://github.com/schlunsen/claude-templates-go/releases/download/$VERSION/wee-darwin-arm64\"" \
        "      sha256 \"$SHA_DARWIN_ARM64\"" \
        '    else' \
        "      url \"https://github.com/schlunsen/claude-templates-go/releases/download/$VERSION/wee-darwin-amd64\"" \
        "      sha256 \"$SHA_DARWIN_AMD64\"" \
        '    end' \
        '  end' \
        '' \
        '  on_linux do' \
        '    if Hardware::CPU.arm?' \
        "      url \"https://github.com/schlunsen/claude-templates-go/releases/download/$VERSION/wee-linux-arm64\"" \
        "      sha256 \"$SHA_LINUX_ARM64\"" \
        '    else' \
        "      url \"https://github.com/schlunsen/claude-templates-go/releases/download/$VERSION/wee-linux-amd64\"" \
        "      sha256 \"$SHA_LINUX_AMD64\"" \
        '    end' \
        '  end' \
        '' \
        '  def install' \
        '    # The downloaded file is a precompiled binary' \
        '    downloaded_file = Dir["wee-*"].first' \
        '    bin.install downloaded_file '"=>"' "wee"' \
        '    chmod 0755, bin/"wee"' \
        '  end' \
        '' \
        '  test do' \
        '    system "#{bin}/wee", "--help"' \
        '  end' \
        'end' \
        > Formula/wee.rb; \
    \
    git add Formula/wee.rb; \
    git commit -m "chore: update wee formula to $VERSION"; \
    git push origin main; \
    \
    rm -rf /tmp/wee-release; \
    \
    echo ""; \
    echo "✅ Release $VERSION complete!"; \
    echo ""; \
    echo "📦 GitHub Release: https://github.com/schlunsen/claude-templates-go/releases/tag/$VERSION"; \
    echo ""; \
    echo "🍺 Homebrew users can upgrade with:"; \
    echo "   brew update && brew upgrade wee"; \
    echo ""; \
    echo "📝 Or force cache refresh:"; \
    echo "   brew untap schlunsen/wee && brew tap schlunsen/wee && brew install wee"; \
    echo "";

# Start server with ngrok tunnel (requires ngrok installed) — macOS/Linux only
[unix]
ngrok: build
    #!/usr/bin/env bash
    set -euo pipefail

    # Load .env file if it exists
    if [ -f .env ]; then
        export $(cat .env | grep -v '^#' | xargs)
    else
        echo "⚠️  .env file not found"
        echo ""
        echo "Please create a .env file from .env.example:"
        echo "  cp .env.example .env"
        echo ""
        echo "Then edit .env and set your NGROK_DOMAIN"
        exit 1
    fi

    # Check if NGROK_DOMAIN is set
    if [ -z "${NGROK_DOMAIN:-}" ]; then
        echo "❌ NGROK_DOMAIN not set in .env file"
        exit 1
    fi

    # Check if ngrok is installed
    if ! command -v ngrok &> /dev/null; then
        echo "❌ ngrok is not installed"
        echo ""
        echo "Install ngrok:"
        echo "  macOS: brew install ngrok"
        echo "  Linux: https://ngrok.com/download"
        echo "  Windows: https://ngrok.com/download"
        exit 1
    fi

    # Check if ngrok authtoken is configured
    if ! ngrok config check > /dev/null 2>&1; then
        echo "⚠️  ngrok authtoken not configured"
        echo ""
        echo "Set up your ngrok authtoken:"
        echo "  1. Get your token from: https://dashboard.ngrok.com/get-started/your-authtoken"
        echo "  2. Run: ngrok config add-authtoken YOUR_TOKEN"
        echo ""
        read -p "Continue anyway? (y/n) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    fi

    echo "🚀 Starting Wee with ngrok tunnel..."
    echo "📡 Using ngrok domain: $NGROK_DOMAIN"
    echo ""

    # Start ngrok and the Go server
    # ngrok will provide HTTPS, so we disable TLS on the Go server
    ngrok http --url="$NGROK_DOMAIN" 3333 &
    NGROK_PID=$!

    # Give ngrok a moment to start
    sleep 2

    # Set up a trap to kill ngrok when this script exits (including on Ctrl+C)
    # Also handle SIGTERM and SIGINT properly
    cleanup() {
        echo ""
        echo "🛑 Shutting down..."
        kill $NGROK_PID 2>/dev/null || true
        wait $NGROK_PID 2>/dev/null || true
        exit 0
    }
    trap cleanup EXIT SIGINT SIGTERM

    # Start the server with TLS disabled (ngrok provides HTTPS)
    WEE_DISABLE_TLS=true ./wee --analytics

# Update Homebrew formula only (use after manual release) — macOS/Linux only
[unix]
update-homebrew version:
    #!/usr/bin/env bash
    set -euo pipefail
    \
    echo "🍺 Updating Homebrew formula for v{{version}}..."; \
    \
    if [[ "{{version}}" != v* ]]; then \
        VERSION="v{{version}}"; \
    else \
        VERSION="{{version}}"; \
    fi; \
    \
    echo "📦 Downloading binaries and calculating checksums..."; \
    mkdir -p /tmp/wee-release; \
    cd /tmp/wee-release; \
    \
    curl -sL "https://github.com/schlunsen/claude-templates-go/releases/download/$VERSION/wee-darwin-arm64" -o wee-darwin-arm64; \
    curl -sL "https://github.com/schlunsen/claude-templates-go/releases/download/$VERSION/wee-darwin-amd64" -o wee-darwin-amd64; \
    curl -sL "https://github.com/schlunsen/claude-templates-go/releases/download/$VERSION/wee-linux-arm64" -o wee-linux-arm64; \
    curl -sL "https://github.com/schlunsen/claude-templates-go/releases/download/$VERSION/wee-linux-amd64" -o wee-linux-amd64; \
    \
    SHA_DARWIN_ARM64=$(shasum -a 256 wee-darwin-arm64 | awk '{print $1}'); \
    SHA_DARWIN_AMD64=$(shasum -a 256 wee-darwin-amd64 | awk '{print $1}'); \
    SHA_LINUX_ARM64=$(shasum -a 256 wee-linux-arm64 | awk '{print $1}'); \
    SHA_LINUX_AMD64=$(shasum -a 256 wee-linux-amd64 | awk '{print $1}'); \
    \
    echo "✅ Checksums calculated"; \
    \
    VERSION_NUM="${VERSION#v}"; \
    \
    cd "${WEE_HOMEBREW_REPO:-$HOME/projects/homebrew-wee}"; \
    \
    printf '%s\n' \
        'class Wee '"<"' Formula' \
        '  desc "High-performance CLI tool for Claude Code component templates and analytics"' \
        '  homepage "https://github.com/schlunsen/claude-templates-go"' \
        "  version \"$VERSION_NUM\"" \
        '' \
        '  # This is a precompiled binary, no build tools required' \
        '  uses_from_macos "unzip" '"=>"' :build' \
        '' \
        '  on_macos do' \
        '    if Hardware::CPU.arm?' \
        "      url \"https://github.com/schlunsen/claude-templates-go/releases/download/$VERSION/wee-darwin-arm64\"" \
        "      sha256 \"$SHA_DARWIN_ARM64\"" \
        '    else' \
        "      url \"https://github.com/schlunsen/claude-templates-go/releases/download/$VERSION/wee-darwin-amd64\"" \
        "      sha256 \"$SHA_DARWIN_AMD64\"" \
        '    end' \
        '  end' \
        '' \
        '  on_linux do' \
        '    if Hardware::CPU.arm?' \
        "      url \"https://github.com/schlunsen/claude-templates-go/releases/download/$VERSION/wee-linux-arm64\"" \
        "      sha256 \"$SHA_LINUX_ARM64\"" \
        '    else' \
        "      url \"https://github.com/schlunsen/claude-templates-go/releases/download/$VERSION/wee-linux-amd64\"" \
        "      sha256 \"$SHA_LINUX_AMD64\"" \
        '    end' \
        '  end' \
        '' \
        '  def install' \
        '    # The downloaded file is a precompiled binary' \
        '    downloaded_file = Dir["wee-*"].first' \
        '    bin.install downloaded_file '"=>"' "wee"' \
        '    chmod 0755, bin/"wee"' \
        '  end' \
        '' \
        '  test do' \
        '    system "#{bin}/wee", "--help"' \
        '  end' \
        'end' \
        > Formula/wee.rb; \
    \
    git add Formula/wee.rb; \
    git commit -m "chore: update wee formula to $VERSION"; \
    git push origin main; \
    \
    rm -rf /tmp/wee-release; \
    \
    echo ""; \
    echo "✅ Homebrew formula updated to $VERSION!"; \
    echo ""; \
    echo "🍺 Users can upgrade with:"; \
    echo "   brew update && brew upgrade wee"; \
    echo ""; \
    echo "📝 Or force cache refresh:"; \
    echo "   brew untap schlunsen/wee && brew tap schlunsen/wee && brew install wee"; \
    echo "";

# ──────────────────────────────────────────────────────
# Tauri Desktop App
# ──────────────────────────────────────────────────────

# Build the Tauri desktop app (.app + .dmg on macOS)
[unix]
tauri-build: build
    #!/usr/bin/env bash
    set -euo pipefail
    source "$HOME/.cargo/env" 2>/dev/null || true
    echo "🖥️  Building Tauri desktop app..."
    echo ""
    echo "📦 Preparing sidecar binary..."
    node src-tauri/scripts/build-sidecar.mjs
    echo ""
    echo "🦀 Building Tauri app (cargo tauri build)..."
    npm run tauri:build
    echo ""
    echo "📦 Copying build artifacts to dist/tauri/..."
    mkdir -p dist/tauri
    cp -R src-tauri/target/release/bundle/macos/*.app dist/tauri/
    cp src-tauri/target/release/bundle/dmg/*.dmg dist/tauri/
    echo ""
    echo "✅ Tauri build complete!"
    ls -lh dist/tauri/

# Run Tauri in development mode
[unix]
tauri-dev:
    #!/usr/bin/env bash
    source "$HOME/.cargo/env" 2>/dev/null || true
    npm run tauri:dev

# Install Tauri prerequisites
[unix]
tauri-setup:
    @bash src-tauri/scripts/setup.sh

# Deploy to VPS (build locally, deploy source, build on VPS) — macOS/Linux only
[unix]
vps-deploy:
    #!/usr/bin/env bash
    set -euo pipefail

    VPS_HOST="root@wee.cat"
    VPS_PATH="/root/wee"
    LOCAL_PATH="$HOME/projects/wee"
    BINARY_NAME="wee"

    # Get current branch name
    cd "$LOCAL_PATH"
    CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)

    echo "🚀 Deploying to VPS..."
    echo "   From: $LOCAL_PATH"
    echo "   Branch: $CURRENT_BRANCH"
    echo "   To:   $VPS_HOST:$VPS_PATH"
    echo ""

    # Build frontend locally first
    cd "$LOCAL_PATH"
    echo "📦 Building frontend locally..."
    cd internal/server/frontend && npm run generate
    cd "$LOCAL_PATH"

    echo ""
    echo "📤 Syncing source code to VPS..."

    # Create directory on VPS if it doesn't exist
    ssh "$VPS_HOST" "mkdir -p $VPS_PATH"

    # Sync only git-tracked files (excluding .git, node_modules, build artifacts)
    git ls-files -z | rsync --files-from=- --from0 -avz --delete \
        "$LOCAL_PATH/" "$VPS_HOST:$VPS_PATH/"

    echo ""
    echo "📤 Syncing Nuxt build output (.output) to VPS..."

    # Sync the Nuxt build output directory
    if [ -d "$LOCAL_PATH/internal/server/frontend/.output" ]; then
        rsync -avz --delete \
            "$LOCAL_PATH/internal/server/frontend/.output/" \
            "$VPS_HOST:$VPS_PATH/internal/server/frontend/.output/"
        echo "✅ Nuxt build output synced"
    else
        echo "⚠️  .output directory not found (may not have been built)"
    fi

    echo ""
    echo "🔨 Building on VPS (with CGO enabled for SQLite)..."

    # Build on VPS with CGO_ENABLED=1 for sqlite3 support
    # Use -buildvcs=false since .git wasn't synced to VPS
    ssh "$VPS_HOST" "cd $VPS_PATH && CGO_ENABLED=1 go build -buildvcs=false -o $BINARY_NAME ./cmd/wee"

    echo ""
    echo "✅ Deployment complete!"
    echo "   Binary: $VPS_PATH/$BINARY_NAME"
    echo "   Run: ssh $VPS_HOST '$VPS_PATH/$BINARY_NAME --analytics'"
