package deploy

import (
	"bytes"
	"context"
	"os"
	"os/exec"
)

// lookPath wraps exec.LookPath. Separated so vercel_auth.go can call it cleanly.
func lookPath(file string) (string, error) {
	return exec.LookPath(file)
}

// runCapture runs a command and returns its combined stdout+stderr output.
func runCapture(ctx context.Context, name string, args ...string) (string, error) {
	var buf bytes.Buffer
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.String(), err
}

// runInherited runs a command inheriting the parent process's stdin/stdout/stderr,
// so interactive prompts reach the user directly.
func runInherited(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
