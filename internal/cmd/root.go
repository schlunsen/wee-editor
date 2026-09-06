package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/schlunsen/wee-editor/internal/docker"
	"github.com/schlunsen/wee-editor/internal/installer"
	"github.com/schlunsen/wee-editor/internal/server"
	"github.com/schlunsen/wee-editor/internal/version"
	"github.com/spf13/cobra"
)

// Use version constants from version package
var (
	Version = version.Version
	Name    = version.Name
)

var (
	// Global flags
	verbose   bool
	directory string
	yesFlag   bool
	dryRun    bool

	// Docker flags
	dockerCommand string
	dockerInit    bool
	dockerBuild   bool
	dockerRun     bool
	dockerStop    bool
	dockerLogs    bool
	dockerCompose bool
	dockerType    string
	dockerMCPs    string

	// Claude installer flag
	installClaude bool

	// Tunnel flag
	tunnelFlag bool
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "wee",
	Short: "Wee - Control center for Claude Code",
	Long: `Wee - A powerful wrapper and control center for Claude Code.

🎮 Launch Claude, run analytics, and deploy with Docker
📊 Real-time analytics dashboard with WebSocket monitoring
🐳 Docker support for containerized Claude environments`,
	Version: Version,
	Run: func(cmd *cobra.Command, args []string) {
		// Handle Claude installation first
		if installClaude {
			if err := installer.InstallInteractive(); err != nil {
				ShowError(fmt.Sprintf("Installation failed: %v", err))
				os.Exit(1)
			}
			return
		}

		// Check if this is the default mode (no flags provided)
		isDefaultMode := !dockerInit && !dockerBuild && !dockerRun && !dockerStop && !dockerLogs && !dockerCompose &&
			!installClaude

		// Launch analytics by default, respecting the verbose flag.
		if isDefaultMode {
			spinner := ShowSpinner("Launching Analytics Dashboard...")

			srv := createAnalyticsServer(directory)

			spinner.Success("Analytics Dashboard starting!")
			ShowInfo("Press Ctrl+C to stop")

			if err := srv.Setup(); err != nil {
				ShowError(fmt.Sprintf("Failed to setup server: %v", err))
				return
			}

			// Set up signal handler for graceful shutdown
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

			// Start server in a goroutine
			errChan := make(chan error, 1)
			go func() {
				errChan <- srv.Start()
			}()

			// Give the server a moment to start up, then open the browser
			// Skip browser open when running as a Tauri sidecar (WEE_NO_BROWSER=1)
			if os.Getenv("WEE_NO_BROWSER") != "1" {
				go func() {
					time.Sleep(1 * time.Second) // Wait for server to be ready
					openBrowser("https://localhost:3333")
				}()
			}

			// Wait for either a signal or server error
			select {
			case err := <-errChan:
				if err != nil {
					ShowError(fmt.Sprintf("Server error: %v", err))
				}
			case sig := <-sigChan:
				fmt.Printf("\n🛑 Received signal: %v\n", sig)

				// Create a channel to track shutdown completion
				shutdownDone := make(chan struct{})
				go func() {
					if err := srv.Shutdown(); err != nil && !srv.IsQuiet() {
						ShowError(fmt.Sprintf("Error during shutdown: %v", err))
					} else if !srv.IsQuiet() {
						ShowSuccess("Server shut down gracefully")
					}
					close(shutdownDone)
				}()

				// Wait for shutdown to complete with a timeout
				shutdownTimeout := time.NewTimer(10 * time.Second)
				defer shutdownTimeout.Stop()

				select {
				case <-shutdownDone:
					// Shutdown completed successfully
				case <-shutdownTimeout.C:
					ShowError("Server shutdown timeout - forcing exit")
				}

				// Wait for server goroutine to exit (should be quick after Shutdown)
				select {
				case <-errChan:
					// Server goroutine exited
				case <-time.After(2 * time.Second):
					// Force exit if server doesn't stop quickly
					ShowError("Server did not stop in time - forcing exit")
				}
			}
			return
		}

		// Handle different modes
		handleCommand(cmd, args)
	},
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose logging")
	rootCmd.PersistentFlags().StringVarP(&directory, "directory", "d", ".", "target directory")
	rootCmd.PersistentFlags().BoolVarP(&yesFlag, "yes", "y", false, "skip prompts and use defaults")
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be copied without copying")

	// Docker flags
	rootCmd.Flags().BoolVar(&dockerInit, "docker-init", false, "initialize Docker files (Dockerfile, .dockerignore)")
	rootCmd.Flags().BoolVar(&dockerBuild, "docker-build", false, "build Docker image")
	rootCmd.Flags().BoolVar(&dockerRun, "docker-run", false, "run Docker container")
	rootCmd.Flags().BoolVar(&dockerStop, "docker-stop", false, "stop Docker container")
	rootCmd.Flags().BoolVar(&dockerLogs, "docker-logs", false, "view Docker container logs")
	rootCmd.Flags().BoolVar(&dockerCompose, "docker-compose", false, "generate docker-compose.yml")
	rootCmd.Flags().StringVar(&dockerType, "docker-type", "claude", "Docker type: base, claude, analytics, full")
	rootCmd.Flags().StringVar(&dockerMCPs, "docker-mcps", "", "MCPs to include (comma-separated)")
	rootCmd.Flags().StringVar(&dockerCommand, "docker-command", "", "command to run in container")

	// Claude installer flag
	rootCmd.Flags().BoolVar(&installClaude, "install-claude", false, "install Claude CLI automatically")

	// Tunnel flag
	rootCmd.Flags().BoolVar(&tunnelFlag, "tunnel", false, "enable ngrok tunnel for public access")
}

func handleCommand(cmd *cobra.Command, args []string) {
	// Docker commands
	if dockerInit || dockerBuild || dockerRun || dockerStop || dockerLogs || dockerCompose {
		handleDockerCommands(directory)
		return
	}

	// If we get here, show help
	ShowError("No valid command provided. Use --help to see available options.")
}

// parseComponentList parses comma-separated component names
func parseComponentList(input string) []string {
	if input == "" {
		return nil
	}

	parts := strings.Split(input, ",")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

// loadEnvFile loads environment variables from a .env file if it exists.
// It does NOT override already-set environment variables.
func loadEnvFile() {
	data, err := os.ReadFile(".env")
	if err != nil {
		return // No .env file, that's fine
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		// Don't override existing env vars
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}
}

// createAnalyticsServer creates an analytics server instance
func createAnalyticsServer(targetDir string) *server.Server {
	// Load .env file (won't override existing env vars)
	loadEnvFile()

	// Get Claude directory (default to ~/.claude)
	claudeDir := filepath.Join(os.Getenv("HOME"), ".claude")

	// Check if custom directory specified
	if targetDir != "." && targetDir != "" {
		claudeDir = filepath.Join(targetDir, ".claude")
	}

	// Get port from environment variable (default to 3333)
	port := 3333
	if portStr := os.Getenv("WEE_PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil && p > 0 && p < 65536 {
			port = p
		}
	}

	// Create server with verbose flag from CLI
	srv := server.NewServerWithOptions(claudeDir, port, false, verbose)

	// Disable TLS if WEE_DISABLE_TLS is set
	if os.Getenv("WEE_DISABLE_TLS") == "true" {
		srv.DisableTLS()
	}

	// Enable tunnel if --tunnel flag is set
	if tunnelFlag {
		os.Setenv("WEE_TUNNEL_ENABLED", "true")
	}

	return srv
}

// handleDockerCommands handles all Docker-related operations
func handleDockerCommands(targetDir string) {
	dm := docker.NewDockerManager(targetDir)

	// Check if Docker is available
	if !dm.IsDockerAvailable() {
		ShowError("Docker is not installed or not running")
		ShowInfo("Please install Docker: https://docs.docker.com/get-docker/")
		return
	}

	// Parse MCPs list if provided
	mcpsList := parseComponentList(dockerMCPs)

	// Docker init - generate Dockerfile and .dockerignore
	if dockerInit {
		fmt.Println("🐳 Initializing Docker files...")

		generator := docker.NewDockerfileGenerator(targetDir)

		// Parse docker type
		var dockerfileType docker.DockerfileType
		switch dockerType {
		case "base":
			dockerfileType = docker.DockerfileBase
		case "claude":
			dockerfileType = docker.DockerfileClaude
		case "analytics":
			dockerfileType = docker.DockerfileAnalytics
		case "full":
			dockerfileType = docker.DockerfileFull
		default:
			dockerfileType = docker.DockerfileClaude
		}

		// Generate Dockerfile
		dockerfilePath := filepath.Join(targetDir, "Dockerfile")
		if err := generator.GenerateDockerfile(dockerfileType, dockerfilePath, mcpsList); err != nil {
			ShowError(fmt.Sprintf("Failed to generate Dockerfile: %v", err))
			return
		}

		// Generate .dockerignore
		dockerignorePath := filepath.Join(targetDir, ".dockerignore")
		if err := generator.GenerateDockerIgnore(dockerignorePath); err != nil {
			ShowError(fmt.Sprintf("Failed to generate .dockerignore: %v", err))
			return
		}

		ShowSuccess("Docker files generated successfully!")
		ShowInfo(fmt.Sprintf("Dockerfile: %s", dockerfilePath))
		ShowInfo(fmt.Sprintf(".dockerignore: %s", dockerignorePath))
		return
	}

	// Docker compose - generate docker-compose.yml
	if dockerCompose {
		fmt.Println("🐳 Generating docker-compose.yml...")

		generator := docker.NewComposeGenerator(targetDir)

		// Parse compose template
		var composeTemplate docker.ComposeTemplate
		switch dockerType {
		case "simple":
			composeTemplate = docker.ComposeSimple
		case "analytics":
			composeTemplate = docker.ComposeAnalytics
		case "database":
			composeTemplate = docker.ComposeDatabase
		case "full":
			composeTemplate = docker.ComposeFull
		default:
			composeTemplate = docker.ComposeSimple
		}

		// Generate docker-compose.yml
		composePath := filepath.Join(targetDir, "docker-compose.yml")
		if err := generator.GenerateCompose(composeTemplate, composePath, mcpsList); err != nil {
			ShowError(fmt.Sprintf("Failed to generate docker-compose.yml: %v", err))
			return
		}

		// Generate .env.example
		envPath := filepath.Join(targetDir, ".env.example")
		if err := generator.GenerateEnvFile(envPath); err != nil {
			ShowError(fmt.Sprintf("Failed to generate .env.example: %v", err))
			return
		}

		ShowSuccess("Docker Compose files generated successfully!")
		ShowInfo(fmt.Sprintf("docker-compose.yml: %s", composePath))
		ShowInfo(fmt.Sprintf(".env.example: %s", envPath))
		ShowInfo("Copy .env.example to .env and configure your environment variables")
		return
	}

	// Docker build
	if dockerBuild {
		dockerfilePath := filepath.Join(targetDir, "Dockerfile")
		if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
			ShowError("Dockerfile not found. Run with --docker-init first")
			return
		}

		// Build the wee binary first
		ShowInfo("Building wee binary for Docker image...")
		buildCmd := exec.Command("make", "build")
		buildCmd.Dir = targetDir
		buildCmd.Stdout = os.Stdout
		buildCmd.Stderr = os.Stderr

		if err := buildCmd.Run(); err != nil {
			ShowError(fmt.Sprintf("Failed to build wee binary: %v", err))
			ShowInfo("Please ensure you have Go installed and run 'make build' manually")
			return
		}

		// Check if binary exists in target directory
		binaryPath := filepath.Join(targetDir, "wee")
		if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
			ShowError("wee binary not found after build. Please run 'make build' manually")
			return
		}

		ShowSuccess("wee binary built successfully!")

		if err := dm.BuildImage(dockerfilePath); err != nil {
			ShowError(fmt.Sprintf("Failed to build Docker image: %v", err))
			return
		}
		return
	}

	// Docker run
	if dockerRun {
		opts := docker.NewRunOptions()

		// Default port mapping for analytics
		opts.Ports[3333] = 3333

		// Mount current directory
		absPath, _ := filepath.Abs(targetDir)
		opts.Volumes[absPath] = "/workspace"

		// Mount .claude directory
		claudeDir := filepath.Join(os.Getenv("HOME"), ".claude")
		opts.Volumes[claudeDir] = "/root/.claude"

		// Set command if provided
		if dockerCommand != "" {
			opts.Command = dockerCommand
		}

		if err := dm.RunContainer(opts); err != nil {
			ShowError(fmt.Sprintf("Failed to run Docker container: %v", err))
			return
		}

		ShowInfo("To view logs: wee --docker-logs")
		ShowInfo("To stop container: wee --docker-stop")
		return
	}

	// Docker stop
	if dockerStop {
		if err := dm.StopContainer(); err != nil {
			ShowError(fmt.Sprintf("Failed to stop Docker container: %v", err))
			return
		}
		ShowSuccess("Docker container stopped successfully!")
		return
	}

	// Docker logs
	if dockerLogs {
		fmt.Println("📋 Docker container logs:")
		if err := dm.GetContainerLogs(false); err != nil {
			ShowError(fmt.Sprintf("Failed to get logs: %v", err))
			return
		}
		return
	}
}

// openBrowser attempts to open a URL in the default browser
func openBrowser(url string) {
	var cmd *exec.Cmd

	// Detect the operating system and use the appropriate command
	switch runtime.GOOS {
	case "windows":
		// Windows
		cmd = exec.Command("cmd", "/c", "start", url)
	case "darwin":
		// macOS
		cmd = exec.Command("open", url)
	default:
		// Linux and other Unix-like systems
		cmd = exec.Command("xdg-open", url)
	}

	// Execute the command, but don't block or error out if it fails
	if err := cmd.Run(); err != nil {
		// Silently fail - this is not a critical error
		// The user can still access the dashboard manually
	}
}
