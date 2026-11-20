package llm

import (
	"testing"

	"github.com/Yashh56/atlas/internal/config"
)

func TestProviderForModel(t *testing.T) {
	tests := []struct {
		model string
		want  string
	}{
		{"openai/gpt-oss-20b", "groq"},
		{"openai/gpt-oss-120b", "groq"},
		{"groq", "groq"},
		{"llama-3.3-70b-versatile", "groq"},
		{"llama3-70b", "groq"},
		{"mixtral-8x7b-32768", "groq"},
		{"o3-mini", "openai"},
		{"gpt-5.6-sol", "openai"},
		{"claude-sonnet-5", "anthropic"},
		{"gemini-3.5-flash", "gemini"},
		{"local", "local"},
		{"llama3", "local"},
	}

	for _, tc := range tests {
		got := ProviderForModel(tc.model)
		if got != tc.want {
			t.Errorf("ProviderForModel(%q) = %q, want %q", tc.model, got, tc.want)
		}
	}
}

func TestResolveModel_Groq(t *testing.T) {
	t.Setenv("GROQ_API_KEY", "test-groq-key")
	cfg := &config.Config{
		LLMProvider:  "groq",
		DefaultModel: "openai/gpt-oss-20b",
	}
	model, err := ResolveModel(cfg, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model == nil {
		t.Fatal("expected non-nil model")
	}
	if model.ModelID() != "openai/gpt-oss-20b" {
		t.Errorf("expected model ID 'openai/gpt-oss-20b', got %q", model.ModelID())
	}
}

func TestResolveModel_GroqModelNameAsProvider(t *testing.T) {
	t.Setenv("GROQ_API_KEY", "test-groq-key")
	cfg := &config.Config{
		LLMProvider: "openai/gpt-oss-20b",
	}
	model, err := ResolveModel(cfg, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model == nil {
		t.Fatal("expected non-nil model")
	}
	if model.ModelID() != "openai/gpt-oss-20b" {
		t.Errorf("expected model ID 'openai/gpt-oss-20b', got %q", model.ModelID())
	}
}

func TestNewClient_GroqNameAndModelID(t *testing.T) {
	t.Setenv("GROQ_API_KEY", "test-groq-key")
	cfg := &config.Config{
		LLMProvider: "openai/gpt-oss-20b",
	}
	client, err := NewClient(cfg, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.Name() != "groq" {
		t.Errorf("expected client.Name() == 'groq', got %q", client.Name())
	}
	if client.ModelID() != "openai/gpt-oss-20b" {
		t.Errorf("expected client.ModelID() == 'openai/gpt-oss-20b', got %q", client.ModelID())
	}
}

func TestResolveModel_OpenAIO3(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-openai-key")
	cfg := &config.Config{
		LLMProvider:  "openai",
		DefaultModel: "o3-mini",
	}
	model, err := ResolveModel(cfg, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model == nil {
		t.Fatal("expected non-nil model")
	}
	if model.ModelID() != "o3-mini" {
		t.Errorf("expected model ID 'o3-mini', got %q", model.ModelID())
	}
}
