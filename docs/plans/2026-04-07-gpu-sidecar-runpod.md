# GPU Sidecar via RunPod

**Date**: 2026-04-07
**Status**: Design complete, ready for implementation

## Summary

Add GPU sidecar support to wee sandboxes. Users connect their RunPod API key as a connector in wee.cat, then launch GPU VMs alongside their existing sandbox for ML training (LLMs, voice models, etc.). The sandbox remains the primary workspace; the GPU box is ephemeral and on-demand.

## Motivation

Users want to run heavy ML workloads (LLM fine-tuning, voice model training) inside their wee environment. Current sandboxes have no GPU. Rather than replacing the sandbox, we add a GPU sidecar that spins up on-demand — keeping the cheap always-on sandbox for coding and the expensive GPU box only for training.

## Architecture Overview

```
┌──────────────┐         ┌─────────────────┐        ┌──────────┐
│  wee.cat     │         │  Sandbox        │        │  RunPod  │
│  (control    │────────▶│  danny.wee.cat  │───────▶│  GPU Pod │
│   plane)     │         │                 │  SSH   │          │
│              │         │  wee gpu shell ─│───────▶│          │
│  Holds       │         │  wee gpu run ───│───────▶│          │
│  RunPod key  │         │  wee gpu pull ◀─│────────│          │
│  Proxies API │         │                 │        │          │
└──────────────┘         └─────────────────┘        └──────────┘
```

Key decisions:
- **Sidecar model**: GPU box runs alongside the sandbox, not replacing it
- **Proxied through wee.cat**: Sandbox never sees the RunPod API key
- **RunPod templates**: Use RunPod's existing templates, no custom image needed initially
- **User brings their own key**: No spending enforcement, but we show live cost from RunPod API

## User Experience

### TUI (interactive)

```
wee gpu
```

```
┌─ Launch GPU Machine ─────────────────────────────────────┐
│                                                           │
│  Select GPU tier:                                         │
│  > Starter    T4 16GB      ~$0.20/hr                     │
│    Pro        A100 80GB    ~$1.50/hr                      │
│    Beast      H100 80GB    ~$3.50/hr                      │
│                                                           │
│  Select template:                                         │
│  > PyTorch 2.4 + CUDA 12.4     (general ML)              │
│    Hugging Face Transformers     (LLM fine-tuning)        │
│    AudioCraft                    (voice/music)            │
│    ComfyUI                       (image gen)              │
│    Jupyter Lab + PyTorch         (notebooks)              │
│    Custom template ID...                                  │
│                                                           │
│  up/down select  enter launch  q quit                     │
└───────────────────────────────────────────────────────────┘
```

### CLI (direct)

```bash
wee gpu launch --tier pro --template pytorch
wee gpu status
wee gpu shell
wee gpu run python train.py --epochs 10
wee gpu push /app/data/
wee gpu pull /app/output/
wee gpu stop
wee gpu resume
wee gpu kill
```

### Live Status TUI

```
┌─ GPU Status ─────────────────────────────────────┐
│  Running    A100-80GB (Pro tier)                  │
│  Template:  PyTorch 2.4 + CUDA 12.4              │
│  Uptime:    1h 12m                                │
│  Cost:      $1.97 ($1.64/hr)                      │
│                                                   │
│  GPU Util: ########.. 82%                         │
│  VRAM:     ######.... 61.2 / 80 GB               │
│                                                   │
│  [s] shell  [x] stop  [q] back                   │
└───────────────────────────────────────────────────┘
```

## wee.cat Backend (Django)

### Models

```python
class GPUConnector(models.Model):
    user = models.ForeignKey(User, on_delete=models.CASCADE)
    provider = models.CharField(max_length=50, default="runpod")  # future: lambda, vast.ai
    api_key = models.CharField(max_length=255)  # encrypted at rest
    is_active = models.BooleanField(default=True)
    created_at = models.DateTimeField(auto_now_add=True)

class GPUSession(models.Model):
    sandbox = models.ForeignKey(Sandbox, on_delete=models.CASCADE)
    connector = models.ForeignKey(GPUConnector, on_delete=models.CASCADE)
    pod_id = models.CharField(max_length=100)
    template_name = models.CharField(max_length=100)
    tier = models.CharField(max_length=20)       # starter/pro/beast
    gpu_type = models.CharField(max_length=50)    # T4, A100-80GB, H100
    status = models.CharField(max_length=20)      # creating/running/stopped/terminated
    ssh_host = models.CharField(max_length=255, null=True)
    ssh_port = models.IntegerField(null=True)
    started_at = models.DateTimeField(null=True)
    stopped_at = models.DateTimeField(null=True)
    # Cost queried live from RunPod API, not stored
```

### API Endpoints

```
POST   /api/gpu/templates    List available RunPod templates (cached)
POST   /api/gpu/launch       Spin up a pod (tier + template + sandbox SSH pubkey)
POST   /api/gpu/stop         Stop pod (keeps volume, stops billing)
POST   /api/gpu/resume       Resume stopped pod
DELETE /api/gpu/terminate     Destroy pod entirely
GET    /api/gpu/status        Current pod status + live cost from RunPod
GET    /api/gpu/sessions      History of GPU sessions for this sandbox
```

### Launch Flow

```
Sandbox: wee gpu launch --tier pro --template pytorch

  1. Sandbox generates SSH keypair (if not exists)
  2. Sandbox POSTs to wee.cat /api/gpu/launch
     {tier: "pro", template: "pytorch", ssh_public_key: "ssh-ed25519 ..."}
  3. wee.cat looks up user's GPUConnector for RunPod API key
  4. wee.cat calls RunPod API: POST /v2/pods
     {gpu: "A100-80GB", template_id: "...", env: {PUBLIC_KEY: "..."}}
  5. RunPod returns {pod_id, host, ssh_port}
  6. wee.cat creates GPUSession record
  7. wee.cat returns {host, ssh_port, pod_id} to sandbox
  8. Sandbox polls SSH until reachable
  9. Ready!
```

## Wee Go Binary (Sandbox Side)

### New Package Structure

```
internal/
  gpu/
    gpu.go              # Core GPU manager - talks to wee.cat API
    ssh.go              # SSH connection management (key gen, connect, exec)
    sync.go             # rsync wrapper for push/pull
    tui.go              # Bubbletea TUI for tier/template picker + status
    mcp_tools.go        # MCP tool definitions for Claude agent access
```

### GPU Manager

```go
type GPUManager struct {
    weeCatURL  string  // https://wee.cat
    sandboxID  string  // from WEE_SUBDOMAIN env
    authToken  string  // sandbox auth token
    sshKeyPath string  // ~/.ssh/wee-gpu-key
}

func (g *GPUManager) Launch(tier, template string) (*GPUSession, error)
func (g *GPUManager) Stop() error
func (g *GPUManager) Resume() error
func (g *GPUManager) Kill() error
func (g *GPUManager) Status() (*GPUStatus, error)
func (g *GPUManager) Shell() error           // interactive SSH
func (g *GPUManager) Run(cmd string) error   // remote exec via SSH
func (g *GPUManager) Push(path string) error // rsync to GPU box
func (g *GPUManager) Pull(path string) error // rsync from GPU box
```

### MCP Tools (for Claude Agent Access)

```go
tools := []mcp.Tool{
    {Name: "gpu_launch",  Description: "Launch a GPU machine for ML training",  Params: {tier, template}},
    {Name: "gpu_run",     Description: "Execute a command on the GPU machine",  Params: {command}},
    {Name: "gpu_push",    Description: "Sync files to the GPU machine",         Params: {path}},
    {Name: "gpu_pull",    Description: "Pull files from the GPU machine",       Params: {path}},
    {Name: "gpu_status",  Description: "Get GPU status, utilization, and cost"},
    {Name: "gpu_stop",    Description: "Stop the GPU machine"},
}
```

### Agent Workflow Example

A Claude agent conversation could autonomously:

```
User: "Fine-tune Llama 3 on my dataset in /app/data/"

Claude:
  -> gpu_launch(tier: "pro", template: "huggingface")
  -> gpu_push(path: "/app/data")
  -> gpu_run("python -m transformers ... --data /app/data")
  ... monitors training ...
  -> gpu_pull(path: "/app/output")
  -> gpu_stop()

"Done! Model checkpoint saved to /app/output/. GPU cost: $4.12"
```

## File Sync Strategy

- **On launch**: No auto-sync (user controls what goes to the GPU box)
- **`wee gpu push [path]`**: rsync from sandbox to GPU, defaults to /app
- **`wee gpu pull [path]`**: rsync from GPU to sandbox, defaults to /app
- **Why not live mount**: NFS/SSHFS adds latency during training I/O, hurts performance

## GPU Tiers

| Tier    | GPU         | VRAM  | Approx Cost |
|---------|-------------|-------|-------------|
| Starter | T4          | 16GB  | ~$0.20/hr   |
| Pro     | A100 80GB   | 80GB  | ~$1.50/hr   |
| Beast   | H100 80GB   | 80GB  | ~$3.50/hr   |

Prices from RunPod, shown to user but not enforced (their key, their spend).

## Future Considerations

- **Additional providers**: Lambda Labs, Vast.ai, custom GPU servers
- **Custom wee-gpu image**: Bake in wee-gpu-agent for health reporting, auto-sync, metrics streaming
- **Multi-GPU**: Support 2x/4x/8x GPU pods for distributed training
- **Persistent volumes**: Keep datasets across GPU sessions to avoid re-upload
- **Spending alerts**: Optional notifications when cost exceeds user-defined threshold
- **GPU in wee.cat dashboard**: Show active GPU sessions, cost history, utilization graphs
