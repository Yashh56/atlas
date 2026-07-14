package tools_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Yashh56/atlas/internal/orchestrator"
	"github.com/Yashh56/atlas/internal/session"
	"github.com/Yashh56/atlas/internal/state"
	"github.com/Yashh56/atlas/internal/tools"
)

func TestRunTests_Success(t *testing.T) {
	sessDir := t.TempDir()
	workspace := t.TempDir()

	// Use "go" framework which resolves to `go test ./...`
	// To make `go test ./...` succeed, we need a valid go module with passing tests.
	if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module example.com/ok\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "main_test.go"), []byte("package main\nimport \"testing\"\nfunc TestOK(t *testing.T){}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := tools.RunTests{
		WorkspaceRoot:  workspace,
		Framework:      "go",
		PackageManager: "",
		SessionDir:     sessDir,
	}

	result, err := tool.Execute(context.Background(), &session.Session{})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected Success=true; Error=%q, LogPath=%q", result.Error, result.Output)
	}

	// Verify test.json on disk.
	var ts struct {
		ExitCode   int    `json:"exit_code"`
		DurationMs int64  `json:"duration_ms"`
		LogPath    string `json:"log_path"`
	}
	if err := state.LoadJSON(sessDir, "test.json", &ts); err != nil {
		t.Fatalf("Load test.json: %v", err)
	}
	if ts.ExitCode != 0 {
		t.Errorf("test.json exit_code = %d, want 0", ts.ExitCode)
	}
	if ts.LogPath == "" {
		t.Error("test.json log_path should not be empty")
	}

	// Verify test.log exists.
	if _, statErr := os.Stat(ts.LogPath); statErr != nil {
		t.Errorf("test.log not found at %s: %v", ts.LogPath, statErr)
	}
}

func TestRunTests_Failure(t *testing.T) {
	sessDir := t.TempDir()
	workspace := t.TempDir()

	// Go project with a failing test.
	if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module example.com/fail\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "main_test.go"), []byte("package main\nimport \"testing\"\nfunc TestFail(t *testing.T){ t.Fatal(\"failed\") }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := tools.RunTests{
		WorkspaceRoot:  workspace,
		Framework:      "go",
		PackageManager: "",
		SessionDir:     sessDir,
	}

	result, err := tool.Execute(context.Background(), &session.Session{})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if result.Success {
		t.Error("expected Success=false for broken tests")
	}

	var ts struct {
		ExitCode int    `json:"exit_code"`
		LogPath  string `json:"log_path"`
	}
	if err := state.LoadJSON(sessDir, "test.json", &ts); err != nil {
		t.Fatalf("Load test.json: %v", err)
	}
	if ts.ExitCode == 0 {
		t.Errorf("test.json exit_code = 0, want non-zero for failed tests")
	}

	data, readErr := os.ReadFile(ts.LogPath)
	if readErr != nil {
		t.Fatalf("test.log not found: %v", readErr)
	}
	if len(data) == 0 {
		t.Error("test.log should not be empty on failure")
	}
}

func TestRunTests_UnknownFramework(t *testing.T) {
	sessDir := t.TempDir()
	workspace := t.TempDir()

	tool := tools.RunTests{
		WorkspaceRoot:  workspace,
		Framework:      "rust",
		PackageManager: "",
		SessionDir:     sessDir,
	}

	result, err := tool.Execute(context.Background(), &session.Session{})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if result.Success {
		t.Error("expected Success=false for unknown framework")
	}
	if result.Error == "" {
		t.Error("expected non-empty Error for unknown framework")
	}

	// Ensure no test.json was written (resolution failed before exec).
	_, statErr := os.Stat(filepath.Join(sessDir, "test.json"))
	if !os.IsNotExist(statErr) {
		t.Error("test.json should not exist when resolution fails before exec")
	}
}

var _ = fmt.Sprintf
var _ = orchestrator.Run
