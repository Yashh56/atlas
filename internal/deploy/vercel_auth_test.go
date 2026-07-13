package deploy_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/Yashh56/atlas/internal/config"
	"github.com/Yashh56/atlas/internal/credentials"
	"github.com/Yashh56/atlas/internal/deploy"
)

// fakeRunner implements deploy.CommandRunner for testing — no real exec calls.
type fakeRunner struct {
	lookPaths map[string]bool // file -> found?
	runs      map[string]runResult
}

type runResult struct {
	output string
	err    error
}

func (f *fakeRunner) LookPath(file string) (string, error) {
	if found, ok := f.lookPaths[file]; ok && found {
		return "/usr/bin/" + file, nil
	}
	return "", errors.New(file + ": not found")
}

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) (string, error) {
	key := name
	if len(args) > 0 {
		key = name + " " + args[0]
	}
	if r, ok := f.runs[key]; ok {
		return r.output, r.err
	}
	return "", errors.New("unexpected command: " + key)
}

func (f *fakeRunner) RunInteractive(_ context.Context, name string, args ...string) error {
	key := name
	if len(args) > 0 {
		key = name + " " + args[0]
	}
	if r, ok := f.runs[key]; ok {
		return r.err
	}
	return errors.New("unexpected interactive command: " + key)
}

func openTempStore(t *testing.T) *credentials.Store {
	t.Helper()
	s, err := credentials.OpenWithDir(t.TempDir())
	if err != nil {
		t.Fatalf("OpenWithDir: %v", err)
	}
	return s
}

// --- Tests ---

func TestEnsureVercelAuth_EnvVar(t *testing.T) {
	t.Setenv("VERCEL_TOKEN", "tok-xxx")
	var out bytes.Buffer
	err := deploy.EnsureVercelAuthFull(context.Background(), nil, &config.Config{}, &fakeRunner{}, strings.NewReader(""), &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "env_var") {
		t.Errorf("expected env_var in output, got: %s", out.String())
	}
}

func TestEnsureVercelAuth_StoredToken(t *testing.T) {
	// Make sure VERCEL_TOKEN is not set.
	os.Unsetenv("VERCEL_TOKEN")
	store := openTempStore(t)
	_ = store.SetMeta(credentials.ProviderCredential{
		Provider: "vercel",
		Method:   credentials.MethodStoredToken,
		Account:  "stored@example.com",
	})

	var out bytes.Buffer
	err := deploy.EnsureVercelAuthFull(context.Background(), store, &config.Config{}, &fakeRunner{}, strings.NewReader(""), &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "stored_token") {
		t.Errorf("expected stored_token in output, got: %s", out.String())
	}
}

func TestEnsureVercelAuth_CLIMissingUserDeclines(t *testing.T) {
	os.Unsetenv("VERCEL_TOKEN")
	store := openTempStore(t)
	runner := &fakeRunner{
		lookPaths: map[string]bool{"npm": true, "node": true}, // npm/node present
		// vercel not in lookPaths → not found
	}

	var out bytes.Buffer
	err := deploy.EnsureVercelAuthFull(context.Background(), store, &config.Config{}, runner, strings.NewReader("N\n"), &out)
	if err == nil {
		t.Fatal("expected error when user declines install")
	}
	if !strings.Contains(err.Error(), "declined") {
		t.Errorf("expected 'declined' in error, got: %v", err)
	}
}

func TestEnsureVercelAuth_CLIAlreadyLoggedIn(t *testing.T) {
	os.Unsetenv("VERCEL_TOKEN")
	store := openTempStore(t)
	runner := &fakeRunner{
		lookPaths: map[string]bool{"vercel": true},
		runs: map[string]runResult{
			"vercel whoami": {output: "you@example.com", err: nil},
		},
	}

	var out bytes.Buffer
	err := deploy.EnsureVercelAuthFull(context.Background(), store, &config.Config{}, runner, strings.NewReader(""), &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "cli_session") {
		t.Errorf("expected cli_session in output, got: %s", out.String())
	}
	if !strings.Contains(out.String(), "you@example.com") {
		t.Errorf("expected account in output, got: %s", out.String())
	}
	// Verify metadata was written.
	meta, ok, err := store.GetMeta("vercel")
	if err != nil || !ok {
		t.Fatalf("GetMeta: err=%v ok=%v", err, ok)
	}
	if meta.Account != "you@example.com" {
		t.Errorf("account: want %q got %q", "you@example.com", meta.Account)
	}
}

func TestEnsureVercelAuth_CLILoginSucceeds(t *testing.T) {
	os.Unsetenv("VERCEL_TOKEN")
	store := openTempStore(t)

	whoamiCall := 0
	runner := &fakeRunnerDynamic{
		lookPaths: map[string]bool{"vercel": true},
		runFn: func(name string, args ...string) (string, error) {
			key := name
			if len(args) > 0 {
				key += " " + args[0]
			}
			if key == "vercel whoami" {
				whoamiCall++
				if whoamiCall == 1 {
					// first call: not logged in
					return "", errors.New("not logged in")
				}
				// second call: logged in
				return "fresh@example.com", nil
			}
			return "", errors.New("unexpected: " + key)
		},
		runInteractiveFn: func(name string, args ...string) error {
			if name == "vercel" && len(args) > 0 && args[0] == "login" {
				return nil // login succeeds
			}
			return errors.New("unexpected interactive: " + name)
		},
	}

	var out bytes.Buffer
	err := deploy.EnsureVercelAuthFull(context.Background(), store, &config.Config{}, runner, strings.NewReader(""), &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "fresh@example.com") {
		t.Errorf("expected account after login, got: %s", out.String())
	}
}

func TestEnsureVercelAuth_CLILoginFails(t *testing.T) {
	os.Unsetenv("VERCEL_TOKEN")
	store := openTempStore(t)
	runner := &fakeRunnerDynamic{
		lookPaths: map[string]bool{"vercel": true},
		runFn: func(name string, args ...string) (string, error) {
			return "", errors.New("not logged in")
		},
		runInteractiveFn: func(name string, args ...string) error {
			return errors.New("login failed")
		},
	}

	var out bytes.Buffer
	err := deploy.EnsureVercelAuthFull(context.Background(), store, &config.Config{}, runner, strings.NewReader(""), &out)
	if err == nil {
		t.Fatal("expected error when login fails")
	}
	if !strings.Contains(err.Error(), "login failed") {
		t.Errorf("expected 'login failed' in error, got: %v", err)
	}
}

// fakeRunnerDynamic allows per-call logic via function callbacks.
type fakeRunnerDynamic struct {
	lookPaths        map[string]bool
	runFn            func(name string, args ...string) (string, error)
	runInteractiveFn func(name string, args ...string) error
}

func (f *fakeRunnerDynamic) LookPath(file string) (string, error) {
	if found, ok := f.lookPaths[file]; ok && found {
		return "/usr/bin/" + file, nil
	}
	return "", errors.New(file + ": not found")
}
func (f *fakeRunnerDynamic) Run(_ context.Context, name string, args ...string) (string, error) {
	return f.runFn(name, args...)
}
func (f *fakeRunnerDynamic) RunInteractive(_ context.Context, name string, args ...string) error {
	return f.runInteractiveFn(name, args...)
}
