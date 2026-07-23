package deploy

import (
	"bytes"
	"context"
	"os"
	"os/exec"
)

// CommandRunner is the interface for running shell commands, injected for testability.
type CommandRunner interface {
	LookPath(file string) (string, error)
	Run(ctx context.Context, dir string, name string, args ...string) (string, error)
	RunInteractive(ctx context.Context, dir string, name string, args ...string) error
}

// OSCommandRunner is the real CommandRunner that calls the actual OS.
type OSCommandRunner struct{}

func (OSCommandRunner) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

func (OSCommandRunner) Run(ctx context.Context, dir string, name string, args ...string) (string, error) {
	return runCapture(ctx, dir, name, args...)
}

func (OSCommandRunner) RunInteractive(ctx context.Context, dir string, name string, args ...string) error {
	return runInherited(ctx, dir, name, args...)
}

// runCapture runs a command and returns its combined stdout+stderr output.
func runCapture(ctx context.Context, dir string, name string, args ...string) (string, error) {
	var buf bytes.Buffer
	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.String(), err
}

// runInherited runs a command inheriting the parent process's stdin/stdout/stderr,
// so interactive prompts reach the user directly.
func runInherited(ctx context.Context, dir string, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
