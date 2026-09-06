# GPU TUI Control Center

## Overview

Enhance the GPU TUI from a launch-only wizard into a full sidebar-based control center for managing GPU pods, resizing storage, and managing RunPod Network Volumes.

## Current State

The TUI is a linear launch flow: Tier -> Template -> Storage -> Launch -> Done -> Exit. All other pod management (status, stop, resume, kill) is CLI-only. No support for disk resizing or network volumes.

## Design

### Screen Flow

```
                    +-------------+
         +---------|  Dashboard  |-----------+
         |         +------+------+           |
         |                |                  |
    [Launch New]     [Resize Disk]      [Volumes]
         |                |                  |
   +-----v-----+   +-----v-----+   +--------v------+
   | Select Tier|   |  Resize   |   | Volume List   |
   +-----+------+   |  Storage  |   +--------+------+
         |          +-----------+            |
   +-----v------+                   +--------v------+
   |  Select    |                   | Create New    |
   |  Template  |                   |   Volume      |
   +-----+------+                   +---------------+
         |
   +-----v------+
   |  Config    |
   |  Storage   |
   +-----+------+
         |
   +-----v------+
   | Launching  |--- Back to Dashboard
   +------------+
```

### Sidebar Layout

The TUI uses a two-pane layout: a fixed-width sidebar on the left with navigation and context-sensitive actions, and a content panel on the right.

```
+-- wee gpu ----------+------------------------------------------+
|                     |                                           |
|  > Dashboard        |  (content panel - changes per screen)    |
|    Volumes          |                                           |
|    Launch New       |                                           |
|                     |                                           |
|  ----------------   |                                           |
|                     |                                           |
|  Pod Actions        |                                           |
|    Stop             |                                           |
|    Resize Disk      |                                           |
|    Kill             |                                           |
|                     |                                           |
|  ----------------   |                                           |
|    q quit           |                                           |
+---------------------+------------------------------------------+
```

Focus toggles between sidebar and content panel with `tab`. Focused zone has highlighted border, unfocused is dim. Arrow keys navigate within the focused zone.

### Dashboard Panel

Shows the most relevant pod (running > creating > stopped > exited). Status dot: green=running, yellow=creating, red=stopped/exited, x=error.

Displays: pod ID, GPU type, tier, template, uptime, cost (total + per hour), volume/disk sizes, SSH connection string.

Auto-refreshes every 5 seconds via tea.Tick on the dashboard screen only.

When no pods exist, shows "No active GPU pods" with launch prompt.

Sidebar actions are context-sensitive: stop only shows when running, resume only when stopped.

Other sessions shown as "+N other sessions" info label (not interactive).

### Network Volumes Panel

Right panel shows volumes grouped by data center:

```
US-TX-3
> my-models        100 GB   mounted
  training-data     50 GB   free

EU-RO-1
  checkpoints      200 GB   free
```

Sidebar actions: Create New, Mount to Pod, Delete.

**Create** opens inline form: name (text input), size GB (number input), data center (scrollable picker). Standard data centers from RunPod.

**Delete** shows confirmation prompt.

**Mount to Pod** carries the selected volume into the launch flow, locking the data center. Tier selection shows: "Launching in US-TX-3 (volume: my-models)".

When launching with a network volume, it replaces the regular volumeInGb parameter with networkVolumeId and sets the data center.

### Resize Disk Panel

Shows current volume and container disk sizes with number input fields to set new values.

Constraints:
- Sizes can only increase, never shrink
- Input below current shows inline validation error
- Volume-only changes apply immediately
- Container disk changes show restart warning and require confirmation
- After apply, returns to dashboard

### Launch Flow

Existing flow (tier -> template -> storage -> launching) embedded in the content panel. After successful launch, returns to dashboard instead of exiting.

When a network volume is selected (via volumes screen), the launch flow:
- Shows locked data center indicator on tier screen
- Skips the volume size field in storage config (network volume replaces it)
- Passes networkVolumeId to the launch API

## Backend Changes

### RunPod Client (runpod_client.py)

New methods:

```python
def edit_pod(self, pod_id, volume_gb=None, container_disk_gb=None) -> dict
    # GraphQL: podEditJob mutation

def list_network_volumes(self) -> list[NetworkVolume]
    # GraphQL: myself { networkVolumes { id, name, size, dataCenterId } }

def create_network_volume(self, name, size_gb, data_center_id) -> NetworkVolume
    # GraphQL: createNetworkVolume mutation

def delete_network_volume(self, volume_id) -> dict
    # GraphQL: deleteNetworkVolume mutation
```

Modify create_pod() to accept optional network_volume_id. When provided, replaces volumeInGb and adds dataCenterId constraint.

New dataclass:

```python
@dataclass
class NetworkVolume:
    id: str
    name: str
    size: int
    data_center_id: str
```

### Control Plane API

New endpoints:

```
POST   /api/gpu/sessions/{id}/resize/     -> calls edit_pod
GET    /api/gpu/network-volumes/          -> calls list_network_volumes
POST   /api/gpu/network-volumes/          -> calls create_network_volume
DELETE /api/gpu/network-volumes/{id}/     -> calls delete_network_volume
```

Modify launch endpoint to accept network_volume_id.

### Go Client (gpu.go)

New struct:

```go
type NetworkVolume struct {
    ID           string `json:"id"`
    Name         string `json:"name"`
    SizeGB       int    `json:"size_gb"`
    DataCenterID string `json:"data_center_id"`
    MountedPodID string `json:"mounted_pod_id,omitempty"`
}
```

New methods:

```go
func (m *Manager) ResizePod(sessionID int, volumeGB, containerDiskGB int) error
func (m *Manager) ListNetworkVolumes() ([]NetworkVolume, error)
func (m *Manager) CreateNetworkVolume(name string, sizeGB int, dataCenterID string) (*NetworkVolume, error)
func (m *Manager) DeleteNetworkVolume(volumeID string) error
```

Modify Launch() signature to accept optional networkVolumeID string.

### MCP Tools (mcp_tools.go)

New tools: gpu_resize, gpu_list_volumes, gpu_create_volume, gpu_delete_volume.

## File Structure

```
internal/gpu/
  tui.go              # Main model, Update loop, sidebar, focus management
  tui_dashboard.go    # Dashboard panel: pod status, live polling
  tui_volumes.go      # Volumes panel: list, create form, delete confirm
  tui_launch.go       # Launch flow: tier, template, storage config
  tui_resize.go       # Resize panel: inputs, validation, confirm
  tui_styles.go       # All lipgloss styles, colors, shared renderers
  gpu.go              # Manager client (existing + new methods)
  ssh.go              # SSH (unchanged)
  sync.go             # File sync (unchanged)
  mcp_tools.go        # MCP tools (existing + new)
```

## Implementation Order

1. Refactor: extract styles and launch flow views into separate files
2. Build sidebar layout and focus system in tui.go
3. Dashboard panel with live polling
4. Lifecycle controls (stop/resume/kill) wired to sidebar actions
5. Backend: resize endpoint (Python + Go client)
6. Resize disk panel
7. Backend: network volumes CRUD (Python + Go client)
8. Volumes panel with list, create, delete
9. Mount-to-pod flow (volume -> launch integration)
10. MCP tools for resize and volumes
