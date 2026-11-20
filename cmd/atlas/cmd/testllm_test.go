package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTestLLMCmd_Flags(t *testing.T) {
	providerFlag := testllmCmd.Flags().Lookup("provider")
	if providerFlag == nil {
		t.Fatal("expected --provider flag on testllmCmd")
	}
	if providerFlag.Value.Type() != "string" {
		t.Errorf("expected string type for --provider, got %s", providerFlag.Value.Type())
	}

	modelFlag := testllmCmd.Flags().Lookup("model")
	if modelFlag == nil {
		t.Fatal("expected --model flag on testllmCmd")
	}
	if modelFlag.Value.Type() != "string" {
		t.Errorf("expected string type for --model, got %s", modelFlag.Value.Type())
	}
}

func TestTestLLMCmd_MalformedConfig(t *testing.T) {
	tempDir := t.TempDir()
	atlasDir := filepath.Join(tempDir, ".atlas")
	if err := os.MkdirAll(atlasDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(atlasDir, "config.json"), []byte("{malformed json"), 0o644); err != nil {
		t.Fatal(err)
	}

	testLLMProvider = ""
	testLLMModel = ""
	pathFlag = ""
	defer func() {
		testLLMProvider = ""
		testLLMModel = ""
		pathFlag = ""
	}()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"testllm", tempDir})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when config.json is malformed")
	}
	if !strings.Contains(err.Error(), "failed to load config") {
		t.Errorf("expected 'failed to load config' error, got %v", err)
	}
}

func TestTestLLMCmd_MissingKeyError(t *testing.T) {
	tempDir := t.TempDir()
	atlasDir := filepath.Join(tempDir, ".atlas")
	if err := os.MkdirAll(atlasDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfgContent := `{"llm_provider": "openai", "default_model": "gpt-5.6-sol"}`
	if err := os.WriteFile(filepath.Join(atlasDir, "config.json"), []byte(cfgContent), 0o644); err != nil {
		t.Fatal(err)
	}

	_ = os.Unsetenv("OPENAI_API_KEY")

	testLLMProvider = ""
	testLLMModel = ""
	pathFlag = ""
	defer func() {
		testLLMProvider = ""
		testLLMModel = ""
		pathFlag = ""
	}()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"testllm", "--provider", "openai", tempDir})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error due to missing API key")
	}
	if !strings.Contains(err.Error(), "failed to initialize LLM client") {
		t.Errorf("expected 'failed to initialize LLM client' error, got %v", err)
	}
}
