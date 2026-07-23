package netlify

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/Yashh56/atlas/internal/config"
	"github.com/Yashh56/atlas/internal/credentials"
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

func (f *fakeRunner) Run(_ context.Context, dir string, name string, args ...string) (string, error) {
	key := name
	if len(args) > 0 {
		key = name + " " + args[0]
	}
	if r, ok := f.runs[key]; ok {
		return r.output, r.err
	}
	return "", errors.New("unexpected command: " + key)
}

func (f *fakeRunner) RunInteractive(_ context.Context, dir string, name string, args ...string) error {
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

func TestEnsureNetlifyAuth_EnvVar(t *testing.T) {
	t.Setenv("NETLIFY_TOKEN", "tok-xxx")
	var out bytes.Buffer

	runner := &fakeRunner{
		lookPaths: map[string]bool{"netlify": true},
	}

	err := EnsureNetlifyAuthFull(context.Background(), nil, &config.Config{}, runner, strings.NewReader(""), &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "env_var") {
		t.Errorf("expected env_var in output, got: %s", out.String())
	}
}

func TestEnsureNetlifyAuth_CLILoginSucceeds(t *testing.T) {
	os.Unsetenv("NETLIFY_TOKEN")
	store := openTempStore(t)

	whoamiCall := 0
	runner := &fakeRunnerDynamic{
		lookPaths: map[string]bool{"netlify": true},
		runFn: func(dir string, name string, args ...string) (string, error) {
			key := name
			if len(args) > 0 {
				key += " " + args[0]
			}
			if key == "netlify api" {
				whoamiCall++
				if whoamiCall == 1 {
					// first call: not logged in
					return "", errors.New("not logged in")
				}
				// second call: logged in
				return `{"slug": "yashh56", "full_name": "Yash"}`, nil
			}
			return "", errors.New("unexpected: " + key)
		},
		runInteractiveFn: func(dir string, name string, args ...string) error {
			if name == "netlify" && len(args) > 0 && args[0] == "login" {
				return nil // login succeeds
			}
			return errors.New("unexpected interactive: " + name)
		},
	}

	var out bytes.Buffer
	err := EnsureNetlifyAuthFull(context.Background(), store, &config.Config{}, runner, strings.NewReader(""), &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "yashh56") {
		t.Errorf("expected account after login, got: %s", out.String())
	}
}

// fakeRunnerDynamic allows per-call logic via function callbacks.
type fakeRunnerDynamic struct {
	lookPaths        map[string]bool
	runFn            func(dir string, name string, args ...string) (string, error)
	runInteractiveFn func(dir string, name string, args ...string) error
}

func (f *fakeRunnerDynamic) LookPath(file string) (string, error) {
	if found, ok := f.lookPaths[file]; ok && found {
		return "/usr/bin/" + file, nil
	}
	return "", errors.New(file + ": not found")
}
func (f *fakeRunnerDynamic) Run(_ context.Context, dir string, name string, args ...string) (string, error) {
	return f.runFn(dir, name, args...)
}
func (f *fakeRunnerDynamic) RunInteractive(_ context.Context, dir string, name string, args ...string) error {
	return f.runInteractiveFn(dir, name, args...)
}
