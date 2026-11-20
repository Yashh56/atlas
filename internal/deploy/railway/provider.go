package railway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Yashh56/atlas/internal/deploy"
	"github.com/Yashh56/atlas/internal/state"
)

// RailwayProvider implements deploy.Provider for Railway.
type RailwayProvider struct {
	Runner       deploy.CommandRunner
	PollInterval time.Duration // default 4s, overridable in tests
	PollTimeout  time.Duration // default 3m, overridable in tests
	HTTPClient   *http.Client  // default http.DefaultClient, overridable in tests
}

func (p *RailwayProvider) runner() deploy.CommandRunner {
	if p.Runner == nil {
		return deploy.OSCommandRunner{}
	}
	return p.Runner
}

func (p *RailwayProvider) pollInterval() time.Duration {
	if p.PollInterval > 0 {
		return p.PollInterval
	}
	return 4 * time.Second
}

func (p *RailwayProvider) pollTimeout() time.Duration {
	if p.PollTimeout > 0 {
		return p.PollTimeout
	}
	return 3 * time.Minute
}

func (p *RailwayProvider) httpClient() *http.Client {
	if p.HTTPClient != nil {
		return p.HTTPClient
	}
	return http.DefaultClient
}

func (p *RailwayProvider) Name() string { return "railway" }

type projectData struct {
	Framework        *string `json:"framework,omitempty"`
	PackageManager   *string `json:"package_manager,omitempty"`
	RailwayProjectID *string `json:"railway_project_id,omitempty"`
	RailwayServiceID *string `json:"railway_service_id,omitempty"`
	RequiresDatabase *bool   `json:"requires_database,omitempty"`
	DjangoModule     *string `json:"django_module,omitempty"`
}

func (p *RailwayProvider) Deploy(ctx context.Context, in deploy.DeployInput) (*deploy.Deployment, error) {
	var proj projectData
	if err := state.LoadJSON(in.SessionDir, "project.json", &proj); err != nil {
		_ = state.LoadJSON(filepath.Join(in.WorkspaceRoot, ".atlas"), "project.json", &proj)
	}

	// 1. Linking check via railway status
	statusOut, statusErr := p.runner().Run(ctx, in.WorkspaceRoot, "railway", "status")
	isUnlinked := statusErr != nil ||
		strings.Contains(strings.ToLower(statusOut), "not linked") ||
		strings.Contains(strings.ToLower(statusOut), "no project linked") ||
		strings.Contains(strings.ToLower(statusOut), "no project found")

	if isUnlinked {
		if proj.RailwayProjectID != nil && *proj.RailwayProjectID != "" {
			if _, err := p.runner().Run(ctx, in.WorkspaceRoot, "railway", "link", "-p", *proj.RailwayProjectID); err != nil {
				projName := filepath.Base(in.WorkspaceRoot)
				initOut, err := p.runner().Run(ctx, in.WorkspaceRoot, "railway", "init", "-n", projName)
				if err != nil {
					return nil, fmt.Errorf("railway init failed: %w (output: %s)", err, initOut)
				}
				if id := extractRailwayProjectID(initOut); id != "" {
					proj.RailwayProjectID = &id
				}
			}
		} else {
			projName := filepath.Base(in.WorkspaceRoot)
			initOut, err := p.runner().Run(ctx, in.WorkspaceRoot, "railway", "init", "-n", projName)
			if err != nil {
				return nil, fmt.Errorf("railway init failed: %w (output: %s)", err, initOut)
			}
			if id := extractRailwayProjectID(initOut); id != "" {
				proj.RailwayProjectID = &id
			}
		}
	} else {
		if id := extractRailwayProjectID(statusOut); id != "" && (proj.RailwayProjectID == nil || *proj.RailwayProjectID == "") {
			proj.RailwayProjectID = &id
		}
	}

	// 2. Database provisioning if required (PostgreSQL parity with Render)
	needsDB := false
	if proj.RequiresDatabase != nil && *proj.RequiresDatabase {
		needsDB = true
	}
	if proj.Framework != nil && *proj.Framework == "django" {
		needsDB = true
	}

	if needsDB {
		addOut, err := p.runner().Run(ctx, in.WorkspaceRoot, "railway", "add", "--database", "postgres")
		if err != nil {
			lower := strings.ToLower(addOut)
			if !strings.Contains(lower, "already") && !strings.Contains(lower, "exists") {
				return nil, fmt.Errorf("railway add database failed: %w (output: %s)", err, addOut)
			}
		}
	}

	// 3. Deploy via railway up --detach
	upArgs := []string{"up", "--detach"}
	if proj.RailwayServiceID != nil && *proj.RailwayServiceID != "" {
		upArgs = append(upArgs, "--service", *proj.RailwayServiceID)
	}

	if in.Token != "" && os.Getenv("RAILWAY_TOKEN") == "" {
		_ = os.Setenv("RAILWAY_TOKEN", in.Token)
	}

	upOut, err := p.runner().Run(ctx, in.WorkspaceRoot, "railway", upArgs...)
	if err != nil {
		return nil, fmt.Errorf("railway deploy failed: %w (output: %s)", err, upOut)
	}

	// 4. Ensure public domain
	domainOut, err := p.runner().Run(ctx, in.WorkspaceRoot, "railway", "domain")
	if err != nil {
		domainOut, _ = p.runner().Run(ctx, in.WorkspaceRoot, "railway", "domain", "list")
	}

	deployURL, err := parseRailwayDomain(domainOut)
	if err != nil {
		if u, fallbackErr := parseRailwayDomain(upOut); fallbackErr == nil {
			deployURL = u
		} else {
			return nil, fmt.Errorf("failed to resolve railway public domain: %w (domain output: %s)", err, domainOut)
		}
	}

	// 5. Wait for deployment readiness
	if err := p.waitForDeploymentReady(ctx, in.WorkspaceRoot, deployURL); err != nil {
		return nil, err
	}

	// Save state
	_ = state.SaveJSON(in.SessionDir, "project.json", &proj)
	_ = state.SaveJSON(filepath.Join(in.WorkspaceRoot, ".atlas"), "project.json", &proj)

	return &deploy.Deployment{
		URL:           deployURL,
		Provider:      "railway",
		ProviderRef:   deployURL,
		DeployedAt:    time.Now().UTC(),
		WorkspaceRoot: in.WorkspaceRoot,
	}, nil
}

func (p *RailwayProvider) waitForDeploymentReady(ctx context.Context, wsRoot, deployURL string) error {
	timeout := p.pollTimeout()
	interval := p.pollInterval()

	pollCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client := p.httpClient()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	printedDot := false
	for {
		// 1. Probe HTTP GET on deployURL
		req, err := http.NewRequestWithContext(pollCtx, http.MethodGet, deployURL, nil)
		if err == nil {
			resp, httpErr := client.Do(req)
			if httpErr == nil {
				_ = resp.Body.Close()
				if resp.StatusCode < 400 {
					if printedDot {
						fmt.Println()
					}
					return nil
				}
			}
		}

		// 2. Check railway status or railway deployment list
		statusOut, _ := p.runner().Run(pollCtx, wsRoot, "railway", "status")
		upperStatus := strings.ToUpper(statusOut)
		if strings.Contains(upperStatus, "CRASHED") || strings.Contains(upperStatus, "FAILED") {
			if printedDot {
				fmt.Println()
			}
			return fmt.Errorf("railway deployment failed in cloud: %s", statusOut)
		}
		if strings.Contains(upperStatus, "SUCCESS") {
			if printedDot {
				fmt.Println()
			}
			return nil
		}

		fmt.Print(".")
		printedDot = true

		select {
		case <-ctx.Done():
			if printedDot {
				fmt.Println()
			}
			return ctx.Err()
		case <-pollCtx.Done():
			if printedDot {
				fmt.Println()
			}
			// If timeout reached: log warning and continue so HealthCheck gets a chance
			fmt.Printf("Warning: railway deployment readiness polling timed out after %v; proceeding to healthcheck...\n", timeout)
			return nil
		case <-ticker.C:
		}
	}
}

var domainRegex = regexp.MustCompile(`https?://[a-zA-Z0-9.-]+\.up\.railway\.app[^\s]*`)
var nakedDomainRegex = regexp.MustCompile(`[a-zA-Z0-9.-]+\.up\.railway\.app`)
var generalURLRegex = regexp.MustCompile(`https?://[a-zA-Z0-9.-]+(?:/[^\s]*)?`)

func parseRailwayDomain(output string) (string, error) {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return "", fmt.Errorf("empty domain output")
	}

	// Try JSON first: {"domain": "..."}
	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start != -1 && end != -1 && start < end {
		var d struct {
			Domain string `json:"domain"`
			URL    string `json:"url"`
		}
		if err := json.Unmarshal([]byte(trimmed[start:end+1]), &d); err == nil {
			if d.URL != "" {
				return ensureHTTPS(d.URL), nil
			}
			if d.Domain != "" {
				return ensureHTTPS(d.Domain), nil
			}
		}
	}

	if m := domainRegex.FindString(trimmed); m != "" {
		return ensureHTTPS(m), nil
	}
	if m := nakedDomainRegex.FindString(trimmed); m != "" {
		return ensureHTTPS(m), nil
	}
	if m := generalURLRegex.FindString(trimmed); m != "" {
		return ensureHTTPS(m), nil
	}

	lines := strings.Split(trimmed, "\n")
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if strings.Contains(l, ".") && !strings.Contains(l, " ") {
			return ensureHTTPS(l), nil
		}
	}

	return "", fmt.Errorf("no valid domain found in output: %s", output)
}

func ensureHTTPS(url string) string {
	url = strings.TrimSpace(url)
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return url
	}
	return "https://" + url
}

var projectIDRegex = regexp.MustCompile(`(?:Project ID|project[_\s]id|Project:)\s*[:=]?\s*([a-zA-Z0-9_\-]+)`)

func extractRailwayProjectID(output string) string {
	if m := projectIDRegex.FindStringSubmatch(output); len(m) > 1 {
		return m[1]
	}
	return ""
}

func (p *RailwayProvider) HealthCheck(ctx context.Context, d *deploy.Deployment) error {
	if d == nil || d.URL == "" {
		return fmt.Errorf("railway healthcheck: missing deployment URL")
	}
	return deploy.HTTPHealthCheck(ctx, d.URL, 200)
}

func (p *RailwayProvider) Rollback(ctx context.Context, to *deploy.Deployment, in deploy.DeployInput) error {
	out, err := p.runner().Run(ctx, in.WorkspaceRoot, "railway", "redeploy", "-y")
	if err != nil {
		out2, err2 := p.runner().Run(ctx, in.WorkspaceRoot, "railway", "redeploy")
		if err2 != nil {
			return fmt.Errorf("railway rollback failed: %w (output: %s / %s)", err, out, out2)
		}
	}
	return nil
}
