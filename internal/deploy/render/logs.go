package render

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"time"

	"github.com/Yashh56/atlas/internal/credentials"
	"github.com/Yashh56/atlas/internal/deploy"
	"github.com/Yashh56/atlas/internal/state"
)

// Logs streams runtime logs from Render.
func (r *RenderProvider) Logs(ctx context.Context, d *deploy.Deployment, w io.Writer, follow bool) error {
	var proj struct {
		RenderServiceID string `json:"render_service_id"`
	}
	if err := state.LoadJSON(filepath.Join(d.WorkspaceRoot, ".atlas"), "project.json", &proj); err != nil {
		return fmt.Errorf("could not load project config: %w", err)
	}

	serviceID := proj.RenderServiceID
	if serviceID == "" {
		return fmt.Errorf("render_service_id not found in project state")
	}

	store, _ := credentials.Open()
	token, err := deploy.EnsureProviderAuth("render", "RENDER_TOKEN", store, io.Discard)
	if err != nil {
		return err
	}

	baseURL := r.BaseURL
	if baseURL == "" {
		baseURL = "https://api.render.com"
	}

	// First, fetch the owner ID for the user (we need ownerId for the logs query)
	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/v1/owners", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch render owners: status %d", resp.StatusCode)
	}

	var owners []struct {
		Owner struct {
			ID string `json:"id"`
		} `json:"owner"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&owners); err != nil {
		return err
	}
	if len(owners) == 0 {
		return fmt.Errorf("no render owners found for token")
	}
	ownerID := owners[0].Owner.ID

	var startTime string

	for {
		reqURL := fmt.Sprintf("%s/v1/logs?resource=%s&ownerId=%s&direction=forward&limit=100", baseURL, serviceID, ownerID)
		if startTime != "" {
			reqURL += "&startTime=" + startTime
		}

		req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
		if err != nil {
			return fmt.Errorf("render logs: create request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("render logs: request failed: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return fmt.Errorf("render logs: bad status %d", resp.StatusCode)
		}

		var page struct {
			HasMore       bool   `json:"hasMore"`
			NextStartTime string `json:"nextStartTime"`
			Logs          []struct {
				Timestamp string `json:"timestamp"`
				Message   string `json:"message"`
			} `json:"logs"`
		}
		err = json.NewDecoder(resp.Body).Decode(&page)
		resp.Body.Close()
		if err != nil {
			return fmt.Errorf("render logs: parse response: %w", err)
		}

		for _, item := range page.Logs {
			fmt.Fprintln(w, item.Message)
			// Keep track of the last timestamp in case nextStartTime is empty
			startTime = item.Timestamp
		}

		if page.NextStartTime != "" {
			startTime = page.NextStartTime
		}

		if !page.HasMore {
			if !follow {
				break
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(2 * time.Second):
				// Poll again
			}
		}
	}

	return nil
}
