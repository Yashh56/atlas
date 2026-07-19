package llm_test

import (
	"os"
	"testing"

	"github.com/Yashh56/atlas/internal/config"
	"github.com/Yashh56/atlas/internal/credentials"
	"github.com/Yashh56/atlas/internal/llm"
)

func TestResolveModel_AllProviders(t *testing.T) {
	providers := []struct {
		name  string
		model string
	}{
		{"anthropic", "claude-3-5-sonnet-20240620"},
		{"openai", "gpt-4o"},
		{"gemini", "gemini-2.5-flash"},
		{"mistral", "mistral-large-latest"},
		{"groq", "llama-3.3-70b-versatile"},
		{"grok", "grok-3"},
		{"local", "llama3"},
	}

	envVars := map[string]string{
		"anthropic": "ANTHROPIC_API_KEY",
		"openai":    "OPENAI_API_KEY",
		"gemini":    "GEMINI_API_KEY",
		"mistral":   "MISTRAL_API_KEY",
		"groq":      "GROQ_API_KEY",
		"grok":      "XAI_API_KEY",
	}

	for _, tt := range providers {
		t.Run(tt.name, func(t *testing.T) {
			if envVar, ok := envVars[tt.name]; ok {
				t.Setenv(envVar, "dummy-key")
			}
			
			cfg := &config.Config{
				LLMProvider:     tt.name,
				DefaultModel:    tt.model,
				LocalLLMBaseURL: "http://localhost:11434/v1",
			}
			model, err := llm.ResolveModel(cfg, nil)
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
	_, err := llm.ResolveModel(cfg, nil)
	if err == nil {
		t.Fatal("expected error for unknown provider, got nil")
	}
}

func TestNewClient_ReturnsClient(t *testing.T) {
	cfg := &config.Config{
		LLMProvider:  "anthropic",
		DefaultModel: "claude-3-5-sonnet-20240620",
	}
	t.Setenv("ANTHROPIC_API_KEY", "dummy-key")
	
	client, err := llm.NewClient(cfg, nil)
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	if client.Name() != "anthropic" {
		t.Errorf("expected Name() = 'anthropic', got %q", client.Name())
	}
}

func TestResolveModel_WithStore(t *testing.T) {
	// Create a fresh store in a temp directory
	store, err := credentials.OpenWithDir(t.TempDir())
	if err != nil {
		t.Fatalf("opening test store: %v", err)
	}

	// Store a test key
	err = store.SetSecret("llm:mistral", "stored-mistral-key")
	if err != nil {
		t.Fatalf("setting test secret: %v", err)
	}

	// Make sure env var is absolutely NOT set
	t.Setenv("MISTRAL_API_KEY", "")
	os.Unsetenv("MISTRAL_API_KEY") // Just to be totally certain

	cfg := &config.Config{
		LLMProvider:  "mistral",
		DefaultModel: "mistral-large-latest",
	}
	
	// Should resolve successfully using the store
	model, err := llm.ResolveModel(cfg, store)
	if err != nil {
		t.Fatalf("ResolveModel with store returned error: %v", err)
	}
	if model == nil {
		t.Fatal("ResolveModel with store returned nil model")
	}

	// Verify that the process-local env var mutation worked as expected
	if got := os.Getenv("MISTRAL_API_KEY"); got != "stored-mistral-key" {
		t.Errorf("expected MISTRAL_API_KEY to be set to %q process-locally, got %q", "stored-mistral-key", got)
	}
}
