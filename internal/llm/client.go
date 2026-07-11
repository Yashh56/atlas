// Package llm provides multi-provider LLM clients for Atlas.
package llm

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/Yashh56/atlas/internal/config"
)

// Client is the interface for all LLM providers.
type Client interface {
	Name() string
	Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}

// httpClient is used to allow dependency injection for testing.
var defaultHTTPClient = http.DefaultClient

// NewClient creates a new LLM client based on the configuration.
func NewClient(cfg *config.Config) (Client, error) {
	if cfg.LLMProvider == "anthropic" {
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		return NewAnthropicClient(apiKey, cfg.DefaultModel, defaultHTTPClient), nil
	}

	preset, ok := presets[cfg.LLMProvider]
	if !ok && cfg.LLMProvider != "local" {
		return nil, fmt.Errorf("llm: unrecognized provider %q", cfg.LLMProvider)
	}

	baseURL := cfg.LocalLLMBaseURL
	apiKey := ""
	if cfg.LLMProvider != "local" {
		baseURL = preset.BaseURL
		apiKey = os.Getenv(preset.EnvVar)
	}

	return NewOpenAICompatClient(cfg.LLMProvider, baseURL, apiKey, cfg.DefaultModel, defaultHTTPClient), nil
}
