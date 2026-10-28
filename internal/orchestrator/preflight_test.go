package orchestrator

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Yashh56/atlas/internal/config"
	"github.com/Yashh56/atlas/internal/deploy"
	"github.com/Yashh56/atlas/internal/workspace"
)

type mockDeployProvider struct {
	deployed bool
}

func (m *mockDeployProvider) Name() string { return "mock" }
func (m *mockDeployProvider) Deploy(ctx context.Context, input deploy.DeployInput) (*deploy.Deployment, error) {
	m.deployed = true
	return &deploy.Deployment{URL: "https://mock.deploy"}, nil
}
func (m *mockDeployProvider) HealthCheck(ctx context.Context, d *deploy.Deployment) error {
	return nil
}
func (m *mockDeployProvider) Rollback(ctx context.Context, prev *deploy.Deployment, input deploy.DeployInput) error {
	return nil
}

func TestExecuteDeploy_StrictEnvMissing(t *testing.T) {
	tempDir := t.TempDir()
	envExample := filepath.Join(tempDir, ".env.example")
	_ = os.WriteFile(envExample, []byte("REQUIRED_DB_KEY=\n"), 0644)

	ws := &workspace.Workspace{
		Root:   tempDir,
		Exists: true,
	}
	sessDir := t.TempDir()
	planner := NewPlanner("deploy")
	cfg := &config.Config{Approval: "auto"}
	dep := NewDeployment("mock")
	mockProv := &mockDeployProvider{}

	opts := RunOptions{
		StrictEnv:      true,
		NonInteractive: true,
		JSONMode:       true,
	}

	err := executeDeploy(context.Background(), ws, sessDir, planner, cfg, dep, mockProv, "mock", opts)
	if err == nil {
		t.Fatal("expected error due to StrictEnv and missing secret, got nil")
	}

	if !strings.Contains(err.Error(), "pre-flight secret audit failed") || !strings.Contains(err.Error(), "--strict-env is set") {
		t.Errorf("unexpected error message: %v", err)
	}

	if mockProv.deployed {
		t.Errorf("deploy should not have been called when strict env fails")
	}
}

func TestExecuteDeploy_NonInteractiveProceedsWithWarning(t *testing.T) {
	tempDir := t.TempDir()
	envExample := filepath.Join(tempDir, ".env.example")
	_ = os.WriteFile(envExample, []byte("REQUIRED_DB_KEY=\n"), 0644)

	ws := &workspace.Workspace{
		Root:   tempDir,
		Exists: true,
	}
	sessDir := t.TempDir()
	planner := NewPlanner("deploy")
	cfg := &config.Config{Approval: "auto"}
	dep := NewDeployment("mock")
	mockProv := &mockDeployProvider{}

	opts := RunOptions{
		StrictEnv:      false,
		NonInteractive: true,
		JSONMode:       true,
	}

	err := executeDeploy(context.Background(), ws, sessDir, planner, cfg, dep, mockProv, "mock", opts)
	if err != nil {
		t.Fatalf("expected non-interactive without strict-env to proceed, got: %v", err)
	}

	if !mockProv.deployed {
		t.Errorf("deploy should have been called")
	}
}

func TestExecuteDeploy_StrictEnvAuditError(t *testing.T) {
	tempDir := t.TempDir()
	envExample := filepath.Join(tempDir, ".env.example")
	_ = os.Mkdir(envExample, 0755)

	ws := &workspace.Workspace{
		Root:   tempDir,
		Exists: true,
	}
	sessDir := t.TempDir()
	planner := NewPlanner("deploy")
	cfg := &config.Config{Approval: "auto"}
	dep := NewDeployment("mock")
	mockProv := &mockDeployProvider{}

	opts := RunOptions{
		StrictEnv:      true,
		NonInteractive: true,
		JSONMode:       true,
	}

	err := executeDeploy(context.Background(), ws, sessDir, planner, cfg, dep, mockProv, "mock", opts)
	if err == nil {
		t.Fatal("expected error due to StrictEnv and audit error, got nil")
	}

	if !strings.Contains(err.Error(), "pre-flight secret audit failed:") {
		t.Errorf("unexpected error message: %v", err)
	}

	if mockProv.deployed {
		t.Errorf("deploy should not have been called when strict env fails")
	}
}

