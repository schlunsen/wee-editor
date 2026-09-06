package gpu

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	m := NewManager("https://example.com/", "sandbox-1", "tok-123", "/tmp/key")
	if m.weeCatURL != "https://example.com" {
		t.Errorf("expected trailing slash stripped, got %q", m.weeCatURL)
	}
	if m.sandboxID != "sandbox-1" {
		t.Errorf("expected sandbox-1, got %q", m.sandboxID)
	}
	if m.authToken != "tok-123" {
		t.Errorf("expected tok-123, got %q", m.authToken)
	}
	if m.SSHKeyPath() != "/tmp/key" {
		t.Errorf("expected /tmp/key, got %q", m.SSHKeyPath())
	}
}

func TestNewManagerFromEnv(t *testing.T) {
	// Missing env vars should error
	t.Run("missing URL", func(t *testing.T) {
		t.Setenv("WEE_CONTROL_PLANE_URL", "")
		t.Setenv("WEE_SANDBOX_ID", "sb-1")
		t.Setenv("WEE_CONTROL_PLANE_TOKEN", "tok")
		_, err := NewManagerFromEnv()
		if err == nil {
			t.Fatal("expected error for missing URL")
		}
	})

	t.Run("missing sandbox ID", func(t *testing.T) {
		t.Setenv("WEE_CONTROL_PLANE_URL", "https://wee.cat")
		t.Setenv("WEE_SANDBOX_ID", "")
		t.Setenv("WEE_CONTROL_PLANE_TOKEN", "tok")
		_, err := NewManagerFromEnv()
		if err == nil {
			t.Fatal("expected error for missing sandbox ID")
		}
	})

	t.Run("missing token", func(t *testing.T) {
		t.Setenv("WEE_CONTROL_PLANE_URL", "https://wee.cat")
		t.Setenv("WEE_SANDBOX_ID", "sb-1")
		t.Setenv("WEE_CONTROL_PLANE_TOKEN", "")
		_, err := NewManagerFromEnv()
		if err == nil {
			t.Fatal("expected error for missing token")
		}
	})

	t.Run("success", func(t *testing.T) {
		t.Setenv("WEE_CONTROL_PLANE_URL", "https://wee.cat")
		t.Setenv("WEE_SANDBOX_ID", "sb-1")
		t.Setenv("WEE_CONTROL_PLANE_TOKEN", "tok-abc")
		m, err := NewManagerFromEnv()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.weeCatURL != "https://wee.cat" {
			t.Errorf("unexpected URL: %q", m.weeCatURL)
		}
		if m.sandboxID != "sb-1" {
			t.Errorf("unexpected sandbox ID: %q", m.sandboxID)
		}
		if m.authToken != "tok-abc" {
			t.Errorf("unexpected token: %q", m.authToken)
		}
	})
}

func TestDoRequest_AuthHeader(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	}))
	defer srv.Close()

	m := NewManager(srv.URL, "sb-1", "my-secret-token", "/tmp/key")
	resp, err := m.doRequest("GET", "/api/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()

	expected := "SandboxToken my-secret-token"
	if gotAuth != expected {
		t.Errorf("expected auth header %q, got %q", expected, gotAuth)
	}
}

func TestLaunch(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/gpu/launch/" {
			t.Errorf("expected /api/gpu/launch/, got %s", r.URL.Path)
		}

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["sandbox_id"] != "sb-1" {
			t.Errorf("unexpected sandbox_id: %q", body["sandbox_id"])
		}
		if body["tier"] != "hobby" {
			t.Errorf("unexpected tier: %q", body["tier"])
		}

		session := GPUSession{
			ID:           42,
			PodID:        "pod-abc",
			TemplateName: "pytorch",
			Tier:         "hobby",
			GPUType:      "RTX 4090",
			Status:       "starting",
			CreatedAt:    now,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(session)
	}))
	defer srv.Close()

	m := NewManager(srv.URL, "sb-1", "tok", "/tmp/key")
	session, err := m.Launch("hobby", "tpl-1", "ssh-ed25519 AAAA...", 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session.ID != 42 {
		t.Errorf("expected ID 42, got %d", session.ID)
	}
	if session.PodID != "pod-abc" {
		t.Errorf("expected pod-abc, got %q", session.PodID)
	}
	if session.Status != "starting" {
		t.Errorf("expected starting, got %q", session.Status)
	}
}

func TestLaunch_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid tier"}`))
	}))
	defer srv.Close()

	m := NewManager(srv.URL, "sb-1", "tok", "/tmp/key")
	_, err := m.Launch("invalid", "tpl-1", "ssh-ed25519 AAAA...", 0, 0)
	if err == nil {
		t.Fatal("expected error for bad request")
	}
}

func TestStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/gpu/sessions/":
			wrapper := map[string][]GPUSession{
				"sessions": {{ID: 10, PodID: "pod-xyz", Status: "running", SSHHost: "gpu.example.com", SSHPort: 22}},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(wrapper)
		case "/api/gpu/sessions/10/status/":
			session := GPUSession{
				ID:     10,
				Status: "running",
				Live: &LiveData{
					CostPerHr:     0.74,
					UptimeSeconds: 3600,
					TotalCost:     0.74,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(session)
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	m := NewManager(srv.URL, "sb-1", "tok", "/tmp/key")
	session, err := m.Status()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session.Live == nil {
		t.Fatal("expected live data")
	}
	if session.Live.CostPerHr != 0.74 {
		t.Errorf("expected cost 0.74, got %f", session.Live.CostPerHr)
	}
}

func TestListTemplates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/gpu/templates/" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		wrapper := map[string][]Template{
			"templates": {
				{ID: "tpl-1", Name: "PyTorch", ImageName: "pytorch/pytorch:latest"},
				{ID: "tpl-2", Name: "TensorFlow", ImageName: "tensorflow/tensorflow:latest"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(wrapper)
	}))
	defer srv.Close()

	m := NewManager(srv.URL, "sb-1", "tok", "/tmp/key")
	templates, err := m.ListTemplates()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(templates) != 2 {
		t.Fatalf("expected 2 templates, got %d", len(templates))
	}
	if templates[0].Name != "PyTorch" {
		t.Errorf("expected PyTorch, got %q", templates[0].Name)
	}
}

func TestListSessions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("sandbox_id") != "sb-1" {
			t.Errorf("expected sandbox_id=sb-1, got %q", r.URL.Query().Get("sandbox_id"))
		}
		wrapper := map[string][]GPUSession{
			"sessions": {
				{ID: 1, Status: "running"},
				{ID: 2, Status: "stopped"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(wrapper)
	}))
	defer srv.Close()

	m := NewManager(srv.URL, "sb-1", "tok", "/tmp/key")
	sessions, err := m.ListSessions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}
}

func TestActiveSession(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			wrapper := map[string][]GPUSession{
				"sessions": {
					{ID: 1, Status: "stopped"},
					{ID: 2, Status: "running", SSHHost: "gpu.example.com", SSHPort: 22},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(wrapper)
		}))
		defer srv.Close()

		m := NewManager(srv.URL, "sb-1", "tok", "/tmp/key")
		session, err := m.ActiveSession()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if session == nil {
			t.Fatal("expected active session")
		}
		if session.ID != 2 {
			t.Errorf("expected ID 2, got %d", session.ID)
		}
	})

	t.Run("none", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			wrapper := map[string][]GPUSession{
				"sessions": {
					{ID: 1, Status: "stopped"},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(wrapper)
		}))
		defer srv.Close()

		m := NewManager(srv.URL, "sb-1", "tok", "/tmp/key")
		session, err := m.ActiveSession()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if session != nil {
			t.Errorf("expected nil, got session ID %d", session.ID)
		}
	})
}
