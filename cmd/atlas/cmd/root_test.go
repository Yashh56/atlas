package cmd

import (
	"bytes"
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
