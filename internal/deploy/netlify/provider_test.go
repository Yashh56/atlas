package netlify

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Yashh56/atlas/internal/deploy"
	"github.com/Yashh56/atlas/internal/state"
)

func TestNetlifyProvider_Deploy_SiteResolution(t *testing.T) {
	sessDir := t.TempDir()
	workspaceDir := t.TempDir()
	
	// Create a mock project.json with framework
	proj := projectData{
		Framework: ptr("react"),
	}
	_ = state.SaveJSON(sessDir, "project.json", &proj)
	
	// Create mock vite.config.js to force "dist" output dir
	_ = os.WriteFile(filepath.Join(workspaceDir, "vite.config.js"), []byte{}, 0644)

	calls := 0
	runner := &fakeRunnerDynamic{
		lookPaths: map[string]bool{"netlify": true},
		runFn: func(dir string, name string, args ...string) (string, error) {
			if name != "netlify" {
				return "", errors.New("unexpected cmd")
			}
			
			if args[0] == "sites:create" {
				calls++
				if calls == 1 {
					// Simulate a collision error on first attempt
					return "", errors.New("already taken")
				}
				// Succeed on second attempt
				return `{"id": "test-site-id-123"}`, nil
			}
			
			if args[0] == "deploy" {
				// Assert args
				cmdLine := strings.Join(args, " ")
				if !strings.Contains(cmdLine, "--site test-site-id-123") {
					t.Errorf("deploy missing site id, got %s", cmdLine)
				}
				if !strings.Contains(cmdLine, "--dir dist") {
					t.Errorf("deploy missing resolved dir, got %s", cmdLine)
				}
				return `{"url": "https://test.netlify.app"}`, nil
			}
			
			return "", errors.New("unexpected args")
		},
	}
	
	provider := &NetlifyProvider{Runner: runner}
	
	res, err := provider.Deploy(context.Background(), deploy.DeployInput{
		WorkspaceRoot: workspaceDir,
		SessionDir:    sessDir,
	})
	
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}
	if res.URL != "https://test.netlify.app" {
		t.Errorf("unexpected URL: %s", res.URL)
	}
	if calls != 2 {
		t.Errorf("expected 2 sites:create calls due to collision retry, got %d", calls)
	}
	
	// Check if project.json was updated with site ID
	var updatedProj projectData
	_ = state.LoadJSON(sessDir, "project.json", &updatedProj)
	if updatedProj.NetlifySiteID != "test-site-id-123" {
		t.Errorf("expected site ID to be saved, got %s", updatedProj.NetlifySiteID)
	}
}

func ptr(s string) *string {
	return &s
}
