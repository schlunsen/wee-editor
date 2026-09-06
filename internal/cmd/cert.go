package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pterm/pterm"
	"github.com/schlunsen/wee-editor/internal/server"
	"github.com/spf13/cobra"
)

var (
	certRegenerate bool
	certInfo       bool
	certMkcert     bool
)

// certCmd represents the cert command
var certCmd = &cobra.Command{
	Use:   "cert",
	Short: "Manage TLS certificates for analytics server",
	Long: `Manage TLS certificates used by the analytics server.

Wee supports two types of certificates:
1. mkcert - Locally-trusted development certificates (recommended for PWA)
2. self-signed - Basic HTTPS support (requires browser security warnings)

Examples:
  # Show certificate information
  wee cert --info

  # Regenerate certificates (auto-detects mkcert)
  wee cert --regenerate

  # Force mkcert certificate generation
  wee cert --regenerate --mkcert

  # Install mkcert and generate certificates
  brew install mkcert && mkcert -install
  wee cert --regenerate --mkcert`,
	RunE: handleCertCommand,
}

func init() {
	rootCmd.AddCommand(certCmd)

	certCmd.Flags().BoolVar(&certInfo, "info", false, "Show certificate information")
	certCmd.Flags().BoolVar(&certRegenerate, "regenerate", false, "Regenerate certificates")
	certCmd.Flags().BoolVar(&certMkcert, "mkcert", false, "Force mkcert certificate generation (requires mkcert installed)")
}

func handleCertCommand(cmd *cobra.Command, args []string) error {
	// Get Claude directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	claudeDir := filepath.Join(homeDir, ".claude")
	certManager := server.NewCertificateManager(claudeDir)

	// Handle --info flag
	if certInfo {
		return showCertificateInfo(certManager)
	}

	// Handle --regenerate flag
	if certRegenerate {
		return regenerateCertificates(certManager, certMkcert)
	}

	// Default: show info if no flags provided
	return showCertificateInfo(certManager)
}

func showCertificateInfo(certManager *server.CertificateManager) error {
	info, err := certManager.GetCertificateInfo()
	if err != nil {
		return fmt.Errorf("failed to get certificate info: %w", err)
	}

	pterm.DefaultHeader.Println("TLS Certificate Information")
	pterm.Println()

	// Certificate type
	certType := info["type"].(string)
	if certType == "none" {
		pterm.Warning.Println("No certificates found")
		pterm.Info.Println("Run 'wee --analytics' to generate certificates")
		return nil
	}

	// Show certificate details
	if certType == "mkcert" {
		pterm.Success.Println("✓ Certificate Type: mkcert (trusted)")
		pterm.Info.Printfln("  Certificate: %s", info["cert_path"])
		pterm.Info.Printfln("  Private Key: %s", info["key_path"])
	} else if certType == "self-signed" {
		pterm.Warning.Println("⚠ Certificate Type: self-signed (not trusted)")
		pterm.Info.Printfln("  Certificate: %s", info["cert_path"])
		pterm.Info.Printfln("  Private Key: %s", info["key_path"])
	}

	// Validity
	if info["valid"].(bool) {
		daysUntilExpiry := info["days_until_expiry"].(int)
		if daysUntilExpiry < 30 {
			pterm.Warning.Printfln("⚠ Expires in: %d days", daysUntilExpiry)
			pterm.Info.Println("  Consider regenerating: wee cert --regenerate")
		} else {
			pterm.Success.Printfln("✓ Expires in: %d days", daysUntilExpiry)
		}
	} else {
		pterm.Error.Println("✗ Certificate is invalid or expired")
		pterm.Info.Println("  Run: wee cert --regenerate")
	}

	pterm.Println()

	// mkcert status
	mkcertInstalled := info["mkcert_installed"].(bool)
	if mkcertInstalled {
		pterm.Success.Println("✓ mkcert is installed")
		if certType != "mkcert" {
			pterm.Info.Println("  You can generate trusted certificates with: wee cert --regenerate --mkcert")
		}
	} else {
		pterm.Info.Println("ℹ mkcert is not installed")
		pterm.Println()
		pterm.Info.Println("For PWA support with trusted certificates, install mkcert:")
		pterm.Info.Println("  macOS:   brew install mkcert && mkcert -install")
		pterm.Info.Println("  Linux:   apt install mkcert && mkcert -install")
		pterm.Info.Println("  Windows: choco install mkcert && mkcert -install")
		pterm.Println()
		pterm.Info.Println("Then run: wee cert --regenerate --mkcert")
	}

	pterm.Println()

	// PWA compatibility
	if certType == "mkcert" && info["valid"].(bool) {
		pterm.Success.Println("✓ PWA Installation: Ready")
		pterm.Info.Println("  Your browser will trust the certificate without warnings")
	} else if certType == "self-signed" {
		pterm.Warning.Println("⚠ PWA Installation: Limited")
		pterm.Info.Println("  Browser will show security warnings")
		pterm.Info.Println("  Install mkcert for better PWA support")
	}

	return nil
}

func regenerateCertificates(certManager *server.CertificateManager, useMkcert bool) error {
	pterm.Info.Println("Regenerating TLS certificates...")
	pterm.Println()

	// Auto-detect mkcert if not explicitly requested
	if !useMkcert {
		info, _ := certManager.GetCertificateInfo()
		if info != nil && info["mkcert_installed"].(bool) {
			pterm.Info.Println("mkcert detected - will generate trusted certificates")
			useMkcert = true
		}
	}

	// Regenerate
	if err := certManager.RegenerateCertificates(useMkcert); err != nil {
		return fmt.Errorf("certificate regeneration failed: %w", err)
	}

	pterm.Println()

	// Show new certificate info
	return showCertificateInfo(certManager)
}
