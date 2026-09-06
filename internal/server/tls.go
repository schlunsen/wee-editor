package server

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/pterm/pterm"
	"github.com/schlunsen/wee-editor/internal/logging"
)

// TLSConfig holds the TLS certificate and key paths
type TLSConfig struct {
	CertPath  string
	KeyPath   string
	Enabled   bool
	CertType  string // "mkcert", "self-signed"
	IsTrusted bool   // Whether cert is trusted by system
}

// CertificateManager handles certificate generation and validation
type CertificateManager struct {
	certsDir   string
	certFile   string
	keyFile    string
	mkcertCert string
	mkcertKey  string
	quiet      bool
}

// NewCertificateManager creates a new certificate manager
func NewCertificateManager(claudeDir string) *CertificateManager {
	certsDir := filepath.Join(claudeDir, "wee", "certs")
	return &CertificateManager{
		certsDir:   certsDir,
		certFile:   filepath.Join(certsDir, "server.crt"),
		keyFile:    filepath.Join(certsDir, "server.key"),
		mkcertCert: filepath.Join(certsDir, "localhost.pem"),
		mkcertKey:  filepath.Join(certsDir, "localhost-key.pem"),
		quiet:      false,
	}
}

// SetQuiet sets whether to suppress output messages
func (cm *CertificateManager) SetQuiet(quiet bool) {
	cm.quiet = quiet
}

// EnsureCertificates checks if certificates exist and are valid, generates new ones if needed
func (cm *CertificateManager) EnsureCertificates() (*TLSConfig, error) {
	// Create certs directory if it doesn't exist
	if err := os.MkdirAll(cm.certsDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create certs directory: %w", err)
	}

	// Strategy 1: Check for existing mkcert certificates
	if cm.hasMkcertCerts() {
		if valid, daysUntilExpiry := cm.validateCertificateFile(cm.mkcertCert); valid {
			if daysUntilExpiry < 30 && !cm.quiet {
				pterm.Warning.Printf("mkcert certificate expires in %d days. Consider regenerating with 'wee cert --regenerate'\n", daysUntilExpiry)
			}
			if !cm.quiet {
				pterm.Success.Println("Using existing mkcert certificates (trusted)")
			}
			logging.Info("Using existing mkcert certificates (expires in %d days)", daysUntilExpiry)
			return &TLSConfig{
				CertPath:  cm.mkcertCert,
				KeyPath:   cm.mkcertKey,
				Enabled:   true,
				CertType:  "mkcert",
				IsTrusted: true,
			}, nil
		}
		logging.Info("Existing mkcert certificate is invalid or expired")
	}

	// Strategy 2: Try to generate mkcert certificates
	if cm.isMkcertInstalled() {
		logging.Info("mkcert detected - generating trusted certificates")
		if err := cm.generateMkcertCert(); err != nil {
			logging.Warning("Failed to generate mkcert certificate: %v", err)
			logging.Info("Falling back to self-signed certificate")
		} else {
			if !cm.quiet {
				pterm.Success.Println("Trusted mkcert certificates generated successfully")
			}
			logging.Info("Trusted mkcert certificates generated successfully")
			return &TLSConfig{
				CertPath:  cm.mkcertCert,
				KeyPath:   cm.mkcertKey,
				Enabled:   true,
				CertType:  "mkcert",
				IsTrusted: true,
			}, nil
		}
	} else {
		logging.Info("mkcert not installed, using self-signed certificates")
	}

	// Strategy 3: Check for existing self-signed certificates
	if cm.hasSelfSignedCerts() {
		if valid, daysUntilExpiry := cm.validateCertificateFile(cm.certFile); valid {
			if daysUntilExpiry < 30 && !cm.quiet {
				pterm.Warning.Printf("Self-signed certificate expires in %d days. Consider regenerating.\n", daysUntilExpiry)
			}
			logging.Info("Using existing self-signed certificate (expires in %d days)", daysUntilExpiry)
			return &TLSConfig{
				CertPath:  cm.certFile,
				KeyPath:   cm.keyFile,
				Enabled:   true,
				CertType:  "self-signed",
				IsTrusted: false,
			}, nil
		}
		logging.Info("Existing self-signed certificate is invalid or expired")
	}

	// Strategy 4: Generate new self-signed certificate
	logging.Info("Generating self-signed TLS certificate")
	if err := cm.generateSelfSignedCert(); err != nil {
		return nil, fmt.Errorf("failed to generate certificate: %w", err)
	}

	if !cm.quiet {
		pterm.Success.Println("Self-signed certificate generated successfully")
	}
	logging.Info("Self-signed certificate generated successfully")

	return &TLSConfig{
		CertPath:  cm.certFile,
		KeyPath:   cm.keyFile,
		Enabled:   true,
		CertType:  "self-signed",
		IsTrusted: false,
	}, nil
}

// generateSelfSignedCert creates a new self-signed certificate
func (cm *CertificateManager) generateSelfSignedCert() error {
	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}

	// Create certificate template
	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour) // Valid for 1 year

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return fmt.Errorf("failed to generate serial number: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Wee"},
			CommonName:   "localhost",
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}

	// Create certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %w", err)
	}

	// Write certificate to file
	certOut, err := os.OpenFile(cm.certFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to open cert file for writing: %w", err)
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return fmt.Errorf("failed to write certificate: %w", err)
	}

	// Write private key to file
	keyOut, err := os.OpenFile(cm.keyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to open key file for writing: %w", err)
	}
	defer keyOut.Close()

	privBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %w", err)
	}

	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	return nil
}

// isMkcertInstalled checks if mkcert is installed and available in PATH
func (cm *CertificateManager) isMkcertInstalled() bool {
	_, err := exec.LookPath("mkcert")
	return err == nil
}

// hasMkcertCerts checks if mkcert certificates exist
func (cm *CertificateManager) hasMkcertCerts() bool {
	return fileExists(cm.mkcertCert) && fileExists(cm.mkcertKey)
}

// hasSelfSignedCerts checks if self-signed certificates exist
func (cm *CertificateManager) hasSelfSignedCerts() bool {
	return fileExists(cm.certFile) && fileExists(cm.keyFile)
}

// generateMkcertCert generates certificates using mkcert
func (cm *CertificateManager) generateMkcertCert() error {
	// Check if mkcert CA is installed
	cmd := exec.Command("mkcert", "-CAROOT")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("mkcert not properly installed: %w", err)
	}

	caRoot := strings.TrimSpace(string(output))
	if caRoot == "" {
		return fmt.Errorf("mkcert CA root not found - run 'mkcert -install' first")
	}

	// Generate certificates for localhost
	cmd = exec.Command("mkcert",
		"-cert-file", cm.mkcertCert,
		"-key-file", cm.mkcertKey,
		"localhost",
		"127.0.0.1",
		"::1",
	)

	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mkcert generation failed: %w - output: %s", err, string(output))
	}

	// Set proper permissions
	if err := os.Chmod(cm.mkcertCert, 0600); err != nil {
		return fmt.Errorf("failed to set cert permissions: %w", err)
	}
	if err := os.Chmod(cm.mkcertKey, 0600); err != nil {
		return fmt.Errorf("failed to set key permissions: %w", err)
	}

	return nil
}

// RegenerateCertificates forces regeneration of certificates
func (cm *CertificateManager) RegenerateCertificates(useMkcert bool) error {
	// Remove existing certificates
	os.Remove(cm.mkcertCert)
	os.Remove(cm.mkcertKey)
	os.Remove(cm.certFile)
	os.Remove(cm.keyFile)

	pterm.Info.Println("Removed existing certificates")

	// Generate new certificates
	if useMkcert && cm.isMkcertInstalled() {
		pterm.Info.Println("Generating new mkcert certificates...")
		if err := cm.generateMkcertCert(); err != nil {
			return fmt.Errorf("failed to generate mkcert certificates: %w", err)
		}
		pterm.Success.Println("Trusted mkcert certificates generated successfully")
		return nil
	}

	pterm.Info.Println("Generating new self-signed certificates...")
	if err := cm.generateSelfSignedCert(); err != nil {
		return fmt.Errorf("failed to generate self-signed certificates: %w", err)
	}
	pterm.Success.Println("Self-signed certificates generated successfully")
	return nil
}

// GetCertificateInfo returns information about the current certificates
func (cm *CertificateManager) GetCertificateInfo() (map[string]interface{}, error) {
	info := make(map[string]interface{})

	// Check which certificates exist
	if cm.hasMkcertCerts() {
		valid, daysUntilExpiry := cm.validateCertificateFile(cm.mkcertCert)
		info["type"] = "mkcert"
		info["cert_path"] = cm.mkcertCert
		info["key_path"] = cm.mkcertKey
		info["valid"] = valid
		info["days_until_expiry"] = daysUntilExpiry
		info["trusted"] = true
	} else if cm.hasSelfSignedCerts() {
		valid, daysUntilExpiry := cm.validateCertificateFile(cm.certFile)
		info["type"] = "self-signed"
		info["cert_path"] = cm.certFile
		info["key_path"] = cm.keyFile
		info["valid"] = valid
		info["days_until_expiry"] = daysUntilExpiry
		info["trusted"] = false
	} else {
		info["type"] = "none"
		info["valid"] = false
	}

	info["mkcert_installed"] = cm.isMkcertInstalled()

	return info, nil
}

// validateCertificateFile checks if a certificate file is valid
func (cm *CertificateManager) validateCertificateFile(certPath string) (bool, int) {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return false, 0
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return false, 0
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return false, 0
	}

	// Check if certificate is expired or not yet valid
	now := time.Now()
	if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {
		return false, 0
	}

	// Calculate days until expiry
	daysUntilExpiry := int(cert.NotAfter.Sub(now).Hours() / 24)

	return true, daysUntilExpiry
}

// GetTLSConfig returns the TLS configuration
func (cm *CertificateManager) GetTLSConfig() *TLSConfig {
	return &TLSConfig{
		CertPath: cm.certFile,
		KeyPath:  cm.keyFile,
		Enabled:  true,
	}
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
