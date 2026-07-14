package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Yashh56/atlas/internal/build"
	"github.com/Yashh56/atlas/internal/session"
	"github.com/Yashh56/atlas/internal/state"
)

const testTimeout = 5 * 60 * time.Second // 5 minutes

// testJSON mirrors the shape we write out for build.json
type testJSON struct {
	Command    string    `json:"command"`
	ExitCode   int       `json:"exit_code"`
	DurationMs int64     `json:"duration_ms"`
	LogPath    string    `json:"log_path"`
	StartedAt  time.Time `json:"started_at"`
}

// RunTests resolves the test command from framework/package-manager,
// executes it, streams output to a log file, and writes test.json.
type RunTests struct {
	WorkspaceRoot  string
	Framework      string
	PackageManager string
	// SessionDir is where test.json and logs/ will be written.
	SessionDir string
}

// Name returns the canonical tool identifier.
func (r RunTests) Name() string { return "run_tests" }

// Execute runs the test command, writes test.log and test.json.
func (r RunTests) Execute(ctx context.Context, s *session.Session) (ToolResult, error) {
	startedAt := time.Now().UTC()

	// Resolve the command.
	cmdBin, cmdArgs, err := build.ResolveTestCommand(r.Framework, r.PackageManager)
	if err != nil {
		return ToolResult{
			Success:  false,
			Error:    err.Error(),
			Duration: time.Since(startedAt),
		}, nil
	}

	if cmdBin == "" {
		return ToolResult{
			Success:  true,
			Output:   "no tests required",
			Duration: time.Since(startedAt),
		}, nil
	}

	commandStr := cmdBin + " " + strings.Join(cmdArgs, " ")

	// Prepare log file.
	logsDir := filepath.Join(r.SessionDir, "logs")
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		return ToolResult{
			Success:  false,
			Error:    fmt.Sprintf("run_tests: creating logs dir: %v", err),
			Duration: time.Since(startedAt),
		}, nil
	}
	logPath := filepath.Join(logsDir, "test.log")

	// Apply test timeout (wraps the parent context).
	testCtx, cancel := context.WithTimeout(ctx, testTimeout)
	defer cancel()

	// Run via RunCommand, capturing combined output.
	rc := RunCommand{Command: cmdBin, Args: cmdArgs, Dir: r.WorkspaceRoot}
	result, _ := rc.Execute(testCtx, s)

	duration := result.Duration

	// Write log file.
	if writeErr := os.WriteFile(logPath, []byte(result.Output), 0o644); writeErr != nil {
		return ToolResult{
			Success:  false,
			Error:    fmt.Sprintf("run_tests: writing test log: %v", writeErr),
			Duration: duration,
		}, nil
	}

	// Determine exit code.
	exitCode := 0
	if !result.Success {
		exitCode = 1
		// Try to parse "exit code N" from the error string.
		var code int
		if n, _ := fmt.Sscanf(result.Error, "exit code %d", &code); n == 1 {
			exitCode = code
		}
	}

	// Persist test.json.
	ts := testJSON{
		Command:    commandStr,
		ExitCode:   exitCode,
		DurationMs: duration.Milliseconds(),
		LogPath:    logPath,
		StartedAt:  startedAt,
	}
	if r.SessionDir != "" {
		if saveErr := state.SaveJSON(r.SessionDir, "test.json", ts); saveErr != nil {
			return ToolResult{
				Success:  false,
				Error:    fmt.Sprintf("run_tests: saving test.json: %v", saveErr),
				Duration: duration,
			}, nil
		}
	}

	if !result.Success {
		return ToolResult{
			Success:  false,
			Output:   logPath, // Path to logs rather than embedding all of it
			Error:    result.Error,
			Duration: duration,
		}, nil
	}

	return ToolResult{
		Success:  true,
		Output:   logPath,
		Duration: duration,
	}, nil
}
