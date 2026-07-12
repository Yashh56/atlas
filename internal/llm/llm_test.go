package llm_test

import (
	"testing"

	"github.com/Yashh56/atlas/internal/config"
	"github.com/Yashh56/atlas/internal/llm"
)

func TestResolveModel_AllProviders(t *testing.T) {
	providers := []struct {
		name  string
		model string
	}{
		{"anthropic", "claude-sonnet-4-6"},
		{"openai", "gpt-4o"},
		{"gemini", "gemini-2.5-flash"},
		{"mistral", "mistral-large-latest"},
		{"groq", "llama-3.3-70b-versatile"},
		{"grok", "grok-3"},
		{"local", "llama3"},
	}

	for _, tt := range providers {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				LLMProvider:    tt.name,
				DefaultModel:   tt.model,
				LocalLLMBaseURL: "http://localhost:11434/v1",
			}
			model, err := llm.ResolveModel(cfg)
			if err != nil {
				t.Fatalf("ResolveModel(%q) returned error: %v", tt.name, err)
			}
			if model == nil {
				t.Fatalf("ResolveModel(%q) returned nil model", tt.name)
			}
			if model.ModelID() != tt.model {
				t.Errorf("expected ModelID %q, got %q", tt.model, model.ModelID())
			}
		})
	}
}

func TestResolveModel_UnknownProvider(t *testing.T) {
	cfg := &config.Config{
		LLMProvider:  "nonexistent",
		DefaultModel: "some-model",
	}
	_, err := llm.ResolveModel(cfg)
	if err == nil {
		t.Fatal("expected error for unknown provider, got nil")
	}
}

func TestNewClient_ReturnsClient(t *testing.T) {
	cfg := &config.Config{
		LLMProvider:  "anthropic",
		DefaultModel: "claude-sonnet-4-6",
	}
	client, err := llm.NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	if client.Name() != "anthropic" {
		t.Errorf("expected Name() = 'anthropic', got %q", client.Name())
	}
}
