package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Yashh56/atlas/internal/config"
)

func TestLoad_MissingFile_ReturnsDefaults(t *testing.T) {
	cfg, err := config.Load(filepath.Join(t.TempDir(), "nonexistent.json"))
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if cfg.DefaultModel != "" {
		t.Errorf("DefaultModel = %q, want %q", cfg.DefaultModel, "")
	}
	if cfg.Approval != "manual" {
		t.Errorf("Approval = %q, want %q", cfg.Approval, "manual")
	}
}

func TestLoad_ValidFile_ReturnsValues(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "atlas.json")
	content := `{"default_model":"gpt-4o","approval":"auto"}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DefaultModel != "gpt-4o" {
		t.Errorf("DefaultModel = %q, want %q", cfg.DefaultModel, "gpt-4o")
	}
	if cfg.Approval != "auto" {
		t.Errorf("Approval = %q, want %q", cfg.Approval, "auto")
	}
}

func TestLoad_MalformedFile_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "atlas.json")
	if err := os.WriteFile(path, []byte("{not valid json"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
}

func TestSave_ValidConfig_PersistsValues(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "atlas.json")
	
	cfg := &config.Config{
		DefaultModel: "test-model",
		LLMProvider: "test-provider",
		LocalLLMBaseURL: "http://test:1234",
	}
	
	if err := cfg.Save(path); err != nil {
		t.Fatalf("unexpected error saving config: %v", err)
	}
	
	loaded, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error loading saved config: %v", err)
	}
	
	if loaded.DefaultModel != "test-model" {
		t.Errorf("expected test-model, got %q", loaded.DefaultModel)
	}
	if loaded.LocalLLMBaseURL != "http://test:1234" {
		t.Errorf("expected http://test:1234, got %q", loaded.LocalLLMBaseURL)
	}
}
