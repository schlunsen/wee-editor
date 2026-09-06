// Package gpu provides a client for managing GPU sidecar sessions via the
// wee.cat control plane. It handles launching, monitoring, stopping, and
// resuming RunPod GPU pods that are associated with a sandbox.
package gpu

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// GPUSession represents a GPU pod session managed by the control plane.
type GPUSession struct {
	ID           int        `json:"id"`
	PodID        string     `json:"pod_id"`
	TemplateName string     `json:"template_name"`
	Tier         string     `json:"tier"`
	GPUType         string `json:"gpu_type"`
	VolumeGB        *int   `json:"volume_gb,omitempty"`
	ContainerDiskGB *int   `json:"container_disk_gb,omitempty"`
	Status       string     `json:"status"`
	SSHHost      string     `json:"ssh_host"`
	SSHPort      int        `json:"ssh_port"`
	ErrorMessage string     `json:"error_message"`
	StartedAt    *time.Time `json:"started_at"`
	StoppedAt    *time.Time `json:"stopped_at"`
	CreatedAt    time.Time  `json:"created_at"`
	Live         *LiveData  `json:"live,omitempty"`
}

// LiveData contains real-time cost and uptime information for a running session.
type LiveData struct {
	CostPerHr     float64 `json:"cost_per_hr"`
	UptimeSeconds int     `json:"uptime_seconds"`
	TotalCost     float64 `json:"total_cost"`
}

// Template represents a GPU pod template available for launching.
type Template struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	ImageName string `json:"imageName"`
}

// NetworkVolume represents a persistent network-attached storage volume.
type NetworkVolume struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	SizeGB       int    `json:"size_gb"`
	DataCenterID string `json:"data_center_id"`
	MountedPodID string `json:"mounted_pod_id,omitempty"`
}

// DataCenter represents an available RunPod data center location.
type DataCenter struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Manager handles communication with the wee.cat control plane for GPU operations.
type Manager struct {
	weeCatURL  string
	sandboxID  string
	authToken  string
	sshKeyPath string
	client     *http.Client
}

// NewManager creates a new GPU Manager with the given configuration.
func NewManager(weeCatURL, sandboxID, authToken, sshKeyPath string) *Manager {
	return &Manager{
		weeCatURL:  strings.TrimRight(weeCatURL, "/"),
		sandboxID:  sandboxID,
		authToken:  authToken,
		sshKeyPath: sshKeyPath,
		client:     &http.Client{Timeout: 30 * time.Second},
	}
}

// NewManagerFromEnv creates a Manager using environment variables:
//   - WEE_CONTROL_PLANE_URL: base URL for the control plane API
//   - WEE_SANDBOX_ID: sandbox identifier
//   - WEE_CONTROL_PLANE_TOKEN: authentication token
//
// SSH key path defaults to ~/.ssh/wee_gpu_ed25519.
func NewManagerFromEnv() (*Manager, error) {
	url := os.Getenv("WEE_CONTROL_PLANE_URL")
	if url == "" {
		return nil, fmt.Errorf("WEE_CONTROL_PLANE_URL environment variable is not set")
	}
	sandboxID := os.Getenv("WEE_SANDBOX_ID")
	if sandboxID == "" {
		return nil, fmt.Errorf("WEE_SANDBOX_ID environment variable is not set")
	}
	token := os.Getenv("WEE_CONTROL_PLANE_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("WEE_CONTROL_PLANE_TOKEN environment variable is not set")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to determine home directory: %w", err)
	}
	sshKeyPath := home + "/.ssh/wee_gpu_ed25519"

	return NewManager(url, sandboxID, token, sshKeyPath), nil
}

// SSHKeyPath returns the path to the SSH private key used for GPU pod access.
func (m *Manager) SSHKeyPath() string {
	return m.sshKeyPath
}

// doRequest executes an HTTP request against the control plane API.
// It sets the Authorization header using the SandboxToken scheme.
func (m *Manager) doRequest(method, path string, body io.Reader) (*http.Response, error) {
	url := m.weeCatURL + path
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "SandboxToken "+m.authToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to %s %s failed: %w", method, path, err)
	}
	return resp, nil
}

// decodeOrClose reads the response body into target, closing the body afterwards.
// If the status code is not in the 2xx range, it returns an error with the body content.
func decodeOrClose(resp *http.Response, target interface{}) error {
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}
	if target == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

// Launch starts a new GPU pod with the specified tier, template, SSH public key,
// and optional storage sizes. Pass 0 for volumeGB or containerDiskGB to use tier defaults.
func (m *Manager) Launch(tier, templateID, sshPublicKey string, volumeGB, containerDiskGB int, networkVolumeID ...string) (*GPUSession, error) {
	payload := map[string]interface{}{
		"sandbox_id":     m.sandboxID,
		"tier":           tier,
		"template_id":    templateID,
		"ssh_public_key": sshPublicKey,
	}
	if volumeGB > 0 {
		payload["volume_gb"] = volumeGB
	}
	if containerDiskGB > 0 {
		payload["container_disk_gb"] = containerDiskGB
	}
	if len(networkVolumeID) > 0 && networkVolumeID[0] != "" {
		payload["network_volume_id"] = networkVolumeID[0]
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal launch payload: %w", err)
	}

	resp, err := m.doRequest("POST", "/api/gpu/launch/", bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, err
	}
	var session GPUSession
	if err := decodeOrClose(resp, &session); err != nil {
		return nil, fmt.Errorf("failed to launch GPU session: %w", err)
	}
	return &session, nil
}

// ListSessions returns all GPU sessions for this sandbox.
func (m *Manager) ListSessions() ([]GPUSession, error) {
	resp, err := m.doRequest("GET", "/api/gpu/sessions/?sandbox_id="+m.sandboxID, nil)
	if err != nil {
		return nil, err
	}
	var wrapper struct {
		Sessions []GPUSession `json:"sessions"`
	}
	if err := decodeOrClose(resp, &wrapper); err != nil {
		return nil, fmt.Errorf("failed to list GPU sessions: %w", err)
	}
	return wrapper.Sessions, nil
}

// ActiveSession returns the currently active GPU session (running or creating),
// or nil if no session is active.
func (m *Manager) ActiveSession() (*GPUSession, error) {
	sessions, err := m.ListSessions()
	if err != nil {
		return nil, err
	}
	// Prefer running sessions, fall back to creating
	var creating *GPUSession
	for i := range sessions {
		switch sessions[i].Status {
		case "running":
			return &sessions[i], nil
		case "creating":
			if creating == nil {
				creating = &sessions[i]
			}
		}
	}
	return creating, nil
}

// Status returns the status of the active GPU session.
// Returns an error if no active session is found.
func (m *Manager) Status() (*GPUSession, error) {
	active, err := m.ActiveSession()
	if err != nil {
		return nil, err
	}
	if active == nil {
		// Check if there are any sessions at all (e.g. errored)
		sessions, _ := m.ListSessions()
		if len(sessions) > 0 {
			last := sessions[len(sessions)-1]
			if last.Status == "error" {
				return &last, nil
			}
		}
		return nil, fmt.Errorf("no active GPU session found")
	}
	// If session is still creating and has no pod_id, return what we have
	if active.PodID == "" {
		return active, nil
	}
	path := fmt.Sprintf("/api/gpu/sessions/%d/status/", active.ID)
	resp, err := m.doRequest("GET", path, nil)
	if err != nil {
		// Fall back to what we have if status endpoint fails
		return active, nil
	}
	var session GPUSession
	if err := decodeOrClose(resp, &session); err != nil {
		return active, nil
	}
	return &session, nil
}

// Stop stops the active GPU session.
func (m *Manager) Stop() error {
	active, err := m.ActiveSession()
	if err != nil {
		return err
	}
	if active == nil {
		return fmt.Errorf("no active GPU session to stop")
	}
	path := fmt.Sprintf("/api/gpu/sessions/%d/stop/", active.ID)
	resp, err := m.doRequest("POST", path, nil)
	if err != nil {
		return err
	}
	return decodeOrClose(resp, nil)
}

// Resume resumes a stopped GPU session.
func (m *Manager) Resume() error {
	sessions, err := m.ListSessions()
	if err != nil {
		return err
	}
	for _, s := range sessions {
		if s.Status == "stopped" || s.Status == "exited" {
			path := fmt.Sprintf("/api/gpu/sessions/%d/resume/", s.ID)
			resp, err := m.doRequest("POST", path, nil)
			if err != nil {
				return err
			}
			return decodeOrClose(resp, nil)
		}
	}
	return fmt.Errorf("no stopped GPU session to resume")
}

// Kill terminates and deletes a GPU session permanently.
func (m *Manager) Kill() error {
	active, err := m.ActiveSession()
	if err != nil {
		return err
	}
	if active == nil {
		return fmt.Errorf("no active GPU session to terminate")
	}
	path := fmt.Sprintf("/api/gpu/sessions/%d/terminate/", active.ID)
	resp, err := m.doRequest("DELETE", path, nil)
	if err != nil {
		return err
	}
	return decodeOrClose(resp, nil)
}

// ListTemplates returns available GPU pod templates.
func (m *Manager) ListTemplates() ([]Template, error) {
	resp, err := m.doRequest("GET", "/api/gpu/templates/", nil)
	if err != nil {
		return nil, err
	}
	var wrapper struct {
		Templates []Template `json:"templates"`
	}
	if err := decodeOrClose(resp, &wrapper); err != nil {
		return nil, fmt.Errorf("failed to list GPU templates: %w", err)
	}
	return wrapper.Templates, nil
}

// ResizePod resizes the storage of a GPU session.
func (m *Manager) ResizePod(sessionID int, volumeGB, containerDiskGB int) error {
	payload := map[string]interface{}{}
	if volumeGB > 0 {
		payload["volume_gb"] = volumeGB
	}
	if containerDiskGB > 0 {
		payload["container_disk_gb"] = containerDiskGB
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal resize payload: %w", err)
	}
	path := fmt.Sprintf("/api/gpu/sessions/%d/resize/", sessionID)
	resp, err := m.doRequest("POST", path, bytes.NewReader(payloadBytes))
	if err != nil {
		return err
	}
	return decodeOrClose(resp, nil)
}

// ListNetworkVolumes returns all network volumes for this sandbox.
func (m *Manager) ListNetworkVolumes() ([]NetworkVolume, error) {
	resp, err := m.doRequest("GET", "/api/gpu/network-volumes/?sandbox_id="+m.sandboxID, nil)
	if err != nil {
		return nil, err
	}
	var wrapper struct {
		Volumes []NetworkVolume `json:"volumes"`
	}
	if err := decodeOrClose(resp, &wrapper); err != nil {
		return nil, fmt.Errorf("failed to list network volumes: %w", err)
	}
	return wrapper.Volumes, nil
}

// CreateNetworkVolume creates a new persistent network volume.
func (m *Manager) CreateNetworkVolume(name string, sizeGB int, dataCenterID string) (*NetworkVolume, error) {
	payload := map[string]interface{}{
		"sandbox_id":     m.sandboxID,
		"name":           name,
		"size_gb":        sizeGB,
		"data_center_id": dataCenterID,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal create volume payload: %w", err)
	}
	resp, err := m.doRequest("POST", "/api/gpu/network-volumes/", bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, err
	}
	var vol NetworkVolume
	if err := decodeOrClose(resp, &vol); err != nil {
		return nil, fmt.Errorf("failed to create network volume: %w", err)
	}
	return &vol, nil
}

// DeleteNetworkVolume permanently deletes a network volume.
func (m *Manager) DeleteNetworkVolume(volumeID string) error {
	resp, err := m.doRequest("DELETE", "/api/gpu/network-volumes/"+volumeID+"/", nil)
	if err != nil {
		return err
	}
	return decodeOrClose(resp, nil)
}

// ListDataCenters returns available RunPod data centers.
func (m *Manager) ListDataCenters() ([]DataCenter, error) {
	resp, err := m.doRequest("GET", "/api/gpu/data-centers/", nil)
	if err != nil {
		return nil, err
	}
	var wrapper struct {
		DataCenters []DataCenter `json:"data_centers"`
	}
	if err := decodeOrClose(resp, &wrapper); err != nil {
		return nil, fmt.Errorf("failed to list data centers: %w", err)
	}
	return wrapper.DataCenters, nil
}
