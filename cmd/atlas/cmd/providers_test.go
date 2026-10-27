package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Yashh56/atlas/internal/credentials"
)

func TestProvidersSet_UnimplementedProvider(t *testing.T) {
	err := runProvidersSet(nil, []string{"heroku"})
	if err == nil || !strings.Contains(err.Error(), "not implemented yet") {
		t.Fatalf("expected not implemented error, got %v", err)
	}
}

func TestProvidersSetAndUnset(t *testing.T) {
	tempDir := t.TempDir()
	originalOpen := openCredentials
	openCredentials = func() (*credentials.Store, error) {
		return credentials.OpenWithDir(tempDir)
	}
	defer func() { openCredentials = originalOpen }()

	originalPrompt := promptSecret
	promptSecret = func(label string) (string, error) {
		return "vercel-token-123", nil
	}
	defer func() { promptSecret = originalPrompt }()

	// 1. Set the key
	err := runProvidersSet(nil, []string{"vercel"})
	if err != nil {
		t.Fatalf("set failed: %v", err)
	}

	// Verify store has the key
	store, _ := openCredentials()
	key, err := store.GetSecret("vercel")
	if err != nil {
		t.Fatalf("failed to get stored secret: %v", err)
	}
	if key != "vercel-token-123" {
		t.Errorf("expected vercel-token-123, got %s", key)
	}

	// 2. Unset the key
	err = runProvidersUnset(nil, []string{"vercel"})
	if err != nil {
		t.Fatalf("unset failed: %v", err)
	}

	// Verify store is cleared
	_, err = store.GetSecret("vercel")
	if err == nil {
		t.Errorf("expected error getting deleted secret, got nil")
	}
}

func TestProvidersJSON(t *testing.T) {
	origJSON := jsonFlag
	defer func() { jsonFlag = origJSON }()

	jsonFlag = true
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"providers", "--json"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("providers execution failed: %v", err)
	}

	var statuses []ProviderStatusJSON
	if err := json.Unmarshal(buf.Bytes(), &statuses); err != nil {
		t.Fatalf("expected valid JSON array, got: %v; output: %s", err, buf.String())
	}

	if len(statuses) != 5 {
		t.Errorf("expected 5 providers, got %d", len(statuses))
	}
}
