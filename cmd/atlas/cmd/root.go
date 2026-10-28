// Package cmd wires together all Atlas CLI commands.
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/Yashh56/atlas/internal/cliutil"
	"github.com/Yashh56/atlas/internal/orchestrator"
	"github.com/Yashh56/atlas/internal/updater"
	"github.com/Yashh56/atlas/internal/version"
	"github.com/Yashh56/atlas/internal/workspace"
)

var (
	pathFlag           string
	dirFlag            string
	workspaceFlag      string
	strictEnvFlag      bool
	deployModel        string
	deployAction       string
	deployProvider     string
	deployAllowDirty   bool
	outputDirFlag      string
	autoRollbackFlag   bool
	nonInteractiveFlag bool
	jsonFlag           bool
)

// supportedProviders contains deployment providers supported in v0.2.0.
var supportedProviders = map[string]bool{
	"vercel":  true,
	"render":  true,
	"netlify": true,
}

// plannedProviders are roadmapped providers scheduled for v0.3.0.
var plannedProviders = map[string]bool{
	"fly":     true,
	"railway": true,
}

var rootCmd = &cobra.Command{
	Use:          "atlas [path]",
	Short:        "Atlas — autonomous deployment agent",
	Long:         "Atlas is a CLI tool that autonomously deploys your projects.",
	Args:         cobra.MaximumNArgs(1),
	SilenceUsage: true,
	RunE:         runPipeline,
	Version:      version.Version,
}

var latestRelease *updater.Release

// Execute is the entry point called from main.
func Execute() error {
	isUpdateCmd := len(os.Args) > 1 && os.Args[1] == "update"

	if !isUpdateCmd {
		go func() {
			rel, _ := updater.CheckUpdate(version.Version)
			if rel != nil {
				latestRelease = rel
			}
		}()
	}

	err := rootCmd.Execute()

	if latestRelease != nil && !isUpdateCmd && !jsonFlag {
		fmt.Printf("\n%s A new version of Atlas is available: %s -> %s\n", cliutil.StyleHighlight.Render("UPDATE:"), version.Version, latestRelease.TagName)
		fmt.Printf("Run %s to update.\n\n", cliutil.StylePrompt.Render("atlas update"))
	}

	return err
}

func init() {
	rootCmd.Flags().StringVar(&pathFlag, "path", "", "project path (alternative to positional arg, for paths that collide with a subcommand name)")
	rootCmd.Flags().StringVar(&dirFlag, "dir", "", "target subdirectory within a monorepo (e.g. apps/web)")
	rootCmd.Flags().StringVar(&workspaceFlag, "workspace", "", "target package name within a monorepo (e.g. web or @myorg/web)")
	rootCmd.Flags().BoolVar(&strictEnvFlag, "strict-env", false, "abort deployment if any required environment variables are missing from local environment or .env files")
	rootCmd.Flags().StringVar(&deployModel, "model", "", "LLM provider to use (anthropic, openai, etc.)")
	rootCmd.Flags().StringVar(&deployAction, "action", "", "action mode: build, test, deploy, test-and-deploy")
	rootCmd.Flags().StringVar(&deployProvider, "provider", "", "deployment provider (vercel, render, netlify)")
	rootCmd.Flags().BoolVar(&deployAllowDirty, "allow-dirty", false, "skip the dirty working-tree check and proceed even with uncommitted changes")
	rootCmd.Flags().StringVar(&outputDirFlag, "output-dir", "", "manual output directory override (e.g. build, dist, out)")
	rootCmd.Flags().BoolVar(&autoRollbackFlag, "auto-rollback-on-unhealthy", false, "automatically rollback without prompting if post-deploy health check fails")
	rootCmd.Flags().BoolVarP(&nonInteractiveFlag, "yes", "y", false, "auto-approve deployment and run non-interactively without prompting")
	rootCmd.Flags().BoolVar(&nonInteractiveFlag, "non-interactive", false, "run non-interactively without prompting (alias for -y/--yes)")

	rootCmd.PersistentFlags().BoolVar(&jsonFlag, "json", false, "output in machine-readable JSON format")

	rootCmd.AddCommand(debugCmd)
	rootCmd.AddCommand(testllmCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(modelsCmd)
	rootCmd.AddCommand(providersCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(auditCmd)

	if vf := rootCmd.Flags().Lookup("version"); vf != nil {
		vf.Shorthand = "v"
	}

	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		fmt.Printf("\n%s\n", cliutil.FormatHeader(fmt.Sprintf("%s - %s", cmd.Name(), cmd.Short)))
		if cmd.Long != "" {
			fmt.Printf("\n%s\n", cmd.Long)
		}
		fmt.Printf("\n%s\n", cliutil.StyleHighlight.Render("USAGE"))
		if cmd.Runnable() {
			fmt.Printf("  %s\n", cmd.UseLine())
		}
		if cmd.HasAvailableSubCommands() {
			fmt.Printf("  %s [command]\n", cmd.CommandPath())
			fmt.Printf("\n%s\n", cliutil.StyleHighlight.Render("AVAILABLE COMMANDS"))
			for _, c := range cmd.Commands() {
				if c.IsAvailableCommand() {
					fmt.Printf("  %-15s %s\n", cliutil.StylePrompt.Render(c.Name()), cliutil.StyleSubtext.Render(c.Short))
				}
			}
		}
		if cmd.HasAvailableLocalFlags() {
			fmt.Printf("\n%s\n", cliutil.StyleHighlight.Render("FLAGS"))
			fmt.Printf("%s\n", cliutil.StyleSubtext.Render(strings.TrimRight(cmd.LocalFlags().FlagUsages(), "\n")))
		}
		fmt.Println()
	})
}

func actionNeedsProvider(a orchestrator.Action) bool {
	return a.IsDeploy()
}

func runPipeline(cmd *cobra.Command, args []string) error {
	var path string

	if len(args) == 0 && cmd.Flags().NFlag() == 0 {
		if !jsonFlag {
			cliutil.PrintWelcome()
		}
		return nil
	}

	if len(args) > 0 {
		path = args[0]
	} else if pathFlag != "" {
		path = pathFlag
	}

	isInteractive := term.IsTerminal(int(os.Stdin.Fd()))
	if nonInteractiveFlag || jsonFlag {
		isInteractive = false
	}

	if path == "" {
		return fmt.Errorf("no project path given. Please provide a path as an argument or via --path")
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("path %q does not exist", path)
		}
		return fmt.Errorf("error checking path %q: %w", path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path %q is not a directory", path)
	}

	ws, err := workspace.ResolveWithSubdir(path, dirFlag, workspaceFlag)
	if err != nil {
		return err
	}

	if dirFlag == "" && workspaceFlag == "" && ws.Monorepo != nil && ws.Root == ws.MonorepoRoot && len(ws.Monorepo.Packages) > 1 && !hasRootAppFramework(ws.Root, true) {
		if !isInteractive {
			names := make([]string, len(ws.Monorepo.Packages))
			for i, p := range ws.Monorepo.Packages {
				names[i] = p.Name
			}
			return fmt.Errorf("monorepo detected at %s with multiple packages (%s). Please specify target package with --workspace <name> or --dir <path>", ws.Root, strings.Join(names, ", "))
		}

		var selectedPkg string
		options := make([]huh.Option[string], len(ws.Monorepo.Packages))
		for i, p := range ws.Monorepo.Packages {
			label := fmt.Sprintf("%s (%s)", p.Name, p.RelPath)
			options[i] = huh.NewOption(label, p.Name)
		}
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Multiple packages detected. Select package to deploy:").
					Options(options...).
					Value(&selectedPkg),
			),
		)
		if err := form.Run(); err != nil {
			return fmt.Errorf("package selection aborted: %w", err)
		}
		pkg, err := workspace.ResolveWorkspacePackage(ws.Monorepo.Root, selectedPkg)
		if err != nil {
			return err
		}
		ws.Root = pkg.AbsPath
	}

	path = ws.Root

	envFiles := []string{".env.production.local", ".env.local", ".env.production", ".env"}
	for _, f := range envFiles {
		_ = godotenv.Load(filepath.Join(ws.Root, f))
	}
	if ws.MonorepoRoot != "" && ws.MonorepoRoot != ws.Root {
		for _, f := range envFiles {
			_ = godotenv.Load(filepath.Join(ws.MonorepoRoot, f))
		}
	}

	needsWizard := isInteractive && (deployModel == "" || deployAction == "" || (actionNeedsProvider(orchestrator.Action(deployAction)) && deployProvider == ""))

	if needsWizard {
		m, a, p, err := RunWizard(deployModel, orchestrator.Action(deployAction), deployProvider)
		if err != nil {
			return fmt.Errorf("wizard failed or aborted: %w", err)
		}
		deployModel = m
		deployAction = string(a)
		deployProvider = p
	} else if !isInteractive {
		if deployModel == "" {
			return fmt.Errorf("--model flag is required when running non-interactively")
		}
		if deployAction == "" {
			return fmt.Errorf("--action flag is required when running non-interactively")
		}
		if actionNeedsProvider(orchestrator.Action(deployAction)) && deployProvider == "" {
			return fmt.Errorf("--provider flag is required for action %q when running non-interactively", deployAction)
		}
	}

	if deployModel == "" {
		return fmt.Errorf("no model resolved")
	}
	if deployAction == "" {
		return fmt.Errorf("no action resolved")
	}
	if actionNeedsProvider(orchestrator.Action(deployAction)) && deployProvider == "" {
		return fmt.Errorf("no provider resolved")
	}

	if deployProvider != "" {
		pLower := strings.ToLower(deployProvider)
		if plannedProviders[pLower] {
			return fmt.Errorf("provider %q is planned for v0.3.0 and not yet supported. Supported providers: netlify, render, vercel", deployProvider)
		}
		if !supportedProviders[pLower] {
			return fmt.Errorf(
				"unknown provider %q — supported providers: netlify, render, vercel",
				deployProvider,
			)
		}
	}

	opts := orchestrator.RunOptions{
		AllowDirty:     deployAllowDirty,
		ModelOverride:  deployModel,
		Action:         orchestrator.Action(deployAction),
		IsInteractive:  isInteractive,
		NonInteractive: nonInteractiveFlag || jsonFlag,
		OutputDir:      outputDirFlag,
		AutoRollback:   autoRollbackFlag,
		JSONMode:       jsonFlag,
		StrictEnv:      strictEnvFlag,
	}
	return orchestrator.Run(cmd.Context(), path, strings.ToLower(deployProvider), opts)
}

func hasRootAppFramework(root string, isMonorepoRoot bool) bool {
	prefixes := []string{
		"next.config.",
		"vite.config.",
		"nuxt.config.",
		"svelte.config.",
		"astro.config.",
		"remix.config.",
	}
	if entries, err := os.ReadDir(root); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			for _, prefix := range prefixes {
				if strings.HasPrefix(name, prefix) {
					return true
				}
			}
		}
	}

	nonJSFiles := []string{
		"manage.py",
		"Pipfile",
		"uv.lock",
		"poetry.lock",
		"requirements.txt",
		"pyproject.toml",
		"Cargo.toml",
	}
	for _, f := range nonJSFiles {
		if fileExists(filepath.Join(root, f)) {
			return true
		}
	}
	if !isMonorepoRoot && fileExists(filepath.Join(root, "go.mod")) {
		return true
	}

	pkgPath := filepath.Join(root, "package.json")
	if data, err := os.ReadFile(pkgPath); err == nil {
		var pkg struct {
			Scripts         map[string]string `json:"scripts"`
			Dependencies    map[string]string `json:"dependencies"`
			DevDependencies map[string]string `json:"devDependencies"`
		}
		if err := json.Unmarshal(data, &pkg); err == nil {
			if !isMonorepoRoot {
				if b, ok := pkg.Scripts["build"]; ok && strings.TrimSpace(b) != "" {
					return true
				}
			}
			targetDeps := []string{"next", "vite", "react", "vue", "svelte", "express", "hono"}
			for _, dep := range targetDeps {
				if _, ok := pkg.Dependencies[dep]; ok {
					return true
				}
				if _, ok := pkg.DevDependencies[dep]; ok {
					return true
				}
			}
		}
	}

	return false
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

