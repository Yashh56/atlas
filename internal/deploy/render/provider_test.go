package render_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Yashh56/atlas/internal/deploy"
	"github.com/Yashh56/atlas/internal/deploy/render"
	"github.com/Yashh56/atlas/internal/state"
)

func TestRenderProvider_Deploy(t *testing.T) {
	t.Setenv("RENDER_API_KEY", "mock-token")
	// Set up mock server
	var requestedTrigger bool
	var requestedStatus bool
	var requestedService bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer mock-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		switch {
		case r.Method == "POST" && r.URL.Path == "/v1/services/srv-123/deploys":
			requestedTrigger = true
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if body["commitId"] != "abcdef123" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"id": "dep-456"}`))

		case r.Method == "GET" && r.URL.Path == "/v1/services/srv-123/deploys/dep-456":
			requestedStatus = true
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			// Mocking that the deploy succeeded immediately
			w.Write([]byte(`{"status": "live"}`))

		case r.Method == "GET" && r.URL.Path == "/v1/services/srv-123":
			requestedService = true
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"serviceDetails": {
					"webServiceDetails": {
						"url": "https://myapp.onrender.com"
					}
				}
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	os.Setenv("RENDER_TOKEN", "mock-token")
	defer os.Unsetenv("RENDER_TOKEN")

	// Create a dummy workspace and project.json
	wsRoot := t.TempDir()
	atlasDir := filepath.Join(wsRoot, ".atlas")
	os.MkdirAll(atlasDir, 0755)

	serviceID := "srv-123"
	sha := "abcdef123"
	proj := struct {
		RenderServiceID *string `json:"render_service_id"`
		Git             struct {
			CommitSHA *string `json:"commit_sha"`
		} `json:"git"`
	}{
		RenderServiceID: &serviceID,
	}
	proj.Git.CommitSHA = &sha
	state.SaveJSON(atlasDir, "project.json", &proj)

	provider := &render.RenderProvider{
		BaseURL:      srv.URL,
		TickInterval: 10 * time.Millisecond,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	in := deploy.DeployInput{
		WorkspaceRoot: wsRoot,
		SessionDir:    atlasDir,
		Environment:   "production",
	}
	dep, err := provider.Deploy(ctx, in)
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	if !requestedTrigger {
		t.Error("Expected trigger request")
	}
	if !requestedStatus {
		t.Error("Expected status request")
	}
	if !requestedService {
		t.Error("Expected service fetch request")
	}

	if dep == nil {
		t.Fatal("Expected deployment, got nil")
	}
	if dep.URL != "https://myapp.onrender.com" {
		t.Errorf("Expected URL %q, got %q", "https://myapp.onrender.com", dep.URL)
	}
	if dep.Provider != "render" {
		t.Errorf("Expected provider 'render', got %q", dep.Provider)
	}
}
