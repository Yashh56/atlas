package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Yashh56/atlas/internal/cliutil"
	"github.com/Yashh56/atlas/internal/config"
	"github.com/Yashh56/atlas/internal/credentials"
	"github.com/Yashh56/atlas/internal/llm"
)

// llmProviderEnvVars maps each GoAI provider name to its expected API key env var.
var llmProviderEnvVars = []struct {
	Name   string
	EnvVar string
	Models []string
}{
	{"anthropic", "ANTHROPIC_API_KEY", []string{"claude-sonnet-5", "claude-opus-4-8", "claude-haiku-4-5-20251001"}},
	{"openai", "OPENAI_API_KEY", []string{"gpt-5.6-sol", "gpt-5.6-terra", "gpt-5.6-luna"}},
	{"gemini", "GEMINI_API_KEY", []string{"gemini-3.5-flash", "gemini-3.1-pro"}},
	{"mistral", "MISTRAL_API_KEY", []string{"mistral-large-latest", "mistral-small-latest"}}, // auto-updating aliases
	{"groq", "GROQ_API_KEY", []string{"openai/gpt-oss-120b", "openai/gpt-oss-20b"}},
	{"grok", "XAI_API_KEY", []string{"grok-4.5", "grok-4.3"}},
	{"local", "", []string{}}, // local models are discovered, not guessed — see NewModelLister
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

var modelNameFlag string

func init() {
	modelsSetCmd.Flags().StringVar(&modelNameFlag, "model-name", "", "Model name to select (local provider only)")
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
		activeModel := ""
		if cfg, err := config.Load(configPath); err == nil {
			activeModel = cfg.DefaultModel
		}
		if activeName == "local" && activeModel != "" {
			header += fmt.Sprintf("   (active: %s — %s, from config.json)", activeName, activeModel)
		} else {
			header += fmt.Sprintf("   (active: %s — from config.json's llm_provider)", activeName)
		}
	}
	fmt.Println(cliutil.FormatHeader(header))

	store, _ := openCredentials()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	for _, p := range llmProviderEnvVars {
		pad := strings.Repeat(" ", colWidth-len(p.Name))

		// Resolve the API key from store or env.
		apiKey := os.Getenv(p.EnvVar)
		if apiKey == "" && store != nil {
			if secret, err := store.GetSecret("llm:" + p.Name); err == nil {
				apiKey = secret
			}
		}

		// local provider doesn't use an API key
		if apiKey == "" && p.Name != "local" {
			fmt.Printf("  %s%s%s %s not set\n",
				cliutil.StyleHighlight.Render(p.Name), pad, cliutil.IconError, p.EnvVar)
			continue
		}

		keySource := p.EnvVar
		if p.Name == "local" {
			baseURL := "http://localhost:11434/v1"
			if cfg, err := config.Load(configPath); err == nil && cfg.LocalLLMBaseURL != "" {
				baseURL = cfg.LocalLLMBaseURL
			}
			keySource = fmt.Sprintf("runtime detected at %s", baseURL)
		} else if store != nil {
			if meta, ok, _ := store.GetMeta("llm:" + p.Name); ok && meta.Method == credentials.MethodStoredToken {
				keySource = "stored"
			}
		}

		// Key is present (or local) — attempt live model discovery.
		baseURL := "http://localhost:11434/v1"
		if cfg, err := config.Load(configPath); err == nil && cfg.LocalLLMBaseURL != "" {
			baseURL = cfg.LocalLLMBaseURL
		}
		if lister, ok := llm.NewModelLister(p.Name, apiKey, baseURL); ok {
			if models, err := lister.ListModels(ctx); err == nil && len(models) > 0 {
				const maxDisplay = 10
				displayed := models
				more := 0
				if len(models) > maxDisplay {
					displayed = models[:maxDisplay]
					more = len(models) - maxDisplay
				}
				summary := strings.Join(displayed, ", ")
				if more > 0 {
					summary += fmt.Sprintf(", +%d more", more)
				}
				fmt.Printf("  %s%s%s %s — %d models (%s)\n",
					cliutil.StyleHighlight.Render(p.Name), pad, cliutil.IconSuccess,
					keySource, len(models), cliutil.StyleSubtext.Render(summary))
				continue
			}
			// ListModels failed — fall through to static display.
		}

		if p.Name == "local" {
			baseURL := "http://localhost:11434/v1"
			if cfg, err := config.Load(configPath); err == nil && cfg.LocalLLMBaseURL != "" {
				baseURL = cfg.LocalLLMBaseURL
			}
			fmt.Printf("  %s%s%s no local model runtime found at %s — is Ollama running?\n",
				cliutil.StyleHighlight.Render(p.Name), pad, cliutil.IconError, baseURL)
			continue
		}

		// Fallback: key present but no live list (ListModels failed or no lister).
		fmt.Printf("  %s%s%s %s\n",
			cliutil.StyleHighlight.Render(p.Name), pad, cliutil.IconSuccess, keySource)
		for _, m := range p.Models {
			fmt.Printf("    %s %s\n", cliutil.StyleSubtext.Render("↳"), cliutil.StyleSubtext.Render(m))
		}
	}
	return nil
}

func getProviderForModel(modelName string) string {
	for _, p := range llmProviderEnvVars {
		if p.Name == modelName {
			return p.Name // fallback for legacy
		}
		for _, m := range p.Models {
			if m == modelName {
				return p.Name
			}
		}
	}
	return "local"
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
		configPath := filepath.Join(".atlas", "config.json")
		cfg, err := config.Load(configPath)
		if err != nil {
			return err
		}
		baseURL := cfg.LocalLLMBaseURL
		if baseURL == "" {
			baseURL = "http://localhost:11434/v1"
		}

		lister, ok := llm.NewModelLister("local", "", baseURL)
		if !ok {
			return fmt.Errorf("local model lister not available")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		models, err := lister.ListModels(ctx)
		if err != nil {
			return err // this will return the "no local model runtime found..." error
		}

		if len(models) == 0 {
			return fmt.Errorf("No models found — pull one first, e.g. 'ollama pull qwen2.5-coder:7b'")
		}

		var chosen string
		if modelNameFlag != "" {
			// Verify it exists
			found := false
			for _, m := range models {
				if m == modelNameFlag {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("model %q not found in local runtime", modelNameFlag)
			}
			chosen = modelNameFlag
		} else {
			// Ask interactively
			for i, m := range models {
				fmt.Printf("  [%d] %s\n", i+1, m)
			}
			fmt.Printf("\nSelect a model [1-%d]: ", len(models))
			var choice int
			if _, err := fmt.Scanf("%d", &choice); err != nil || choice < 1 || choice > len(models) {
				return fmt.Errorf("invalid selection")
			}
			chosen = models[choice-1]
		}

		cfg.LLMProvider = "local"
		cfg.DefaultModel = chosen
		if err := cfg.Save(configPath); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}
		fmt.Printf("%s Stored %s in config.json.\n", cliutil.IconSuccess, chosen)
		return nil
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
