package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---- RunDiagnostic tests ----

func TestRunDiagnostic_Allowlist(t *testing.T) {
	ctx := context.Background()
	ws := "/fake/workspace"

	tests := []struct {
		name    string
		command string
		wantErr bool
		errSubstr string
	}{
		{
			name:    "allowed: go vet",
			command: "go vet ./...",
		},
		{
			name:    "allowed: go build -n",
			command: "go build -n ./...",
		},
		{
			name:      "disallowed: arbitrary command",
			command:   "rm -rf /",
			wantErr:   true,
			errSubstr: "allowlist",
		},
		{
			name:      "disallowed: shell chaining",
			command:   "go vet ./... && rm -rf /",
			wantErr:   true,
			errSubstr: "allowlist",
		},
		{
			name:      "disallowed: grep is no longer on allowlist",
			command:   "grep",
			wantErr:   true,
			errSubstr: "allowlist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := RunDiagnostic{WorkspaceRoot: ws}
			_, err := tool.Execute(ctx, RunDiagnosticInput{Command: tt.command})

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.errSubstr != "" && !strings.Contains(err.Error(), tt.errSubstr) {
					t.Fatalf("expected error containing %q, got: %v", tt.errSubstr, err)
				}
			} else {
				// Non-allowlist errors are exec failures (fake ws), not validation failures
				if err != nil && strings.Contains(err.Error(), "allowlist") {
					t.Fatalf("unexpected allowlist block: %v", err)
				}
			}
		})
	}
}

// ---- SearchSymbol tests ----

// makeSearchFixture writes a small tree of source files plus directories that
// must be excluded, and returns the root path.
func makeSearchFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	write := func(rel, content string) {
		t.Helper()
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Files that SHOULD be searched
	write("main.go", "package main\n\nfunc main() {\n\tfmt.Println(AddNumbers(1, 2))\n}\n")
	write("utils/utils.go", "package utils\n\nfunc AddNumbers(a, b int) int {\n\treturn a + b\n}\n")
	write("src/index.ts", "export function AddNumbers(a: number, b: number): number {\n\treturn a + b;\n}\n")

	// Files inside excluded dirs that must NOT be searched
	write("node_modules/lodash/index.js", "function AddNumbers(a, b) { return a+b; }")
	write(".git/COMMIT_EDITMSG", "feat: AddNumbers refactor")
	write(".atlas/sessions/sess_abc/build.log", "AddNumbers compiled ok")
	write("vendor/github.com/foo/bar.go", "func AddNumbers() {}")
	write("dist/bundle.js", "AddNumbers(1,2)")

	return root
}

func TestSearchSymbol_SingleMatch(t *testing.T) {
	root := makeSearchFixture(t)
	ctx := context.Background()

	res, err := SearchSymbol{WorkspaceRoot: root, Pattern: "func AddNumbers"}.Execute(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected success, got error: %s", res.Error)
	}
	// Must find utils.go
	if !strings.Contains(res.Output, "utils/utils.go") {
		t.Errorf("expected utils/utils.go in output, got:\n%s", res.Output)
	}
	// Must NOT include excluded directories
	for _, bad := range []string{"node_modules", ".git", ".atlas", "vendor", "dist"} {
		if strings.Contains(res.Output, bad) {
			t.Errorf("excluded dir %q leaked into search results:\n%s", bad, res.Output)
		}
	}
}

func TestSearchSymbol_MultipleFiles(t *testing.T) {
	root := makeSearchFixture(t)
	ctx := context.Background()

	// "AddNumbers" appears in main.go, utils.go, and src/index.ts (but not excluded dirs)
	res, err := SearchSymbol{WorkspaceRoot: root, Pattern: "AddNumbers"}.Execute(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected success, got error: %s", res.Error)
	}
	for _, expected := range []string{"main.go", "utils/utils.go", "src/index.ts"} {
		if !strings.Contains(res.Output, filepath.ToSlash(expected)) {
			t.Errorf("expected %q in output, got:\n%s", expected, res.Output)
		}
	}
}

func TestSearchSymbol_ZeroMatches(t *testing.T) {
	root := makeSearchFixture(t)
	ctx := context.Background()

	res, err := SearchSymbol{WorkspaceRoot: root, Pattern: "ThisSymbolDefinitelyDoesNotExist_XYZ"}.Execute(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected success (not an error), got: %s", res.Error)
	}
	if !strings.Contains(res.Output, "No matches") {
		t.Errorf("expected 'No matches' message, got: %s", res.Output)
	}
}

func TestSearchSymbol_TruncatesAt20(t *testing.T) {
	root := t.TempDir()
	ctx := context.Background()

	// Write 25 files each containing the target token once
	for i := 0; i < 25; i++ {
		name := filepath.Join(root, "src", strings.Repeat("a", i+1)+".go")
		_ = os.MkdirAll(filepath.Dir(name), 0o755)
		_ = os.WriteFile(name, []byte("package src\n\nfunc FINDSYMBOL() {}\n"), 0o644)
	}

	res, err := SearchSymbol{WorkspaceRoot: root, Pattern: "FINDSYMBOL"}.Execute(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected success, got: %s", res.Error)
	}
	lines := strings.Split(strings.TrimSpace(res.Output), "\n")
	matchLines := 0
	hasTruncNote := false
	for _, l := range lines {
		if strings.Contains(l, "FINDSYMBOL") {
			matchLines++
		}
		if strings.Contains(l, "more match") {
			hasTruncNote = true
		}
	}
	if matchLines > searchSymbolMaxMatches {
		t.Errorf("got %d match lines, expected at most %d", matchLines, searchSymbolMaxMatches)
	}
	if !hasTruncNote {
		t.Errorf("expected truncation note in output, got:\n%s", res.Output)
	}
}

func TestSearchSymbol_ExcludedDirsNotSearched(t *testing.T) {
	root := makeSearchFixture(t)
	ctx := context.Background()

	// "ONLY_IN_EXCLUDED" only exists inside excluded dirs — must never appear in results
	onlyInExcluded := "ONLY_IN_EXCLUDED_XYZ"
	for _, excl := range []string{
		"node_modules/a/file.js",
		".git/config",
		".atlas/x.json",
		"vendor/x/y.go",
		"dist/app.js",
	} {
		abs := filepath.Join(root, filepath.FromSlash(excl))
		_ = os.MkdirAll(filepath.Dir(abs), 0o755)
		_ = os.WriteFile(abs, []byte(onlyInExcluded), 0o644)
	}

	res, err := SearchSymbol{WorkspaceRoot: root, Pattern: onlyInExcluded}.Execute(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected success, got: %s", res.Error)
	}
	if !strings.Contains(res.Output, "No matches") {
		t.Errorf("expected no matches (all in excluded dirs), got:\n%s", res.Output)
	}
}

func TestSearchSymbol_InvalidRegexp(t *testing.T) {
	root := t.TempDir()
	ctx := context.Background()

	res, err := SearchSymbol{WorkspaceRoot: root, Pattern: "["}.Execute(ctx, nil)
	if err != nil {
		t.Fatalf("unexpected tool-level error: %v", err)
	}
	if res.Success {
		t.Fatalf("expected failure for invalid regexp, got success")
	}
	if !strings.Contains(res.Error, "invalid pattern") {
		t.Errorf("expected 'invalid pattern' in error, got: %s", res.Error)
	}
}

// ---- ToGoAITools isolation test ----

func TestToGoAITools_Isolation(t *testing.T) {
	ws := "/fake/workspace"
	tools := ToGoAITools(ws)

	// Must now expose exactly 3 tools: read_file, run_diagnostic, search_symbol
	if len(tools) != 3 {
		t.Fatalf("expected 3 tools, got %d", len(tools))
	}

	requiredTools := map[string]bool{
		"read_file":      false,
		"run_diagnostic": false,
		"search_symbol":  false,
	}
	mutatingNames := map[string]bool{
		"write_file":       true,
		"patch_file":       true,
		"run_command":      true,
		"delete_file":      true,
		"rename_file":      true,
		"analyze_project":  true,
	}

	for _, tool := range tools {
		if mutatingNames[tool.Name] {
			t.Fatalf("CRITICAL ISOLATION FAILURE: mutating tool %q found in ToGoAITools output", tool.Name)
		}
		if _, ok := requiredTools[tool.Name]; ok {
			requiredTools[tool.Name] = true
		}
	}

	for name, seen := range requiredTools {
		if !seen {
			t.Errorf("expected tool %q in ToGoAITools output but it was missing", name)
		}
	}
}
