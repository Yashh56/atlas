package tools_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Yashh56/atlas/internal/session"
	"github.com/Yashh56/atlas/internal/tools"
)

// initGitRepo runs git init and configures dummy user so commits work in CI.
func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	gitRun := func(args ...string) {
		t.Helper()
		rc := tools.RunCommand{Command: "git", Args: args, Dir: dir}
		res, err := rc.Execute(context.Background(), &session.Session{})
		if err != nil || !res.Success {
			t.Logf("git %v output: %s err: %v success: %v", args, res.Output, err, res.Success)
		}
	}
	gitRun("init")
	gitRun("config", "user.email", "test@atlas.local")
	gitRun("config", "user.name", "Atlas Test")
}

func TestGitValidate_CleanRepo(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)

	// Write and commit a file so HEAD exists.
	f := filepath.Join(dir, "README.md")
	if err := os.WriteFile(f, []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	rc := tools.RunCommand{Command: "git", Args: []string{"add", "."}, Dir: dir}
	rc.Execute(context.Background(), &session.Session{})
	rc2 := tools.RunCommand{Command: "git", Args: []string{"commit", "-m", "init"}, Dir: dir}
	rc2.Execute(context.Background(), &session.Session{})

	sessDir := t.TempDir()
	tool := tools.GitValidate{WorkspaceRoot: dir, GitRoot: dir, SessionDir: sessDir}
	result, err := tool.Execute(context.Background(), &session.Session{})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected Success=true, got false; Error=%q", result.Error)
	}
	// Output should encode is_clean:true
	if result.Output != "is_clean:true" {
		t.Errorf("Output = %q, want %q", result.Output, "is_clean:true")
	}
}

func TestGitValidate_DirtyRepo(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)

	// Commit initial state.
	f := filepath.Join(dir, "main.go")
	if err := os.WriteFile(f, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rc := tools.RunCommand{Command: "git", Args: []string{"add", "."}, Dir: dir}
	rc.Execute(context.Background(), &session.Session{})
	rc2 := tools.RunCommand{Command: "git", Args: []string{"commit", "-m", "init"}, Dir: dir}
	rc2.Execute(context.Background(), &session.Session{})

	// Modify a tracked file to make it dirty.
	if err := os.WriteFile(f, []byte("package main\n// dirty\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	sessDir := t.TempDir()
	tool := tools.GitValidate{WorkspaceRoot: dir, GitRoot: dir, SessionDir: sessDir}
	result, err := tool.Execute(context.Background(), &session.Session{})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected Success=true, got false; Error=%q", result.Error)
	}
	if result.Output != "is_clean:false" {
		t.Errorf("Output = %q, want %q", result.Output, "is_clean:false")
	}
}

func TestGitValidate_NoRemote(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)

	f := filepath.Join(dir, "go.mod")
	if err := os.WriteFile(f, []byte("module example.com/x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rc := tools.RunCommand{Command: "git", Args: []string{"add", "."}, Dir: dir}
	rc.Execute(context.Background(), &session.Session{})
	rc2 := tools.RunCommand{Command: "git", Args: []string{"commit", "-m", "init"}, Dir: dir}
	rc2.Execute(context.Background(), &session.Session{})

	sessDir := t.TempDir()
	tool := tools.GitValidate{WorkspaceRoot: dir, GitRoot: dir, SessionDir: sessDir}
	result, err := tool.Execute(context.Background(), &session.Session{})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected Success=true, got false; Error=%q", result.Error)
	}
	// No remote — tool must still succeed and is_clean must be set.
	if result.Output != "is_clean:true" && result.Output != "is_clean:false" {
		t.Errorf("Output = %q — expected is_clean status even without remote", result.Output)
	}
}

func TestGitValidate_NotAGitRepo(t *testing.T) {
	dir := t.TempDir()

	sessDir := t.TempDir()
	// GitRoot "" signals "not a git repo"
	tool := tools.GitValidate{WorkspaceRoot: dir, GitRoot: "", SessionDir: sessDir}
	result, err := tool.Execute(context.Background(), &session.Session{})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected Success=true even for non-git workspace; Error=%q", result.Error)
	}
	if result.Output != "no_git_repo" {
		t.Errorf("Output = %q, want %q", result.Output, "no_git_repo")
	}
}

// ensure RunCommand is accessible from test (exported).
var _ = runtime.GOOS
