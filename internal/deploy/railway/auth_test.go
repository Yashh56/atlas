package railway

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

type fakeRunner struct {
	lookPaths map[string]bool
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

func openTempStore(t *testing.T) *credentials.Store {
	t.Helper()
	s, err := credentials.OpenWithDir(t.TempDir())
	if err != nil {
		t.Fatalf("OpenWithDir: %v", err)
	}
	return s
}

func TestEnsureRailwayAuth_TokenEnvVar(t *testing.T) {
	t.Setenv("RAILWAY_TOKEN", "proj-tok-123")
	os.Unsetenv("RAILWAY_API_TOKEN")
	var out bytes.Buffer

	runner := &fakeRunner{
		lookPaths: map[string]bool{"railway": true},
	}

	err := EnsureRailwayAuthFull(context.Background(), nil, &config.Config{}, runner, strings.NewReader(""), &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "RAILWAY_TOKEN") {
		t.Errorf("expected RAILWAY_TOKEN in output, got: %s", out.String())
	}
}

func TestEnsureRailwayAuth_APITokenEnvVar(t *testing.T) {
	os.Unsetenv("RAILWAY_TOKEN")
	t.Setenv("RAILWAY_API_TOKEN", "api-tok-456")
	var out bytes.Buffer

	runner := &fakeRunner{
		lookPaths: map[string]bool{"railway": true},
	}

	err := EnsureRailwayAuthFull(context.Background(), nil, &config.Config{}, runner, strings.NewReader(""), &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "RAILWAY_API_TOKEN") {
		t.Errorf("expected RAILWAY_API_TOKEN in output, got: %s", out.String())
	}
}

func TestEnsureRailwayAuth_StoredCredential(t *testing.T) {
	os.Unsetenv("RAILWAY_TOKEN")
	os.Unsetenv("RAILWAY_API_TOKEN")
	store := openTempStore(t)
	_ = store.SetSecret("railway", "stored-secret-token")
	_ = store.SetMeta(credentials.ProviderCredential{
		Provider: "railway",
		Method:   credentials.MethodStoredToken,
		Account:  "railway-user@example.com",
	})

	var out bytes.Buffer
	runner := &fakeRunner{
		lookPaths: map[string]bool{"railway": true},
	}

	err := EnsureRailwayAuthFull(context.Background(), store, &config.Config{}, runner, strings.NewReader(""), &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "stored_token") || !strings.Contains(out.String(), "railway-user@example.com") {
		t.Errorf("expected stored token account in output, got: %s", out.String())
	}
}

func TestEnsureRailwayAuth_CLISession(t *testing.T) {
	os.Unsetenv("RAILWAY_TOKEN")
	os.Unsetenv("RAILWAY_API_TOKEN")
	store := openTempStore(t)
	var out bytes.Buffer

	runner := &fakeRunner{
		lookPaths: map[string]bool{"railway": true},
		runs: map[string]runResult{
			"railway whoami": {output: "Logged in as Jane Doe (jane@example.com)", err: nil},
		},
	}

	err := EnsureRailwayAuthFull(context.Background(), store, &config.Config{}, runner, strings.NewReader(""), &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "cli_session") || !strings.Contains(out.String(), "jane@example.com") {
		t.Errorf("expected cli session with account in output, got: %s", out.String())
	}
}

func TestEnsureRailwayAuth_UnauthenticatedInteractiveLogin(t *testing.T) {
	os.Unsetenv("RAILWAY_TOKEN")
	os.Unsetenv("RAILWAY_API_TOKEN")
	store := openTempStore(t)

	whoamiCalls := 0
	runner := &fakeRunnerDynamic{
		lookPaths: map[string]bool{"railway": true},
		runFn: func(dir string, name string, args ...string) (string, error) {
			if name == "railway" && len(args) > 0 && args[0] == "whoami" {
				whoamiCalls++
				if whoamiCalls == 1 {
					return "", errors.New("not logged in")
				}
				return "Logged in as dev@railway.app", nil
			}
			return "", errors.New("unexpected command")
		},
		runInteractiveFn: func(dir string, name string, args ...string) error {
			if name == "railway" && len(args) > 0 && args[0] == "login" {
				return nil
			}
			return errors.New("unexpected interactive command")
		},
	}

	var out bytes.Buffer
	err := EnsureRailwayAuthFull(context.Background(), store, &config.Config{}, runner, strings.NewReader(""), &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "dev@railway.app") {
		t.Errorf("expected account in output after login, got: %s", out.String())
	}
}

func TestEnsureRailwayAuth_MissingCLIInstallDeclined(t *testing.T) {
	os.Unsetenv("RAILWAY_TOKEN")
	os.Unsetenv("RAILWAY_API_TOKEN")
	var out bytes.Buffer

	declinedRunner := &fakeRunner{
		lookPaths: map[string]bool{"node": true, "npm": true},
	}
	err := EnsureRailwayAuthFull(context.Background(), nil, &config.Config{}, declinedRunner, strings.NewReader("n\n"), &out)
	if err == nil {
		t.Fatal("expected error on declined install, got nil")
	}
}

func TestParseRailwayAccount(t *testing.T) {
	cases := []struct {
		input    string
		expected string
		wantErr  bool
	}{
		{input: `{"name":"John Doe","email":"john@example.com"}`, expected: "john@example.com"},
		{input: `{"name":"Jane Doe"}`, expected: "Jane Doe"},
		{input: "Logged in as John Doe (john@example.com)", expected: "john@example.com"},
		{input: "Logged in as admin@company.io", expected: "admin@company.io"},
		{input: "user@domain.com", expected: "user@domain.com"},
		{input: "Not logged in", wantErr: true},
		{input: "", wantErr: true},
	}

	for _, c := range cases {
		acc, err := parseRailwayAccount(c.input)
		if (err != nil) != c.wantErr {
			t.Errorf("parseRailwayAccount(%q) error = %v, wantErr %v", c.input, err, c.wantErr)
			continue
		}
		if !c.wantErr && acc != c.expected {
			t.Errorf("parseRailwayAccount(%q) = %q, expected %q", c.input, acc, c.expected)
		}
	}
}

func TestResolveRailwayInstallCommand(t *testing.T) {
	// 1. Windows with scoop
	wScoop := &fakeRunner{lookPaths: map[string]bool{"scoop": true}}
	cmd := resolveRailwayInstallCommandForOS(wScoop, "windows")
	if strings.Join(cmd, " ") != "scoop install railway" {
		t.Errorf("expected scoop install railway, got: %v", cmd)
	}

	// 2. Windows with winget (no scoop)
	wWinget := &fakeRunner{lookPaths: map[string]bool{"winget": true}}
	cmd = resolveRailwayInstallCommandForOS(wWinget, "windows")
	if strings.Join(cmd, " ") != "winget install --id Railway.Railway -e" {
		t.Errorf("expected winget install, got: %v", cmd)
	}

	// 3. Windows with npm (no scoop, no winget)
	wNpm := &fakeRunner{lookPaths: map[string]bool{"npm": true}}
	cmd = resolveRailwayInstallCommandForOS(wNpm, "windows")
	if strings.Join(cmd, " ") != "npm install -g @railway/cli" {
		t.Errorf("expected npm install, got: %v", cmd)
	}

	// 4. Windows with none (default scoop)
	wNone := &fakeRunner{lookPaths: map[string]bool{}}
	cmd = resolveRailwayInstallCommandForOS(wNone, "windows")
	if strings.Join(cmd, " ") != "scoop install railway" {
		t.Errorf("expected default scoop install railway, got: %v", cmd)
	}

	// 5. Darwin with brew
	dBrew := &fakeRunner{lookPaths: map[string]bool{"brew": true}}
	cmd = resolveRailwayInstallCommandForOS(dBrew, "darwin")
	if strings.Join(cmd, " ") != "brew install railway" {
		t.Errorf("expected brew install railway, got: %v", cmd)
	}

	// 6. Darwin with npm
	dNpm := &fakeRunner{lookPaths: map[string]bool{"npm": true}}
	cmd = resolveRailwayInstallCommandForOS(dNpm, "darwin")
	if strings.Join(cmd, " ") != "npm install -g @railway/cli" {
		t.Errorf("expected npm install on darwin, got: %v", cmd)
	}

	// 7. Darwin with none (default brew)
	dNone := &fakeRunner{lookPaths: map[string]bool{}}
	cmd = resolveRailwayInstallCommandForOS(dNone, "darwin")
	if strings.Join(cmd, " ") != "brew install railway" {
		t.Errorf("expected default brew on darwin, got: %v", cmd)
	}

	// 8. Linux with brew
	lBrew := &fakeRunner{lookPaths: map[string]bool{"brew": true}}
	cmd = resolveRailwayInstallCommandForOS(lBrew, "linux")
	if strings.Join(cmd, " ") != "brew install railway" {
		t.Errorf("expected brew install on linux, got: %v", cmd)
	}

	// 9. Linux with none (default npm)
	lNone := &fakeRunner{lookPaths: map[string]bool{}}
	cmd = resolveRailwayInstallCommandForOS(lNone, "linux")
	if strings.Join(cmd, " ") != "npm install -g @railway/cli" {
		t.Errorf("expected default npm on linux, got: %v", cmd)
	}

	// 10. resolveRailwayInstallCommand uses current runtime OS
	cmdCurrent := resolveRailwayInstallCommand(wScoop)
	if len(cmdCurrent) == 0 {
		t.Errorf("expected non-empty command for current OS")
	}
}
