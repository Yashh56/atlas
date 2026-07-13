package cmd

import (
	"strings"
	"testing"

	"github.com/Yashh56/atlas/internal/credentials"
)

func TestModelsSet_UnknownProvider(t *testing.T) {
	err := runModelsSet(nil, []string{"unknown"})
	if err == nil || !strings.Contains(err.Error(), "unknown provider") {
		t.Fatalf("expected unknown provider error, got %v", err)
	}
}

func TestModelsSet_LocalProvider(t *testing.T) {
	err := runModelsSet(nil, []string{"local"})
	if err == nil || !strings.Contains(err.Error(), "does not use an API key") {
		t.Fatalf("expected local provider error, got %v", err)
	}
}

func TestModelsSetAndUnset(t *testing.T) {
	// Mock openCredentials to use TempDir
	tempDir := t.TempDir()
	originalOpen := openCredentials
	openCredentials = func() (*credentials.Store, error) {
		return credentials.OpenWithDir(tempDir)
	}
	defer func() { openCredentials = originalOpen }()

	// Mock promptSecret
	originalPrompt := promptSecret
	promptSecret = func(label string) (string, error) {
		return "test-key-123", nil
	}
	defer func() { promptSecret = originalPrompt }()

	// 1. Set the key
	err := runModelsSet(nil, []string{"mistral"})
	if err != nil {
		t.Fatalf("set failed: %v", err)
	}

	// Verify store has the key
	store, _ := openCredentials()
	key, err := store.GetSecret("llm:mistral")
	if err != nil {
		t.Fatalf("failed to get stored secret: %v", err)
	}
	if key != "test-key-123" {
		t.Errorf("expected test-key-123, got %s", key)
	}
	meta, ok, _ := store.GetMeta("llm:mistral")
	if !ok || meta.Method != credentials.MethodStoredToken {
		t.Errorf("expected stored token metadata, got %v", meta)
	}

	// 2. Unset the key
	err = runModelsUnset(nil, []string{"mistral"})
	if err != nil {
		t.Fatalf("unset failed: %v", err)
	}

	// Verify store is cleared
	_, err = store.GetSecret("llm:mistral")
	if err == nil {
		t.Errorf("expected error getting deleted secret, got nil")
	}
	meta, ok, _ = store.GetMeta("llm:mistral")
	if !ok || meta.Method != credentials.MethodEnvVar { // Unset changes it to MethodEnvVar
		t.Errorf("expected env var metadata after unset, got %v", meta)
	}
}

func TestModelsUnset_EnvVarOnly(t *testing.T) {
	tempDir := t.TempDir()
	originalOpen := openCredentials
	openCredentials = func() (*credentials.Store, error) {
		return credentials.OpenWithDir(tempDir)
	}
	defer func() { openCredentials = originalOpen }()

	t.Setenv("MISTRAL_API_KEY", "env-key")

	err := runModelsUnset(nil, []string{"mistral"})
	if err != nil {
		t.Fatalf("unset failed on env var only: %v", err)
	}
	// Should succeed and print clarifying message (captured manually or just assert no error here)
}
