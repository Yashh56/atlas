package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/Yashh56/atlas/internal/build"
	"github.com/Yashh56/atlas/internal/session"
	"github.com/Yashh56/atlas/internal/state"
)

// ValidateDjangoBuild sets up an ephemeral virtual environment, installs dependencies,
// and runs `manage.py check` to ensure the project can be built and run.
type ValidateDjangoBuild struct {
	WorkspaceRoot string
	SessionDir    string
	VenvPath      string
}

func (v ValidateDjangoBuild) Name() string { return "validate_django_build" }

func (v ValidateDjangoBuild) Execute(ctx context.Context, s *session.Session) (ToolResult, error) {
	start := time.Now()

	// Ensure venv is cleaned up on exit
	defer os.RemoveAll(v.VenvPath)

	// Determine the python command to use for creating the venv
	pyCmd := build.ResolvePythonBinary(v.WorkspaceRoot, "python")

	logsDir := filepath.Join(v.SessionDir, "logs")
	_ = os.MkdirAll(logsDir, 0o755)
	logPath := filepath.Join(logsDir, "build.log")

	// 1. Create ephemeral venv
	createCmd := exec.CommandContext(ctx, pyCmd, "-m", "venv", v.VenvPath)
	createCmd.Dir = v.WorkspaceRoot
	if output, err := createCmd.CombinedOutput(); err != nil {
		_ = os.WriteFile(logPath, output, 0o644)
		if v.SessionDir != "" {
			_ = state.SaveJSON(v.SessionDir, "build.json", buildJSON{
				Command:    "python -m venv " + v.VenvPath,
				ExitCode:   1,
				DurationMs: time.Since(start).Milliseconds(),
				LogPath:    logPath,
				StartedAt:  start,
			})
		}
		return ToolResult{
			Success:  false,
			Error:    fmt.Sprintf("Failed to create ephemeral venv:\n%s\nError: %v", string(output), err),
			Duration: time.Since(start),
		}, nil
	}

	// Paths inside the venv
	binDir := "bin"
	if runtime.GOOS == "windows" {
		binDir = "Scripts"
	}
	pythonBin := filepath.Join(v.VenvPath, binDir, "python")

	// 2. Install requirements
	installCmd := exec.CommandContext(ctx, pythonBin, "-m", "pip", "install", "-r", "requirements.txt")
	installCmd.Dir = v.WorkspaceRoot
	if output, err := installCmd.CombinedOutput(); err != nil {
		_ = os.WriteFile(logPath, output, 0o644)
		if v.SessionDir != "" {
			_ = state.SaveJSON(v.SessionDir, "build.json", buildJSON{
				Command:    "pip install -r requirements.txt",
				ExitCode:   1,
				DurationMs: time.Since(start).Milliseconds(),
				LogPath:    logPath,
				StartedAt:  start,
			})
		}
		return ToolResult{
			Success:  false,
			Error:    fmt.Sprintf("Failed to install requirements in ephemeral venv:\n%s\nError: %v", string(output), err),
			Duration: time.Since(start),
		}, nil
	}
	
	// 3. Run manage.py check
	checkCmd := exec.CommandContext(ctx, pythonBin, "manage.py", "check")
	checkCmd.Dir = v.WorkspaceRoot
	output, err := checkCmd.CombinedOutput()
	
	_ = os.WriteFile(logPath, output, 0o644)
	
	exitCode := 0
	if err != nil {
		exitCode = 1
	}

	// Save build.json
	bs := buildJSON{
		Command:    "python manage.py check",
		ExitCode:   exitCode,
		DurationMs: time.Since(start).Milliseconds(),
		LogPath:    logPath,
		StartedAt:  start,
	}
	if v.SessionDir != "" {
		_ = state.SaveJSON(v.SessionDir, "build.json", bs)
	}

	if err != nil {
		return ToolResult{
			Success:  false,
			Error:    fmt.Sprintf("Django check failed:\n%s\nError: %v", string(output), err),
			Duration: time.Since(start),
		}, nil
	}

	return ToolResult{
		Success:  true,
		Output:   logPath, // Use Output as logPath for consistency
		Duration: time.Since(start),
	}, nil
}
