// Package cmd wires together all Atlas CLI commands.
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/Yashh56/atlas/internal/orchestrator"
	"github.com/Yashh56/atlas/internal/version"
	"github.com/Yashh56/atlas/internal/cliutil"
)

var (
	pathFlag         string
	deployModel      string
	deployAction     string
	deployProvider   string
	deployAllowDirty bool
	outputDirFlag    string
	autoRollbackFlag bool
)

// validProviders is the exhaustive list of accepted --provider values.
var validProviders = map[string]bool{
	"vercel":  true,
	"render":  true,
	"netlify": true,
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

// Execute is the entry point called from main.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.Flags().StringVar(&pathFlag, "path", "", "project path (alternative to positional arg, for paths that collide with a subcommand name)")
	rootCmd.Flags().StringVar(&deployModel, "model", "", "LLM provider to use (anthropic, openai, etc.)")
	rootCmd.Flags().StringVar(&deployAction, "action", "", "action mode: build, test, deploy, test-and-deploy")
	rootCmd.Flags().StringVar(&deployProvider, "provider", "", "deployment provider (vercel, render, netlify, fly, railway)")
	rootCmd.Flags().BoolVar(&deployAllowDirty, "allow-dirty", false, "skip the dirty working-tree check and proceed even with uncommitted changes")
	rootCmd.Flags().StringVar(&outputDirFlag, "output-dir", "", "manual output directory override (e.g. build, dist, out)")
	rootCmd.Flags().BoolVar(&autoRollbackFlag, "auto-rollback-on-unhealthy", false, "automatically rollback without prompting if post-deploy health check fails")

	rootCmd.AddCommand(debugCmd)
	rootCmd.AddCommand(testllmCmd)
	rootCmd.AddCommand(modelsCmd)
	rootCmd.AddCommand(providersCmd)

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
	return a == orchestrator.ActionDeploy || a == orchestrator.ActionTestAndDeploy
}

func runPipeline(cmd *cobra.Command, args []string) error {
	var path string

	if len(args) == 0 && cmd.Flags().NFlag() == 0 {
		return cmd.Help()
	}

	if len(args) > 0 {
		path = args[0]
	} else if pathFlag != "" {
		path = pathFlag
	}

	if path != "" {
		_ = godotenv.Load(filepath.Join(path, ".env"))
	}

	isInteractive := term.IsTerminal(int(os.Stdin.Fd()))

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

	if deployProvider != "" && !validProviders[strings.ToLower(deployProvider)] {
		return fmt.Errorf(
			"unknown provider %q — valid options are: fly, netlify, railway, render, vercel",
			deployProvider,
		)
	}

	opts := orchestrator.RunOptions{
		AllowDirty:    deployAllowDirty,
		ModelOverride: deployModel,
		Action:        orchestrator.Action(deployAction),
		IsInteractive: isInteractive,
		OutputDir:     outputDirFlag,
		AutoRollback:  autoRollbackFlag,
	}
	return orchestrator.Run(cmd.Context(), path, strings.ToLower(deployProvider), opts)
}
