package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPathResolutionPriority(t *testing.T) {
	origPathFlag := pathFlag
	defer func() { pathFlag = origPathFlag }()

	cases := []struct {
		name      string
		args      []string
		wantError string
	}{
		{
			name:      "positional overrides path flag",
			args:      []string{"--path", "from-flag", "from-positional", "--model", "local", "--action", "build"},
			wantError: "path \"from-positional\" does not exist",
		},
		{
			name:      "missing path non-interactive",
			args:      []string{"--model", "local", "--action", "build"},
			wantError: "no project path given",
		},
		{
			name:      "path from flag",
			args:      []string{"--path", "some-dir", "--model", "local", "--action", "build"},
			wantError: "", // It will try to run orchestrator and fail on workspace resolve
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pathFlag = ""
			
			cmd := rootCmd
			cmd.SetArgs(tc.args)
			cmd.SetOut(new(bytes.Buffer))
			cmd.SetErr(new(bytes.Buffer))
			
			err := cmd.Execute()
			if tc.wantError != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantError)
				}
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("expected error containing %q, got %q", tc.wantError, err.Error())
				}
			} else {
				if err != nil && strings.Contains(err.Error(), "no project path given") {
					t.Fatalf("expected path to be resolved, got error: %v", err)
				}
			}
		})
	}
}

func TestProviderValidation(t *testing.T) {
	origModel := deployModel
	origAction := deployAction
	origProvider := deployProvider
	origNonInteractive := nonInteractiveFlag
	defer func() {
		deployModel = origModel
		deployAction = origAction
		deployProvider = origProvider
		nonInteractiveFlag = origNonInteractive
	}()

	cases := []struct {
		name      string
		args      []string
		wantError string
	}{
		{
			name:      "planned provider fly fails fast",
			args:      []string{"--path", ".", "--model", "openai", "--action", "deploy", "--provider", "fly"},
			wantError: `provider "fly" is planned for v0.3.0 and not yet supported. Supported providers: netlify, render, vercel`,
		},
		{
			name:      "planned provider railway fails fast",
			args:      []string{"--path", ".", "--model", "openai", "--action", "deploy", "--provider", "railway"},
			wantError: `provider "railway" is planned for v0.3.0 and not yet supported. Supported providers: netlify, render, vercel`,
		},
		{
			name:      "unknown provider fails fast",
			args:      []string{"--path", ".", "--model", "openai", "--action", "deploy", "--provider", "unknown"},
			wantError: `unknown provider "unknown" — supported providers: netlify, render, vercel`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			deployModel = ""
			deployAction = ""
			deployProvider = ""
			nonInteractiveFlag = false

			cmd := rootCmd
			cmd.SetArgs(tc.args)
			cmd.SetOut(new(bytes.Buffer))
			cmd.SetErr(new(bytes.Buffer))

			err := cmd.Execute()
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantError)
			}
			if !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("expected error containing %q, got %q", tc.wantError, err.Error())
			}
		})
	}
}

func TestFlagsRegistration(t *testing.T) {
	flags := []string{"dir", "workspace", "strict-env"}
	for _, f := range flags {
		if rootCmd.Flags().Lookup(f) == nil {
			t.Errorf("expected flag --%s to be registered on rootCmd", f)
		}
	}

	foundAudit := false
	for _, c := range rootCmd.Commands() {
		if c.Name() == "audit" {
			foundAudit = true
			break
		}
	}
	if !foundAudit {
		t.Errorf("expected audit subcommand to be registered on rootCmd")
	}
}

func TestMonorepoResolution_NonInteractive_MultiplePackages(t *testing.T) {
	origModel := deployModel
	origAction := deployAction
	origNonInteractive := nonInteractiveFlag
	origDirFlag := dirFlag
	origWorkspaceFlag := workspaceFlag
	defer func() {
		deployModel = origModel
		deployAction = origAction
		nonInteractiveFlag = origNonInteractive
		dirFlag = origDirFlag
		workspaceFlag = origWorkspaceFlag
	}()

	tempDir := t.TempDir()

	// Create pnpm-workspace.yaml
	pnpmWs := filepath.Join(tempDir, "pnpm-workspace.yaml")
	_ = os.WriteFile(pnpmWs, []byte("packages:\n  - 'apps/*'\n"), 0644)

	// Create apps/web and apps/api
	webDir := filepath.Join(tempDir, "apps", "web")
	_ = os.MkdirAll(webDir, 0755)
	_ = os.WriteFile(filepath.Join(webDir, "package.json"), []byte(`{"name": "web"}`), 0644)

	apiDir := filepath.Join(tempDir, "apps", "api")
	_ = os.MkdirAll(apiDir, 0755)
	_ = os.WriteFile(filepath.Join(apiDir, "package.json"), []byte(`{"name": "api"}`), 0644)

	deployModel = ""
	deployAction = ""
	dirFlag = ""
	workspaceFlag = ""
	nonInteractiveFlag = false

	cmd := rootCmd
	cmd.SetArgs([]string{"--path", tempDir, "--model", "openai", "--action", "build", "--yes"})
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error due to unselected monorepo package in non-interactive mode, got nil")
	}

	expectedSubstrings := []string{
		"monorepo detected",
		"with multiple packages",
		"Please specify target package with --workspace <name> or --dir <path>",
	}
	for _, sub := range expectedSubstrings {
		if !strings.Contains(err.Error(), sub) {
			t.Errorf("expected error to contain %q, got: %s", sub, err.Error())
		}
	}
}

func TestMonorepoResolution_ExplicitSubpackage_BypassesRootPrompt(t *testing.T) {
	origModel := deployModel
	origAction := deployAction
	origNonInteractive := nonInteractiveFlag
	origDirFlag := dirFlag
	origWorkspaceFlag := workspaceFlag
	origPathFlag := pathFlag
	defer func() {
		deployModel = origModel
		deployAction = origAction
		nonInteractiveFlag = origNonInteractive
		dirFlag = origDirFlag
		workspaceFlag = origWorkspaceFlag
		pathFlag = origPathFlag
	}()

	tempDir := t.TempDir()

	// Create pnpm-workspace.yaml
	pnpmWs := filepath.Join(tempDir, "pnpm-workspace.yaml")
	_ = os.WriteFile(pnpmWs, []byte("packages:\n  - 'apps/*'\n"), 0644)

	// Create apps/web and apps/api
	webDir := filepath.Join(tempDir, "apps", "web")
	_ = os.MkdirAll(webDir, 0755)
	_ = os.WriteFile(filepath.Join(webDir, "package.json"), []byte(`{"name": "web"}`), 0644)

	apiDir := filepath.Join(tempDir, "apps", "api")
	_ = os.MkdirAll(apiDir, 0755)
	_ = os.WriteFile(filepath.Join(apiDir, "package.json"), []byte(`{"name": "api"}`), 0644)

	deployModel = ""
	deployAction = ""
	dirFlag = ""
	workspaceFlag = ""
	pathFlag = ""
	nonInteractiveFlag = false

	cmd := rootCmd
	cmd.SetArgs([]string{"--path", webDir, "--model", "openai", "--action", "build", "--yes"})
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))

	err := cmd.Execute()
	if err != nil && strings.Contains(err.Error(), "monorepo detected") && strings.Contains(err.Error(), "with multiple packages") {
		t.Fatalf("expected subpackage path to bypass root prompt, but got monorepo error: %v", err)
	}
}

func TestHasRootAppFramework(t *testing.T) {
	tempDir := t.TempDir()
	if hasRootAppFramework(tempDir, false) || hasRootAppFramework(tempDir, true) {
		t.Errorf("empty dir should not have root app framework")
	}

	// With next.config.js (recognized for both monorepo and non-monorepo root)
	nextConfig := filepath.Join(tempDir, "next.config.js")
	_ = os.WriteFile(nextConfig, []byte("module.exports = {};"), 0644)
	if !hasRootAppFramework(tempDir, false) {
		t.Errorf("expected next.config.js to indicate app framework for non-monorepo")
	}
	if !hasRootAppFramework(tempDir, true) {
		t.Errorf("expected next.config.js to indicate app framework for monorepo root")
	}
	_ = os.Remove(nextConfig)

	// With package.json containing generic build script
	pkgJSON := filepath.Join(tempDir, "package.json")
	_ = os.WriteFile(pkgJSON, []byte(`{"scripts": {"build": "turbo run build"}}`), 0644)
	if !hasRootAppFramework(tempDir, false) {
		t.Errorf("expected generic build script to indicate app framework for non-monorepo")
	}
	if hasRootAppFramework(tempDir, true) {
		t.Errorf("generic build script should be ignored for monorepo root")
	}

	// With package.json containing framework dependency (e.g. express)
	_ = os.WriteFile(pkgJSON, []byte(`{"scripts": {"build": "turbo run build"}, "dependencies": {"express": "^4.18.2"}}`), 0644)
	if !hasRootAppFramework(tempDir, true) {
		t.Errorf("expected framework dependency to indicate app framework for monorepo root")
	}
	_ = os.Remove(pkgJSON)

	// Non-JS app markers: manage.py, Pipfile, uv.lock, poetry.lock, requirements.txt, pyproject.toml, Cargo.toml
	nonJSFiles := []string{"manage.py", "Pipfile", "uv.lock", "poetry.lock", "requirements.txt", "pyproject.toml", "Cargo.toml"}
	for _, f := range nonJSFiles {
		filePath := filepath.Join(tempDir, f)
		_ = os.WriteFile(filePath, []byte(""), 0644)
		if !hasRootAppFramework(tempDir, false) {
			t.Errorf("expected %s to indicate app framework for non-monorepo", f)
		}
		if !hasRootAppFramework(tempDir, true) {
			t.Errorf("expected %s to indicate app framework for monorepo root", f)
		}
		_ = os.Remove(filePath)
	}

	// go.mod is recognized only if !isMonorepoRoot
	goMod := filepath.Join(tempDir, "go.mod")
	_ = os.WriteFile(goMod, []byte("module example.com/app\n"), 0644)
	if !hasRootAppFramework(tempDir, false) {
		t.Errorf("expected go.mod to indicate app framework for non-monorepo")
	}
	if hasRootAppFramework(tempDir, true) {
		t.Errorf("expected go.mod to NOT indicate root app framework for monorepo root")
	}
	_ = os.Remove(goMod)
}

