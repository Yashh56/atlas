package doctor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Yashh56/atlas/internal/config"
	"github.com/Yashh56/atlas/internal/credentials"
	"github.com/Yashh56/atlas/internal/llm"
	"github.com/Yashh56/atlas/internal/updater"
	"github.com/Yashh56/atlas/internal/version"
)

type CheckStatus string

const (
	StatusPass CheckStatus = "pass"
	StatusWarn CheckStatus = "warn"
	StatusFail CheckStatus = "fail"
)

type CheckItem struct {
	Name      string      `json:"name"`
	Status    CheckStatus `json:"status"`
	Details   string      `json:"details,omitempty"`
	Version   string      `json:"version,omitempty"`
	LatencyMs int64       `json:"latency_ms,omitempty"`
	Error     string      `json:"error,omitempty"`
}

type CategoryReport struct {
	Category string      `json:"category"`
	Checks   []CheckItem `json:"checks"`
}

type DiagnosticReport struct {
	Timestamp  time.Time        `json:"timestamp"`
	Overall    CheckStatus      `json:"overall"`
	Categories []CategoryReport `json:"categories"`
}

var newLLMClient = llm.NewClient

func title(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// RunDiagnostics executes all diagnostic health checks.
func RunDiagnostics(ctx context.Context, workspaceRoot string) (*DiagnosticReport, error) {
	report := &DiagnosticReport{
		Timestamp: time.Now().UTC(),
		Overall:   StatusPass,
	}

	report.Categories = append(report.Categories, checkTools(ctx, workspaceRoot))
	report.Categories = append(report.Categories, checkSecurity())
	report.Categories = append(report.Categories, checkLLM(ctx, workspaceRoot))
	report.Categories = append(report.Categories, checkVersion())

	for _, cat := range report.Categories {
		for _, check := range cat.Checks {
			if check.Status == StatusFail {
				report.Overall = StatusFail
			} else if check.Status == StatusWarn && report.Overall != StatusFail {
				report.Overall = StatusWarn
			}
		}
	}

	return report, nil
}

func checkTools(ctx context.Context, workspaceRoot string) CategoryReport {
	cat := CategoryReport{Category: "Tools & Environment"}

	// Git
	gitCheck := checkExecutable(ctx, "git", "version")
	if gitCheck.Status == StatusPass {
		cleanCmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
		cleanCmd.Dir = workspaceRoot
		if out, err := cleanCmd.Output(); err == nil {
			if len(strings.TrimSpace(string(out))) == 0 {
				gitCheck.Details = "clean working tree"
			} else {
				gitCheck.Details = "uncommitted changes detected"
			}
		} else {
			gitCheck.Details = "not in a git repository"
		}
	} else {
		gitCheck.Status = StatusFail
		gitCheck.Error = "Git is required by Atlas but was not found in PATH"
	}
	cat.Checks = append(cat.Checks, gitCheck)

	// Node.js & package managers
	cat.Checks = append(cat.Checks, checkExecutable(ctx, "node", "--version"))
	cat.Checks = append(cat.Checks, checkExecutable(ctx, "npm", "--version"))
	cat.Checks = append(cat.Checks, checkOptionalExecutable(ctx, "pnpm", "--version"))
	cat.Checks = append(cat.Checks, checkOptionalExecutable(ctx, "yarn", "--version"))
	cat.Checks = append(cat.Checks, checkOptionalExecutable(ctx, "bun", "--version"))

	// Go
	cat.Checks = append(cat.Checks, checkOptionalExecutable(ctx, "go", "version"))

	// Python
	pyCheck := checkOptionalExecutable(ctx, "python", "--version")
	if pyCheck.Status != StatusPass {
		if runtime.GOOS == "windows" {
			pyCheck = checkOptionalExecutable(ctx, "py", "--version")
		}
		if pyCheck.Status != StatusPass {
			pyCheck = checkOptionalExecutable(ctx, "python3", "--version")
		}
	}
	pyCheck.Name = "Python"
	cat.Checks = append(cat.Checks, pyCheck)

	// Docker
	dockerCheck := checkOptionalExecutable(ctx, "docker", "--version")
	if dockerCheck.Status == StatusPass {
		infoCmd := exec.CommandContext(ctx, "docker", "info")
		if err := infoCmd.Run(); err != nil {
			dockerCheck.Status = StatusWarn
			dockerCheck.Details = "CLI present, but daemon is unreachable"
		} else {
			dockerCheck.Details = "daemon reachable"
		}
	}
	cat.Checks = append(cat.Checks, dockerCheck)

	return cat
}

func checkSecurity() CategoryReport {
	cat := CategoryReport{Category: "Security & Credentials"}

	store, err := credentials.Open()
	if err != nil {
		cat.Checks = append(cat.Checks, CheckItem{
			Name:   "Keyring",
			Status: StatusWarn,
			Error:  fmt.Sprintf("Failed to open credentials store: %v", err),
		})
	} else {
		fallbackFile := filepath.Join(store.Dir(), "secrets.fallback.json")
		if _, statErr := os.Stat(fallbackFile); statErr == nil {
			cat.Checks = append(cat.Checks, CheckItem{
				Name:    "Keyring",
				Status:  StatusWarn,
				Details: "Using 0600 filesystem fallback (OS keychain unavailable)",
			})
		} else {
			cat.Checks = append(cat.Checks, CheckItem{
				Name:    "Keyring",
				Status:  StatusPass,
				Details: "Native OS keychain available",
			})
		}
	}

	providers := []string{"vercel", "render", "netlify"}
	envVars := map[string]string{
		"vercel":  "VERCEL_TOKEN",
		"render":  "RENDER_TOKEN",
		"netlify": "NETLIFY_TOKEN",
	}

	for _, p := range providers {
		item := CheckItem{Name: title(p)}
		if os.Getenv(envVars[p]) != "" {
			item.Status = StatusPass
			item.Details = fmt.Sprintf("Authenticated via %s", envVars[p])
		} else if store != nil {
			if meta, ok, _ := store.GetMeta(p); ok {
				item.Status = StatusPass
				item.Details = fmt.Sprintf("Authenticated (%s)", meta.Method)
			} else {
				item.Status = StatusWarn
				item.Details = "Not configured"
			}
		} else {
			item.Status = StatusWarn
			item.Details = "Not configured"
		}
		cat.Checks = append(cat.Checks, item)
	}

	return cat
}

func checkLLM(ctx context.Context, workspaceRoot string) CategoryReport {
	cat := CategoryReport{Category: "LLM Configuration"}

	cfgPath := filepath.Join(workspaceRoot, ".atlas", "config.json")
	hasConfig := true
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		hasConfig = false
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		cfg = &config.Config{
			LLMProvider:  "anthropic",
			DefaultModel: "",
		}
	}

	store, _ := credentials.Open()

	if !hasConfig || !llm.HasProviderKey(cfg.LLMProvider, store) {
		if detected, ok := llm.DetectConfiguredProvider(cfg.LLMProvider, store); ok {
			cfg.LLMProvider = detected.Name
			if cfg.DefaultModel == "" {
				cfg.DefaultModel = detected.DefaultModel
			}
		}
	}

	displayModel := cfg.DefaultModel
	if displayModel == "" {
		for _, p := range llm.KnownProviders {
			if p.Name == cfg.LLMProvider {
				displayModel = p.DefaultModel
				break
			}
		}
	}

	item := CheckItem{
		Name:    fmt.Sprintf("Model (%s)", displayModel),
		Details: fmt.Sprintf("Provider: %s", cfg.LLMProvider),
	}

	if !llm.HasProviderKey(cfg.LLMProvider, store) && cfg.LLMProvider != "local" {
		item.Status = StatusWarn
		item.Error = "No API key configured — set MISTRAL_API_KEY, ANTHROPIC_API_KEY, OPENAI_API_KEY, etc. or run atlas models set <provider>"
		cat.Checks = append(cat.Checks, item)
		return cat
	}

	client, err := newLLMClient(cfg, store)
	if err != nil {
		item.Status = StatusWarn
		item.Error = fmt.Sprintf("Client initialization failed: %v", err)
		cat.Checks = append(cat.Checks, item)
		return cat
	}

	pingCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	start := time.Now()
	_, err = client.Complete(pingCtx, "You are a test agent.", "ping")
	latency := time.Since(start).Milliseconds()

	if err != nil {
		item.Status = StatusWarn
		item.Error = fmt.Sprintf("API ping failed: %v", err)
	} else {
		item.Status = StatusPass
		item.LatencyMs = latency
		item.Details = fmt.Sprintf("Provider: %s — Connected (latency: %dms)", cfg.LLMProvider, latency)
	}

	cat.Checks = append(cat.Checks, item)
	return cat
}

func checkVersion() CategoryReport {
	cat := CategoryReport{Category: "Atlas Version"}
	current := version.Version
	item := CheckItem{
		Name:    "Atlas CLI",
		Version: current,
	}

	if current == "dev" {
		item.Status = StatusPass
		item.Details = "Running development build"
		cat.Checks = append(cat.Checks, item)
		return cat
	}

	rel, err := updater.CheckUpdate(current)
	if err != nil {
		item.Status = StatusWarn
		item.Details = fmt.Sprintf("Update check skipped: %v", err)
	} else if rel != nil {
		item.Status = StatusWarn
		item.Details = fmt.Sprintf("Update available: %s -> %s (run `atlas update`)", current, rel.TagName)
	} else {
		item.Status = StatusPass
		item.Details = "Up to date with latest release"
	}

	cat.Checks = append(cat.Checks, item)
	return cat
}

func checkExecutable(ctx context.Context, name string, args ...string) CheckItem {
	item := CheckItem{Name: title(name)}
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.Output()
	if err != nil {
		item.Status = StatusFail
		item.Error = "Executable not found or execution failed"
		return item
	}
	item.Status = StatusPass
	item.Version = strings.TrimSpace(string(out))
	return item
}

var checkOptionalExecutable = func(ctx context.Context, name string, args ...string) CheckItem {
	item := CheckItem{Name: title(name)}
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.Output()
	if err != nil {
		item.Status = StatusWarn
		item.Details = "Not installed (optional)"
		return item
	}
	item.Status = StatusPass
	item.Version = strings.TrimSpace(string(out))
	return item
}
