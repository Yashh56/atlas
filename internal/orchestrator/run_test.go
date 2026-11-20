package orchestrator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Yashh56/atlas/internal/session"
	"github.com/Yashh56/atlas/internal/state"
	"github.com/Yashh56/atlas/internal/workspace"
	"github.com/zendev-sh/goai/provider"
)

type mockLLMModel struct {
	fn func(params provider.GenerateParams) (*provider.GenerateResult, error)
}

func (m *mockLLMModel) ModelID() string { return "mock-llm" }

func (m *mockLLMModel) DoGenerate(_ context.Context, params provider.GenerateParams) (*provider.GenerateResult, error) {
	if m.fn != nil {
		return m.fn(params)
	}
	return &provider.GenerateResult{Text: "{}", FinishReason: provider.FinishStop}, nil
}

func (m *mockLLMModel) DoStream(_ context.Context, _ provider.GenerateParams) (*provider.StreamResult, error) {
	return nil, nil
}

func TestExecuteBuildLoop_PreflightMissingNodeModules_FailsOnInstallError(t *testing.T) {
	tempDir := t.TempDir()
	sessDir := t.TempDir()

	_ = os.WriteFile(filepath.Join(tempDir, "package.json"), []byte(`{"name": "test-pkg", "scripts": {"build": "nonexistent_build_cmd_xyz"}}`), 0644)
	ws := &workspace.Workspace{Root: tempDir, Exists: true}
	sess := session.New(tempDir)
	planner := NewPlanner("deploy")
	llm := &mockLLMModel{}

	opts := RunOptions{JSONMode: true}
	err := executeBuildLoop(context.Background(), ws, sess, sessDir, planner, llm, nil, "react", "invalid_pm_binary_xyz", "railway", opts)
	if err == nil {
		t.Fatal("expected error due to invalid package manager install, got nil")
	}
	if !strings.Contains(err.Error(), "failed to install dependencies") {
		t.Errorf("expected 'failed to install dependencies', got: %v", err)
	}
}

func TestExecuteBuildLoop_PreflightNodeModulesExists_SkipsPreflightInstall(t *testing.T) {
	tempDir := t.TempDir()
	sessDir := t.TempDir()

	_ = os.WriteFile(filepath.Join(tempDir, "package.json"), []byte(`{"name": "test-pkg", "scripts": {"build": "nonexistent_build_cmd_xyz"}}`), 0644)
	_ = os.Mkdir(filepath.Join(tempDir, "node_modules"), 0755)

	ws := &workspace.Workspace{Root: tempDir, Exists: true}
	sess := session.New(tempDir)
	planner := NewPlanner("deploy")
	llm := &mockLLMModel{}

	opts := RunOptions{JSONMode: true}
	// invalid_pm_binary_xyz would fail if install was attempted. Because node_modules exists, it skips preflight install.
	err := executeBuildLoop(context.Background(), ws, sess, sessDir, planner, llm, nil, "react", "invalid_pm_binary_xyz", "railway", opts)
	if err == nil {
		t.Fatal("expected build failure, got nil")
	}
	// The error should be from run_build_command failing, NOT from "failed to install dependencies"
	if strings.Contains(err.Error(), "failed to install dependencies") {
		t.Errorf("expected preflight install to be skipped, but it failed on dependency install: %v", err)
	}
}

func TestExecuteBuildLoop_Ceiling_MaxTotalBuildAttempts(t *testing.T) {
	tempDir := t.TempDir()
	sessDir := t.TempDir()

	ws := &workspace.Workspace{Root: tempDir, Exists: true}
	sess := session.New(tempDir)
	planner := NewPlanner("deploy")

	state.SaveJSON(sessDir, "project.json", map[string]interface{}{"framework": "react"})

	dummyFile := filepath.Join(tempDir, "app.js")
	_ = os.WriteFile(dummyFile, []byte("const x = 0;"), 0644)

	callCount := 0
	llm := &mockLLMModel{
		fn: func(params provider.GenerateParams) (*provider.GenerateResult, error) {
			callCount++
			// Return a patch that modifies app.js each time so the build error signature is different
			content := fmt.Sprintf(`{"file": "app.js", "old_str": "const x = %d;", "new_str": "const x = %d;", "reasoning": "fix"}`, callCount-1, callCount)
			return &provider.GenerateResult{
				Text:         content,
				FinishReason: provider.FinishStop,
			}, nil
		},
	}

	commitSHA := "abc1234"
	opts := RunOptions{JSONMode: true}

	err := executeBuildLoop(context.Background(), ws, sess, sessDir, planner, llm, &commitSHA, "react", "npm", "railway", opts)
	if err == nil {
		t.Fatal("expected ceiling error, got nil")
	}
	if !strings.Contains(err.Error(), "maximum total build attempts (5)") {
		t.Errorf("expected error containing 'maximum total build attempts (5)', got: %v", err)
	}
	if planner.CurrentStep != "escalated" {
		t.Errorf("expected planner step 'escalated', got: %s", planner.CurrentStep)
	}
}

func TestExecuteBuildLoop_Ceiling_FixAttemptsExhaustedNoPatch(t *testing.T) {
	tempDir := t.TempDir()
	sessDir := t.TempDir()

	ws := &workspace.Workspace{Root: tempDir, Exists: true}
	sess := session.New(tempDir)
	planner := NewPlanner("deploy")

	state.SaveJSON(sessDir, "project.json", map[string]interface{}{"framework": "react"})

	llm := &mockLLMModel{
		fn: func(params provider.GenerateParams) (*provider.GenerateResult, error) {
			// Fail or return invalid output so patch is never applied
			return &provider.GenerateResult{
				Text:         `{"file": "nonexistent.js", "old_str": "missing", "new_str": "fixed"}`,
				FinishReason: provider.FinishStop,
			}, nil
		},
	}

	commitSHA := "abc1234"
	opts := RunOptions{JSONMode: true}

	err := executeBuildLoop(context.Background(), ws, sess, sessDir, planner, llm, &commitSHA, "react", "npm", "railway", opts)
	if err == nil {
		t.Fatal("expected fix attempts exhausted error, got nil")
	}
	if !strings.Contains(err.Error(), "exhausted fix attempts") {
		t.Errorf("expected error containing 'exhausted fix attempts', got: %v", err)
	}
	if planner.CurrentStep != "escalated" {
		t.Errorf("expected planner step 'escalated', got: %s", planner.CurrentStep)
	}
}
