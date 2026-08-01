package render

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/Yashh56/atlas/internal/build"
	"github.com/Yashh56/atlas/internal/credentials"
	"github.com/Yashh56/atlas/internal/deploy"
	"github.com/Yashh56/atlas/internal/state"
)

type RenderProvider struct {
	BaseURL      string        // exposed for testing
	TickInterval time.Duration // exposed for testing
}

func (r *RenderProvider) Name() string { return "render" }

func (r *RenderProvider) Deploy(ctx context.Context, in deploy.DeployInput) (*deploy.Deployment, error) {
	var proj struct {
		RenderServiceID *string `json:"render_service_id"`
		Git             struct {
			CommitSHA *string `json:"commit_sha"`
			Branch    *string `json:"branch"`
			Remote    *string `json:"remote"`
		} `json:"git"`
		Framework      *string `json:"framework"`
		PackageManager *string `json:"package_manager"`
	}
	_ = state.LoadJSON(in.SessionDir, "project.json", &proj)

	token := in.Token
	if token == "" {
		token = os.Getenv("RENDER_TOKEN")
		if token == "" {
			store, storeErr := credentials.Open()
			if storeErr == nil && store != nil {
				token, _ = store.GetSecret("render")
			}
		}
	}
	if token == "" {
		return nil, fmt.Errorf("render deploy: unauthorized. No RENDER_TOKEN or stored credential found")
	}

	commitSHA := ""
	if proj.Git.CommitSHA != nil {
		commitSHA = *proj.Git.CommitSHA
	}
	branch := "main"
	if proj.Git.Branch != nil {
		branch = *proj.Git.Branch
	}
	remote := ""
	if proj.Git.Remote != nil {
		remote = *proj.Git.Remote
	}

	var serviceID string
	if proj.RenderServiceID == nil || *proj.RenderServiceID == "" {
		// Attempt to create the service
		newServiceID, err := r.createService(ctx, token, remote, branch, proj.Framework, proj.PackageManager, in.WorkspaceRoot)
		if err != nil {
			return nil, fmt.Errorf("render deploy: failed to automatically create service: %w", err)
		}
		serviceID = newServiceID
		// Save it back to project.json
		proj.RenderServiceID = &serviceID
		if saveErr := state.SaveJSON(filepath.Join(in.WorkspaceRoot, ".atlas"), "project.json", &proj); saveErr != nil {
			fmt.Printf("Warning: failed to save newly created Render Service ID to project.json: %v\n", saveErr)
		}
	} else {
		serviceID = *proj.RenderServiceID
	}

	// 1. Trigger the deploy
	payload := map[string]string{
		"clearCache": "do_not_clear",
	}
	if commitSHA != "" {
		payload["commitId"] = commitSHA
	}

	bodyData, _ := json.Marshal(payload)

	baseURL := r.BaseURL
	if baseURL == "" {
		baseURL = "https://api.render.com"
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/v1/services/%s/deploys", baseURL, serviceID), bytes.NewReader(bodyData))
	if err != nil {
		return nil, fmt.Errorf("render deploy: creating trigger request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("render deploy: triggering deploy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("render deploy: trigger failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var triggerResult struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&triggerResult); err != nil {
		return nil, fmt.Errorf("render deploy: decoding trigger response: %w", err)
	}

	deployID := triggerResult.ID
	if deployID == "" {
		return nil, fmt.Errorf("render deploy: no deploy ID returned")
	}

	// 2. Poll for status
	timeout := time.After(5 * time.Minute)
	interval := r.TickInterval
	if interval == 0 {
		interval = 5 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timeout:
			return nil, fmt.Errorf("render deploy: timed out after 5 minutes waiting for deploy %s", deployID)
		case <-ticker.C:
			baseURL := r.BaseURL
			if baseURL == "" {
				baseURL = "https://api.render.com"
			}
			statusReq, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/v1/services/%s/deploys/%s", baseURL, serviceID, deployID), nil)
			if err != nil {
				return nil, fmt.Errorf("render deploy: creating status request: %w", err)
			}
			statusReq.Header.Set("Authorization", "Bearer "+token)
			statusReq.Header.Set("Accept", "application/json")

			statusResp, err := http.DefaultClient.Do(statusReq)
			if err != nil {
				return nil, fmt.Errorf("render deploy: checking status: %w", err)
			}

			if statusResp.StatusCode >= 400 {
				statusResp.Body.Close()
				return nil, fmt.Errorf("render deploy: status check failed with %d", statusResp.StatusCode)
			}

			var statusResult struct {
				Status string `json:"status"`
			}
			err = json.NewDecoder(statusResp.Body).Decode(&statusResult)
			statusResp.Body.Close()
			if err != nil {
				return nil, fmt.Errorf("render deploy: decoding status: %w", err)
			}

			switch statusResult.Status {
			case "live":
				fmt.Println() // Complete the line of dots
				return r.fetchServiceURL(ctx, serviceID, token, commitSHA)
			case "build_failed", "update_failed", "canceled", "deactivated":
				fmt.Println() // Complete the line of dots
				return nil, fmt.Errorf("render deploy: deploy failed with status %q", statusResult.Status)
			default:
				// Print visual feedback while polling
				fmt.Print(".")
			}
		}
	}
}

func (r *RenderProvider) fetchServiceURL(ctx context.Context, serviceID, token, commitSHA string) (*deploy.Deployment, error) {
	baseURL := r.BaseURL
	if baseURL == "" {
		baseURL = "https://api.render.com"
	}
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/v1/services/%s", baseURL, serviceID), nil)
	if err != nil {
		return nil, fmt.Errorf("render deploy: creating fetch service request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("render deploy: fetching service details: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("render deploy: fetching service details failed with status %d", resp.StatusCode)
	}

	var serviceResp struct {
		ServiceDetails map[string]interface{} `json:"serviceDetails"`
		Service        *struct {
			ServiceDetails map[string]interface{} `json:"serviceDetails"`
		} `json:"service"`
	}
	
	bodyBytes, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(bodyBytes, &serviceResp); err != nil {
		return nil, fmt.Errorf("render deploy: decoding service details: %w", err)
	}

	url := ""
	
	// Check if wrapped in "service"
	details := serviceResp.ServiceDetails
	if serviceResp.Service != nil && serviceResp.Service.ServiceDetails != nil {
		details = serviceResp.Service.ServiceDetails
	}
	
	if u, exists := details["url"].(string); exists && u != "" {
		url = u
	} else {
		// Fallback for older API versions where url might be nested
		for _, v := range details {
			if detailMap, ok := v.(map[string]interface{}); ok {
				if nestedU, exists := detailMap["url"].(string); exists {
					url = nestedU
					break
				}
			}
		}
	}

	if url == "" {
		return nil, fmt.Errorf("render deploy: could not find URL in service details. Response was: %s", string(bodyBytes))
	}



	return &deploy.Deployment{
		URL:         url,
		Provider:    "render",
		ProviderRef: commitSHA, // Guaranteed to be set by pre-deploy checks
		DeployedAt:  time.Now().UTC(),
	}, nil
}

func (r *RenderProvider) HealthCheck(ctx context.Context, d *deploy.Deployment) error {
	return deploy.HTTPHealthCheck(ctx, d.URL, 200)
}

func (r *RenderProvider) Rollback(ctx context.Context, to *deploy.Deployment, in deploy.DeployInput) error {
	var proj struct {
		RenderServiceID *string `json:"render_service_id"`
	}
	_ = state.LoadJSON(in.SessionDir, "project.json", &proj)

	serviceID := ""
	if proj.RenderServiceID != nil {
		serviceID = *proj.RenderServiceID
	}
	if serviceID == "" {
		return fmt.Errorf("render rollback: no render_service_id found in project.json")
	}

	token := in.Token
	if token == "" {
		return fmt.Errorf("render rollback: unauthorized. No token provided")
	}

	if to.ProviderRef == "" {
		return fmt.Errorf("render rollback: missing ProviderRef (commit_sha) on deployment")
	}

	payload := map[string]string{
		"commitId": to.ProviderRef,
		"clearCache": "do_not_clear",
	}

	bodyData, _ := json.Marshal(payload)

	baseURL := r.BaseURL
	if baseURL == "" {
		baseURL = "https://api.render.com"
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/v1/services/%s/deploys", baseURL, serviceID), bytes.NewReader(bodyData))
	if err != nil {
		return fmt.Errorf("render rollback: creating trigger request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("render rollback: triggering deploy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("render rollback: trigger failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var triggerResult struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&triggerResult); err != nil {
		return fmt.Errorf("render rollback: decoding trigger response: %w", err)
	}

	deployID := triggerResult.ID
	if deployID == "" {
		return fmt.Errorf("render rollback: no deploy ID returned")
	}

	// Poll for status
	timeout := time.After(5 * time.Minute)
	interval := r.TickInterval
	if interval == 0 {
		interval = 5 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("render rollback: timed out after 5 minutes waiting for deploy %s", deployID)
		case <-ticker.C:
			statusReq, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/v1/services/%s/deploys/%s", baseURL, serviceID, deployID), nil)
			if err != nil {
				return fmt.Errorf("render rollback: creating status request: %w", err)
			}
			statusReq.Header.Set("Authorization", "Bearer "+token)
			statusReq.Header.Set("Accept", "application/json")

			statusResp, err := http.DefaultClient.Do(statusReq)
			if err != nil {
				return fmt.Errorf("render rollback: checking status: %w", err)
			}

			if statusResp.StatusCode >= 400 {
				statusResp.Body.Close()
				return fmt.Errorf("render rollback: status check failed with %d", statusResp.StatusCode)
			}

			var statusResult struct {
				Status string `json:"status"`
			}
			err = json.NewDecoder(statusResp.Body).Decode(&statusResult)
			statusResp.Body.Close()
			if err != nil {
				return fmt.Errorf("render rollback: decoding status: %w", err)
			}

			switch statusResult.Status {
			case "live":
				fmt.Println() // visual formatting
				return nil
			case "build_failed", "update_failed", "canceled", "deactivated":
				fmt.Println() // visual formatting
				return fmt.Errorf("render rollback: deploy failed with status %q", statusResult.Status)
			default:
				fmt.Print(".") // visual feedback
			}
		}
	}
}

func (r *RenderProvider) createService(ctx context.Context, token, remote, branch string, fw, pm *string, workspaceRoot string) (string, error) {
	if remote == "" {
		return "", fmt.Errorf("no remote Git repository configured. Please push your code to GitHub/GitLab first")
	}

	// Fetch owner ID
	baseURL := r.BaseURL
	if baseURL == "" {
		baseURL = "https://api.render.com"
	}
	reqOwner, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/v1/owners", nil)
	if err != nil {
		return "", err
	}
	reqOwner.Header.Set("Authorization", "Bearer "+token)
	reqOwner.Header.Set("Accept", "application/json")

	respOwner, err := http.DefaultClient.Do(reqOwner)
	if err != nil {
		return "", err
	}
	defer respOwner.Body.Close()

	if respOwner.StatusCode >= 400 {
		return "", fmt.Errorf("fetching owners failed with status %d", respOwner.StatusCode)
	}

	var owners []struct {
		Owner struct {
			ID string `json:"id"`
		} `json:"owner"`
	}
	if err := json.NewDecoder(respOwner.Body).Decode(&owners); err != nil {
		return "", err
	}
	if len(owners) == 0 {
		return "", fmt.Errorf("no Render owners found for this token")
	}
	ownerID := owners[0].Owner.ID

	// Determine service details based on heuristics
	serviceType := "web_service"
	buildCommand := "npm install && npm run build"
	startCommand := "npm start"
	publishPath := ""

	packageManager := "npm"
	if pm != nil && *pm != "" {
		packageManager = *pm
	}

	framework := ""
	if fw != nil {
		framework = *fw
	}

	// Determine service type and publish path
	switch framework {
	case "react", "vue":
		serviceType = "static_site"
		if p, err := build.ResolvePublishDir(framework, packageManager, workspaceRoot); err == nil {
			publishPath = p
		} else {
			publishPath = "dist"
		}
	case "nextjs", "express", "node", "go":
		serviceType = "web_service"
	}

	// Resolve build command using the shared table
	cmd, args, _ := build.ResolveBuildCommand(framework, packageManager)
	if cmd != "" {
		// Render needs a single shell string
		buildStr := cmd
		for _, arg := range args {
			buildStr += " " + arg
		}
		
		if cmd == "go" {
			buildCommand = "go build -o app"
		} else {
			// Prepend the install command for JS/TS
			buildCommand = fmt.Sprintf("%s install && %s", packageManager, buildStr)
		}
	}

	// Resolve start command
	if framework == "go" {
		startCommand = "./app"
	} else if framework == "nextjs" {
		startCommand = fmt.Sprintf("%s run start", packageManager)
	} else if framework == "express" || framework == "node" {
		startCommand = "node index.js"
	} else {
		startCommand = fmt.Sprintf("%s start", packageManager)
	}

	repoName := filepath.Base(remote)
	repoName = repoName[:len(repoName)-len(filepath.Ext(repoName))] // strip .git if present
	if repoName == "" {
		repoName = "atlas-service"
	}

	// Payload for POST /v1/services
	payload := map[string]interface{}{
		"type":    serviceType,
		"ownerId": ownerID,
		"name":    repoName,
		"repo":    remote, // e.g. https://github.com/user/repo
		"branch":  branch,
		"buildFilter": map[string]interface{}{
			"paths": []string{
				"src/**",
				"public/**",
				"package.json",
				"package-lock.json",
				"vite.config.js",
			},
			"ignoredPaths": []string{
				"README.md",
				".gitignore",
				"docs/**",
				".github/**",
			},
		},
	}

	serviceDetails := map[string]interface{}{}

	// Add type-specific details
	if serviceType == "static_site" {
		serviceDetails["buildCommand"] = buildCommand
		serviceDetails["publishPath"] = publishPath
	} else {
		// web_service configuration
		serviceDetails["pullRequestPreviewsEnabled"] = "no"
		serviceDetails["previews"] = map[string]string{"generation": "off"}
		serviceDetails["plan"] = "free"
		
		env := "node"
		if fw != nil && *fw == "go" {
			env = "go"
		}
		
		serviceDetails["runtime"] = env
		serviceDetails["env"] = env
		serviceDetails["envSpecificDetails"] = map[string]string{
			"buildCommand": buildCommand,
			"startCommand": startCommand,
		}
	}
	
	payload["serviceDetails"] = serviceDetails

	bodyData, _ := json.Marshal(payload)
	reqCreate, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/v1/services", bytes.NewReader(bodyData))
	if err != nil {
		return "", err
	}
	reqCreate.Header.Set("Authorization", "Bearer "+token)
	reqCreate.Header.Set("Content-Type", "application/json")
	reqCreate.Header.Set("Accept", "application/json")

	respCreate, err := http.DefaultClient.Do(reqCreate)
	if err != nil {
		return "", err
	}
	defer respCreate.Body.Close()

	if respCreate.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(respCreate.Body)
		return "", fmt.Errorf("failed to create service (status %d): %s\n\n⚠ WARNING: Ensure you have granted Render access to your GitHub repository in the Render Dashboard.", respCreate.StatusCode, string(bodyBytes))
	}

	var createResult struct {
		Service struct {
			ID string `json:"id"`
		} `json:"service"`
	}
	if err := json.NewDecoder(respCreate.Body).Decode(&createResult); err != nil {
		return "", err
	}
	if createResult.Service.ID == "" {
		return "", fmt.Errorf("no service ID returned from Render API")
	}

	return createResult.Service.ID, nil
}

