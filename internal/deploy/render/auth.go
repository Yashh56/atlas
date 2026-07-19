package render

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/Yashh56/atlas/internal/cliutil"
	"github.com/Yashh56/atlas/internal/config"
	"github.com/Yashh56/atlas/internal/credentials"
	"github.com/Yashh56/atlas/internal/deploy"
)

func EnsureRenderAuth(ctx context.Context, store *credentials.Store, cfg *config.Config, runner deploy.CommandRunner) error {
	return EnsureRenderAuthFull(ctx, store, cfg, runner, os.Stdin, os.Stdout)
}

func EnsureRenderAuthFull(
	ctx context.Context,
	store *credentials.Store,
	cfg *config.Config,
	runner deploy.CommandRunner,
	stdin io.Reader,
	stdout io.Writer,
) error {
	// 1 & 2: Env var and Stored token check
	token, err := deploy.EnsureProviderAuth("render", "RENDER_TOKEN", store, stdout)
	if err != nil {
		return err
	}

	if token == "" {
		fmt.Fprintln(stdout, "→ Checking Render authentication...")
		fmt.Fprintln(stdout, "  Not logged in to Render. Need a personal API key.")

		if stdin == os.Stdin {
			token, err = cliutil.PromptSecret("Render API Key")
		} else {
			token, err = cliutil.PromptSecretFromReaderForTest("Render API Key", stdin)
		}
		if err != nil {
			return err
		}

		if store != nil {
			if err := store.SetSecret("render", token); err != nil {
				return fmt.Errorf("storing secret: %w", err)
			}
			_ = store.SetMeta(credentials.ProviderCredential{
				Provider:   "render",
				Method:     credentials.MethodStoredToken,
				VerifiedAt: time.Now().UTC(),
			})
		}
	}

	// 3: Verify the token by calling /v1/owners and store ownerId if we don't have it
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.render.com/v1/owners", nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("calling Render API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Render API rejected token (status %d): %s", resp.StatusCode, string(body))
	}

	var owners []struct {
		Owner struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		} `json:"owner"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&owners); err != nil {
		return fmt.Errorf("decoding Render owners: %w", err)
	}

	if len(owners) == 0 {
		return fmt.Errorf("Render API returned no owners for this token")
	}

	// For now, we take the first owner.
	ownerID := owners[0].Owner.ID
	email := owners[0].Owner.Email

	// In the real code we should store ownerID in project.json, but here we just have config.Config.
	// For now, we'll let the Provider.Deploy handle saving it to project.json, or we can just return success.
	fmt.Fprintf(stdout, "✓ Render authenticated (%s, owner: %s)\n", email, ownerID)
	return nil
}
