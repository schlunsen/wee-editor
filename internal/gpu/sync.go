package gpu

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const defaultRemotePath = "/workspace/project/"

// Push syncs files from a local path to the active GPU pod.
// If localPath is empty, it defaults to the current working directory.
// Uses native SSH (tar over SSH) — no rsync binary required.
// Falls back to rsync if available for better performance on large syncs.
func (m *Manager) Push(localPath string) error {
	session, err := m.Status()
	if err != nil {
		return err
	}
	if session.SSHHost == "" || session.SSHPort == 0 {
		return fmt.Errorf("GPU session %s does not have SSH connection details yet (pod may still be starting)", session.PodID)
	}

	if localPath == "" {
		localPath, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}
	// Ensure trailing slash so rsync copies contents, not the directory itself
	if localPath[len(localPath)-1] != '/' {
		localPath += "/"
	}

	// Try rsync first (faster for incremental syncs)
	if hasCommand("rsync") {
		return m.rsyncPush(localPath, session)
	}

	// Fallback: tar over native SSH
	return m.tarPush(localPath, session)
}

// Pull syncs files from the active GPU pod to a local path.
// If remotePath is empty, it defaults to "/app/".
// Uses native SSH (tar over SSH) — no rsync binary required.
func (m *Manager) Pull(remotePath string) error {
	session, err := m.Status()
	if err != nil {
		return err
	}
	if session.SSHHost == "" || session.SSHPort == 0 {
		return fmt.Errorf("GPU session %s does not have SSH connection details yet (pod may still be starting)", session.PodID)
	}

	if remotePath == "" {
		remotePath = defaultRemotePath
	}
	if remotePath[len(remotePath)-1] != '/' {
		remotePath += "/"
	}

	// Try rsync first
	if hasCommand("rsync") {
		return m.rsyncPull(remotePath, session)
	}

	// Fallback: tar over native SSH
	return m.tarPull(remotePath, session)
}

// PushCapture is like Push but captures and returns output (for MCP).
// remotePath defaults to defaultRemotePath if empty.
func (m *Manager) PushCapture(localPath string, remotePath ...string) (string, error) {
	session, err := m.Status()
	if err != nil {
		return "", err
	}
	if session.SSHHost == "" || session.SSHPort == 0 {
		return "", fmt.Errorf("GPU session %s does not have SSH connection details yet (pod may still be starting)", session.PodID)
	}

	if localPath == "" {
		localPath, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get current directory: %w", err)
		}
	}
	if localPath[len(localPath)-1] != '/' {
		localPath += "/"
	}

	destPath := defaultRemotePath
	if len(remotePath) > 0 && remotePath[0] != "" {
		destPath = remotePath[0]
		if destPath[len(destPath)-1] != '/' {
			destPath += "/"
		}
	}

	// Prefer rsync for incremental syncs (faster and more reliable)
	if hasCommand("rsync") {
		return m.rsyncPushCapture(localPath, session, destPath)
	}

	return m.tarPushCapture(localPath, session, destPath)
}

// PullCapture is like Pull but captures and returns output (for MCP).
func (m *Manager) PullCapture(remotePath string) (string, error) {
	session, err := m.Status()
	if err != nil {
		return "", err
	}
	if session.SSHHost == "" || session.SSHPort == 0 {
		return "", fmt.Errorf("GPU session %s does not have SSH connection details yet (pod may still be starting)", session.PodID)
	}

	if remotePath == "" {
		remotePath = defaultRemotePath
	}
	if remotePath[len(remotePath)-1] != '/' {
		remotePath += "/"
	}

	return m.tarPullCapture(remotePath, session)
}

// tarPush sends files via tar piped through native SSH.
func (m *Manager) tarPush(localPath string, session *GPUSession) error {
	client, err := m.dialSSH(session)
	if err != nil {
		return err
	}
	defer client.Close()

	sshSession, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer sshSession.Close()

	// Pipe local tar output into remote tar extract
	stdinPipe, err := sshSession.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	sshSession.Stdout = os.Stdout
	sshSession.Stderr = os.Stderr

	// Remote command: extract tar to /app/
	if err := sshSession.Start(fmt.Sprintf("tar xzf - -C %s", defaultRemotePath)); err != nil {
		return fmt.Errorf("failed to start remote tar: %w", err)
	}

	// Local: create tar of the directory and pipe to remote
	tarArgs := []string{"czf", "-", "-C", localPath}
	for _, exc := range defaultExcludes {
		tarArgs = append(tarArgs, "--exclude="+exc)
	}
	tarArgs = append(tarArgs, ".")
	tarCmd := exec.Command("tar", tarArgs...)
	tarCmd.Stdout = stdinPipe
	tarCmd.Stderr = os.Stderr

	if err := tarCmd.Run(); err != nil {
		stdinPipe.Close()
		return fmt.Errorf("local tar failed: %w", err)
	}
	stdinPipe.Close()

	return sshSession.Wait()
}

// tarPushCapture sends files and returns a summary.
func (m *Manager) tarPushCapture(localPath string, session *GPUSession, destPath string) (string, error) {
	client, err := m.dialSSH(session)
	if err != nil {
		return "", err
	}
	defer client.Close()

	sshSession, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer sshSession.Close()

	stdinPipe, err := sshSession.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	var output bytes.Buffer
	sshSession.Stdout = &output
	sshSession.Stderr = &output

	// Ensure remote directory exists and extract there
	if err := sshSession.Start(fmt.Sprintf("mkdir -p %s && tar xvzf - -C %s", destPath, destPath)); err != nil {
		return "", fmt.Errorf("failed to start remote tar: %w", err)
	}

	tarArgs := []string{"czf", "-", "-C", localPath}
	for _, exc := range defaultExcludes {
		tarArgs = append(tarArgs, "--exclude="+exc)
	}
	tarArgs = append(tarArgs, ".")
	tarCmd := exec.Command("tar", tarArgs...)
	tarCmd.Stdout = stdinPipe

	if err := tarCmd.Run(); err != nil {
		stdinPipe.Close()
		return output.String(), fmt.Errorf("local tar failed: %w", err)
	}
	stdinPipe.Close()

	err = sshSession.Wait()

	// Count files from verbose output
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	fileCount := 0
	for _, l := range lines {
		if l != "" && !strings.HasSuffix(l, "/") {
			fileCount++
		}
	}

	summary := fmt.Sprintf("Pushed %d files from %s to %s:%s", fileCount, localPath, session.SSHHost, destPath)
	if fileCount <= 20 {
		summary += "\n" + output.String()
	}

	return summary, err
}

// tarPull receives files via tar piped through native SSH.
func (m *Manager) tarPull(remotePath string, session *GPUSession) error {
	client, err := m.dialSSH(session)
	if err != nil {
		return err
	}
	defer client.Close()

	sshSession, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer sshSession.Close()

	localDest, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Pipe remote tar output into local tar extract
	stdoutPipe, err := sshSession.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}
	// Capture remote stderr so we can report meaningful errors
	var remoteErr bytes.Buffer
	sshSession.Stderr = &remoteErr

	if err := sshSession.Start(fmt.Sprintf("tar czf - -C %s .", remotePath)); err != nil {
		return fmt.Errorf("failed to start remote tar: %w", err)
	}

	// Local: extract tar
	tarCmd := exec.Command("tar", "xzf", "-", "-C", localDest)
	tarCmd.Stdin = stdoutPipe
	tarCmd.Stdout = os.Stdout
	tarCmd.Stderr = os.Stderr

	localErr := tarCmd.Run()

	// Check remote tar exit status first — if the remote path doesn't exist,
	// this gives us the real error instead of a confusing local tar failure.
	if waitErr := sshSession.Wait(); waitErr != nil {
		errMsg := remoteErr.String()
		if errMsg != "" {
			return fmt.Errorf("remote tar failed: %w\n%s", waitErr, errMsg)
		}
		return fmt.Errorf("remote tar failed: %w", waitErr)
	}

	if localErr != nil {
		return fmt.Errorf("local tar extract failed: %w", localErr)
	}

	return nil
}

// tarPullCapture receives files and returns a summary.
func (m *Manager) tarPullCapture(remotePath string, session *GPUSession) (string, error) {
	client, err := m.dialSSH(session)
	if err != nil {
		return "", err
	}
	defer client.Close()

	sshSession, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer sshSession.Close()

	localDest, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	stdoutPipe, err := sshSession.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to get stdout pipe: %w", err)
	}
	// Capture stderr so remote tar errors are visible in MCP responses
	var remoteErr bytes.Buffer
	sshSession.Stderr = &remoteErr

	if err := sshSession.Start(fmt.Sprintf("tar czf - -C %s .", remotePath)); err != nil {
		return "", fmt.Errorf("failed to start remote tar: %w", err)
	}

	// Extract with verbose to capture file list
	tarCmd := exec.Command("tar", "xvzf", "-", "-C", localDest)
	tarCmd.Stdin = stdoutPipe
	var tarOutput bytes.Buffer
	tarCmd.Stdout = &tarOutput
	tarCmd.Stderr = &tarOutput

	localErr := tarCmd.Run()

	// Check remote tar exit status — if the remote path doesn't exist or
	// permissions fail, this is where we catch it.
	if waitErr := sshSession.Wait(); waitErr != nil {
		errMsg := remoteErr.String()
		if errMsg != "" {
			return errMsg, fmt.Errorf("remote tar failed: %w\n%s", waitErr, errMsg)
		}
		return "", fmt.Errorf("remote tar failed: %w", waitErr)
	}

	if localErr != nil {
		return tarOutput.String(), fmt.Errorf("local tar extract failed: %w", localErr)
	}

	lines := strings.Split(strings.TrimSpace(tarOutput.String()), "\n")
	fileCount := 0
	for _, l := range lines {
		if l != "" && !strings.HasSuffix(l, "/") {
			fileCount++
		}
	}

	summary := fmt.Sprintf("Pulled %d files from %s:%s to %s", fileCount, session.SSHHost, remotePath, localDest)
	if fileCount <= 20 {
		summary += "\n" + tarOutput.String()
	}

	return summary, nil
}

// defaultExcludes are patterns excluded from push syncs to avoid transferring
// large or irrelevant directories.
var defaultExcludes = []string{
	".git",
	"node_modules",
	"__pycache__",
	".venv",
	"venv",
	".mypy_cache",
	".pytest_cache",
	"*.pyc",
}

// rsyncPush uses rsync binary for efficient incremental sync.
func (m *Manager) rsyncPush(localPath string, session *GPUSession) error {
	remoteDest := fmt.Sprintf("root@%s:%s", session.SSHHost, defaultRemotePath)
	sshCmd := fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o LogLevel=ERROR -p %d",
		m.sshKeyPath, session.SSHPort)

	args := []string{"-avz", "--progress", "-e", sshCmd}
	for _, exc := range defaultExcludes {
		args = append(args, "--exclude="+exc)
	}
	args = append(args, localPath, remoteDest)

	cmd := exec.Command("rsync", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("rsync push failed: %w", err)
	}
	return nil
}

// rsyncPushCapture uses rsync and captures output for MCP responses.
func (m *Manager) rsyncPushCapture(localPath string, session *GPUSession, destPath string) (string, error) {
	remoteDest := fmt.Sprintf("root@%s:%s", session.SSHHost, destPath)
	sshCmd := fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o LogLevel=ERROR -p %d",
		m.sshKeyPath, session.SSHPort)

	args := []string{"-avz", "-e", sshCmd}
	for _, exc := range defaultExcludes {
		args = append(args, "--exclude="+exc)
	}
	args = append(args, localPath, remoteDest)

	cmd := exec.Command("rsync", args...)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	if err := cmd.Run(); err != nil {
		return output.String(), fmt.Errorf("rsync push failed: %w", err)
	}

	// Parse rsync output to count files
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	fileCount := 0
	for _, l := range lines {
		if l != "" && !strings.HasSuffix(l, "/") && !strings.HasPrefix(l, "sending") && !strings.HasPrefix(l, "sent ") && !strings.HasPrefix(l, "total ") {
			fileCount++
		}
	}

	summary := fmt.Sprintf("Pushed %d files from %s to %s:%s (rsync)", fileCount, localPath, session.SSHHost, destPath)
	out := output.String()
	// Truncate long output
	const maxOutput = 5000
	if len(out) > maxOutput {
		out = out[:maxOutput] + "\n... [output truncated]"
	}
	summary += "\n" + out
	return summary, nil
}

// rsyncPull uses rsync binary for efficient incremental sync.
func (m *Manager) rsyncPull(remotePath string, session *GPUSession) error {
	remoteSrc := fmt.Sprintf("root@%s:%s", session.SSHHost, remotePath)
	sshCmd := fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o LogLevel=ERROR -p %d",
		m.sshKeyPath, session.SSHPort)

	localDest, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	cmd := exec.Command("rsync", "-avz", "--progress",
		"-e", sshCmd,
		remoteSrc, filepath.Clean(localDest)+"/",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("rsync pull failed: %w", err)
	}
	return nil
}
