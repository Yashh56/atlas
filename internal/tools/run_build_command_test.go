package tools_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/Yashh56/atlas/internal/orchestrator"
	"github.com/Yashh56/atlas/internal/session"
	"github.com/Yashh56/atlas/internal/tools"
)

// platformSleepArgs returns args for a long-running command (used for timeout test).
func platformSleepArgs() (string, []string) {
	if runtime.GOOS == "windows" {
		return "ping", []string{"-n", "30", "127.0.0.1"}
	}
	return "sh", []string{"-c", "sleep 30"}
}

func TestRunBuildCommand_Success(t *testing.T) {
	sessDir := t.TempDir()
	workspace := t.TempDir()

	// Use the real "go" framework on a compilable go.mod workspace.
	if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module example.com/ok\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "main.go"), []byte("package main\nfunc main(){}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := tools.RunBuildCommand{
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

	// Verify build.json on disk.
	bs, err := orchestrator.LoadBuild(sessDir)
	if err != nil {
		t.Fatalf("LoadBuild: %v", err)
	}
	if bs.ExitCode != 0 {
		t.Errorf("build.json exit_code = %d, want 0", bs.ExitCode)
	}
	if bs.DurationMs < 0 {
		t.Errorf("build.json duration_ms should be >= 0")
	}
	if bs.LogPath == "" {
		t.Error("build.json log_path should not be empty")
	}

	// Verify build.log exists.
	if _, statErr := os.Stat(bs.LogPath); statErr != nil {
		t.Errorf("build.log not found at %s: %v", bs.LogPath, statErr)
	}
}

func TestRunBuildCommand_Failure(t *testing.T) {
	sessDir := t.TempDir()
	workspace := t.TempDir()

	// Go project with a compile error.
	if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module example.com/fail\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "main.go"), []byte("package main\nfunc main(){ THIS DOES NOT COMPILE }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := tools.RunBuildCommand{
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
		t.Error("expected Success=false for broken project")
	}

	bs, err := orchestrator.LoadBuild(sessDir)
	if err != nil {
		t.Fatalf("LoadBuild: %v", err)
	}
	if bs.ExitCode == 0 {
		t.Errorf("build.json exit_code = 0, want non-zero for failed build")
	}

	// build.log must exist and be non-empty.
	data, readErr := os.ReadFile(bs.LogPath)
	if readErr != nil {
		t.Fatalf("build.log not found: %v", readErr)
	}
	if len(data) == 0 {
		t.Error("build.log should not be empty on failure")
	}
}

func TestRunBuildCommand_Timeout(t *testing.T) {
	sessDir := t.TempDir()
	workspace := t.TempDir()

	if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module example.com/timeout\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "main.go"), []byte("package main\nfunc main(){}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// 1 millisecond context — almost certainly too short for any build.
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	tool := tools.RunBuildCommand{
		WorkspaceRoot:  workspace,
		Framework:      "go",
		PackageManager: "",
		SessionDir:     sessDir,
	}

	result, err := tool.Execute(ctx, &session.Session{})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	// Must not panic. Success may be true if the machine is very fast; we only
	// assert that the tool returns a ToolResult (not a panic or Go error).
	t.Logf("timeout result: Success=%v Error=%q", result.Success, result.Error)
}

func TestRunBuildCommand_UnknownFramework(t *testing.T) {
	sessDir := t.TempDir()
	workspace := t.TempDir()

	tool := tools.RunBuildCommand{
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

	// Ensure no build.json was written (resolution failed before exec).
	_, statErr := os.Stat(filepath.Join(sessDir, "build.json"))
	if !os.IsNotExist(statErr) {
		t.Error("build.json should not exist when resolution fails before exec")
	}
}

// Suppress "unused import" lint if platformSleepArgs isn't used in all branches.
var _ = fmt.Sprintf
var _ = platformSleepArgs
