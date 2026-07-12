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
)

// Model is GoAI's resolved provider handle. Exposed so callers that need
// structured output (FixCode) can use it directly with GenerateStructured.
type Model = provider.LanguageModel

// ResolveModel maps Atlas's config.LLMProvider value to a GoAI provider model.
func ResolveModel(cfg *config.Config) (Model, error) {
	switch cfg.LLMProvider {
	case "anthropic":
		return anthropic.Chat(cfg.DefaultModel), nil
	case "openai":
		return openai.Chat(cfg.DefaultModel), nil
	case "gemini":
		return google.Chat(cfg.DefaultModel), nil
	case "mistral":
		return mistral.Chat(cfg.DefaultModel), nil
	case "groq":
		return groq.Chat(cfg.DefaultModel), nil
	case "grok":
		return xai.Chat(cfg.DefaultModel), nil
	case "local":
		return compat.Chat(cfg.DefaultModel, compat.WithBaseURL(cfg.LocalLLMBaseURL)), nil
	default:
		return nil, fmt.Errorf("unknown llm_provider: %s", cfg.LLMProvider)
	}
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

func NewClient(cfg *config.Config) (Client, error) {
	model, err := ResolveModel(cfg)
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
func GenerateStructured[T any](ctx context.Context, model Model, systemPrompt, userPrompt string) (T, error) {
	var zero T
	result, err := goai.GenerateObject[T](ctx, model,
		goai.WithSystem(systemPrompt),
		goai.WithPrompt(userPrompt),
	)
	if err != nil {
		return zero, err
	}
	return result.Object, nil
}
