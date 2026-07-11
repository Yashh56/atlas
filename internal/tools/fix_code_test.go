package tools_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Yashh56/atlas/internal/session"
	"github.com/Yashh56/atlas/internal/state"
	"github.com/Yashh56/atlas/internal/tools"
)

type fakeLLMClient struct {
	response string
	err      error
}

func (f *fakeLLMClient) Name() string { return "fake" }
func (f *fakeLLMClient) Complete(ctx context.Context, sys, user string) (string, error) {
	return f.response, f.err
}

func TestFixCode_Success(t *testing.T) {
	ws := t.TempDir()
	sessDir := filepath.Join(ws, ".atlas", "sessions", "test_sess")
	os.MkdirAll(filepath.Join(sessDir, "logs"), 0o755)

	// Setup context files
	fw := "go"
	state.SaveJSON(sessDir, "project.json", map[string]interface{}{
		"framework": fw,
	})
	
	logPath := filepath.Join(sessDir, "logs", "build.log")
	os.WriteFile(logPath, []byte("main.go:4: syntax error"), 0o644)
	state.SaveJSON(sessDir, "build.json", map[string]interface{}{
		"log_path": logPath,
	})

	// Setup target file
	os.WriteFile(filepath.Join(ws, "main.go"), []byte("broken code"), 0o644)

	client := &fakeLLMClient{
		response: `{"file": "main.go", "content": "fixed code", "reasoning": "fixed it"}`,
	}

	// We need to create a dummy skills/fix_build.md relative to the test execution path.
	// Since the test runs in internal/tools, the relative path to repo root is ../../
	// Wait, the tool looks in skills/fix_build.md OR ../../skills/fix_build.md.
	// Since we are running `go test ./internal/tools/...`, the working dir is internal/tools.
	// The tool's fallback path `../../skills/fix_build.md` will naturally find the real skill file!
	
	tool := tools.FixCode{
		WorkspaceRoot: ws,
		Client:        client,
		SessionDir:    sessDir,
	}

	res, err := tool.Execute(context.Background(), &session.Session{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected success, got %v: %s", res.Success, res.Error)
	}

	expectedOutput := `Fixed: main.go — "fixed it"`
	if res.Output != expectedOutput {
		t.Errorf("expected %q, got %q", expectedOutput, res.Output)
	}

	content, err := os.ReadFile(filepath.Join(ws, "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "fixed code" {
		t.Errorf("expected 'fixed code', got %q", string(content))
	}
}

func TestFixCode_MalformedJSON(t *testing.T) {
	ws := t.TempDir()
	sessDir := filepath.Join(ws, ".atlas", "sessions", "test_sess")
	os.MkdirAll(filepath.Join(sessDir, "logs"), 0o755)

	state.SaveJSON(sessDir, "project.json", map[string]interface{}{})
	state.SaveJSON(sessDir, "build.json", map[string]interface{}{})

	client := &fakeLLMClient{
		response: `not json`,
	}
	
	tool := tools.FixCode{
		WorkspaceRoot: ws,
		Client:        client,
		SessionDir:    sessDir,
	}

	res, err := tool.Execute(context.Background(), &session.Session{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Malformed JSON should be a clean failure (Success = false, but no error returned from Execute)
	// Wait, ToolResult has an Error string for errors.
	if res.Success {
		t.Fatal("expected failure")
	}
	if res.Output != "LLM returned malformed JSON" {
		t.Errorf("expected malformed JSON output, got %q", res.Output)
	}
}

func TestFixCode_NullFile(t *testing.T) {
	ws := t.TempDir()
	sessDir := filepath.Join(ws, ".atlas", "sessions", "test_sess")
	os.MkdirAll(filepath.Join(sessDir, "logs"), 0o755)

	state.SaveJSON(sessDir, "project.json", map[string]interface{}{})
	state.SaveJSON(sessDir, "build.json", map[string]interface{}{})

	client := &fakeLLMClient{
		response: `{"file": null, "content": null, "reasoning": "too hard"}`,
	}
	
	tool := tools.FixCode{
		WorkspaceRoot: ws,
		Client:        client,
		SessionDir:    sessDir,
	}

	res, err := tool.Execute(context.Background(), &session.Session{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Success {
		t.Fatal("expected failure")
	}
	if res.Output != "LLM declined to fix: too hard" {
		t.Errorf("unexpected output: %q", res.Output)
	}
}
