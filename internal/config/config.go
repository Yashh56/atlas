// Package config handles loading Atlas configuration from disk.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
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
		DefaultModel:      "claude-sonnet-4-6",
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
