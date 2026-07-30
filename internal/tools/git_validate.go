package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/Yashh56/atlas/internal/session"
	"github.com/Yashh56/atlas/internal/state"
)

// GitValidate populates the git block in project.json using local git commands.
// It reuses RunCommand so there is exactly one code path for shelling out.
type GitValidate struct {
	WorkspaceRoot string // path to the project directory
	GitRoot       string // path returned by workspace.Resolve; "" if not a git repo
	SessionDir    string // where project.json lives; "" → skip write (unit tests)
	Provider      string // target deployment provider
}

// Name returns the canonical tool identifier.
func (g GitValidate) Name() string { return "git_validate" }

// (gitPatch struct removed as it is no longer used)

type gitBlock struct {
	Branch    *string `json:"branch"`
	CommitSHA *string `json:"commit_sha"`
	IsClean   *bool   `json:"is_clean"`
	Remote    *string `json:"remote"`
}

// IsCleanResult returns whether git.is_clean was true (false if nil or false).
// Callers use this to check the dirty-tree policy.
func IsCleanResult(output string) *bool {
	switch output {
	case "is_clean:true":
		t := true
		return &t
	case "is_clean:false":
		f := false
		return &f
	default:
		return nil
	}
}

// Execute shells to local git to fill the four git fields. If GitRoot is empty
// (not a git repo) all fields are left nil and a one-line warning is logged.
func (g GitValidate) Execute(ctx context.Context, s *session.Session) (ToolResult, error) {
	start := time.Now()

	if g.GitRoot == "" {
		log.Printf("git_validate: workspace %q is not inside a git repository — skipping git fields", g.WorkspaceRoot)
		return ToolResult{
			Success:  true,
			Output:   "no_git_repo",
			Duration: time.Since(start),
		}, nil
	}

	run := func(args ...string) (string, bool) {
		rc := RunCommand{Command: "git", Args: args, Dir: g.GitRoot}
		r, _ := rc.Execute(ctx, s)
		return strings.TrimSpace(r.Output), r.Success
	}

	var gb gitBlock

	// Branch.
	if out, ok := run("rev-parse", "--abbrev-ref", "HEAD"); ok && out != "" {
		v := out
		gb.Branch = &v
	}

	// Commit SHA.
	if out, ok := run("rev-parse", "HEAD"); ok && out != "" {
		v := out
		gb.CommitSHA = &v
	}

	// Is clean? (empty porcelain output ↔ clean)
	if out, ok := run("status", "--porcelain"); ok {
		clean := true
		relWorkspace, err := filepath.Rel(g.GitRoot, g.WorkspaceRoot)
		var atlasPathPrefix string
		if err == nil && relWorkspace != "." && relWorkspace != "" {
			atlasPathPrefix = filepath.ToSlash(relWorkspace) + "/.atlas"
		} else {
			atlasPathPrefix = ".atlas"
		}

		for _, line := range strings.Split(out, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if len(line) > 3 {
				path := line[3:]
				path = strings.Trim(path, `"`)
				if strings.HasPrefix(path, atlasPathPrefix+"/") || path == atlasPathPrefix || path == atlasPathPrefix+"/" {
					continue
				}
			}
			clean = false
			break
		}
		gb.IsClean = &clean
	}

	// Remote — missing remote is not an error.
	if out, ok := run("remote", "get-url", "origin"); ok && out != "" {
		v := out
		gb.Remote = &v
	}

	// Persist git block into project.json (merging with existing fields).
	if g.SessionDir != "" {
		if err := g.patchProjectGit(gb); err != nil {
			return ToolResult{
				Success:  false,
				Error:    err.Error(),
				Duration: time.Since(start),
			}, nil
		}
	}

	// Encode is_clean status into Output so orchestrator can read it without
	// loading project.json again.
	output := "no_git_info"
	if gb.IsClean != nil {
		if *gb.IsClean {
			output = "is_clean:true"
		} else {
			output = "is_clean:false"
		}
	}

	// For render provider, verify that the local HEAD matches the remote branch HEAD
	if g.Provider == "render" {
		if gb.Remote == nil || *gb.Remote == "" {
			return ToolResult{
				Success:  false,
				Error:    "Render requires a remote Git repository to deploy from. Please add a remote (e.g., origin) and push your code.",
				Duration: time.Since(start),
			}, nil
		}

		if gb.CommitSHA != nil && gb.Branch != nil {
			remoteRef := "origin/" + *gb.Branch
			if remoteSHA, ok := run("rev-parse", remoteRef); ok && remoteSHA != "" {
				if *gb.CommitSHA != remoteSHA {
					return ToolResult{
						Success:  false,
						Error:    fmt.Sprintf("Local commit %s hasn't been pushed — Render deploys from the remote branch, not local disk. Push your changes first.", (*gb.CommitSHA)[:7]),
						Duration: time.Since(start),
					}, nil
				}
			} else {
				return ToolResult{
					Success:  false,
					Error:    fmt.Sprintf("Local commit %s hasn't been pushed — Render deploys from the remote branch, not local disk. Push your changes first.", (*gb.CommitSHA)[:7]),
					Duration: time.Since(start),
				}, nil
			}
		}
	}

	return ToolResult{
		Success:  true,
		Output:   output,
		Duration: time.Since(start),
	}, nil
}

// patchProjectGit reads project.json, updates only the git block, and saves.
func (g GitValidate) patchProjectGit(gb gitBlock) error {
	// Load existing project.json if present; otherwise start fresh.
	var proj map[string]interface{}
	data, err := state.LoadJSONBytes(g.SessionDir, "project.json")
	if err == nil {
		_ = json.Unmarshal(data, &proj)
	}
	if proj == nil {
		proj = make(map[string]interface{})
	}
	proj["git"] = gb
	return state.SaveJSON(g.SessionDir, "project.json", proj)
}
