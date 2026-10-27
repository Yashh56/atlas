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
		{"anthropic", "claude-sonnet-5"},
		{"openai", "gpt-5.6-sol"},
		{"gemini", "gemini-3.5-flash"},
		{"mistral", "mistral-large-latest"},
		{"groq", "openai/gpt-oss-120b"},
		{"grok", "grok-4.5"},
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
				if tt.name == "local" && model.ModelID() != "" {
					t.Logf("local model resolved to %q (auto-detected)", model.ModelID())
				} else {
					t.Errorf("expected ModelID %q, got %q", tt.model, model.ModelID())
				}
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
		DefaultModel: "claude-sonnet-5",
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

func TestHasProviderKey(t *testing.T) {
	t.Setenv("MISTRAL_API_KEY", "mistral-val")
	t.Setenv("ANTHROPIC_API_KEY", "")
	os.Unsetenv("ANTHROPIC_API_KEY")

	if !llm.HasProviderKey("mistral", nil) {
		t.Errorf("expected HasProviderKey('mistral') to be true")
	}
	if llm.HasProviderKey("anthropic", nil) {
		t.Errorf("expected HasProviderKey('anthropic') to be false")
	}
	if llm.HasProviderKey("unknown", nil) {
		t.Errorf("expected HasProviderKey('unknown') to be false")
	}
	if llm.HasProviderKey("local", nil) {
		t.Errorf("expected HasProviderKey('local') to be false")
	}

	// Test with credentials store
	store, err := credentials.OpenWithDir(t.TempDir())
	if err != nil {
		t.Fatalf("opening credentials store: %v", err)
	}
	if err := store.SetSecret("llm:openai", "store-openai-key"); err != nil {
		t.Fatalf("setting secret: %v", err)
	}
	defer store.DeleteSecret("llm:openai")
	t.Setenv("OPENAI_API_KEY", "")
	os.Unsetenv("OPENAI_API_KEY")

	if !llm.HasProviderKey("openai", store) {
		t.Errorf("expected HasProviderKey('openai') to be true with store")
	}
}

func TestDetectConfiguredProvider(t *testing.T) {
	for _, p := range llm.KnownProviders {
		if p.EnvVar != "" {
			t.Setenv(p.EnvVar, "")
			os.Unsetenv(p.EnvVar)
		}
	}

	// 1. None configured
	if _, ok := llm.DetectConfiguredProvider("", nil); ok {
		t.Errorf("expected false when no provider configured")
	}

	// 2. Mistral configured in env
	t.Setenv("MISTRAL_API_KEY", "mistral-key")
	spec, ok := llm.DetectConfiguredProvider("", nil)
	if !ok || spec.Name != "mistral" {
		t.Errorf("expected mistral to be detected, got ok=%v, spec=%+v", ok, spec)
	}

	// When requestedProvider is "anthropic" (unconfigured), fallback to mistral
	spec, ok = llm.DetectConfiguredProvider("anthropic", nil)
	if !ok || spec.Name != "mistral" {
		t.Errorf("expected mistral fallback for unconfigured anthropic, got ok=%v, spec=%+v", ok, spec)
	}

	// 3. Anthropic configured
	t.Setenv("ANTHROPIC_API_KEY", "anthropic-key")
	spec, ok = llm.DetectConfiguredProvider("anthropic", nil)
	if !ok || spec.Name != "anthropic" {
		t.Errorf("expected anthropic when anthropic key is set, got ok=%v, spec=%+v", ok, spec)
	}
}

func TestResolveModel_AutoFallback(t *testing.T) {
	for _, p := range llm.KnownProviders {
		if p.EnvVar != "" {
			t.Setenv(p.EnvVar, "")
			os.Unsetenv(p.EnvVar)
		}
	}

	t.Setenv("MISTRAL_API_KEY", "mistral-key")

	// Default config (anthropic) should fallback to mistral
	cfg := &config.Config{
		LLMProvider:  "anthropic",
		DefaultModel: "claude-sonnet-5",
	}
	model, err := llm.ResolveModel(cfg, nil)
	if err != nil {
		t.Fatalf("ResolveModel unexpected error: %v", err)
	}
	if cfg.LLMProvider != "mistral" {
		t.Errorf("expected cfg.LLMProvider to be updated to 'mistral', got %q", cfg.LLMProvider)
	}
	if model.ModelID() != "mistral-small-latest" && model.ModelID() != "mistral-large-latest" && model.ModelID() != "mistral-medium-3.5" {
		t.Errorf("expected mistral model, got %q", model.ModelID())
	}
}
