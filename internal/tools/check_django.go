package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Yashh56/atlas/internal/session"
)

// djangoUsesDatabase reports whether the settings.py content indicates the project
// requires a database. The heuristic: presence of any of django.contrib.auth,
// django.contrib.admin, or django.contrib.sessions in INSTALLED_APPS — these three
// apps all require a working database and ship in virtually every real Django project.
// Their absence is the signal for "genuinely doesn't use a database."
func djangoUsesDatabase(settingsContent string) bool {
	dbApps := []string{
		"django.contrib.auth",
		"django.contrib.admin",
		"django.contrib.sessions",
	}
	for _, app := range dbApps {
		if strings.Contains(settingsContent, app) {
			return true
		}
	}
	return false
}

// CheckDjango validates preconditions for a Django project before building or deploying.
type CheckDjango struct {
	WorkspaceRoot string
	IsDeploy      bool
	ProviderName  string
}

func (c CheckDjango) Name() string { return "check_django" }

func (c CheckDjango) Execute(ctx context.Context, s *session.Session) (ToolResult, error) {
	start := time.Now()
	var failures []string

	// 1 & 2: requirements.txt checks — only enforced for Render deploys
	reqPath := filepath.Join(c.WorkspaceRoot, "requirements.txt")
	reqBytes, err := os.ReadFile(reqPath)
	if err != nil {
		if c.IsDeploy && c.ProviderName == "render" {
			failures = append(failures, "No requirements.txt found. Activate your virtual environment and run: `pip freeze > requirements.txt`")
		}
	} else if c.IsDeploy && c.ProviderName == "render" {
		// Strip null bytes — PowerShell's `pip freeze > requirements.txt` produces
		// UTF-16LE files with a null byte between every character.
		reqContent := strings.ReplaceAll(strings.ToLower(string(reqBytes)), "\x00", "")
		reqContent = strings.ReplaceAll(reqContent, "\r", "")
		settingsMatches, _ := filepath.Glob(filepath.Join(c.WorkspaceRoot, "*", "settings.py"))
		requiresDatabase := true // default safely to true if we can't tell
		if len(settingsMatches) == 1 {
			if b, err := os.ReadFile(settingsMatches[0]); err == nil {
				requiresDatabase = djangoUsesDatabase(string(b))
			}
		}

		hasGunicorn, _ := regexp.MatchString(`(?m)^gunicorn([>=<~].*)?$`, reqContent)
		hasUvicorn, _ := regexp.MatchString(`(?m)^uvicorn([>=<~].*)?$`, reqContent)
		
		var missing []string
		if !hasGunicorn && !hasUvicorn {
			missing = append(missing, "gunicorn (or uvicorn)")
		}

		requiredPkgs := []string{"whitenoise"}
		if requiresDatabase {
			requiredPkgs = append(requiredPkgs, "dj-database-url")
		}

		for _, pkg := range requiredPkgs {
			matched, _ := regexp.MatchString(`(?m)^`+pkg+`([>=<~].*)?$`, reqContent)
			if !matched {
				missing = append(missing, pkg)
			}
		}

		if requiresDatabase {
			psycopgFound := false
			if matched, _ := regexp.MatchString(`(?m)^psycopg2(-binary)?([>=<~].*)?$`, reqContent); matched {
				psycopgFound = true
			}
			if !psycopgFound {
				missing = append(missing, "psycopg2-binary")
			}
		}

		if len(missing) > 0 {
			failures = append(failures, fmt.Sprintf("Missing required packages in requirements.txt: %s.\nRun: python -m pip install %s\nThen update requirements.txt.", strings.Join(missing, ", "), strings.Join(missing, " ")))
		}
	}

	// 3: settings.py checks — enforced for Render and Vercel deploys
	matches, _ := filepath.Glob(filepath.Join(c.WorkspaceRoot, "*", "settings.py"))
	if len(matches) == 0 {
		failures = append(failures, "Could not locate settings.py in any subdirectory.")
	} else if len(matches) > 1 {
		failures = append(failures, "Found multiple settings.py files. Cannot uniquely identify the Django module.")
	} else if c.IsDeploy && (c.ProviderName == "render" || c.ProviderName == "vercel") {
		settingsPath := matches[0]
		b, _ := os.ReadFile(settingsPath)
		settingsContent := string(b)

		// 4: WhiteNoiseMiddleware (only required for Render)
		if c.ProviderName == "render" && !strings.Contains(settingsContent, "WhiteNoiseMiddleware") {
			failures = append(failures, "WhiteNoiseMiddleware not found in MIDDLEWARE. Add 'whitenoise.middleware.WhiteNoiseMiddleware' to MIDDLEWARE in settings.py.")
		}

		// 5: Database discriminator — only check DB config if the project actually uses a DB
		if djangoUsesDatabase(settingsContent) {
			if !strings.Contains(settingsContent, "dj_database_url") {
				providerNameStr := "Render"
				if c.ProviderName == "vercel" {
					providerNameStr = "Vercel"
				}
				failures = append(failures,
					fmt.Sprintf("This project is still using SQLite. %s's disks are ephemeral — anything written to a SQLite "+
						"file is lost on the next deploy or restart. Configure PostgreSQL via dj-database-url before "+
						"deploying.", providerNameStr))
			}
		}

		// 6: SECRET_KEY
		if !regexp.MustCompile(`SECRET_KEY\s*=\s*(os\.environ|os\.getenv)`).MatchString(settingsContent) {
			failures = append(failures, "SECRET_KEY must be read from environment variables (e.g. os.environ.get('SECRET_KEY', '...')). Hardcoded secrets are not allowed.")
		}

		// 7: DEBUG — must be env-controlled, not hardcoded True
		if !regexp.MustCompile(`DEBUG\s*=\s*(os\.environ|os\.getenv)`).MatchString(settingsContent) {
			envVarExample := "RENDER"
			if c.ProviderName == "vercel" {
				envVarExample = "VERCEL"
			}
			failures = append(failures,
				fmt.Sprintf("DEBUG is hardcoded to True (or not env-controlled) in settings.py — this exposes stack traces and "+
					"environment details in production. Set it from an env var, e.g.:\n"+
					"  DEBUG = os.environ.get('%s') is None", envVarExample))
		}

		// 8: ALLOWED_HOSTS
		if c.ProviderName == "render" {
			if !strings.Contains(settingsContent, "RENDER_EXTERNAL_HOSTNAME") {
				failures = append(failures, "ALLOWED_HOSTS does not seem to include RENDER_EXTERNAL_HOSTNAME. Add os.environ.get('RENDER_EXTERNAL_HOSTNAME') to ALLOWED_HOSTS.")
			}
		} else if c.ProviderName == "vercel" {
			if !strings.Contains(settingsContent, "VERCEL_URL") && !strings.Contains(settingsContent, "VERCEL_PROJECT_PRODUCTION_URL") {
				failures = append(failures, "ALLOWED_HOSTS does not seem to include VERCEL_URL. Add os.environ.get('VERCEL_URL') to ALLOWED_HOSTS.")
			}
		}

		// 8.1: STATIC_ROOT
		if !strings.Contains(settingsContent, "STATIC_ROOT") {
			failures = append(failures, "STATIC_ROOT is not configured in settings.py. Django's collectstatic command requires this to function.")
		}

		// 8.5: Root URL Pattern for Health Checks
		urlsPath := filepath.Join(filepath.Dir(settingsPath), "urls.py")
		if bUrls, err := os.ReadFile(urlsPath); err == nil {
			urlsContent := string(bUrls)
			if !strings.Contains(urlsContent, "path('',") && !strings.Contains(urlsContent, `path("",`) && !strings.Contains(urlsContent, `re_path(r'^$'`) {
				failures = append(failures, "No root URL pattern found in urls.py. The default health check (GET /) will report this deployment as unhealthy even if it's working. Add a root view, or configure a custom health-check path.")
			}
		} else {
			failures = append(failures, "urls.py not found next to settings.py. A root URL pattern is required for health checks.")
		}
	}

	// 9: build.sh — only enforced for Render deploys
	if c.IsDeploy && c.ProviderName == "render" {
		buildShPath := filepath.Join(c.WorkspaceRoot, "build.sh")
		buildShBytes, err := os.ReadFile(buildShPath)
		if err != nil {
			failures = append(failures, "build.sh not found. Create a build.sh file containing:\n#!/usr/bin/env bash\nset -o errexit\npython -m pip install -r requirements.txt\npython manage.py collectstatic --no-input\npython manage.py migrate")
		} else {
			buildShContent := string(buildShBytes)
			hasCollect := strings.Contains(buildShContent, "collectstatic")
			hasMigrate := strings.Contains(buildShContent, "migrate")
			if !hasCollect || !hasMigrate {
				failures = append(failures, "build.sh is missing required commands. It should include 'python manage.py collectstatic --no-input' and 'python manage.py migrate'.")
			}
		}
	}

	if len(failures) > 0 {
		return ToolResult{
			Success:  false,
			Error:    "Django project checks failed:\n- " + strings.Join(failures, "\n- "),
			Duration: time.Since(start),
		}, nil
	}

	return ToolResult{
		Success:  true,
		Output:   "Django checks passed.",
		Duration: time.Since(start),
	}, nil
}
