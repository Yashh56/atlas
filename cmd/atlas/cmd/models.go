package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Yashh56/atlas/internal/config"
)

// llmProviderEnvVars maps each GoAI provider name to its expected API key env var.
var llmProviderEnvVars = []struct {
	Name   string
	EnvVar string
}{
	{"anthropic", "ANTHROPIC_API_KEY"},
	{"openai", "OPENAI_API_KEY"},
	{"gemini", "GEMINI_API_KEY"},
	{"mistral", "MISTRAL_API_KEY"},
	{"groq", "GROQ_API_KEY"},
	{"grok", "XAI_API_KEY"},
	{"local", ""}, // no key required
}

var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "Show API key status for all supported LLM providers",
	Long:  `models lists all supported LLM providers and shows whether their API keys are detected in the environment.`,
	RunE:  runModels,
}

func runModels(_ *cobra.Command, _ []string) error {
	// Try to detect the active provider from config.json in the current dir.
	activeName := ""
	configPath := filepath.Join(".atlas", "config.json")
	if cfg, err := config.Load(configPath); err == nil {
		activeName = cfg.LLMProvider
	}

	const colWidth = 11
	header := "LLM PROVIDERS"
	if activeName != "" {
		header += fmt.Sprintf("   (active: %s — from config.json's llm_provider)", activeName)
	}
	fmt.Println(header)

	for _, p := range llmProviderEnvVars {
		pad := strings.Repeat(" ", colWidth-len(p.Name))

		if p.EnvVar == "" {
			// local provider — no key needed, show base URL if we have a config.
			baseURL := "http://localhost:11434/v1"
			if cfg, err := config.Load(configPath); err == nil && cfg.LocalLLMBaseURL != "" {
				baseURL = cfg.LocalLLMBaseURL
			}
			fmt.Printf("  %s%s–  configured base URL: %s (not checked — no key needed)\n", p.Name, pad, baseURL)
			continue
		}

		if os.Getenv(p.EnvVar) != "" {
			fmt.Printf("  %s%s✓ %s detected\n", p.Name, pad, p.EnvVar)
		} else {
			fmt.Printf("  %s%s✗ %s not set\n", p.Name, pad, p.EnvVar)
		}
	}
	return nil
}
