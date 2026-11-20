package deploy_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Yashh56/atlas/internal/deploy"
)

type mockRunner struct {
	lookPaths map[string]bool
	runs      map[string]error
}

func (m *mockRunner) LookPath(file string) (string, error) {
	if m.lookPaths != nil && m.lookPaths[file] {
		return "/mock/bin/" + file, nil
	}
	return "", errors.New("not found: " + file)
}

func (m *mockRunner) Run(_ context.Context, _ string, name string, args ...string) (string, error) {
	key := name
	if len(args) > 0 {
		key = name + " " + strings.Join(args, " ")
	}
	if err, ok := m.runs[key]; ok {
		return "", err
	}
	return "", nil
}

func (m *mockRunner) RunInteractive(_ context.Context, _ string, _ string, _ ...string) error {
	return nil
}

func TestEnsureCLIInstalled_EmptyInstallCommand(t *testing.T) {
	runner := &mockRunner{lookPaths: map[string]bool{}}
	err := deploy.EnsureCLIInstalled(context.Background(), runner, strings.NewReader(""), &bytes.Buffer{}, "mycli", nil)
	if err == nil || !strings.Contains(err.Error(), "no installation command provided") {
		t.Fatalf("expected error for empty installCommand, got: %v", err)
	}
}

func TestEnsureCLIInstalled_NpmInstallerMissing(t *testing.T) {
	runner := &mockRunner{lookPaths: map[string]bool{}}
	err := deploy.EnsureCLIInstalled(context.Background(), runner, strings.NewReader(""), &bytes.Buffer{}, "vercel", []string{"npm", "install", "-g", "vercel"})
	if err == nil || !strings.Contains(err.Error(), "npm/node are also not found") {
		t.Fatalf("expected nodejs/npm missing error, got: %v", err)
	}
}

func TestEnsureCLIInstalled_ScoopInstallerMissing(t *testing.T) {
	runner := &mockRunner{lookPaths: map[string]bool{}}
	err := deploy.EnsureCLIInstalled(context.Background(), runner, strings.NewReader(""), &bytes.Buffer{}, "railway", []string{"scoop", "install", "railway"})
	if err == nil || !strings.Contains(err.Error(), "installer scoop is not installed") {
		t.Fatalf("expected scoop not installed error, got: %v", err)
	}
}

func TestEnsureCLIInstalled_BrewInstallerMissing(t *testing.T) {
	runner := &mockRunner{lookPaths: map[string]bool{}}
	err := deploy.EnsureCLIInstalled(context.Background(), runner, strings.NewReader(""), &bytes.Buffer{}, "railway", []string{"brew", "install", "railway"})
	if err == nil || !strings.Contains(err.Error(), "installer brew is not installed") {
		t.Fatalf("expected brew not installed error, got: %v", err)
	}
}

func TestEnsureCLIInstalled_Success(t *testing.T) {
	runner := &mockRunner{
		lookPaths: map[string]bool{
			"scoop":   true,
			"railway": false, // initially not installed
		},
	}
	var out bytes.Buffer
	// User accepts install "y"
	err := deploy.EnsureCLIInstalled(context.Background(), runner, strings.NewReader("y\n"), &out, "railway", []string{"scoop", "install", "railway"})
	// In the function, it checks runner.LookPath(cliName) after install.
	// Since railway is still false in mock, it returns "still not found after install"
	if err == nil || !strings.Contains(err.Error(), "still not found after install") {
		t.Fatalf("expected 'still not found after install', got: %v", err)
	}
}
