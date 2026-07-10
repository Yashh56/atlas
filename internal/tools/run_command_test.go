package tools_test

import (
	"context"
	"runtime"
	"strings"
	"testing"

	"github.com/Yashh56/atlas/internal/session"
	"github.com/Yashh56/atlas/internal/tools"
)

func TestRunCommand_Success(t *testing.T) {
	cmd, args := echoCommand("hello")
	tool := tools.RunCommand{Command: cmd, Args: args}

	result, err := tool.Execute(context.Background(), &session.Session{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Errorf("expected Success=true, got false; Error=%q", result.Error)
	}
	if !strings.Contains(result.Output, "hello") {
		t.Errorf("output %q does not contain %q", result.Output, "hello")
	}
}

func TestRunCommand_NonexistentCommand(t *testing.T) {
	tool := tools.RunCommand{Command: "this-command-absolutely-does-not-exist-atlas"}

	result, err := tool.Execute(context.Background(), &session.Session{})
	if err != nil {
		t.Fatalf("unexpected Go error (should be in ToolResult): %v", err)
	}
	if result.Success {
		t.Error("expected Success=false for nonexistent command")
	}
	if result.Error == "" {
		t.Error("expected non-empty Error string for nonexistent command")
	}
}

func TestRunCommand_ContextCancelled(t *testing.T) {
	var cmd string
	var args []string
	if runtime.GOOS == "windows" {
		cmd = "ping"
		args = []string{"-n", "10", "127.0.0.1"}
	} else {
		cmd = "sleep"
		args = []string{"10"}
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	tool := tools.RunCommand{Command: cmd, Args: args}
	result, err := tool.Execute(ctx, &session.Session{})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if result.Success {
		t.Error("expected Success=false for cancelled context")
	}
}

// echoCommand returns the platform-appropriate command and args to echo a string.
func echoCommand(text string) (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd", []string{"/C", "echo " + text}
	}
	return "echo", []string{text}
}
