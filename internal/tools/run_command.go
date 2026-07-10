package tools

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"

	"github.com/Yashh56/atlas/internal/session"
)

// RunCommand executes an arbitrary command and captures combined stdout+stderr.
type RunCommand struct {
	Command string   // executable name or path
	Args    []string // arguments passed to the executable
	Dir     string   // working directory; "" means current directory
}

// Name returns the canonical tool identifier.
func (r RunCommand) Name() string { return "run_command" }

// Execute runs Command with Args in Dir, captures combined stdout+stderr, and
// returns a ToolResult. Context cancellation terminates the process. A non-zero
// exit code is represented as Success=false + Error string — it is not a Go
// error. Only infrastructure failures (e.g. exec not found, dir missing) cause
// a non-nil returned error.
func (r RunCommand) Execute(ctx context.Context, _ *session.Session) (ToolResult, error) {
	start := time.Now()

	cmd := exec.CommandContext(ctx, r.Command, r.Args...)
	cmd.Dir = r.Dir

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	runErr := cmd.Run()
	duration := time.Since(start)
	output := buf.String()

	if runErr == nil {
		return ToolResult{
			Success:  true,
			Output:   output,
			Duration: duration,
		}, nil
	}

	var exitErr *exec.ExitError
	if errors.As(runErr, &exitErr) {
		// The command ran but exited with a non-zero code — not a Go error.
		return ToolResult{
			Success:  false,
			Output:   output,
			Error:    fmt.Sprintf("exit code %d", exitErr.ExitCode()),
			Duration: duration,
		}, nil
	}

	// Could not start the process (binary missing, bad dir, ctx cancelled, …).
	// Return as ToolResult so callers have a single result path.
	return ToolResult{
		Success:  false,
		Output:   output,
		Error:    runErr.Error(),
		Duration: duration,
	}, nil
}
