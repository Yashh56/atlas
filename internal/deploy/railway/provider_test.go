package railway

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/Yashh56/atlas/internal/deploy"
	"github.com/Yashh56/atlas/internal/state"
)

func TestRailwayProvider_Deploy_HappyPath(t *testing.T) {
	sessDir := t.TempDir()
	workspaceDir := t.TempDir()

	proj := projectData{
		RailwayProjectID: ptr("proj_123"),
	}
	_ = state.SaveJSON(sessDir, "project.json", &proj)

	runner := &fakeRunnerDynamic{
		lookPaths: map[string]bool{"railway": true},
		runFn: func(dir string, name string, args ...string) (string, error) {
			if name != "railway" {
				return "", errors.New("unexpected command")
			}
			cmd := strings.Join(args, " ")
			if args[0] == "status" {
				return "Project: proj_123 SUCCESS", nil
			}
			if args[0] == "up" {
				if !strings.Contains(cmd, "--detach") {
					t.Errorf("expected --detach in up args: %s", cmd)
				}
				return "Build complete. Deployed.", nil
			}
			if args[0] == "domain" {
				return "https://my-service.up.railway.app", nil
			}
			return "", errors.New("unexpected args: " + cmd)
		},
		runInteractiveFn: func(dir string, name string, args ...string) error { return nil },
	}

	provider := &RailwayProvider{Runner: runner}
	res, err := provider.Deploy(context.Background(), deploy.DeployInput{
		WorkspaceRoot: workspaceDir,
		SessionDir:    sessDir,
	})
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	if res.URL != "https://my-service.up.railway.app" {
		t.Errorf("unexpected URL: %s", res.URL)
	}
	if res.Provider != "railway" {
		t.Errorf("unexpected provider: %s", res.Provider)
	}
}

func TestRailwayProvider_Deploy_UnlinkedInitAndDatabase(t *testing.T) {
	sessDir := t.TempDir()
	workspaceDir := t.TempDir()

	proj := projectData{
		RequiresDatabase: ptr(true),
	}
	_ = state.SaveJSON(sessDir, "project.json", &proj)

	initCalled := false
	dbCalled := false

	runner := &fakeRunnerDynamic{
		lookPaths: map[string]bool{"railway": true},
		runFn: func(dir string, name string, args ...string) (string, error) {
			cmd := strings.Join(args, " ")
			if args[0] == "status" {
				if !initCalled {
					return "", errors.New("No project linked")
				}
				return "Project: proj_new_999 SUCCESS", nil
			}
			if args[0] == "init" {
				initCalled = true
				return "Project ID: proj_new_999", nil
			}
			if args[0] == "add" && strings.Contains(cmd, "postgres") {
				dbCalled = true
				return "Postgres database added", nil
			}
			if args[0] == "up" {
				return "Deployed", nil
			}
			if args[0] == "domain" {
				return `{"domain": "newapp.up.railway.app"}`, nil
			}
			return "", errors.New("unexpected args: " + cmd)
		},
		runInteractiveFn: func(dir string, name string, args ...string) error { return nil },
	}

	provider := &RailwayProvider{Runner: runner}
	res, err := provider.Deploy(context.Background(), deploy.DeployInput{
		WorkspaceRoot: workspaceDir,
		SessionDir:    sessDir,
	})
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}
	if !initCalled {
		t.Error("expected railway init to be called")
	}
	if !dbCalled {
		t.Error("expected database provisioning to be called")
	}
	if res.URL != "https://newapp.up.railway.app" {
		t.Errorf("unexpected URL: %s", res.URL)
	}

	// Verify project ID was saved
	var updatedProj projectData
	_ = state.LoadJSON(sessDir, "project.json", &updatedProj)
	if updatedProj.RailwayProjectID == nil || *updatedProj.RailwayProjectID != "proj_new_999" {
		val := "<nil>"
		if updatedProj.RailwayProjectID != nil {
			val = *updatedProj.RailwayProjectID
		}
		t.Errorf("expected project ID proj_new_999 saved, got: %s", val)
	}
}

func TestRailwayProvider_Deploy_DjangoAutoDatabase(t *testing.T) {
	sessDir := t.TempDir()
	workspaceDir := t.TempDir()

	proj := projectData{
		Framework: ptr("django"),
	}
	_ = state.SaveJSON(sessDir, "project.json", &proj)

	dbCalled := false
	runner := &fakeRunnerDynamic{
		lookPaths: map[string]bool{"railway": true},
		runFn: func(dir string, name string, args ...string) (string, error) {
			cmd := strings.Join(args, " ")
			if args[0] == "status" {
				return "Project: proj_django SUCCESS", nil
			}
			if args[0] == "add" && strings.Contains(cmd, "postgres") {
				dbCalled = true
				return "Postgres added", nil
			}
			if args[0] == "up" {
				return "Deployed", nil
			}
			if args[0] == "domain" {
				return "django-service.up.railway.app", nil
			}
			return "", errors.New("unexpected args: " + cmd)
		},
		runInteractiveFn: func(dir string, name string, args ...string) error { return nil },
	}

	provider := &RailwayProvider{Runner: runner}
	res, err := provider.Deploy(context.Background(), deploy.DeployInput{
		WorkspaceRoot: workspaceDir,
		SessionDir:    sessDir,
	})
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}
	if !dbCalled {
		t.Error("expected database provisioning for Django project")
	}
	if res.URL != "https://django-service.up.railway.app" {
		t.Errorf("unexpected URL: %s", res.URL)
	}
}

func TestRailwayProvider_HealthCheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider := &RailwayProvider{}
	err := provider.HealthCheck(context.Background(), &deploy.Deployment{URL: server.URL})
	if err != nil {
		t.Errorf("expected health check to pass, got: %v", err)
	}
}

func TestRailwayProvider_Rollback(t *testing.T) {
	redeployCalled := false
	runner := &fakeRunnerDynamic{
		lookPaths: map[string]bool{"railway": true},
		runFn: func(dir string, name string, args ...string) (string, error) {
			if name == "railway" && len(args) > 0 && args[0] == "redeploy" {
				redeployCalled = true
				return "Redeploy triggered", nil
			}
			return "", errors.New("unexpected command")
		},
		runInteractiveFn: func(dir string, name string, args ...string) error { return nil },
	}

	provider := &RailwayProvider{Runner: runner}
	err := provider.Rollback(context.Background(), &deploy.Deployment{URL: "https://old.up.railway.app"}, deploy.DeployInput{WorkspaceRoot: t.TempDir()})
	if err != nil {
		t.Fatalf("rollback failed: %v", err)
	}
	if !redeployCalled {
		t.Error("expected railway redeploy to be called")
	}
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	for _, arg := range os.Args[3:] {
		os.Stdout.WriteString(arg + "\n")
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

func TestRailwayProvider_Logs(t *testing.T) {
	origExec := execCommandContext
	execCommandContext = fakeExecCommandContext
	defer func() { execCommandContext = origExec }()

	provider := &RailwayProvider{}
	var buf bytes.Buffer
	err := provider.Logs(context.Background(), &deploy.Deployment{WorkspaceRoot: t.TempDir()}, &buf, true)
	if err != nil {
		t.Fatalf("Logs failed: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "logs") || !strings.Contains(out, "--follow") {
		t.Errorf("expected logs and --follow in output, got: %s", out)
	}
}

func ptr[T any](v T) *T {
	return &v
}

func TestRailwayProvider_WaitForDeploymentReady_LiveHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider := &RailwayProvider{
		HTTPClient: server.Client(),
	}

	err := provider.waitForDeploymentReady(context.Background(), t.TempDir(), server.URL)
	if err != nil {
		t.Fatalf("expected nil error on HTTP 200, got: %v", err)
	}
}

func TestRailwayProvider_WaitForDeploymentReady_FailedStatus(t *testing.T) {
	runner := &fakeRunnerDynamic{
		lookPaths: map[string]bool{"railway": true},
		runFn: func(dir string, name string, args ...string) (string, error) {
			if args[0] == "status" {
				return "Service: web\nDeployment Status: CRASHED", nil
			}
			return "", nil
		},
	}

	provider := &RailwayProvider{
		Runner: runner,
	}

	err := provider.waitForDeploymentReady(context.Background(), t.TempDir(), "http://127.0.0.1:0/unreachable")
	if err == nil {
		t.Fatal("expected error on CRASHED status, got nil")
	}
	if !strings.Contains(err.Error(), "railway deployment failed in cloud") {
		t.Errorf("expected 'railway deployment failed in cloud', got: %v", err)
	}
}

func TestRailwayProvider_WaitForDeploymentReady_Timeout(t *testing.T) {
	runner := &fakeRunnerDynamic{
		lookPaths: map[string]bool{"railway": true},
		runFn: func(dir string, name string, args ...string) (string, error) {
			if args[0] == "status" {
				return "Service: web\nDeployment Status: BUILDING", nil
			}
			return "", nil
		},
	}

	provider := &RailwayProvider{
		Runner:       runner,
		PollInterval: 5 * time.Millisecond,
		PollTimeout:  25 * time.Millisecond,
	}

	err := provider.waitForDeploymentReady(context.Background(), t.TempDir(), "http://127.0.0.1:0/unreachable")
	if err != nil {
		t.Fatalf("expected nil error on timeout, got: %v", err)
	}
}
