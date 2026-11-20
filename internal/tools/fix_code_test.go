package tools_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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


func TestFixCode_EmbeddedSkillFallback(t *testing.T) {
	ws := t.TempDir()
	sessDir := filepath.Join(ws, ".atlas", "sessions", "test_sess")
	os.MkdirAll(filepath.Join(sessDir, "logs"), 0o755)

	state.SaveJSON(sessDir, "project.json", map[string]interface{}{"framework": "go"})
	logPath := filepath.Join(sessDir, "logs", "build.log")
	os.WriteFile(logPath, []byte("main.go:4: syntax error"), 0o644)
	state.SaveJSON(sessDir, "build.json", map[string]interface{}{"log_path": logPath})
	os.WriteFile(filepath.Join(ws, "main.go"), []byte("package main\nvar x = 1\n"), 0o644)

	model := &fakeLanguageModel{
		response: `{"file": "main.go", "old_str": "var x = 1", "new_str": "var x = 2", "reasoning": "increment x"}`,
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
		t.Fatalf("expected success with embedded skill, got error: %s", res.Error)
	}
}

func TestFixCode_WorkspaceSkillOverride(t *testing.T) {
	ws := t.TempDir()
	sessDir := filepath.Join(ws, ".atlas", "sessions", "test_sess")
	os.MkdirAll(filepath.Join(sessDir, "logs"), 0o755)

	customSkillDir := filepath.Join(ws, ".atlas", "skills")
	os.MkdirAll(customSkillDir, 0o755)
	customSkillPath := filepath.Join(customSkillDir, "fix_build.md")
	os.WriteFile(customSkillPath, []byte("Custom fix build instructions"), 0o644)

	state.SaveJSON(sessDir, "project.json", map[string]interface{}{"framework": "go"})
	logPath := filepath.Join(sessDir, "logs", "build.log")
	os.WriteFile(logPath, []byte("main.go:4: syntax error"), 0o644)
	state.SaveJSON(sessDir, "build.json", map[string]interface{}{"log_path": logPath})
	os.WriteFile(filepath.Join(ws, "main.go"), []byte("package main\nvar x = 1\n"), 0o644)

	model := &fakeLanguageModel{
		response: `{"file": "main.go", "old_str": "var x = 1", "new_str": "var x = 2", "reasoning": "increment x"}`,
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
		t.Fatalf("expected success with workspace skill override, got error: %s", res.Error)
	}
}

type fakeInspectModel struct {
	fn func(params provider.GenerateParams) (*provider.GenerateResult, error)
}

func (f *fakeInspectModel) ModelID() string { return "fake-inspect-model" }

func (f *fakeInspectModel) DoGenerate(_ context.Context, params provider.GenerateParams) (*provider.GenerateResult, error) {
	if f.fn != nil {
		return f.fn(params)
	}
	return &provider.GenerateResult{Text: `{"file":"main.go","old_str":"a","new_str":"b"}`, FinishReason: provider.FinishStop}, nil
}

func (f *fakeInspectModel) DoStream(_ context.Context, _ provider.GenerateParams) (*provider.StreamResult, error) {
	return nil, nil
}

func TestFixCode_ContextBudget_SkipsSubdirectoriesInWorkspaceRoot(t *testing.T) {
	ws := t.TempDir()
	sessDir := filepath.Join(ws, ".atlas", "sessions", "test_sess")
	os.MkdirAll(filepath.Join(sessDir, "logs"), 0o755)

	state.SaveJSON(sessDir, "project.json", map[string]interface{}{"framework": "react"})
	logPath := filepath.Join(sessDir, "logs", "build.log")
	os.WriteFile(logPath, []byte("package.json:2: invalid json"), 0o644)
	state.SaveJSON(sessDir, "build.json", map[string]interface{}{"log_path": logPath})

	os.WriteFile(filepath.Join(ws, "package.json"), []byte(`{"name": "root-app"}`), 0o644)

	srcDir := filepath.Join(ws, "src")
	os.MkdirAll(srcDir, 0o755)
	os.WriteFile(filepath.Join(srcDir, "App.tsx"), []byte("UNIQUE_TOKEN_APP_SHOULD_BE_SKIPPED"), 0o644)

	promptSawAppFile := false
	inspectModel := &fakeInspectModel{
		fn: func(params provider.GenerateParams) (*provider.GenerateResult, error) {
			for _, m := range params.Messages {
				for _, part := range m.Content {
					if strings.Contains(part.Text, "UNIQUE_TOKEN_APP_SHOULD_BE_SKIPPED") {
						promptSawAppFile = true
					}
				}
			}
			return &provider.GenerateResult{
				Text:         `{"file": "package.json", "old_str": "root-app", "new_str": "fixed-app", "reasoning": "fixed"}`,
				FinishReason: provider.FinishStop,
			}, nil
		},
	}

	tool := tools.FixCode{
		WorkspaceRoot: ws,
		Model:         inspectModel,
		SessionDir:    sessDir,
	}

	res, err := tool.Execute(context.Background(), &session.Session{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected success, got error: %s", res.Error)
	}
	if promptSawAppFile {
		t.Errorf("expected App.tsx in subdirectory to be skipped when target is in root dir, but it was included in prompt")
	}
}

func TestFixCode_TokenOrRateLimitRetry(t *testing.T) {
	ws := t.TempDir()
	sessDir := filepath.Join(ws, ".atlas", "sessions", "test_sess")
	os.MkdirAll(filepath.Join(sessDir, "logs"), 0o755)

	state.SaveJSON(sessDir, "project.json", map[string]interface{}{"framework": "react"})
	logPath := filepath.Join(sessDir, "logs", "build.log")
	os.WriteFile(logPath, []byte("package.json:2: syntax error"), 0o644)
	state.SaveJSON(sessDir, "build.json", map[string]interface{}{"log_path": logPath})

	os.WriteFile(filepath.Join(ws, "package.json"), []byte(`{"name": "test-app"}`), 0o644)

	calls := 0
	inspectModel := &fakeInspectModel{
		fn: func(params provider.GenerateParams) (*provider.GenerateResult, error) {
			calls++
			if calls == 1 {
				// First call fails with rate limit / token limit
				return nil, fmt.Errorf("Request too large for model `openai/gpt-oss-120b` on tokens per minute (TPM): Limit 8000, Requested 8500")
			}
			// Second retry call succeeds
			return &provider.GenerateResult{
				Text:         `{"file": "package.json", "old_str": "test-app", "new_str": "fixed-app", "reasoning": "fixed"}`,
				FinishReason: provider.FinishStop,
			}, nil
		},
	}

	tool := tools.FixCode{
		WorkspaceRoot: ws,
		Model:         inspectModel,
		SessionDir:    sessDir,
	}

	res, err := tool.Execute(context.Background(), &session.Session{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected success after rate limit retry, got: %s", res.Error)
	}
	if calls != 2 {
		t.Errorf("expected 2 calls (1 failure + 1 minimal retry), got: %d", calls)
	}
}

func TestFixCode_TokenOrRateLimitExhaustion(t *testing.T) {
	ws := t.TempDir()
	sessDir := filepath.Join(ws, ".atlas", "sessions", "test_sess")
	os.MkdirAll(filepath.Join(sessDir, "logs"), 0o755)

	state.SaveJSON(sessDir, "project.json", map[string]interface{}{"framework": "react"})
	logPath := filepath.Join(sessDir, "logs", "build.log")
	os.WriteFile(logPath, []byte("package.json:2: syntax error"), 0o644)
	state.SaveJSON(sessDir, "build.json", map[string]interface{}{"log_path": logPath})

	os.WriteFile(filepath.Join(ws, "package.json"), []byte(`{"name": "test-app"}`), 0o644)

	inspectModel := &fakeInspectModel{
		fn: func(params provider.GenerateParams) (*provider.GenerateResult, error) {
			return nil, fmt.Errorf("Request too large: Limit 8000 TPM")
		},
	}

	tool := tools.FixCode{
		WorkspaceRoot: ws,
		Model:         inspectModel,
		SessionDir:    sessDir,
	}

	res, err := tool.Execute(context.Background(), &session.Session{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Success {
		t.Fatalf("expected failure on rate limit exhaustion")
	}
	if !strings.Contains(res.Error, "token or rate limit exhaustion") {
		t.Errorf("expected error to contain 'token or rate limit exhaustion', got: %s", res.Error)
	}
}
