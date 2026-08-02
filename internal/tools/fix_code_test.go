package tools_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Yashh56/atlas/internal/session"
	"github.com/Yashh56/atlas/internal/state"
	"github.com/Yashh56/atlas/internal/tools"
	"github.com/zendev-sh/goai/provider"
)

type fakeLanguageModel struct {
	response      string
	err           error
	toolCallCount int
	maxToolCalls  int
	callCount     int
}

func (f *fakeLanguageModel) ModelID() string { return "fake-model" }

func (f *fakeLanguageModel) DoGenerate(_ context.Context, params provider.GenerateParams) (*provider.GenerateResult, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.callCount++
	
	if f.toolCallCount > 0 && f.callCount <= f.maxToolCalls {
		return &provider.GenerateResult{
			ToolCalls: []provider.ToolCall{
				{
					ID:    fmt.Sprintf("call_%d", f.callCount),
					Name:  "read_file",
					Input: json.RawMessage(`{"path": "some_file.go"}`),
				},
			},
			FinishReason: provider.FinishToolCalls,
		}, nil
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

func TestFixCode_BoundedLoopSuccess(t *testing.T) {
	ws := t.TempDir()
	sessDir := filepath.Join(ws, ".atlas", "sessions", "test_sess")
	os.MkdirAll(filepath.Join(sessDir, "logs"), 0o755)

	state.SaveJSON(sessDir, "project.json", map[string]interface{}{})
	state.SaveJSON(sessDir, "build.json", map[string]interface{}{})
	os.WriteFile(filepath.Join(ws, "some_file.go"), []byte("test content"), 0o644)

	model := &fakeLanguageModel{
		response:      `{"file": "some_file.go", "old_str": "test", "new_str": "fixed", "reasoning": "done"}`,
		toolCallCount: 2, // Request 2 tool calls
		maxToolCalls:  2,
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
	if !res.Success {
		t.Fatalf("expected success, got error: %s", res.Error)
	}
	// Verify it actually called the model 3 times (2 tool calls + 1 final answer)
	if model.callCount != 3 {
		t.Fatalf("expected 3 calls, got %d", model.callCount)
	}
}

func TestFixCode_BoundedLoopTermination(t *testing.T) {
	ws := t.TempDir()
	sessDir := filepath.Join(ws, ".atlas", "sessions", "test_sess")
	os.MkdirAll(filepath.Join(sessDir, "logs"), 0o755)

	state.SaveJSON(sessDir, "project.json", map[string]interface{}{})
	state.SaveJSON(sessDir, "build.json", map[string]interface{}{})
	os.WriteFile(filepath.Join(ws, "some_file.go"), []byte("test content"), 0o644)

	model := &fakeLanguageModel{
		response:      `{"file": "some_file.go", "old_str": "test", "new_str": "fixed", "reasoning": "done"}`,
		toolCallCount: 100, // Attempt an infinite loop
		maxToolCalls:  100,
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
		t.Fatalf("expected failure from infinite loop")
	}
	// goai's max steps error
	if res.Error == "" {
		t.Fatalf("expected max steps error, got empty string")
	}
	// The call count shouldn't exceed 8 (3 from GenerateStructured + 5 from GenerateText fallback)
	if model.callCount > 8 {
		t.Fatalf("model was called too many times: %d, maxSteps enforcement failed", model.callCount)
	}
}
