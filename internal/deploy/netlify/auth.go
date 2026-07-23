package netlify

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Yashh56/atlas/internal/config"
	"github.com/Yashh56/atlas/internal/credentials"
	"github.com/Yashh56/atlas/internal/deploy"
)

// EnsureNetlifyAuth verifies that atlas can authenticate with Netlify.
func EnsureNetlifyAuth(ctx context.Context, store *credentials.Store, cfg *config.Config, runner deploy.CommandRunner) error {
	return EnsureNetlifyAuthFull(ctx, store, cfg, runner, os.Stdin, os.Stdout)
}

// EnsureNetlifyAuthFull is the testable variant.
func EnsureNetlifyAuthFull(ctx context.Context, store *credentials.Store, cfg *config.Config, runner deploy.CommandRunner, stdin io.Reader, stdout io.Writer) error {
	opts := deploy.CLIOptions{
		ProviderName:   "netlify",
		EnvVar:         "NETLIFY_TOKEN",
		CLIName:        "netlify",
		InstallCommand: []string{"npm", "install", "-g", "netlify-cli"},
		LoginCommand:   []string{"login"},
		WhoamiCommand:  []string{"api", "getCurrentUser"},
		ParseAccount: func(output string) (string, error) {
			start := strings.Index(output, "{")
			end := strings.LastIndex(output, "}")
			if start == -1 || end == -1 || start > end {
				return "", fmt.Errorf("no JSON object found in output")
			}
			jsonStr := output[start : end+1]

			var user struct {
				Slug string `json:"slug"`
			}
			if err := json.Unmarshal([]byte(jsonStr), &user); err != nil {
				return "", fmt.Errorf("failed to parse getCurrentUser json: %w", err)
			}
			if user.Slug == "" {
				return "", fmt.Errorf("empty user slug")
			}
			return user.Slug, nil
		},
	}
	return deploy.EnsureCLIAuthFull(ctx, store, runner, stdin, stdout, opts)
}
