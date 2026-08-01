package orchestrator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Yashh56/atlas/internal/cliutil"
	"github.com/Yashh56/atlas/internal/config"
	"github.com/Yashh56/atlas/internal/credentials"
	"github.com/Yashh56/atlas/internal/deploy"
	"github.com/Yashh56/atlas/internal/deploy/netlify"
	"github.com/Yashh56/atlas/internal/deploy/render"
	"github.com/Yashh56/atlas/internal/deploy/vercel"
	"github.com/Yashh56/atlas/internal/llm"
	"github.com/Yashh56/atlas/internal/session"
	"github.com/Yashh56/atlas/internal/state"
	"github.com/Yashh56/atlas/internal/tools"
	"github.com/Yashh56/atlas/internal/workspace"
)

var (
	styleCheck = cliutil.IconSuccess
	styleArrow = cliutil.IconArrow
	styleCross = cliutil.IconError
	styleWarn  = cliutil.IconWarning
)

// Action defines the sequence of operations for the pipeline.
type Action string

const (
	ActionBuild         Action = "build"           // analyze → git validate → build, stop
	ActionTest          Action = "test"            // ...→ build → run tests, stop
	ActionDeploy        Action = "deploy"          // ...→ build → fix loop → deploy (today's default)
	ActionTestAndDeploy Action = "test-and-deploy" // ...→ build → fix loop → run tests → deploy
)

// IsDeploy returns true if the pipeline action involves deployment.
func (a Action) IsDeploy() bool {
	return a == ActionDeploy || a == ActionTestAndDeploy || a == "" // default is deploy
}

// RunOptions carries per-invocation flags from the CLI into the pipeline.
type RunOptions struct {
	AllowDirty    bool
	ModelOverride string
	Action        Action
	IsInteractive bool
	OutputDir     string
	AutoRollback  bool
}

// Run executes the full Atlas pipeline by dispatching to modular steps.
func Run(ctx context.Context, workspacePath, providerName string, opts RunOptions) error {
	cfg, ws, sess, sessDir, llmModel, provider, dep, err := executeSetup(ctx, workspacePath, providerName, opts)
	if err != nil {
		return err
	}

	goal := string(opts.Action)
	if goal == "" {
		goal = string(ActionDeploy)
	}
	planner := NewPlanner(goal)
	_ = SavePlanner(sessDir, planner)

	var didStash bool
	defer func() {
		if didStash {
			fmt.Printf("\n%s Restoring stashed uncommitted changes...\n", styleArrow)
			popCmd := tools.RunCommand{
				Command: "git",
				Args:    []string{"stash", "pop"},
				Dir:     workspacePath,
			}
			res, _ := popCmd.Execute(ctx, sess)
			if !res.Success {
				fmt.Printf("%s Failed to pop stash automatically. Please run `git stash pop` manually to resolve any conflicts.\n", styleWarn)
			} else {
				fmt.Printf("%s Stash restored successfully.\n", styleCheck)
			}
		}

		// Print token usage if tracking is enabled/recorded, even if zero.
		if planner != nil {
			fmt.Printf("\n%s Session Token Usage: %d total (%d prompt, %d completion)\n", styleCheck, planner.TokenUsage.TotalTokens, planner.TokenUsage.InputTokens, planner.TokenUsage.OutputTokens)
		}
	}()

	commitSHA, framework, packageManager, didStash, err := executeAnalyzeAndValidate(ctx, ws, sess, sessDir, planner, providerName, opts)
	if err != nil {
		return err
	}

	err = executeBuildLoop(ctx, ws, sess, sessDir, planner, llmModel, commitSHA, framework, packageManager, providerName, opts)
	if err != nil {
		return err
	}

	if opts.Action == ActionBuild {
		fmt.Printf("%s Action 'build' complete. Stopping before tests or deploy.\n", styleCheck)
		planner.CurrentStep = "done"
		_ = SavePlanner(sessDir, planner)
		return nil
	}

	if opts.Action == ActionTest || opts.Action == ActionTestAndDeploy {
		err = executeTests(ctx, ws, sess, sessDir, planner, framework, packageManager, opts)
		if err != nil {
			return err
		}
		if opts.Action == ActionTest {
			fmt.Printf("%s Action 'test' complete. Stopping before deploy.\n", styleCheck)
			planner.CurrentStep = "done"
			_ = SavePlanner(sessDir, planner)
			return nil
		}
	}

	return executeDeploy(ctx, ws, sessDir, planner, cfg, dep, provider, providerName, opts)
}

func ensureAtlasGitignore(wsRoot string) {
	atlasDir := filepath.Join(wsRoot, ".atlas")
	_ = os.MkdirAll(atlasDir, 0755)
	gitignorePath := filepath.Join(atlasDir, ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		_ = os.WriteFile(gitignorePath, []byte("*\n"), 0644)
	}
}

func executeSetup(ctx context.Context, workspacePath, providerName string, opts RunOptions) (
	cfg *config.Config,
	ws *workspace.Workspace,
	sess *session.Session,
	sessDir string,
	llmModel llm.Model,
	provider deploy.Provider,
	dep *DeploymentState,
	err error,
) {
	ensureAtlasGitignore(workspacePath)
	cfg, err = config.Load(filepath.Join(workspacePath, ".atlas", "config.json"))
	if err != nil {
		err = fmt.Errorf("orchestrator: loading config: %w", err)
		return
	}
	fmt.Printf("%s\n", cliutil.FormatSuccess("Config", "loaded"))

	ws, err = workspace.Resolve(workspacePath)
	if err != nil {
		err = fmt.Errorf("orchestrator: resolving workspace: %w", err)
		return
	}
	if !ws.Exists {
		err = fmt.Errorf("orchestrator: workspace path %q does not exist", workspacePath)
		return
	}
	fmt.Printf("%s\n\n", cliutil.FormatSuccess("Workspace", "resolved"))

	store, storeErr := credentials.Open()
	if storeErr != nil {
		fmt.Printf("%s Could not open credential store: %v — continuing without it\n\n", styleWarn, storeErr)
	}

	if opts.ModelOverride != "" {
		cfg.LLMProvider = opts.ModelOverride
	}
	llmModel, err = llm.ResolveModel(cfg, store)
	if err != nil {
		err = fmt.Errorf("orchestrator: resolving LLM model: %w", err)
		return
	}

	if providerName != "" && opts.Action.IsDeploy() {
		registry := deploy.NewRegistry()
		registry.Register(&vercel.VercelProvider{})
		registry.Register(&render.RenderProvider{})
		registry.Register(&netlify.NetlifyProvider{})

		var ok bool
		provider, ok = registry.Get(providerName)
		if !ok {
			err = fmt.Errorf("orchestrator: provider %q not supported yet", providerName)
			return
		}

		switch providerName {
		case "vercel":
			fmt.Printf("%s Checking Vercel authentication...\n", styleArrow)
			if authErr := vercel.EnsureVercelAuth(ctx, store, cfg, deploy.OSCommandRunner{}); authErr != nil {
				err = fmt.Errorf("Vercel auth failed: %w", authErr)
				return
			}
		case "render":
			fmt.Printf("%s Checking Render authentication...\n", styleArrow)
			if authErr := render.EnsureRenderAuth(ctx, store, cfg, deploy.OSCommandRunner{}); authErr != nil {
				err = fmt.Errorf("Render auth failed: %w", authErr)
				return
			}
		case "netlify":
			fmt.Printf("%s Checking Netlify authentication...\n", styleArrow)
			if authErr := netlify.EnsureNetlifyAuth(ctx, store, cfg, deploy.OSCommandRunner{}); authErr != nil {
				err = fmt.Errorf("Netlify auth failed: %w", authErr)
				return
			}
		}
	}

	sessionsDir := filepath.Join(ws.Root, ".atlas", "sessions")
	sess = session.New(ws.Root)
	if err = sess.Save(sessionsDir); err != nil {
		err = fmt.Errorf("orchestrator: saving session: %w", err)
		return
	}

	sessDir = session.SessionDir(sessionsDir, sess.ID)

	if opts.Action.IsDeploy() {
		dep = NewDeployment(providerName)
		_ = SaveDeployment(sessDir, dep)
	}

	var proj ProjectState
	if p, err := LoadProject(filepath.Join(ws.Root, ".atlas")); err == nil {
		proj = *p
	}
	_ = SaveProject(sessDir, &proj)

	fmt.Printf("%s\n\n", cliutil.FormatSuccess("Session", sess.ID))
	return
}

func executeAnalyzeAndValidate(
	ctx context.Context, ws *workspace.Workspace, sess *session.Session,
	sessDir string, planner *PlannerState, providerName string, opts RunOptions,
) (*string, string, string, bool, error) {

	planner.CurrentStep = "analyze_project"

	spinner := cliutil.StartSpinner("Analyzing project...")

	analyzeTool := tools.AnalyzeProject{WorkspaceRoot: ws.Root, SessionDir: sessDir}
	analyzeResult, err := analyzeTool.Execute(ctx, sess)

	spinner.Stop()

	if err != nil {
		return nil, "", "", false, fmt.Errorf("orchestrator: analyze_project: %w", err)
	}
	if analyzeResult.Success {
		fmt.Printf("%s\n", cliutil.FormatSuccess("Analysis", analyzeResult.Output))
		planner.Completed = append(planner.Completed, "analyze_project")
	}

	planner.CurrentStep = "git_validate"

	spinner = cliutil.StartSpinner("Validating git repository...")

	gitVal := tools.GitValidate{WorkspaceRoot: ws.Root, GitRoot: ws.GitRoot, SessionDir: sessDir, Provider: providerName}
	gitResult, err := gitVal.Execute(ctx, sess)

	spinner.Stop()

	if err != nil {
		return nil, "", "", false, fmt.Errorf("orchestrator: git_validate: %w", err)
	}

	if gitResult.Success && gitResult.Output != "no_git_repo" {
		fmt.Printf("%s\n", cliutil.FormatSuccess("Git", gitResult.Output))
		if proj, err := LoadProject(sessDir); err == nil && proj != nil {
			if proj.Git.Branch != nil && *proj.Git.Branch != "" {
				fmt.Printf("%s\n", cliutil.FormatSuccess("Branch", *proj.Git.Branch))
			}
			if proj.Git.CommitSHA != nil && *proj.Git.CommitSHA != "" {
				sha := *proj.Git.CommitSHA
				if len(sha) > 7 {
					sha = sha[:7]
				}
				fmt.Printf("%s\n", cliutil.FormatSuccess("Commit", sha))
			}
			if proj.Git.Remote != nil && *proj.Git.Remote != "" {
				fmt.Printf("%s\n", cliutil.FormatSuccess("Remote", *proj.Git.Remote))
			}
		}
		fmt.Println()
		planner.Completed = append(planner.Completed, "git_validate")

		if gitResult.Output == "is_clean:false" && !opts.AllowDirty {
			if opts.Action.IsDeploy() {
				if !opts.IsInteractive {
					_ = SavePlanner(sessDir, planner)
					return nil, "", "", false, fmt.Errorf("Working tree has uncommitted changes. Commit or stash them, or re-run with --allow-dirty")
				}

				choice := PromptDirtyDeploy(providerName, os.Stdin, os.Stdout)
				switch choice {
				case 1:
					fmt.Printf("%s Committing fixes...\n", styleArrow)
					commitCmd := tools.RunCommand{
						Command: "git",
						Args:    []string{"commit", "-am", "chore: auto-commit before deploy"},
						Dir:     ws.Root,
					}
					commitCmd.Execute(ctx, sess)

					fmt.Printf("%s Pushing fixes to remote...\n", styleArrow)
					pushCmd := tools.RunCommand{
						Command: "git",
						Args:    []string{"push"},
						Dir:     ws.Root,
					}
					pushRes, pushErr := pushCmd.Execute(ctx, sess)
					if pushErr != nil || !pushRes.Success {
						return nil, "", "", false, fmt.Errorf("failed to push changes to remote: %s", pushRes.Error)
					}
					fmt.Printf("%s Fixes committed and pushed successfully!\n", styleCheck)

					// Re-validate to get the new commit SHA
					gitVal.Execute(ctx, sess)
					if proj, err := LoadProject(sessDir); err == nil && proj != nil && proj.Git.CommitSHA != nil && *proj.Git.CommitSHA != "" {
						sha := *proj.Git.CommitSHA
						if len(sha) > 7 {
							sha = sha[:7]
						}
						fmt.Printf("%s\n\n", cliutil.FormatSuccess("New Commit", sha))
					} else {
						fmt.Println()
					}
				case 2:
					if strings.ToLower(providerName) != "vercel" {
						fmt.Printf("%s Stashing uncommitted changes...\n", styleArrow)
						stashCmd := tools.RunCommand{
							Command: "git",
							Args:    []string{"stash", "push", "-u", "-m", "atlas pre-deploy stash"},
							Dir:     ws.Root,
						}
						stashCmd.Execute(ctx, sess)

						// Load project state so we return the framework/package manager
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

						return commitSHA, framework, packageManager, true, nil
					}
				case 3:
					return nil, "", "", false, fmt.Errorf("deployment cancelled due to uncommitted changes")
				}
			}
		}
	} else if gitResult.Success && gitResult.Output == "no_git_repo" {
		fmt.Printf("%s Git: workspace is not inside a git repository — skipping git validation\n\n", styleWarn)
	}

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

	return commitSHA, framework, packageManager, false, nil
}

func executeBuildLoop(
	ctx context.Context, ws *workspace.Workspace, sess *session.Session, sessDir string,
	planner *PlannerState, llmModel llm.Model, commitSHA *string, framework, packageManager, providerName string, opts RunOptions,
) error {
	buildSuccess := false
	var lastErrorMsg string
	stuckCount := 0

	for {
		planner.CurrentStep = "run_build_command"
		_ = SavePlanner(sessDir, planner)

		spinner := cliutil.StartSpinner("Running build command...")

		buildTool := tools.RunBuildCommand{
			WorkspaceRoot:  ws.Root,
			Framework:      framework,
			PackageManager: packageManager,
			SessionDir:     sessDir,
		}
		buildResult, err := buildTool.Execute(ctx, sess)

		spinner.Stop()

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

			// Clean up backups on success
			os.RemoveAll(filepath.Join(sessDir, "backups"))

			retry := planner.Retries["fix_and_rebuild"]
			if retry.Count > 0 {
				fmt.Printf("%s Build succeeded after %d fix attempt(s) (%.1fs)\n\n", cliutil.IconSuccess, retry.Count, durationSec)

				// Show colorized diff of what was fixed
				diffCmd := tools.RunCommand{
					Command: "git",
					Args:    []string{"diff", "--color=always"},
					Dir:     ws.Root,
				}
				if diffRes, err := diffCmd.Execute(ctx, sess); err == nil && diffRes.Output != "" {
					fmt.Println(diffRes.Output)
				}

				if CheckCommitApproval(os.Stdin, os.Stdout) {
					// Commit the changes
					fmt.Printf("%s Committing fixes...\n", styleArrow)
					commitCmd := tools.RunCommand{
						Command: "git",
						Args:    []string{"commit", "-am", "chore: auto-fix build errors"},
						Dir:     ws.Root,
					}
					commitCmd.Execute(ctx, sess)

					// Push the changes
					fmt.Printf("%s Pushing fixes to remote...\n", styleArrow)
					pushCmd := tools.RunCommand{
						Command: "git",
						Args:    []string{"push"},
						Dir:     ws.Root,
					}
					pushCmd.Execute(ctx, sess)
					fmt.Printf("%s Fixes committed and pushed successfully!\n\n", styleCheck)

					// Re-run git validation to update project.json with the NEW commit SHA
					// so that the deploy provider deploys the fixed code instead of the old broken commit.
					gitVal := tools.GitValidate{
						WorkspaceRoot: ws.Root,
						GitRoot:       ws.GitRoot,
						SessionDir:    sessDir,
						Provider:      providerName,
					}
					gitVal.Execute(ctx, sess)
				} else {
					if providerName == "render" {
						fmt.Printf("%s Warning: You chose not to push the fixes. Remote providers like Render will deploy the old broken commit.\n\n", styleWarn)
					} else {
						fmt.Printf("%s Warning: You chose not to commit the fixes. They will remain in your working directory.\n\n", styleWarn)
					}
				}
			} else {
				fmt.Printf("%s Build succeeded (%.1fs) → %s\n\n", styleCheck, durationSec, buildResult.Output)
			}
			break
		}

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

		if errorMsg != "" && errorMsg == lastErrorMsg {
			stuckCount++
		} else {
			stuckCount = 0
		}
		lastErrorMsg = errorMsg

		if stuckCount >= 2 {
			fmt.Printf("%s Stuck loop detected (same error %d times in a row). Escalating...\n", styleWarn, stuckCount+1)
			retry.Count = retry.Max + 1 // Force exhaustion
		}

		step := "run_build_command"
		now := time.Now().UTC()
		planner.Error = ErrorInfo{Step: &step, Message: &errorMsg, OccurredAt: &now}
		_ = SavePlanner(sessDir, planner)

		fmt.Printf("%s Build failed (attempt %d/%d, %.1fs) → %s\n", styleCross, retry.Count, retry.Max, durationSec, buildResult.Output)

		if commitSHA == nil {
			fmt.Printf("%s No git repository detected. Skipping automatic fix to avoid unsafe writes.\n", styleWarn)
			break
		}

		if retry.Count > retry.Max {
			fmt.Printf("%s Exhausted retries. Escalating and reverting changes...\n", styleArrow)

			// Restore from backups instead of git checkout
			backupDir := filepath.Join(sessDir, "backups")
			if _, err := os.Stat(backupDir); err == nil {
				filepath.Walk(backupDir, func(path string, info os.FileInfo, err error) error {
					if err != nil || info.IsDir() {
						return nil
					}
					rel, _ := filepath.Rel(backupDir, path)
					targetPath := filepath.Join(ws.Root, rel)
					bytes, err := os.ReadFile(path)
					if err == nil {
						_ = os.WriteFile(targetPath, bytes, info.Mode())
					}
					return nil
				})
			}

			planner.CurrentStep = "escalated"
			_ = SavePlanner(sessDir, planner)
			return fmt.Errorf("build failed and exhausted fix attempts. reverted using file snapshots")
		}

		fmt.Printf("%s Attempting automatic fix...\n", styleArrow)
		planner.CurrentStep = "fix_code"
		_ = SavePlanner(sessDir, planner)

		fixTool := tools.FixCode{
			WorkspaceRoot: ws.Root,
			Model:         llmModel,
			SessionDir:    sessDir,
		}
		fixRes, err := fixTool.Execute(ctx, sess)
		if err != nil {
			return fmt.Errorf("orchestrator: fix_code: %w", err)
		}

		planner.AddUsage(fixRes.TokenUsage)

		if fixRes.Success {
			fmt.Printf("  %s\n\n", fixRes.Output)
			planner.Completed = append(planner.Completed, "fix_code")
		} else {
			fmt.Printf("  Fix attempt failed: %s\n\n", fixRes.Error)
			planner.Failed = append(planner.Failed, "fix_code")
		}
		_ = SavePlanner(sessDir, planner)
		fmt.Printf("%s Rebuilding...\n", styleArrow)
	}

	if !buildSuccess {
		return fmt.Errorf("action %q aborted due to build failure", opts.Action)
	}
	return nil
}

func executeTests(
	ctx context.Context, ws *workspace.Workspace, sess *session.Session,
	sessDir string, planner *PlannerState, framework, packageManager string, opts RunOptions,
) error {
	planner.CurrentStep = "run_tests"
	_ = SavePlanner(sessDir, planner)

	fmt.Printf("%s Running tests...\n", styleArrow)
	testTool := tools.RunTests{
		WorkspaceRoot:  ws.Root,
		Framework:      framework,
		PackageManager: packageManager,
		SessionDir:     sessDir,
	}
	testResult, testErr := testTool.Execute(ctx, sess)
	if testErr != nil {
		return fmt.Errorf("orchestrator: run_tests: %w", testErr)
	}

	var ts struct {
		DurationMs int64 `json:"duration_ms"`
	}
	_ = state.LoadJSON(sessDir, "test.json", &ts)
	durationSec := float64(ts.DurationMs) / 1000.0

	if testResult.Success {
		fmt.Printf("%s Tests passed (%.1fs) → %s\n\n", styleCheck, durationSec, testResult.Output)
		planner.Completed = append(planner.Completed, "run_tests")
		_ = SavePlanner(sessDir, planner)
	} else {
		fmt.Printf("%s Tests failed (%.1fs) → %s\n", styleCross, durationSec, testResult.Output)
		planner.Failed = append(planner.Failed, "run_tests")
		_ = SavePlanner(sessDir, planner)
		return fmt.Errorf("action %q aborted due to test failure", opts.Action)
	}
	return nil
}

func executeDeploy(
	ctx context.Context, ws *workspace.Workspace, sessDir string, planner *PlannerState,
	cfg *config.Config, dep *DeploymentState, provider deploy.Provider, providerName string, opts RunOptions,
) error {
	approved := CheckApproval(cfg, dep, os.Stdin, os.Stdout)
	if !approved {
		fmt.Println("\nDeployment cancelled by user.")
		return nil
	}

	fmt.Printf("%s Deploying to %s (%s)...\n", styleArrow, providerName, dep.Environment)
	planner.CurrentStep = "deploy"
	_ = SavePlanner(sessDir, planner)

	store, _ := credentials.Open()
	var token string
	if store != nil {
		token, _ = store.GetSecret(providerName)
	}
	if token == "" {
		token = os.Getenv(strings.ToUpper(providerName) + "_TOKEN")
	}

	deployInput := deploy.DeployInput{
		WorkspaceRoot: ws.Root,
		SessionDir:    sessDir,
		Environment:   dep.Environment,
		Token:         token,
	}

	deployRes, err := provider.Deploy(ctx, deployInput)
	if err != nil {
		planner.Failed = append(planner.Failed, "deploy")
		_ = SavePlanner(sessDir, planner)
		return fmt.Errorf("deployment failed: %w", err)
	}

	RecordDeployment(dep, deployRes.URL)
	_ = SaveDeployment(sessDir, dep)

	fmt.Printf("\n%s\n\n", cliutil.FormatBox(fmt.Sprintf("%s Deployed successfully to %s", cliutil.IconSuccess, cliutil.StyleHighlight.Render(deployRes.URL))))

	fmt.Printf("%s Running post-deployment health check...\n", styleArrow)
	healthErr := provider.HealthCheck(ctx, deployRes)

	if healthErr == nil {
		fmt.Printf("%s Health check passed!\n", styleCheck)
		dep.LastHealthyDeployment = deployRes
		_ = SaveDeployment(sessDir, dep)
	} else {
		fmt.Printf("%s Health check failed: %v\n", styleCross, healthErr)

		if dep.LastHealthyDeployment == nil {
			planner.Failed = append(planner.Failed, "deploy")
			_ = SavePlanner(sessDir, planner)
			return fmt.Errorf("health check failed and no prior healthy deployment exists to rollback to")
		}

		doRollback := opts.AutoRollback
		if !doRollback && opts.IsInteractive {
			// Prompt for rollback
			fmt.Printf("\n%s A previous healthy deployment exists. Do you want to rollback to %s? [y/N]: ", styleWarn, dep.LastHealthyDeployment.URL)
			var resp string
			fmt.Scanln(&resp)
			if strings.ToLower(strings.TrimSpace(resp)) == "y" {
				doRollback = true
			}
		}

		if doRollback {
			fmt.Printf("%s Rolling back to previous healthy deployment...\n", styleArrow)
			rbErr := provider.Rollback(ctx, dep.LastHealthyDeployment, deployInput)
			if rbErr != nil {
				return fmt.Errorf("rollback failed: %w", rbErr)
			}

			fmt.Printf("%s Rollback executed. Verifying health...\n", styleArrow)
			rbHealthErr := provider.HealthCheck(ctx, dep.LastHealthyDeployment)
			if rbHealthErr != nil {
				return fmt.Errorf("rolled back deployment failed health check: %w", rbHealthErr)
			}

			fmt.Printf("%s Successfully rolled back to %s\n", styleCheck, dep.LastHealthyDeployment.URL)

			// Leave dep.LastHealthyDeployment as is, because it's still the last healthy one
		} else {
			return fmt.Errorf("deployment is unhealthy and rollback was declined")
		}
	}

	planner.Completed = append(planner.Completed, "deploy")
	planner.CurrentStep = "done"
	_ = SavePlanner(sessDir, planner)

	return nil
}
