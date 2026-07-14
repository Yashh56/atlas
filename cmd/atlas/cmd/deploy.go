package cmd

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
	"github.com/spf13/cobra"

	"github.com/Yashh56/atlas/internal/orchestrator"
)

// validProviders is the exhaustive list of accepted --provider values.
var validProviders = map[string]bool{
	"vercel":  true,
	"render":  true,
	"netlify": true,
	"fly":     true,
	"railway": true,
}

var deployProvider string
var deployModel string
var deployAction string
var deployAllowDirty bool

var deployCmd = &cobra.Command{
	Use:           "deploy <path>",
	Short:         "Analyze, validate, and build a project for deployment",
	Long:          `deploy resolves the workspace, creates a session, analyzes the project, validates git state, and runs the build. Deployment provider integration is Week 4.`,
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]

		needsWizard := (deployProvider == "" || deployModel == "") && term.IsTerminal(int(os.Stdin.Fd()))

		if needsWizard {
			m, p, a, err := RunWizard()
			if err != nil {
				return fmt.Errorf("wizard failed or aborted: %w", err)
			}
			deployModel = m
			deployProvider = p
			deployAction = string(a)
		} else {
			if deployProvider == "" {
				return fmt.Errorf("flag --provider is required when running non-interactively\n\nUsage:\n  %s", cmd.UseLine())
			}
			if deployModel == "" {
				// No strict need for model if not using wizard, but if it is required, we would fail.
				// Wait, the existing code didn't require --model. Only --provider was required.
				// We should fail if we needed the wizard but stdin wasn't a terminal.
				if deployProvider == "" {
					return fmt.Errorf("flag --provider is required when running non-interactively\n\nUsage:\n  %s", cmd.UseLine())
				}
			}
		}

		// Re-check after wizard
		if deployProvider == "" {
			return fmt.Errorf("no provider resolved")
		}

		if !validProviders[strings.ToLower(deployProvider)] {
			return fmt.Errorf(
				"unknown provider %q — valid options are: %s",
				deployProvider,
				validProviderList(),
			)
		}

		opts := orchestrator.RunOptions{
			AllowDirty:    deployAllowDirty,
			ModelOverride: deployModel,
			Action:        orchestrator.Action(deployAction),
		}
		return orchestrator.Run(cmd.Context(), path, strings.ToLower(deployProvider), opts)
	},
}

func init() {
	deployCmd.Flags().StringVar(&deployProvider, "provider", "", "deployment provider (vercel, render, netlify, fly, railway)")
	deployCmd.Flags().StringVar(&deployModel, "model", "", "LLM provider override (anthropic, openai, etc.)")
	deployCmd.Flags().StringVar(&deployAction, "action", "deploy", "action mode: build, test, deploy, test-and-deploy")
	deployCmd.Flags().BoolVar(&deployAllowDirty, "allow-dirty", false, "skip the dirty working-tree check and proceed even with uncommitted changes")
}

// validProviderList returns a deterministic, human-readable list of valid providers.
func validProviderList() string {
	return strings.Join([]string{"fly", "netlify", "railway", "render", "vercel"}, ", ")
}
