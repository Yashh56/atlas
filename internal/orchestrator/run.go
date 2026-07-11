package orchestrator

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/Yashh56/atlas/internal/config"
	"github.com/Yashh56/atlas/internal/session"
	"github.com/Yashh56/atlas/internal/tools"
	"github.com/Yashh56/atlas/internal/workspace"
)

// Run executes the Week 2 sequential pipeline:
//  1. Load config
//  2. Resolve workspace
//  3. Create session + four context files
//  4. Run AnalyzeProject
//  5. Save state
//
// It prints progress for each step. Detection failures are soft warnings, not
// hard errors. Build execution is not implemented yet (Week 3).
func Run(ctx context.Context, workspacePath, provider string) error {
	// 1. Load config.
	cfg, err := config.Load(filepath.Join(workspacePath, ".atlas", "config.json"))
	if err != nil {
		return fmt.Errorf("orchestrator: loading config: %w", err)
	}
	_ = cfg // will be used by Week 3 (LLM model selection)
	fmt.Println("✓ Config loaded")

	// 2. Resolve workspace.
	ws, err := workspace.Resolve(workspacePath)
	if err != nil {
		return fmt.Errorf("orchestrator: resolving workspace: %w", err)
	}
	if !ws.Exists {
		return fmt.Errorf("orchestrator: workspace path %q does not exist", workspacePath)
	}
	fmt.Println("✓ Workspace resolved")

	// 3. Create session + context files.
	sessionsDir := filepath.Join(ws.Root, ".atlas", "sessions")
	sess := session.New(ws.Root)
	if err := sess.Save(sessionsDir); err != nil {
		return fmt.Errorf("orchestrator: saving session: %w", err)
	}

	sessDir := session.SessionDir(sessionsDir, sess.ID)

	planner := NewPlanner("deploy")
	if err := SavePlanner(sessDir, planner); err != nil {
		return fmt.Errorf("orchestrator: saving planner: %w", err)
	}

	dep := NewDeployment(provider)
	if err := SaveDeployment(sessDir, dep); err != nil {
		return fmt.Errorf("orchestrator: saving deployment: %w", err)
	}

	// Write an empty project.json; AnalyzeProject will overwrite it.
	if err := SaveProject(sessDir, &ProjectState{}); err != nil {
		return fmt.Errorf("orchestrator: saving initial project state: %w", err)
	}

	fmt.Printf("✓ Session created (%s)\n", sess.ID)

	// 4. Run AnalyzeProject.
	planner.CurrentStep = "analyze_project"
	analyzeTool := tools.AnalyzeProject{
		WorkspaceRoot: ws.Root,
		SessionDir:    sessDir,
	}
	analyzeResult, err := analyzeTool.Execute(ctx, sess)
	if err != nil {
		return fmt.Errorf("orchestrator: analyze_project: %w", err)
	}

	if analyzeResult.Success {
		fmt.Printf("✓ Project analyzed → %s\n", analyzeResult.Output)
		planner.Completed = append(planner.Completed, "analyze_project")
	} else {
		fmt.Printf("⚠ Framework: unknown (%s)\n", analyzeResult.Error)
	}

	// 5. Save final planner state.
	if err := SavePlanner(sessDir, planner); err != nil {
		return fmt.Errorf("orchestrator: saving planner after analyze: %w", err)
	}
	fmt.Println("✓ State saved")
	fmt.Println()
	fmt.Println("Stopping here — build execution not implemented yet (Week 3).")

	return nil
}
