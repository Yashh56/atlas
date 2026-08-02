package tools

import (
	"context"
	"strings"
	"testing"
)

func TestRunDiagnostic_Allowlist(t *testing.T) {
	ctx := context.Background()
	ws := "/fake/workspace"

	tests := []struct {
		name    string
		command string
		args    []string
		wantErr bool
	}{
		{
			name:    "allowed simple command",
			command: "go vet ./...",
			wantErr: false,
		},
		{
			name:    "allowed with complex args",
			command: "go build -n ./...",
			wantErr: false,
		},
		{
			name:    "disallowed arbitrary command",
			command: "rm -rf /",
			wantErr: true,
		},
		{
			name:    "attempt shell chaining inside command",
			command: "go vet ./... && rm -rf /",
			wantErr: true,
		},
		{
			name:    "grep without paths is allowed (fails to match, but doesn't error before exec)",
			command: "grep",
			args:    []string{"-r", "pattern"},
			wantErr: false, // will fail in exec since grep exits 2 without a path, but passes allowlist
		},
		{
			name:    "grep with safe path",
			command: "grep",
			args:    []string{"pattern", "safe/path.go"},
			wantErr: false,
		},
		{
			name:    "grep path traversal escape",
			command: "grep",
			args:    []string{"pattern", "../../../etc/passwd"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := RunDiagnostic{WorkspaceRoot: ws}
			_, err := tool.Execute(ctx, RunDiagnosticInput{
				Command: tt.command,
				Args:    tt.args,
			})

			// If it expects an error (like allowlist block or path traversal), it must fail BEFORE exec (which complains about missing binary in /fake/ws)
			// Actually since /fake/workspace doesn't exist, exec will fail anyway. But we check for OUR specific errors.
			
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !strings.Contains(err.Error(), "allowlist") && !strings.Contains(err.Error(), "invalid path") {
					t.Fatalf("expected allowlist or invalid path error, got: %v", err)
				}
			} else {
				// If it's a valid command, it will fail at exec because we didn't mock exec, but the error will NOT be allowlist/path.
				if err != nil && (strings.Contains(err.Error(), "allowlist") || strings.Contains(err.Error(), "invalid path")) {
					t.Fatalf("expected exec error (since it's a real exec on fake paths), but got validation block: %v", err)
				}
			}
		})
	}
}

func TestToGoAITools_Isolation(t *testing.T) {
	ws := "/fake/workspace"
	tools := ToGoAITools(ws)

	// Ensure exactly 2 tools are provided (read_file, run_diagnostic)
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}

	// Strictly assert no mutating tools leaked into this array
	mutatingNames := map[string]bool{
		"write_file":   true,
		"patch_file":   true,
		"run_command":  true,
		"delete_file":  true,
		"rename_file":  true,
		"analyze_project": true, // Not mutating but not allowed here
	}

	for _, tool := range tools {
		if mutatingNames[tool.Name] {
			t.Fatalf("CRITICAL ISOLATION FAILURE: tool %q found in ToGoAITools output", tool.Name)
		}
	}
}
