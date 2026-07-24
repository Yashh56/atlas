package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"time"

	"github.com/spf13/cobra"

	"github.com/Yashh56/atlas/internal/cliutil"
	"github.com/Yashh56/atlas/internal/config"
	"github.com/Yashh56/atlas/internal/credentials"
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

var promptSecret = cliutil.PromptSecret
var openCredentials = credentials.Open

var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "Show API key status for all supported LLM providers",
	Long:  `models lists all supported LLM providers and shows whether their API keys are detected in the environment.`,
	RunE:  runModels,
}

var modelsSetCmd = &cobra.Command{
	Use:   "set <provider>",
	Short: "Securely store an LLM API key",
	Args:  cobra.ExactArgs(1),
	RunE:  runModelsSet,
}

var modelsUnsetCmd = &cobra.Command{
	Use:   "unset <provider>",
	Short: "Remove a stored LLM API key",
	Args:  cobra.ExactArgs(1),
	RunE:  runModelsUnset,
}

func init() {
	modelsCmd.AddCommand(modelsSetCmd)
	modelsCmd.AddCommand(modelsUnsetCmd)
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
	fmt.Println(cliutil.FormatHeader(header))

	store, _ := openCredentials()

	for _, p := range llmProviderEnvVars {
		pad := strings.Repeat(" ", colWidth-len(p.Name))

		if p.EnvVar == "" {
			// local provider — no key needed, show base URL if we have a config.
			baseURL := "http://localhost:11434/v1"
			if cfg, err := config.Load(configPath); err == nil && cfg.LocalLLMBaseURL != "" {
				baseURL = cfg.LocalLLMBaseURL
			}
			fmt.Printf("  %s%s%s  configured base URL: %s (not checked — no key needed)\n", cliutil.StyleHighlight.Render(p.Name), pad, cliutil.IconDot, cliutil.StyleSubtext.Render(baseURL))
			continue
		}

		// Check store first
		if store != nil {
			if meta, ok, _ := store.GetMeta("llm:" + p.Name); ok && meta.Method == credentials.MethodStoredToken {
				fmt.Printf("  %s%s%s stored\n", cliutil.StyleHighlight.Render(p.Name), pad, cliutil.IconSuccess)
				continue
			}
		}

		if os.Getenv(p.EnvVar) != "" {
			fmt.Printf("  %s%s%s %s detected\n", cliutil.StyleHighlight.Render(p.Name), pad, cliutil.IconSuccess, p.EnvVar)
		} else {
			fmt.Printf("  %s%s%s %s not set\n", cliutil.StyleHighlight.Render(p.Name), pad, cliutil.IconError, p.EnvVar)
		}
	}
	return nil
}

func getLLMEnvVar(provider string) string {
	for _, p := range llmProviderEnvVars {
		if p.Name == provider {
			return p.EnvVar
		}
	}
	return ""
}

func runModelsSet(_ *cobra.Command, args []string) error {
	provider := args[0]
	if provider == "local" {
		return fmt.Errorf("'local' provider does not use an API key")
	}
	envVar := getLLMEnvVar(provider)
	if envVar == "" {
		return fmt.Errorf("unknown provider %q", provider)
	}

	key, err := promptSecret(fmt.Sprintf("API key for %s", provider))
	if err != nil {
		return err
	}

	store, err := openCredentials()
	if err != nil {
		return fmt.Errorf("opening credential store: %w", err)
	}

	if err := store.SetSecret("llm:"+provider, key); err != nil {
		return fmt.Errorf("storing secret: %w", err)
	}

	if err := store.SetMeta(credentials.ProviderCredential{
		Provider:   "llm:" + provider,
		Method:     credentials.MethodStoredToken,
		VerifiedAt: time.Now().UTC(),
	}); err != nil {
		return fmt.Errorf("storing metadata: %w", err)
	}

	fmt.Printf("%s Stored. Run `atlas models` to confirm.\n", cliutil.IconSuccess)
	return nil
}

func runModelsUnset(_ *cobra.Command, args []string) error {
	provider := args[0]
	if provider == "local" {
		return fmt.Errorf("'local' provider does not use an API key")
	}
	envVar := getLLMEnvVar(provider)
	if envVar == "" {
		return fmt.Errorf("unknown provider %q", provider)
	}

	store, err := openCredentials()
	if err != nil {
		return fmt.Errorf("opening credential store: %w", err)
	}

	meta, ok, _ := store.GetMeta("llm:" + provider)
	if !ok || meta.Method != credentials.MethodStoredToken {
		if os.Getenv(envVar) != "" {
			fmt.Printf("%q is set via %s, not a stored key — unset the env var instead\n", provider, envVar)
			return nil
		}
		fmt.Printf("No stored key found for %q\n", provider)
		return nil
	}

	if err := store.DeleteSecret("llm:" + provider); err != nil {
		return fmt.Errorf("deleting secret: %w", err)
	}

	// Remove metadata by writing an empty credentials record (or deleting if we had a method for it)
	// We'll just set it to env var mode so it's cleared from stored token mode.
	store.SetMeta(credentials.ProviderCredential{
		Provider: "llm:" + provider,
		Method:   credentials.MethodEnvVar,
	})

	fmt.Printf("%s Removed stored key for %q.\n", cliutil.IconSuccess, provider)
	return nil
}
