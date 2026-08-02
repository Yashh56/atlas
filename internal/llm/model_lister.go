package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ModelLister is an optional capability for LLM clients that can enumerate
// available models from the provider's API. Not all providers implement this.
// Callers must check with a type assertion:
//
//	if lister, ok := llm.NewModelLister(provider, apiKey); ok {
//	    models, err := lister.ListModels(ctx)
//	}
type ModelLister interface {
	ListModels(ctx context.Context) ([]string, error)
}

// openAIListResponse is the standard shape returned by OpenAI-compatible
// /models endpoints and the Anthropic /v1/models endpoint (same .data[].id shape,
// different auth headers — verified against anthropic-sdk-go/model.go source).
type openAIListResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

// openaiCompatLister implements ModelLister for any provider that exposes
// GET {baseURL}/models with Authorization: Bearer <key> and returns the
// standard OpenAI-compat response shape.
type openaiCompatLister struct {
	baseURL string
	apiKey  string
	client  *http.Client
	isLocal bool
}

func (l *openaiCompatLister) ListModels(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, l.baseURL+"/models", nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+l.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := l.client.Do(req)
	if err != nil {
		if l.isLocal {
			return nil, fmt.Errorf("no local model runtime found at %s — is Ollama running?", l.baseURL)
		}
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("provider returned %d: %s", resp.StatusCode, string(body))
	}

	var result openAIListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	ids := make([]string, 0, len(result.Data))
	for _, m := range result.Data {
		if m.ID != "" {
			ids = append(ids, m.ID)
		}
	}
	return ids, nil
}

// anthropicLister implements ModelLister for Anthropic's /v1/models endpoint.
// Auth differs from OpenAI-compat: x-api-key header + anthropic-version header.
// Response shape is identical: {"data":[{"id":"..."},...]}
// Source: https://github.com/anthropics/anthropic-sdk-go/blob/main/model.go
type anthropicLister struct {
	apiKey string
	client *http.Client
}

func (l *anthropicLister) ListModels(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://api.anthropic.com/v1/models", nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("x-api-key", l.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Accept", "application/json")

	resp, err := l.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("anthropic returned %d: %s", resp.StatusCode, string(body))
	}

	var result openAIListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	ids := make([]string, 0, len(result.Data))
	for _, m := range result.Data {
		if m.ID != "" {
			ids = append(ids, m.ID)
		}
	}
	return ids, nil
}

// providerBaseURLs maps Atlas provider names to their OpenAI-compat base URLs.
// Gemini's compatibility shim may not expose /models uniformly — callers must
// handle failure gracefully (atlas models falls back to static display on error).
var providerBaseURLs = map[string]string{
	"openai":  "https://api.openai.com/v1",
	"groq":    "https://api.groq.com/openai/v1",
	"grok":    "https://api.x.ai/v1",
	"mistral": "https://api.mistral.ai/v1",
	"gemini":  "https://generativelanguage.googleapis.com/v1beta/openai",
}

// NewModelLister returns a ModelLister for providers that support live model
// enumeration, along with true. Returns (nil, false) for unsupported providers.
func NewModelLister(provider, apiKey, localBaseURL string) (ModelLister, bool) {
	httpClient := &http.Client{Timeout: 10 * time.Second}

	if provider == "local" {
		return &openaiCompatLister{baseURL: localBaseURL, apiKey: "", client: httpClient, isLocal: true}, true
	}

	if provider == "anthropic" {
		return &anthropicLister{apiKey: apiKey, client: httpClient}, true
	}

	if baseURL, ok := providerBaseURLs[provider]; ok {
		return &openaiCompatLister{baseURL: baseURL, apiKey: apiKey, client: httpClient}, true
	}

	return nil, false
}
