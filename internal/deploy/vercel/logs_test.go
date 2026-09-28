package vercel

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/Yashh56/atlas/internal/deploy"
)

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	// Print args for verification
	for _, arg := range os.Args[3:] {
		os.Stdout.WriteString(arg + "\n")
	}

	// Simulate a long running process if --follow is passed
	follow := false
	for _, arg := range os.Args {
		if arg == "--follow" {
			follow = true
		}
	}

	if follow {
		// Wait to be killed by context cancellation
		time.Sleep(1 * time.Hour)
	}
	os.Exit(0)
}

func fakeExecCommandContext(ctx context.Context, command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.CommandContext(ctx, os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

func TestVercelProvider_Logs(t *testing.T) {
	// Temporarily override the command constructor
	originalExec := execCommandContext
	execCommandContext = fakeExecCommandContext
	defer func() { execCommandContext = originalExec }()

	t.Run("without follow", func(t *testing.T) {
		p := &VercelProvider{}
		d := &deploy.Deployment{URL: "https://foo.vercel.app"}
		var buf bytes.Buffer

		err := p.Logs(context.Background(), d, &buf, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := buf.String()
		if !bytes.Contains(buf.Bytes(), []byte("logs")) {
			t.Errorf("expected 'logs' in args, got: %s", output)
		}
		if !bytes.Contains(buf.Bytes(), []byte("https://foo.vercel.app")) {
			t.Errorf("expected url in args, got: %s", output)
		}
		if bytes.Contains(buf.Bytes(), []byte("--follow")) {
			t.Errorf("did not expect '--follow' in args, got: %s", output)
		}
	})

	t.Run("with follow and context cancellation", func(t *testing.T) {
		p := &VercelProvider{}
		d := &deploy.Deployment{URL: "https://foo.vercel.app"}
		var buf bytes.Buffer

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		err := p.Logs(ctx, d, &buf, true)
		if err == nil {
			t.Fatalf("expected context cancellation error, got nil")
		}

		output := buf.String()
		if !bytes.Contains(buf.Bytes(), []byte("--follow")) {
			t.Errorf("expected '--follow' in args, got: %s", output)
		}
	})
}
