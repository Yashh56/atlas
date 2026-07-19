package deploy

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Yashh56/atlas/internal/credentials"
)

// EnsureProviderAuth checks for authentication via environment variable or stored token.
// It returns the resolved token if found, or an empty string if not found.
// This is used as the first step for all providers before falling back to provider-specific auth (like CLI).
func EnsureProviderAuth(
	providerName string,
	envVar string,
	store *credentials.Store,
	stdout io.Writer,
) (string, error) {
	titleName := strings.Title(providerName)

	// 1. Check env var first.
	if envVal := os.Getenv(envVar); envVal != "" {
		fmt.Fprintf(stdout, "✓ %s authenticated (env_var, %s)\n", titleName, envVar)
		return envVal, nil
	}

	// 2. Check stored token.
	if store != nil {
		if meta, ok, err := store.GetMeta(providerName); err == nil && ok {
			if meta.Method == credentials.MethodStoredToken {
				token, err := store.GetSecret(providerName)
				if err == nil && token != "" {
					account := meta.Account
					if account == "" {
						account = "unknown"
					}
					fmt.Fprintf(stdout, "✓ %s authenticated (stored_token, %s)\n", titleName, account)
					return token, nil
				}
			}
		}
	}

	return "", nil
}

