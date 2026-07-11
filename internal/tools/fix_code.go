package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Yashh56/atlas/internal/llm"
	"github.com/Yashh56/atlas/internal/session"
	"github.com/Yashh56/atlas/internal/state"
)

type FixCode struct {
	WorkspaceRoot string
	Client        llm.Client
	SessionDir    string // needed to load project.json and build.json
}

func (f FixCode) Name() string { return "fix_code" }

type fixResponse struct {
	File      *string `json:"file"`
	Content   *string `json:"content"`
	Reasoning string  `json:"reasoning"`
}

func (f FixCode) Execute(ctx context.Context, sess *session.Session) (ToolResult, error) {
	start := time.Now()

	// 1. Load skill file
	// In production, this path would be relative to the atlas binary or a configured skills dir.
	// For this test/project, we assume the skills dir is at the root of the project running atlas.
	// We can use a relative path if atlas is run from the project root, or we can look it up.
	// For simplicity per spec, we'll assume "skills/fix_build.md" is accessible from the current working dir.
	skillPath := "skills/fix_build.md"
	
	// A robust way to find it during tests vs runtime:
	// We'll try the current dir, and if it fails, maybe we are in a test and need to go up.
	// For now, let's just read it directly, assuming the binary is run from the devopsAgent root.
	// In tests, we might mock this or set a different path.
	
	skillBytes, err := os.ReadFile(skillPath)
	if err != nil {
		// Fallback for tests running from internal/tools
		skillBytes, err = os.ReadFile("../../skills/fix_build.md")
		if err != nil {
			return ToolResult{Success: false, Error: "failed to load fix_build skill: " + err.Error(), Duration: time.Since(start)}, nil
		}
	}
	systemPrompt := string(skillBytes)

	// 2. Load context (framework + build log)
	var proj struct {
		Framework *string `json:"framework"`
	}
	if err := state.LoadJSON(f.SessionDir, "project.json", &proj); err != nil {
		return ToolResult{Success: false, Error: "failed to load project state: " + err.Error(), Duration: time.Since(start)}, nil
	}
	
	var bs struct {
		LogPath string `json:"log_path"`
	}
	if err := state.LoadJSON(f.SessionDir, "build.json", &bs); err != nil {
		return ToolResult{Success: false, Error: "failed to load build state: " + err.Error(), Duration: time.Since(start)}, nil
	}

	framework := "unknown"
	if proj.Framework != nil {
		framework = *proj.Framework
	}

	buildLog := ""
	if bs.LogPath != "" {
		tail, err := ReadTailLines(bs.LogPath, 40)
		if err == nil {
			buildLog = tail
		} else {
			buildLog = "<failed to read build log>"
		}
	} else {
		buildLog = "<no build log path>"
	}

	// 3. Attempt to find a relevant file to include
	// Very simple heuristic: look for something that looks like a file path with an extension.
	// E.g. main.go, src/index.ts, etc.
	// We'll just look for a word containing a dot and a slash, or just an extension we know.
	var fileContext string
	filePath := extractFilePath(buildLog)
	if filePath != "" {
		absPath := filepath.Join(f.WorkspaceRoot, filePath)
		contentBytes, err := os.ReadFile(absPath)
		if err == nil {
			fileContext = fmt.Sprintf("\nContents of %s:\n```\n%s\n```\n", filePath, string(contentBytes))
		}
	}

	userPrompt := fmt.Sprintf("Framework: %s\n\nBuild Log Excerpt:\n```\n%s\n```\n%s", framework, buildLog, fileContext)

	// 4. Call LLM
	respText, err := f.Client.Complete(ctx, systemPrompt, userPrompt)
	if err != nil {
		return ToolResult{Success: false, Error: "llm call failed: " + err.Error(), Duration: time.Since(start)}, nil
	}

	// 5. Parse JSON
	var fix fixResponse
	// The LLM might wrap the JSON in markdown blocks despite instructions.
	// We strip them just in case.
	respText = stripMarkdownFences(respText)
	
	if err := json.Unmarshal([]byte(respText), &fix); err != nil {
		// Malformed JSON is a failed attempt, not a crash.
		return ToolResult{
			Success:  false,
			Output:   "LLM returned malformed JSON",
			Error:    err.Error(),
			Duration: time.Since(start),
		}, nil
	}

	if fix.File == nil || fix.Content == nil {
		return ToolResult{
			Success:  false,
			Output:   "LLM declined to fix: " + fix.Reasoning,
			Duration: time.Since(start),
		}, nil
	}

	// 6. Write file
	writer := WriteFile{
		WorkspaceRoot: f.WorkspaceRoot,
		Path:          *fix.File,
		Content:       *fix.Content,
	}
	
	writeRes, err := writer.Execute(ctx, sess)
	if err != nil {
		return ToolResult{Success: false, Error: err.Error(), Duration: time.Since(start)}, nil
	}
	if !writeRes.Success {
		return ToolResult{Success: false, Error: writeRes.Error, Duration: time.Since(start)}, nil
	}

	output := fmt.Sprintf("Fixed: %s — %q", *fix.File, fix.Reasoning)
	return ToolResult{
		Success:  true,
		Output:   output,
		Duration: time.Since(start),
	}, nil
}

// stripMarkdownFences removes ```json and ``` from the start/end of a string if present.
func stripMarkdownFences(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
	}
	if strings.HasSuffix(s, "```") {
		s = strings.TrimSuffix(s, "```")
	}
	return strings.TrimSpace(s)
}

// extractFilePath is a simple heuristic to find a file path in the build log.
func extractFilePath(log string) string {
	// Look for typical file extensions in paths. e.g. foo/bar.go, src/app.tsx
	re := regexp.MustCompile(`([a-zA-Z0-9_\-\.\/]+\.(go|ts|tsx|js|jsx|json|py|rs|c|cpp|h))`)
	matches := re.FindStringSubmatch(log)
	if len(matches) > 1 {
		// Often errors format like ".\main.go:4:7" so we clean up leading ".\" or "./"
		path := matches[1]
		path = strings.TrimPrefix(path, ".\\")
		path = strings.TrimPrefix(path, "./")
		return path
	}
	return ""
}
