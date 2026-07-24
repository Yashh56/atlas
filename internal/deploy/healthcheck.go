package deploy

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// HTTPHealthCheck performs an HTTP GET request to the given URL and verifies
// that the status code matches expectedStatus. It retries on failure to handle
// cold starts or brief DNS propagation delays.
func HTTPHealthCheck(ctx context.Context, url string, expectedStatus int) error {
	if url == "" {
		return fmt.Errorf("healthcheck: URL is empty")
	}
	
	// Add http prefix if missing (shouldn't be, but safe)
	if len(url) < 4 || url[:4] != "http" {
		url = "https://" + url
	}

	maxRetries := 5
	baseBackoff := 2 * time.Second

	for i := 0; i < maxRetries; i++ {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return fmt.Errorf("healthcheck: failed to create request: %w", err)
		}

		resp, err := http.DefaultClient.Do(req)
		
		// If network error or context canceled, evaluate if we can retry
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			// Wait and retry
			time.Sleep(baseBackoff * time.Duration(i+1))
			continue
		}
		
		defer resp.Body.Close()

		if resp.StatusCode == expectedStatus {
			return nil // Healthy!
		}

		// 5xx errors or 404 Not Found often recover during cold starts/propagation, wait and retry.
		if (resp.StatusCode >= 500 && resp.StatusCode < 600) || resp.StatusCode == 404 {
			time.Sleep(baseBackoff * time.Duration(i+1))
			continue
		}

		// Other 4xx or unexpected status -> immediately unhealthy
		return fmt.Errorf("healthcheck: expected status %d, got %d", expectedStatus, resp.StatusCode)
	}

	return fmt.Errorf("healthcheck: failed after %d retries, unable to get status %d from %s", maxRetries, expectedStatus, url)
}
