package tools

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/zendev-sh/goai"

	"github.com/Yashh56/atlas/internal/session"
)

var allowedDiagnostics = map[string]bool{
	"go vet ./...":      true,
	"go build -n ./...": true,
	"npx tsc --noEmit":  true,
	"grep":              true,
}

// RunDiagnostic is a bounded, read-only tool for the FixCode agentic loop.
type RunDiagnostic struct {
	WorkspaceRoot string
}

func (r RunDiagnostic) Name() string { return "run_diagnostic" }

type RunDiagnosticInput struct {
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
}

func (r RunDiagnostic) Execute(ctx context.Context, input RunDiagnosticInput) (string, error) {
	if !allowedDiagnostics[input.Command] {
		return "", fmt.Errorf("command %q is not on the allowlist", input.Command)
	}

	cmdName := input.Command
	args := input.Args

	// Special handling for the grep command
	if input.Command == "grep" {
		cmdName = "grep"
		// A safer approach: parse args sequentially.
		var safeArgs []string
		patternSeen := false
		for _, arg := range args {
			// Options
			if len(arg) > 0 && arg[0] == '-' {
				safeArgs = append(safeArgs, arg)
				continue
			}
			
			// First non-option is pattern
			if !patternSeen {
				safeArgs = append(safeArgs, arg)
				patternSeen = true
				continue
			}
			
			// Subsequent non-options are paths
			absPath, _, err := ResolveWorkspacePath(r.WorkspaceRoot, arg)
			if err != nil {
				return "", fmt.Errorf("invalid path %q in grep arguments: %w", arg, err)
			}
			safeArgs = append(safeArgs, absPath)
		}
		args = safeArgs
	} else {
		// For allowlisted exact commands like "go vet ./...", the input.Command is literally "go vet ./..."
		// But exec.Command needs the binary name and args separated.
		// Since our allowlist entries are exact strings, we split them here for execution.
		// Note: The allowlist has exact literal strings, so we know they are safe.
		if input.Command == "go vet ./..." {
			cmdName = "go"
			args = []string{"vet", "./..."}
		} else if input.Command == "go build -n ./..." {
			cmdName = "go"
			args = []string{"build", "-n", "./..."}
		} else if input.Command == "npx tsc --noEmit" {
			cmdName = "npx"
			args = []string{"tsc", "--noEmit"}
		}
	}

	cmd := exec.CommandContext(ctx, cmdName, args...)
	cmd.Dir = r.WorkspaceRoot

	out, err := cmd.CombinedOutput()
	// Diagnostic commands (like go vet or grep) often exit with status 1 if they find issues/no match,
	// which is the exact output we want to return to the model, not a Go error!
	if err != nil {
		if len(out) == 0 {
			return "", fmt.Errorf("failed to run diagnostic: %w", err)
		}
	}

	return string(out), nil
}

// ToGoAITools exposes the read-only bounded tools for the FixCode agentic loop.
func ToGoAITools(workspaceRoot string) []goai.Tool {
	readTool := goai.NewTool(
		"read_file",
		"Read the contents of a file in the workspace to investigate an issue.",
		func(ctx context.Context, input struct {
			Path string `json:"path"`
		}) (string, error) {
			absPath, _, err := ResolveWorkspacePath(workspaceRoot, input.Path)
			if err != nil {
				return "", err
			}
			
			// Execute the existing ReadFile tool using the safely resolved absolute path
			res, err := ReadFile{Path: absPath}.Execute(ctx, &session.Session{})
			if err != nil {
				return "", err
			}
			if !res.Success {
				return "", fmt.Errorf("%s", res.Error)
			}
			return res.Output, nil
		},
	)

	diagTool := goai.NewTool(
		"run_diagnostic",
		"Run a read-only diagnostic command (e.g. go vet ./..., npx tsc --noEmit, grep) to investigate an error. For grep, pass 'grep' as command and use args for options/pattern/path.",
		func(ctx context.Context, input RunDiagnosticInput) (string, error) {
			return RunDiagnostic{WorkspaceRoot: workspaceRoot}.Execute(ctx, input)
		},
	)

	return []goai.Tool{readTool, diagTool}
}
