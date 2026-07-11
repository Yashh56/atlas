package orchestrator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Yashh56/atlas/internal/config"
	"github.com/Yashh56/atlas/internal/deploy"
	"github.com/Yashh56/atlas/internal/llm"
	"github.com/Yashh56/atlas/internal/session"
	"github.com/Yashh56/atlas/internal/tools"
	"github.com/Yashh56/atlas/internal/workspace"
)

// RunOptions carries per-invocation flags from the CLI into the pipeline.
type RunOptions struct {
	AllowDirty bool
}

// Run executes the full Atlas pipeline.
func Run(ctx context.Context, workspacePath, providerName string, opts RunOptions) error {
	// 1. Load config.
	cfg, err := config.Load(filepath.Join(workspacePath, ".atlas", "config.json"))
	if err != nil {
		return fmt.Errorf("orchestrator: loading config: %w", err)
	}
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

	// 3. Initialize Provider and LLM Client.
	llmClient, err := llm.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("orchestrator: initializing LLM client: %w", err)
	}

	registry := deploy.NewRegistry()
	registry.Register(&deploy.VercelProvider{})
	
	provider, ok := registry.Get(providerName)
	if !ok {
		return fmt.Errorf("orchestrator: provider %q not supported yet", providerName)
	}

	// 4. Create session + context files.
	sessionsDir := filepath.Join(ws.Root, ".atlas", "sessions")
	sess := session.New(ws.Root)
	if err := sess.Save(sessionsDir); err != nil {
		return fmt.Errorf("orchestrator: saving session: %w", err)
	}

	sessDir := session.SessionDir(sessionsDir, sess.ID)
	planner := NewPlanner("deploy")
	_ = SavePlanner(sessDir, planner)
	
	dep := NewDeployment(providerName)
	_ = SaveDeployment(sessDir, dep)
	_ = SaveProject(sessDir, &ProjectState{})

	fmt.Printf("✓ Session created (%s)\n", sess.ID)

	// 5. Analyze project.
	planner.CurrentStep = "analyze_project"
	analyzeTool := tools.AnalyzeProject{WorkspaceRoot: ws.Root, SessionDir: sessDir}
	analyzeResult, err := analyzeTool.Execute(ctx, sess)
	if err != nil {
		return fmt.Errorf("orchestrator: analyze_project: %w", err)
	}
	if analyzeResult.Success {
		fmt.Printf("✓ Project analyzed → %s\n", analyzeResult.Output)
		planner.Completed = append(planner.Completed, "analyze_project")
	}

	// 6. Git validate.
	planner.CurrentStep = "git_validate"
	gitTool := tools.GitValidate{WorkspaceRoot: ws.Root, GitRoot: ws.GitRoot, SessionDir: sessDir}
	gitResult, err := gitTool.Execute(ctx, sess)
	if err != nil {
		return fmt.Errorf("orchestrator: git_validate: %w", err)
	}

	if gitResult.Success && gitResult.Output != "no_git_repo" {
		fmt.Printf("✓ Git validated → %s\n", gitResult.Output)
		planner.Completed = append(planner.Completed, "git_validate")

		if gitResult.Output == "is_clean:false" && !opts.AllowDirty {
			_ = SavePlanner(sessDir, planner)
			return fmt.Errorf("✗ Working tree has uncommitted changes. Commit or stash them, or re-run with --allow-dirty")
		}
	} else if gitResult.Success && gitResult.Output == "no_git_repo" {
		fmt.Println("⚠ Git: workspace is not inside a git repository — skipping git validation")
	}

	// Read git commit sha for rollback.
	var commitSHA *string
	proj, _ := LoadProject(sessDir)
	if proj != nil && proj.Git.CommitSHA != nil {
		commitSHA = proj.Git.CommitSHA
	}

	framework := ""
	if proj != nil && proj.Framework != nil {
		framework = *proj.Framework
	}
	packageManager := ""
	if proj != nil && proj.PackageManager != nil {
		packageManager = *proj.PackageManager
	}

	// 7. Fix & Rebuild Loop.
	buildSuccess := false
	for {
		planner.CurrentStep = "run_build_command"
		_ = SavePlanner(sessDir, planner)

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
		durationSec := 0.0
		if bs != nil {
			durationSec = float64(bs.DurationMs) / 1000.0
		}

		if buildResult.Success {
			buildSuccess = true
			planner.Completed = append(planner.Completed, "run_build_command")
			_ = SavePlanner(sessDir, planner)
			
			retry := planner.Retries["fix_and_rebuild"]
			if retry.Count > 0 {
				fmt.Printf("✓ Build succeeded after %d fix attempt(s) (%.1fs)\n", retry.Count, durationSec)
			} else {
				fmt.Printf("✓ Build succeeded (%.1fs) → %s\n", durationSec, buildResult.Output)
			}
			break
		}

		// Build failed.
		retry := planner.Retries["fix_and_rebuild"]
		retry.Count++
		planner.Retries["fix_and_rebuild"] = retry
		planner.Failed = append(planner.Failed, "run_build_command")
		
		errorMsg := buildResult.Error
		if bs != nil && bs.LogPath != "" {
			if tail, tailErr := tools.ReadTailLines(bs.LogPath, 20); tailErr == nil && tail != "" {
				errorMsg = tail
			}
		}
		step := "run_build_command"
		now := time.Now().UTC()
		planner.Error = ErrorInfo{Step: &step, Message: &errorMsg, OccurredAt: &now}
		_ = SavePlanner(sessDir, planner)

		fmt.Printf("✗ Build failed (attempt %d/%d, %.1fs) → %s\n", retry.Count, retry.Max, durationSec, buildResult.Output)

		if commitSHA == nil {
			fmt.Println("No git repository detected. Skipping automatic fix to avoid unsafe writes.")
			break
		}

		if retry.Count > retry.Max {
			fmt.Println("→ Exhausted retries. Escalating and reverting changes...")
			// Revert
			revertCmd := tools.RunCommand{
				Command: "git",
				Args:    []string{"checkout", *commitSHA, "--", "."},
				Dir:     ws.Root,
			}
			revertCmd.Execute(ctx, nil)
			planner.CurrentStep = "escalated"
			_ = SavePlanner(sessDir, planner)
			return fmt.Errorf("build failed and exhausted fix attempts. reverted to commit %s", *commitSHA)
		}

		fmt.Println("→ Attempting automatic fix...")
		planner.CurrentStep = "fix_code"
		_ = SavePlanner(sessDir, planner)
		
		fixTool := tools.FixCode{
			WorkspaceRoot: ws.Root,
			Client:        llmClient,
			SessionDir:    sessDir,
		}
		fixRes, err := fixTool.Execute(ctx, sess)
		if err != nil {
			return fmt.Errorf("orchestrator: fix_code: %w", err)
		}

		if fixRes.Success {
			fmt.Printf("  %s\n", fixRes.Output)
			planner.Completed = append(planner.Completed, "fix_code")
		} else {
			fmt.Printf("  Fix attempt failed: %s\n", fixRes.Output)
			planner.Failed = append(planner.Failed, "fix_code")
		}
		_ = SavePlanner(sessDir, planner)
		fmt.Println("→ Rebuilding...")
	}

	if !buildSuccess {
		return fmt.Errorf("deployment aborted due to build failure")
	}

	// 8. Approval Gate.
	approved := CheckApproval(cfg, dep, os.Stdin, os.Stdout)
	if !approved {
		fmt.Println("\nDeployment cancelled by user.")
		return nil
	}

	// 9. Deploy.
	fmt.Printf("→ Deploying to %s (%s)...\n", providerName, dep.Environment)
	planner.CurrentStep = "deploy"
	_ = SavePlanner(sessDir, planner)

	deployRes, err := provider.Deploy(ctx, deploy.DeployInput{
		WorkspaceRoot: ws.Root,
		Environment:   dep.Environment,
	})
	if err != nil {
		planner.Failed = append(planner.Failed, "deploy")
		_ = SavePlanner(sessDir, planner)
		return fmt.Errorf("deployment failed: %w", err)
	}

	RecordDeployment(dep, deployRes.URL)
	_ = SaveDeployment(sessDir, dep)

	planner.Completed = append(planner.Completed, "deploy")
	planner.CurrentStep = "done"
	_ = SavePlanner(sessDir, planner)

	fmt.Printf("\n✓ Deployed successfully to %s\n", deployRes.URL)
	return nil
}
