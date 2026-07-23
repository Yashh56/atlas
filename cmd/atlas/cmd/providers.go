package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/Yashh56/atlas/internal/credentials"
	"github.com/Yashh56/atlas/internal/orchestrator"
	"github.com/Yashh56/atlas/internal/workspace"
)

var providersCmd = &cobra.Command{
	Use:   "providers",
	Short: "Show authentication status for all supported deploy providers",
	Long:  `providers lists all supported deployment providers and shows whether Atlas has valid credentials for each.`,
	RunE:  runProviders,
}

var providersSetCmd = &cobra.Command{
	Use:   "set <provider>",
	Short: "Securely store a deployment provider token",
	Args:  cobra.ExactArgs(1),
	RunE:  runProvidersSet,
}

var providersUnsetCmd = &cobra.Command{
	Use:   "unset <provider>",
	Short: "Remove a stored deployment provider token",
	Args:  cobra.ExactArgs(1),
	RunE:  runProvidersUnset,
}

func init() {
	providersSetCmd.Flags().String("service-id", "", "Render service ID for the current project")
	providersCmd.AddCommand(providersSetCmd)
	providersCmd.AddCommand(providersUnsetCmd)
}

func runProviders(_ *cobra.Command, _ []string) error {
	store, err := openCredentials()
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠  Could not open credential store: %v\n", err)
		store = nil
	}

	fmt.Println("DEPLOY PROVIDERS")
	printProviderStatus("vercel", "VERCEL_TOKEN", store)
	printProviderStatus("render", "RENDER_TOKEN", store)
	printProviderStatus("netlify", "NETLIFY_TOKEN", store)
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

func runProvidersSet(cmd *cobra.Command, args []string) error {
	provider := args[0]
	if provider != "vercel" && provider != "render" {
		return fmt.Errorf("provider %q is not implemented yet", provider)
	}

	key, err := promptSecret(fmt.Sprintf("%s token", strings.Title(provider)))
	if err != nil {
		return err
	}

	store, err := openCredentials()
	if err != nil {
		return fmt.Errorf("opening credential store: %w", err)
	}

	if err := store.SetSecret(provider, key); err != nil {
		return fmt.Errorf("storing secret: %w", err)
	}

	if err := store.SetMeta(credentials.ProviderCredential{
		Provider:   provider,
		Method:     credentials.MethodStoredToken,
		VerifiedAt: time.Now().UTC(),
	}); err != nil {
		return fmt.Errorf("storing metadata: %w", err)
	}

	fmt.Println("✓ Stored token.")

	if provider == "render" {
		serviceID, _ := cmd.Flags().GetString("service-id")
		if serviceID != "" {
			dir, _ := os.Getwd()
			ws, err := workspace.Resolve(dir)
			if err != nil {
				return fmt.Errorf("could not resolve workspace to save service ID: %w", err)
			}

			var proj orchestrator.ProjectState
			if p, err := orchestrator.LoadProject(filepath.Join(ws.Root, ".atlas")); err == nil {
				proj = *p
			}
			proj.RenderServiceID = &serviceID
			if err := orchestrator.SaveProject(filepath.Join(ws.Root, ".atlas"), &proj); err != nil {
				return fmt.Errorf("storing project config: %w", err)
			}
			fmt.Println("✓ Stored Render service ID in project configuration.")
		}
	}

	fmt.Println("This will be used instead of the CLI-delegated login next time you deploy.")
	return nil
}

func runProvidersUnset(_ *cobra.Command, args []string) error {
	provider := args[0]
	if provider != "vercel" && provider != "render" {
		return fmt.Errorf("provider %q is not implemented yet", provider)
	}

	store, err := openCredentials()
	if err != nil {
		return fmt.Errorf("opening credential store: %w", err)
	}

	meta, ok, _ := store.GetMeta(provider)
	if !ok || meta.Method != credentials.MethodStoredToken {
		fmt.Printf("No stored key found for %q\n", provider)
		return nil
	}

	if err := store.DeleteSecret(provider); err != nil {
		return fmt.Errorf("deleting secret: %w", err)
	}

	store.SetMeta(credentials.ProviderCredential{
		Provider: provider,
		Method:   credentials.MethodEnvVar,
	})

	fmt.Printf("✓ Removed stored key for %q.\n", provider)
	return nil
}
