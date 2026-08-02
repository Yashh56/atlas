// Package config handles loading Atlas configuration from disk.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds the Atlas runtime configuration.
type Config struct {
	DefaultModel      string `json:"default_model"`
	LLMProvider       string `json:"llm_provider"`
	LocalLLMBaseURL   string `json:"local_llm_base_url"`
	Approval          string `json:"approval"` // "manual" | "auto"
}

// defaults returns a Config with sane out-of-the-box values.
func defaults() *Config {
	return &Config{
		DefaultModel:      "",
		LLMProvider:       "anthropic",
		LocalLLMBaseURL:   "http://localhost:11434/v1",
		Approval:          "manual",
	}
}

// Load reads a Config from the JSON file at path.
// If the file does not exist, sane defaults are returned without error.
// If the file exists but contains malformed JSON, a descriptive error is returned.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return defaults(), nil
		}
		return nil, fmt.Errorf("config: reading %s: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: malformed JSON in %s: %w", path, err)
	}
	return &cfg, nil
}

// Save writes the Config to the JSON file at path.
// It creates any necessary parent directories.
func (c *Config) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("config: marshaling json: %w", err)
	}
	
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("config: creating dir %s: %w", filepath.Dir(path), err)
	}
	
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("config: writing %s: %w", path, err)
	}
	
	return nil
}
