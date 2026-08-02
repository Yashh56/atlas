package cmd

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Yashh56/atlas/internal/config"
	"github.com/Yashh56/atlas/internal/credentials"
)

func TestModelsSet_UnknownProvider(t *testing.T) {
	err := runModelsSet(nil, []string{"unknown"})
	if err == nil || !strings.Contains(err.Error(), "unknown provider") {
		t.Fatalf("expected unknown provider error, got %v", err)
	}
}

func TestModelsSet_LocalProvider_NetworkError(t *testing.T) {
	tempDir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(origWd)
	
	// Create .atlas to satisfy config paths
	os.MkdirAll(".atlas", 0755)

	// Write a config pointing to an invalid port to ensure connection refused
	os.WriteFile(filepath.Join(".atlas", "config.json"), []byte(`{"local_llm_base_url":"http://127.0.0.1:65432/v1"}`), 0644)

	err := runModelsSet(nil, []string{"local"})
	if err == nil || !strings.Contains(err.Error(), "no local model runtime found") {
		t.Fatalf("expected network error, got %v", err)
	}
}

func TestModelsSet_LocalProvider_ZeroModels(t *testing.T) {
	tempDir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(origWd)
	
	os.MkdirAll(".atlas", 0755)
	
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"data":[]}`)
	}))
	defer ts.Close()
	
	os.WriteFile(filepath.Join(".atlas", "config.json"), []byte(fmt.Sprintf(`{"local_llm_base_url":"%s"}`, ts.URL)), 0644)

	err := runModelsSet(nil, []string{"local"})
	if err == nil || !strings.Contains(err.Error(), "No models found") {
		t.Fatalf("expected zero models error, got %v", err)
	}
}

func TestModelsSet_LocalProvider_NonInteractiveValid(t *testing.T) {
	tempDir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(origWd)
	
	os.MkdirAll(".atlas", 0755)
	
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"data":[{"id":"qwen2.5-coder:7b"}]}`)
	}))
	defer ts.Close()
	
	os.WriteFile(filepath.Join(".atlas", "config.json"), []byte(fmt.Sprintf(`{"local_llm_base_url":"%s"}`, ts.URL)), 0644)
	
	// Set the flag
	originalFlag := modelNameFlag
	modelNameFlag = "qwen2.5-coder:7b"
	defer func() { modelNameFlag = originalFlag }()

	err := runModelsSet(nil, []string{"local"})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	
	// verify it was written
	cfg, err := config.Load(filepath.Join(".atlas", "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LLMProvider != "local" || cfg.DefaultModel != "qwen2.5-coder:7b" {
		t.Fatalf("config not saved correctly: %+v", cfg)
	}
}

func TestModelsSet_LocalProvider_NonInteractiveInvalid(t *testing.T) {
	tempDir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(origWd)
	
	os.MkdirAll(".atlas", 0755)
	
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"data":[{"id":"qwen2.5-coder:7b"}]}`)
	}))
	defer ts.Close()
	
	os.WriteFile(filepath.Join(".atlas", "config.json"), []byte(fmt.Sprintf(`{"local_llm_base_url":"%s"}`, ts.URL)), 0644)
	
	// Set the flag
	originalFlag := modelNameFlag
	modelNameFlag = "llama3"
	defer func() { modelNameFlag = originalFlag }()

	err := runModelsSet(nil, []string{"local"})
	if err == nil || !strings.Contains(err.Error(), "model \"llama3\" not found in local runtime") {
		t.Fatalf("expected missing model error, got %v", err)
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
