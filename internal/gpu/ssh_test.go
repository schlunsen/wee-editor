package gpu

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureSSHKey(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test_key")

	// First call should generate the key
	pubKey, err := EnsureSSHKey(keyPath)
	if err != nil {
		t.Fatalf("failed to generate SSH key: %v", err)
	}
	if !strings.HasPrefix(pubKey, "ssh-ed25519 ") {
		t.Errorf("expected ed25519 public key, got %q", pubKey)
	}
	if !strings.Contains(pubKey, "wee-gpu-sidecar") {
		t.Errorf("expected comment wee-gpu-sidecar in public key, got %q", pubKey)
	}

	// Verify private key file exists
	if _, err := os.Stat(keyPath); err != nil {
		t.Errorf("private key file not found: %v", err)
	}

	// Verify public key file exists
	if _, err := os.Stat(keyPath + ".pub"); err != nil {
		t.Errorf("public key file not found: %v", err)
	}

	// Second call should be idempotent and return same key
	pubKey2, err := EnsureSSHKey(keyPath)
	if err != nil {
		t.Fatalf("idempotent call failed: %v", err)
	}
	if pubKey != pubKey2 {
		t.Errorf("idempotent call returned different key:\n  first:  %q\n  second: %q", pubKey, pubKey2)
	}
}

func TestEnsureSSHKey_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "nested", "dir", "test_key")

	pubKey, err := EnsureSSHKey(keyPath)
	if err != nil {
		t.Fatalf("failed to generate SSH key in nested dir: %v", err)
	}
	if !strings.HasPrefix(pubKey, "ssh-ed25519 ") {
		t.Errorf("expected ed25519 public key, got %q", pubKey)
	}
}

func TestBuildSSHArgs(t *testing.T) {
	args := buildSSHArgs("/home/user/.ssh/key", "gpu.example.com", 2222)

	expected := []string{
		"-i", "/home/user/.ssh/key",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "LogLevel=ERROR",
		"-p", "2222",
		"root@gpu.example.com",
	}

	if len(args) != len(expected) {
		t.Fatalf("expected %d args, got %d: %v", len(expected), len(args), args)
	}
	for i, arg := range args {
		if arg != expected[i] {
			t.Errorf("arg[%d]: expected %q, got %q", i, expected[i], arg)
		}
	}
}

func TestBuildSSHArgs_DefaultPort(t *testing.T) {
	args := buildSSHArgs("/tmp/key", "host.com", 22)
	// Find the port arg
	for i, arg := range args {
		if arg == "-p" {
			if args[i+1] != "22" {
				t.Errorf("expected port 22, got %q", args[i+1])
			}
			return
		}
	}
	t.Error("port flag not found in args")
}
