package cmd

import (
	"fmt"
	"strings"

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

var deployCmd = &cobra.Command{
	Use:   "deploy <path>",
	Short: "Analyze a project and prepare a deployment session",
	Long: `deploy resolves the workspace, creates a session, and runs project analysis.
It does not execute a build or deploy in Week 2 — that is Week 3.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]

		if deployProvider == "" {
			return fmt.Errorf("flag --provider is required\n\nUsage:\n  %s", cmd.UseLine())
		}

		if !validProviders[strings.ToLower(deployProvider)] {
			return fmt.Errorf(
				"unknown provider %q — valid options are: %s",
				deployProvider,
				validProviderList(),
			)
		}

		return orchestrator.Run(cmd.Context(), path, strings.ToLower(deployProvider))
	},
}

func init() {
	deployCmd.Flags().StringVar(&deployProvider, "provider", "", "deployment provider (vercel, render, netlify, fly, railway)")
}

// validProviderList returns a deterministic, human-readable list of valid providers.
func validProviderList() string {
	return strings.Join([]string{"fly", "netlify", "railway", "render", "vercel"}, ", ")
}
