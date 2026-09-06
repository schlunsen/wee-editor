#!/bin/bash
set -e

echo "🔧 Wee Desktop — Setup"
echo ""

# Check Node.js
if ! command -v node &> /dev/null; then
    echo "❌ Node.js is required. Install from https://nodejs.org/"
    exit 1
fi

NODE_VERSION=$(node -v | sed 's/v//' | cut -d. -f1)
if [ "$NODE_VERSION" -lt 20 ]; then
    echo "❌ Node.js 20+ is required (found v$NODE_VERSION)"
    exit 1
fi
echo "✅ Node.js $(node -v)"

# Check Go
if ! command -v go &> /dev/null; then
    echo "❌ Go is required. Install from https://go.dev/dl/"
    exit 1
fi
echo "✅ Go $(go version | awk '{print $3}')"

# Check Rust
if ! command -v rustc &> /dev/null; then
    echo "📦 Installing Rust..."
    curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
    source "$HOME/.cargo/env"
fi
echo "✅ Rust $(rustc --version | awk '{print $2}')"

# Check Tauri CLI
if ! cargo tauri --version &> /dev/null 2>&1; then
    echo "📦 Installing Tauri CLI v2..."
    cargo install tauri-cli --version "^2"
fi
echo "✅ Tauri CLI $(cargo tauri --version 2>/dev/null || echo 'installed')"

# Platform-specific dependencies
case "$(uname -s)" in
    Darwin)
        if ! xcode-select -p &> /dev/null; then
            echo "📦 Installing Xcode command line tools..."
            xcode-select --install
        fi
        echo "✅ Xcode CLI tools"
        ;;
    Linux)
        echo "📦 Checking Linux dependencies..."
        DEPS="libwebkit2gtk-4.1-dev build-essential libssl-dev librsvg2-dev libxdo-dev libayatana-appindicator3-dev"
        MISSING=""
        for dep in $DEPS; do
            if ! dpkg -l "$dep" &> /dev/null 2>&1; then
                MISSING="$MISSING $dep"
            fi
        done
        if [ -n "$MISSING" ]; then
            echo "📦 Installing missing dependencies:$MISSING"
            sudo apt-get update && sudo apt-get install -y $MISSING
        fi
        echo "✅ Linux dependencies"
        ;;
esac

# Install npm dependencies
echo ""
echo "📦 Installing npm dependencies..."
cd "$(dirname "$0")/../.."
if [ -d "internal/server/frontend" ]; then
    cd internal/server/frontend && npm install && cd ../../..
fi

echo ""
echo "✅ Setup complete! You can now run:"
echo "   npm run tauri:dev     — Development mode"
echo "   npm run tauri:build   — Production build"
