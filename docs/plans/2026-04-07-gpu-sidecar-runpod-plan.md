# GPU Sidecar via RunPod — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add GPU sidecar support so sandbox users can launch RunPod GPU VMs for ML training, with both TUI and CLI interfaces, proxied through wee.cat.

**Architecture:** Sandbox (`wee` Go binary) talks to wee.cat Django backend, which proxies RunPod API calls. SSH connection between sandbox and GPU pod for shell/run/sync. MCP tools expose GPU to Claude agents.

**Tech Stack:** Django (wee.cat backend), Go + Cobra + Bubbletea (wee CLI/TUI), RunPod API, SSH/rsync

**Design Doc:** `docs/plans/2026-04-07-gpu-sidecar-runpod.md`

---

## Part 0: wee.cat Dashboard Frontend

### Task 0: Add "GPU Providers" Section to Settings Page

**Files:**
- Modify: `wee.cat/frontend/src/pages/dashboard/settings.astro`

**Step 1: Add the GPU Providers section HTML**

Add a new section between Integrations and Danger Zone in `settings.astro`. Insert after line 35 (`</section>`) and before the Danger Zone `<hr>`:

```astro
    <hr class="divider" />

    <section class="settings-section">
      <h2 class="section-label">GPU Providers</h2>
      <div class="gpu-provider-list" id="gpu-provider-list">
        <p class="muted">Loading...</p>
      </div>
    </section>
```

**Step 2: Add JavaScript for GPU provider management**

Add these functions in the `<script>` tag, after the `loadIntegrations()` call:

```javascript
  async function loadGPUProviders() {
    const listEl = document.getElementById("gpu-provider-list");
    if (!listEl) return;

    try {
      const res = await fetch("/api/gpu/connectors/", { credentials: "include" });
      if (!res.ok) throw new Error("Failed to load GPU providers");

      const connectors = await res.json();

      const rowStyle = "display:flex; align-items:center; justify-content:space-between; padding:14px 0; gap:16px;";
      const rowBorderStyle = rowStyle + " border-top:1px solid var(--color-border);";
      const leftStyle = "display:flex; align-items:center; gap:14px;";
      const infoStyle = "display:flex; flex-direction:column; gap:3px;";
      const nameStyle = "font-size:0.88rem; font-weight:500;";
      const metaStyle = "font-size:0.8rem; color:var(--color-text-muted);";
      const iconStyle = "width:36px; height:36px; border-radius:50%; background:var(--color-border); display:flex; align-items:center; justify-content:center; font-size:1.1rem; flex-shrink:0;";
      const btnStyle = "display:inline-flex; align-items:center; padding:6px 16px; border-radius:var(--radius); font-size:0.8rem; font-weight:500; font-family:inherit; cursor:pointer; transition:all 0.15s; white-space:nowrap; flex-shrink:0;";
      const btnConnectStyle = btnStyle + " border:1px solid var(--color-border); background:var(--color-bg); color:var(--color-text);";
      const btnDisconnectStyle = btnStyle + " border:none; background:none; color:var(--color-text-muted);";

      let html = "";

      if (connectors.length === 0) {
        html += `
        <div style="padding:20px 0; text-align:center;">
          <p style="font-size:0.88rem; color:var(--color-text); margin:0 0 6px;">No GPU providers connected</p>
          <p style="${metaStyle} margin:0 0 18px; line-height:1.5;">Connect a GPU provider to launch GPU machines from your sandboxes.</p>
          <button style="${btnConnectStyle}" onclick="showGPUKeyModal()"
            onmouseover="this.style.borderColor='var(--color-text-muted)'"
            onmouseout="this.style.borderColor='var(--color-border)'">+ Add RunPod API Key</button>
        </div>`;
      } else {
        for (let i = 0; i < connectors.length; i++) {
          const c = connectors[i];
          const connDate = new Date(c.created_at).toLocaleDateString();
          const providerLabel = c.provider === "runpod" ? "RunPod" : c.provider;
          const style = i === 0 ? rowStyle : rowBorderStyle;
          html += `
          <div style="${style}">
            <div style="${leftStyle}">
              <div style="${iconStyle}">🖥️</div>
              <div style="${infoStyle}">
                <span style="${nameStyle}">${escapeHtml(providerLabel)}</span>
                <span style="${metaStyle}">Connected ${connDate}</span>
              </div>
            </div>
            <button style="${btnDisconnectStyle}" data-gpu-disconnect-id="${c.id}"
              onmouseover="this.style.color='#ef4444'"
              onmouseout="this.style.color='var(--color-text-muted)'">Disconnect</button>
          </div>`;
        }
        html += `
        <div style="${rowBorderStyle}">
          <span style="${metaStyle}">Connect another GPU provider</span>
          <button style="${btnConnectStyle}" onclick="showGPUKeyModal()"
            onmouseover="this.style.borderColor='var(--color-text-muted)'"
            onmouseout="this.style.borderColor='var(--color-border)'">+ Add API Key</button>
        </div>`;
      }
      listEl.innerHTML = html;

      listEl.querySelectorAll("[data-gpu-disconnect-id]").forEach((btn) => {
        btn.addEventListener("click", () => disconnectGPU(parseInt(btn.dataset.gpuDisconnectId)));
      });
    } catch {
      listEl.innerHTML = `<p class="muted">Could not load GPU providers</p>`;
    }
  }

  function showGPUKeyModal() {
    // Remove existing modal if any
    const existing = document.getElementById("gpu-key-modal");
    if (existing) existing.remove();

    const overlayStyle = "position:fixed; inset:0; background:rgba(0,0,0,0.5); z-index:999; display:flex; align-items:center; justify-content:center;";
    const modalStyle = "background:var(--color-bg); border:1px solid var(--color-border); border-radius:12px; padding:32px; max-width:440px; width:90%; box-shadow:0 8px 30px rgba(0,0,0,0.2);";
    const titleStyle = "font-size:1.1rem; font-weight:500; margin:0 0 8px;";
    const descStyle = "font-size:0.84rem; color:var(--color-text-muted); margin:0 0 20px; line-height:1.5;";
    const inputStyle = "width:100%; padding:10px 14px; border:1px solid var(--color-border); border-radius:var(--radius); font-size:0.85rem; font-family:var(--font-mono); background:var(--color-bg); color:var(--color-text); box-sizing:border-box; outline:none;";
    const btnRowStyle = "display:flex; justify-content:flex-end; gap:10px; margin-top:20px;";
    const btnCancelStyle = "padding:8px 16px; border:1px solid var(--color-border); border-radius:var(--radius); font-size:0.84rem; font-weight:500; cursor:pointer; background:none; color:var(--color-text); font-family:inherit;";
    const btnSaveStyle = "padding:8px 20px; border:none; border-radius:var(--radius); font-size:0.84rem; font-weight:500; cursor:pointer; background:var(--color-text); color:var(--color-bg); font-family:inherit;";

    const modal = document.createElement("div");
    modal.id = "gpu-key-modal";
    modal.style.cssText = overlayStyle;
    modal.innerHTML = `
      <div style="${modalStyle}">
        <h3 style="${titleStyle}">Connect RunPod</h3>
        <p style="${descStyle}">Paste your RunPod API key. You can find it at <a href="https://www.runpod.io/console/user/settings" target="_blank" style="color:inherit; text-decoration:underline;">runpod.io/console/user/settings</a></p>
        <input type="password" id="gpu-key-input" placeholder="rp_..." style="${inputStyle}" />
        <div style="${btnRowStyle}">
          <button style="${btnCancelStyle}" onclick="document.getElementById('gpu-key-modal').remove()">Cancel</button>
          <button style="${btnSaveStyle}" onclick="saveGPUKey()">Connect</button>
        </div>
      </div>
    `;
    modal.addEventListener("click", (e) => { if (e.target === modal) modal.remove(); });
    document.body.appendChild(modal);
    document.getElementById("gpu-key-input").focus();
  }

  async function saveGPUKey() {
    const input = document.getElementById("gpu-key-input");
    const key = input?.value?.trim();
    if (!key) { showToast("Please enter an API key.", true); return; }

    try {
      const res = await fetch("/api/gpu/connectors/connect/", {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ provider: "runpod", api_key: key }),
      });
      if (res.ok || res.status === 201) {
        document.getElementById("gpu-key-modal")?.remove();
        showToast("RunPod connected!");
        loadGPUProviders();
      } else {
        const err = await res.json().catch(() => ({}));
        showToast(err.detail || "Failed to save API key.", true);
      }
    } catch {
      showToast("Failed to connect.", true);
    }
  }

  async function disconnectGPU(connectorId) {
    if (!confirm("Disconnect this GPU provider? Active GPU sessions will be terminated.")) return;
    try {
      const res = await fetch(`/api/gpu/connectors/${connectorId}/disconnect/`, {
        method: "DELETE",
        credentials: "include",
      });
      if (res.ok || res.status === 204) {
        showToast("GPU provider disconnected.");
        loadGPUProviders();
      } else {
        showToast("Failed to disconnect.", true);
      }
    } catch {
      showToast("Failed to disconnect.", true);
    }
  }

  loadGPUProviders();
```

**Step 3: Verify the page renders correctly**

```bash
cd /path/to/wee/wee.cat/frontend
npm run dev
# Open http://localhost:4321/dashboard/settings and verify the GPU Providers section appears
```

**Step 4: Commit**

```bash
git add wee.cat/frontend/src/pages/dashboard/settings.astro
git commit -m "feat: add GPU Providers section to settings page"
```

---

## Part 1: wee.cat Django Backend

### Task 1: GPU Django App Setup

**Files:**
- Create: `wee.cat/backend/gpu/__init__.py`
- Create: `wee.cat/backend/gpu/models.py`
- Create: `wee.cat/backend/gpu/admin.py`
- Create: `wee.cat/backend/gpu/apps.py`
- Modify: `wee.cat/backend/config/settings.py` (add to INSTALLED_APPS)
- Modify: `wee.cat/backend/config/urls.py` (add gpu urls)

**Step 1: Create the Django app skeleton**

```bash
cd /path/to/wee/wee.cat/backend
python manage.py startapp gpu
```

**Step 2: Create the models**

Create `wee.cat/backend/gpu/models.py`:

```python
from django.conf import settings
from django.db import models

from integrations.encryption import encrypt_token, decrypt_token


class GPUConnector(models.Model):
    """A user's GPU cloud provider credentials (e.g. RunPod API key)."""

    PROVIDER_CHOICES = [
        ("runpod", "RunPod"),
    ]

    user = models.ForeignKey(
        settings.AUTH_USER_MODEL,
        on_delete=models.CASCADE,
        related_name="gpu_connectors",
    )
    provider = models.CharField(max_length=50, choices=PROVIDER_CHOICES, default="runpod")
    encrypted_api_key = models.BinaryField(
        help_text="Fernet-encrypted API key for the GPU provider",
    )
    is_active = models.BooleanField(default=True)
    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)

    class Meta:
        unique_together = ("user", "provider")

    def __str__(self):
        return f"{self.user.username} - {self.provider}"

    def set_api_key(self, plaintext_key: str) -> None:
        self.encrypted_api_key = encrypt_token(plaintext_key)

    def get_api_key(self) -> str:
        return decrypt_token(bytes(self.encrypted_api_key))


class GPUSession(models.Model):
    """Tracks an active or past GPU pod session."""

    STATUS_CHOICES = [
        ("creating", "Creating"),
        ("running", "Running"),
        ("stopped", "Stopped"),
        ("terminated", "Terminated"),
        ("error", "Error"),
    ]

    sandbox = models.ForeignKey(
        "sandboxes.Sandbox",
        on_delete=models.CASCADE,
        related_name="gpu_sessions",
    )
    connector = models.ForeignKey(
        GPUConnector,
        on_delete=models.CASCADE,
        related_name="sessions",
    )
    pod_id = models.CharField(max_length=100, blank=True, default="")
    template_id = models.CharField(max_length=100)
    template_name = models.CharField(max_length=200)
    tier = models.CharField(max_length=20)
    gpu_type = models.CharField(max_length=50)
    status = models.CharField(max_length=20, choices=STATUS_CHOICES, default="creating")
    ssh_host = models.CharField(max_length=255, blank=True, default="")
    ssh_port = models.IntegerField(null=True, blank=True)
    error_message = models.TextField(blank=True, default="")
    started_at = models.DateTimeField(null=True, blank=True)
    stopped_at = models.DateTimeField(null=True, blank=True)
    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)

    class Meta:
        ordering = ["-created_at"]

    def __str__(self):
        return f"GPU {self.tier} ({self.status}) - {self.sandbox.subdomain}"
```

**Step 3: Register in settings and urls**

Add `"gpu"` to `INSTALLED_APPS` in `wee.cat/backend/config/settings.py`.

Add to `wee.cat/backend/config/urls.py`:
```python
path("api/gpu/", include("gpu.urls")),
```

**Step 4: Create and run migration**

```bash
cd /path/to/wee/wee.cat/backend
python manage.py makemigrations gpu
python manage.py migrate
```

**Step 5: Create admin registration**

Create `wee.cat/backend/gpu/admin.py`:
```python
from django.contrib import admin
from .models import GPUConnector, GPUSession

@admin.register(GPUConnector)
class GPUConnectorAdmin(admin.ModelAdmin):
    list_display = ("user", "provider", "is_active", "created_at")
    list_filter = ("provider", "is_active")

@admin.register(GPUSession)
class GPUSessionAdmin(admin.ModelAdmin):
    list_display = ("sandbox", "tier", "gpu_type", "status", "created_at")
    list_filter = ("status", "tier")
```

**Step 6: Commit**

```bash
git add wee.cat/backend/gpu/ wee.cat/backend/config/settings.py wee.cat/backend/config/urls.py
git commit -m "feat: add gpu Django app with GPUConnector and GPUSession models"
```

---

### Task 2: RunPod API Client

**Files:**
- Create: `wee.cat/backend/gpu/runpod_client.py`
- Create: `wee.cat/backend/gpu/tests/__init__.py`
- Create: `wee.cat/backend/gpu/tests/test_runpod_client.py`

**Step 1: Write tests for the RunPod client**

Create `wee.cat/backend/gpu/tests/test_runpod_client.py`:

```python
from unittest.mock import patch, MagicMock
from django.test import TestCase
from gpu.runpod_client import RunPodClient, GPUTier


class RunPodClientTest(TestCase):
    def setUp(self):
        self.client = RunPodClient(api_key="test-key")

    def test_gpu_tier_mapping(self):
        self.assertEqual(GPUTier.STARTER.gpu_id, "NVIDIA GeForce RTX 4090")
        self.assertEqual(GPUTier.PRO.gpu_id, "NVIDIA A100 80GB PCIe")
        self.assertEqual(GPUTier.BEAST.gpu_id, "NVIDIA H100 80GB HBM3")

    @patch("gpu.runpod_client.requests.get")
    def test_list_templates(self, mock_get):
        mock_get.return_value = MagicMock(
            status_code=200,
            json=MagicMock(return_value=[
                {"id": "tpl-1", "name": "PyTorch 2.4", "imageName": "runpod/pytorch:2.4"},
            ]),
        )
        templates = self.client.list_templates()
        self.assertEqual(len(templates), 1)
        self.assertEqual(templates[0]["name"], "PyTorch 2.4")

    @patch("gpu.runpod_client.requests.post")
    def test_create_pod(self, mock_post):
        mock_post.return_value = MagicMock(
            status_code=200,
            json=MagicMock(return_value={
                "id": "pod-123",
                "desiredStatus": "RUNNING",
                "machine": {"podHostId": "host-1"},
            }),
        )
        result = self.client.create_pod(
            tier=GPUTier.PRO,
            template_id="tpl-1",
            ssh_public_key="ssh-ed25519 AAAA...",
        )
        self.assertEqual(result["id"], "pod-123")

    @patch("gpu.runpod_client.requests.get")
    def test_get_pod_status(self, mock_get):
        mock_get.return_value = MagicMock(
            status_code=200,
            json=MagicMock(return_value={
                "id": "pod-123",
                "desiredStatus": "RUNNING",
                "runtime": {"uptimeInSeconds": 3600, "ports": [{"ip": "1.2.3.4", "publicPort": 22}]},
                "costPerHr": 1.64,
            }),
        )
        status = self.client.get_pod("pod-123")
        self.assertEqual(status["costPerHr"], 1.64)

    @patch("gpu.runpod_client.requests.post")
    def test_stop_pod(self, mock_post):
        mock_post.return_value = MagicMock(status_code=200, json=MagicMock(return_value={"id": "pod-123", "desiredStatus": "EXITED"}))
        result = self.client.stop_pod("pod-123")
        self.assertEqual(result["desiredStatus"], "EXITED")

    @patch("gpu.runpod_client.requests.post")
    def test_resume_pod(self, mock_post):
        mock_post.return_value = MagicMock(status_code=200, json=MagicMock(return_value={"id": "pod-123", "desiredStatus": "RUNNING"}))
        result = self.client.resume_pod("pod-123")
        self.assertEqual(result["desiredStatus"], "RUNNING")

    @patch("gpu.runpod_client.requests.delete")
    def test_terminate_pod(self, mock_delete):
        mock_delete.return_value = MagicMock(status_code=200)
        self.client.terminate_pod("pod-123")  # Should not raise
```

**Step 2: Run tests to verify they fail**

```bash
cd /path/to/wee/wee.cat/backend
python manage.py test gpu.tests.test_runpod_client -v 2
```

Expected: ImportError (module doesn't exist yet)

**Step 3: Implement RunPod client**

Create `wee.cat/backend/gpu/runpod_client.py`:

```python
"""RunPod API client for GPU pod management."""

import enum
import logging

import requests

logger = logging.getLogger(__name__)

RUNPOD_API_BASE = "https://api.runpod.io/v2"
RUNPOD_GRAPHQL = "https://api.runpod.io/graphql"


class GPUTier(enum.Enum):
    STARTER = ("starter", "NVIDIA GeForce RTX 4090", 24, 0.39)
    PRO = ("pro", "NVIDIA A100 80GB PCIe", 80, 1.64)
    BEAST = ("beast", "NVIDIA H100 80GB HBM3", 80, 3.49)

    def __init__(self, tier_name: str, gpu_id: str, vram_gb: int, cost_per_hr: float):
        self.tier_name = tier_name
        self.gpu_id = gpu_id
        self.vram_gb = vram_gb
        self.cost_per_hr = cost_per_hr

    @classmethod
    def from_name(cls, name: str) -> "GPUTier":
        for tier in cls:
            if tier.tier_name == name:
                return tier
        raise ValueError(f"Unknown GPU tier: {name}")


class RunPodClient:
    """Thin wrapper around RunPod's REST API."""

    def __init__(self, api_key: str):
        self.api_key = api_key
        self.session = requests.Session()
        self.session.headers.update({"Authorization": f"Bearer {api_key}"})

    def list_templates(self) -> list[dict]:
        """Fetch available pod templates."""
        resp = requests.get(
            f"{RUNPOD_API_BASE}/templates",
            headers={"Authorization": f"Bearer {self.api_key}"},
        )
        resp.raise_for_status()
        return resp.json()

    def create_pod(
        self,
        tier: GPUTier,
        template_id: str,
        ssh_public_key: str,
        name: str = "wee-gpu",
        volume_size_gb: int = 50,
    ) -> dict:
        """Create a new GPU pod."""
        payload = {
            "name": name,
            "imageName": "",  # Will use template
            "gpuTypeId": tier.gpu_id,
            "templateId": template_id,
            "volumeInGb": volume_size_gb,
            "containerDiskInGb": 20,
            "env": {
                "PUBLIC_KEY": ssh_public_key,
            },
        }
        resp = requests.post(
            f"{RUNPOD_API_BASE}/pods",
            json=payload,
            headers={"Authorization": f"Bearer {self.api_key}"},
        )
        resp.raise_for_status()
        return resp.json()

    def get_pod(self, pod_id: str) -> dict:
        """Get pod status and details including cost."""
        resp = requests.get(
            f"{RUNPOD_API_BASE}/pods/{pod_id}",
            headers={"Authorization": f"Bearer {self.api_key}"},
        )
        resp.raise_for_status()
        return resp.json()

    def stop_pod(self, pod_id: str) -> dict:
        """Stop a pod (keeps volume, stops billing)."""
        resp = requests.post(
            f"{RUNPOD_API_BASE}/pods/{pod_id}/stop",
            headers={"Authorization": f"Bearer {self.api_key}"},
        )
        resp.raise_for_status()
        return resp.json()

    def resume_pod(self, pod_id: str) -> dict:
        """Resume a stopped pod."""
        resp = requests.post(
            f"{RUNPOD_API_BASE}/pods/{pod_id}/resume",
            headers={"Authorization": f"Bearer {self.api_key}"},
        )
        resp.raise_for_status()
        return resp.json()

    def terminate_pod(self, pod_id: str) -> None:
        """Terminate and delete a pod entirely."""
        resp = requests.delete(
            f"{RUNPOD_API_BASE}/pods/{pod_id}",
            headers={"Authorization": f"Bearer {self.api_key}"},
        )
        resp.raise_for_status()
```

**Note:** The RunPod API may use GraphQL for some endpoints. Verify actual API shape against https://docs.runpod.io/reference and adjust accordingly. The structure above follows their REST-style docs but the real API might need GraphQL mutations for create/stop/resume.

**Step 4: Run tests to verify they pass**

```bash
cd /path/to/wee/wee.cat/backend
python manage.py test gpu.tests.test_runpod_client -v 2
```

Expected: All 7 tests pass.

**Step 5: Commit**

```bash
git add wee.cat/backend/gpu/runpod_client.py wee.cat/backend/gpu/tests/
git commit -m "feat: add RunPod API client with GPU tier mapping"
```

---

### Task 3: GPU Connector API (Connect/Disconnect RunPod Key)

**Files:**
- Create: `wee.cat/backend/gpu/serializers.py`
- Create: `wee.cat/backend/gpu/views.py`
- Create: `wee.cat/backend/gpu/urls.py`
- Create: `wee.cat/backend/gpu/tests/test_views.py`

**Step 1: Write tests for connector endpoints**

Create `wee.cat/backend/gpu/tests/test_views.py`:

```python
from django.test import TestCase, RequestFactory
from django.contrib.auth import get_user_model
from gpu.models import GPUConnector

User = get_user_model()


class GPUConnectorViewsTest(TestCase):
    def setUp(self):
        self.user = User.objects.create_user(
            username="testuser", email="test@test.com", password="testpass123"
        )
        self.client.login(username="testuser", password="testpass123")

    def test_connect_runpod(self):
        resp = self.client.post(
            "/api/gpu/connectors/connect/",
            {"provider": "runpod", "api_key": "rp_test_key_123"},
            content_type="application/json",
        )
        self.assertEqual(resp.status_code, 201)
        self.assertTrue(GPUConnector.objects.filter(user=self.user, provider="runpod").exists())

    def test_connect_runpod_replaces_existing(self):
        # Create first connector
        self.client.post(
            "/api/gpu/connectors/connect/",
            {"provider": "runpod", "api_key": "old_key"},
            content_type="application/json",
        )
        # Replace with new key
        resp = self.client.post(
            "/api/gpu/connectors/connect/",
            {"provider": "runpod", "api_key": "new_key"},
            content_type="application/json",
        )
        self.assertEqual(resp.status_code, 201)
        self.assertEqual(GPUConnector.objects.filter(user=self.user).count(), 1)

    def test_disconnect_runpod(self):
        self.client.post(
            "/api/gpu/connectors/connect/",
            {"provider": "runpod", "api_key": "rp_key"},
            content_type="application/json",
        )
        connector = GPUConnector.objects.get(user=self.user)
        resp = self.client.delete(f"/api/gpu/connectors/{connector.id}/disconnect/")
        self.assertEqual(resp.status_code, 204)
        self.assertFalse(GPUConnector.objects.filter(user=self.user).exists())

    def test_list_connectors(self):
        self.client.post(
            "/api/gpu/connectors/connect/",
            {"provider": "runpod", "api_key": "rp_key"},
            content_type="application/json",
        )
        resp = self.client.get("/api/gpu/connectors/")
        self.assertEqual(resp.status_code, 200)
        data = resp.json()
        self.assertEqual(len(data), 1)
        self.assertEqual(data[0]["provider"], "runpod")
        # API key should NOT be in the response
        self.assertNotIn("api_key", data[0])
        self.assertNotIn("encrypted_api_key", data[0])

    def test_unauthenticated_rejected(self):
        self.client.logout()
        resp = self.client.get("/api/gpu/connectors/")
        self.assertEqual(resp.status_code, 403)
```

**Step 2: Run tests to verify they fail**

```bash
python manage.py test gpu.tests.test_views -v 2
```

**Step 3: Implement serializers**

Create `wee.cat/backend/gpu/serializers.py`:

```python
from rest_framework import serializers


class ConnectGPUSerializer(serializers.Serializer):
    provider = serializers.ChoiceField(choices=["runpod"], default="runpod")
    api_key = serializers.CharField(min_length=1, max_length=255)


class GPUConnectorSerializer(serializers.Serializer):
    id = serializers.IntegerField(read_only=True)
    provider = serializers.CharField(read_only=True)
    is_active = serializers.BooleanField(read_only=True)
    created_at = serializers.DateTimeField(read_only=True)


class GPUSessionSerializer(serializers.Serializer):
    id = serializers.IntegerField(read_only=True)
    pod_id = serializers.CharField(read_only=True)
    template_name = serializers.CharField(read_only=True)
    tier = serializers.CharField(read_only=True)
    gpu_type = serializers.CharField(read_only=True)
    status = serializers.CharField(read_only=True)
    ssh_host = serializers.CharField(read_only=True)
    ssh_port = serializers.IntegerField(read_only=True)
    started_at = serializers.DateTimeField(read_only=True)
    stopped_at = serializers.DateTimeField(read_only=True)
    created_at = serializers.DateTimeField(read_only=True)


class LaunchGPUSerializer(serializers.Serializer):
    tier = serializers.ChoiceField(choices=["starter", "pro", "beast"])
    template_id = serializers.CharField(max_length=100)
    ssh_public_key = serializers.CharField()
    sandbox_id = serializers.IntegerField()
```

**Step 4: Implement views**

Create `wee.cat/backend/gpu/views.py`:

```python
import logging
import threading

from django.utils import timezone
from rest_framework import permissions, status
from rest_framework.decorators import api_view, authentication_classes, permission_classes
from rest_framework.response import Response

from sandboxes.views import CsrfExemptSessionAuth

from .models import GPUConnector, GPUSession
from .runpod_client import GPUTier, RunPodClient
from .serializers import (
    ConnectGPUSerializer,
    GPUConnectorSerializer,
    GPUSessionSerializer,
    LaunchGPUSerializer,
)

logger = logging.getLogger(__name__)


# --- Connector endpoints ---


@api_view(["GET"])
@permission_classes([permissions.IsAuthenticated])
def list_connectors(request):
    connectors = GPUConnector.objects.filter(user=request.user)
    serializer = GPUConnectorSerializer(connectors, many=True)
    return Response(serializer.data)


@api_view(["POST"])
@authentication_classes([CsrfExemptSessionAuth])
@permission_classes([permissions.IsAuthenticated])
def connect_gpu(request):
    serializer = ConnectGPUSerializer(data=request.data)
    serializer.is_valid(raise_exception=True)

    provider = serializer.validated_data["provider"]
    api_key = serializer.validated_data["api_key"]

    connector, _created = GPUConnector.objects.update_or_create(
        user=request.user,
        provider=provider,
        defaults={"is_active": True},
    )
    connector.set_api_key(api_key)
    connector.save()

    return Response(
        GPUConnectorSerializer(connector).data,
        status=status.HTTP_201_CREATED,
    )


@api_view(["DELETE"])
@authentication_classes([CsrfExemptSessionAuth])
@permission_classes([permissions.IsAuthenticated])
def disconnect_gpu(request, connector_id):
    try:
        connector = GPUConnector.objects.get(id=connector_id, user=request.user)
    except GPUConnector.DoesNotExist:
        return Response(status=status.HTTP_404_NOT_FOUND)

    connector.delete()
    return Response(status=status.HTTP_204_NO_CONTENT)


# --- GPU session endpoints ---


@api_view(["GET"])
@permission_classes([permissions.IsAuthenticated])
def list_templates(request):
    """List available RunPod templates. Proxied through wee.cat."""
    connector = GPUConnector.objects.filter(user=request.user, provider="runpod", is_active=True).first()
    if not connector:
        return Response(
            {"error": "No RunPod connector configured. Connect your API key first."},
            status=status.HTTP_400_BAD_REQUEST,
        )
    try:
        client = RunPodClient(api_key=connector.get_api_key())
        templates = client.list_templates()
        return Response(templates)
    except Exception as e:
        logger.exception("Failed to fetch RunPod templates")
        return Response({"error": str(e)}, status=status.HTTP_502_BAD_GATEWAY)


@api_view(["POST"])
@authentication_classes([CsrfExemptSessionAuth])
@permission_classes([permissions.IsAuthenticated])
def launch_gpu(request):
    """Launch a GPU pod via RunPod."""
    serializer = LaunchGPUSerializer(data=request.data)
    serializer.is_valid(raise_exception=True)

    sandbox_id = serializer.validated_data["sandbox_id"]
    tier_name = serializer.validated_data["tier"]
    template_id = serializer.validated_data["template_id"]
    ssh_public_key = serializer.validated_data["ssh_public_key"]

    # Validate sandbox ownership
    from sandboxes.models import Sandbox
    try:
        sandbox = Sandbox.objects.get(id=sandbox_id, owner=request.user)
    except Sandbox.DoesNotExist:
        return Response({"error": "Sandbox not found"}, status=status.HTTP_404_NOT_FOUND)

    # Get connector
    connector = GPUConnector.objects.filter(user=request.user, provider="runpod", is_active=True).first()
    if not connector:
        return Response({"error": "No RunPod connector configured"}, status=status.HTTP_400_BAD_REQUEST)

    # Check for existing running session
    existing = GPUSession.objects.filter(sandbox=sandbox, status__in=["creating", "running"]).first()
    if existing:
        return Response(
            {"error": "Sandbox already has an active GPU session", "session_id": existing.id},
            status=status.HTTP_409_CONFLICT,
        )

    tier = GPUTier.from_name(tier_name)

    # Create session record
    session = GPUSession.objects.create(
        sandbox=sandbox,
        connector=connector,
        template_id=template_id,
        template_name="",  # Will be updated async
        tier=tier_name,
        gpu_type=tier.gpu_id,
        status="creating",
    )

    # Launch async
    thread = threading.Thread(
        target=_launch_pod_async,
        args=(session.id, connector.get_api_key(), tier, template_id, ssh_public_key),
        daemon=True,
    )
    thread.start()

    return Response(GPUSessionSerializer(session).data, status=status.HTTP_201_CREATED)


def _launch_pod_async(session_id: int, api_key: str, tier: GPUTier, template_id: str, ssh_public_key: str):
    """Background thread to create RunPod pod."""
    try:
        session = GPUSession.objects.get(id=session_id)
        client = RunPodClient(api_key=api_key)

        result = client.create_pod(
            tier=tier,
            template_id=template_id,
            ssh_public_key=ssh_public_key,
            name=f"wee-{session.sandbox.subdomain}",
        )

        session.pod_id = result.get("id", "")
        session.status = "running"
        session.started_at = timezone.now()

        # Extract SSH connection info from pod details
        pod_info = client.get_pod(session.pod_id)
        runtime = pod_info.get("runtime", {})
        ports = runtime.get("ports", [])
        for port in ports:
            if port.get("privatePort") == 22:
                session.ssh_host = port.get("ip", "")
                session.ssh_port = port.get("publicPort", 22)
                break

        session.save()
        logger.info(f"GPU pod launched: {session.pod_id} for sandbox {session.sandbox.subdomain}")

    except Exception as e:
        logger.exception(f"Failed to launch GPU pod for session {session_id}")
        try:
            session = GPUSession.objects.get(id=session_id)
            session.status = "error"
            session.error_message = str(e)
            session.save()
        except Exception:
            pass


@api_view(["GET"])
@permission_classes([permissions.IsAuthenticated])
def gpu_status(request, session_id):
    """Get GPU session status with live cost from RunPod."""
    try:
        session = GPUSession.objects.get(id=session_id, sandbox__owner=request.user)
    except GPUSession.DoesNotExist:
        return Response(status=status.HTTP_404_NOT_FOUND)

    data = GPUSessionSerializer(session).data

    # Enrich with live RunPod data if running
    if session.status == "running" and session.pod_id:
        try:
            client = RunPodClient(api_key=session.connector.get_api_key())
            pod_info = client.get_pod(session.pod_id)
            runtime = pod_info.get("runtime", {})
            data["live"] = {
                "cost_per_hr": pod_info.get("costPerHr"),
                "uptime_seconds": runtime.get("uptimeInSeconds", 0),
                "total_cost": round(
                    (runtime.get("uptimeInSeconds", 0) / 3600) * pod_info.get("costPerHr", 0),
                    2,
                ),
            }
        except Exception as e:
            data["live"] = {"error": str(e)}

    return Response(data)


@api_view(["POST"])
@authentication_classes([CsrfExemptSessionAuth])
@permission_classes([permissions.IsAuthenticated])
def stop_gpu(request, session_id):
    """Stop a GPU pod (keeps volume, stops billing)."""
    try:
        session = GPUSession.objects.get(id=session_id, sandbox__owner=request.user)
    except GPUSession.DoesNotExist:
        return Response(status=status.HTTP_404_NOT_FOUND)

    if session.status != "running":
        return Response({"error": "Session is not running"}, status=status.HTTP_400_BAD_REQUEST)

    try:
        client = RunPodClient(api_key=session.connector.get_api_key())
        client.stop_pod(session.pod_id)
        session.status = "stopped"
        session.stopped_at = timezone.now()
        session.save()
        return Response(GPUSessionSerializer(session).data)
    except Exception as e:
        return Response({"error": str(e)}, status=status.HTTP_502_BAD_GATEWAY)


@api_view(["POST"])
@authentication_classes([CsrfExemptSessionAuth])
@permission_classes([permissions.IsAuthenticated])
def resume_gpu(request, session_id):
    """Resume a stopped GPU pod."""
    try:
        session = GPUSession.objects.get(id=session_id, sandbox__owner=request.user)
    except GPUSession.DoesNotExist:
        return Response(status=status.HTTP_404_NOT_FOUND)

    if session.status != "stopped":
        return Response({"error": "Session is not stopped"}, status=status.HTTP_400_BAD_REQUEST)

    try:
        client = RunPodClient(api_key=session.connector.get_api_key())
        client.resume_pod(session.pod_id)
        session.status = "running"
        session.stopped_at = None
        session.save()
        return Response(GPUSessionSerializer(session).data)
    except Exception as e:
        return Response({"error": str(e)}, status=status.HTTP_502_BAD_GATEWAY)


@api_view(["DELETE"])
@authentication_classes([CsrfExemptSessionAuth])
@permission_classes([permissions.IsAuthenticated])
def terminate_gpu(request, session_id):
    """Terminate and delete a GPU pod entirely."""
    try:
        session = GPUSession.objects.get(id=session_id, sandbox__owner=request.user)
    except GPUSession.DoesNotExist:
        return Response(status=status.HTTP_404_NOT_FOUND)

    try:
        if session.pod_id:
            client = RunPodClient(api_key=session.connector.get_api_key())
            client.terminate_pod(session.pod_id)
        session.status = "terminated"
        session.stopped_at = timezone.now()
        session.save()
        return Response(status=status.HTTP_204_NO_CONTENT)
    except Exception as e:
        return Response({"error": str(e)}, status=status.HTTP_502_BAD_GATEWAY)


@api_view(["GET"])
@permission_classes([permissions.IsAuthenticated])
def list_sessions(request):
    """List GPU sessions for the current user's sandboxes."""
    sessions = GPUSession.objects.filter(sandbox__owner=request.user)
    sandbox_id = request.query_params.get("sandbox_id")
    if sandbox_id:
        sessions = sessions.filter(sandbox_id=sandbox_id)
    serializer = GPUSessionSerializer(sessions, many=True)
    return Response(serializer.data)
```

**Step 5: Create URL routing**

Create `wee.cat/backend/gpu/urls.py`:

```python
from django.urls import path
from . import views

urlpatterns = [
    # Connector management
    path("connectors/", views.list_connectors, name="gpu-connector-list"),
    path("connectors/connect/", views.connect_gpu, name="gpu-connector-connect"),
    path("connectors/<int:connector_id>/disconnect/", views.disconnect_gpu, name="gpu-connector-disconnect"),
    # Templates
    path("templates/", views.list_templates, name="gpu-templates"),
    # Session management
    path("sessions/", views.list_sessions, name="gpu-session-list"),
    path("launch/", views.launch_gpu, name="gpu-launch"),
    path("sessions/<int:session_id>/status/", views.gpu_status, name="gpu-status"),
    path("sessions/<int:session_id>/stop/", views.stop_gpu, name="gpu-stop"),
    path("sessions/<int:session_id>/resume/", views.resume_gpu, name="gpu-resume"),
    path("sessions/<int:session_id>/terminate/", views.terminate_gpu, name="gpu-terminate"),
]
```

**Step 6: Run tests**

```bash
python manage.py test gpu.tests.test_views -v 2
```

Expected: All tests pass.

**Step 7: Commit**

```bash
git add wee.cat/backend/gpu/
git commit -m "feat: add GPU connector and session API endpoints"
```

---

### Task 4: Sandbox-to-wee.cat Auth for GPU Requests

**Files:**
- Create: `wee.cat/backend/gpu/sandbox_auth.py`
- Modify: `wee.cat/backend/gpu/views.py` (add sandbox auth to GPU endpoints)
- Modify: `wee.cat/backend/sandboxes/provisioner.py` (inject GPU auth token)

The sandbox needs to authenticate with wee.cat to make GPU API calls. Since the sandbox can't use session auth, we need a sandbox-specific token.

**Step 1: Create sandbox auth mechanism**

Create `wee.cat/backend/gpu/sandbox_auth.py`:

```python
"""Sandbox authentication for GPU API calls.

Each sandbox gets a signed token at provisioning time, injected as
WEE_CONTROL_PLANE_TOKEN env var. The token contains the sandbox ID
and user ID, signed with Django's SECRET_KEY.
"""

import json
import time

from django.conf import settings
from django.core.signing import Signer, BadSignature
from rest_framework.authentication import BaseAuthentication
from rest_framework.exceptions import AuthenticationFailed


signer = Signer()


def create_sandbox_token(sandbox_id: int, user_id: int) -> str:
    """Create a signed token for a sandbox to authenticate with wee.cat."""
    payload = json.dumps({"sandbox_id": sandbox_id, "user_id": user_id, "ts": int(time.time())})
    return signer.sign(payload)


def verify_sandbox_token(token: str) -> dict:
    """Verify and decode a sandbox token. Returns {"sandbox_id": int, "user_id": int}."""
    try:
        payload = signer.unsign(token)
        return json.loads(payload)
    except (BadSignature, json.JSONDecodeError) as e:
        raise AuthenticationFailed(f"Invalid sandbox token: {e}")


class SandboxTokenAuthentication(BaseAuthentication):
    """DRF authentication class for sandbox tokens.

    Expects header: Authorization: SandboxToken <token>
    """

    def authenticate(self, request):
        auth_header = request.META.get("HTTP_AUTHORIZATION", "")
        if not auth_header.startswith("SandboxToken "):
            return None

        token = auth_header[len("SandboxToken "):]
        payload = verify_sandbox_token(token)

        from django.contrib.auth import get_user_model
        User = get_user_model()
        try:
            user = User.objects.get(id=payload["user_id"])
        except User.DoesNotExist:
            raise AuthenticationFailed("User not found")

        # Attach sandbox_id to request for downstream use
        request.sandbox_id = payload["sandbox_id"]
        return (user, None)
```

**Step 2: Update provisioner to inject token**

Add to `wee.cat/backend/sandboxes/provisioner.py`, in the `provision_sandbox` function where env vars are built, add:

```python
from gpu.sandbox_auth import create_sandbox_token

# In _build_env_args or equivalent:
token = create_sandbox_token(sandbox.id, sandbox.owner_id)
env_args.extend(["-e", f"WEE_CONTROL_PLANE_TOKEN={token}"])
env_args.extend(["-e", f"WEE_CONTROL_PLANE_URL=https://wee.cat"])
env_args.extend(["-e", f"WEE_SANDBOX_ID={sandbox.id}"])
```

**Step 3: Add SandboxTokenAuthentication to GPU views**

Update the `@authentication_classes` decorators on GPU session endpoints (launch, stop, resume, terminate, status, templates) to also accept `SandboxTokenAuthentication`:

```python
from .sandbox_auth import SandboxTokenAuthentication

@api_view(["POST"])
@authentication_classes([CsrfExemptSessionAuth, SandboxTokenAuthentication])
@permission_classes([permissions.IsAuthenticated])
def launch_gpu(request):
    ...
```

**Step 4: Commit**

```bash
git add wee.cat/backend/gpu/sandbox_auth.py wee.cat/backend/gpu/views.py wee.cat/backend/sandboxes/provisioner.py
git commit -m "feat: add sandbox token auth for GPU API proxy"
```

---

## Part 2: Wee Go Binary (Sandbox Side)

### Task 5: GPU Package — Core Types and API Client

**Files:**
- Create: `internal/gpu/gpu.go`
- Create: `internal/gpu/gpu_test.go`

**Step 1: Write failing test**

Create `internal/gpu/gpu_test.go`:

```go
package gpu

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewManager(t *testing.T) {
	m := NewManager("https://wee.cat", "sandbox-1", "token-abc", "/tmp/ssh-key")
	if m.weeCatURL != "https://wee.cat" {
		t.Errorf("expected wee.cat URL, got %s", m.weeCatURL)
	}
	if m.sandboxID != "sandbox-1" {
		t.Errorf("expected sandbox-1, got %s", m.sandboxID)
	}
}

func TestManagerFromEnv(t *testing.T) {
	t.Setenv("WEE_CONTROL_PLANE_URL", "https://test.wee.cat")
	t.Setenv("WEE_SANDBOX_ID", "42")
	t.Setenv("WEE_CONTROL_PLANE_TOKEN", "tok-123")

	m, err := NewManagerFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.weeCatURL != "https://test.wee.cat" {
		t.Errorf("expected https://test.wee.cat, got %s", m.weeCatURL)
	}
	if m.sandboxID != "42" {
		t.Errorf("expected 42, got %s", m.sandboxID)
	}
}

func TestManagerFromEnvMissing(t *testing.T) {
	t.Setenv("WEE_CONTROL_PLANE_URL", "")
	t.Setenv("WEE_SANDBOX_ID", "")
	t.Setenv("WEE_CONTROL_PLANE_TOKEN", "")

	_, err := NewManagerFromEnv()
	if err == nil {
		t.Fatal("expected error for missing env vars")
	}
}

func TestLaunch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/gpu/launch/" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "SandboxToken test-token" {
			t.Errorf("unexpected auth header: %s", r.Header.Get("Authorization"))
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":        1,
			"pod_id":    "pod-abc",
			"status":    "creating",
			"tier":      "pro",
			"gpu_type":  "NVIDIA A100 80GB PCIe",
			"ssh_host":  "",
			"ssh_port":  nil,
			"created_at": "2026-04-07T00:00:00Z",
		})
	}))
	defer server.Close()

	m := NewManager(server.URL, "42", "test-token", "/tmp/test-key")
	session, err := m.Launch("pro", "tpl-123", "ssh-ed25519 AAAA...")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session.Status != "creating" {
		t.Errorf("expected creating, got %s", session.Status)
	}
	if session.Tier != "pro" {
		t.Errorf("expected pro, got %s", session.Tier)
	}
}

func TestStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":     1,
			"status": "running",
			"tier":   "pro",
			"live": map[string]interface{}{
				"cost_per_hr":     1.64,
				"uptime_seconds":  3600,
				"total_cost":      1.64,
			},
		})
	}))
	defer server.Close()

	m := NewManager(server.URL, "42", "test-token", "/tmp/test-key")
	status, err := m.Status()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Status != "running" {
		t.Errorf("expected running, got %s", status.Status)
	}
	if status.Live.TotalCost != 1.64 {
		t.Errorf("expected cost 1.64, got %f", status.Live.TotalCost)
	}
}
```

**Step 2: Run test to verify it fails**

```bash
cd /path/to/wee
go test ./internal/gpu/ -v
```

Expected: Package doesn't exist yet.

**Step 3: Implement GPU manager**

Create `internal/gpu/gpu.go`:

```go
package gpu

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// GPUSession represents a GPU session from the wee.cat API.
type GPUSession struct {
	ID           int        `json:"id"`
	PodID        string     `json:"pod_id"`
	TemplateName string     `json:"template_name"`
	Tier         string     `json:"tier"`
	GPUType      string     `json:"gpu_type"`
	Status       string     `json:"status"`
	SSHHost      string     `json:"ssh_host"`
	SSHPort      int        `json:"ssh_port"`
	ErrorMessage string     `json:"error_message"`
	StartedAt    *time.Time `json:"started_at"`
	StoppedAt    *time.Time `json:"stopped_at"`
	CreatedAt    time.Time  `json:"created_at"`
	Live         *LiveData  `json:"live,omitempty"`
}

// LiveData contains real-time data from RunPod.
type LiveData struct {
	CostPerHr     float64 `json:"cost_per_hr"`
	UptimeSeconds int     `json:"uptime_seconds"`
	TotalCost     float64 `json:"total_cost"`
}

// Template represents a RunPod template.
type Template struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	ImageName string `json:"imageName"`
}

// Manager handles GPU operations by talking to the wee.cat control plane.
type Manager struct {
	weeCatURL  string
	sandboxID  string
	authToken  string
	sshKeyPath string
	client     *http.Client
}

// NewManager creates a GPU manager with explicit configuration.
func NewManager(weeCatURL, sandboxID, authToken, sshKeyPath string) *Manager {
	return &Manager{
		weeCatURL:  weeCatURL,
		sandboxID:  sandboxID,
		authToken:  authToken,
		sshKeyPath: sshKeyPath,
		client:     &http.Client{Timeout: 30 * time.Second},
	}
}

// NewManagerFromEnv creates a GPU manager from environment variables.
func NewManagerFromEnv() (*Manager, error) {
	url := os.Getenv("WEE_CONTROL_PLANE_URL")
	sandboxID := os.Getenv("WEE_SANDBOX_ID")
	token := os.Getenv("WEE_CONTROL_PLANE_TOKEN")

	if url == "" || sandboxID == "" || token == "" {
		return nil, fmt.Errorf("missing required env vars: WEE_CONTROL_PLANE_URL, WEE_SANDBOX_ID, WEE_CONTROL_PLANE_TOKEN")
	}

	sshKeyPath := filepath.Join(os.Getenv("HOME"), ".ssh", "wee-gpu-key")

	return NewManager(url, sandboxID, token, sshKeyPath), nil
}

// Launch creates a new GPU pod.
func (m *Manager) Launch(tier, templateID, sshPublicKey string) (*GPUSession, error) {
	payload := map[string]interface{}{
		"tier":           tier,
		"template_id":    templateID,
		"ssh_public_key": sshPublicKey,
		"sandbox_id":     m.sandboxID,
	}
	body, _ := json.Marshal(payload)

	resp, err := m.doRequest("POST", "/api/gpu/launch/", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("launch request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		errBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("launch failed (status %d): %s", resp.StatusCode, string(errBody))
	}

	var session GPUSession
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &session, nil
}

// Status gets the current GPU session status with live cost data.
func (m *Manager) Status() (*GPUSession, error) {
	// First get the active session
	sessions, err := m.ListSessions()
	if err != nil {
		return nil, err
	}

	// Find the active one
	for _, s := range sessions {
		if s.Status == "running" || s.Status == "creating" {
			return m.getSessionStatus(s.ID)
		}
	}

	return nil, fmt.Errorf("no active GPU session")
}

func (m *Manager) getSessionStatus(sessionID int) (*GPUSession, error) {
	resp, err := m.doRequest("GET", fmt.Sprintf("/api/gpu/sessions/%d/status/", sessionID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var session GPUSession
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return nil, fmt.Errorf("failed to decode status: %w", err)
	}
	return &session, nil
}

// Stop stops the active GPU pod.
func (m *Manager) Stop() error {
	session, err := m.Status()
	if err != nil {
		return err
	}
	resp, err := m.doRequest("POST", fmt.Sprintf("/api/gpu/sessions/%d/stop/", session.ID), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		errBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("stop failed: %s", string(errBody))
	}
	return nil
}

// Resume resumes a stopped GPU pod.
func (m *Manager) Resume() error {
	sessions, err := m.ListSessions()
	if err != nil {
		return err
	}
	for _, s := range sessions {
		if s.Status == "stopped" {
			resp, err := m.doRequest("POST", fmt.Sprintf("/api/gpu/sessions/%d/resume/", s.ID), nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				errBody, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("resume failed: %s", string(errBody))
			}
			return nil
		}
	}
	return fmt.Errorf("no stopped GPU session to resume")
}

// Kill terminates the GPU pod entirely.
func (m *Manager) Kill() error {
	session, err := m.Status()
	if err != nil {
		return err
	}
	resp, err := m.doRequest("DELETE", fmt.Sprintf("/api/gpu/sessions/%d/terminate/", session.ID), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 204 {
		errBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("terminate failed: %s", string(errBody))
	}
	return nil
}

// ListTemplates fetches available RunPod templates.
func (m *Manager) ListTemplates() ([]Template, error) {
	resp, err := m.doRequest("GET", "/api/gpu/templates/", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var templates []Template
	if err := json.NewDecoder(resp.Body).Decode(&templates); err != nil {
		return nil, fmt.Errorf("failed to decode templates: %w", err)
	}
	return templates, nil
}

// ListSessions fetches GPU sessions for this sandbox.
func (m *Manager) ListSessions() ([]GPUSession, error) {
	resp, err := m.doRequest("GET", fmt.Sprintf("/api/gpu/sessions/?sandbox_id=%s", m.sandboxID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var sessions []GPUSession
	if err := json.NewDecoder(resp.Body).Decode(&sessions); err != nil {
		return nil, fmt.Errorf("failed to decode sessions: %w", err)
	}
	return sessions, nil
}

// ActiveSession returns the SSH connection details for the active session.
func (m *Manager) ActiveSession() (*GPUSession, error) {
	session, err := m.Status()
	if err != nil {
		return nil, err
	}
	if session.SSHHost == "" {
		return nil, fmt.Errorf("GPU session is %s but SSH not ready yet", session.Status)
	}
	return session, nil
}

func (m *Manager) doRequest(method, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, m.weeCatURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "SandboxToken "+m.authToken)
	req.Header.Set("Content-Type", "application/json")

	return m.client.Do(req)
}
```

**Step 4: Run tests**

```bash
go test ./internal/gpu/ -v
```

Expected: All tests pass.

**Step 5: Commit**

```bash
git add internal/gpu/
git commit -m "feat: add GPU manager package with wee.cat API client"
```

---

### Task 6: SSH and Sync Helpers

**Files:**
- Create: `internal/gpu/ssh.go`
- Create: `internal/gpu/sync.go`
- Create: `internal/gpu/ssh_test.go`

**Step 1: Write failing test for SSH key generation**

Create `internal/gpu/ssh_test.go`:

```go
package gpu

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureSSHKey(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test-gpu-key")

	pubKey, err := EnsureSSHKey(keyPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Private key should exist
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Error("private key file not created")
	}

	// Public key should exist
	if _, err := os.Stat(keyPath + ".pub"); os.IsNotExist(err) {
		t.Error("public key file not created")
	}

	// Public key string should be non-empty
	if pubKey == "" {
		t.Error("public key string is empty")
	}

	// Calling again should return the same key (idempotent)
	pubKey2, err := EnsureSSHKey(keyPath)
	if err != nil {
		t.Fatalf("unexpected error on second call: %v", err)
	}
	if pubKey != pubKey2 {
		t.Error("second call returned different public key")
	}
}

func TestBuildSSHArgs(t *testing.T) {
	args := buildSSHArgs("/tmp/key", "1.2.3.4", 22222)
	expected := []string{
		"-i", "/tmp/key",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-p", "22222",
		"root@1.2.3.4",
	}
	if len(args) != len(expected) {
		t.Fatalf("expected %d args, got %d", len(expected), len(args))
	}
	for i, arg := range args {
		if arg != expected[i] {
			t.Errorf("arg[%d]: expected %s, got %s", i, expected[i], arg)
		}
	}
}
```

**Step 2: Run tests to verify failure**

```bash
go test ./internal/gpu/ -v -run TestEnsureSSHKey
```

**Step 3: Implement SSH helpers**

Create `internal/gpu/ssh.go`:

```go
package gpu

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// EnsureSSHKey generates an ed25519 SSH keypair if it doesn't exist.
// Returns the public key string.
func EnsureSSHKey(keyPath string) (string, error) {
	pubKeyPath := keyPath + ".pub"

	// Check if key already exists
	if _, err := os.Stat(keyPath); err == nil {
		pubKey, err := os.ReadFile(pubKeyPath)
		if err != nil {
			return "", fmt.Errorf("private key exists but can't read public key: %w", err)
		}
		return strings.TrimSpace(string(pubKey)), nil
	}

	// Generate new keypair
	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-f", keyPath, "-N", "", "-q")
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ssh-keygen failed: %w", err)
	}

	pubKey, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read generated public key: %w", err)
	}

	return strings.TrimSpace(string(pubKey)), nil
}

// buildSSHArgs constructs the common SSH arguments.
func buildSSHArgs(keyPath, host string, port int) []string {
	return []string{
		"-i", keyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-p", strconv.Itoa(port),
		"root@" + host,
	}
}

// Shell opens an interactive SSH session to the GPU pod.
func (m *Manager) Shell() error {
	session, err := m.ActiveSession()
	if err != nil {
		return err
	}

	args := buildSSHArgs(m.sshKeyPath, session.SSHHost, session.SSHPort)
	cmd := exec.Command("ssh", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// Run executes a command on the GPU pod via SSH and streams output.
func (m *Manager) Run(command string) error {
	session, err := m.ActiveSession()
	if err != nil {
		return err
	}

	args := buildSSHArgs(m.sshKeyPath, session.SSHHost, session.SSHPort)
	args = append(args, command)

	cmd := exec.Command("ssh", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
```

**Step 4: Implement sync helpers**

Create `internal/gpu/sync.go`:

```go
package gpu

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
)

// Push syncs files from the sandbox to the GPU pod via rsync over SSH.
func (m *Manager) Push(localPath string) error {
	session, err := m.ActiveSession()
	if err != nil {
		return err
	}

	if localPath == "" {
		localPath = "/app/"
	}

	sshCmd := fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -p %d",
		m.sshKeyPath, session.SSHPort)

	remotePath := fmt.Sprintf("root@%s:%s", session.SSHHost, localPath)

	cmd := exec.Command("rsync", "-avz", "--progress",
		"-e", sshCmd,
		localPath, remotePath,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// Pull syncs files from the GPU pod to the sandbox via rsync over SSH.
func (m *Manager) Pull(remotePath string) error {
	session, err := m.ActiveSession()
	if err != nil {
		return err
	}

	if remotePath == "" {
		remotePath = "/app/"
	}

	sshCmd := fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -p %s",
		m.sshKeyPath, strconv.Itoa(session.SSHPort))

	remoteFullPath := fmt.Sprintf("root@%s:%s", session.SSHHost, remotePath)

	cmd := exec.Command("rsync", "-avz", "--progress",
		"-e", sshCmd,
		remoteFullPath, remotePath,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
```

**Step 5: Run tests**

```bash
go test ./internal/gpu/ -v
```

**Step 6: Commit**

```bash
git add internal/gpu/ssh.go internal/gpu/sync.go internal/gpu/ssh_test.go
git commit -m "feat: add SSH and rsync helpers for GPU sidecar"
```

---

### Task 7: GPU CLI Command (Cobra)

**Files:**
- Create: `internal/cmd/gpu.go`

**Step 1: Implement the gpu command tree**

Create `internal/cmd/gpu.go`:

```go
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/pterm/pterm"
	"github.com/schlunsen/wee-editor/internal/gpu"
	"github.com/spf13/cobra"
)

var (
	gpuTier       string
	gpuTemplateID string
	gpuPath       string
)

var gpuCmd = &cobra.Command{
	Use:   "gpu",
	Short: "Manage GPU sidecar for ML training",
	Long:  "Launch, manage, and connect to GPU VMs for machine learning workloads.",
	RunE:  runGPUTUI,
}

var gpuLaunchCmd = &cobra.Command{
	Use:   "launch",
	Short: "Launch a GPU machine",
	RunE:  runGPULaunch,
}

var gpuStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show GPU status with live cost and utilization",
	RunE:  runGPUStatus,
}

var gpuShellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Open SSH shell to the GPU machine",
	RunE:  runGPUShell,
}

var gpuRunCmd = &cobra.Command{
	Use:   "run [command...]",
	Short: "Execute a command on the GPU machine",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runGPURun,
}

var gpuPushCmd = &cobra.Command{
	Use:   "push [path]",
	Short: "Sync files to the GPU machine",
	RunE:  runGPUPush,
}

var gpuPullCmd = &cobra.Command{
	Use:   "pull [path]",
	Short: "Pull files from the GPU machine",
	RunE:  runGPUPull,
}

var gpuStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop GPU machine (keeps volume, stops billing)",
	RunE:  runGPUStop,
}

var gpuResumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "Resume a stopped GPU machine",
	RunE:  runGPUResume,
}

var gpuKillCmd = &cobra.Command{
	Use:   "kill",
	Short: "Terminate GPU machine entirely",
	RunE:  runGPUKill,
}

func init() {
	rootCmd.AddCommand(gpuCmd)

	gpuCmd.AddCommand(gpuLaunchCmd)
	gpuCmd.AddCommand(gpuStatusCmd)
	gpuCmd.AddCommand(gpuShellCmd)
	gpuCmd.AddCommand(gpuRunCmd)
	gpuCmd.AddCommand(gpuPushCmd)
	gpuCmd.AddCommand(gpuPullCmd)
	gpuCmd.AddCommand(gpuStopCmd)
	gpuCmd.AddCommand(gpuResumeCmd)
	gpuCmd.AddCommand(gpuKillCmd)

	gpuLaunchCmd.Flags().StringVar(&gpuTier, "tier", "", "GPU tier: starter, pro, beast")
	gpuLaunchCmd.Flags().StringVar(&gpuTemplateID, "template", "", "RunPod template ID")

	gpuPushCmd.Flags().StringVar(&gpuPath, "path", "/app/", "Local path to sync")
	gpuPullCmd.Flags().StringVar(&gpuPath, "path", "/app/", "Remote path to pull")
}

func getGPUManager() (*gpu.Manager, error) {
	m, err := gpu.NewManagerFromEnv()
	if err != nil {
		return nil, fmt.Errorf("GPU sidecar requires a wee.cat sandbox environment.\n"+
			"Missing env vars: WEE_CONTROL_PLANE_URL, WEE_SANDBOX_ID, WEE_CONTROL_PLANE_TOKEN\n\n"+
			"Are you running inside a wee sandbox?")
	}
	return m, nil
}

func runGPUTUI(cmd *cobra.Command, args []string) error {
	// If --tier and --template are both set, skip TUI
	if gpuTier != "" && gpuTemplateID != "" {
		return runGPULaunch(cmd, args)
	}

	// Launch interactive TUI
	m, err := getGPUManager()
	if err != nil {
		return err
	}
	return gpu.RunTUI(m)
}

func runGPULaunch(cmd *cobra.Command, args []string) error {
	if gpuTier == "" || gpuTemplateID == "" {
		// Fall through to TUI if flags missing
		return runGPUTUI(cmd, args)
	}

	m, err := getGPUManager()
	if err != nil {
		return err
	}

	spinner := ShowSpinner("Generating SSH key...")
	pubKey, err := gpu.EnsureSSHKey(m.SSHKeyPath())
	if err != nil {
		spinner.Fail("Failed to generate SSH key")
		return err
	}
	spinner.Success("SSH key ready")

	spinner = ShowSpinner(fmt.Sprintf("Launching %s GPU with template %s...", gpuTier, gpuTemplateID))
	session, err := m.Launch(gpuTier, gpuTemplateID, pubKey)
	if err != nil {
		spinner.Fail("Launch failed")
		return err
	}
	spinner.Success(fmt.Sprintf("GPU pod created (status: %s)", session.Status))

	pterm.Info.Println("Pod is starting up. Use 'wee gpu status' to check progress.")
	pterm.Info.Println("Once running, use 'wee gpu shell' to connect.")

	return nil
}

func runGPUStatus(cmd *cobra.Command, args []string) error {
	m, err := getGPUManager()
	if err != nil {
		return err
	}

	session, err := m.Status()
	if err != nil {
		pterm.Warning.Println("No active GPU session")
		return nil
	}

	statusIcon := "🟡"
	switch session.Status {
	case "running":
		statusIcon = "🟢"
	case "stopped":
		statusIcon = "🔴"
	case "error":
		statusIcon = "❌"
	}

	content := fmt.Sprintf("%s %s  %s (%s tier)\n", statusIcon, session.Status, session.GPUType, session.Tier)
	content += fmt.Sprintf("Template: %s\n", session.TemplateName)

	if session.Live != nil {
		hours := session.Live.UptimeSeconds / 3600
		mins := (session.Live.UptimeSeconds % 3600) / 60
		content += fmt.Sprintf("Uptime:   %dh %dm\n", hours, mins)
		content += fmt.Sprintf("Cost:     $%.2f ($%.2f/hr)\n", session.Live.TotalCost, session.Live.CostPerHr)
	}

	if session.SSHHost != "" {
		content += fmt.Sprintf("\nSSH:      %s:%d", session.SSHHost, session.SSHPort)
	}

	ShowBox("GPU Status", content)
	return nil
}

func runGPUShell(cmd *cobra.Command, args []string) error {
	m, err := getGPUManager()
	if err != nil {
		return err
	}
	pterm.Info.Println("Connecting to GPU machine...")
	return m.Shell()
}

func runGPURun(cmd *cobra.Command, args []string) error {
	m, err := getGPUManager()
	if err != nil {
		return err
	}
	command := strings.Join(args, " ")
	pterm.Info.Printfln("Running on GPU: %s", command)
	return m.Run(command)
}

func runGPUPush(cmd *cobra.Command, args []string) error {
	m, err := getGPUManager()
	if err != nil {
		return err
	}
	path := gpuPath
	if len(args) > 0 {
		path = args[0]
	}
	pterm.Info.Printfln("Syncing %s to GPU machine...", path)
	return m.Push(path)
}

func runGPUPull(cmd *cobra.Command, args []string) error {
	m, err := getGPUManager()
	if err != nil {
		return err
	}
	path := gpuPath
	if len(args) > 0 {
		path = args[0]
	}
	pterm.Info.Printfln("Pulling %s from GPU machine...", path)
	return m.Pull(path)
}

func runGPUStop(cmd *cobra.Command, args []string) error {
	m, err := getGPUManager()
	if err != nil {
		return err
	}
	spinner := ShowSpinner("Stopping GPU machine...")
	if err := m.Stop(); err != nil {
		spinner.Fail("Failed to stop GPU")
		return err
	}
	spinner.Success("GPU stopped (volume preserved, billing stopped)")
	return nil
}

func runGPUResume(cmd *cobra.Command, args []string) error {
	m, err := getGPUManager()
	if err != nil {
		return err
	}
	spinner := ShowSpinner("Resuming GPU machine...")
	if err := m.Resume(); err != nil {
		spinner.Fail("Failed to resume GPU")
		return err
	}
	spinner.Success("GPU resumed")
	return nil
}

func runGPUKill(cmd *cobra.Command, args []string) error {
	m, err := getGPUManager()
	if err != nil {
		return err
	}

	pterm.Warning.Println("This will permanently terminate the GPU pod and delete its volume.")
	fmt.Print("Are you sure? [y/N] ")
	var confirm string
	fmt.Scanln(&confirm)
	if confirm != "y" && confirm != "Y" {
		pterm.Info.Println("Cancelled")
		return nil
	}

	spinner := ShowSpinner("Terminating GPU machine...")
	if err := m.Kill(); err != nil {
		spinner.Fail("Failed to terminate GPU")
		return err
	}
	spinner.Success("GPU terminated")
	return nil
}
```

Also add a `SSHKeyPath()` accessor to `internal/gpu/gpu.go`:

```go
// SSHKeyPath returns the SSH key path.
func (m *Manager) SSHKeyPath() string {
	return m.sshKeyPath
}
```

**Step 2: Verify it compiles**

```bash
go build ./cmd/wee
```

**Step 3: Commit**

```bash
git add internal/cmd/gpu.go internal/gpu/gpu.go
git commit -m "feat: add wee gpu CLI command tree"
```

---

### Task 8: GPU TUI (Bubbletea)

**Files:**
- Create: `internal/gpu/tui.go`

**Step 1: Implement the TUI**

Create `internal/gpu/tui.go`:

```go
package gpu

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// GPU tier definitions for TUI display.
var gpuTiers = []tierInfo{
	{Name: "starter", Label: "Starter", GPU: "T4 16GB", Cost: "~$0.20/hr", Color: lipgloss.Color("10")},
	{Name: "pro", Label: "Pro", GPU: "A100 80GB", Cost: "~$1.50/hr", Color: lipgloss.Color("11")},
	{Name: "beast", Label: "Beast", GPU: "H100 80GB", Cost: "~$3.50/hr", Color: lipgloss.Color("9")},
}

type tierInfo struct {
	Name  string
	Label string
	GPU   string
	Cost  string
	Color lipgloss.Color
}

type tuiScreen int

const (
	screenSelectTier tuiScreen = iota
	screenSelectTemplate
	screenLaunching
	screenDone
	screenError
)

type tuiModel struct {
	manager       *Manager
	screen        tuiScreen
	tierCursor    int
	tmplCursor    int
	templates     []Template
	selectedTier  string
	selectedTmpl  string
	sshPubKey     string
	err           error
	width, height int
}

// Messages
type templatesLoadedMsg struct{ templates []Template }
type templatesErrorMsg struct{ err error }
type launchDoneMsg struct{ session *GPUSession }
type launchErrorMsg struct{ err error }
type sshKeyReadyMsg struct{ pubKey string }
type sshKeyErrorMsg struct{ err error }

func RunTUI(m *Manager) error {
	model := tuiModel{manager: m, screen: screenSelectTier}
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (m tuiModel) Init() tea.Cmd {
	// Generate SSH key in background
	return func() tea.Msg {
		pubKey, err := EnsureSSHKey(m.manager.sshKeyPath)
		if err != nil {
			return sshKeyErrorMsg{err}
		}
		return sshKeyReadyMsg{pubKey}
	}
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			switch m.screen {
			case screenSelectTier:
				if m.tierCursor > 0 {
					m.tierCursor--
				}
			case screenSelectTemplate:
				if m.tmplCursor > 0 {
					m.tmplCursor--
				}
			}
		case "down", "j":
			switch m.screen {
			case screenSelectTier:
				if m.tierCursor < len(gpuTiers)-1 {
					m.tierCursor++
				}
			case screenSelectTemplate:
				if m.tmplCursor < len(m.templates)-1 {
					m.tmplCursor++
				}
			}
		case "enter":
			switch m.screen {
			case screenSelectTier:
				m.selectedTier = gpuTiers[m.tierCursor].Name
				m.screen = screenSelectTemplate
				// Load templates
				return m, func() tea.Msg {
					templates, err := m.manager.ListTemplates()
					if err != nil {
						return templatesErrorMsg{err}
					}
					return templatesLoadedMsg{templates}
				}
			case screenSelectTemplate:
				if len(m.templates) > 0 {
					m.selectedTmpl = m.templates[m.tmplCursor].ID
					m.screen = screenLaunching
					return m, func() tea.Msg {
						session, err := m.manager.Launch(m.selectedTier, m.selectedTmpl, m.sshPubKey)
						if err != nil {
							return launchErrorMsg{err}
						}
						return launchDoneMsg{session}
					}
				}
			case screenDone, screenError:
				return m, tea.Quit
			}
		case "esc":
			switch m.screen {
			case screenSelectTemplate:
				m.screen = screenSelectTier
			default:
				return m, tea.Quit
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case sshKeyReadyMsg:
		m.sshPubKey = msg.pubKey

	case sshKeyErrorMsg:
		m.err = msg.err
		m.screen = screenError

	case templatesLoadedMsg:
		m.templates = msg.templates

	case templatesErrorMsg:
		m.err = msg.err
		m.screen = screenError

	case launchDoneMsg:
		m.screen = screenDone

	case launchErrorMsg:
		m.err = msg.err
		m.screen = screenError
	}

	return m, nil
}

func (m tuiModel) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99")).MarginBottom(1)
	selectedStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	var b strings.Builder

	switch m.screen {
	case screenSelectTier:
		b.WriteString(titleStyle.Render("Launch GPU Machine"))
		b.WriteString("\n\n")
		b.WriteString("Select GPU tier:\n\n")

		for i, tier := range gpuTiers {
			cursor := "  "
			style := dimStyle
			if i == m.tierCursor {
				cursor = "> "
				style = selectedStyle
			}
			dot := lipgloss.NewStyle().Foreground(tier.Color).Render("●")
			line := fmt.Sprintf("%s %s %-10s %-14s %s", cursor, dot, tier.Label, tier.GPU, tier.Cost)
			b.WriteString(style.Render(line))
			b.WriteString("\n")
		}

		b.WriteString("\n")
		b.WriteString(dimStyle.Render("↑/↓ select  ⏎ next  q quit"))

	case screenSelectTemplate:
		b.WriteString(titleStyle.Render("Launch GPU Machine"))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf("Tier: %s\n\n", gpuTiers[m.tierCursor].Label))
		b.WriteString("Select template:\n\n")

		if len(m.templates) == 0 {
			b.WriteString(dimStyle.Render("Loading templates..."))
		} else {
			for i, tmpl := range m.templates {
				cursor := "  "
				style := dimStyle
				if i == m.tmplCursor {
					cursor = "> "
					style = selectedStyle
				}
				line := fmt.Sprintf("%s %s", cursor, tmpl.Name)
				b.WriteString(style.Render(line))
				b.WriteString("\n")
			}
		}

		b.WriteString("\n")
		b.WriteString(dimStyle.Render("↑/↓ select  ⏎ launch  esc back  q quit"))

	case screenLaunching:
		b.WriteString(titleStyle.Render("Launching GPU..."))
		b.WriteString("\n\n")
		b.WriteString("Creating pod on RunPod. This takes 15-30 seconds...\n")

	case screenDone:
		b.WriteString(titleStyle.Render("GPU Launched!"))
		b.WriteString("\n\n")
		b.WriteString("Your GPU machine is starting up.\n\n")
		b.WriteString("  wee gpu status   — check progress\n")
		b.WriteString("  wee gpu shell    — connect via SSH\n")
		b.WriteString("  wee gpu push     — sync files to GPU\n")
		b.WriteString("  wee gpu stop     — stop billing\n")
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("Press any key to exit"))

	case screenError:
		b.WriteString(titleStyle.Render("Error"))
		b.WriteString("\n\n")
		if m.err != nil {
			b.WriteString(fmt.Sprintf("Error: %v\n", m.err))
		}
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("Press any key to exit"))
	}

	return b.String()
}
```

**Step 2: Verify it compiles**

```bash
go build ./cmd/wee
```

**Step 3: Commit**

```bash
git add internal/gpu/tui.go
git commit -m "feat: add Bubbletea TUI for GPU tier and template selection"
```

---

### Task 9: MCP Tools for GPU

**Files:**
- Create: `internal/gpu/mcp_tools.go`
- Modify: `internal/mcp/server.go` (register GPU tools)

**Step 1: Implement MCP tool handlers**

Create `internal/gpu/mcp_tools.go`:

```go
package gpu

import (
	"encoding/json"
	"fmt"
)

// MCPContentBlock matches the MCP protocol content block type.
type MCPContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// MCPToolResult matches the MCP protocol tool result type.
type MCPToolResult struct {
	Content []MCPContentBlock `json:"content"`
	IsError *bool             `json:"isError,omitempty"`
}

// ToolDefinition describes an MCP tool for the tools/list response.
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// GetToolDefinitions returns MCP tool definitions for GPU operations.
func GetToolDefinitions() []ToolDefinition {
	return []ToolDefinition{
		{
			Name:        "gpu_launch",
			Description: "Launch a GPU machine for ML training. Requires tier (starter/pro/beast) and template_id.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"tier":        map[string]string{"type": "string", "description": "GPU tier: starter, pro, or beast"},
					"template_id": map[string]string{"type": "string", "description": "RunPod template ID"},
				},
				"required": []string{"tier", "template_id"},
			},
		},
		{
			Name:        "gpu_run",
			Description: "Execute a command on the GPU machine via SSH.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"command": map[string]string{"type": "string", "description": "Command to execute"},
				},
				"required": []string{"command"},
			},
		},
		{
			Name:        "gpu_push",
			Description: "Sync files from the sandbox to the GPU machine.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]string{"type": "string", "description": "Local path to sync (default: /app/)"},
				},
			},
		},
		{
			Name:        "gpu_pull",
			Description: "Pull files from the GPU machine to the sandbox.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]string{"type": "string", "description": "Remote path to pull (default: /app/)"},
				},
			},
		},
		{
			Name:        "gpu_status",
			Description: "Get GPU machine status, utilization, and running cost.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "gpu_stop",
			Description: "Stop the GPU machine (preserves volume, stops billing).",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
	}
}

// HandleToolCall dispatches an MCP tool call to the appropriate GPU operation.
func HandleToolCall(manager *Manager, toolName string, args map[string]interface{}) MCPToolResult {
	switch toolName {
	case "gpu_launch":
		return handleLaunch(manager, args)
	case "gpu_run":
		return handleRun(manager, args)
	case "gpu_push":
		return handlePush(manager, args)
	case "gpu_pull":
		return handlePull(manager, args)
	case "gpu_status":
		return handleStatus(manager)
	case "gpu_stop":
		return handleStop(manager)
	default:
		return errorResult(fmt.Sprintf("unknown GPU tool: %s", toolName))
	}
}

func handleLaunch(m *Manager, args map[string]interface{}) MCPToolResult {
	tier, _ := args["tier"].(string)
	templateID, _ := args["template_id"].(string)

	if tier == "" || templateID == "" {
		return errorResult("tier and template_id are required")
	}

	pubKey, err := EnsureSSHKey(m.sshKeyPath)
	if err != nil {
		return errorResult(fmt.Sprintf("SSH key generation failed: %v", err))
	}

	session, err := m.Launch(tier, templateID, pubKey)
	if err != nil {
		return errorResult(fmt.Sprintf("Launch failed: %v", err))
	}

	return successResult(session)
}

func handleRun(m *Manager, args map[string]interface{}) MCPToolResult {
	command, _ := args["command"].(string)
	if command == "" {
		return errorResult("command is required")
	}

	if err := m.Run(command); err != nil {
		return errorResult(fmt.Sprintf("Command failed: %v", err))
	}

	return successResult(map[string]string{"status": "completed", "command": command})
}

func handlePush(m *Manager, args map[string]interface{}) MCPToolResult {
	path, _ := args["path"].(string)
	if err := m.Push(path); err != nil {
		return errorResult(fmt.Sprintf("Push failed: %v", err))
	}
	return successResult(map[string]string{"status": "synced", "path": path})
}

func handlePull(m *Manager, args map[string]interface{}) MCPToolResult {
	path, _ := args["path"].(string)
	if err := m.Pull(path); err != nil {
		return errorResult(fmt.Sprintf("Pull failed: %v", err))
	}
	return successResult(map[string]string{"status": "synced", "path": path})
}

func handleStatus(m *Manager) MCPToolResult {
	session, err := m.Status()
	if err != nil {
		return errorResult(fmt.Sprintf("No active GPU session: %v", err))
	}
	return successResult(session)
}

func handleStop(m *Manager) MCPToolResult {
	if err := m.Stop(); err != nil {
		return errorResult(fmt.Sprintf("Stop failed: %v", err))
	}
	return successResult(map[string]string{"status": "stopped"})
}

func successResult(data interface{}) MCPToolResult {
	jsonData, _ := json.MarshalIndent(data, "", "  ")
	return MCPToolResult{
		Content: []MCPContentBlock{{Type: "text", Text: string(jsonData)}},
	}
}

func errorResult(msg string) MCPToolResult {
	isError := true
	errObj := map[string]string{"error": msg}
	data, _ := json.Marshal(errObj)
	return MCPToolResult{
		Content: []MCPContentBlock{{Type: "text", Text: string(data)}},
		IsError: &isError,
	}
}
```

**Step 2: Register GPU tools in MCP server**

Add to `internal/mcp/server.go` in `handleToolsList()`:

```go
// Add GPU tools if running in sandbox
if os.Getenv("WEE_CONTROL_PLANE_URL") != "" {
    gpuTools := gpu.GetToolDefinitions()
    for _, t := range gpuTools {
        tools = append(tools, Tool{
            Name:        t.Name,
            Description: t.Description,
            InputSchema: InputSchema{/* convert from t.InputSchema */},
        })
    }
}
```

And in `handleToolCall()`:

```go
// GPU tools
case "gpu_launch", "gpu_run", "gpu_push", "gpu_pull", "gpu_status", "gpu_stop":
    gpuManager, err := gpu.NewManagerFromEnv()
    if err != nil {
        return s.errorResponse(req.ID, -32603, fmt.Sprintf("GPU not available: %v", err))
    }
    result := gpu.HandleToolCall(gpuManager, params.Name, params.Arguments)
    return MCPResponse{JSONRPC: "2.0", ID: req.ID, Result: result}
```

**Step 3: Verify it compiles**

```bash
go build ./cmd/wee
```

**Step 4: Commit**

```bash
git add internal/gpu/mcp_tools.go internal/mcp/server.go
git commit -m "feat: add GPU MCP tools for Claude agent access"
```

---

### Task 10: Integration Test — Full Flow

**Files:**
- Create: `internal/gpu/integration_test.go`

**Step 1: Write integration test with mock wee.cat server**

Create `internal/gpu/integration_test.go`:

```go
package gpu

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFullLaunchFlow(t *testing.T) {
	// Mock wee.cat backend
	mux := http.NewServeMux()

	mux.HandleFunc("/api/gpu/templates/", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]Template{
			{ID: "tpl-pytorch", Name: "PyTorch 2.4", ImageName: "runpod/pytorch:2.4"},
			{ID: "tpl-hf", Name: "Hugging Face", ImageName: "runpod/hf:latest"},
		})
	})

	mux.HandleFunc("/api/gpu/launch/", func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		json.NewDecoder(r.Body).Decode(&payload)

		if payload["tier"] != "pro" {
			t.Errorf("expected tier pro, got %v", payload["tier"])
		}

		w.WriteHeader(201)
		json.NewEncoder(w).Encode(GPUSession{
			ID:     1,
			PodID:  "pod-test-123",
			Status: "creating",
			Tier:   "pro",
		})
	})

	mux.HandleFunc("/api/gpu/sessions/", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]GPUSession{
			{ID: 1, PodID: "pod-test-123", Status: "running", Tier: "pro",
				SSHHost: "1.2.3.4", SSHPort: 22222},
		})
	})

	mux.HandleFunc("/api/gpu/sessions/1/status/", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(GPUSession{
			ID: 1, PodID: "pod-test-123", Status: "running", Tier: "pro",
			SSHHost: "1.2.3.4", SSHPort: 22222,
			Live: &LiveData{CostPerHr: 1.64, UptimeSeconds: 1800, TotalCost: 0.82},
		})
	})

	mux.HandleFunc("/api/gpu/sessions/1/stop/", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(GPUSession{ID: 1, Status: "stopped"})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	m := NewManager(server.URL, "42", "test-token", "/tmp/test-key")

	// 1. List templates
	templates, err := m.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates failed: %v", err)
	}
	if len(templates) != 2 {
		t.Errorf("expected 2 templates, got %d", len(templates))
	}

	// 2. Launch
	session, err := m.Launch("pro", "tpl-pytorch", "ssh-ed25519 AAAA...")
	if err != nil {
		t.Fatalf("Launch failed: %v", err)
	}
	if session.PodID != "pod-test-123" {
		t.Errorf("expected pod-test-123, got %s", session.PodID)
	}

	// 3. Status
	status, err := m.Status()
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}
	if status.Live.TotalCost != 0.82 {
		t.Errorf("expected cost 0.82, got %f", status.Live.TotalCost)
	}
	if status.SSHHost != "1.2.3.4" {
		t.Errorf("expected SSH host 1.2.3.4, got %s", status.SSHHost)
	}

	// 4. Stop
	if err := m.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}
```

**Step 2: Run the integration test**

```bash
go test ./internal/gpu/ -v -run TestFullLaunchFlow
```

Expected: All assertions pass.

**Step 3: Run all GPU tests**

```bash
go test ./internal/gpu/ -v
```

**Step 4: Commit**

```bash
git add internal/gpu/integration_test.go
git commit -m "test: add GPU sidecar integration tests"
```

---

### Task 11: Build Verification

**Step 1: Run full test suite**

```bash
cd /path/to/wee
go test ./... -v
```

**Step 2: Build binary**

```bash
go build -o wee ./cmd/wee
```

**Step 3: Verify GPU commands exist**

```bash
./wee gpu --help
./wee gpu launch --help
./wee gpu status --help
./wee gpu shell --help
```

**Step 4: Run Django tests**

```bash
cd /path/to/wee/wee.cat/backend
python manage.py test gpu -v 2
```

**Step 5: Commit any remaining fixes**

```bash
git add -A
git commit -m "chore: build verification and cleanup"
```

---

## Summary

| Task | What | Where |
|------|------|-------|
| 1 | Django GPU app + models | `wee.cat/backend/gpu/` |
| 2 | RunPod API client | `wee.cat/backend/gpu/runpod_client.py` |
| 3 | Connector + session API | `wee.cat/backend/gpu/views.py` |
| 4 | Sandbox token auth | `wee.cat/backend/gpu/sandbox_auth.py` |
| 5 | Go GPU manager package | `internal/gpu/gpu.go` |
| 6 | SSH + rsync helpers | `internal/gpu/ssh.go`, `sync.go` |
| 7 | Cobra CLI commands | `internal/cmd/gpu.go` |
| 8 | Bubbletea TUI | `internal/gpu/tui.go` |
| 9 | MCP tools | `internal/gpu/mcp_tools.go` |
| 10 | Integration tests | `internal/gpu/integration_test.go` |
| 11 | Build verification | Full project |
