package tools_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Yashh56/atlas/internal/session"
	"github.com/Yashh56/atlas/internal/tools"
)

func TestCheckDjango_Incomplete(t *testing.T) {
	wsRoot, _ := filepath.Abs("../../fixtures/django-incomplete")
	// Must be a Render deploy for the full checklist to apply.
	chk := tools.CheckDjango{WorkspaceRoot: wsRoot, IsDeploy: true, ProviderName: "render"}
	sess := session.New(wsRoot)

	res, err := chk.Execute(context.Background(), sess)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Success {
		t.Fatalf("expected check to fail, but it succeeded")
	}

	// Verify the specific failures that remain — note the DB message is now
	// the SQLite-ephemeral-disk message (not "dj_database_url not found").
	failures := []string{
		"Missing required packages in requirements.txt",
		"gunicorn (or uvicorn), whitenoise, dj-database-url",
		"WhiteNoiseMiddleware not found",
		"still using SQLite",
		"SECRET_KEY must be read from environment variables",
		"ALLOWED_HOSTS does not seem to include RENDER_EXTERNAL_HOSTNAME",
		"build.sh not found",
	}

	for _, f := range failures {
		if !strings.Contains(res.Error, f) {
			t.Errorf("expected error to contain %q, got: %s", f, res.Error)
		}
	}
}

// TestCheckDjango_NoDBApps verifies that a Django project with no DB-requiring
// apps in INSTALLED_APPS does NOT trigger the SQLite/Postgres failure.
func TestCheckDjango_NoDBApps(t *testing.T) {
	wsRoot, _ := filepath.Abs("../../fixtures/django-no-db")
	chk := tools.CheckDjango{WorkspaceRoot: wsRoot, IsDeploy: true, ProviderName: "render"}
	sess := session.New(wsRoot)

	res, err := chk.Execute(context.Background(), sess)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// It will fail on other checks (no requirements.txt, etc.) but must NOT
	// contain the SQLite message.
	if strings.Contains(res.Error, "still using SQLite") {
		t.Errorf("DB check should be skipped for projects with no DB-requiring apps, got: %s", res.Error)
	}
}

// TestCheckDjango_DebugHardcoded verifies that a hardcoded DEBUG=True triggers a failure.
func TestCheckDjango_DebugHardcoded(t *testing.T) {
	wsRoot, _ := filepath.Abs("../../fixtures/django-debug-true")
	chk := tools.CheckDjango{WorkspaceRoot: wsRoot, IsDeploy: true, ProviderName: "render"}
	sess := session.New(wsRoot)

	res, err := chk.Execute(context.Background(), sess)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Success {
		t.Fatalf("expected check to fail due to hardcoded DEBUG, but succeeded")
	}
	if !strings.Contains(res.Error, "DEBUG is hardcoded") {
		t.Errorf("expected error to contain 'DEBUG is hardcoded', got: %s", res.Error)
	}
}

func TestCheckDjango_Ready(t *testing.T) {
	wsRoot, _ := filepath.Abs("../../fixtures/django-ready")
	chk := tools.CheckDjango{WorkspaceRoot: wsRoot, IsDeploy: true, ProviderName: "render"}
	sess := session.New(wsRoot)

	res, err := chk.Execute(context.Background(), sess)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !res.Success {
		t.Errorf("Expected ready project to pass, but got failures:\n%s", res.Error)
	}
}

func TestCheckDjango_NoRoot(t *testing.T) {
	wsRoot, _ := filepath.Abs("../../fixtures/django-no-root")
	chk := tools.CheckDjango{WorkspaceRoot: wsRoot, IsDeploy: true, ProviderName: "render"}
	sess := session.New(wsRoot)

	res, err := chk.Execute(context.Background(), sess)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Success {
		t.Error("Expected failure for missing root URL pattern, but it passed")
	} else if !strings.Contains(res.Error, "No root URL pattern found") {
		t.Errorf("Expected 'No root URL pattern found' error, got:\n%s", res.Error)
	}
}

func TestCheckDjango_MissingUrls(t *testing.T) {
	wsRoot, _ := filepath.Abs("../../fixtures/django-missing-urls")
	chk := tools.CheckDjango{WorkspaceRoot: wsRoot, IsDeploy: true, ProviderName: "render"}
	sess := session.New(wsRoot)

	res, err := chk.Execute(context.Background(), sess)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Success {
		t.Error("Expected failure for missing urls.py, but it passed")
	} else if !strings.Contains(res.Error, "urls.py not found next to settings.py") {
		t.Errorf("Expected 'urls.py not found' error, got:\n%s", res.Error)
	}
}

func TestCheckDjango_Vercel(t *testing.T) {
	wsRoot, _ := filepath.Abs("../../fixtures/django-incomplete")
	chk := tools.CheckDjango{WorkspaceRoot: wsRoot, IsDeploy: true, ProviderName: "vercel"}
	sess := session.New(wsRoot)

	res, err := chk.Execute(context.Background(), sess)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Success {
		t.Fatalf("expected check to fail, but it succeeded")
	}

	failures := []string{
		"still using SQLite. Vercel's disks are ephemeral",
		"DEBUG = os.environ.get('VERCEL') is None",
		"ALLOWED_HOSTS does not seem to include VERCEL_URL",
		"STATIC_ROOT is not configured",
	}

	for _, f := range failures {
		if !strings.Contains(res.Error, f) {
			t.Errorf("expected error to contain %q, got: %s", f, res.Error)
		}
	}

	// Vercel should NOT require WhiteNoise or build.sh
	if strings.Contains(res.Error, "WhiteNoiseMiddleware not found") {
		t.Errorf("expected Vercel check to SKIP WhiteNoise requirement, but got it")
	}
	if strings.Contains(res.Error, "build.sh not found") {
		t.Errorf("expected Vercel check to SKIP build.sh requirement, but got it")
	}
}
