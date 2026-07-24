package netlify

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
		args = append(args, "--dir", outDir)
	}
	outStr, err := n.runner().Run(ctx, in.WorkspaceRoot, "netlify", args...)
	if err != nil {
		return nil, fmt.Errorf("netlify deploy failed: %w (output: %s)", err, outStr)
	}

	var out struct {
		URL string `json:"url"`
	}
	
	jsonStr, extractErr := extractJSONObject(outStr)
	if extractErr != nil {
		return nil, fmt.Errorf("failed to extract JSON from deploy output: %w\nOutput was: %s", extractErr, outStr)
	}

	if err := json.Unmarshal([]byte(jsonStr), &out); err != nil {
		return nil, fmt.Errorf("failed to parse netlify deploy json: %w\nOutput was: %s", err, outStr)
	}

	return &deploy.Deployment{
		URL:        out.URL,
		Provider:   "netlify",
		DeployedAt: time.Now().UTC(),
	}, nil
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
	args := []string{"sites:create", "--name", name, "--json"}
	if token != "" {
		args = append(args, "--auth", token)
	}
	outStr, err := runner.Run(ctx, workspaceRoot, "netlify", args...)
	if err != nil {
		return "", fmt.Errorf("creation failed: %w (output: %s)", err, outStr)
	}

	var out struct {
		ID string `json:"id"`
	}
	
	jsonStr, extractErr := extractJSONObject(outStr)
	if extractErr != nil {
		return "", fmt.Errorf("failed to extract JSON from sites:create output: %w\nOutput was: %s", extractErr, outStr)
	}

	if err := json.Unmarshal([]byte(jsonStr), &out); err != nil {
		return "", fmt.Errorf("failed to parse netlify sites:create json: %w", err)
	}
	if out.ID == "" {
		return "", fmt.Errorf("site ID was empty in json output")
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
