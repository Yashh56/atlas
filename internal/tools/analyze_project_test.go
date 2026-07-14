package tools_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Yashh56/atlas/internal/orchestrator"
	"github.com/Yashh56/atlas/internal/session"
	"github.com/Yashh56/atlas/internal/tools"
)

func writeFixture(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("writeFixture %s: %v", name, err)
	}
}

func TestAnalyzeProject(t *testing.T) {
	cases := []struct {
		name           string
		setup          func(dir string)
		wantFramework  *string
		wantLanguage   *string
		wantPM         *string
		wantDocker     bool
	}{
		{
			name: "nextjs with pnpm and typescript",
			setup: func(dir string) {
				writeFixture(t, dir, "package.json", `{
					"dependencies": {"next": "14.0.0", "react": "18.0.0"},
					"devDependencies": {}
				}`)
				writeFixture(t, dir, "tsconfig.json", `{}`)
				writeFixture(t, dir, "pnpm-lock.yaml", ``)
				writeFixture(t, dir, "Dockerfile", `FROM node:18`)
			},
			wantFramework: ptr("nextjs"),
			wantLanguage:  ptr("typescript"),
			wantPM:        ptr("pnpm"),
			wantDocker:    true,
		},
		{
			name: "react with yarn",
			setup: func(dir string) {
				writeFixture(t, dir, "package.json", `{
					"dependencies": {"react": "18.0.0"},
					"devDependencies": {}
				}`)
				writeFixture(t, dir, "yarn.lock", ``)
			},
			wantFramework: ptr("react"),
			wantLanguage:  ptr("javascript"),
			wantPM:        ptr("yarn"),
			wantDocker:    false,
		},
		{
			name: "plain go project",
			setup: func(dir string) {
				writeFixture(t, dir, "go.mod", `module example.com/myapp

go 1.22
`)
			},
			wantFramework: ptr("go"),
			wantLanguage:  ptr("go"),
			wantPM:        nil,
			wantDocker:    false,
		},
		{
			name:          "empty directory",
			setup:         func(dir string) {},
			wantFramework: nil,
			wantLanguage:  nil,
			wantPM:        nil,
			wantDocker:    false,
		},
		{
			name: "package.json with no recognizable framework",
			setup: func(dir string) {
				writeFixture(t, dir, "package.json", `{
					"dependencies": {"lodash": "4.0.0"},
					"devDependencies": {}
				}`)
			},
			wantFramework: nil,
			wantLanguage:  nil,
			wantPM:        ptr("npm"),
			wantDocker:    false,
		},
		{
			name: "generic node project with build script",
			setup: func(dir string) {
				writeFixture(t, dir, "package.json", `{
					"scripts": {"build": "tsc"},
					"dependencies": {},
					"devDependencies": {}
				}`)
			},
			wantFramework: ptr("node"),
			wantLanguage:  ptr("javascript"),
			wantPM:        ptr("npm"),
			wantDocker:    false,
		},
	}

	stub := &session.Session{}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			workspace := t.TempDir()
			tc.setup(workspace)

			sessDir := t.TempDir()
			tool := tools.AnalyzeProject{
				WorkspaceRoot: workspace,
				SessionDir:    sessDir,
			}

			result, err := tool.Execute(context.Background(), stub)
			if err != nil {
				t.Fatalf("unexpected Go error: %v", err)
			}
			if !result.Success {
				t.Fatalf("expected Success=true, got false; Error=%q", result.Error)
			}

			// Reload project.json and check values.
			proj, err := orchestrator.LoadProject(sessDir)
			if err != nil {
				t.Fatalf("LoadProject: %v", err)
			}

			assertStringPtr(t, "Framework", proj.Framework, tc.wantFramework)
			assertStringPtr(t, "Language", proj.Language, tc.wantLanguage)
			assertStringPtr(t, "PackageManager", proj.PackageManager, tc.wantPM)

			if proj.Docker != tc.wantDocker {
				t.Errorf("Docker = %v, want %v", proj.Docker, tc.wantDocker)
			}

			// Git fields must all be nil — not populated until Week 3.
			if proj.Git.Branch != nil || proj.Git.CommitSHA != nil ||
				proj.Git.IsClean != nil || proj.Git.Remote != nil {
				t.Error("expected all git fields to be nil")
			}
		})
	}
}

func assertStringPtr(t *testing.T, field string, got, want *string) {
	t.Helper()
	if want == nil && got == nil {
		return
	}
	if want == nil && got != nil {
		t.Errorf("%s = %q, want nil", field, *got)
		return
	}
	if want != nil && got == nil {
		t.Errorf("%s = nil, want %q", field, *want)
		return
	}
	if *got != *want {
		t.Errorf("%s = %q, want %q", field, *got, *want)
	}
}

func ptr(s string) *string { return &s }
