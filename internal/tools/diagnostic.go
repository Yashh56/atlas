package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/zendev-sh/goai"

	"github.com/Yashh56/atlas/internal/session"
)

// allowedDiagnostics is the strict allowlist for run_diagnostic.
// grep has been removed — use the dedicated search_symbol tool instead.
var allowedDiagnostics = map[string]bool{
	"go vet ./...":      true,
	"go build -n ./...": true,
	"npx tsc --noEmit":  true,
}

// RunDiagnostic is a bounded, read-only tool for the FixCode agentic loop.
type RunDiagnostic struct {
	WorkspaceRoot string
}

func (r RunDiagnostic) Name() string { return "run_diagnostic" }

type RunDiagnosticInput struct {
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
}

func (r RunDiagnostic) Execute(ctx context.Context, input RunDiagnosticInput) (string, error) {
	if !allowedDiagnostics[input.Command] {
		return "", fmt.Errorf("command %q is not on the allowlist", input.Command)
	}

	var cmdName string
	var args []string

	switch input.Command {
	case "go vet ./...":
		cmdName = "go"
		args = []string{"vet", "./..."}
	case "go build -n ./...":
		cmdName = "go"
		args = []string{"build", "-n", "./..."}
	case "npx tsc --noEmit":
		cmdName = "npx"
		args = []string{"tsc", "--noEmit"}
	}

	cmd := exec.CommandContext(ctx, cmdName, args...)
	cmd.Dir = r.WorkspaceRoot

	out, err := cmd.CombinedOutput()
	// Diagnostic commands (like go vet) often exit with status 1 when they find issues,
	// which is the exact output we want to return to the model, not a Go error.
	if err != nil {
		if len(out) == 0 {
			return "", fmt.Errorf("failed to run diagnostic: %w", err)
		}
	}

	return string(out), nil
}

// ---- SearchSymbol ----

// excludedDirs lists directories that are never searched.
var excludedDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	".atlas":       true,
	"vendor":       true,
	"dist":         true,
	"build":        true,
	"out":          true,
	".next":        true,
	".nuxt":        true,
	".svelte-kit":  true,
}

const searchSymbolMaxMatches = 20

// searchMatch is a single line match returned by SearchSymbol.
type searchMatch struct {
	File    string
	Line    int
	Content string
}

// SearchSymbol performs a recursive, read-only, native-Go text/regexp search across the
// workspace. It replaces the old grep entry in RunDiagnostic.
type SearchSymbol struct {
	WorkspaceRoot string
	Pattern       string
}

func (s SearchSymbol) Name() string { return "search_symbol" }

func (s SearchSymbol) Execute(ctx context.Context, _ *session.Session) (ToolResult, error) {
	if s.Pattern == "" {
		return ToolResult{Success: false, Error: "search pattern must not be empty"}, nil
	}

	re, err := regexp.Compile(s.Pattern)
	if err != nil {
		return ToolResult{Success: false, Error: fmt.Sprintf("invalid pattern %q: %v", s.Pattern, err)}, nil
	}

	var matches []searchMatch
	totalMatches := 0

	walkErr := filepath.WalkDir(s.WorkspaceRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip inaccessible paths silently
		}
		if d.IsDir() {
			if excludedDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		// Only search source files
		ext := strings.ToLower(filepath.Ext(d.Name()))
		switch ext {
		case ".go", ".ts", ".tsx", ".js", ".jsx", ".py", ".rs",
			".java", ".kt", ".swift", ".vue", ".svelte", ".rb",
			".cs", ".cpp", ".c", ".h", ".toml", ".yaml", ".yml", ".json":
			// continue
		default:
			return nil
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}

		rel, _ := filepath.Rel(s.WorkspaceRoot, path)
		rel = filepath.ToSlash(rel)

		for lineNum, line := range strings.Split(string(data), "\n") {
			if re.MatchString(line) {
				totalMatches++
				if len(matches) < searchSymbolMaxMatches {
					matches = append(matches, searchMatch{
						File:    rel,
						Line:    lineNum + 1,
						Content: strings.TrimSpace(line),
					})
				}
			}
		}
		return nil
	})

	if walkErr != nil {
		return ToolResult{Success: false, Error: fmt.Sprintf("search failed: %v", walkErr)}, nil
	}

	if totalMatches == 0 {
		return ToolResult{
			Success: true,
			Output:  fmt.Sprintf("No matches found for pattern %q.", s.Pattern),
		}, nil
	}

	var sb strings.Builder
	for _, m := range matches {
		sb.WriteString(fmt.Sprintf("%s:%d: %s\n", m.File, m.Line, m.Content))
	}

	if totalMatches > searchSymbolMaxMatches {
		sb.WriteString(fmt.Sprintf("\n... %d more match(es) not shown (truncated at %d).\n",
			totalMatches-searchSymbolMaxMatches, searchSymbolMaxMatches))
	}

	return ToolResult{Success: true, Output: sb.String()}, nil
}

// ---- ToGoAITools ----

// ToGoAITools exposes the read-only bounded tools for the FixCode agentic loop:
// read_file, run_diagnostic, and search_symbol.
func ToGoAITools(workspaceRoot string) []goai.Tool {
	readTool := goai.NewTool(
		"read_file",
		"Read the full contents of a specific file in the workspace. Use this when you already know which file to inspect.",
		func(ctx context.Context, input struct {
			Path string `json:"path"`
		}) (string, error) {
			absPath, _, err := ResolveWorkspacePath(workspaceRoot, input.Path)
			if err != nil {
				return "", err
			}

			res, err := ReadFile{Path: absPath}.Execute(ctx, &session.Session{})
			if err != nil {
				return "", err
			}
			if !res.Success {
				return "", fmt.Errorf("%s", res.Error)
			}
			return res.Output, nil
		},
	)

	diagTool := goai.NewTool(
		"run_diagnostic",
		"Run a read-only diagnostic command to investigate an error. Allowed commands: \"go vet ./...\", \"go build -n ./...\", \"npx tsc --noEmit\".",
		func(ctx context.Context, input RunDiagnosticInput) (string, error) {
			return RunDiagnostic{WorkspaceRoot: workspaceRoot}.Execute(ctx, input)
		},
	)

	searchTool := goai.NewTool(
		"search_symbol",
		"Search recursively across all source files in the workspace for a symbol name, function, type, or any text pattern. "+
			"Use this when the error references something you can't find in the file you were shown — "+
			"call this FIRST before guessing a file path with read_file. Returns up to 20 matches with file path and line number.",
		func(ctx context.Context, input struct {
			Pattern string `json:"pattern"`
		}) (string, error) {
			res, err := SearchSymbol{WorkspaceRoot: workspaceRoot, Pattern: input.Pattern}.Execute(ctx, nil)
			if err != nil {
				return "", err
			}
			if !res.Success {
				return "", fmt.Errorf("%s", res.Error)
			}
			return res.Output, nil
		},
	)

	return []goai.Tool{readTool, diagTool, searchTool}
}
