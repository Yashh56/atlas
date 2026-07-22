package tools_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Yashh56/atlas/internal/session"
	"github.com/Yashh56/atlas/internal/state"
	"github.com/Yashh56/atlas/internal/tools"
	"github.com/zendev-sh/goai/provider"
)

// fakeLanguageModel implements provider.LanguageModel for testing.
// It returns a canned JSON response for FixResponse.
type fakeLanguageModel struct {
	response string
	err      error
}

func (f *fakeLanguageModel) ModelID() string { return "fake-model" }

func (f *fakeLanguageModel) DoGenerate(_ context.Context, params provider.GenerateParams) (*provider.GenerateResult, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &provider.GenerateResult{
		Text:         f.response,
		FinishReason: provider.FinishStop,
	}, nil
}

func (f *fakeLanguageModel) DoStream(_ context.Context, _ provider.GenerateParams) (*provider.StreamResult, error) {
	return nil, nil
}

func TestFixCode_Success(t *testing.T) {
	ws := t.TempDir()
	sessDir := filepath.Join(ws, ".atlas", "sessions", "test_sess")
	os.MkdirAll(filepath.Join(sessDir, "logs"), 0o755)

	fw := "go"
	state.SaveJSON(sessDir, "project.json", map[string]interface{}{
		"framework": fw,
	})

	logPath := filepath.Join(sessDir, "logs", "build.log")
	os.WriteFile(logPath, []byte("main.go:4: syntax error"), 0o644)
	state.SaveJSON(sessDir, "build.json", map[string]interface{}{
		"log_path": logPath,
	})

	os.WriteFile(filepath.Join(ws, "main.go"), []byte("broken code"), 0o644)

	model := &fakeLanguageModel{
		response: `{"file": "main.go", "old_str": "broken code", "new_str": "fixed code", "reasoning": "fixed it"}`,
	}

	tool := tools.FixCode{
		WorkspaceRoot: ws,
		Model:         model,
		SessionDir:    sessDir,
	}

	res, err := tool.Execute(context.Background(), &session.Session{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected success, got error: %s", res.Error)
	}

	content, err := os.ReadFile(filepath.Join(ws, "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "fixed code" {
		t.Errorf("expected 'fixed code', got %q", string(content))
	}
}

func TestFixCode_StructuredGenerationError(t *testing.T) {
	ws := t.TempDir()
	sessDir := filepath.Join(ws, ".atlas", "sessions", "test_sess")
	os.MkdirAll(filepath.Join(sessDir, "logs"), 0o755)

	state.SaveJSON(sessDir, "project.json", map[string]interface{}{})
	state.SaveJSON(sessDir, "build.json", map[string]interface{}{})

	model := &fakeLanguageModel{
		err: fmt.Errorf("API rate limit exceeded"),
	}

	tool := tools.FixCode{
		WorkspaceRoot: ws,
		Model:         model,
		SessionDir:    sessDir,
	}

	res, err := tool.Execute(context.Background(), &session.Session{})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if res.Success {
		t.Fatal("expected failure")
	}
	// This is the blank-error bug fix: Error must be non-empty and descriptive
	if res.Error == "" {
		t.Fatal("ToolResult.Error is blank — the blank-error bug is still present")
	}
	if res.Error == "structured generation failed: " {
		t.Fatal("ToolResult.Error has the blank message pattern — fix didn't work")
	}
	t.Logf("Error message (correct): %s", res.Error)
}

func TestFixCode_ModelDeclines(t *testing.T) {
	ws := t.TempDir()
	sessDir := filepath.Join(ws, ".atlas", "sessions", "test_sess")
	os.MkdirAll(filepath.Join(sessDir, "logs"), 0o755)

	state.SaveJSON(sessDir, "project.json", map[string]interface{}{})
	state.SaveJSON(sessDir, "build.json", map[string]interface{}{})

	model := &fakeLanguageModel{
		response: `{"file": "", "content": "", "reasoning": "cannot determine the fix"}`,
	}

	tool := tools.FixCode{
		WorkspaceRoot: ws,
		Model:         model,
		SessionDir:    sessDir,
	}

	res, err := tool.Execute(context.Background(), &session.Session{})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if res.Success {
		t.Fatal("expected failure when model declines")
	}
	if res.Error == "" {
		t.Fatal("expected non-empty error when model declines")
	}
	t.Logf("Decline message: %s", res.Error)
}
