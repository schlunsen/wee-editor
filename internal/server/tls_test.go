package server

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewCertificateManager(t *testing.T) {
	tempDir := t.TempDir()
	cm := NewCertificateManager(tempDir)

	if cm == nil {
		t.Fatal("NewCertificateManager returned nil")
	}

	expectedCertsDir := filepath.Join(tempDir, "wee", "certs")
	if cm.certsDir != expectedCertsDir {
		t.Errorf("Expected certsDir %s, got %s", expectedCertsDir, cm.certsDir)
	}

	expectedCertFile := filepath.Join(expectedCertsDir, "server.crt")
	if cm.certFile != expectedCertFile {
		t.Errorf("Expected certFile %s, got %s", expectedCertFile, cm.certFile)
	}

	expectedKeyFile := filepath.Join(expectedCertsDir, "server.key")
	if cm.keyFile != expectedKeyFile {
		t.Errorf("Expected keyFile %s, got %s", expectedKeyFile, cm.keyFile)
	}

	expectedMkcertCert := filepath.Join(expectedCertsDir, "localhost.pem")
	if cm.mkcertCert != expectedMkcertCert {
		t.Errorf("Expected mkcertCert %s, got %s", expectedMkcertCert, cm.mkcertCert)
	}

	expectedMkcertKey := filepath.Join(expectedCertsDir, "localhost-key.pem")
	if cm.mkcertKey != expectedMkcertKey {
		t.Errorf("Expected mkcertKey %s, got %s", expectedMkcertKey, cm.mkcertKey)
	}
}

func TestCertificateManager_EnsureCertificates_CreatesDirectory(t *testing.T) {
	tempDir := t.TempDir()
	cm := NewCertificateManager(tempDir)

	// Ensure certificates (will generate self-signed if mkcert unavailable)
	tlsConfig, err := cm.EnsureCertificates()
	if err != nil {
		t.Fatalf("EnsureCertificates failed: %v", err)
	}

	if tlsConfig == nil {
		t.Fatal("TLSConfig is nil")
	}

	if !tlsConfig.Enabled {
		t.Error("Expected TLS to be enabled")
	}

	// Check that certs directory was created
	if _, err := os.Stat(cm.certsDir); os.IsNotExist(err) {
		t.Errorf("Certs directory was not created: %s", cm.certsDir)
	}

	// Check that either mkcert or self-signed certificates exist
	mkcertExists := fileExists(cm.mkcertCert) && fileExists(cm.mkcertKey)
	selfSignedExists := fileExists(cm.certFile) && fileExists(cm.keyFile)

	if !mkcertExists && !selfSignedExists {
		t.Error("Neither mkcert nor self-signed certificates were created")
	}

	// Verify certificate type is set
	if tlsConfig.CertType != "mkcert" && tlsConfig.CertType != "self-signed" {
		t.Errorf("Unexpected certificate type: %s", tlsConfig.CertType)
	}

	// Verify trust status matches certificate type
	if tlsConfig.CertType == "mkcert" && !tlsConfig.IsTrusted {
		t.Error("mkcert certificate should be marked as trusted")
	}
	if tlsConfig.CertType == "self-signed" && tlsConfig.IsTrusted {
		t.Error("self-signed certificate should not be marked as trusted")
	}
}

func TestCertificateManager_GetCertificateInfo_NoCertificates(t *testing.T) {
	tempDir := t.TempDir()
	cm := NewCertificateManager(tempDir)

	info, err := cm.GetCertificateInfo()
	if err != nil {
		t.Fatalf("GetCertificateInfo failed: %v", err)
	}

	if info["type"] != "none" {
		t.Errorf("Expected type 'none' for non-existent certificates, got %v", info["type"])
	}

	if info["valid"] != false {
		t.Error("Expected valid=false for non-existent certificates")
	}
}

func TestCertificateManager_IsMkcertInstalled(t *testing.T) {
	tempDir := t.TempDir()
	cm := NewCertificateManager(tempDir)

	// Just verify the method doesn't panic
	installed := cm.isMkcertInstalled()

	// We can't guarantee mkcert is installed in test environment,
	// but we can verify the method returns a boolean
	if installed != true && installed != false {
		t.Error("isMkcertInstalled should return a boolean")
	}
}

func TestCertificateManager_HasMkcertCerts(t *testing.T) {
	tempDir := t.TempDir()
	cm := NewCertificateManager(tempDir)

	// Initially should not have mkcert certs
	if cm.hasMkcertCerts() {
		t.Error("Should not have mkcert certs initially")
	}

	// Create certs directory
	os.MkdirAll(cm.certsDir, 0700)

	// Create mock mkcert cert files
	os.WriteFile(cm.mkcertCert, []byte("mock cert"), 0600)
	os.WriteFile(cm.mkcertKey, []byte("mock key"), 0600)

	// Now should have mkcert certs
	if !cm.hasMkcertCerts() {
		t.Error("Should have mkcert certs after creating files")
	}
}

func TestCertificateManager_HasSelfSignedCerts(t *testing.T) {
	tempDir := t.TempDir()
	cm := NewCertificateManager(tempDir)

	// Initially should not have self-signed certs
	if cm.hasSelfSignedCerts() {
		t.Error("Should not have self-signed certs initially")
	}

	// Create certs directory
	os.MkdirAll(cm.certsDir, 0700)

	// Create mock self-signed cert files
	os.WriteFile(cm.certFile, []byte("mock cert"), 0600)
	os.WriteFile(cm.keyFile, []byte("mock key"), 0600)

	// Now should have self-signed certs
	if !cm.hasSelfSignedCerts() {
		t.Error("Should have self-signed certs after creating files")
	}
}

func TestTLSConfig_Fields(t *testing.T) {
	config := &TLSConfig{
		CertPath:  "/path/to/cert.pem",
		KeyPath:   "/path/to/key.pem",
		Enabled:   true,
		CertType:  "mkcert",
		IsTrusted: true,
	}

	if config.CertPath != "/path/to/cert.pem" {
		t.Errorf("Unexpected CertPath: %s", config.CertPath)
	}

	if config.KeyPath != "/path/to/key.pem" {
		t.Errorf("Unexpected KeyPath: %s", config.KeyPath)
	}

	if !config.Enabled {
		t.Error("Expected Enabled to be true")
	}

	if config.CertType != "mkcert" {
		t.Errorf("Unexpected CertType: %s", config.CertType)
	}

	if !config.IsTrusted {
		t.Error("Expected IsTrusted to be true")
	}
}

func TestFileExists(t *testing.T) {
	tempDir := t.TempDir()

	// Test non-existent file
	if fileExists(filepath.Join(tempDir, "nonexistent.txt")) {
		t.Error("fileExists returned true for non-existent file")
	}

	// Test existing file
	testFile := filepath.Join(tempDir, "test.txt")
	os.WriteFile(testFile, []byte("test"), 0600)

	if !fileExists(testFile) {
		t.Error("fileExists returned false for existing file")
	}
}
