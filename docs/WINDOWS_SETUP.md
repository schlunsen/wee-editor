# Wee Windows Setup Guide

Hey Josep! This guide will get **Wee** running on your Windows machine. Wee is a web dashboard for running and monitoring AI agent sessions — it's a single Go binary with a built-in web frontend.

## What You're Building

Wee is made of two parts that get compiled into **one single file** (`wee.exe`):

1. **Backend** — written in Go (a programming language), handles the server, API, and database
2. **Frontend** — a web UI written in Vue/TypeScript, gets built into static files and embedded into the Go binary

The final binary is ~15MB and has zero external dependencies at runtime.

---

## Step 1: Install Prerequisites

Open **PowerShell as Administrator** (right-click PowerShell → "Run as administrator") and install everything using `winget` (Windows' built-in package manager):

```powershell
# Git — for cloning the repository
winget install Git.Git

# Go — the programming language the backend is written in
winget install GoLang.Go

# Node.js — needed to build the frontend (LTS version)
winget install OpenJS.NodeJS.LTS

# Just — a task runner we use to build the project (like a simple script launcher)
winget install Casey.Just

# TDM-GCC — a C compiler needed because our database library (SQLite) requires one
winget install jmeubank.TDMGcc
```

> **Why do I need a C compiler?** The SQLite database driver (`go-sqlite3`) is written in C and needs to be compiled when building. TDM-GCC provides this on Windows.

After installing, **close and reopen PowerShell** so it picks up the new programs in your PATH.

### Verify everything installed correctly

```powershell
git --version        # Should show: git version 2.x.x
go version           # Should show: go version go1.25.x (or higher)
node --version       # Should show: v20.x.x or v22.x.x
just --version       # Should show: just 1.x.x
gcc --version        # Should show: gcc (tdm64-...)
```

If any command says "not recognized", close PowerShell and reopen it. If it still doesn't work, you may need to log out and back in (or restart) for PATH changes to take effect.

---

## Step 2: Clone the Repository

Pick a folder where you want the project to live. For example your home folder:

```powershell
cd ~
git clone https://github.com/schlunsen/wee-editor.git
cd wee-editor
```

---

## Step 3: Build Wee

The project uses **`just`** as a task runner. The justfile is cross-platform — it automatically detects Windows and uses PowerShell. From the project root, simply run:

```powershell
just build
```

This single command will:
1. Install frontend npm dependencies (if not already installed)
2. Build the Nuxt frontend into static files
3. Compile the Go binary as `wee.exe` (with CGO enabled for SQLite)

That's it! You should see `wee.exe` in the project root.

### Other useful just commands

```powershell
just --list          # Show all available commands
just build           # Full build (frontend + Go binary)
just build-frontend  # Build only the frontend
just build-go        # Build only the Go binary (faster, skip if frontend unchanged)
just analytics       # Build and run the web dashboard directly
just test            # Run the test suite
just clean           # Remove build artifacts
just deps            # Download/update Go dependencies
just fmt             # Format Go code
```

> **Note:** Some commands like `just release`, `just ngrok`, and `just vps-deploy` are macOS/Linux only and won't appear on Windows. You don't need them.

---

## Step 4: Run Wee

```powershell
.\wee.exe --analytics
```

This starts the web dashboard. It will:

1. Create a config folder at `~\.claude\wee\` (database, TLS certs, etc.)
2. Generate self-signed HTTPS certificates
3. Start the server on **https://localhost:3333**
4. Automatically open your browser

> **Browser warning:** Since the HTTPS certificate is self-signed, your browser will show a security warning. Click "Advanced" → "Proceed to localhost" — this is expected and safe for local use.

### First-time setup

When you first open Wee in the browser, you'll be asked to create an admin account. After that you can:

- Add your **Anthropic API key** (or other provider keys) in the Settings page
- Create **projects** to organize your work
- Start **agent sessions** to chat with Claude

---

## Rebuilding After Updates

When I push updates, pull and rebuild:

```powershell
cd ~/wee-editor
git pull
just build
.\wee.exe --analytics
```

### Quick rebuild (backend-only changes)

If I tell you only Go files changed (no frontend), you can skip the frontend build:

```powershell
cd ~/wee-editor
git pull
just build-go
.\wee.exe --analytics
```

---

## Troubleshooting

### "gcc not found" or CGO errors
Make sure TDM-GCC is installed and in your PATH:
```powershell
gcc --version
```
If not found, reinstall with `winget install jmeubank.TDMGcc` and restart PowerShell.

### "go: command not found"
Close and reopen PowerShell. If still not found:
```powershell
# Check if Go is installed but not in PATH
Test-Path "C:\Program Files\Go\bin\go.exe"

# If True, add it to your PATH for this session:
$env:PATH += ";C:\Program Files\Go\bin"
```

### "node: command not found"
Same as above — restart PowerShell. Node typically installs to `C:\Program Files\nodejs\`.

### Frontend build fails with memory errors
Node.js may need more memory for the build:
```powershell
$env:NODE_OPTIONS = "--max-old-space-size=4096"
just build-frontend
```

### Port 3333 already in use
Something else is using port 3333. Find and kill it:
```powershell
netstat -ano | findstr :3333
# Note the PID (last column), then:
taskkill /PID <the-pid> /F
```

### The page loads but nothing works / blank screen
Try clearing the browser cache or opening in an incognito window. The self-signed cert can sometimes cause issues with cached resources.

---

## Quick Reference

| What | Command |
|------|---------|
| Full build | `just build` |
| Backend-only build | `just build-go` |
| Start Wee | `.\wee.exe --analytics` |
| Config location | `~\.claude\wee\` |
| Database | `~\.claude\wee\wee.db` |
| Web URL | https://localhost:3333 |
| Stop Wee | Press `Ctrl+C` in the PowerShell window |
| Check version | `.\wee.exe --version` |
| Run tests | `just test` |
| See all commands | `just --list` |
