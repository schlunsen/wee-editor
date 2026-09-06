package installer

import "fmt"

// InstallInteractive attempts to install the Claude CLI, prompting the user
// for confirmation and reporting progress on stdout.
func InstallInteractive() error {
	ci := NewClaudeInstaller()
	ci.Verbose = true

	fmt.Println("\n🚀 Claude CLI Installer")
	fmt.Println("========================")

	// Check if already installed (checks PATH and common locations)
	claudePath, err := FindClaudePath()
	if err == nil {
		// Already installed
		fmt.Printf("✓ Claude CLI is already installed\n")
		fmt.Printf("  Location: %s\n", claudePath)

		// Try to get version
		version, vErr := ci.GetClaudeVersion()
		if vErr == nil {
			fmt.Printf("  Version: %s\n", version)
		}
		return nil
	}

	// Show detection info
	nd := NewNodeDetector()
	fmt.Println(nd.FormatNodeInfo())
	fmt.Println()

	// Recommend installation method
	fmt.Println("Recommended: Native binary installation (no Node.js required)")
	fmt.Println()
	fmt.Print("Proceed with automatic installation? (y/n): ")

	var response string
	fmt.Scanln(&response)

	if response != "y" && response != "Y" {
		fmt.Println("\nInstallation cancelled.")
		fmt.Println(GetInstallInstructions())
		return fmt.Errorf("installation cancelled by user")
	}

	// Attempt auto-installation
	fmt.Println("\nInstalling Claude CLI...")
	result := ci.AutoInstall()

	if result.Success {
		fmt.Printf("\n✓ %s\n", result.Message)
		fmt.Printf("  Version: %s\n", result.Version)
		fmt.Printf("  Location: %s\n", result.ClaudePath)
		fmt.Println("\nRun 'claude doctor' to verify your installation.")
		return nil
	}

	// Installation might have succeeded but not in PATH yet
	// Check common locations as a fallback
	claudePath, pathErr := FindClaudePath()
	if pathErr == nil {
		fmt.Printf("\n✓ Claude CLI installed successfully!\n")
		fmt.Printf("  Location: %s\n", claudePath)
		fmt.Println("\nNote: Claude is installed but not in your PATH.")
		fmt.Println("The installer will still find it, but you may want to add ~/.local/bin to your PATH.")
		fmt.Println("\nRun 'claude doctor' to verify your installation.")
		return nil
	}

	// Installation failed
	fmt.Printf("\n✗ Installation failed: %v\n", result.Error)
	if result.Message != "" {
		fmt.Printf("  %s\n", result.Message)
	}
	fmt.Println("\nManual installation instructions:")
	fmt.Println(GetInstallInstructions())

	return result.Error
}
