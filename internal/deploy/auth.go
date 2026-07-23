package deploy

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/Yashh56/atlas/internal/credentials"
)

// EnsureProviderAuth checks for authentication via environment variable or stored token.
// It returns the resolved token if found, or an empty string if not found.
// This is used as the first step for all providers before falling back to provider-specific auth (like CLI).
func EnsureProviderAuth(
	providerName string,
	envVar string,
	store *credentials.Store,
	stdout io.Writer,
) (string, error) {
	titleName := strings.Title(providerName)

	// 1. Check env var first.
	if envVal := os.Getenv(envVar); envVal != "" {
		fmt.Fprintf(stdout, "✓ %s authenticated (env_var, %s)\n", titleName, envVar)
		return envVal, nil
	}

	// 2. Check stored token.
	if store != nil {
		if meta, ok, err := store.GetMeta(providerName); err == nil && ok {
			if meta.Method == credentials.MethodStoredToken {
				token, err := store.GetSecret(providerName)
				if err == nil && token != "" {
					account := meta.Account
					if account == "" {
						account = "unknown"
					}
					fmt.Fprintf(stdout, "✓ %s authenticated (stored_token, %s)\n", titleName, account)
					return token, nil
				}
			}
		}
	}

	return "", nil
}

// CLIOptions holds the provider-specific CLI names and commands for generalized auth.
type CLIOptions struct {
	ProviderName   string
	EnvVar         string
	CLIName        string
	InstallCommand []string // e.g. ["npm", "install", "-g", "vercel"]
	LoginCommand   []string // e.g. ["login"]
	WhoamiCommand  []string // e.g. ["whoami"]
	// ParseAccount extracts the account name from the whoami command's output.
	ParseAccount func(string) (string, error)
}

// EnsureCLIInstalled checks if a CLI tool is installed, and if not, prompts the user to install it.
func EnsureCLIInstalled(
	ctx context.Context,
	runner CommandRunner,
	stdin io.Reader,
	stdout io.Writer,
	cliName string,
	installCommand []string,
) error {
	_, lookErr := runner.LookPath(cliName)
	if lookErr != nil {
		// CLI not found — check if npm/node are available first.
		_, npmErr := runner.LookPath("npm")
		_, nodeErr := runner.LookPath("node")
		if npmErr != nil || nodeErr != nil {
			return fmt.Errorf("%s CLI not found, and npm/node are also not found.\n"+
				"  Install Node.js first: https://nodejs.org, then run: %s", cliName, strings.Join(installCommand, " "))
		}
		titleName := strings.Title(cliName)
		fmt.Fprintf(stdout, "  %s CLI not found. Install it now? (%s) [y/N]\n", titleName, strings.Join(installCommand, " "))
		
		scanner := bufio.NewScanner(stdin)
		scanner.Scan()
		line := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if line != "y" && line != "yes" {
			return fmt.Errorf("%s CLI installation declined — deployment requires the %s CLI.\n"+
				"  Run `%s` manually and re-run atlas deploy.", cliName, titleName, strings.Join(installCommand, " "))
		}
		fmt.Fprintf(stdout, "  Installing %s CLI...\n", cliName)
		if _, err := runner.Run(ctx, "", installCommand[0], installCommand[1:]...); err != nil {
			return fmt.Errorf("failed to install %s CLI: %w", cliName, err)
		}
		_, lookErr = runner.LookPath(cliName)
		if lookErr != nil {
			return fmt.Errorf("%s CLI still not found after install — check your PATH: %w", cliName, lookErr)
		}
	}
	return nil
}

// EnsureCLIAuthFull generalizes the CLI-delegated login pattern used by Vercel and Netlify.
// Priority order:
//  1. Env var — silent, zero-friction.
//  2. Stored token in credential store — silent.
//  3. CLI delegated auth — interactive, prompts user as needed.
func EnsureCLIAuthFull(
	ctx context.Context,
	store *credentials.Store,
	runner CommandRunner,
	stdin io.Reader,
	stdout io.Writer,
	opts CLIOptions,
) error {
	// 1 & 2: Env var and Stored token check via shared logic
	token, err := EnsureProviderAuth(opts.ProviderName, opts.EnvVar, store, stdout)
	if err != nil {
		return err
	}

	titleName := strings.Title(opts.ProviderName)

	// We MUST ensure the CLI is installed, regardless of authentication method,
	// because the deployment step relies on the CLI to execute.
	if err := EnsureCLIInstalled(ctx, runner, stdin, stdout, opts.CLIName, opts.InstallCommand); err != nil {
		return err
	}

	// If we already authenticated via env var or stored token, we can stop here
	// now that we've verified the CLI is installed.
	if token != "" {
		return nil
	}

	// 3. Check for a CLI session credential in store.
	if store != nil {
		if meta, ok, err := store.GetMeta(opts.ProviderName); err == nil && ok {
			if meta.Method == credentials.MethodCLISession {
				fmt.Fprintf(stdout, "✓ %s authenticated (cli_session, %s)\n", titleName, meta.Account)
				return nil
			}
		}
	}

	fmt.Fprintf(stdout, "→ Checking %s authentication...\n", titleName)

	// Run whoami to check if already logged in.
	accountRaw, whoamiErr := runner.Run(ctx, "", opts.CLIName, opts.WhoamiCommand...)
	account, parseErr := opts.ParseAccount(accountRaw)
	if whoamiErr == nil && parseErr == nil && account != "" {
		// Already logged in.
		if store != nil {
			_ = store.SetMeta(credentials.ProviderCredential{
				Provider:   opts.ProviderName,
				Method:     credentials.MethodCLISession,
				VerifiedAt: time.Now().UTC(),
				Account:    account,
			})
		}
		fmt.Fprintf(stdout, "✓ %s authenticated (cli_session, %s)\n", titleName, account)
		return nil
	}

	// Not logged in — run login interactively.
	fmt.Fprintf(stdout, "  Not logged in to %s. Running `%s %s`...\n", titleName, opts.CLIName, strings.Join(opts.LoginCommand, " "))
	if err := runner.RunInteractive(ctx, "", opts.CLIName, opts.LoginCommand...); err != nil {
		return fmt.Errorf("%s login failed: %w", opts.CLIName, err)
	}

	// Re-confirm login.
	accountRaw, whoamiErr = runner.Run(ctx, "", opts.CLIName, opts.WhoamiCommand...)
	account, parseErr = opts.ParseAccount(accountRaw)
	if whoamiErr != nil || parseErr != nil || account == "" {
		return fmt.Errorf("%s login appeared to succeed but `%s %s` still fails — check your login and try again", 
			opts.CLIName, opts.CLIName, strings.Join(opts.WhoamiCommand, " "))
	}

	if store != nil {
		_ = store.SetMeta(credentials.ProviderCredential{
			Provider:   opts.ProviderName,
			Method:     credentials.MethodCLISession,
			VerifiedAt: time.Now().UTC(),
			Account:    account,
		})
	}
	fmt.Fprintf(stdout, "✓ %s authenticated (cli_session, %s)\n", titleName, account)
	return nil
}
