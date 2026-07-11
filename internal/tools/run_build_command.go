package tools

import (
	"bufio"
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

const buildTimeout = 5 * 60 * time.Second // 5 minutes

// buildJSON mirrors orchestrator.BuildState to avoid an import cycle.
type buildJSON struct {
	Command    string    `json:"command"`
	ExitCode   int       `json:"exit_code"`
	DurationMs int64     `json:"duration_ms"`
	LogPath    string    `json:"log_path"`
	StartedAt  time.Time `json:"started_at"`
}

// RunBuildCommand resolves the build command from framework/package-manager,
// executes it, streams output to a log file, and writes build.json.
type RunBuildCommand struct {
	WorkspaceRoot  string
	Framework      string
	PackageManager string
	// SessionDir is where build.json and logs/ will be written.
	SessionDir string
}

// Name returns the canonical tool identifier.
func (r RunBuildCommand) Name() string { return "run_build_command" }

// Execute runs the build, writes build.log and build.json.
func (r RunBuildCommand) Execute(ctx context.Context, s *session.Session) (ToolResult, error) {
	startedAt := time.Now().UTC()

	// Resolve the command.
	cmdBin, cmdArgs, err := build.ResolveBuildCommand(r.Framework, r.PackageManager)
	if err != nil {
		return ToolResult{
			Success:  false,
			Error:    err.Error(),
			Duration: time.Since(startedAt),
		}, nil
	}

	commandStr := cmdBin + " " + strings.Join(cmdArgs, " ")

	// Prepare log file.
	logsDir := filepath.Join(r.SessionDir, "logs")
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		return ToolResult{
			Success:  false,
			Error:    fmt.Sprintf("run_build_command: creating logs dir: %v", err),
			Duration: time.Since(startedAt),
		}, nil
	}
	logPath := filepath.Join(logsDir, "build.log")

	// Apply build timeout (wraps the parent context).
	buildCtx, cancel := context.WithTimeout(ctx, buildTimeout)
	defer cancel()

	// Run via RunCommand, capturing combined output.
	rc := RunCommand{Command: cmdBin, Args: cmdArgs, Dir: r.WorkspaceRoot}
	result, _ := rc.Execute(buildCtx, s)

	duration := result.Duration

	// Write log file.
	if writeErr := os.WriteFile(logPath, []byte(result.Output), 0o644); writeErr != nil {
		return ToolResult{
			Success:  false,
			Error:    fmt.Sprintf("run_build_command: writing build log: %v", writeErr),
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

	// Persist build.json.
	bs := buildJSON{
		Command:    commandStr,
		ExitCode:   exitCode,
		DurationMs: duration.Milliseconds(),
		LogPath:    logPath,
		StartedAt:  startedAt,
	}
	if r.SessionDir != "" {
		if saveErr := state.SaveJSON(r.SessionDir, "build.json", bs); saveErr != nil {
			return ToolResult{
				Success:  false,
				Error:    fmt.Sprintf("run_build_command: saving build.json: %v", saveErr),
				Duration: duration,
			}, nil
		}
	}

	if !result.Success {
		return ToolResult{
			Success:  false,
			Output:   logPath,
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

// TailLines returns the last n lines of text. Used by the orchestrator to
// populate the error.message field in planner.json without storing the whole log.
func TailLines(text string, n int) string {
	lines := strings.Split(text, "\n")
	if len(lines) <= n {
		return text
	}
	return strings.Join(lines[len(lines)-n:], "\n")
}

// ReadTailLines reads the last n lines of a file efficiently.
func ReadTailLines(path string, n int) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > n {
			lines = lines[1:]
		}
	}
	return strings.Join(lines, "\n"), scanner.Err()
}
