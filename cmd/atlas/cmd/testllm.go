package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/Yashh56/atlas/internal/config"
	"github.com/Yashh56/atlas/internal/llm"
)

var testllmCmd = &cobra.Command{
	Use:   "testllm <path>",
	Short: "Test whether the configured LLM API key is valid",
	Long:  `testllm loads the config for the given project path and sends a small ping to the configured LLM to verify that the API key is active.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]
		configPath := filepath.Join(path, ".atlas", "config.json")
		
		cfg, err := config.Load(configPath)
		if err != nil {
			return fmt.Errorf("failed to load config at %s: %w", configPath, err)
		}
		
		fmt.Printf("Testing LLM connection to provider: %s (model: %s)...\n", cfg.LLMProvider, cfg.DefaultModel)
		
		client, err := llm.NewClient(cfg)
		if err != nil {
			return fmt.Errorf("failed to initialize LLM client: %w", err)
		}
		
		systemPrompt := "You are a test agent."
		userPrompt := "Reply with exactly the word 'SUCCESS' if you receive this."
		
		resp, err := client.Complete(cmd.Context(), systemPrompt, userPrompt)
		if err != nil {
			return fmt.Errorf("API call failed: %w", err)
		}
		
		fmt.Printf("LLM Response: %s\n", resp)
		fmt.Println("✓ API key is active and connection is working!")
		return nil
	},
}
