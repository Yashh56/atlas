package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
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
	Short: "Parse and validate a deployment request",
	Long: `deploy parses and validates a deployment request.
It does not perform any actual deployment in Week 1 — it verifies the
provided path and provider arguments and confirms they are understood.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]

		if deployProvider == "" {
			return fmt.Errorf("flag --provider is required\n\nUsage:\n  %s", cmd.UseLine())
		}

		if !validProviders[strings.ToLower(deployProvider)] {
			valid := validProviderList()
			return fmt.Errorf(
				"unknown provider %q — valid options are: %s",
				deployProvider,
				valid,
			)
		}

		fmt.Printf("Project: %s\n", path)
		fmt.Printf("Provider: %s\n", strings.ToLower(deployProvider))
		fmt.Println("✓ Command parsed")
		return nil
	},
}

func init() {
	deployCmd.Flags().StringVar(&deployProvider, "provider", "", "deployment provider (vercel, render, netlify, fly, railway)")
}

// validProviderList returns a human-readable, sorted list of valid providers.
func validProviderList() string {
	list := make([]string, 0, len(validProviders))
	for p := range validProviders {
		list = append(list, p)
	}
	// Deterministic ordering for tests and error messages.
	sorted := []string{"fly", "netlify", "railway", "render", "vercel"}
	_ = list
	return strings.Join(sorted, ", ")
}
