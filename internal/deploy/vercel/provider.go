package vercel

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/Yashh56/atlas/internal/deploy"
	"github.com/Yashh56/atlas/internal/tools"
)

// VercelProvider implements Provider for Vercel.
type VercelProvider struct{}

func (v *VercelProvider) Name() string { return "vercel" }

func (v *VercelProvider) Deploy(ctx context.Context, in deploy.DeployInput) (*deploy.Deployment, error) {
	token := os.Getenv("VERCEL_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("VERCEL_TOKEN not set — see docs for how to generate one")
	}

	// We reuse tools.RunCommand to execute the CLI.
	// We don't need a session here since we just want to run a simple command and get output.
	cmdTool := tools.RunCommand{
		Command: "vercel",
		Args:    []string{"deploy", "--prod", "--yes", "--token", token, "--cwd", in.WorkspaceRoot},
		Dir:     in.WorkspaceRoot,
	}

	res, err := cmdTool.Execute(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("vercel deploy failed to execute: %w", err)
	}

	if !res.Success {
		return nil, fmt.Errorf("vercel deploy failed: %s", res.Error)
	}

	url, err := parseVercelURL(res.Output)
	if err != nil {
		return nil, fmt.Errorf("failed to parse deployment URL from output: %w", err)
	}

	return &deploy.Deployment{
		URL:        url,
		Provider:   "vercel",
		DeployedAt: time.Now().UTC(),
	}, nil
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
