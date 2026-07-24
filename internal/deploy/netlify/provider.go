package netlify

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Yashh56/atlas/internal/build"
	"github.com/Yashh56/atlas/internal/deploy"
	"github.com/Yashh56/atlas/internal/state"
)

// NetlifyProvider implements Provider for Netlify.
type NetlifyProvider struct {
	Runner deploy.CommandRunner
}

func (n *NetlifyProvider) runner() deploy.CommandRunner {
	if n.Runner == nil {
		return deploy.OSCommandRunner{}
	}
	return n.Runner
}

func (n *NetlifyProvider) Name() string { return "netlify" }

type projectData struct {
	Framework      *string `json:"framework,omitempty"`
	PackageManager *string `json:"package_manager,omitempty"`
	NetlifySiteID  string  `json:"netlify_site_id,omitempty"`
}

func (n *NetlifyProvider) Deploy(ctx context.Context, in deploy.DeployInput) (*deploy.Deployment, error) {
	var proj projectData
	if err := state.LoadJSON(in.SessionDir, "project.json", &proj); err != nil {
		return nil, fmt.Errorf("failed to load project state: %w", err)
	}

	outDir := in.OutputDir
	if outDir == "" {
		if _, err := os.Stat(filepath.Join(in.WorkspaceRoot, "netlify.toml")); os.IsNotExist(err) {
			fw := ""
			if proj.Framework != nil {
				fw = *proj.Framework
			}
			pm := ""
			if proj.PackageManager != nil {
				pm = *proj.PackageManager
			}

			resolved, err := build.ResolvePublishDir(fw, pm, in.WorkspaceRoot)
			if err != nil {
				return nil, err
			}
			outDir = resolved
		}
	}

	// Resolve the site ID
	siteID := proj.NetlifySiteID
	if siteID == "" {
		newID, err := resolveNetlifySite(ctx, n.runner(), in.WorkspaceRoot, in.Token)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve netlify site: %w", err)
		}
		siteID = newID
		proj.NetlifySiteID = siteID

		_ = state.SaveJSON(in.SessionDir, "project.json", &proj)
		_ = state.SaveJSON(filepath.Join(in.WorkspaceRoot, ".atlas"), "project.json", &proj)
	}

	args := []string{"deploy", "--prod", "--site", siteID, "--json"}
	if in.Token != "" {
		args = append(args, "--auth", in.Token)
	}
	if outDir != "" {
		if _, err := os.Stat(filepath.Join(in.WorkspaceRoot, outDir)); err == nil {
			args = append(args, "--dir", outDir)
		}
	}
	outStr, err := n.runner().Run(ctx, in.WorkspaceRoot, "netlify", args...)
	if err != nil {
		return nil, fmt.Errorf("netlify deploy failed: %w (output: %s)", err, outStr)
	}

	var out struct {
		URL      string `json:"url"`
		DeployID string `json:"deploy_id"`
	}

	jsonStr, extractErr := extractJSONObject(outStr)
	if extractErr != nil {
		return nil, fmt.Errorf("failed to extract JSON from deploy output: %w\nOutput was: %s", extractErr, outStr)
	}

	if err := json.Unmarshal([]byte(jsonStr), &out); err != nil {
		return nil, fmt.Errorf("failed to parse netlify deploy json: %w\nOutput was: %s", err, outStr)
	}

	return &deploy.Deployment{
		URL:         out.URL,
		Provider:    "netlify",
		ProviderRef: out.DeployID,
		DeployedAt:  time.Now().UTC(),
	}, nil
}

func (n *NetlifyProvider) HealthCheck(ctx context.Context, d *deploy.Deployment) error {
	return deploy.HTTPHealthCheck(ctx, d.URL, 200)
}

func (n *NetlifyProvider) Rollback(ctx context.Context, to *deploy.Deployment, in deploy.DeployInput) error {
	if to.ProviderRef == "" {
		return fmt.Errorf("netlify rollback: no deploy_id (ProviderRef) found in previous deployment")
	}

	var proj struct {
		NetlifySiteID *string `json:"netlify_site_id"`
	}
	_ = state.LoadJSON(in.SessionDir, "project.json", &proj)

	siteID := ""
	if proj.NetlifySiteID != nil {
		siteID = *proj.NetlifySiteID
	}
	if siteID == "" {
		return fmt.Errorf("netlify rollback: no netlify_site_id found in project.json")
	}

	token := in.Token
	if token == "" {
		return fmt.Errorf("netlify rollback: no auth token provided")
	}

	url := fmt.Sprintf("https://api.netlify.com/api/v1/sites/%s/deploys/%s/restore", siteID, to.ProviderRef)
	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return fmt.Errorf("netlify rollback: failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("netlify rollback: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("netlify rollback: api returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func resolveNetlifySite(ctx context.Context, runner deploy.CommandRunner, workspaceRoot string, inToken string) (string, error) {
	baseName := filepath.Base(workspaceRoot)
	// try creating
	id, err := createSite(ctx, runner, workspaceRoot, baseName, inToken)
	if err == nil {
		return id, nil
	}

	// Check if collision
	if strings.Contains(err.Error(), "already taken") || strings.Contains(strings.ToLower(err.Error()), "taken") || err != nil {
		suffix := make([]byte, 2)
		rand.Read(suffix)
		newName := fmt.Sprintf("%s-%s", baseName, hex.EncodeToString(suffix))

		return createSite(ctx, runner, workspaceRoot, newName, inToken)
	}

	return "", err
}

func createSite(ctx context.Context, runner deploy.CommandRunner, workspaceRoot string, name string, token string) (string, error) {
	// First, fetch accounts to get the team slug. This fixes the "No teams available" error.
	accountArgs := []string{"api", "listAccountsForUser"}
	if token != "" {
		accountArgs = append(accountArgs, "--auth", token)
	}
	
	accountsOut, err := runner.Run(ctx, workspaceRoot, "netlify", accountArgs...)
	if err != nil {
		if strings.Contains(accountsOut, "Unauthorized") || strings.Contains(accountsOut, "401") {
			return "", fmt.Errorf("netlify authentication failed: Unauthorized. Please check if your provided auth token is valid. (Token used: %s)", token)
		}
	} else {
		start := strings.Index(accountsOut, "[\n")
		if start == -1 {
			start = strings.Index(accountsOut, "[ \n")
		}
		if start == -1 {
			start = strings.Index(accountsOut, "[")
		}
		end := strings.LastIndex(accountsOut, "\n]")
		if start != -1 && end != -1 && start < end {
			jsonStr := accountsOut[start : end+2]
			var accounts []struct {
				Slug string `json:"slug"`
			}
			if unmarshalErr := json.Unmarshal([]byte(jsonStr), &accounts); unmarshalErr == nil && len(accounts) > 0 {
				// Found an account slug, let's use it
				args := []string{"sites:create", "--name", name, "-a", accounts[0].Slug, "--json"}
				if token != "" {
					args = append(args, "--auth", token)
				}
				outStr, err := runner.Run(ctx, workspaceRoot, "netlify", args...)
				if err != nil {
					return "", fmt.Errorf("creation failed: %w (output: %s)", err, outStr)
				}
				return parseSiteCreationJSON(outStr)
			}
		}
	}

	args := []string{"sites:create", "--name", name, "--json"}
	if token != "" {
		args = append(args, "--auth", token)
	}
	outStr, err := runner.Run(ctx, workspaceRoot, "netlify", args...)
	if err != nil {
		return "", fmt.Errorf("creation failed: %w (output: %s)", err, outStr)
	}
	return parseSiteCreationJSON(outStr)
}

func parseSiteCreationJSON(outStr string) (string, error) {

	jsonStr, extractErr := extractJSONObject(outStr)
	if extractErr != nil {
		return "", fmt.Errorf("failed to extract JSON from sites:create output: %w\nOutput was: %s", extractErr, outStr)
	}

	var out struct {
		ID string `json:"id"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &out); err != nil {
		return "", fmt.Errorf("failed to parse sites:create json: %w\nOutput was: %s", err, outStr)
	}

	if out.ID == "" {
		return "", fmt.Errorf("creation successful but no site id returned. output: %s", outStr)
	}

	return out.ID, nil
}

func extractJSONObject(output string) (string, error) {
	start := strings.Index(output, "{")
	end := strings.LastIndex(output, "}")
	if start == -1 || end == -1 || start > end {
		return "", fmt.Errorf("no JSON object found in output")
	}
	return output[start : end+1], nil
}
