package vercel

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Yashh56/atlas/internal/config"
	"github.com/Yashh56/atlas/internal/credentials"
	"github.com/Yashh56/atlas/internal/deploy"
)



// EnsureVercelAuth verifies that atlas can authenticate with Vercel before the
// pipeline runs. Failing fast here saves a wasted build+fix-loop run.
//
// Priority order:
//  1. VERCEL_TOKEN env var — silent, zero-friction.
//  2. Stored token in credential store — silent.
//  3. CLI delegated auth — interactive, prompts user as needed.
func EnsureVercelAuth(ctx context.Context, store *credentials.Store, cfg *config.Config, runner deploy.CommandRunner) error {
	return EnsureVercelAuthFull(ctx, store, cfg, runner, os.Stdin, os.Stdout)
}

func EnsureVercelAuthFull(ctx context.Context, store *credentials.Store, cfg *config.Config, runner deploy.CommandRunner, stdin io.Reader, stdout io.Writer) error {
	opts := deploy.CLIOptions{
		ProviderName:   "vercel",
		EnvVar:         "VERCEL_TOKEN",
		CLIName:        "vercel",
		InstallCommand: []string{"npm", "install", "-g", "vercel"},
		LoginCommand:   []string{"login"},
		WhoamiCommand:  []string{"whoami"},
		ParseAccount: func(output string) (string, error) {
			account := strings.TrimSpace(output)
			if account == "" {
				return "", fmt.Errorf("empty account string")
			}
			return account, nil
		},
	}
	return deploy.EnsureCLIAuthFull(ctx, store, runner, stdin, stdout, opts)
}
