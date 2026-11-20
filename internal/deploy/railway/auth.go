package railway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"golang.org/x/term"

	"github.com/Yashh56/atlas/internal/config"
	"github.com/Yashh56/atlas/internal/credentials"
	"github.com/Yashh56/atlas/internal/deploy"
)

// EnsureRailwayAuth verifies that atlas can authenticate with Railway.
func EnsureRailwayAuth(ctx context.Context, store *credentials.Store, cfg *config.Config, runner deploy.CommandRunner) error {
	return EnsureRailwayAuthFull(ctx, store, cfg, runner, os.Stdin, os.Stdout)
}

// EnsureRailwayAuthFull is the testable variant with explicit stdin/stdout.
// Priority order:
//  1. Environment variables: RAILWAY_TOKEN (project token) first, then RAILWAY_API_TOKEN (workspace token).
//  2. Stored credential in credentials.Store under "railway".
//  3. CLI installation verification via deploy.EnsureCLIInstalled.
//  4. Delegated session check: railway whoami.
//  5. Interactive login: railway login (or railway login --browserless if non-interactive).
func EnsureRailwayAuthFull(ctx context.Context, store *credentials.Store, cfg *config.Config, runner deploy.CommandRunner, stdin io.Reader, stdout io.Writer) error {
	envVar := "RAILWAY_TOKEN"
	if os.Getenv("RAILWAY_TOKEN") == "" && os.Getenv("RAILWAY_API_TOKEN") != "" {
		envVar = "RAILWAY_API_TOKEN"
	}

	loginCmd := []string{"login"}
	if f, ok := stdin.(*os.File); ok && !term.IsTerminal(int(f.Fd())) {
		loginCmd = []string{"login", "--browserless"}
	}

	opts := deploy.CLIOptions{
		ProviderName:   "railway",
		EnvVar:         envVar,
		CLIName:        "railway",
		InstallCommand: resolveRailwayInstallCommand(runner),
		LoginCommand:   loginCmd,
		WhoamiCommand:  []string{"whoami"},
		ParseAccount:   parseRailwayAccount,
	}
	return deploy.EnsureCLIAuthFull(ctx, store, runner, stdin, stdout, opts)
}

// resolveRailwayInstallCommand determines the platform-preferred command to install the Railway CLI.
func resolveRailwayInstallCommand(runner deploy.CommandRunner) []string {
	return resolveRailwayInstallCommandForOS(runner, runtime.GOOS)
}

func resolveRailwayInstallCommandForOS(runner deploy.CommandRunner, goos string) []string {
	switch goos {
	case "windows":
		if _, err := runner.LookPath("scoop"); err == nil {
			return []string{"scoop", "install", "railway"}
		}
		if _, err := runner.LookPath("winget"); err == nil {
			return []string{"winget", "install", "--id", "Railway.Railway", "-e"}
		}
		if _, err := runner.LookPath("npm"); err == nil {
			return []string{"npm", "install", "-g", "@railway/cli"}
		}
		return []string{"scoop", "install", "railway"}
	case "darwin":
		if _, err := runner.LookPath("brew"); err == nil {
			return []string{"brew", "install", "railway"}
		}
		if _, err := runner.LookPath("npm"); err == nil {
			return []string{"npm", "install", "-g", "@railway/cli"}
		}
		return []string{"brew", "install", "railway"}
	default:
		if _, err := runner.LookPath("brew"); err == nil {
			return []string{"brew", "install", "railway"}
		}
		if _, err := runner.LookPath("npm"); err == nil {
			return []string{"npm", "install", "-g", "@railway/cli"}
		}
		return []string{"npm", "install", "-g", "@railway/cli"}
	}
}

// parseRailwayAccount parses account email or username from railway whoami output.
func parseRailwayAccount(output string) (string, error) {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return "", fmt.Errorf("empty whoami output")
	}
	lower := strings.ToLower(trimmed)
	if strings.Contains(lower, "not logged in") || strings.Contains(lower, "unauthorized") || strings.Contains(lower, "error:") {
		return "", fmt.Errorf("not logged in: %s", trimmed)
	}

	// Try JSON first: {"email": "...", "name": "..."}
	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start != -1 && end != -1 && start < end {
		var u struct {
			Email string `json:"email"`
			Name  string `json:"name"`
		}
		if err := json.Unmarshal([]byte(trimmed[start:end+1]), &u); err == nil {
			if u.Email != "" {
				return u.Email, nil
			}
			if u.Name != "" {
				return u.Name, nil
			}
		}
	}

	// Strip "Logged in as " prefix if present
	if strings.Contains(trimmed, "Logged in as ") {
		trimmed = strings.TrimPrefix(trimmed, "Logged in as ")
		trimmed = strings.TrimSpace(trimmed)
	}

	// If in format "Name (email)", extract email
	if startParen := strings.LastIndex(trimmed, "("); startParen != -1 {
		if endParen := strings.LastIndex(trimmed, ")"); endParen > startParen {
			email := strings.TrimSpace(trimmed[startParen+1 : endParen])
			if email != "" {
				return email, nil
			}
		}
	}

	lines := strings.Split(trimmed, "\n")
	firstLine := strings.TrimSpace(lines[0])
	if firstLine == "" {
		return "", fmt.Errorf("unable to parse railway account from output: %s", output)
	}
	return firstLine, nil
}
