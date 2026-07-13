package cmd

import (
	"strings"
	"testing"

	"github.com/Yashh56/atlas/internal/credentials"
)

func TestProvidersSet_UnimplementedProvider(t *testing.T) {
	err := runProvidersSet(nil, []string{"render"})
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
