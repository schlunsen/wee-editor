package gpu

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

// EnsureSSHKey generates an ed25519 SSH keypair at keyPath if it does not
// already exist. Returns the public key string. This function is idempotent:
// if the key already exists, it simply reads and returns the public key.
// Uses Go's crypto library — no dependency on ssh-keygen binary.
func EnsureSSHKey(keyPath string) (string, error) {
	pubPath := keyPath + ".pub"

	// If key already exists, read and return the public key
	if _, err := os.Stat(keyPath); err == nil {
		pub, err := os.ReadFile(pubPath)
		if err != nil {
			return "", fmt.Errorf("SSH key exists but failed to read public key %s: %w", pubPath, err)
		}
		return strings.TrimSpace(string(pub)), nil
	}

	// Ensure parent directory exists
	dir := filepath.Dir(keyPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("failed to create SSH key directory %s: %w", dir, err)
	}

	// Generate ed25519 keypair using Go crypto
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", fmt.Errorf("failed to generate ed25519 key: %w", err)
	}

	// Marshal private key to OpenSSH format
	pemBlock, err := ssh.MarshalPrivateKey(privKey, "wee-gpu-sidecar")
	if err != nil {
		return "", fmt.Errorf("failed to marshal private key: %w", err)
	}
	privPEM := pem.EncodeToMemory(pemBlock)
	if err := os.WriteFile(keyPath, privPEM, 0600); err != nil {
		return "", fmt.Errorf("failed to write private key: %w", err)
	}

	// Marshal public key to authorized_keys format
	sshPub, err := ssh.NewPublicKey(pubKey)
	if err != nil {
		return "", fmt.Errorf("failed to create SSH public key: %w", err)
	}
	pubStr := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPub))) + " wee-gpu-sidecar"
	if err := os.WriteFile(pubPath, []byte(pubStr+"\n"), 0644); err != nil {
		return "", fmt.Errorf("failed to write public key: %w", err)
	}

	return pubStr, nil
}

// dialSSH creates an SSH client connection to the GPU pod using the native
// Go SSH library. No external ssh binary required.
func (m *Manager) dialSSH(session *GPUSession) (*ssh.Client, error) {
	keyBytes, err := os.ReadFile(m.sshKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read SSH key %s: %w", m.sshKeyPath, err)
	}

	signer, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SSH key: %w", err)
	}

	config := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", session.SSHHost, session.SSHPort)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", addr, err)
	}

	return client, nil
}

// Shell opens an interactive SSH session to the active GPU pod.
// Uses Go's native SSH library — no ssh binary needed.
func (m *Manager) Shell() error {
	// Use Status() which refreshes SSH details from RunPod
	gpuSession, err := m.Status()
	if err != nil {
		return err
	}
	if gpuSession.SSHHost == "" || gpuSession.SSHPort == 0 {
		return fmt.Errorf("GPU session %s does not have SSH connection details yet (pod may still be starting)", gpuSession.PodID)
	}

	client, err := m.dialSSH(gpuSession)
	if err != nil {
		return err
	}
	defer client.Close()

	sshSession, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer sshSession.Close()

	// Set up terminal raw mode for interactive use
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		oldState, err := term.MakeRaw(fd)
		if err != nil {
			return fmt.Errorf("failed to set terminal raw mode: %w", err)
		}
		defer term.Restore(fd, oldState)

		w, h, err := term.GetSize(fd)
		if err != nil {
			w, h = 80, 24
		}

		modes := ssh.TerminalModes{
			ssh.ECHO:          1,
			ssh.TTY_OP_ISPEED: 14400,
			ssh.TTY_OP_OSPEED: 14400,
		}

		if err := sshSession.RequestPty("xterm-256color", h, w, modes); err != nil {
			return fmt.Errorf("failed to request PTY: %w", err)
		}

		// Handle window size changes
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGWINCH)
		go func() {
			for range sigCh {
				if newW, newH, err := term.GetSize(fd); err == nil {
					_ = sshSession.WindowChange(newH, newW)
				}
			}
		}()
		defer signal.Stop(sigCh)
	}

	sshSession.Stdin = os.Stdin
	sshSession.Stdout = os.Stdout
	sshSession.Stderr = os.Stderr

	if err := sshSession.Shell(); err != nil {
		return fmt.Errorf("failed to start shell: %w", err)
	}

	return sshSession.Wait()
}

// Run executes a command on the active GPU pod via SSH and streams output
// to stdout/stderr. Uses Go's native SSH library — no ssh binary needed.
func (m *Manager) Run(command string) error {
	gpuSession, err := m.Status()
	if err != nil {
		return err
	}
	if gpuSession.SSHHost == "" || gpuSession.SSHPort == 0 {
		return fmt.Errorf("GPU session %s does not have SSH connection details yet (pod may still be starting)", gpuSession.PodID)
	}

	client, err := m.dialSSH(gpuSession)
	if err != nil {
		return err
	}
	defer client.Close()

	sshSession, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer sshSession.Close()

	// Pipe stdout and stderr
	stdout, err := sshSession.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to pipe stdout: %w", err)
	}
	stderr, err := sshSession.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to pipe stderr: %w", err)
	}

	if err := sshSession.Start(command); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	// Stream output
	go io.Copy(os.Stdout, stdout)
	go io.Copy(os.Stderr, stderr)

	return sshSession.Wait()
}

// RunCapture executes a command on the active GPU pod via SSH and returns
// the combined stdout+stderr output as a string. Used by MCP tools so the
// agent can read command output.
func (m *Manager) RunCapture(command string) (string, error) {
	gpuSession, err := m.Status()
	if err != nil {
		return "", err
	}
	if gpuSession.SSHHost == "" || gpuSession.SSHPort == 0 {
		return "", fmt.Errorf("GPU session %s does not have SSH connection details yet (pod may still be starting)", gpuSession.PodID)
	}

	client, err := m.dialSSH(gpuSession)
	if err != nil {
		return "", err
	}
	defer client.Close()

	sshSession, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer sshSession.Close()

	// Capture both stdout and stderr into a single buffer
	var combined bytes.Buffer
	sshSession.Stdout = &combined
	sshSession.Stderr = &combined

	err = sshSession.Run(command)

	output := combined.String()
	// Truncate very long output to avoid overwhelming MCP responses
	const maxOutput = 100000 // ~100KB
	if len(output) > maxOutput {
		output = output[:maxOutput] + "\n... [output truncated at 100KB]"
	}

	if err != nil {
		return output, fmt.Errorf("command failed: %w\nOutput: %s", err, output)
	}
	return output, nil
}

// buildSSHArgs returns the common SSH arguments for connecting to a GPU pod.
// Used by Push/Pull which need to invoke rsync with an ssh command string.
func buildSSHArgs(keyPath, host string, port int) []string {
	return []string{
		"-i", keyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "LogLevel=ERROR",
		"-p", fmt.Sprintf("%d", port),
		"root@" + host,
	}
}

// SCPUpload uploads a local file to the remote GPU pod using the native
// SSH library (SFTP-like via a single exec). Falls back to rsync if available.
func (m *Manager) scpUpload(localPath, remotePath string, gpuSession *GPUSession) error {
	client, err := m.dialSSH(gpuSession)
	if err != nil {
		return err
	}
	defer client.Close()

	local, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %w", err)
	}
	defer local.Close()

	stat, err := local.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat local file: %w", err)
	}

	sshSession, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer sshSession.Close()

	go func() {
		w, _ := sshSession.StdinPipe()
		defer w.Close()
		fmt.Fprintf(w, "C0644 %d %s\n", stat.Size(), filepath.Base(localPath))
		io.Copy(w, local)
		fmt.Fprint(w, "\x00")
	}()

	return sshSession.Run(fmt.Sprintf("scp -t %s", remotePath))
}

// hasCommand checks if a command is available in PATH.
func hasCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// sshPortForward creates a local port forward (for use with rsync/scp binaries
// if they become available). Not currently used but available for future use.
func (m *Manager) sshPortForward(gpuSession *GPUSession, localPort int, remoteAddr string) (net.Listener, error) {
	client, err := m.dialSSH(gpuSession)
	if err != nil {
		return nil, err
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", localPort))
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to listen on local port %d: %w", localPort, err)
	}

	go func() {
		defer client.Close()
		for {
			local, err := listener.Accept()
			if err != nil {
				return
			}
			remote, err := client.Dial("tcp", remoteAddr)
			if err != nil {
				local.Close()
				continue
			}
			go func() {
				defer local.Close()
				defer remote.Close()
				go io.Copy(remote, local)
				io.Copy(local, remote)
			}()
		}
	}()

	return listener, nil
}
