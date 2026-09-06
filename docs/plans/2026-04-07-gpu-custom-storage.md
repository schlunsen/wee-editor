# GPU Custom Storage Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Allow users to specify custom volume and container disk sizes when launching GPU pods, via TUI, CLI, MCP tools, and the control plane API.

**Architecture:** Thread an optional `volume_gb` and `container_disk_gb` through every layer — from the Django API serializer and RunPod GraphQL mutation, through the Go client and MCP tools, up to the TUI and CLI. Each tier keeps its current defaults as minimums; users can only increase, not decrease. The GPUSession Django model gains two new fields so storage choices are tracked and visible in status.

**Tech Stack:** Python/Django (backend), Go (client/TUI/MCP), RunPod GraphQL API, Bubble Tea (TUI)

---

### Task 1: Add storage fields to Django GPUSession model

**Files:**
- Modify: `wee.cat/backend/gpu/models.py`
- Create: `wee.cat/backend/gpu/migrations/0002_gpusession_storage_fields.py` (auto-generated)

**Step 1: Add fields to GPUSession model**

In `wee.cat/backend/gpu/models.py`, add two fields after `gpu_type`:

```python
    volume_gb = models.IntegerField(
        null=True,
        blank=True,
        help_text="Persistent volume size in GB (null = tier default)",
    )
    container_disk_gb = models.IntegerField(
        null=True,
        blank=True,
        help_text="Container disk size in GB (null = tier default)",
    )
```

**Step 2: Generate and apply migration**

Run:
```bash
cd wee.cat/backend
python manage.py makemigrations gpu --name gpusession_storage_fields
python manage.py migrate
```

Expected: Migration created and applied successfully.

**Step 3: Commit**

```bash
git add wee.cat/backend/gpu/models.py wee.cat/backend/gpu/migrations/0002_gpusession_storage_fields.py
git commit -m "feat(gpu): add volume_gb and container_disk_gb fields to GPUSession model"
```

---

### Task 2: Update Django serializers to accept and expose storage params

**Files:**
- Modify: `wee.cat/backend/gpu/serializers.py`

**Step 1: Add storage fields to LaunchPodSerializer**

Add these two optional fields to `LaunchPodSerializer`:

```python
class LaunchPodSerializer(serializers.Serializer):
    tier = serializers.ChoiceField(
        choices=[(t.value, t.value) for t in GPUTier],
    )
    template_id = serializers.CharField(max_length=100)
    ssh_public_key = serializers.CharField(required=False, default="", allow_blank=True)
    sandbox_id = serializers.IntegerField()
    volume_gb = serializers.IntegerField(required=False, default=None, min_value=1, max_value=1000)
    container_disk_gb = serializers.IntegerField(required=False, default=None, min_value=1, max_value=500)
```

**Step 2: Add storage fields to GPUSessionSerializer and GPUSessionStatusSerializer**

In `GPUSessionSerializer.Meta.fields`, add `"volume_gb"` and `"container_disk_gb"` to the fields list (after `"gpu_type"`).

In `GPUSessionStatusSerializer.Meta.fields`, add the same two fields (after `"gpu_type"`).

**Step 3: Commit**

```bash
git add wee.cat/backend/gpu/serializers.py
git commit -m "feat(gpu): accept volume_gb and container_disk_gb in launch serializer"
```

---

### Task 3: Thread storage through the RunPod client and launch view

**Files:**
- Modify: `wee.cat/backend/gpu/runpod_client.py`
- Modify: `wee.cat/backend/gpu/views.py`

**Step 1: Add optional storage overrides to `create_pod`**

Update the `create_pod` method signature and body in `runpod_client.py`:

```python
    def create_pod(
        self,
        name: str,
        template_id: str,
        tier: GPUTier,
        ssh_public_key: str = "",
        volume_gb: int | None = None,
        container_disk_gb: int | None = None,
    ) -> RunPodPod:
```

In the body, after `config = TIER_CONFIG[tier]`, resolve the actual sizes:

```python
        # Use custom storage if provided, otherwise tier defaults.
        # Enforce tier minimums so users can't shrink below the default.
        actual_volume_gb = max(volume_gb, config["volume_gb"]) if volume_gb else config["volume_gb"]
        actual_container_disk_gb = max(container_disk_gb, config["container_disk_gb"]) if container_disk_gb else config["container_disk_gb"]
```

Then in the `variables` dict, replace the hardcoded values:

```python
                "volumeInGb": actual_volume_gb,
                "containerDiskInGb": actual_container_disk_gb,
```

**Step 2: Thread storage through `launch_pod` view and `_launch_pod_async`**

In `views.py` `launch_pod()`, extract the new fields from the serializer:

```python
    volume_gb = serializer.validated_data.get("volume_gb")
    container_disk_gb = serializer.validated_data.get("container_disk_gb")
```

Store them on the session:

```python
    session = GPUSession.objects.create(
        sandbox=sandbox,
        connector=connector,
        template_id=template_id,
        template_name="",
        tier=tier_value,
        gpu_type=tier_config["display_name"],
        volume_gb=volume_gb,
        container_disk_gb=container_disk_gb,
        status="creating",
    )
```

Pass them to `_launch_pod_async`:

```python
    thread = threading.Thread(
        target=_launch_pod_async,
        args=(session.id, template_id, tier_enum, ssh_public_key, volume_gb, container_disk_gb),
        daemon=True,
    )
```

Update `_launch_pod_async` signature:

```python
def _launch_pod_async(session_id, template_id, tier_enum, ssh_public_key, volume_gb=None, container_disk_gb=None):
```

And pass the storage params to `client.create_pod()`:

```python
                pod = client.create_pod(
                    name=pod_name,
                    template_id=template_id,
                    tier=try_tier,
                    ssh_public_key=ssh_public_key,
                    volume_gb=volume_gb,
                    container_disk_gb=container_disk_gb,
                )
```

**Step 3: Commit**

```bash
git add wee.cat/backend/gpu/runpod_client.py wee.cat/backend/gpu/views.py
git commit -m "feat(gpu): thread custom storage sizes through RunPod client and launch flow"
```

---

### Task 4: Update Go GPU client to support storage params

**Files:**
- Modify: `internal/gpu/gpu.go`

**Step 1: Add storage fields to the Launch method**

Update the `Launch` method signature and payload:

```go
// Launch starts a new GPU pod with the specified tier, template, SSH public key,
// and optional storage overrides. Pass 0 for volumeGB or containerDiskGB to use
// the tier default.
func (m *Manager) Launch(tier, templateID, sshPublicKey string, volumeGB, containerDiskGB int) (*GPUSession, error) {
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
```

Add `"bytes"` to the imports.

**Step 2: Add storage fields to GPUSession struct**

```go
type GPUSession struct {
	ID              int        `json:"id"`
	PodID           string     `json:"pod_id"`
	TemplateName    string     `json:"template_name"`
	Tier            string     `json:"tier"`
	GPUType         string     `json:"gpu_type"`
	VolumeGB        *int       `json:"volume_gb,omitempty"`
	ContainerDiskGB *int       `json:"container_disk_gb,omitempty"`
	Status          string     `json:"status"`
	SSHHost         string     `json:"ssh_host"`
	SSHPort         int        `json:"ssh_port"`
	ErrorMessage    string     `json:"error_message"`
	StartedAt       *time.Time `json:"started_at"`
	StoppedAt       *time.Time `json:"stopped_at"`
	CreatedAt       time.Time  `json:"created_at"`
	Live            *LiveData  `json:"live,omitempty"`
}
```

**Step 3: Fix all callers of Launch()**

There are two call sites that need updating:

1. `internal/gpu/tui.go` line ~172 in `launchPod()`:
```go
func (m model) launchPod() tea.Cmd {
	return func() tea.Msg {
		pubKey, err := EnsureSSHKey(m.manager.SSHKeyPath())
		if err != nil {
			return launchDoneMsg{err: fmt.Errorf("SSH key error: %w", err)}
		}
		session, err := m.manager.Launch(m.selectedTier.Name, m.selectedTempl.ID, pubKey, m.volumeGB, m.containerDiskGB)
		return launchDoneMsg{session: session, err: err}
	}
}
```

2. `internal/gpu/mcp_tools.go` in `handleGPULaunch()`:
```go
	volumeGB, _ := args["volume_gb"].(float64)
	containerDiskGB, _ := args["container_disk_gb"].(float64)

	session, err := m.Launch(tier, templateID, pubKey, int(volumeGB), int(containerDiskGB))
```

3. `internal/cmd/gpu.go` in `gpuLaunchCmd`:
```go
	session, err := mgr.Launch(gpuTier, gpuTemplate, pubKey, gpuVolumeGB, gpuContainerDiskGB)
```

(The TUI model fields and CLI variables are added in Tasks 5 and 6.)

**Step 4: Commit**

```bash
git add internal/gpu/gpu.go
git commit -m "feat(gpu): add storage params to Go GPU client Launch method"
```

---

### Task 5: Add storage configuration to the TUI

**Files:**
- Modify: `internal/gpu/tui.go`

**Step 1: Add a storage configuration screen**

Add a new screen constant after `screenSelectTemplate`:

```go
const (
	screenSelectTier screen = iota
	screenSelectTemplate
	screenConfigStorage   // NEW
	screenLaunching
	screenDone
	screenError
)
```

Add storage fields to the `model` struct:

```go
type model struct {
	manager        *Manager
	screen         screen
	cursor         int
	selectedTier   tierInfo
	templates      []Template
	selectedTempl  Template
	session        *GPUSession
	err            error
	width          int
	height         int
	volumeGB       int  // 0 = tier default
	containerDiskGB int // 0 = tier default
	storageInput   string // text input buffer for the active field
	storageCursor  int    // 0 = volume, 1 = container disk, 2 = confirm
}
```

**Step 2: Update handleEnter for template selection to go to storage screen**

Change the `screenSelectTemplate` case in `handleEnter`:

```go
	case screenSelectTemplate:
		if m.cursor < len(m.templates) {
			m.selectedTempl = m.templates[m.cursor]
			// Set defaults from tier
			m.volumeGB = 0
			m.containerDiskGB = 0
			m.storageInput = ""
			m.storageCursor = 0
			m.screen = screenConfigStorage
		}
```

**Step 3: Add storage screen update handling**

In the `Update` method, add a case for `screenConfigStorage` key handling. This screen shows the tier defaults and lets users type in custom values, or just press Enter on "Launch" to use defaults:

```go
	case tea.KeyMsg:
		if m.screen == screenConfigStorage {
			return m.updateStorage(msg)
		}
		// ... existing switch
```

Add the `updateStorage` method:

```go
func (m model) updateStorage(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc":
		m.screen = screenSelectTemplate
		m.cursor = 0
		return m, nil
	case "up", "k":
		if m.storageCursor > 0 {
			m.storageCursor--
		}
	case "down", "j":
		if m.storageCursor < 2 {
			m.storageCursor++
		}
	case "enter":
		if m.storageCursor == 2 {
			// "Launch" selected — apply any typed value first
			m.screen = screenLaunching
			return m, m.launchPod()
		}
	case "backspace":
		if len(m.storageInput) > 0 {
			m.storageInput = m.storageInput[:len(m.storageInput)-1]
			m.applyStorageInput()
		}
	default:
		// Accept digits only
		if len(msg.String()) == 1 && msg.String()[0] >= '0' && msg.String()[0] <= '9' {
			m.storageInput += msg.String()
			m.applyStorageInput()
		}
	}
	return m, nil
}

func (m *model) applyStorageInput() {
	val := 0
	for _, c := range m.storageInput {
		val = val*10 + int(c-'0')
	}
	if m.storageCursor == 0 {
		m.volumeGB = val
	} else if m.storageCursor == 1 {
		m.containerDiskGB = val
	}
}
```

**Step 4: Add the storage screen view**

```go
func (m model) viewConfigStorage() string {
	s := renderLogo() + "\n\n"
	s += renderDivider(0) + "\n\n"

	tierColor := lipgloss.NewStyle().Foreground(m.selectedTier.Color).Bold(true)
	s += titleStyle.Render("Storage Configuration") + "  "
	s += tierColor.Render(fmt.Sprintf("%s %s", m.selectedTier.Emoji, m.selectedTier.Label)) + "\n"
	s += subtitleStyle.Render("Customize disk sizes (or press Enter to use defaults)") + "\n\n"

	// Look up tier defaults for display
	tierDefaults := map[string][2]int{
		"starter": {20, 20},
		"pro":     {50, 40},
		"beast":   {100, 50},
	}
	defaults := tierDefaults[m.selectedTier.Name]
	volDefault, diskDefault := defaults[0], defaults[1]

	volDisplay := fmt.Sprintf("%d GB", volDefault)
	if m.volumeGB > 0 {
		volDisplay = fmt.Sprintf("%d GB", m.volumeGB)
	}
	diskDisplay := fmt.Sprintf("%d GB", diskDefault)
	if m.containerDiskGB > 0 {
		diskDisplay = fmt.Sprintf("%d GB", m.containerDiskGB)
	}

	items := []struct {
		label, value, help string
	}{
		{"Volume (persistent)", volDisplay, "Persists across stop/resume"},
		{"Container Disk", diskDisplay, "Temporary, lost on terminate"},
		{"▶ Launch with these settings", "", ""},
	}

	for i, item := range items {
		prefix := "  "
		style := normalStyle
		if i == m.storageCursor {
			prefix = lipgloss.NewStyle().Foreground(catCyan).Bold(true).Render("▸ ")
			style = selectedStyle
		}
		if i < 2 {
			s += prefix + style.Render(fmt.Sprintf("%-22s", item.label)) + "  " + valueStyle.Render(item.value)
			if i == m.storageCursor {
				s += dimStyle.Render("  ← type number")
			}
			s += "\n" + strings.Repeat(" ", 4) + dimStyle.Render(item.help) + "\n\n"
		} else {
			s += prefix + successStyle.Render(item.label) + "\n"
		}
	}

	s += "\n" + helpStyle.Render("↑/↓ navigate  0-9 set size  enter confirm  esc back")
	return s
}
```

**Step 5: Wire the view into the main View() method**

In the `View()` method switch, add:

```go
	case screenConfigStorage:
		content = m.viewConfigStorage()
```

**Step 6: Reset storageInput when cursor moves between fields**

In `updateStorage`, when "up"/"down" changes `storageCursor`, clear `storageInput`:

```go
	case "up", "k":
		if m.storageCursor > 0 {
			m.storageCursor--
			m.storageInput = ""
		}
	case "down", "j":
		if m.storageCursor < 2 {
			m.storageCursor++
			m.storageInput = ""
		}
```

**Step 7: Commit**

```bash
git add internal/gpu/tui.go
git commit -m "feat(gpu): add storage configuration screen to TUI"
```

---

### Task 6: Add storage flags to CLI

**Files:**
- Modify: `internal/cmd/gpu.go`

**Step 1: Add CLI flag variables**

Add to the `var` block at the top:

```go
var (
	gpuTier           string
	gpuTemplate       string
	gpuPath           string
	gpuVolumeGB       int
	gpuContainerDiskGB int
)
```

**Step 2: Register flags on launch and root gpu commands**

In `init()`, add:

```go
	// Storage flags on root gpu command
	gpuCmd.Flags().IntVar(&gpuVolumeGB, "volume", 0, "Persistent volume size in GB (0 = tier default)")
	gpuCmd.Flags().IntVar(&gpuContainerDiskGB, "disk", 0, "Container disk size in GB (0 = tier default)")

	// Storage flags on launch subcommand
	gpuLaunchCmd.Flags().IntVar(&gpuVolumeGB, "volume", 0, "Persistent volume size in GB (0 = tier default)")
	gpuLaunchCmd.Flags().IntVar(&gpuContainerDiskGB, "disk", 0, "Container disk size in GB (0 = tier default)")
```

**Step 3: Pass storage to Launch() in gpuLaunchCmd**

Update the `Launch` call:

```go
	session, err := mgr.Launch(gpuTier, gpuTemplate, pubKey, gpuVolumeGB, gpuContainerDiskGB)
```

**Step 4: Show storage in the launch success output**

Update the `ShowBox` call to include storage info:

```go
	content := fmt.Sprintf(
		"Pod ID:   %s\nTier:     %s\nGPU:      %s\nStatus:   %s",
		session.PodID, session.Tier, session.GPUType, session.Status,
	)
	if session.VolumeGB != nil {
		content += fmt.Sprintf("\nVolume:   %d GB", *session.VolumeGB)
	}
	if session.ContainerDiskGB != nil {
		content += fmt.Sprintf("\nDisk:     %d GB", *session.ContainerDiskGB)
	}
	ShowBox("GPU Session", content)
```

**Step 5: Show storage in status command**

In `gpuStatusCmd`, add after the template line:

```go
	if session.VolumeGB != nil {
		content += fmt.Sprintf("\nVolume:   %d GB", *session.VolumeGB)
	}
	if session.ContainerDiskGB != nil {
		content += fmt.Sprintf("\nDisk:     %d GB", *session.ContainerDiskGB)
	}
```

(Note: this depends on the control plane returning these fields in the session JSON, which it will after Task 2.)

**Step 6: Update CLI help text**

In the `gpuCmd` Long description, add a STORAGE section after TIERS:

```
STORAGE:
  Each tier has default storage sizes. Use --volume and --disk to override:
    --volume 100     Persistent volume (survives stop/resume, lost on kill)
    --disk 50        Container disk (temporary scratch space)
  Values below the tier default are raised to the tier minimum.
```

**Step 7: Commit**

```bash
git add internal/cmd/gpu.go
git commit -m "feat(gpu): add --volume and --disk flags to GPU CLI commands"
```

---

### Task 7: Add storage params to MCP tools

**Files:**
- Modify: `internal/gpu/mcp_tools.go`

**Step 1: Update `gpu_launch` tool definition**

Add `volume_gb` and `container_disk_gb` to the `gpu_launch` InputSchema properties:

```go
{
	Name:        "gpu_launch",
	Description: "Launch a GPU pod for heavy compute (ML training, inference). Requires tier and template_id. Optionally set volume_gb and container_disk_gb for extra storage.",
	InputSchema: map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"tier": map[string]interface{}{
				"type":        "string",
				"description": "GPU tier: starter (RTX 4090 ~$0.39/hr), pro (A100 80GB ~$1.64/hr), or beast (H100 80GB ~$3.49/hr)",
				"enum":        []string{"starter", "pro", "beast"},
			},
			"template_id": map[string]interface{}{
				"type":        "string",
				"description": "Template ID for the GPU pod (use gpu_templates to list)",
			},
			"volume_gb": map[string]interface{}{
				"type":        "integer",
				"description": "Persistent volume size in GB. Survives stop/resume. Tier defaults: starter=20, pro=50, beast=100. Values below tier default are raised automatically.",
			},
			"container_disk_gb": map[string]interface{}{
				"type":        "integer",
				"description": "Container disk size in GB (scratch space). Tier defaults: starter=20, pro=40, beast=50. Values below tier default are raised automatically.",
			},
		},
		"required": []string{"tier", "template_id"},
	},
},
```

**Step 2: Update `handleGPULaunch` to extract and pass storage params**

```go
func handleGPULaunch(m *Manager, args map[string]interface{}) MCPToolResult {
	tier, _ := args["tier"].(string)
	if tier == "" {
		return mcpError("tier is required (starter, pro, or beast)")
	}
	templateID, _ := args["template_id"].(string)
	if templateID == "" {
		return mcpError("template_id is required")
	}

	pubKey, err := EnsureSSHKey(m.SSHKeyPath())
	if err != nil {
		return mcpError(fmt.Sprintf("SSH key error: %v", err))
	}

	volumeGB, _ := args["volume_gb"].(float64)
	containerDiskGB, _ := args["container_disk_gb"].(float64)

	session, err := m.Launch(tier, templateID, pubKey, int(volumeGB), int(containerDiskGB))
	if err != nil {
		return mcpError(fmt.Sprintf("Launch failed: %v", err))
	}

	result := map[string]interface{}{
		"success":  true,
		"pod_id":   session.PodID,
		"tier":     session.Tier,
		"gpu_type": session.GPUType,
		"status":   session.Status,
	}
	if session.VolumeGB != nil {
		result["volume_gb"] = *session.VolumeGB
	}
	if session.ContainerDiskGB != nil {
		result["container_disk_gb"] = *session.ContainerDiskGB
	}
	if session.SSHHost != "" {
		result["ssh_host"] = session.SSHHost
		result["ssh_port"] = session.SSHPort
	}
	return mcpSuccess(result)
}
```

**Step 3: Update `gpu_status` handler to include storage in output**

In `handleGPUStatus`, add after setting `"template_name"`:

```go
	if session.VolumeGB != nil {
		result["volume_gb"] = *session.VolumeGB
	}
	if session.ContainerDiskGB != nil {
		result["container_disk_gb"] = *session.ContainerDiskGB
	}
```

**Step 4: Update `gpu_templates` handler to show tier storage defaults**

Update the tiers map in `handleGPUTemplates`:

```go
	"tiers": map[string]interface{}{
		"starter": "RTX 4090 (~$0.39/hr) - 20GB volume + 20GB disk default",
		"pro":     "A100 80GB (~$1.64/hr) - 50GB volume + 40GB disk default",
		"beast":   "H100 80GB (~$3.49/hr) - 100GB volume + 50GB disk default",
	},
```

**Step 5: Commit**

```bash
git add internal/gpu/mcp_tools.go
git commit -m "feat(gpu): add volume_gb and container_disk_gb to MCP gpu_launch tool"
```

---

### Task 8: Build and verify compilation

**Step 1: Build the Go binary**

Run:
```bash
make build
```

Expected: Clean build with no errors.

**Step 2: Verify TUI launches (smoke test)**

Run:
```bash
./wee gpu --help
```

Expected: Help text includes `--volume` and `--disk` flags, plus the STORAGE section.

**Step 3: Verify MCP tool list includes storage params**

Run:
```bash
./wee mcp-server
```

Then send a `tools/list` JSON-RPC request and verify `gpu_launch` includes `volume_gb` and `container_disk_gb` properties.

**Step 4: Commit (if any fixups needed)**

```bash
git add -A
git commit -m "fix(gpu): compilation fixes for storage params"
```

---

### Task 9: Update CLAUDE.md documentation

**Files:**
- Modify: `CLAUDE.md`

**Step 1: Update the GPU tiers table**

In the `TIERS:` section of the gpu command docs, add storage defaults:

```
TIERS:
  starter   RTX 4090 24GB     ~$0.39/hr   20GB vol + 20GB disk
  pro       A100 80GB         ~$1.64/hr   50GB vol + 40GB disk
  beast     H100 80GB HBM3    ~$3.49/hr   100GB vol + 50GB disk
```

**Step 2: Add storage examples**

Add to the EXAMPLES section:

```
STORAGE:
  wee gpu launch --tier starter --template abc123 --volume 100    # 100GB persistent volume
  wee gpu launch --tier pro --template abc123 --disk 200          # 200GB container disk
```

**Step 3: Add storage params to MCP tool docs**

In the MCP tools section, note that `gpu_launch` accepts optional `volume_gb` and `container_disk_gb`.

**Step 4: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: add GPU storage configuration to CLAUDE.md"
```
