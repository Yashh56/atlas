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

	"github.com/charmbracelet/lipgloss"
	"github.com/zendev-sh/goai"
	"github.com/zendev-sh/goai/provider"

	"github.com/Yashh56/atlas/internal/llm"
	"github.com/Yashh56/atlas/internal/session"
	"github.com/Yashh56/atlas/internal/state"
)

// FixCode uses GoAI structured generation to fix build errors.
// It depends on llm.Model (not llm.Client) so it can use GenerateStructured.
type FixCode struct {
	WorkspaceRoot    string
	Model            llm.Model // was: Client llm.Client
	ProviderName     string    // e.g. "mistral", "groq", "openai" — used to skip incompatible paths
	SessionDir       string    // needed to load project.json and build.json
	LastPatchError   string    // set by orchestrator when the previous patch application failed
	ForceIncludeFile string    // always include this file in context (e.g. the file last modified)
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

	// 3. Attempt to find all relevant files to include as context
	sourceExts := map[string]bool{
		".go": true, ".ts": true, ".tsx": true, ".js": true, ".jsx": true,
		".json": true, ".vue": true, ".svelte": true, ".py": true,
		".rs": true, ".toml": true, ".yaml": true, ".yml": true,
		".html": true, ".css": true, ".md": true,
	}

	var fileContext string
	loadedPaths := map[string]bool{}

	// Always load ForceIncludeFile first — this is the file we previously modified,
	// so the model must see its CURRENT state even if the build log points elsewhere.
	if f.ForceIncludeFile != "" {
		forceAbs := filepath.Join(f.WorkspaceRoot, f.ForceIncludeFile)
		if content, err := os.ReadFile(forceAbs); err == nil {
			fileContext = fmt.Sprintf("\nCurrent contents of %s (file previously modified — read carefully before proposing old_str):\n```\n%s\n```\n", f.ForceIncludeFile, string(content))
			loadedPaths[forceAbs] = true
		}
	}

	filePath := extractFilePath(buildLog)
	if filePath != "" {
		absPath := filepath.Join(f.WorkspaceRoot, filePath)
		contentBytes, err := os.ReadFile(absPath)
		if err == nil {
			fileContext = fmt.Sprintf("\nContents of %s:\n```\n%s\n```\n", filePath, string(contentBytes))
			loadedPaths[absPath] = true

			// Load sibling source files from the same directory for immediate context
			dirPath := filepath.Dir(absPath)
			entries, dirErr := os.ReadDir(dirPath)
			if dirErr == nil {
				for _, entry := range entries {
					if entry.IsDir() {
						// One level deep — walk into subdirectories to find package files
						// (e.g. utils/utils.go) that the error might originate from.
						skipDir := map[string]bool{
							".git": true, "node_modules": true, ".atlas": true,
							"vendor": true, "dist": true, "build": true, "out": true,
						}
						if skipDir[entry.Name()] {
							continue
						}
						subDir := filepath.Join(dirPath, entry.Name())
						subEntries, subErr := os.ReadDir(subDir)
						if subErr != nil {
							continue
						}
						for _, subEntry := range subEntries {
							if subEntry.IsDir() {
								continue
							}
							ext := strings.ToLower(filepath.Ext(subEntry.Name()))
							subAbs := filepath.Join(subDir, subEntry.Name())
							if loadedPaths[subAbs] || !sourceExts[ext] {
								continue
							}
							subBytes, readErr := os.ReadFile(subAbs)
							if readErr == nil && len(subBytes) < 8000 {
								relSub, _ := filepath.Rel(f.WorkspaceRoot, subAbs)
								relSub = filepath.ToSlash(relSub)
								fileContext += fmt.Sprintf("\nContents of %s:\n```\n%s\n```\n", relSub, string(subBytes))
								loadedPaths[subAbs] = true
							}
						}
						continue
					}
					ext := strings.ToLower(filepath.Ext(entry.Name()))
					siblingAbs := filepath.Join(dirPath, entry.Name())
					if loadedPaths[siblingAbs] || !sourceExts[ext] {
						continue
					}
					siblingBytes, readErr := os.ReadFile(siblingAbs)
					if readErr == nil && len(siblingBytes) < 8000 {
						relSiblingPath := filepath.Join(filepath.Dir(filePath), entry.Name())
						fileContext += fmt.Sprintf("\nContents of %s:\n```\n%s\n```\n", relSiblingPath, string(siblingBytes))
						loadedPaths[siblingAbs] = true
					}
				}
			}

			// Always include package.json if this is a JS/TS/config file and we haven't already
			pkgJSON := filepath.Join(f.WorkspaceRoot, "package.json")
			if !loadedPaths[pkgJSON] {
				if pkgBytes, pkgErr := os.ReadFile(pkgJSON); pkgErr == nil {
					fileContext += fmt.Sprintf("\nContents of package.json:\n```\n%s\n```\n", string(pkgBytes))
					loadedPaths[pkgJSON] = true
				}
			}
		}
	}

	userPrompt := fmt.Sprintf("Framework: %s\n\nBuild Log Excerpt:\n```\n%s\n```\n%s", framework, buildLog, fileContext)
	if f.LastPatchError != "" {
		userPrompt += fmt.Sprintf("\n\n⚠️ Previous fix attempt FAILED to apply: %s\nThe old_str you used did not match the file exactly. Study the file contents above carefully and try a different, shorter, more exact old_str.", f.LastPatchError)
	}

	// 4. Call LLM with structured generation — GoAI handles JSON schema enforcement and tool loop.
	// Providers that are known to reject tool calling + JSON mode simultaneously (e.g. Mistral)
	// skip directly to the text-generation fallback to avoid the full timeout cost.
	// Mistral-family providers (mistral, codestral, devstral, pixtral) reject the combination
	// of JSON structured output + tool calling. Skip straight to the text-generation fallback
	// to avoid burning the full 60-second timeout on every fix attempt.
	isMistralFamily := strings.EqualFold(f.ProviderName, "mistral") ||
		strings.HasPrefix(strings.ToLower(f.ProviderName), "codestral") ||
		strings.HasPrefix(strings.ToLower(f.ProviderName), "devstral") ||
		strings.HasPrefix(strings.ToLower(f.ProviderName), "pixtral")
	skipStructured := isMistralFamily

	var fix FixResponse
	var usage *provider.Usage
	var genErr error

	if !skipStructured {
		llmCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
		defer cancel()

		fix, usage, genErr = llm.GenerateStructured[FixResponse](
			llmCtx, f.Model, systemPrompt, userPrompt,
			goai.WithMaxSteps(3),
			goai.WithTools(ToGoAITools(f.WorkspaceRoot)...),
		)
		if genErr != nil {
			// Attempt to recover from a parsing error if the raw output is in the error message
			if strings.Contains(genErr.Error(), "parsing structured output") && strings.Contains(genErr.Error(), "(raw:") {
				parts := strings.SplitN(genErr.Error(), "(raw:", 2)
				if len(parts) == 2 {
					rawJSON := strings.TrimSpace(strings.TrimSuffix(parts[1], ")"))
					startIdx := strings.Index(rawJSON, "{")
					endIdx := strings.LastIndex(rawJSON, "}")
					if startIdx != -1 && endIdx != -1 && endIdx > startIdx {
						extracted := rawJSON[startIdx : endIdx+1]
						if parseErr := json.Unmarshal([]byte(extracted), &fix); parseErr == nil {
							genErr = nil
						}
					}
				}
			}
		}
	}

	// Fall back to plain text generation when:
	//  - The provider doesn't support tool calling + JSON mode together (Mistral)
	//  - The structured call failed with a known incompatibility or timeout
	needsFallback := skipStructured || (genErr != nil && (strings.Contains(genErr.Error(), "json mode cannot be combined with tool/function calling") || strings.Contains(strings.ToLower(genErr.Error()), "json mode") || strings.Contains(strings.ToLower(genErr.Error()), "max steps") || strings.Contains(genErr.Error(), "parsing structured output") || strings.Contains(genErr.Error(), "reasoning_content is unsupported") || strings.Contains(genErr.Error(), "reasoning_content") || strings.Contains(genErr.Error(), "context deadline exceeded")))
	if needsFallback {
		// Use GenerateText to bypass the strict JSON Schema + Tools conflict.
		// Tools are disabled here to avoid provider-specific bugs (like Groq rejecting reasoning_content in tool-call history).
		// We use a clean system prompt to ensure the model doesn't hallucinate tool calls based on the original skill prompt.
		fallbackPrompt := systemPrompt
		if idx := strings.Index(fallbackPrompt, "Before proposing a fix"); idx != -1 {
			fallbackPrompt = fallbackPrompt[:idx]
		}
		fallbackPrompt += "\n\nIMPORTANT: Output ONLY the raw JSON object, no markdown fences, no explanation. DO NOT CALL ANY TOOLS. YOU MUST OUTPUT THE JSON RESULT IMMEDIATELY.\nYour JSON object MUST conform EXACTLY to this structure:\n{\n  \"file\": \"path to the file to modify\",\n  \"old_str\": \"the exact characters to replace\",\n  \"new_str\": \"the exact characters to replace them with\",\n  \"reasoning\": \"brief explanation of the fix\"\n}"

		fallbackCtx, fallbackCancel := context.WithTimeout(ctx, 60*time.Second)
		defer fallbackCancel()

		textRes, textErr := goai.GenerateText(fallbackCtx, f.Model,
			goai.WithSystem(fallbackPrompt),
			goai.WithPrompt(userPrompt),
			goai.WithMaxSteps(1),
		)
		if textErr != nil {
			return ToolResult{
				Success:  false,
				Error:    fmt.Sprintf("structured fallback (GenerateText) failed: %v", textErr),
				Duration: time.Since(start),
			}, nil
		}

		// Clean markdown fences if the model ignored the instructions
		rawJSON := strings.TrimSpace(textRes.Text)
		startIdx := strings.Index(rawJSON, "{")
		endIdx := strings.LastIndex(rawJSON, "}")
		if startIdx != -1 && endIdx != -1 && endIdx > startIdx {
			rawJSON = rawJSON[startIdx : endIdx+1]
		}

		if parseErr := json.Unmarshal([]byte(rawJSON), &fix); parseErr != nil {
			return ToolResult{
				Success:  false,
				Error:    fmt.Sprintf("failed to parse fallback text as JSON: %v", parseErr),
				Duration: time.Since(start),
			}, nil
		}
		usage = &textRes.TotalUsage
	} else if genErr != nil {
		// Structured generation failed with an unrecognized error — surface it.
		return ToolResult{
			Success:  false,
			Error:    fmt.Sprintf("structured generation failed: %v", genErr),
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

	if fix.OldStr == "" && fix.NewStr == "" {
		// Model signalled it can't confidently fix — treat as a non-actionable response
		errMsg := "LLM returned an empty old_str and new_str (model declined to propose a fix)"
		if fix.Reasoning != "" {
			errMsg += " (Reasoning: " + fix.Reasoning + ")"
		}
		return ToolResult{
			Success:    false,
			Error:      errMsg,
			TargetFile: fix.File,
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
		return ToolResult{Success: false, Error: err.Error(), TargetFile: fix.File, Duration: time.Since(start), TokenUsage: tu}, nil
	}
	if !writeRes.Success {
		return ToolResult{Success: false, Error: writeRes.Error, TargetFile: fix.File, Duration: time.Since(start), TokenUsage: tu}, nil
	}

	// Generate a visual diff
	diffBuilder := &strings.Builder{}
	diffBuilder.WriteString(fmt.Sprintf("Fixed: %s — %q\n\n", fix.File, fix.Reasoning))
	
	redStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	greenStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("46"))

	for _, line := range strings.Split(fix.OldStr, "\n") {
		diffBuilder.WriteString(redStyle.Render(fmt.Sprintf("- %s", line)) + "\n")
	}
	for _, line := range strings.Split(fix.NewStr, "\n") {
		diffBuilder.WriteString(greenStyle.Render(fmt.Sprintf("+ %s", line)) + "\n")
	}
	output := diffBuilder.String()
	return ToolResult{
		Success:    true,
		Output:     output,
		Duration:   time.Since(start),
		TokenUsage: tu,
	}, nil
}

// extractFilePath is a simple heuristic to find a file path in the build log.
func extractFilePath(log string) string {
	re := regexp.MustCompile(`([a-zA-Z0-9_\-\.\/\\]+\.(go|ts|tsx|js|jsx|json|py|rs|c|cpp|h)):\d+`)
	matches := re.FindStringSubmatch(log)
	if len(matches) > 1 {
		path := matches[1]
		path = strings.TrimPrefix(path, ".\\")
		path = strings.TrimPrefix(path, "./")
		return path
	}
	return ""
}
