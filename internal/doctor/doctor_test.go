package doctor

import (
	"context"
	"encoding/json"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/Yashh56/atlas/internal/config"
	"github.com/Yashh56/atlas/internal/credentials"
	"github.com/Yashh56/atlas/internal/llm"
)

type testMockClient struct {
	providerName string
}

func (m *testMockClient) Name() string { return m.providerName }
func (m *testMockClient) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	return "pong", nil
}

func TestRunDiagnostics(t *testing.T) {
	origNewClient := newLLMClient
	newLLMClient = func(cfg *config.Config, store *credentials.Store) (llm.Client, error) {
		return &testMockClient{providerName: cfg.LLMProvider}, nil
	}
	defer func() { newLLMClient = origNewClient }()

	report, err := RunDiagnostics(context.Background(), ".")
	if err != nil {
		t.Fatalf("unexpected error running diagnostics: %v", err)
	}

	if len(report.Categories) == 0 {
		t.Fatal("expected diagnostic categories, got 0")
	}

	// Verify JSON serialization
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("failed to marshal report to JSON: %v", err)
	}

	var decoded DiagnosticReport
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal JSON into DiagnosticReport: %v", err)
	}

	if decoded.Overall == "" {
		t.Error("expected overall status to be set")
	}
}

func TestCheckTools_PythonLauncherFallback(t *testing.T) {
	orig := checkOptionalExecutable
	defer func() { checkOptionalExecutable = orig }()

	called := []string{}
	checkOptionalExecutable = func(ctx context.Context, name string, args ...string) CheckItem {
		called = append(called, name)
		if name == "python" {
			return CheckItem{Name: "Python", Status: StatusWarn, Details: "Not installed (optional)"}
		}
		if name == "py" {
			return CheckItem{Name: "Py", Status: StatusPass, Version: "Python 3.12.0"}
		}
		if name == "python3" {
			return CheckItem{Name: "Python3", Status: StatusPass, Version: "Python 3.12.0"}
		}
		return orig(ctx, name, args...)
	}

	cat := checkTools(context.Background(), t.TempDir())
	var pyItem *CheckItem
	for _, c := range cat.Checks {
		if c.Name == "Python" {
			pyItem = &c
			break
		}
	}

	if pyItem == nil {
		t.Fatal("expected Python check item in category")
	}

	if runtime.GOOS == "windows" {
		if pyItem.Status != StatusPass {
			t.Errorf("expected StatusPass for Python on windows, got %v", pyItem.Status)
		}
		if pyItem.Version != "Python 3.12.0" {
			t.Errorf("expected Version 'Python 3.12.0', got %q", pyItem.Version)
		}
		foundPy := false
		for _, name := range called {
			if name == "py" {
				foundPy = true
				break
			}
		}
		if !foundPy {
			t.Errorf("expected 'py' to be called on windows when 'python' fails")
		}
	} else {
		if pyItem.Status != StatusPass {
			t.Errorf("expected StatusPass for Python on unix, got %v", pyItem.Status)
		}
	}
}

func TestCheckLLM_AutoDetectMistral(t *testing.T) {
	origNewClient := newLLMClient
	var capturedCfg *config.Config
	newLLMClient = func(cfg *config.Config, store *credentials.Store) (llm.Client, error) {
		capturedCfg = cfg
		return &testMockClient{providerName: cfg.LLMProvider}, nil
	}
	defer func() { newLLMClient = origNewClient }()

	t.Setenv("MISTRAL_API_KEY", "test-mistral-key")
	t.Setenv("ANTHROPIC_API_KEY", "")
	os.Unsetenv("ANTHROPIC_API_KEY")

	tempDir := t.TempDir()
	cat := checkLLM(context.Background(), tempDir)

	if len(cat.Checks) == 0 {
		t.Fatal("expected LLM checks")
	}
	item := cat.Checks[0]
	if item.Status != StatusPass {
		t.Fatalf("expected StatusPass, got %v: %s", item.Status, item.Error)
	}
	if !strings.Contains(item.Name, "mistral-small-latest") {
		t.Errorf("expected model name to contain 'mistral-small-latest', got %q", item.Name)
	}
	if !strings.Contains(item.Details, "Provider: mistral") {
		t.Errorf("expected details to contain 'Provider: mistral', got %q", item.Details)
	}
	if capturedCfg == nil || capturedCfg.LLMProvider != "mistral" {
		t.Errorf("expected captured cfg.LLMProvider to be 'mistral', got %+v", capturedCfg)
	}
}

func TestCheckLLM_NoAPIKeyConfigured(t *testing.T) {
	for _, p := range llm.KnownProviders {
		if p.EnvVar != "" {
			t.Setenv(p.EnvVar, "")
			os.Unsetenv(p.EnvVar)
		}
	}

	tempDir := t.TempDir()
	cat := checkLLM(context.Background(), tempDir)

	if len(cat.Checks) == 0 {
		t.Fatal("expected LLM checks")
	}
	item := cat.Checks[0]
	if item.Status != StatusWarn {
		t.Fatalf("expected StatusWarn, got %v", item.Status)
	}
	if !strings.Contains(item.Error, "No API key configured") {
		t.Errorf("expected error to contain 'No API key configured', got %q", item.Error)
	}
}
