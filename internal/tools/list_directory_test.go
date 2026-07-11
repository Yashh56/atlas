package tools_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Yashh56/atlas/internal/session"
	"github.com/Yashh56/atlas/internal/tools"
)

func TestListDirectory(t *testing.T) {
	dir := t.TempDir()

	// Create fixtures: files, subdirs, hidden entries.
	writeFile := func(name string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	makeDir := func(name string) {
		if err := os.Mkdir(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	writeFile("package.json")
	writeFile("tsconfig.json")
	writeFile(".gitignore")   // hidden — must be excluded
	makeDir("src")
	makeDir("node_modules")
	makeDir(".git") // hidden dir — must be excluded

	tool := tools.ListDirectory{Path: dir}
	result, err := tool.Execute(context.Background(), &session.Session{})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected Success=true, got false; Error=%q", result.Error)
	}

	entries := strings.Split(strings.TrimSpace(result.Output), "\n")
	entrySet := make(map[string]bool, len(entries))
	for _, e := range entries {
		entrySet[e] = true
	}

	// Visible files.
	if !entrySet["package.json"] {
		t.Error("expected package.json in output")
	}
	if !entrySet["tsconfig.json"] {
		t.Error("expected tsconfig.json in output")
	}
	// Visible dirs (suffixed with /).
	if !entrySet["src/"] {
		t.Error("expected src/ in output")
	}
	if !entrySet["node_modules/"] {
		t.Error("expected node_modules/ in output")
	}
	// Hidden entries must be absent.
	if entrySet[".gitignore"] {
		t.Error("expected .gitignore to be excluded")
	}
	if entrySet[".git/"] {
		t.Error("expected .git/ to be excluded")
	}
}

func TestListDirectory_NonExistentPath(t *testing.T) {
	tool := tools.ListDirectory{Path: "/this/path/does/not/exist/atlas"}
	result, err := tool.Execute(context.Background(), &session.Session{})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if result.Success {
		t.Error("expected Success=false for nonexistent path")
	}
}
