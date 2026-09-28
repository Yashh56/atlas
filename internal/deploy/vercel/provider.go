package vercel

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"regexp"
	"time"

	"github.com/Yashh56/atlas/internal/credentials"
	"github.com/Yashh56/atlas/internal/deploy"
	"github.com/Yashh56/atlas/internal/tools"
)

var execCommandContext = exec.CommandContext

// VercelProvider implements Provider for Vercel.
type VercelProvider struct{}

func (v *VercelProvider) Name() string { return "vercel" }

func (v *VercelProvider) Deploy(ctx context.Context, in deploy.DeployInput) (*deploy.Deployment, error) {
	// If a token is provided in the input, we use it. Otherwise, we rely on the Vercel CLI's internal session.
	args := []string{"deploy", "--prod", "--yes", "--cwd", in.WorkspaceRoot}
	if in.Token != "" {
		args = append(args, "--token", in.Token)
	}

	// We reuse tools.RunCommand to execute the CLI.
	// We don't need a session here since we just want to run a simple command and get output.
	cmdTool := tools.RunCommand{
		Command: "vercel",
		Args:    args,
		Dir:     in.WorkspaceRoot,
	}

	res, err := cmdTool.Execute(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("vercel deploy failed to execute: %w", err)
	}

	if !res.Success {
		return nil, fmt.Errorf("vercel deploy failed (code: %s):\n%s", res.Error, res.Output)
	}

	url, err := parseVercelURL(res.Output)
	if err != nil {
		return nil, fmt.Errorf("failed to parse deployment URL from output: %w", err)
	}

	return &deploy.Deployment{
		URL:         url,
		Provider:    "vercel",
		ProviderRef: url,
		DeployedAt:  time.Now().UTC(),
	}, nil
}

func (v *VercelProvider) HealthCheck(ctx context.Context, d *deploy.Deployment) error {
	return deploy.HTTPHealthCheck(ctx, d.URL, 200)
}

func (v *VercelProvider) Rollback(ctx context.Context, to *deploy.Deployment, in deploy.DeployInput) error {
	args := []string{"rollback", to.ProviderRef, "--yes"}
	if in.Token != "" {
		args = append(args, "--token", in.Token)
	}

	cmdTool := tools.RunCommand{
		Command: "vercel",
		Args:    args,
		Dir:     in.WorkspaceRoot,
	}

	res, err := cmdTool.Execute(ctx, nil)
	if err != nil {
		return fmt.Errorf("vercel rollback execution failed: %w", err)
	}
	if !res.Success {
		return fmt.Errorf("vercel rollback failed: %s\nOutput:\n%s", res.Error, res.Output)
	}
	return nil
}

// parseVercelURL extracts the deployment URL from the stdout of the vercel CLI.
// It looks for a https://*.vercel.app URL on a line by itself or at the end of the output.
func parseVercelURL(output string) (string, error) {
	// Simple regex to find the vercel app URL in the output.
	re := regexp.MustCompile(`(https://[a-zA-Z0-9\-\.]+\.vercel\.app)`)
	matches := re.FindAllStringSubmatch(output, -1)

	if len(matches) > 0 {
		// Vercel usually prints the prod URL last
		return matches[len(matches)-1][1], nil
	}

	return "", fmt.Errorf("no vercel.app URL found in output")
}

func (p *VercelProvider) Logs(ctx context.Context, d *deploy.Deployment, w io.Writer, follow bool) error {
	token, _ := resolveVercelToken()

	args := []string{"logs", d.URL}
	if token != "" {
		args = append(args, "--token", token)
	}
	if follow {
		args = append(args, "--follow")
	}

	cmdName := "vercel"
	if runtime.GOOS == "windows" {
		if p, err := exec.LookPath(cmdName); err == nil {
			ext := strings.ToLower(filepath.Ext(p))
			if ext == ".cmd" || ext == ".bat" {
				args = append([]string{"/d", "/c", p}, args...)
				cmdName = "cmd.exe"
			}
		}
	}

	cmd := execCommandContext(ctx, cmdName, args...)
	cmd.Stdout = w
	cmd.Stderr = w
	return cmd.Run()
}

func resolveVercelToken() (string, error) {
	// A bit hacky, but replicates deploy.EnsureProviderAuth's behavior.
	// Since we don't have store easily, we load it.
	store, _ := credentials.Open()
	token, _ := deploy.EnsureProviderAuth("vercel", "VERCEL_TOKEN", store, io.Discard)
	// Even if token is empty, vercel CLI might be authenticated via ~/.local/share/com.vercel.cli
	return token, nil
}
