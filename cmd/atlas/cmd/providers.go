package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Yashh56/atlas/internal/credentials"
)

var providersCmd = &cobra.Command{
	Use:   "providers",
	Short: "Show authentication status for all supported deploy providers",
	Long:  `providers lists all supported deployment providers and shows whether Atlas has valid credentials for each.`,
	RunE:  runProviders,
}

func runProviders(_ *cobra.Command, _ []string) error {
	store, err := credentials.Open()
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠  Could not open credential store: %v\n", err)
		store = nil
	}

	fmt.Println("DEPLOY PROVIDERS")
	printProviderStatus("vercel", "VERCEL_TOKEN", store)
	printProviderStatus("render", "RENDER_API_KEY", store)
	printProviderStatus("netlify", "NETLIFY_AUTH_TOKEN", store)
	printProviderStatus("fly", "FLY_API_TOKEN", store)
	printProviderStatus("railway", "RAILWAY_API_TOKEN", store)
	return nil
}

func printProviderStatus(name, envVar string, store *credentials.Store) {
	const colWidth = 10

	pad := strings.Repeat(" ", colWidth-len(name))

	// 1. Check env var first.
	if os.Getenv(envVar) != "" {
		fmt.Printf("  %s%s✓ authenticated   (env_var, %s)\n", name, pad, envVar)
		return
	}

	// 2. Check credential store metadata.
	if store != nil {
		if meta, ok, storeErr := store.GetMeta(name); storeErr == nil && ok {
			verified := meta.VerifiedAt.Format(time.DateTime)
			account := meta.Account
			if account == "" {
				account = "unknown"
			}
			fmt.Printf("  %s%s✓ authenticated   (%s, %s, verified %s)\n",
				name, pad, meta.Method, account, verified)
			return
		}
	}

	// 3. Not configured.
	fmt.Printf("  %s%s✗ not configured\n", name, pad)
}
