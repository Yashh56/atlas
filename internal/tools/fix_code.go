package tools

import (
	"context"
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

// FixCode uses GoAI structured generation to fix build errors.
// It depends on llm.Model (not llm.Client) so it can use GenerateStructured.
type FixCode struct {
	WorkspaceRoot string
	Model         llm.Model // was: Client llm.Client
	SessionDir    string    // needed to load project.json and build.json
}

func (f FixCode) Name() string { return "fix_code" }

// FixResponse is the structured output type for the fix-code LLM call.
// GoAI's GenerateObject auto-generates the JSON schema from this struct.
type FixResponse struct {
	File      string `json:"file"`
	OldStr    string `json:"old_str"`
	NewStr    string `json:"new_str"`
	Reasoning string `json:"reasoning"`
}

func (f FixCode) Execute(ctx context.Context, sess *session.Session) (ToolResult, error) {
	start := time.Now()

	// 1. Load skill file
	skillPath := "skills/fix_build.md"
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

	// 4. Call LLM with structured generation — GoAI handles JSON schema enforcement
	fix, usage, err := llm.GenerateStructured[FixResponse](ctx, f.Model, systemPrompt, userPrompt)
	if err != nil {
		// THIS is the error that was going missing before — make sure it actually
		// reaches ToolResult.Error and gets printed by the orchestrator.
		return ToolResult{
			Success:  false,
			Error:    fmt.Sprintf("structured generation failed: %v", err),
			Duration: time.Since(start),
		}, nil
	}

	var tu *TokenUsage
	if usage != nil {
		tu = &TokenUsage{
			InputTokens:  usage.InputTokens,
			OutputTokens: usage.OutputTokens,
			TotalTokens:  usage.TotalTokens,
		}
	}

	if fix.File == "" {
		return ToolResult{
			Success:    false,
			Error:      "LLM did not return a file to modify",
			Duration:   time.Since(start),
			TokenUsage: tu,
		}, nil
	}

	// 5. Write file
	patcher := PatchFile{
		WorkspaceRoot: f.WorkspaceRoot,
		SessionDir:    f.SessionDir,
		Path:          fix.File,
		OldStr:        fix.OldStr,
		NewStr:        fix.NewStr,
	}

	writeRes, err := patcher.Execute(ctx, sess)
	if err != nil {
		return ToolResult{Success: false, Error: err.Error(), Duration: time.Since(start), TokenUsage: tu}, nil
	}
	if !writeRes.Success {
		return ToolResult{Success: false, Error: writeRes.Error, Duration: time.Since(start), TokenUsage: tu}, nil
	}

	output := fmt.Sprintf("Fixed: %s — %q", fix.File, fix.Reasoning)
	return ToolResult{
		Success:    true,
		Output:     output,
		Duration:   time.Since(start),
		TokenUsage: tu,
	}, nil
}

// extractFilePath is a simple heuristic to find a file path in the build log.
func extractFilePath(log string) string {
	re := regexp.MustCompile(`([a-zA-Z0-9_\-\.\/]+\.(go|ts|tsx|js|jsx|json|py|rs|c|cpp|h))`)
	matches := re.FindStringSubmatch(log)
	if len(matches) > 1 {
		path := matches[1]
		path = strings.TrimPrefix(path, ".\\")
		path = strings.TrimPrefix(path, "./")
		return path
	}
	return ""
}
