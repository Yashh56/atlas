package tools

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/Yashh56/atlas/internal/session"
	"github.com/Yashh56/atlas/internal/state"
)

// projectData mirrors orchestrator.ProjectState for JSON serialisation.
// It is defined here to avoid an import cycle (tools ↔ orchestrator).
type projectData struct {
	Framework      *string     `json:"framework"`
	Language       *string     `json:"language"`
	Runtime        *string     `json:"runtime"`
	PackageManager *string     `json:"package_manager"`
	Docker         bool        `json:"docker"`
	Git            projectGit  `json:"git"`
}

type projectGit struct {
	Branch    *string `json:"branch"`
	CommitSHA *string `json:"commit_sha"`
	IsClean   *bool   `json:"is_clean"`
	Remote    *string `json:"remote"`
}

// AnalyzeProject inspects the workspace at WorkspaceRoot using heuristic
// detection (no LLM) and writes the result to project.json in SessionDir.
type AnalyzeProject struct {
	WorkspaceRoot string
	// SessionDir is the directory where project.json will be written.
	// If empty, project.json is not written (useful in unit tests that
	// verify the returned ToolResult only).
	SessionDir string
}

// Name returns the canonical tool identifier.
func (a AnalyzeProject) Name() string { return "analyze_project" }

// Execute runs framework/language/tooling detection and persists the result.
func (a AnalyzeProject) Execute(ctx context.Context, s *session.Session) (ToolResult, error) {
	start := time.Now()

	proj := &projectData{}

	// --- List the workspace root to discover top-level files. ---
	listTool := ListDirectory{Path: a.WorkspaceRoot}
	listResult, _ := listTool.Execute(ctx, s)

	var rootEntries []string
	if listResult.Success && listResult.Output != "" {
		rootEntries = strings.Split(strings.TrimSpace(listResult.Output), "\n")
	}

	entrySet := make(map[string]bool, len(rootEntries))
	for _, e := range rootEntries {
		entrySet[e] = true
	}

	// Docker detection.
	proj.Docker = entrySet["Dockerfile"]

	// --- Framework detection ---
	if entrySet["package.json"] {
		proj = a.detectJS(ctx, s, proj, entrySet)
	} else if entrySet["go.mod"] {
		fw := "go"
		lang := "go"
		proj.Framework = &fw
		proj.Language = &lang
	}
	// If neither, leave Framework/Language nil.

	// --- Persist to project.json ---
	if a.SessionDir != "" {
		if err := state.SaveJSON(a.SessionDir, "project.json", proj); err != nil {
			return ToolResult{
				Success:  false,
				Error:    err.Error(),
				Duration: time.Since(start),
			}, nil
		}
	}

	return ToolResult{
		Success:  true,
		Output:   projectSummary(proj.Framework, proj.PackageManager),
		Duration: time.Since(start),
	}, nil
}

// detectJS populates proj fields from package.json + lockfile heuristics.
func (a AnalyzeProject) detectJS(
	ctx context.Context,
	s *session.Session,
	proj *projectData,
	entrySet map[string]bool,
) *projectData {
	readTool := ReadFile{Path: a.WorkspaceRoot + "/package.json"}
	readResult, _ := readTool.Execute(ctx, s)

	if readResult.Success {
		proj = parsePackageJSON(readResult.Output, proj)
	}

	// Language: TypeScript if tsconfig.json exists.
	if entrySet["tsconfig.json"] {
		lang := "typescript"
		proj.Language = &lang
	} else if proj.Framework != nil {
		lang := "javascript"
		proj.Language = &lang
	}

	// Package manager (priority order).
	switch {
	case entrySet["pnpm-lock.yaml"]:
		pm := "pnpm"
		proj.PackageManager = &pm
	case entrySet["yarn.lock"]:
		pm := "yarn"
		proj.PackageManager = &pm
	case entrySet["package-lock.json"]:
		pm := "npm"
		proj.PackageManager = &pm
	default:
		pm := "npm"
		proj.PackageManager = &pm
	}

	return proj
}

type pkgJSON struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
	Scripts         map[string]string `json:"scripts"`
}

func parsePackageJSON(content string, proj *projectData) *projectData {
	var pkg pkgJSON
	if err := json.Unmarshal([]byte(content), &pkg); err != nil {
		return proj
	}

	allDeps := make(map[string]bool)
	for k := range pkg.Dependencies {
		allDeps[k] = true
	}
	for k := range pkg.DevDependencies {
		allDeps[k] = true
	}

	type match struct {
		dep  string
		name string
	}
	matchers := []match{
		{"next", "nextjs"},
		{"react", "react"},
		{"express", "express"},
	}

	for _, m := range matchers {
		if allDeps[m.dep] {
			name := m.name
			proj.Framework = &name
			break
		}
	}

	if proj.Framework == nil && pkg.Scripts["build"] != "" {
		fw := "node"
		proj.Framework = &fw
	}

	return proj
}

func projectSummary(fw, pm *string) string {
	f := "unknown"
	if fw != nil {
		f = *fw
	}
	p := "unknown"
	if pm != nil {
		p = *pm
	}
	return "framework: " + f + ", package_manager: " + p
}
