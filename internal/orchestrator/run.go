package orchestrator

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/Yashh56/atlas/internal/config"
	"github.com/Yashh56/atlas/internal/session"
	"github.com/Yashh56/atlas/internal/tools"
	"github.com/Yashh56/atlas/internal/workspace"
)

// RunOptions carries per-invocation flags from the CLI into the pipeline.
type RunOptions struct {
	AllowDirty bool // skip the dirty-tree abort check
}

// Run executes the Week 3 sequential pipeline:
//  1. Load config
//  2. Resolve workspace
//  3. Create session + four context files
//  4. AnalyzeProject
//  5. GitValidate (dirty-tree abort if !AllowDirty)
//  6. RunBuildCommand
//
// Progress is printed to stdout at each step.
func Run(ctx context.Context, workspacePath, provider string, opts RunOptions) error {
	// 1. Load config (missing file → sane defaults, not an error).
	cfg, err := config.Load(filepath.Join(workspacePath, ".atlas", "config.json"))
	if err != nil {
		return fmt.Errorf("orchestrator: loading config: %w", err)
	}
	_ = cfg
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

	if err := SaveProject(sessDir, &ProjectState{}); err != nil {
		return fmt.Errorf("orchestrator: saving initial project state: %w", err)
	}

	fmt.Printf("✓ Session created (%s)\n", sess.ID)

	// 4. Analyze project.
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

	// 5. Git validate.
	planner.CurrentStep = "git_validate"
	gitTool := tools.GitValidate{
		WorkspaceRoot: ws.Root,
		GitRoot:       ws.GitRoot,
		SessionDir:    sessDir,
	}
	gitResult, err := gitTool.Execute(ctx, sess)
	if err != nil {
		return fmt.Errorf("orchestrator: git_validate: %w", err)
	}

	if gitResult.Success {
		if gitResult.Output == "no_git_repo" {
			fmt.Println("⚠ Git: workspace is not inside a git repository — skipping git validation")
		} else {
			fmt.Printf("✓ Git validated → %s\n", gitResult.Output)
			planner.Completed = append(planner.Completed, "git_validate")

			// Dirty-tree policy: abort unless --allow-dirty.
			if gitResult.Output == "is_clean:false" && !opts.AllowDirty {
				_ = SavePlanner(sessDir, planner)
				return fmt.Errorf("✗ Working tree has uncommitted changes. Commit or stash them, or re-run with --allow-dirty")
			}
		}
	}

	// 6. Run build.
	// Load current project state to get framework + package manager.
	proj, err := LoadProject(sessDir)
	if err != nil {
		return fmt.Errorf("orchestrator: loading project state: %w", err)
	}

	framework := ""
	if proj.Framework != nil {
		framework = *proj.Framework
	}
	packageManager := ""
	if proj.PackageManager != nil {
		packageManager = *proj.PackageManager
	}

	planner.CurrentStep = "run_build_command"
	if err := SavePlanner(sessDir, planner); err != nil {
		return fmt.Errorf("orchestrator: saving planner before build: %w", err)
	}

	buildTool := tools.RunBuildCommand{
		WorkspaceRoot:  ws.Root,
		Framework:      framework,
		PackageManager: packageManager,
		SessionDir:     sessDir,
	}
	buildResult, err := buildTool.Execute(ctx, sess)
	if err != nil {
		return fmt.Errorf("orchestrator: run_build_command: %w", err)
	}

	bs, _ := LoadBuild(sessDir)

	if buildResult.Success {
		// Success branch.
		planner.CurrentStep = "build_complete"
		planner.Completed = append(planner.Completed, "run_build_command")

		durationSec := 0.0
		if bs != nil {
			durationSec = float64(bs.DurationMs) / 1000.0
		}
		if err := SavePlanner(sessDir, planner); err != nil {
			return fmt.Errorf("orchestrator: saving planner after build: %w", err)
		}

		fmt.Printf("✓ Build succeeded (%.1fs) → %s\n", durationSec, buildResult.Output)
		fmt.Println("Stopping here — deployment not implemented yet (Week 4).")
		return nil
	}

	// Failure branch.
	retry := planner.Retries["fix_and_rebuild"]
	retry.Count++
	planner.Retries["fix_and_rebuild"] = retry
	planner.Failed = append(planner.Failed, "run_build_command")

	// Populate error info with last ~20 lines of build.log.
	errorMsg := buildResult.Error
	if bs != nil && bs.LogPath != "" {
		if tail, tailErr := tools.ReadTailLines(bs.LogPath, 20); tailErr == nil && tail != "" {
			errorMsg = tail
		}
	}
	step := "run_build_command"
	now := time.Now().UTC()
	planner.Error = ErrorInfo{
		Step:       &step,
		Message:    &errorMsg,
		OccurredAt: &now,
	}

	if err := SavePlanner(sessDir, planner); err != nil {
		return fmt.Errorf("orchestrator: saving planner after failed build: %w", err)
	}

	durationSec := 0.0
	if bs != nil {
		durationSec = float64(bs.DurationMs) / 1000.0
	}
	fmt.Printf("✗ Build failed (attempt %d/%d, %.1fs) → %s\n",
		retry.Count, retry.Max, durationSec, buildResult.Output)
	fmt.Println("No automatic fix available yet (Week 4). Exiting.")

	return fmt.Errorf("build failed")
}
