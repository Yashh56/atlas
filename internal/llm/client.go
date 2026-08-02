// Package llm provides multi-provider LLM clients for Atlas, built on GoAI v0.8.6.
//
// Verified API signatures via `go doc`:
//   - goai.GenerateText(ctx, model, ...Option) -> (*TextResult, error); result.Text
//   - goai.GenerateObject[T](ctx, model, ...Option) -> (*ObjectResult[T], error); result.Object
//   - goai.WithSystem(string), goai.WithPrompt(string) Option constructors
//   - provider.LanguageModel interface: ModelID(), DoGenerate(), DoStream()
//   - All providers: anthropic.Chat(), openai.Chat(), google.Chat(),
//     mistral.Chat(), groq.Chat(), xai.Chat(), compat.Chat() -> provider.LanguageModel
//   - compat.WithBaseURL(string) for local/custom endpoints
package llm

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/zendev-sh/goai"
	"github.com/zendev-sh/goai/provider"
	"github.com/zendev-sh/goai/provider/anthropic"
	"github.com/zendev-sh/goai/provider/compat"
	"github.com/zendev-sh/goai/provider/google"
	"github.com/zendev-sh/goai/provider/groq"
	"github.com/zendev-sh/goai/provider/mistral"
	"github.com/zendev-sh/goai/provider/openai"
	"github.com/zendev-sh/goai/provider/xai"

	"github.com/Yashh56/atlas/internal/config"
	"github.com/Yashh56/atlas/internal/credentials"
)

// Model is GoAI's resolved provider handle. Exposed so callers that need
// structured output (FixCode) can use it directly with GenerateStructured.
type Model = provider.LanguageModel

// resolveAPIKey tries env vars first, then the credential store.
func resolveAPIKey(store *credentials.Store, provider, envVar string) (string, error) {
	if v := os.Getenv(envVar); v != "" {
		return v, nil
	}
	if store != nil {
		if secret, err := store.GetSecret("llm:" + provider); err == nil && secret != "" {
			return secret, nil
		}
	}
	return "", fmt.Errorf("no API key for %s — set %s or run `atlas models set %s`", provider, envVar, provider)
}

// ResolveModel maps Atlas's config.LLMProvider value to a GoAI provider model.
func ResolveModel(cfg *config.Config, store *credentials.Store) (Model, error) {
	modelName := cfg.LLMProvider // user now selects exact model name here

	// Legacy fallback if someone still has a bare provider name in their config.
	// These are routing defaults only — not recommendations for what to use.
	// Model catalog display is handled separately in cmd/atlas/cmd/models.go.
	switch modelName {
	case "anthropic":
		modelName = "claude-sonnet-5"
	case "openai":
		modelName = "gpt-5.6-sol"
	case "gemini":
		modelName = "gemini-3.5-flash"
	case "mistral":
		modelName = "mistral-medium-3.5" // auto-updating alias, intentionally not a dated snapshot
	case "groq":
		modelName = "openai/gpt-oss-120b"
	case "grok":
		modelName = "grok-4.5"
	case "local":
		modelName = "llama3" // routing fallback only — local models are not statically enumerable
	}

	// Support cfg.DefaultModel if it's explicitly provided and valid
	if cfg.DefaultModel != "" && cfg.DefaultModel != "claude-sonnet-4-6" {
		modelName = cfg.DefaultModel
	}

	if modelName == "" {
		modelName = "claude-sonnet-5" // default when no provider configured
	}

	// Match by prefix or explicit name
	if strings.HasPrefix(modelName, "claude") {
		key, err := resolveAPIKey(store, "anthropic", "ANTHROPIC_API_KEY")
		if err != nil {
			return nil, err
		}
		os.Setenv("ANTHROPIC_API_KEY", key)
		return anthropic.Chat(modelName), nil
	}

	if strings.HasPrefix(modelName, "gpt-") || strings.HasPrefix(modelName, "o1-") {
		key, err := resolveAPIKey(store, "openai", "OPENAI_API_KEY")
		if err != nil {
			return nil, err
		}
		os.Setenv("OPENAI_API_KEY", key)
		return openai.Chat(modelName), nil
	}

	if strings.HasPrefix(modelName, "gemini") {
		key, err := resolveAPIKey(store, "gemini", "GEMINI_API_KEY")
		if err != nil {
			return nil, err
		}
		os.Setenv("GEMINI_API_KEY", key)
		return google.Chat(modelName), nil
	}

	if strings.HasPrefix(modelName, "mistral") || strings.HasPrefix(modelName, "pixtral") {
		key, err := resolveAPIKey(store, "mistral", "MISTRAL_API_KEY")
		if err != nil {
			return nil, err
		}
		os.Setenv("MISTRAL_API_KEY", key)
		return mistral.Chat(modelName), nil
	}

	if cfg.LLMProvider == "groq" || ((strings.HasPrefix(modelName, "llama-") || strings.HasPrefix(modelName, "llama3-") || strings.HasPrefix(modelName, "mixtral-")) && cfg.LLMProvider != "local" && modelName != "llama3") {
		key, err := resolveAPIKey(store, "groq", "GROQ_API_KEY")
		if err != nil {
			return nil, err
		}
		os.Setenv("GROQ_API_KEY", key)
		return groq.Chat(modelName), nil
	}

	if strings.HasPrefix(modelName, "grok") {
		key, err := resolveAPIKey(store, "grok", "XAI_API_KEY")
		if err != nil {
			return nil, err
		}
		os.Setenv("XAI_API_KEY", key)
		return xai.Chat(modelName), nil
	}

	if cfg.LLMProvider == "local" || modelName == "llama3" || strings.HasPrefix(modelName, "local") {
		actualModel := modelName
		if strings.HasPrefix(modelName, "local:") {
			actualModel = strings.TrimPrefix(modelName, "local:")
		}
		
		if actualModel == "local" || actualModel == "llama3" {
			actualModel = "llama3"
			// Try to auto-detect if possible to avoid failing when llama3 isn't pulled
			if lister, ok := NewModelLister("local", "", cfg.LocalLLMBaseURL); ok {
				if models, err := lister.ListModels(context.Background()); err == nil && len(models) > 0 {
					hasLlama3 := false
					for _, m := range models {
						if m == "llama3" {
							hasLlama3 = true
							break
						}
					}
					if !hasLlama3 {
						actualModel = models[0] // pick the first available local model
					}
				}
			}
		}

		return compat.Chat(actualModel, compat.WithBaseURL(cfg.LocalLLMBaseURL)), nil
	}

	return nil, fmt.Errorf("unknown LLM provider or model: %q", cfg.LLMProvider)
}

// Client covers plain-text completion — used anywhere structured output isn't needed.
type Client interface {
	Name() string
	Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}

type goaiClient struct {
	providerName string
	model        Model
}

func NewClient(cfg *config.Config, store *credentials.Store) (Client, error) {
	model, err := ResolveModel(cfg, store)
	if err != nil {
		return nil, err
	}
	return &goaiClient{providerName: cfg.LLMProvider, model: model}, nil
}

func (c *goaiClient) Name() string { return c.providerName }

func (c *goaiClient) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	result, err := goai.GenerateText(ctx, c.model,
		goai.WithSystem(systemPrompt),
		goai.WithPrompt(userPrompt),
	)
	if err != nil {
		return "", fmt.Errorf("goai generate text (%s): %w", c.providerName, err)
	}
	return result.Text, nil
}

// GenerateStructured is a free generic function, not a Client method, because
// Go interfaces can't have generic methods. FixCode calls this directly.
func GenerateStructured[T any](ctx context.Context, model Model, systemPrompt, userPrompt string, opts ...goai.Option) (T, *provider.Usage, error) {
	var zero T

	// Combine default prompts with user options
	allOpts := []goai.Option{
		goai.WithSystem(systemPrompt),
		goai.WithPrompt(userPrompt),
	}
	allOpts = append(allOpts, opts...)

	result, err := goai.GenerateObject[T](ctx, model, allOpts...)
	if err != nil {
		return zero, nil, err
	}
	return result.Object, &result.Usage, nil
}
