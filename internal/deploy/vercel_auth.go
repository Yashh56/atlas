package deploy

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/Yashh56/atlas/internal/config"
	"github.com/Yashh56/atlas/internal/credentials"
)

// CommandRunner is the interface for running shell commands, injected for testability.
type CommandRunner interface {
	// LookPath searches for the named executable in PATH.
	// Returns ("", error) if not found.
	LookPath(file string) (string, error)
	// Run executes the named command with args, capturing combined output.
	// Returns (stdout, error).
	Run(ctx context.Context, name string, args ...string) (string, error)
	// RunInteractive executes the named command inheriting the parent's
	// stdin/stdout/stderr so the user sees live prompts.
	RunInteractive(ctx context.Context, name string, args ...string) error
}

// OSCommandRunner is the real CommandRunner that calls the actual OS.
type OSCommandRunner struct{}

func (OSCommandRunner) LookPath(file string) (string, error) {
	return lookPath(file)
}

func (OSCommandRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	return runCapture(ctx, name, args...)
}

func (OSCommandRunner) RunInteractive(ctx context.Context, name string, args ...string) error {
	return runInherited(ctx, name, args...)
}

// EnsureVercelAuth verifies that atlas can authenticate with Vercel before the
// pipeline runs. Failing fast here saves a wasted build+fix-loop run.
//
// Priority order:
//  1. VERCEL_TOKEN env var — silent, zero-friction.
//  2. Stored token in credential store — silent.
//  3. CLI delegated auth — interactive, prompts user as needed.
func EnsureVercelAuth(ctx context.Context, store *credentials.Store, cfg *config.Config, runner CommandRunner) error {
	return EnsureVercelAuthFull(ctx, store, cfg, runner, os.Stdin, os.Stdout)
}

// EnsureVercelAuthFull is the testable variant that accepts io.Reader/Writer
// so tests can inject fake stdin/stdout without spawning real processes.
func EnsureVercelAuthFull(
	ctx context.Context,
	store *credentials.Store,
	cfg *config.Config,
	runner CommandRunner,
	stdin io.Reader,
	stdout io.Writer,
) error {
	// 1. VERCEL_TOKEN env var.
	if os.Getenv("VERCEL_TOKEN") != "" {
		fmt.Fprintln(stdout, "✓ Vercel authenticated (env_var, VERCEL_TOKEN)")
		return nil
	}

	// 2. Stored token in credential store.
	if store != nil {
		if meta, ok, err := store.GetMeta("vercel"); err == nil && ok {
			if meta.Method == credentials.MethodStoredToken {
				fmt.Fprintf(stdout, "✓ Vercel authenticated (stored_token, %s)\n", meta.Account)
				return nil
			}
		}
		// Also check for a CLI session credential.
		if meta, ok, err := store.GetMeta("vercel"); err == nil && ok {
			if meta.Method == credentials.MethodCLISession {
				fmt.Fprintf(stdout, "✓ Vercel authenticated (cli_session, %s)\n", meta.Account)
				return nil
			}
		}
	}

	// 3. CLI-delegated auth.
	fmt.Fprintln(stdout, "→ Checking Vercel authentication...")

	cliPath, lookErr := runner.LookPath("vercel")
	if lookErr != nil {
		// CLI not found — check if npm/node are available first.
		_, npmErr := runner.LookPath("npm")
		_, nodeErr := runner.LookPath("node")
		if npmErr != nil || nodeErr != nil {
			return fmt.Errorf("vercel CLI not found, and npm/node are also not found.\n" +
				"  Install Node.js first: https://nodejs.org, then run: npm install -g vercel")
		}
		fmt.Fprintln(stdout, "  Vercel CLI not found. Install it now? (npm install -g vercel) [y/N]")
		line, _ := bufio.NewReader(stdin).ReadString('\n')
		line = strings.TrimSpace(strings.ToLower(line))
		if line != "y" && line != "yes" {
			return fmt.Errorf("vercel CLI installation declined — deployment requires the Vercel CLI.\n" +
				"  Run `npm install -g vercel` manually and re-run atlas deploy.")
		}
		fmt.Fprintln(stdout, "  Installing vercel CLI...")
		if _, err := runner.Run(ctx, "npm", "install", "-g", "vercel"); err != nil {
			return fmt.Errorf("failed to install vercel CLI: %w", err)
		}
		cliPath, lookErr = runner.LookPath("vercel")
		if lookErr != nil {
			return fmt.Errorf("vercel CLI still not found after install — check your PATH: %w", lookErr)
		}
	}
	_ = cliPath

	// Run `vercel whoami` to check if already logged in.
	account, whoamiErr := runner.Run(ctx, "vercel", "whoami")
	account = strings.TrimSpace(account)
	if whoamiErr == nil && account != "" {
		// Already logged in.
		if store != nil {
			_ = store.SetMeta(credentials.ProviderCredential{
				Provider:   "vercel",
				Method:     credentials.MethodCLISession,
				VerifiedAt: time.Now().UTC(),
				Account:    account,
			})
		}
		fmt.Fprintf(stdout, "✓ Vercel authenticated (cli_session, %s)\n", account)
		return nil
	}

	// Not logged in — run `vercel login` interactively.
	fmt.Fprintln(stdout, "  Not logged in to Vercel. Running `vercel login`...")
	if err := runner.RunInteractive(ctx, "vercel", "login"); err != nil {
		return fmt.Errorf("vercel login failed: %w", err)
	}

	// Re-confirm login.
	account, whoamiErr = runner.Run(ctx, "vercel", "whoami")
	account = strings.TrimSpace(account)
	if whoamiErr != nil || account == "" {
		return fmt.Errorf("vercel login appeared to succeed but `vercel whoami` still fails — " +
			"check your login and try again")
	}

	if store != nil {
		_ = store.SetMeta(credentials.ProviderCredential{
			Provider:   "vercel",
			Method:     credentials.MethodCLISession,
			VerifiedAt: time.Now().UTC(),
			Account:    account,
		})
	}
	fmt.Fprintf(stdout, "✓ Vercel authenticated (cli_session, %s)\n", account)
	return nil
}
